package service

import (
	"context"

	"github.com/viczem/userhub/services/userhub/internal/domain"
)

type runtimeRepository interface {
	CreateRuntimeSession(ctx context.Context, session *domain.RuntimeSession) error
	DeleteRuntimeSession(ctx context.Context) error
}

// CreateRuntimeSession generates and persists a replacement runtime session.
func (s Service[R]) CreateRuntimeSession(ctx context.Context) (*domain.RuntimeSession, error) {
	var errKind domain.ErrorKind = "runtime create session"

	session, err := domain.NewRuntimeSession(
		s.cfg.KeyringHMAC,
		s.cfg.ConfigSessionIdleTimeout,
		s.cfg.ConfigSessionTTL,
	)
	if err != nil {
		return nil, errKind.WrapError(err, "new runtime session")
	}

	repo, err := s.db.NewRepository(ctx)
	if err != nil {
		return nil, errKind.WrapError(err, "new repository")
	}

	defer repo.Close(ctx)

	if err := repo.CreateRuntimeSession(ctx, session); err != nil {
		return nil, errKind.WrapError(err, "repo - create runtime session")
	}

	if err := repo.Commit(ctx); err != nil {
		return nil, errKind.WrapError(err, "commit")
	}

	return session, nil
}

// DeleteRuntimeSession invalidates the current runtime session.
func (s Service[R]) DeleteRuntimeSession(ctx context.Context) error {
	var errKind domain.ErrorKind = "runtime delete session"

	repo, err := s.db.NewRepository(ctx)
	if err != nil {
		return errKind.WrapError(err, "new repository")
	}

	defer repo.Close(ctx)

	if err := repo.DeleteRuntimeSession(ctx); err != nil {
		return errKind.WrapError(err, "repo - delete runtime session")
	}

	if err := repo.Commit(ctx); err != nil {
		return errKind.WrapError(err, "commit")
	}

	return nil
}
