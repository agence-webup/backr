package tasks

import (
	"context"
	"time"
	"webup/backr"
	"webup/backr/archive"
	"webup/backr/state"

	"fmt"

	log "github.com/sirupsen/logrus"
)

// PerformBackup executes the process that start backups from a specific time, and executes the associated command
// returns false if a backup has failed
func PerformBackup(ctx context.Context) bool {

	opts, ok := backr.SettingsFromContext(ctx)
	if !ok {
		log.Errorln("Unable to get options from context")
		return false
	}

	// get a state storage
	stateStorage, err := state.GetStorage(opts)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorln("Unable to connect to state storage")
		return false
	}

	backupExecutionTime := time.Now()

	log.Debugln("Backup process started.")

	// fetch all configured backups
	projects, err := stateStorage.ConfiguredProjects(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorln("Unable to get configured projects from state storage")
		return false
	}

	backupFailed := false

	for _, project := range projects {

		// iterate over each item
		backupDone := false
		for i := range project.Backups {
			backup := project.Backups[i]

			// prepare a log entry
			logEntry := log.WithFields(log.Fields{
				"name":                project.Name,
				"min_age":             backup.MinAge,
				"period_unit":         backup.PeriodUnit,
				"ignore_startup_time": backup.IgnoreStartupTime,
				"last_exec":           backup.LastExecution,
			})

			logEntry.Debugln("Next execution scheduled at", backup.GetNextBackupTime(opts.TimeSpec, opts.StartupTime))

			// if the backup is needed
			if backupIsNeeded(backup, opts) {

				// check if a backup is already done with a previous item
				if !backupDone {
					logEntry.Infoln("Executing backup...")

					// perform backup command
					_, err := archive.ExecuteBackup(project, backup, false, opts)
					if err != nil {
						logEntry.Errorln("Backup execution error:", err)
						backupFailed = true
					} else {
						logEntry.Infoln("Backup execution OK")

						backupDone = true
					}

				} else {
					logEntry.Infoln("Backup already done. Skipping.")
				}

				// if the backup is successful (or a previous one), store the execution time
				if backupDone {
					// store the backup time for this backup
					backup.LastExecution = backupExecutionTime

					logEntry.WithField("next", backup.GetNextBackupTime(opts.TimeSpec, opts.StartupTime)).Infoln("Next backup scheduled.")
				}

				project.Backups[i] = backup
			}
		}

		// save changes into state storage
		err = stateStorage.SaveProject(ctx, project)
		if err != nil {
			log.WithFields(log.Fields{
				"name": project.Name,
				"err":  err,
			}).Errorln("Unable to update state in state storage")
		}

	}

	log.Debugln("Backup process finished.")

	return backupFailed
}

func PerformStandaloneBackup(ctx context.Context, projectName string) (*backr.UploadedArchiveInfo, error) {
	opts, ok := backr.SettingsFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("Unable to get options from context")
	}

	// get a state storage
	stateStorage, err := state.GetStorage(opts)
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to state storage: %v", err)
	}

	project, err := stateStorage.GetProject(ctx, projectName)
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch project from state storage: %v", err)
	}

	if project == nil {
		return nil, fmt.Errorf("Project not found")
	}

	standaloneBackup := backr.Backup{
		BackupSpec: backr.BackupSpec{
			MinAge: 1,
		},
	}

	info, err := archive.ExecuteBackup(*project, standaloneBackup, true, opts)
	if err != nil {
		return nil, fmt.Errorf("Backup execution error: %v", err)
	}

	return info, nil
}

func backupIsNeeded(backup backr.Backup, opts backr.Settings) bool {
	nextBackupTime := backup.GetNextBackupTime(opts.TimeSpec, opts.StartupTime)
	now := time.Now()

	log.WithFields(log.Fields{"next": nextBackupTime, "compare_to": now}).Debugln("Comparing dates to check if backup is needed...")

	if nextBackupTime.Before(now) || nextBackupTime.Equal(now) {
		return true
	}

	return false
}
