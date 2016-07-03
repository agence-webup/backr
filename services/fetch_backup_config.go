package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
	"webup/backoops/config"
	"webup/backoops/domain"
	"webup/backoops/options"

	log "github.com/Sirupsen/logrus"
	etcd "github.com/coreos/etcd/client"

	"golang.org/x/net/context"
)

// FetchBackupConfig runs every X seconds to fetch the backup.yml files inside watched directories
func FetchBackupConfig(ctx context.Context, runningState chan map[string]bool) {

	ticker := time.NewTicker(10 * time.Second)

	runningBackups := make(map[string]bool)

	go func() {
		for {
			select {
			case <-ticker.C:
				// execute the backup process
				run(ctx, &runningBackups)
			case running := <-runningState:
				// update the list of the running backups when a map is received from the channel
				runningBackups = running
			}

			// fmt.Println("")

		}
	}()

	log.Infoln("'Fetch backup config' service is started.")

	// waiting for ctx to cancel
	<-ctx.Done()

	ticker.Stop()
	log.Infoln("Stopping backup fetching daemon.")

}

func run(ctx context.Context, running *map[string]bool) {

	log.Infoln("Fetching backup config files...")

	options, ok := options.FromContext(ctx)
	if !ok {
		log.Errorln("Unable to get options from context")
		return
	}

	etcdCli, err := config.GetEtcdConnection(options.EtcdEndpoints)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorln("Unable to connect to etcd")
		return
	}

	configFiles := []string{}

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && info.Name() == "backup.yml" {
			configFiles = append(configFiles, path)
			return filepath.SkipDir
		}

		if info.IsDir() && (strings.HasPrefix(info.Name(), ".") || info.Name() == "node_modules" || info.Name() == "vendor") {
			return filepath.SkipDir
		}

		return nil
	}

	// log.Println(" ▶︎ Updating config with backup.yml files...")

	for _, dir := range options.WatchDirs {
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

	// log.Info(" ▶︎ Processing config files...")

	// ctx := context.Background()
	rootDir := options.BackupRootDir
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

		var project domain.Project

		if err != nil && etcd.IsKeyNotFound(err) {
			log.WithFields(log.Fields{
				"key": key,
			}).Infoln("Backup config not found in etcd. Create it.")

			project = domain.NewProject(backupConfig)

		} else {
			// log.WithFields(log.Fields{
			// 	"key": key,
			// }).Infoln("Backup config already exists in etcd. Update it.")

			project = domain.Project{}
			json.Unmarshal([]byte(currentStateData.Node.Value), &project)

			// if a backup is running, delay the update for later
			// if project.IsRunning {
			if _, ok := (*running)[project.Name]; ok {
				log.WithFields(log.Fields{
					"key": key,
				}).Infoln("Backup is currently running. Delay the update to next iteration.")
				continue
			}

			project.Update(backupConfig)

		}

		// set the directory of the config path
		project.Dir, _ = filepath.Abs(filepath.Dir(file))

		// get json data
		jsonData, _ := json.Marshal(project)
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
