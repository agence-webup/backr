package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"
	"webup/backr"
	"webup/backr/http"
	"webup/backr/privatehttp"
	"webup/backr/randstr"
	"webup/backr/state"
	"webup/backr/tasks"

	"io/ioutil"

	log "github.com/Sirupsen/logrus"
	jwt "github.com/dgrijalva/jwt-go"
	cli "github.com/jawher/mow.cli"
	homedir "github.com/mitchellh/go-homedir"
)

func main() {
	app := cli.App("backr", "Perform backups")

	app.Version("v version", "Backr 3 (build 8)")

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})

	app.Command("daemon", "Start the backup process", func(cmd *cli.Cmd) {

		// cmd.Spec = "-w... --etcd|--local [--time] [--config-refresh-rate]"
		cmd.Spec = "-w... --etcd|--local [--time] [--secret-file-path] [--api-listen] [--debug]"

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
			currentSettings.Swift = swiftSettings
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

			// start HTTP API daemons
			startAPI(ctx)
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

	app.Command("token", "Create a token to access to backup archives", func(cmd *cli.Cmd) {

		cmd.Spec = "[--secret-file-path] [-q]"

		secretFilePath := cmd.StringOpt("secret-file-path", "~/.backr/jwt_secret", "Path to the file storing the secret used for generating access token to backup files")
		quiet := cmd.BoolOpt("q quiet", false, "Just display the token")

		cmd.Action = func() {

			filepath, _ := homedir.Expand(*secretFilePath)

			secret, err := ioutil.ReadFile(filepath)
			if err != nil {
				if !*quiet {
					fmt.Println("Secret file not found. Create it at", filepath)
				}
				secret = randstr.SecureRandomBytes(64)
				err := ioutil.WriteFile(filepath, secret, 0600)
				if err != nil {
					fmt.Println("Unable to create secret file")
					fmt.Println(err)
					cli.Exit(1)
				}
			}

			// Create a new token object, specifying signing method and the claims
			// you would like it to contain.
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"exp": time.Now().Add(24 * time.Hour).Unix(),
			})

			// Sign and get the complete encoded token as a string using the secret
			tokenString, err := token.SignedString(secret)
			if err != nil {
				fmt.Println("Unable to generate token")
				fmt.Println(err)
				cli.Exit(1)
			}

			fmt.Println(tokenString)
		}

	})

	app.Command("now", "Execute a backup immediately", func(cmd *cli.Cmd) {

		cmd.Spec = "[--url] PROJECT_NAME"

		url := cmd.StringOpt("url", "http://127.0.0.1:22258", "URL of private API")
		projectName := cmd.StringArg("PROJECT_NAME", "", "A project name configured inside backr")

		cmd.Action = func() {
			client := privatehttp.NewClient(*url)
			err := client.Backup(*projectName)
			if err != nil {
				fmt.Printf("ERROR: %v\n", err)
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

func startPrivateAPI(ctx context.Context) {
	go func() {
		api := privatehttp.NewAPI()
		api.Listen(ctx)
	}()
}
