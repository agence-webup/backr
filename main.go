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

	"github.com/jawher/mow.cli"
	"github.com/ncw/swift"

	"golang.org/x/net/context"
)

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
					AuthURL:    *swiftURL,
					User:       *swiftUser,
					APIKey:     *swiftAPIKey,
					TenantName: *swiftTenantName,
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
			c, err := config.GetSwiftConnection(options.SwiftOptions{
				AuthURL:    *swiftURL,
				User:       *swiftUser,
				APIKey:     *swiftAPIKey,
				TenantName: *swiftTenantName,
			})
			if err != nil {
				panic(err)
			}

			objects, err := c.ObjectsAll("backups", &swift.ObjectsOpts{
				Prefix: *name,
			})
			if err != nil {
				panic(err)
			}

			if len(objects) == 0 {
				fmt.Println("No project or backup found.")
				cli.Exit(0)
			}

			_, headers, _ := c.Account()

			for i, obj := range objects {
				url := c.ObjectTempUrl("backups", obj.Name, headers["X-Account-Meta-Temp-Url-Key"], "GET", time.Now().Add(2*time.Minute))

				_, objHeaders, _ := c.Object("backups", obj.Name)
				timestamp, _ := strconv.ParseInt(objHeaders["X-Delete-At"], 10, 64)
				expire := time.Unix(timestamp, 0)

				fmt.Printf("Backup #%d\n", i+1)
				fmt.Printf("    name: %s\n", obj.Name)
				fmt.Printf(" expires: %v\n", expire)
				fmt.Printf("     url: %s\n", url)
			}
		}

	})

	app.Run(os.Args)
}
