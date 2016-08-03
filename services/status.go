package services

import (
	"fmt"
	"time"
	"webup/backoops/config"
	"webup/backoops/options"
	"webup/backoops/state"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
)

const (
	configFile = "backup.yml"
)

// StatusBackupConfig prints the status of the configured backup for the current project
func StatusBackupConfig(ctx context.Context) {

	opts, ok := options.FromContext(ctx)
	if !ok {
		log.Errorln("Unable to get options from context")
		return
	}

	// get a state storage
	stateStorage, err := state.GetStorage(opts)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorln("Unable to connect to state storage")
		return
	}
	defer stateStorage.CleanUp()

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

	project, err := stateStorage.GetProject(ctx, backupConfig.Name)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorln("Unable to get the project state from state storage")
		return
	}

	if project == nil {
		log.WithFields(log.Fields{
			"name": backupConfig.Name,
		}).Warnln("Backup is not configured")
		return
	}

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
		fmt.Println("  Next scheduled:", backup.GetNextBackupTime(opts.TimeSpec, time.Now()))
		fmt.Println("  ---")
	}
	fmt.Println("-------------------------------------")

}
