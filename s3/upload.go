package s3

import (
	"fmt"
	"time"
	"webup/backr"

	"github.com/minio/minio-go/v6"
	log "github.com/sirupsen/logrus"
)

// Upload is responsible to upload a backup file to a S3 storage
func Upload(project backr.Project, backup backr.Backup, file string, fileExt string, returnBackupURL bool, settings backr.S3Settings) (*backr.UploadedArchiveInfo, error) {
	c, err := getS3Client(settings)
	if err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("%s/%s.%s", project.Name, time.Now().Format(time.RFC3339), fileExt)

	info := backr.UploadedArchiveInfo{
		Name: filename,
	}

	log.WithFields(log.Fields{
		"bucket": settings.Bucket,
		"file":   filename,
	}).Debugln("Uploading to S3...")

	n, err := c.FPutObject(settings.Bucket, filename, file, minio.PutObjectOptions{})
	if err != nil {
		return &info, fmt.Errorf("unable to upload file to S3: %w", err)
	}

	log.WithFields(log.Fields{
		"bucket": settings.Bucket,
		"file":   filename,
		"size":   n,
	}).Debugln("file successfully uploaded to S3")

	if returnBackupURL {
		url, err := c.PresignedGetObject(settings.Bucket, filename, 10*time.Minute, nil)
		if err != nil {
			log.WithFields(log.Fields{
				"bucket": settings.Bucket,
				"file":   filename,
			}).Debugln("unable to generate a presigned URL for a S3 file")
		} else {
			info.URL = url.String()
		}
	}

	return &info, nil
}
