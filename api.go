package backr

import (
	"context"
)

type API interface {
	Listen(ctx context.Context) error
}

type PrivateAPI interface {
	Listen(ctx context.Context) error
}

type PrivateAPIClient interface {
	Backup(projectName string) (*UploadedArchiveInfo, error)
}
