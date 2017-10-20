package http

import (
	"context"
	"io/ioutil"
	"net/http"
	"webup/backr"
	"webup/backr/tasks"

	"fmt"

	"encoding/json"

	jwt "github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"
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

	http.HandleFunc("/backups", api.GetBackups(ctx))
	http.HandleFunc("/status", api.GetStatus(ctx))
	http.HandleFunc("/health", api.GetHealth(ctx))

	log.Infof("API listening on %v", opts.ApiListen)
	return http.ListenAndServe(opts.ApiListen, nil)
}

func (api *HTTPApi) GetBackups(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		opts, ok := backr.SettingsFromContext(ctx)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// get 'name' param
		name := r.URL.Query().Get("name")
		if name == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "'name' param is required")
			return
		}

		// get 'token' param
		rawToken := r.URL.Query().Get("token")
		if rawToken == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "'token' param is required")
			return
		}

		// check token validity
		// Parse takes the token string and a function for looking up the key. The latter is especially
		// useful if you use multiple keys for your application.  The standard is to use 'kid' in the
		// head of the token to identify which key to use, but the parsed token (head and claims) is provided
		// to the callback, providing flexibility.
		token, err := jwt.Parse(rawToken, func(token *jwt.Token) (interface{}, error) {
			// Don't forget to validate the alg is what you expect:
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}

			// get the secret
			secret, err := ioutil.ReadFile(opts.SecretFilepath)
			if err != nil {
				return nil, fmt.Errorf("Secret file not found")
			}

			return secret, nil
		})

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, err)
			return
		}

		if token.Valid {
			// get backups
			results, err := tasks.GetBackups(name, ctx)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, err)
				return
			}

			jsonData, err := json.Marshal(results)
			w.Write(jsonData)
			return
		}

		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, "token invalid")
		return
	}
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
