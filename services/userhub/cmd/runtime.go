package main

import (
	"context"
	"fmt"
	"time"

	"github.com/viczem/userhub/services/userhub/internal/config"
	"github.com/viczem/userhub/services/userhub/internal/repository"
	"github.com/viczem/userhub/services/userhub/internal/service"
)

func runtime(cfg *config.Config, action string) error {
	db, err := repository.Open(context.Background(), cfg)
	if err != nil {
		return err
	}

	defer db.Close()

	srv := service.NewService(cfg, db)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	switch action {
	case "create":
		session, err := srv.CreateRuntimeSession(ctx)
		if err != nil {
			return err
		}
		fmt.Println(session.Token)
	case "delete":
		if err := srv.DeleteRuntimeSession(ctx); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	return nil
}
