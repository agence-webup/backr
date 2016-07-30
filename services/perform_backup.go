package services

import (
	"time"
	"webup/backoops/execution"
	"webup/backoops/options"
	"webup/backoops/state"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
)

// PerformBackup executes the process that start backups from a specific time, and executes the associated command
func PerformBackup(ctx context.Context, running chan<- map[string]bool) {

	opts, ok := options.FromContext(ctx)
	if !ok {
		log.Errorln("Unable to get options from context")
		return
	}

	// get a state storage
	stateStorage, err := state.GetStorage(opts)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorln("Unable to connect to state storage")
		return
	}

	runningBackups := make(map[string]bool)

	startupTime := time.Now()
	ticker := time.NewTicker(3 * time.Minute)

	go func() {
		isRunning := false

		for tickerTime := range ticker.C {

			// check if a backup is already running
			if !isRunning {

				log.Infoln("Backup iteration started.")

				isRunning = true

				// fetch all configured backups
				projects, err := stateStorage.ConfiguredProjects(ctx)
				if err != nil {
					log.WithFields(log.Fields{
						"err": err,
					}).Errorln("Unable to get configured projects from state storage")
					continue
				}

				for _, project := range projects {

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
							"name":    project.Name,
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

					// save changes into state storage
					err = stateStorage.SaveProject(ctx, project)
					if err != nil {
						log.WithFields(log.Fields{
							"name": project.Name,
							"err":  err,
						}).Errorln("Unable to update state in state storage")
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
