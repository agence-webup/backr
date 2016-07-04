package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
	"webup/backoops/config"
	"webup/backoops/options"
	"webup/backoops/services"

	log "github.com/Sirupsen/logrus"
	"github.com/jawher/mow.cli"
	"github.com/ncw/swift"

	"golang.org/x/net/context"
)

const (
	swiftContainerName = "backups"
	etcdRootDir        = "/backups"
)

type backupInfo struct {
	Name   string
	Expire time.Time
	URL    string
}

func (info backupInfo) String() string {
	return fmt.Sprintf("    name: %s\n", info.Name) +
		fmt.Sprintf(" expires: %v\n", info.Expire) +
		fmt.Sprintf("     url: %s\n", info.URL)
}

func main() {
	app := cli.App("backoops", "Perform backups")

	swiftURL := app.String(cli.StringOpt{
		Name:   "swift-auth-url",
		Value:  "",
		Desc:   "Swift auth URL",
		EnvVar: "OS_AUTH_URL",
	})
	swiftUser := app.String(cli.StringOpt{
		Name:   "swift-user",
		Value:  "",
		Desc:   "Swift username",
		EnvVar: "OS_USERNAME",
	})
	swiftAPIKey := app.String(cli.StringOpt{
		Name:   "swift-password",
		Value:  "",
		Desc:   "Swift API Key / Password",
		EnvVar: "OS_PASSWORD",
	})
	swiftTenantName := app.String(cli.StringOpt{
		Name:   "swift-tenant-name",
		Value:  "",
		Desc:   "Swift Tenant name",
		EnvVar: "OS_TENANT_NAME",
	})

	app.Command("daemon", "Start the backup process", func(cmd *cli.Cmd) {

		cmd.Spec = "-w... [--etcd] [--time]"

		etcdEndpoints := getEtcdOptionsFromCli(cmd)

		watchDirs := cmd.StringsOpt("w watch", []string{}, "Specifies the directories to watch for finding backup.yml files")
		timeOpt := cmd.StringOpt("time", "01:00", "Specifies the moment when the backup process will be started")

		cmd.Action = func() {

			ctx, cancel := context.WithCancel(context.Background())

			// prepare options
			opts := options.NewDefaultOptions()
			opts.EtcdEndpoints = strings.Split(*etcdEndpoints, ",")
			opts.WatchDirs = *watchDirs
			opts.Swift = options.SwiftOptions{
				AuthURL:       *swiftURL,
				User:          *swiftUser,
				APIKey:        *swiftAPIKey,
				TenantName:    *swiftTenantName,
				ContainerName: swiftContainerName,
			}

			// parse the time option
			if timeOpt != nil {
				parsedTime, err := time.Parse("15:04", *timeOpt)
				if err == nil {
					opts.TimeSpec.Hour = parsedTime.Hour()
					opts.TimeSpec.Minute = parsedTime.Minute()
				} else {
					log.Warnf("Time option is not correctly formatted, must be like '00:00'. Default option will be used instead")
				}
			}

			ctx = options.NewContext(ctx, opts)

			// handle the SIGINT signal
			waiting := make(chan os.Signal, 1)
			signal.Notify(waiting, os.Interrupt)

			// store the state of the running backups (by project)
			runningState := make(chan map[string]bool)

			// start backup fetching daemon
			go services.FetchBackupConfig(ctx, runningState)
			// start backup routine
			go services.PerformBackup(ctx, runningState)

			// waiting for signal
			<-waiting

			// cancelling ctx
			cancel()

			// check https://blog.golang.org/pipelines
			// for {
			// 	select {
			// 	case <-waiting:
			// 		cancel()
			// 	case <-ctx.Done():
			// 		fmt.Println("\n Exiting.")
			// 		return
			// 	}

			// }
		}
	})

	app.Command("get", "List the available backup archives for a project", func(cmd *cli.Cmd) {

		cmd.Spec = "NAME"

		name := cmd.StringArg("NAME", "", "The name of the project")

		cmd.Action = func() {

			fmt.Println("Searching...")

			swiftOptions := options.SwiftOptions{
				AuthURL:       *swiftURL,
				User:          *swiftUser,
				APIKey:        *swiftAPIKey,
				TenantName:    *swiftTenantName,
				ContainerName: swiftContainerName,
			}

			// get a swift connection
			c, err := config.GetSwiftConnection(swiftOptions)
			if err != nil {
				log.Errorln(err)
				cli.Exit(1)
				return
			}

			// fetch all backups with the name starting with the term passed as param
			objects, err := c.ObjectsAll(swiftOptions.ContainerName, &swift.ObjectsOpts{
				Prefix: *name,
			})
			if err != nil {
				log.Errorln(err)
				cli.Exit(1)
				return
			}

			if len(objects) == 0 {
				fmt.Println("No project or backup found.")
				cli.Exit(0)
				return
			}

			// fetch the headers for the Account (allowing to get the key for temp urls)
			_, headers, accountErr := c.Account()

			fmt.Println("Results:")

			results := make(chan backupInfo)

			for _, obj := range objects {

				go func(obj swift.Object) {
					// prepare info for this backup
					info := backupInfo{
						Name: obj.Name,
					}

					// get the expire time
					_, objHeaders, objErr := c.Object(swiftOptions.ContainerName, obj.Name)
					if objErr == nil {
						if deleteAt, ok := objHeaders["X-Delete-At"]; ok {
							timestamp, _ := strconv.ParseInt(deleteAt, 10, 64)
							expire := time.Unix(timestamp, 0)
							info.Expire = expire
						}
					}

					// generate a temp url for download
					if accountErr == nil {
						info.URL = c.ObjectTempUrl(swiftOptions.ContainerName, obj.Name, headers["X-Account-Meta-Temp-Url-Key"], "GET", time.Now().Add(2*time.Minute))
					}

					results <- info

				}(obj)
			}

			for i := 0; i < len(objects); i++ {
				info := <-results
				fmt.Println("------------------------------------------------------")
				fmt.Println(info)
			}
		}

	})

	app.Command("config", "Enable, disable or check the status of a backup config", func(cmd *cli.Cmd) {

		etcdEndpoints := getEtcdOptionsFromCli(cmd)

		cmd.Before = func() {
			// check for a backup.yml file
			if _, err := os.Stat("backup.yml"); os.IsNotExist(err) {
				fmt.Println("'backup.yml' file not found in the current directory")
				cli.Exit(1)
			}
		}

		cmd.Command("status", "Display the status of the config", func(subcmd *cli.Cmd) {

			subcmd.Action = func() {
				ctx := context.Background()

				opts := options.NewDefaultOptions()
				opts.EtcdEndpoints = strings.Split(*etcdEndpoints, ",")

				ctx = options.NewContext(ctx, opts)

				services.StatusBackupConfig(ctx)
			}
		})

	})

	app.Run(os.Args)
}

func getEtcdOptionsFromCli(cmd *cli.Cmd) *string {
	return cmd.String(cli.StringOpt{
		Name:   "etcd",
		Value:  "http://localhost:2379",
		Desc:   "Endpoints for etcd (separated by a comma)",
		EnvVar: "ETCD_ADVERTISE_URLS",
	})
}
