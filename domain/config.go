package domain

import (
	"crypto/md5"
	"fmt"
	"strconv"
	"time"
)

// BackupConfig represents the content of a backup.yml file
type BackupConfig struct {
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

// BackupTimeSpec specifies the time options for performing backup
type BackupTimeSpec struct {
	Hour   int
	Minute int
	Period time.Duration
}

func (b BackupSpec) GetChecksum() string {
	data := []byte(strconv.Itoa(b.TimeToLive) + strconv.Itoa(b.MinAge))
	return fmt.Sprintf("%x", md5.Sum(data))
}

// IsValid returns a boolean indicating if the parsed backup.yml is valid
func (b BackupConfig) IsValid() bool {
	if b.Name == "" {
		return false
	}

	if len(b.Backups) == 0 {
		return false
	}

	return true
}
