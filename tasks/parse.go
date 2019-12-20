package tasks

import (
	"io/ioutil"
	"webup/backr"

	yaml "gopkg.in/yaml.v2"
)

// ParseSpecFile parses a backup.yml file
func parseSpecFile(filepath string) (backr.ProjectBackupSpec, error) {
	projectSpec := backr.ProjectBackupSpec{}

	fileContent, err := ioutil.ReadFile(filepath)
	if err != nil {
		return projectSpec, err
	}

	err = yaml.Unmarshal(fileContent, &projectSpec)
	if err != nil {
		return projectSpec, err
	}

	return projectSpec, nil
}
