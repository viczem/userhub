package main

import (
	"log/slog"
	"os"

	"github.com/viczem/userhub/services/identity/internal/config"
)

func main() {
	if _, err := config.Load(); err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("invalid configuration", "error", err)
		os.Exit(1)
	}
}
