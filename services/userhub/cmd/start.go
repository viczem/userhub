package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/viczem/userhub/services/userhub/internal/config"
	"github.com/viczem/userhub/services/userhub/internal/repository"
	"github.com/viczem/userhub/services/userhub/internal/rest"
)

func start(cfg *config.Config) error {
	db, err := repository.Open(context.Background(), cfg)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("close", slog.String("tag", "database"), slog.String("error", err.Error()))
		}
	}()

	if err := db.Ready(context.Background()); err != nil {
		slog.Warn("database unavailable during startup", slog.String("tag", "database"))
	}

	router := rest.NewREST(cfg, db.Ready)
	server := &http.Server{
		Handler:           router,
		Addr:              cfg.HTTP.Addr,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
		MaxHeaderBytes:    cfg.HTTP.MaxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(slog.With(slog.String("tag", "server")).Handler(), slog.LevelInfo),
	}

	serverErrors := make(chan error, 1)

	go func() {
		slog.Info("start", slog.String("tag", "server"), slog.String("addr", cfg.HTTP.Addr))

		serverErrors <- server.ListenAndServe()
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-quit:
	}

	db.StopReadiness()
	slog.Info("shutdown", slog.String("tag", "server"))

	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.GracefulShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return err
	}

	slog.Info("goodbye", slog.String("tag", "server"))
	return nil
}
