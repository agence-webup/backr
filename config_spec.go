package backr

import (
	"crypto/md5"
	"fmt"
	"strconv"
)

// ProjectBackupSpec represents the content of a backup.yml file
type ProjectBackupSpec struct {
	Name    string   `yaml:"name"`
	Command []string `yaml:"command"`
	Backups []BackupSpec
}

// BackupSpec represents a backup specification
type BackupSpec struct {
	TimeToLive int `yaml:"ttl"`
	MinAge     int `yaml:"min_age"`
}

// OrderedBackupSpec allows to order the backups by TTL
type OrderedBackupSpec []BackupSpec

func (b OrderedBackupSpec) Len() int {
	return len(b)
}

func (b OrderedBackupSpec) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b OrderedBackupSpec) Less(i, j int) bool {
	return b[i].TimeToLive < b[j].TimeToLive
}

// GetChecksum returns a hash of the backup allowing to detect changes
func (b BackupSpec) GetChecksum() string {
	data := []byte(strconv.Itoa(b.TimeToLive) + strconv.Itoa(b.MinAge))
	return fmt.Sprintf("%x", md5.Sum(data))
}

// IsValid returns a boolean indicating if the parsed backup.yml is valid
func (b ProjectBackupSpec) IsValid() bool {
	if b.Name == "" {
		return false
	}

	if len(b.Backups) == 0 {
		return false
	}

	return true
}
