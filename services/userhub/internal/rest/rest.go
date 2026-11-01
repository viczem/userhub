// Package rest provides UserHub Service's HTTP layer.
package rest

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/viczem/userhub/services/userhub/internal/config"
)

type service interface {
	HealthReady() bool
	configService
}

// REST exposes the UserHub service over HTTP.
type REST struct {
	router *chi.Mux
	srv    service
}

var _ http.Handler = (*REST)(nil)

// NewREST returns a chi router.
func NewREST(cfg *config.Config, srv service) *REST {
	api := &REST{
		router: chi.NewRouter(),
		srv:    srv,
	}
	api.router.Use(requestIDMiddleware)

	// TODO: Add a setting to configure where the user's IP address is obtained from.
	// Currently assumes the server is running behind nginx.
	api.router.Use(middleware.ClientIPFromHeader("X-Real-IP"))

	api.router.Use(loggerMiddleware()) // Logger should come before Recoverer

	if cfg.AppEnv == config.AppEnvProduction {
		api.router.Use(recovererMiddleware())
	}

	api.router.Use(middleware.NoCache)
	api.router.Use(maxBodyBytesMiddleware(cfg.HTTP.MaxBodyBytes))

	api.router.Get("/health/live", api.live)
	api.router.Get("/health/ready", api.ready)

	api.router.Route("/api", func(r chi.Router) {
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, r, http.StatusNotFound, nil)
		})

		r.Route("/config", func(r chi.Router) {
			r.Use(validateConfigSessionMiddleware())
			r.Get("/roles", api.readUsersRoles)
			r.Post("/roles", api.createUsersRole)
			r.Patch("/roles/{id}", api.updateUsersRole)
			r.Delete("/roles/{id}", api.deleteUsersRole)
		})
	})
	return api
}

func (api *REST) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.router.ServeHTTP(w, r)
}
