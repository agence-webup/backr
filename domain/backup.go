package domain

import (
	"time"

	log "github.com/Sirupsen/logrus"
)

// Project represents a backup project executed by backoops
type Project struct {
	Name    string
	Backups []Backup
	Dir     string
}

// Backup represents the state of a backup
type Backup struct {
	BackupSpec
	Command       []string
	Checksum      string
	LastExecution time.Time
}

type updateReport struct {
	Created   int
	Unchanged int
	Deleted   int
}

// NewProject returns a new Project.
// Use it to initialize a new backup project.
func NewProject(config BackupConfig) Project {
	project := Project{}
	project.Name = config.Name

	project.Update(config)

	return project
}

// Update a Project from a config and log the report
func (p *Project) Update(config BackupConfig) {

	report := updateReport{}

	// this value will be decremented for each backup found
	report.Deleted = len(p.Backups)

	// map the checksums to each item
	backupsByChecksum := map[string]Backup{}
	for i := range p.Backups {
		backupsByChecksum[p.Backups[i].Checksum] = p.Backups[i]
	}

	backups := []Backup{}
	for _, backupSpec := range config.Backups {

		checksum := backupSpec.GetChecksum()
		var backup Backup

		// search if the item already exists
		if existingBackup, ok := backupsByChecksum[checksum]; ok {
			backup = existingBackup

			report.Unchanged++
			report.Deleted--
		} else {
			backup = Backup{
				BackupSpec: backupSpec,
				Checksum:   checksum,
			}

			// setup the first

			report.Created++
		}

		backups = append(backups, backup)
	}

	p.Backups = backups

	// log only when a config has been updated
	if report.Created > 0 || report.Deleted > 0 {
		log.WithFields(log.Fields{
			"name":      config.Name,
			"created":   report.Created,
			"unchanged": report.Unchanged,
			"deleted":   report.Deleted,
		}).Infoln("Backup successfully configured")
	}

}

// GetNextBackupTime returns the time representing the moment where the backup should be executed,
// according to the last backup time
// 'period' indicates the duration used by values in backup.yml files (ttl and minAge)
func (backup *Backup) GetNextBackupTime(timeSpec BackupTimeSpec, startupTime time.Time) time.Time {
	// returns the date only if it's the first backup or the min age has been reached
	// force the execution at a the specified start hour, to avoid performing backup at unwanted time
	if backup.LastExecution.IsZero() || backup.LastExecution.Add(time.Duration(backup.MinAge)*timeSpec.Period).Before(startupTime) {
		date := time.Date(startupTime.Year(), startupTime.Month(), startupTime.Day(), timeSpec.Hour, timeSpec.Minute, 0, 0, time.Local)

		// if the next date is before than the current time, then pick the next day at the same hour
		if date.Before(startupTime) {
			date = date.AddDate(0, 0, 1)
		}

		return date
	}

	date := time.Date(backup.LastExecution.Year(), backup.LastExecution.Month(), backup.LastExecution.Day(), timeSpec.Hour, timeSpec.Minute, 0, 0, time.Local)
	return date.Add(time.Duration(backup.MinAge) * timeSpec.Period)
}
