package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/viczem/userhub/services/identity/internal/config"
	"github.com/viczem/userhub/services/identity/internal/rest"
)

func start(cfg *config.Config, logger *slog.Logger) error {
	server := &http.Server{
		Handler:           rest.NewREST(cfg, logger),
		Addr:              cfg.HTTPAddr,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		ReadTimeout:       cfg.HTTPReadTimeout,
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
		MaxHeaderBytes:    cfg.HTTPMaxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(logger.With(slog.String("tag", "server")).Handler(), slog.LevelInfo),
	}

	go func() {
		logger.Info("start", slog.String("tag", "server"), slog.String("addr", cfg.HTTPAddr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("start", slog.String("tag", "server"), slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutdown", slog.String("tag", "server"))
	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPGracefulShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown", slog.String("tag", "server"), slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("goodbye", slog.String("tag", "server"))
	return nil
}
