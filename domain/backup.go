package domain

import (
	"time"

	log "github.com/Sirupsen/logrus"
)

// BackupState represents a backup state executed by backoops
type BackupState struct {
	Items     []BackupItemState
	IsRunning bool
}

// BackupItemState represents the state of a backup item
type BackupItemState struct {
	BackupSpec
	Command    []string
	Checksum   string
	LastBackup time.Time
}

type updateReport struct {
	Created   int
	Unchanged int
	Deleted   int
}

// NewBackupState returns a new BackupState.
// Use it to initialize a new backup item.
func NewBackupState(config BackupConfig) BackupState {
	backupState := BackupState{}
	backupState.Update(config)

	return backupState
}

func getDefaultCommand(name string, outputDir string) []string {
	return []string{
		"pliz",
		"backup",
		"--files",
		"--db",
	}
}

// Update a BackupState from a config and log the report
func (b *BackupState) Update(config BackupConfig) {

	report := updateReport{}

	// this value will be decremented for each item found
	report.Deleted = len(b.Items)

	// map the checksums to each item
	itemsByChecksum := map[string]BackupItemState{}
	for i := range b.Items {
		itemsByChecksum[b.Items[i].Checksum] = b.Items[i]
	}

	items := []BackupItemState{}
	for _, backup := range config.Backups {

		checksum := backup.GetChecksum()
		var backupState BackupItemState

		// search if the item already exists
		if existingState, ok := itemsByChecksum[checksum]; ok {
			backupState = existingState

			report.Unchanged++
			report.Deleted--
		} else {
			backupState = BackupItemState{
				BackupSpec: backup,
				Checksum:   checksum,
			}

			// setup the first

			report.Created++
		}

		items = append(items, backupState)
	}

	b.Items = items

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
func (item *BackupItemState) GetNextBackupTime(startHour int, period time.Duration, startupTime time.Time) time.Time {
	// returns the date only if it's the first backup or the min age has been reached
	// force the execution at a the specified start hour, to avoid performing backup at unwanted time
	if item.LastBackup.IsZero() || item.LastBackup.Add(time.Duration(item.MinAge)*period).Before(startupTime) {
		date := time.Date(startupTime.Year(), startupTime.Month(), startupTime.Day(), startHour, 30, 0, 0, time.Local)

		// if the next date is before than the current time, then pick the next day at the same hour
		if date.Before(startupTime) {
			date = date.AddDate(0, 0, 1)
		}

		return date
	}

	date := time.Date(item.LastBackup.Year(), item.LastBackup.Month(), item.LastBackup.Day(), startHour, 30, 0, 0, time.Local)
	return date.Add(time.Duration(item.MinAge) * period)
}
