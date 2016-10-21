package swift

import (
	"fmt"
	"os"
	"strconv"
	"time"
	"webup/backr"

	log "github.com/Sirupsen/logrus"
	"github.com/ncw/swift"
)

const (
	containerName = "backups"
)

func Upload(project backr.Project, backup backr.Backup, file string, fileExt string, settings backr.SwiftSettings) error {
	// Create a connection
	c, err := getSwiftConnection(settings)
	if err != nil {
		return err
	}

	// Check if the container for backups is created. If not, create it
	containers, err := c.ContainerNames(nil)
	if err != nil {
		return err
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
			return err
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

	return err
}
