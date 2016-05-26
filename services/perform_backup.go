package services

import (
	"fmt"
	"time"
	"webup/backoops/config"
	"webup/backoops/options"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
)

type state struct {
	Next time.Time
}

func getNext() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()+1, 0, 0, time.Local)
}

func PerformBackup(ctx context.Context) {

	options, ok := options.FromContext(ctx)
	if !ok {
		log.Errorln("Unable to get options from context")
		return
	}

	etcdCli, err := config.GetEtcdConnection(options.EtcdEndpoints)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorln("Unable to connect to etcd")
		return
	}

	start := getNext()
	currentState := state{Next: start}

	ticker := time.NewTicker(10 * time.Second)

	go func() {
		for range ticker.C {
			if time.Now().After(currentState.Next) {
				fmt.Println("DEBUG: start backup at", currentState.Next)

				config, _ := etcdCli.Get(ctx, options.BackupRootDir, nil)

				if config != nil && config.Node != nil {
					for _, backupConfigKey := range config.Node.Nodes {

						fmt.Println(backupConfigKey)

						// if , ok := configuredBackups[existingConfigKey.Key]; !ok {
						//     log.WithFields(log.Fields{
						//         "key": existingConfigKey.Key,
						//     }).Infoln("Backup config no longer exists. Remove it from etcd.")
						//     etcdCli.Delete(ctx, existingConfigKey.Key, nil)
						// }
					}
				}

				time.Sleep(12 * time.Second)

				currentState.Next = getNext()
				fmt.Println("DEBUG: Done. Next at", currentState.Next)

			}
		}
	}()

	log.Infoln("'Perform backup' service is started.")

	<-ctx.Done()

	ticker.Stop()
}
