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

	app.Command("start", "Start the backup process", func(cmd *cli.Cmd) {

		cmd.Spec = "-w... [--etcd]"

		etcdEndpoints := cmd.String(cli.StringOpt{
			Name:   "etcd",
			Value:  "http://localhost:2379",
			Desc:   "Endpoints for etcd (separated by a comma)",
			EnvVar: "ETCD_ADVERTISE_URLS",
		})

		watchDirs := cmd.StringsOpt("w watch", []string{}, "Specifies the directories to watch for finding backup.yml files")

		cmd.Action = func() {

			ctx, cancel := context.WithCancel(context.Background())

			ctx = options.NewContext(ctx, options.Options{
				EtcdEndpoints: strings.Split(*etcdEndpoints, ","),
				WatchDirs:     *watchDirs,
				BackupRootDir: "/backups",
				StartHour:     1,
				Swift: options.SwiftOptions{
					AuthURL:       *swiftURL,
					User:          *swiftUser,
					APIKey:        *swiftAPIKey,
					TenantName:    *swiftTenantName,
					ContainerName: swiftContainerName,
				},
			})

			// handle the SIGINT signal
			waiting := make(chan os.Signal, 1)
			signal.Notify(waiting, os.Interrupt)

			// start backup fetching daemon
			go services.FetchBackupConfig(ctx)
			// start backup routine
			go services.PerformBackup(ctx)

			// waiting for signal
			<-waiting

			// cancelling ctx
			cancel()

			fmt.Println("\n Exiting.")
		}
	})

	app.Command("get", "Fetch the backups for a project", func(cmd *cli.Cmd) {

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

	app.Run(os.Args)
}
