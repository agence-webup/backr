package state

import (
	"encoding/json"
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

	CleanUp()
}

func GetStorage(opts options.Options) (Storer, error) {
	// return NewEtcdStorage(opts)
	return NewBoltStorage(opts)
}

func getProjectFromJSON(jsonData string) domain.Project {
	project := domain.Project{}
	json.Unmarshal([]byte(jsonData), &project)
	return project
}
