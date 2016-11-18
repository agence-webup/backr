package http

import (
	"context"
	"net/http"
	"webup/backr"
	"webup/backr/tasks"

	"fmt"

	"encoding/json"

	log "github.com/Sirupsen/logrus"
)

type HTTPApi struct {
}

func NewAPI() backr.API {
	return &HTTPApi{}
}

func (api *HTTPApi) Listen(ctx context.Context) error {

	opts, ok := backr.SettingsFromContext(ctx)
	if !ok {
		return fmt.Errorf("Unable to get options from context")
	}

	http.HandleFunc("/status", api.GetStatus(ctx))
	http.HandleFunc("/health", api.GetHealth(ctx))

	log.Infof("API listening on %v", opts.ApiListen)
	return http.ListenAndServe(opts.ApiListen, nil)
}

func (api *HTTPApi) GetStatus(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := tasks.GetStatus(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, err)
			return
		}

		jsonData, err := json.Marshal(status)
		w.Write(jsonData)
	}
}

type failedBackup struct {
	ProjectName string             `json:"project"`
	Backup      backr.BackupStatus `json:"failed_backup"`
}

func (api *HTTPApi) GetHealth(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := tasks.GetStatus(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, err)
			return
		}

		failedBackups := []failedBackup{}
		healthy := true

		for _, project := range status.ConfiguredProjects {
			for _, backup := range project.ConfiguredBackups {

				if backup.IsHealthy == false {
					healthy = false
					failedBackups = append(failedBackups, failedBackup{ProjectName: project.Name, Backup: backup})
				}

			}
		}

		if healthy {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "OK")
		} else {
			w.WriteHeader(http.StatusTeapot)
			jsonData, _ := json.Marshal(failedBackups)
			w.Write(jsonData)
		}

	}
}
