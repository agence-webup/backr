package swift

import (
	"fmt"
	"os"
	"strconv"
	"time"
	"webup/backr"

	"github.com/ncw/swift"
	log "github.com/sirupsen/logrus"
)

const (
	containerName = "backups"
)

// Create a connection
func Upload(project backr.Project, backup backr.Backup, file string, fileExt string, returnBackupURL bool, settings backr.SwiftSettings) (*backr.UploadedArchiveInfo, error) {
	c, err := getSwiftConnection(settings)
	if err != nil {
		return nil, err
	}

	// Check if the container for backups is created. If not, create it
	containers, err := c.ContainerNames(nil)
	if err != nil {
		return nil, err
	}
	found := false
	log.WithFields(log.Fields{"container_name": containerName}).Debugln("Trying to find the backup container...")
	for _, container := range containers {
		if container == containerName {
			found = true
			break
		}
	}
	if !found {
		log.WithFields(log.Fields{"container_name": containerName}).Debugln("Container not found. Create it.")
		err = c.ContainerCreate(containerName, nil)
		if err != nil {
			return nil, err
		}
	}

	filename := fmt.Sprintf("%s/%s.%s", project.Name, time.Now().Format(time.RFC3339), fileExt)

	reader, _ := os.Open(file)
	defer reader.Close()

	expire := (time.Duration(backup.TimeToLive) * time.Duration(24) * time.Hour).Seconds()
	headers := swift.Headers{
		"X-Delete-After": strconv.Itoa(int(expire)),
	}

	log.WithFields(log.Fields{
		"container_name": containerName,
		"file":           filename,
		"x-delete-after": expire,
	}).Debugln("Uploading to Swift...")

	_, err = c.ObjectPut(containerName, filename, reader, true, "", "", headers)

	info := backr.UploadedArchiveInfo{
		Name: filename,
	}

	if returnBackupURL {
		// fetch the headers for the Account (allowing to get the key for temp urls)
		_, headers, accountErr := c.Account()
		// generate a temp url for download
		if accountErr == nil {
			info.URL = c.ObjectTempUrl(settings.ContainerName, filename, headers["X-Account-Meta-Temp-Url-Key"], "GET", time.Now().Add(10*time.Minute))
		}
	}

	return &info, err
}
