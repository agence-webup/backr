package execution

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"webup/backoops/domain"
)

type Executor interface {
	GetOutputFileExtension() string
	Execute(workingDir string, output string) error
}

func ExecuteBackup(project domain.Project, backup domain.Backup) error {

	tmpDir := "._tmp"

	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		os.MkdirAll(tmpDir, os.ModePerm)
	}

	executor := Pliz{}

	outputFile := fmt.Sprintf("%d.%s", time.Now().Unix(), executor.GetOutputFileExtension())
	output, err := filepath.Abs(filepath.Join(tmpDir, outputFile))
	if err != nil {
		return err
	}

	err = executor.Execute(project.Dir, output)

	return err
}
