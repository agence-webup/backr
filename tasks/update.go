package tasks

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"webup/backr"
	"webup/backr/state"

	log "github.com/sirupsen/logrus"
)

// UpdateStateFromSpec runs before each backup to fetch the backup.yml files inside watched directories and update the state
func UpdateStateFromSpec(ctx context.Context) {

	log.Debugln("Updating state from backup.yml files...")

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

		filepath.Walk(dir, walkFunc)
		// if err != nil {
		// 	log.WithFields(log.Fields{
		// 		"path": dir,
		// 	}).Infoln("Skipping directory")
		// 	continue
		// }
	}

	// log.Info(" ▶︎ Processing config files...")

	configuredBackups := map[string]backr.ProjectBackupSpec{}

	// fetch already configured projects before executing
	existingProjects, err := stateStorage.ConfiguredProjects(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorln("Unable to get existing projects from state storage")
		return
	}

	for _, file := range configFiles {

		log.WithFields(log.Fields{
			"file": file,
		}).Debugln("Parsing spec file")

		parsedSpec, err := parseSpecFile(file)
		if err != nil {
			log.WithFields(log.Fields{
				"file": file,
				"err":  err,
			}).Errorln("Unable to parse backup.yml file")
			continue
		}

		if err := parsedSpec.IsValid(); err != nil {
			log.WithFields(log.Fields{
				"file": file,
				"err":  err,
			}).Errorln("The backup.yml file is not valid")
			continue
		}

		// keep the parsed config for delete handling, later (see below)
		configuredBackups[parsedSpec.Name] = parsedSpec

		var project backr.Project

		// trying to find the existing project
		project, ok := existingProjects[parsedSpec.Name]
		if !ok {
			log.WithFields(log.Fields{
				"name": parsedSpec.Name,
			}).Infoln("Backup config not found in current state. Create it.")

			project = backr.NewProject(parsedSpec)

		} else {
			// if _, ok := (*running)[project.Name]; ok {
			// 	log.WithFields(log.Fields{
			// 		"name": project.Name,
			// 	}).Infoln("Backup is currently running. Delay the update to next iteration.")
			// 	continue
			// }

			report := project.Update(parsedSpec)

			// log only when a config has been updated
			if report.Created > 0 || report.Deleted > 0 {
				log.WithFields(log.Fields{
					"name":      parsedSpec.Name,
					"created":   report.Created,
					"unchanged": report.Unchanged,
					"deleted":   report.Deleted,
				}).Infoln("Backup successfully configured")
			}
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

	log.Debugln("State update done")

}
