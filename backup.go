package main

import (
	etcd "github.com/coreos/etcd/client"
)

func StartBackup(etcdCli etcd.KeysAPI, backupRootKey string) error {

    config, _ := etcdCli.Get(ctx, backupRootKey, nil)

    if config != nil && config.Node != nil {
        for _, backupConfigKey := range config.Node.Nodes {

            

            if , ok := configuredBackups[existingConfigKey.Key]; !ok {
                log.WithFields(log.Fields{
                    "key": existingConfigKey.Key,
                }).Infoln("Backup config no longer exists. Remove it from etcd.")
                etcdCli.Delete(ctx, existingConfigKey.Key, nil)
            }
        }
    }

	return nil
}
