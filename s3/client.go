package s3

import (
	"fmt"
	"webup/backr"

	"github.com/minio/minio-go/v6"
)

func getS3Client(settings backr.S3Settings) (*minio.Client, error) {
	// Initialize minio client object.
	minioClient, err := minio.NewWithRegion(settings.Endpoint, settings.AccessKey, settings.SecretKey, settings.UseTLS, "GRA")
	if err != nil {
		return nil, fmt.Errorf("unable to initialize S3 client: %w", err)
	}

	return minioClient, nil
}
