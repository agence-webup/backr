package services

import (
	"encoding/json"
	"time"
	"webup/backoops/config"
	"webup/backoops/domain"
	"webup/backoops/options"

	log "github.com/Sirupsen/logrus"
	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

type state struct {
	Next time.Time
}

func getNext() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), now.Hour() /*now.Minute()+1*/, now.Minute(), now.Second()+30, 0, time.Local)
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
				// fmt.Println("DEBUG: start backup at", currentState.Next)

				config, _ := etcdCli.Get(ctx, options.BackupRootDir, nil)

				if config != nil && config.Node != nil {
					for _, backupConfigKey := range config.Node.Nodes {

						key := backupConfigKey.Key

						backupState, err := backupStateFromEtcd(ctx, etcdCli, key)
						if err != nil {
							log.WithFields(log.Fields{
								"err": err,
								"key": key,
							}).Errorln("Unable to get the key in etcd")
							continue
						}

						// notify that the backup is running
						backupState.IsRunning = true
						updateBackupStateInEtcd(ctx, etcdCli, key, backupState)

						backupDone := false
						for i := range backupState.Items {
							item := backupState.Items[i]
							if item.UnitsBeforeBackup == 0 {
								if !backupDone {
									log.WithFields(log.Fields{
										"key": key,
										"ttl": item.TimeToLive,
									}).Infoln("Performing backup...")

									// TODO: perform backup
									time.Sleep(12 * time.Second)

									backupDone = true

								} else {
									log.WithFields(log.Fields{
										"key": key,
										"ttl": item.TimeToLive,
									}).Infoln("Backup already done. Skip.")
								}

								item.UnitsBeforeBackup = item.MinAge - 1

							} else {
								item.UnitsBeforeBackup--
							}

							backupState.Items[i] = item
						}

						backupState.IsRunning = false

						updateBackupStateInEtcd(ctx, etcdCli, key, backupState)

						// if , ok := configuredBackups[existingConfigKey.Key]; !ok {
						//     log.WithFields(log.Fields{
						//         "key": existingConfigKey.Key,
						//     }).Infoln("Backup config no longer exists. Remove it from etcd.")
						//     etcdCli.Delete(ctx, existingConfigKey.Key, nil)
						// }
					}
				}

				currentState.Next = getNext()
				// fmt.Println("DEBUG: Done. Next at", currentState.Next)

			}
		}
	}()

	log.Infoln("'Perform backup' service is started.")

	<-ctx.Done()

	ticker.Stop()
}

func backupStateFromEtcd(ctx context.Context, etcdCli etcd.KeysAPI, key string) (domain.BackupState, error) {
	state, err := etcdCli.Get(ctx, key, nil)
	if err != nil {
		return domain.BackupState{}, err
	}

	// parse JSON
	backupState := domain.BackupState{}
	json.Unmarshal([]byte(state.Node.Value), &backupState)

	return backupState, nil
}

func updateBackupStateInEtcd(ctx context.Context, etcdCli etcd.KeysAPI, key string, backupState domain.BackupState) error {
	// get json data
	jsonData, _ := json.Marshal(backupState)
	// set the value in etcd
	_, err := etcdCli.Set(ctx, key, string(jsonData), nil)
	return err
}
