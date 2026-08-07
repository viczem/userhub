package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"
	"github.com/viczem/userhub/services/identity/internal/config"
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
		Name: "identity",
		Commands: []*cli.Command{{
			Name:  "start",
			Usage: "run the service",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return start(cfg, logger)
			},
		}},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		logger.Error("command failed", "error", err)
		os.Exit(1)
	}
}
