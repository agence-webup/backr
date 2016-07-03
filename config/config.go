package config

import (
	"io/ioutil"
	"sort"
	"webup/backoops/domain"

	"gopkg.in/yaml.v2"
)

// ParseConfigFile parses a backup.yml file
func ParseConfigFile(filepath string) (domain.BackupConfig, error) {
	backupConfig := domain.BackupConfig{}

	fileContent, err := ioutil.ReadFile(filepath)
	if err != nil {
		return backupConfig, err
	}

	err = yaml.Unmarshal(fileContent, &backupConfig)
	if err != nil {
		return backupConfig, err
	}

	// sort the backups by TTL
	backups := backupConfig.Backups
	sort.Sort(sort.Reverse(domain.OrderedBackupSpec(backups)))
	backupConfig.Backups = backups

	return backupConfig, nil
}
