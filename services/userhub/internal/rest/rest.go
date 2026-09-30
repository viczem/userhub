// Package rest provides UserHub Service's HTTP layer.
package rest

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/viczem/userhub/services/userhub/internal/config"
)

type service interface {
	HealthReady() bool
}

// REST exposes the UserHub service over HTTP.
type REST struct {
	router   *chi.Mux
	srv      service
	validate *validator.Validate
}

var _ http.Handler = (*REST)(nil)

// NewREST returns a chi router.
func NewREST(cfg *config.Config, srv service) *REST {
	api := &REST{
		router:   chi.NewRouter(),
		srv:      srv,
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}
	api.router.Use(middleware.RequestID)

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

	return api
}

func (api *REST) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.router.ServeHTTP(w, r)
}
