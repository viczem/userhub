package rest

import "net/http"

func (api *REST) live(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (api *REST) ready(w http.ResponseWriter, _ *http.Request) {
	if !api.srv.HealthReady() {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
}
