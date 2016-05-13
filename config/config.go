package config

import (
	"io/ioutil"
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

	return backupConfig, nil
}
