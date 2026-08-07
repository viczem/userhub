// Package rest provides the HTTP layer of the application
package rest

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/viczem/userhub/services/iam/internal/config"
)

// NewREST returns a chi router.
func NewREST(cfg *config.Config, logger *slog.Logger) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)

	// TODO: Add a setting to configure where the user's IP address is obtained from.
	// Currently assumes the server is running behind nginx.
	r.Use(middleware.ClientIPFromHeader("X-Real-IP"))

	r.Use(loggerMiddleware(logger)) // Logger should come before Recoverer

	if cfg.AppEnv == config.AppEnvProduction {
		r.Use(middleware.Recoverer)
	}

	r.Use(middleware.NoCache)
	r.Use(maxBodyBytesMiddleware(cfg.HTTPMaxBodyBytes))
	r.Use(middleware.Heartbeat("/health/live"))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	return r
}
