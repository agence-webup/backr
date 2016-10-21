package tasks

import (
	"context"
	"fmt"
	"webup/backr"
	"webup/backr/state"

	log "github.com/Sirupsen/logrus"
)

func DisplayStatus(ctx context.Context) {

	opts, ok := backr.SettingsFromContext(ctx)
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

	// fetch all configured backups
	projects, err := stateStorage.ConfiguredProjects(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorln("Unable to get configured projects from state storage")
		return
	}

	for _, project := range projects {
		fmt.Println("####", project.Name, "####")
		for _, backup := range project.Backups {

			fmt.Println("--------------------------------------------")
			fmt.Println("      ttl:", backup.TimeToLive)
			fmt.Println("  min_age:", backup.MinAge)
			fmt.Println("last_exec:", backup.LastExecution)
			fmt.Println("next_exec:", backup.GetNextBackupTime(opts.TimeSpec, opts.StartupTime))
		}

		fmt.Print("\n\n")
	}

}
