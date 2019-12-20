package backr

import (
	"crypto/md5"
	"errors"
	"fmt"
	"strconv"
)

// ProjectBackupSpec represents the content of a backup.yml file
type ProjectBackupSpec struct {
	Name     string `yaml:"name"`
	Backups  []BackupSpec
	Archiver *Archiver `yaml:"archiver"`
}

// BackupSpec represents a backup specification
type BackupSpec struct {
	MinAge            int  `yaml:"min_age"`
	PeriodUnit        int  `yaml:"period_unit"` // unit for 'min_age', in hours (default to 24h)
	IgnoreStartupTime bool `yaml:"ignore_startup_time"`
}

type Archiver struct {
	Type                string   `yaml:"type"`
	OutputFileExtension string   `yaml:"ext"`
	Command             []string `yaml:"command"`
}

// GetChecksum returns a hash of the backup allowing to detect changes
func (b BackupSpec) GetChecksum() string {
	data := []byte(strconv.Itoa(b.PeriodUnit) + strconv.Itoa(b.MinAge) + strconv.FormatBool(b.IgnoreStartupTime))
	return fmt.Sprintf("%x", md5.Sum(data))
}

// IsValid returns a boolean indicating if the parsed backup.yml is valid
func (b ProjectBackupSpec) IsValid() error {
	if b.Name == "" {
		return errors.New("'name' is required")
	}

	if b.Archiver != nil {
		archiver := *b.Archiver
		if len(archiver.Command) == 0 || archiver.Type != "pliz" && archiver.Type != "stdout" {
			return errors.New("'archiver' type must be 'pliz' or 'stdout', 'command' and 'ext' are required")
		}
	}

	if len(b.Backups) == 0 {
		return errors.New("'backups' cannot be empty")
	}

	return nil
}
