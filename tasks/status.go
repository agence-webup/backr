package tasks

import (
	"context"
	"fmt"
	"webup/backr"
	"webup/backr/state"
)

func GetStatus(ctx context.Context) (backr.Status, error) {

	status := backr.Status{}

	opts, ok := backr.SettingsFromContext(ctx)
	if !ok {
		return status, fmt.Errorf("Unable to get options from context")
	}

	// get a state storage
	stateStorage, err := state.GetStorage(opts)
	if err != nil {
		return status, err
	}

	// fetch all configured backups
	projects, err := stateStorage.ConfiguredProjects(ctx)
	if err != nil {
		return status, err
	}

	configuredProjects := []backr.ProjectStatus{}

	for _, project := range projects {
		configuredBackups := []backr.BackupStatus{}

		for _, backup := range project.Backups {

			status := backr.BackupStatus{
				MinAge:        backup.MinAge,
				PeriodUnit:    backup.PeriodUnit,
				LastExecution: backup.LastExecution,
				NextExecution: backup.GetNextBackupTime(opts.TimeSpec, opts.StartupTime),
				IsHealthy:     backup.GetHealth(opts.TimeSpec, opts.StartupTime),
			}

			configuredBackups = append(configuredBackups, status)
		}

		projectStatus := backr.ProjectStatus{Name: project.Name, ConfiguredBackups: configuredBackups}
		configuredProjects = append(configuredProjects, projectStatus)
	}

	return backr.Status{ConfiguredProjects: configuredProjects}, nil
}
