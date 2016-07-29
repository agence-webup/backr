package state

import (
	"encoding/json"
	"strings"
	"time"
	"webup/backoops/domain"

	"golang.org/x/net/context"

	"webup/backoops/options"

	etcd "github.com/coreos/etcd/client"
)

// EtcdStorage implements the Storer interface to store the state in etcd
type EtcdStorage struct {
	client  etcd.KeysAPI
	rootDir string
}

func NewEtcdStorage(opts options.Options) (*EtcdStorage, error) {
	cfg := etcd.Config{
		Endpoints: opts.EtcdEndpoints,
		Transport: etcd.DefaultTransport,
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: 3 * time.Second,
	}

	c, err := etcd.New(cfg)
	if err != nil {
		return nil, err
	}

	etcdCli := etcd.NewKeysAPI(c)

	return &EtcdStorage{
		client:  etcdCli,
		rootDir: opts.BackupRootDir,
	}, nil
}

func (s *EtcdStorage) ConfiguredProjects(ctx context.Context) (map[string]domain.Project, error) {
	data, err := s.client.Get(ctx, s.rootDir, nil)

	projects := map[string]domain.Project{}

	if data != nil && data.Node != nil {
		for _, projectData := range data.Node.Nodes {
			name := projectData.Key
			name = strings.Replace(name, s.rootDir+"/", "", -1)

			project := domain.Project{}
			json.Unmarshal([]byte(projectData.Value), &project)

			projects[name] = project
		}
	}

	return projects, err
}

func (s *EtcdStorage) SaveProject(ctx context.Context, project domain.Project) error {
	// get json data
	jsonData, _ := json.Marshal(project)
	// set the value in etcd
	_, err := s.client.Set(ctx, s.getKey(project.Name), string(jsonData), nil)
	return err
}

func (s *EtcdStorage) DeleteProject(ctx context.Context, project domain.Project) error {
	_, err := s.client.Delete(ctx, s.getKey(project.Name), nil)
	return err
}

func (s *EtcdStorage) getKey(name string) string {
	return s.rootDir + "/" + name
}
