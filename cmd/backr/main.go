package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"
	"webup/backr"
	"webup/backr/privatehttp"
	"webup/backr/state"
	"webup/backr/tasks"

	cli "github.com/jawher/mow.cli"
	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
)

func main() {
	app := cli.App("backr", "Perform backups")

	app.Version("v version", "Backr 5 (build 10)")

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})

	app.Command("daemon", "Start the backup process", func(cmd *cli.Cmd) {

		// cmd.Spec = "-w... --etcd|--local [--time] [--config-refresh-rate]"
		cmd.Spec = "-w... --etcd|--local [--time] [--secret-file-path] [--api-listen] [--debug]"

		// state storage
		stateStorageSettings := getStateStorateSettings(cmd)

		// S3
		s3Settings := getS3Settings(cmd)
		if s3Settings == nil {
			log.Warnln("S3 upload will be unavailable because some args or env vars are missing to configure S3 upload")
		}

		// options
		watchDirs := cmd.StringsOpt("w watch", []string{}, "Specifies the directories to watch for finding backup.yml files")
		timeOpt := cmd.StringOpt("time", "01:00", "Specifies the moment when the backup process will be started")
		secretFilePath := cmd.StringOpt("secret-file-path", "~/.backr/jwt_secret", "Path to the file storing the secret used for generating access token to backup files")
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
			currentSettings.S3 = s3Settings
			currentSettings.ApiListen = *apiListenOpt

			path, _ := homedir.Expand(*secretFilePath)
			currentSettings.SecretFilepath = path

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
			ticker := time.NewTicker(5 * time.Minute)

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

			// start HTTP API daemons
			startPrivateAPI(ctx)

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

	app.Command("now", "Execute a backup immediately", func(cmd *cli.Cmd) {

		cmd.Spec = "[--url] PROJECT_NAME"

		url := cmd.StringOpt("url", "http://127.0.0.1:22258", "URL of private API")
		projectName := cmd.StringArg("PROJECT_NAME", "", "A project name configured inside backr")

		cmd.Action = func() {
			client := privatehttp.NewClient(*url)
			info, err := client.Backup(*projectName)
			if err != nil {
				fmt.Printf("ERROR: %v\n", err)
			}

			fmt.Println("name:", info.Name)
			fmt.Println("url:", info.URL)
		}

	})

	app.Run(os.Args)
}

func getS3Settings(cmd *cli.Cmd) *backr.S3Settings {
	bucket := cmd.String(cli.StringOpt{
		Name:   "s3-bucket",
		Value:  "",
		Desc:   "S3 bucket name",
		EnvVar: "S3_BUCKET",
	})
	endpoint := cmd.String(cli.StringOpt{
		Name:   "s3-endpoint",
		Value:  "",
		Desc:   "S3 API endpoint",
		EnvVar: "S3_ENDPOINT",
	})
	accessKey := cmd.String(cli.StringOpt{
		Name:   "s3-access-key",
		Value:  "",
		Desc:   "S3 Access Key",
		EnvVar: "S3_ACCESS_KEY",
	})
	secretKey := cmd.String(cli.StringOpt{
		Name:   "s3-secret-key",
		Value:  "",
		Desc:   "S3 Secret Key",
		EnvVar: "S3_SECRET_KEY",
	})
	useTLS := cmd.Bool(cli.BoolOpt{
		Name:   "s3-use-tls",
		Value:  true,
		Desc:   "Use TLS to connect to S3 API",
		EnvVar: "S3_USE_TLS",
	})

	if *bucket != "" && *endpoint != "" && *accessKey != "" && *secretKey != "" {
		return &backr.S3Settings{
			Bucket:    *bucket,
			Endpoint:  *endpoint,
			AccessKey: *accessKey,
			SecretKey: *secretKey,
			UseTLS:    *useTLS,
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

func startPrivateAPI(ctx context.Context) {
	go func() {
		api := privatehttp.NewAPI()
		api.Listen(ctx)
	}()
}
