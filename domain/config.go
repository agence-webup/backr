package domain

import (
	"crypto/md5"
	"fmt"
	"strconv"
)

// BackupConfig represents the content of a backup.yml file
type BackupConfig struct {
	Name    string `yaml:"name"`
	Backups []BackupSpec
}

// BackupSpec represents a backup specification
type BackupSpec struct {
	TimeToLive int `yaml:"ttl"`
	MinAge     int `yaml:"min_age"`
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
