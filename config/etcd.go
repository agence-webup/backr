package config

import (
	"time"

	etcd "github.com/coreos/etcd/client"
)

// GetEtcdConnection initialize a new etcd connection
func GetEtcdConnection(endpoints []string) (etcd.KeysAPI, error) {
	cfg := etcd.Config{
		Endpoints: endpoints,
		Transport: etcd.DefaultTransport,
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: 3 * time.Second,
	}

	c, err := etcd.New(cfg)
	if err != nil {
		return nil, err
	}

	etcdCli := etcd.NewKeysAPI(c)

	return etcdCli, nil
}
