// Package main provides the command-line interface for the IAM service.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"
	"github.com/viczem/userhub/services/iam/internal/config"
	"github.com/viczem/userhub/services/iam/migrations"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		slog.Error("invalid config", "error", err)
		os.Exit(1)
	}

	var logger *slog.Logger
	if cfg.AppEnv == config.AppEnvDevelopment {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	} else {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	cmd := &cli.Command{
		Name: "iam",

		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "run the service",
				Action: func(_ context.Context, _ *cli.Command) error {
					return start(cfg, logger)
				},
			},
			{
				Name:  "migrate",
				Usage: "apply pending database migrations; migrations are forward-only",
				Action: func(_ context.Context, _ *cli.Command) error {
					return migrations.Up(cfg.DBDirectURL)
				},
			},
			{
				Name:  "healthcheck",
				Usage: "probe local service readiness",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return healthCheck(ctx, cfg.HTTPAddr)
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		logger.Error("command failed", "error", err)
		os.Exit(1)
	}
}
