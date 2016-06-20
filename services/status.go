package services

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"
	"webup/backoops/config"
	"webup/backoops/domain"
	"webup/backoops/options"

	log "github.com/Sirupsen/logrus"
	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

const (
	configFile = "backup.yml"
)

// EnableBackupConfig configures the backup for the current project
func EnableBackupConfig(ctx context.Context) {

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

	rootDir := options.BackupRootDir
	file := configFile // in the current directory
	// configuredBackups := map[string]domain.BackupConfig{}

	// existingConfig, _ := etcdCli.Get(ctx, rootDir, nil)

	backupConfig, err := config.ParseConfigFile(file)
	if err != nil {
		log.WithFields(log.Fields{
			"file": file,
			"err":  err,
		}).Errorln("Unable to parse backup.yml file")
		return
	}

	if !backupConfig.IsValid() {
		log.WithFields(log.Fields{
			"file": file,
		}).Errorln("The backup.yml file is not valid: 'name' required and 'backups' > 0")
		return
	}

	key := rootDir + "/" + backupConfig.Name
	// configuredBackups[key] = backupConfig

	currentStateData, err := etcdCli.Get(ctx, key, nil)
	if err != nil && !etcd.IsKeyNotFound(err) {
		log.WithFields(log.Fields{
			"key": key,
			"err": err,
		}).Errorln("Unable to get the key in etcd")
		return
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
		if project.IsRunning {
			log.WithFields(log.Fields{
				"key": key,
			}).Infoln("Backup is currently running, so updating config is not possible. Try again in few minutes.")
			return
		}

		project.Update(backupConfig)

	}

	// set the directory of the config path
	project.Dir, _ = filepath.Abs(filepath.Dir(file))

	// get json data
	jsonData, _ := json.Marshal(project)
	// set the value in etcd
	etcdCli.Set(ctx, key, string(jsonData), nil)

	fmt.Println("")
	fmt.Println("Backup config successfully updated.")

}

// StatusBackupConfig prints the status of the configured backup for the current project
func StatusBackupConfig(ctx context.Context) {

	opts, ok := options.FromContext(ctx)
	if !ok {
		log.Errorln("Unable to get options from context")
		return
	}

	etcdCli, err := config.GetEtcdConnection(opts.EtcdEndpoints)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorln("Unable to connect to etcd")
		return
	}

	rootDir := opts.BackupRootDir
	file := configFile // in the current directory

	backupConfig, err := config.ParseConfigFile(file)
	if err != nil {
		log.WithFields(log.Fields{
			"file": file,
			"err":  err,
		}).Errorln("Unable to parse backup.yml file")
		return
	}

	if !backupConfig.IsValid() {
		log.WithFields(log.Fields{
			"file": file,
		}).Errorln("The backup.yml file is not valid: 'name' required and 'backups' > 0")
		return
	}

	key := rootDir + "/" + backupConfig.Name

	currentStateData, err := etcdCli.Get(ctx, key, nil)
	if err != nil && !etcd.IsKeyNotFound(err) {
		log.WithFields(log.Fields{
			"key": key,
			"err": err,
		}).Errorln("Unable to get the key in etcd")
		return
	}

	if err != nil && etcd.IsKeyNotFound(err) {
		log.WithFields(log.Fields{
			"key": key,
		}).Warnln("Backup is not configured")
		return
	}

	project := domain.Project{}
	json.Unmarshal([]byte(currentStateData.Node.Value), &project)

	fmt.Println("-------------------------------------")
	fmt.Println("Name:", project.Name)
	for _, backup := range project.Backups {
		fmt.Println("  TTL:           ", backup.TimeToLive, "days")
		fmt.Println("  Min age:       ", backup.MinAge, "days")
		if backup.LastExecution.IsZero() {
			fmt.Println("  Last execution:", "Never")
		} else {
			fmt.Println("  Last execution:", backup.LastExecution)
		}
		fmt.Println("  Next scheduled:", backup.GetNextBackupTime(1, time.Duration(86400)*time.Second, time.Now()))
		fmt.Println("  ---")
	}
	fmt.Println("-------------------------------------")

}
