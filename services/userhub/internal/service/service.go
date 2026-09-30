// Package service provides domain services for the application.
package service

import (
	"context"

	"github.com/viczem/userhub/services/userhub/internal/config"
)

// Repository defines operations bound to one transaction.
type Repository interface {
	runtimeRepository
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	Close(ctx context.Context)
}

// Database starts transactions used by the service.
type Database[R Repository] interface {
	NewRepository(ctx context.Context) (R, error)
}

// Service provides domain operations using transactional repositories.
type Service[R Repository] struct {
	cfg *config.Config
	db  Database[R]
}

// NewService creates a new Service with the given database.
func NewService[R Repository](cfg *config.Config, db Database[R]) *Service[R] {
	return &Service[R]{
		cfg: cfg,
		db:  db,
	}
}
