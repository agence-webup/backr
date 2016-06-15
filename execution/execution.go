package execution

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"webup/backoops/domain"
	"webup/backoops/options"

	"github.com/ncw/swift"
)

const (
	containerName = "backups"
)

type Executor interface {
	GetOutputFileExtension() string
	Execute(workingDir string, output string) error
}

func ExecuteBackup(project domain.Project, backup domain.Backup, options options.Options) error {

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
	if err != nil {
		return err
	}

	uploadToSwift(project, backup, output, executor.GetOutputFileExtension(), options)

	return err
}

func uploadToSwift(project domain.Project, backup domain.Backup, file string, fileExt string, options options.Options) error {
	// Create a connection
	c := swift.Connection{
		UserName: options.Swift.User,
		ApiKey:   options.Swift.APIKey,
		AuthUrl:  options.Swift.AuthURL,
		Tenant:   options.Swift.TenantName, // Name of the tenant (v2 auth only)
	}
	// Authenticate
	err := c.Authenticate()
	if err != nil {
		return err
	}

	// Check if the container for backups is created. If not, create it
	containers, err := c.ContainerNames(nil)
	if err != nil {
		return err
	}
	found := false
	for _, container := range containers {
		if container == containerName {
			found = true
			break
		}
	}
	if !found {
		err = c.ContainerCreate(containerName, nil)
		if err != nil {
			return err
		}
	}

	filename := fmt.Sprintf("%s/%s.%s", project.Name, time.Now().Format(time.RFC3339), fileExt)

	reader, _ := os.Open(file)
	defer reader.Close()
	headers := swift.Headers{
		"X-Delete-After": strconv.Itoa(backup.TimeToLive * 86400),
	}
	_, err = c.ObjectPut(containerName, filename, reader, true, "", "", headers)
	if err != nil {
		return err
	}

	return nil
}
