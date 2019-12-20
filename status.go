package backr

import (
	"time"
)

type Status struct {
	ConfiguredProjects []ProjectStatus `json:"projects"`
}

type ProjectStatus struct {
	Name              string         `json:"name"`
	ConfiguredBackups []BackupStatus `json:"backups"`
}

type BackupStatus struct {
	PeriodUnit    int       `json:"period_unit"`
	MinAge        int       `json:"min_age"`
	LastExecution time.Time `json:"last_exec"`
	NextExecution time.Time `json:"next_exec"`
	IsHealthy     bool      `json:"is_healthy"`
}
