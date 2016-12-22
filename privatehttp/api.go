package privatehttp

import (
	"context"
	"net/http"
	"webup/backr"

	"fmt"

	"webup/backr/tasks"

	log "github.com/Sirupsen/logrus"
)

type HTTPApi struct {
}

func NewAPI() backr.PrivateAPI {
	return &HTTPApi{}
}

func (api *HTTPApi) Listen(ctx context.Context) error {

	opts, ok := backr.SettingsFromContext(ctx)
	if !ok {
		return fmt.Errorf("Unable to get options from context")
	}

	http.HandleFunc("/actions/backup", api.Backup(ctx))

	log.Infof("Private API listening on %v", opts.PrivateAPIListen)
	return http.ListenAndServe(opts.PrivateAPIListen, nil)
}

func (api *HTTPApi) Backup(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// opts, ok := backr.SettingsFromContext(ctx)
		// if !ok {
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	return
		// }

		// get 'name' param
		name := r.URL.Query().Get("name")
		if name == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "'name' param is required")
			return
		}

		err := tasks.PerformStandaloneBackup(ctx, name)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
