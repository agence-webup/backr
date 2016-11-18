package backr

import "time"

type BackupExecution interface {
	Execute()
}

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

type UpdateReport struct {
	Created   int
	Unchanged int
	Deleted   int
}

// NewProject returns a new Project.
// Use it to initialize a new backup project.
func NewProject(spec ProjectBackupSpec) Project {
	project := Project{}
	project.Name = spec.Name

	project.Update(spec)

	return project
}

// Update a Project from a spec and log the report
func (p *Project) Update(spec ProjectBackupSpec) UpdateReport {

	report := UpdateReport{}

	// this value will be decremented for each backup found
	report.Deleted = len(p.Backups)

	// map the checksums to each item
	backupsByChecksum := map[string]Backup{}
	for i := range p.Backups {
		backupsByChecksum[p.Backups[i].Checksum] = p.Backups[i]
	}

	backups := []Backup{}
	for _, backupSpec := range spec.Backups {

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

	return report
}

// GetNextBackupTime returns the time representing the moment where the backup should be executed,
// according to the last backup time
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

// GetHealth returns the health of a backup: true is everything is OK, false otherwise
func (backup *Backup) GetHealth(timeSpec BackupTimeSpec, startupTime time.Time) bool {

	// add a tolerance of 1 hour (execution time...)
	nowWithTolerance := time.Now().Add(-10 * time.Minute)

	if backup.GetNextBackupTime(timeSpec, startupTime).Before(nowWithTolerance) {
		return false
	}

	return true
}
