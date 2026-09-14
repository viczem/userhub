// Package rest provides UserHub Service's HTTP layer.
package rest

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/viczem/userhub/services/userhub/internal/config"
)

// NewREST returns a chi router.
func NewREST(cfg *config.Config, readiness func(context.Context) error) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)

	// TODO: Add a setting to configure where the user's IP address is obtained from.
	// Currently assumes the server is running behind nginx.
	r.Use(middleware.ClientIPFromHeader("X-Real-IP"))

	r.Use(loggerMiddleware()) // Logger should come before Recoverer

	if cfg.AppEnv == config.AppEnvProduction {
		r.Use(recovererMiddleware())
	}

	r.Use(middleware.NoCache)
	r.Use(maxBodyBytesMiddleware(cfg.HTTP.MaxBodyBytes))
	r.Get("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		if readiness == nil || readiness(r.Context()) != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte(".")); err != nil {
			slog.Error("write", "error", err)
			return
		}
	})

	r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte("Hello, World!")); err != nil {
			slog.Error("write", "error", err)
			return
		}
	})

	return r
}
