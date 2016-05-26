package domain

import (
	"time"

	log "github.com/Sirupsen/logrus"
)

// BackupState represents a backup state executed by backoops
type BackupState struct {
	Items []BackupItemState
}

// BackupItemState represents the state of a backup item
type BackupItemState struct {
	BackupSpec
	Checksum   string
	LastBackup *time.Time
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
				LastBackup: nil,
			}
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
