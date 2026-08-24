//go:build ignore

// Command migration provides development-only migration operations.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
	"github.com/viczem/userhub/services/iam/migrations"
)

func main() {
	cmd := &cli.Command{
		Name:  "migration",
		Usage: "manage IAM migrations during development",
		Commands: []*cli.Command{
			{
				Name:      "create",
				Usage:     "create the next sequential up/down migration pair",
				ArgsUsage: "NAME",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if cmd.Args().Len() != 1 {
						return errors.New("migration name is required")
					}
					return migrations.Create("migrations", cmd.Args().First())
				},
			},
			{
				Name:  "up",
				Usage: "apply all pending migrations",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if cmd.Args().Len() != 0 {
						return errors.New("migration up does not accept arguments")
					}
					databaseURL, err := requireDatabaseURL()
					if err != nil {
						return err
					}
					return migrations.Up(databaseURL)
				},
			},
			{
				Name:  "down",
				Usage: "roll back the most recently applied migration",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if cmd.Args().Len() != 0 {
						return errors.New("migration down does not accept arguments")
					}
					databaseURL, err := requireDatabaseURL()
					if err != nil {
						return err
					}
					return migrations.Down(databaseURL)
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func requireDatabaseURL() (string, error) {
	value := os.Getenv("DB_URL")
	if value == "" {
		return "", errors.New("DB_URL is required")
	}
	return value, nil
}
