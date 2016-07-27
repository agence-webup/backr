package services

import (
	"encoding/json"
	"time"
	"webup/backoops/config"
	"webup/backoops/domain"
	"webup/backoops/execution"
	"webup/backoops/options"

	log "github.com/Sirupsen/logrus"
	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

// PerformBackup executes the process that start backups from a specific time, and executes the associated command
func PerformBackup(ctx context.Context, running chan<- map[string]bool) {

	opts, ok := options.FromContext(ctx)
	if !ok {
		log.Errorln("Unable to get options from context")
		return
	}

	etcdCli, err := config.GetEtcdConnection(opts.EtcdEndpoints)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorln("Unable to connect to etcd")
		return
	}

	runningBackups := make(map[string]bool)

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
				config, _ := etcdCli.Get(ctx, opts.BackupRootDir, nil)

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
						runningBackups[project.Name] = true
						running <- runningBackups

						// iterate over each item
						backupDone := false
						for i := range project.Backups {
							backup := project.Backups[i]

							// get the scheduled next time for this backup item
							nextBackup := backup.GetNextBackupTime(opts.TimeSpec, startupTime)

							// prepare a log entry
							logEntry := log.WithFields(log.Fields{
								"key":     key,
								"ttl":     backup.TimeToLive,
								"min_age": backup.MinAge,
							})

							logEntry.Debugln("Next scheduled at", nextBackup)

							// if the backup is needed
							if nextBackup.Before(tickerTime) {

								// check if a backup is already done with a previous item
								if !backupDone {
									logEntry.Infoln("Executing backup...")

									// perform backup command
									err := execution.ExecuteBackup(project, backup, opts)
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
										"next": backup.GetNextBackupTime(opts.TimeSpec, startupTime),
									}).Infoln("Next backup scheduled.")
								}

								project.Backups[i] = backup
							}
						}

						// remove the project from the running backups
						delete(runningBackups, project.Name)
						running <- runningBackups

						// save changes into etcd
						err = updateBackupStateInEtcd(ctx, etcdCli, key, project)
						if err != nil {
							log.WithFields(log.Fields{
								"key": key,
								"err": err,
							}).Errorln("Unable to update state in etcd")
						}

					}
				}

				isRunning = false

				log.Infoln("Backup iteration finished.")

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
	err = json.Unmarshal([]byte(state.Node.Value), &project)

	return project, err
}

func updateBackupStateInEtcd(ctx context.Context, etcdCli etcd.KeysAPI, key string, project domain.Project) error {
	// get json data
	jsonData, _ := json.Marshal(project)
	// set the value in etcd
	_, err := etcdCli.Set(ctx, key, string(jsonData), nil)
	return err
}
