package services

import (
	"encoding/json"
	"fmt"
	"time"
	"webup/backoops/config"
	"webup/backoops/domain"
	"webup/backoops/options"

	log "github.com/Sirupsen/logrus"
	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

const (
	periodInConfig = time.Duration(24) * time.Hour // unit of 1 day for ttl and minAge (WARNING: cannot be less (scheduling issues))
)

// type state struct {
// 	Next time.Time
// }
//
// func getNextBackupTime() time.Time {
// 	now := time.Now()
// 	// essayer le truncate pour lancer le backup toutes les 10 min. Ne pas oublier de checker ce qu'il se passe si un backup est déjà en cours
// 	return time.Date(now.Year(), now.Month(), now.Day(), now.Hour() /*+1*/, now.Minute(), now.Second()+30, 0, time.Local)
// }

// PerformBackup executes the process that start backups from a specific time, and executes the associated command
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

	// start := getNextBackupTime()
	// currentState := state{Next: start}

	startupTime := time.Now()

	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		isRunning := false

		for tickerTime := range ticker.C {

			// check if a backup is already running
			if !isRunning {

				log.Infoln("Backup iteration started.")

				isRunning = true

				// fetch all configured backups
				config, _ := etcdCli.Get(ctx, options.BackupRootDir, nil)

				if config != nil && config.Node != nil {
					for _, backupConfigKey := range config.Node.Nodes {

						key := backupConfigKey.Key

						// fetch the current state for this backup
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

						// iterate over each item
						backupDone := false
						for i := range backupState.Items {
							item := backupState.Items[i]

							// stores the current time to avoid to be impacted by the backup's command execution time
							// now := time.Now()
							// get the scheduled next time for this backup item
							nextBackup := item.GetNextBackupTime(options.StartHour, periodInConfig, startupTime)

							fmt.Println("Next:", nextBackup)

							// if the backup is needed
							if nextBackup.Before(tickerTime) {
								// check if a backup is already done with a previous item
								if !backupDone {
									log.WithFields(log.Fields{
										"key":     key,
										"ttl":     item.TimeToLive,
										"min_age": item.MinAge,
									}).Infoln("Performing backup...")

									// TODO: perform backup command
									time.Sleep(8 * time.Second)

									backupDone = true

								} else {
									log.WithFields(log.Fields{
										"key":     key,
										"ttl":     item.TimeToLive,
										"min_age": item.MinAge,
									}).Infoln("Backup already done. Skip.")
								}

								// store the backup time for this backup item
								// with the midnight time (to avoid date equality and skipping a backup unintentionally)
								item.LastBackup = tickerTime

								log.WithFields(log.Fields{
									"next": item.GetNextBackupTime(options.StartHour, periodInConfig, startupTime),
								}).Infoln("Next backup scheduled.")

								backupState.Items[i] = item
							}
						}

						backupState.IsRunning = false

						// save changes into etcd
						updateBackupStateInEtcd(ctx, etcdCli, key, backupState)

					}
				}

				// get the time for the next backup iteration to execute
				// currentState.Next = getNextBackupTime()

				isRunning = false
			} else {
				log.Infoln("Backup process is already ready")
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
