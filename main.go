package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
	"webup/backoops/config"
	"webup/backoops/domain"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	etcd "github.com/coreos/etcd/client"
	"github.com/jawher/mow.cli"
)

func main() {
	app := cli.App("backoops", "Perform backups")

	app.Spec = "-w... [--etcd]"

	etcdEndpoints := app.String(cli.StringOpt{
		Name:   "etcd",
		Value:  "http://localhost:2379",
		Desc:   "Endpoints for etcd (separated by a comma)",
		EnvVar: "ETCD_ADVERTISE_URLS",
	})

	watchDirs := app.StringsOpt("w watch", []string{}, "Specifies the directories to watch for finding backup.yml files")

	app.Action = func() {

		cfg := etcd.Config{
			Endpoints: strings.Split(*etcdEndpoints, ","),
			Transport: etcd.DefaultTransport,
			// set timeout per request to fail fast when the target endpoint is unavailable
			HeaderTimeoutPerRequest: 3 * time.Second,
		}
		c, err := etcd.New(cfg)
		if err != nil {
			log.Fatal(err)
		}
		etcdCli := etcd.NewKeysAPI(c)

		configFiles := []string{}

		walkFunc := func(filepath string, info os.FileInfo, err error) error {
			if !info.IsDir() && info.Name() == "backup.yml" {
				configFiles = append(configFiles, filepath)
			}

			return nil
		}

		log.Println(" ▶︎ Finding backup.yml files...")

		for _, dir := range *watchDirs {
			fileinfo, err := os.Stat(dir)
			if err != nil {
				log.WithFields(log.Fields{
					"path": dir,
					"err":  err,
				}).Errorln("Unable to get file info")
				continue
			}

			// handle only directories
			if !fileinfo.IsDir() {
				log.WithFields(log.Fields{
					"path": dir,
				}).Warnln("Not a directory. Skipped.")
				continue
			}

			err = filepath.Walk(dir, walkFunc)
			if err != nil {
				log.WithFields(log.Fields{
					"path": dir,
					"err":  err,
				}).Errorln("Unable to walk into directory")
				continue
			}
		}

		log.Info(" ▶︎ Processing config files...")

		ctx := context.Background()
		rootDir := "/backups"
		configuredBackups := map[string]domain.BackupConfig{}

		existingConfig, _ := etcdCli.Get(ctx, rootDir, nil)

		for _, file := range configFiles {
			backupConfig, err := config.ParseConfigFile(file)
			if err != nil {
				log.WithFields(log.Fields{
					"file": file,
					"err":  err,
				}).Errorln("Unable to parse backup.yml file")
				continue
			}

			if !backupConfig.IsValid() {
				log.WithFields(log.Fields{
					"file": file,
				}).Errorln("The backup.yml file is not valid: 'name' required and 'backups' > 0")
				continue
			}

			key := rootDir + "/" + backupConfig.Name
			configuredBackups[key] = backupConfig

			currentStateData, err := etcdCli.Get(ctx, key, nil)
			if err != nil && !etcd.IsKeyNotFound(err) {
				log.WithFields(log.Fields{
					"key": key,
					"err": err,
				}).Errorln("Unable to get the key in etcd")
				continue
			}

			var backupState domain.BackupState

			if err != nil && etcd.IsKeyNotFound(err) {
				log.WithFields(log.Fields{
					"key": key,
				}).Infoln("Backup config not found in etcd. Create it.")

				backupState = domain.NewBackupState(backupConfig)

			} else {
				log.WithFields(log.Fields{
					"key": key,
				}).Infoln("Backup config already exists in etcd. Update it.")

				backupState = domain.BackupState{}
				json.Unmarshal([]byte(currentStateData.Node.Value), &backupState)

				backupState.Update(backupConfig)

			}

			// get json data
			jsonData, _ := json.Marshal(backupState)
			// set the value in etcd
			etcdCli.Set(ctx, key, string(jsonData), nil)

		}

		// clean deleted configs
		if existingConfig != nil && existingConfig.Node != nil {
			for _, existingConfigKey := range existingConfig.Node.Nodes {
				if _, ok := configuredBackups[existingConfigKey.Key]; !ok {
					log.WithFields(log.Fields{
						"key": existingConfigKey.Key,
					}).Infoln("Backup config no longer exists. Remove it from etcd.")
					etcdCli.Delete(ctx, existingConfigKey.Key, nil)
				}
			}
		}

	}

	app.Run(os.Args)
}
