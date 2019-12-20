package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"webup/backr"
	"webup/backr/s3"

	log "github.com/sirupsen/logrus"
)

// ExecuteBackup performs backup execution
func ExecuteBackup(project backr.Project, backup backr.Backup, returnBackupURL bool, settings backr.Settings) (*backr.UploadedArchiveInfo, error) {

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
		return nil, err
	}

	// execute the command
	err = executor.Execute(project.Dir, output)
	if err != nil {
		return nil, err
	}

	var info *backr.UploadedArchiveInfo

	// upload to S3
	if settings.S3 != nil {
		log.WithFields(log.Fields{
			"name":      project.Name,
			"s3_upload": true,
			"file":      output,
		}).Debugln("Backup file created")

		info, err = s3.Upload(project, backup, output, executor.GetOutputFileExtension(), returnBackupURL, *settings.S3)
		if err != nil {
			return nil, err
		}

		// delete the file
		os.Remove(output)
	} else {
		log.WithFields(log.Fields{
			"name":      project.Name,
			"s3_upload": false,
			"file":      output,
		}).Debugln("Backup file created")
	}

	return info, nil
}
