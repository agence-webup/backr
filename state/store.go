package state

import (
	"webup/backoops/domain"
	"webup/backoops/options"

	"golang.org/x/net/context"
)

// Storer defines the complete solution to store the state
type Storer interface {
	ConfigStorer
}

// ConfigStorer defines the behaviour for interacting with a storage solution to store the state of backup config
type ConfigStorer interface {
	ConfiguredProjects(ctx context.Context) (map[string]domain.Project, error)
	SaveProject(ctx context.Context, project domain.Project) error
	DeleteProject(ctx context.Context, project domain.Project) error
}

func GetStorage(opts options.Options) (Storer, error) {
	return NewEtcdStorage(opts)
}
