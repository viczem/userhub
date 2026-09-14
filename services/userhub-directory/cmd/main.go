// Package main provides the command-line interface for UserHub Service.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"
	"github.com/viczem/userhub/services/userhub-directory/internal/config"
	"github.com/viczem/userhub/services/userhub-directory/migrations"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		slog.Error("invalid config", "error", err)
		os.Exit(1)
	}

	if cfg.AppEnv == config.AppEnvDevelopment {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))
	} else {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	}

	cmd := newCommand(cfg)

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}

func newCommand(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name: "userhub",

		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "run the service",
				Action: func(_ context.Context, _ *cli.Command) error {
					return start(cfg)
				},
			},
			{
				Name:  "migrate",
				Usage: "apply pending database migrations; migrations are forward-only",
				Action: func(_ context.Context, cmd *cli.Command) error {
					if cmd.Args().Len() != 0 {
						return errors.New("userhub migrate does not accept arguments")
					}

					return migrations.Up(cfg.DB.DirectURL)
				},
			},
			{
				Name:  "healthcheck",
				Usage: "probe local service readiness",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return healthCheck(ctx, cfg.HTTP.Addr)
				},
			},
		},
	}
}
