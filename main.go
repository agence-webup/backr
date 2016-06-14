package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"webup/backoops/options"
	"webup/backoops/services"

	"github.com/jawher/mow.cli"
	"github.com/ncw/swift"

	"golang.org/x/net/context"
)

func main() {
	app := cli.App("backoops", "Perform backups")

	app.Spec = "-w... [--etcd]"

	etcdEndpoints := app.String(cli.StringOpt{
		Name:   "etcd",
		Value:  "http://localhost:2379",
		Desc:   "Endpoints for etcd (separated by a comma)",
		EnvVar: "ETCD_ADVERTISE_URLS",
	})

	watchDirs := app.StringsOpt("w watch", []string{}, "Specifies the directories to watch for finding backup.yml files")

	app.Action = func() {

		ctx, cancel := context.WithCancel(context.Background())

		ctx = options.NewContext(ctx, options.Options{
			EtcdEndpoints: strings.Split(*etcdEndpoints, ","),
			WatchDirs:     *watchDirs,
			BackupRootDir: "/backups",
			StartHour:     2,
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

	app.Run(os.Args)
}
