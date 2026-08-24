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

	"github.com/viczem/userhub/services/iam/internal/config"
	"github.com/viczem/userhub/services/iam/internal/repository"
	"github.com/viczem/userhub/services/iam/internal/rest"
)

func start(cfg *config.Config, logger *slog.Logger) error {
	db, err := repository.Open(context.Background(), cfg)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("close", slog.String("tag", "database"), slog.String("error", err.Error()))
		}
	}()

	if err := db.Ready(context.Background()); err != nil {
		logger.Warn("database unavailable during startup", slog.String("tag", "database"))
	}

	router := rest.NewREST(cfg, logger, db.Ready)
	server := &http.Server{
		Handler:           router,
		Addr:              cfg.HTTPAddr,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		ReadTimeout:       cfg.HTTPReadTimeout,
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
		MaxHeaderBytes:    cfg.HTTPMaxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(logger.With(slog.String("tag", "server")).Handler(), slog.LevelInfo),
	}

	serverErrors := make(chan error, 1)

	go func() {
		logger.Info("start", slog.String("tag", "server"), slog.String("addr", cfg.HTTPAddr))

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
	logger.Info("shutdown", slog.String("tag", "server"))

	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPGracefulShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return err
	}

	logger.Info("goodbye", slog.String("tag", "server"))

	return nil
}
