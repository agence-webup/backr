package backr

import (
	"context"
	"encoding/json"
)

type StateStorage interface {
	Cleanup()
}

// StateStorer defines the behaviour for interacting with a storage solution to store the state of backups
type StateStorer interface {
	StateStorage

	ConfiguredProjects(ctx context.Context) (map[string]Project, error)
	GetProject(ctx context.Context, name string) (*Project, error)
	SaveProject(ctx context.Context, project Project) error
	DeleteProject(ctx context.Context, project Project) error
}

func ProjectFromJSON(jsonData string) Project {
	project := Project{}
	json.Unmarshal([]byte(jsonData), &project)
	return project
}
