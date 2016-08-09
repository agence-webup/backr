package state

import (
	"encoding/json"
	"fmt"
	"webup/backoops/domain"
	"webup/backoops/options"

	"golang.org/x/net/context"
)

// Storer defines the behaviour for interacting with a storage solution to store the state of backups
type Storer interface {
	ConfiguredProjects(ctx context.Context) (map[string]domain.Project, error)
	GetProject(ctx context.Context, name string) (*domain.Project, error)
	SaveProject(ctx context.Context, project domain.Project) error
	DeleteProject(ctx context.Context, project domain.Project) error
}

func CleanupStorage(opts options.Options) {
	if opts.StateStorage.GetType() == options.StateStorageLocal {
		CleanupBoltStorage(opts)
	}
}

func GetStorage(opts options.Options) (Storer, error) {
	if opts.StateStorage.GetType() == options.StateStorageLocal {
		return NewBoltStorage(opts)
	} else if opts.StateStorage.GetType() == options.StateStorageEtcd {
		return NewEtcdStorage(opts)
	}

	return nil, fmt.Errorf("Unable to detect the state storage")
}

func getProjectFromJSON(jsonData string) domain.Project {
	project := domain.Project{}
	json.Unmarshal([]byte(jsonData), &project)
	return project
}
