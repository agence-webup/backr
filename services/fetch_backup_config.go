package services

import (
	"os"
	"path/filepath"
	"strings"
	"time"
	"webup/backoops/config"
	"webup/backoops/domain"
	"webup/backoops/options"
	"webup/backoops/state"

	log "github.com/Sirupsen/logrus"

	"golang.org/x/net/context"
)

// FetchBackupConfig runs every X seconds to fetch the backup.yml files inside watched directories
func FetchBackupConfig(ctx context.Context, runningState chan map[string]bool) {

	opts, ok := options.FromContext(ctx)
	if !ok {
		log.Errorln("Unable to get options from context")
		return
	}

	ticker := time.NewTicker(time.Duration(opts.ConfigRefreshRate) * time.Second)

	runningBackups := make(map[string]bool)

	go func() {
		for {
			select {
			case <-ticker.C:
				// execute the backup process
				run(ctx, opts, &runningBackups)
			case running := <-runningState:
				// update the list of the running backups when a map is received from the channel
				runningBackups = running
			}

			// fmt.Println("")

		}
	}()

	log.Infof("'Fetch backup files' service is started (refresh rate: %d min)", opts.ConfigRefreshRate)

	// waiting for ctx to cancel
	<-ctx.Done()

	ticker.Stop()
	log.Infoln("Stopping backup fetching daemon.")

}

func run(ctx context.Context, opts options.Options, running *map[string]bool) {

	// log.Infoln("Fetching backup config files...")

	// get a state storage
	stateStorage, err := state.GetStorage(opts)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorln("Unable to connect to state storage")
		return
	}

	configFiles := []string{}

	walkFunc := func(path string, info os.FileInfo, err error) error {
		filename := info.Name()
		if !info.IsDir() && filename == "backup.yml" {
			configFiles = append(configFiles, path)
			return filepath.SkipDir
		}

		// the following directories are skipped
		// - node_modules / vendor
		// - hidden directories (except current folder: '.')
		if info.IsDir() && ((strings.HasPrefix(filename, ".") && len(filename) > 1) || filename == "node_modules" || filename == "vendor") {
			return filepath.SkipDir
		}

		return nil
	}

	// log.Println(" ▶︎ Updating config with backup.yml files...")

	for _, dir := range opts.WatchDirs {
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

	configuredBackups := map[string]domain.BackupConfig{}

	// fetch already configured projects before executing
	existingProjects, err := stateStorage.ConfiguredProjects(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorln("Unable to get existing projects from state storage")
		return
	}

	for _, file := range configFiles {
		parsedConfig, err := config.ParseConfigFile(file)
		if err != nil {
			log.WithFields(log.Fields{
				"file": file,
				"err":  err,
			}).Errorln("Unable to parse backup.yml file")
			continue
		}

		if !parsedConfig.IsValid() {
			log.WithFields(log.Fields{
				"file": file,
			}).Errorln("The backup.yml file is not valid: 'name' required and 'backups' > 0")
			continue
		}

		// keep the parsed config for delete handling, later (see below)
		configuredBackups[parsedConfig.Name] = parsedConfig

		var project domain.Project

		// trying to find the existing project
		project, ok := existingProjects[parsedConfig.Name]
		if !ok {
			log.WithFields(log.Fields{
				"name": parsedConfig.Name,
			}).Infoln("Backup config not found in current state. Create it.")

			project = domain.NewProject(parsedConfig)

		} else {
			if _, ok := (*running)[project.Name]; ok {
				log.WithFields(log.Fields{
					"name": project.Name,
				}).Infoln("Backup is currently running. Delay the update to next iteration.")
				continue
			}

			project.Update(parsedConfig)
		}

		// set the directory of the config path
		project.Dir, _ = filepath.Abs(filepath.Dir(file))

		// save the configuration into state storage
		err = stateStorage.SaveProject(ctx, project)
		if err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Errorln("Unable to save project into state storage")
		}
	}

	// clean deleted configurations
	for name, project := range existingProjects {
		if _, ok := configuredBackups[name]; !ok {
			log.WithFields(log.Fields{
				"name": name,
			}).Infoln("Backup config no longer exists. Remove it from the current state.")

			err := stateStorage.DeleteProject(ctx, project)

			if err != nil {
				log.WithFields(log.Fields{
					"name": name,
				}).Errorln("Unable to delete project from the current state.")
			}
		}
	}

	log.Infoln("Configuration update done")
}
