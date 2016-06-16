package services

import (
	"encoding/json"
	"fmt"
	"time"
	"webup/backoops/config"
	"webup/backoops/domain"
	"webup/backoops/execution"
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
						project, err := backupStateFromEtcd(ctx, etcdCli, key)
						if err != nil {
							log.WithFields(log.Fields{
								"err": err,
								"key": key,
							}).Errorln("Unable to get the key in etcd")
							continue
						}

						// notify that the backup is running
						project.IsRunning = true
						updateBackupStateInEtcd(ctx, etcdCli, key, project)

						// iterate over each item
						backupDone := false
						for i := range project.Backups {
							backup := project.Backups[i]

							// stores the current time to avoid to be impacted by the backup's command execution time
							// now := time.Now()
							// get the scheduled next time for this backup item
							nextBackup := backup.GetNextBackupTime(options.StartHour, periodInConfig, startupTime)

							fmt.Println("Next:", nextBackup)

							// if the backup is needed
							if nextBackup.Before(tickerTime) {

								// prepare a log entry
								logEntry := log.WithFields(log.Fields{
									"key":     key,
									"ttl":     backup.TimeToLive,
									"min_age": backup.MinAge,
								})

								// check if a backup is already done with a previous item
								if !backupDone {
									logEntry.Infoln("Executing backup...")

									// perform backup command
									err := execution.ExecuteBackup(project, backup, options)
									if err != nil {
										logEntry.Errorln("Backup execution error:", err)
									} else {
										logEntry.Infoln("Backup execution OK")

										backupDone = true
									}

								} else {
									logEntry.Infoln("Backup already done. Skip.")
								}

								// if the backup is successful (or a previous one), store the execution time
								if backupDone {
									// store the backup time for this backup
									backup.LastExecution = tickerTime

									log.WithFields(log.Fields{
										"next": backup.GetNextBackupTime(options.StartHour, periodInConfig, startupTime),
									}).Infoln("Next backup scheduled.")
								}

								project.Backups[i] = backup
							}
						}

						project.IsRunning = false

						// save changes into etcd
						updateBackupStateInEtcd(ctx, etcdCli, key, project)

					}
				}

				// get the time for the next backup iteration to execute
				// currentState.Next = getNextBackupTime()

				isRunning = false
			} else {
				log.Infoln("Backup process is already running")
			}

		}
	}()

	log.Infoln("'Perform backup' service is started.")

	<-ctx.Done()

	ticker.Stop()
}

func backupStateFromEtcd(ctx context.Context, etcdCli etcd.KeysAPI, key string) (domain.Project, error) {
	state, err := etcdCli.Get(ctx, key, nil)
	if err != nil {
		return domain.Project{}, err
	}

	// parse JSON
	project := domain.Project{}
	json.Unmarshal([]byte(state.Node.Value), &project)

	return project, nil
}

func updateBackupStateInEtcd(ctx context.Context, etcdCli etcd.KeysAPI, key string, project domain.Project) error {
	// get json data
	jsonData, _ := json.Marshal(project)
	// set the value in etcd
	_, err := etcdCli.Set(ctx, key, string(jsonData), nil)
	return err
}
