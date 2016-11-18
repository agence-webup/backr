package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"
	"webup/backr"
	"webup/backr/http"
	"webup/backr/state"
	"webup/backr/swift"
	"webup/backr/tasks"

	log "github.com/Sirupsen/logrus"
	cli "github.com/jawher/mow.cli"
)

func main() {
	app := cli.App("backr", "Perform backups")

	app.Version("v version", "Backr 1 (build 6)")

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})

	app.Command("daemon", "Start the backup process", func(cmd *cli.Cmd) {

		// cmd.Spec = "-w... --etcd|--local [--time] [--config-refresh-rate]"
		cmd.Spec = "-w... --etcd|--local [--time] [--api-listen] [--debug]"

		// state storage
		stateStorageSettings := getStateStorateSettings(cmd)

		// swift
		swiftSettings := getSwiftSettings(cmd)
		if swiftSettings == nil {
			log.Warnln("Swift upload will be unavailable because some args or env vars are missing to configure Swift upload")
		}

		// options
		watchDirs := cmd.StringsOpt("w watch", []string{}, "Specifies the directories to watch for finding backup.yml files")
		timeOpt := cmd.StringOpt("time", "01:00", "Specifies the moment when the backup process will be started")
		apiListenOpt := cmd.StringOpt("api-listen", ":22257", "Configure IP and port for HTTP API")
		debug := cmd.BoolOpt("debug", false, "Enables the debug logs output")

		cmd.Action = func() {

			// set debug log level if needed
			if *debug {
				log.SetLevel(log.DebugLevel)
			}

			ctx, cancel := context.WithCancel(context.Background())

			// prepare options
			currentSettings := backr.NewDefaultSettings()
			currentSettings.StateStorage = stateStorageSettings
			currentSettings.WatchDirs = *watchDirs
			currentSettings.Swift = swiftSettings
			currentSettings.ApiListen = *apiListenOpt

			// parse the time option
			if timeOpt != nil {
				parsedTime, err := time.Parse("15:04", *timeOpt)
				if err == nil {
					currentSettings.TimeSpec.Hour = parsedTime.Hour()
					currentSettings.TimeSpec.Minute = parsedTime.Minute()
				} else {
					log.Warnf("Time option is not correctly formatted, must be like '00:00'. Default option will be used instead")
				}
			}

			ctx = backr.NewContextWithSettings(ctx, currentSettings)

			// handle the SIGINT signal
			waiting := make(chan os.Signal, 1)
			signal.Notify(waiting, os.Interrupt, os.Kill)

			// prepare ticker
			ticker := time.NewTicker(5 * time.Minute) // 1 minute

			go func() {
				isRunning := false

				for {
					select {
					case <-ticker.C:
						log.Debugln("Tick received")

						if !isRunning {
							isRunning = true

							// execute the update of state from specs (yml files)
							tasks.UpdateStateFromSpec(ctx)
							// execute the backup routine
							tasks.PerformBackup(ctx)

							isRunning = false
						} else {
							log.Infoln("Backup process is already running. Skipping.")
						}
					}
				}
			}()

			// start HTTP API daemon
			startAPI(ctx)

			// waiting for signal
			<-waiting
			// stop the ticker
			ticker.Stop()
			// cancelling ctx
			cancel()
			// cleanup current state storage
			state.CleanupStorage(currentSettings)

			log.Infoln("Stopped.")
		}
	})

	app.Command("get", "List the available backup archives for a project", func(cmd *cli.Cmd) {

		cmd.Spec = "NAME [--debug]"

		name := cmd.StringArg("NAME", "", "The name of the project")
		debug := cmd.BoolOpt("debug", false, "Enables the debug logs output")

		// swift
		swiftSettings := getSwiftSettings(cmd)
		if swiftSettings == nil {
			log.Errorln("Swift is not correctly configured because some args or env vars are missing.")
			cli.Exit(1)
			return
		}

		cmd.Action = func() {

			// set debug log level if needed
			if *debug {
				log.SetLevel(log.DebugLevel)
			}

			fmt.Println("Searching...")

			results, err := swift.Get(*name, *swiftSettings)
			if err != nil {
				if richErr, ok := err.(backr.UploadedArchiveError); ok {
					log.Errorln(err)
					if richErr.IsFatal {
						cli.Exit(1)
					} else {
						cli.Exit(0)
					}
				} else {
					log.Errorln(err)
					cli.Exit(1)
				}
			}

			fmt.Println("Results:")
			for _, info := range results {
				fmt.Println("------------------------------------------------------")
				fmt.Println(info)
			}

		}

	})

	app.Run(os.Args)
}

func getSwiftSettings(cmd *cli.Cmd) *backr.SwiftSettings {
	url := cmd.String(cli.StringOpt{
		Name:   "swift-auth-url",
		Value:  "",
		Desc:   "Swift auth URL",
		EnvVar: "OS_AUTH_URL",
	})
	user := cmd.String(cli.StringOpt{
		Name:   "swift-user",
		Value:  "",
		Desc:   "Swift username",
		EnvVar: "OS_USERNAME",
	})
	apiKey := cmd.String(cli.StringOpt{
		Name:   "swift-password",
		Value:  "",
		Desc:   "Swift API Key / Password",
		EnvVar: "OS_PASSWORD",
	})
	tenantName := cmd.String(cli.StringOpt{
		Name:   "swift-tenant-name",
		Value:  "",
		Desc:   "Swift Tenant name",
		EnvVar: "OS_TENANT_NAME",
	})

	if *url != "" && *user != "" && *apiKey != "" && *tenantName != "" {
		return &backr.SwiftSettings{
			AuthURL:       *url,
			User:          *user,
			APIKey:        *apiKey,
			TenantName:    *tenantName,
			ContainerName: "backups",
		}
	}

	return nil
}

func getStateStorateSettings(cmd *cli.Cmd) backr.StateStorageSettings {
	etcdEndpoints := cmd.String(cli.StringOpt{
		Name:   "etcd",
		Value:  "http://localhost:2379",
		Desc:   "Endpoints for etcd (separated by a comma)",
		EnvVar: "ETCD_ADVERTISE_URLS",
	})

	localPath := cmd.String(cli.StringOpt{
		Name:   "local",
		Value:  "",
		Desc:   "Local directory where the state will be stored (r/w permissions required)",
		EnvVar: "STATE_STORAGE_LOCAL",
	})

	return backr.StateStorageSettings{
		EtcdEndpoints: etcdEndpoints,
		LocalPath:     localPath,
	}
}

func startAPI(ctx context.Context) {
	go func() {
		api := http.NewAPI()
		api.Listen(ctx)
	}()
}
