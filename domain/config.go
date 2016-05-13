package domain

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
