package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"webup/backr"
	"webup/backr/swift"

	log "github.com/sirupsen/logrus"
)

// ExecuteBackup performs backup execution
func ExecuteBackup(project backr.Project, backup backr.Backup, settings backr.Settings) error {

	tmpDir := "._tmp"

	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		os.MkdirAll(tmpDir, os.ModePerm)
	}

	var executor backr.Executor

	// archiver
	if project.Archiver.Type == "stdout" {
		executor = Stdout{
			OutputFileExtension: project.Archiver.OutputFileExtension,
			Command:             project.Archiver.Command,
		}
	} else {
		executor = Pliz{}
	}

	outputFile := fmt.Sprintf("%d.%s", time.Now().Unix(), executor.GetOutputFileExtension())
	output, err := filepath.Abs(filepath.Join(tmpDir, outputFile))
	if err != nil {
		return err
	}

	// execute the command
	err = executor.Execute(project.Dir, output)
	if err != nil {
		return err
	}

	// upload to swift
	if settings.Swift != nil {
		log.WithFields(log.Fields{
			"name":         project.Name,
			"swift_upload": true,
			"file":         output,
		}).Debugln("Backup file created")

		err = swift.Upload(project, backup, output, executor.GetOutputFileExtension(), *settings.Swift)
		if err != nil {
			return err
		}

		// delete the file
		os.Remove(output)
	} else {
		log.WithFields(log.Fields{
			"name":         project.Name,
			"swift_upload": false,
			"file":         output,
		}).Debugln("Backup file created")
	}

	return nil
}
