package service

import (
	"context"
	"fmt"

	"github.com/viczem/userhub/services/userhub/internal/domain"
)

// UserRolePatch describes supplied fields of a partial role update.
// Nil fields leave the corresponding role values unchanged.
type UserRolePatch struct {
	Name        *string
	Description *string
}

type configRepository interface {
	ReadRuntimeSession(ctx context.Context) (*domain.RuntimeSession, error)
	UpdateRuntimeSession(ctx context.Context, session *domain.RuntimeSession) error
	ReadUsersRoles(ctx context.Context) ([]domain.UserRole, error)
	ReadUsersRole(ctx context.Context, roleID int32) (domain.UserRole, error)
	ExistsUsersRole(ctx context.Context, name string) (bool, error)
	CreateUsersRole(ctx context.Context, role domain.UserRole) (domain.UserRole, error)
	DeleteUsersRole(ctx context.Context, roleID int32) error
	LockUsersRoles(ctx context.Context) error
	UpdateUsersRole(ctx context.Context, role domain.UserRole) error
}

func (s Service[R]) validateConfigSession(ctx context.Context, repo R) error {
	session, err := repo.ReadRuntimeSession(ctx)
	if err != nil {
		return err
	}

	if session == nil {
		return domain.ErrConfigSession.NewError("not found")
	}

	if err := session.Validate(s.cfg.KeyringHMAC); err != nil {
		if err := repo.DeleteRuntimeSession(ctx); err != nil {
			return err
		}

		if err := repo.Commit(ctx); err != nil {
			return err
		}

		return err
	}

	if err := session.Prolong(s.cfg.ConfigSessionIdleTimeout); err != nil {
		if err := repo.DeleteRuntimeSession(ctx); err != nil {
			return err
		}

		if err := repo.Commit(ctx); err != nil {
			return err
		}

		return err
	}

	if err := repo.UpdateRuntimeSession(ctx, session); err != nil {
		return err
	}

	return nil
}

// ReadUsersRoles returns the configured user roles after validating the
// runtime configuration session.
//
// Errors:
//   - domain.ErrConfigSession: the runtime configuration session is invalid or
//     cannot be read, prolonged, or updated.
//   - Other repository errors: repository creation, role reading, or commit
//     failed.
func (s Service[R]) ReadUsersRoles(ctx context.Context) ([]domain.UserRole, error) {
	repo, err := s.db.NewRepository(ctx)
	if err != nil {
		return nil, err
	}

	defer repo.Close(ctx)

	if err := s.validateConfigSession(ctx, repo); err != nil {
		return nil, err
	}

	roles, err := repo.ReadUsersRoles(ctx)
	if err != nil {
		return nil, err
	}

	if err := repo.Commit(ctx); err != nil {
		return nil, err
	}

	return roles, nil
}

// CreateUsersRole creates a user role after validating the runtime configuration
// session.
//
// Errors:
//   - domain.ErrConfigSession: the runtime configuration session is invalid or
//     cannot be read, prolonged, or updated.
//   - domain.ErrUserRoleConflict: the role already exists.
//   - Other repository errors: repository creation, role existence checking,
//     role creation, or commit failed.
func (s Service[R]) CreateUsersRole(ctx context.Context, role domain.UserRole) (domain.UserRole, error) {
	repo, err := s.db.NewRepository(ctx)
	if err != nil {
		return domain.UserRole{}, err
	}

	defer repo.Close(ctx)

	if err := s.validateConfigSession(ctx, repo); err != nil {
		return domain.UserRole{}, err
	}

	exists, err := repo.ExistsUsersRole(ctx, role.Name)
	if err != nil {
		return domain.UserRole{}, err
	}
	if exists {
		return domain.UserRole{}, domain.ErrUsersRoleConflict
	}

	role, err = repo.CreateUsersRole(ctx, role)
	if err != nil {
		return domain.UserRole{}, err
	}

	if err := repo.Commit(ctx); err != nil {
		return domain.UserRole{}, err
	}

	return role, nil
}

// PatchUsersRole applies supplied fields to a role selected by its ID.
func (s Service[R]) PatchUsersRole(ctx context.Context, roleID int32, patch UserRolePatch) (domain.UserRole, error) {
	repo, err := s.db.NewRepository(ctx)
	if err != nil {
		return domain.UserRole{}, fmt.Errorf("begin user role patch: %w", err)
	}

	defer repo.Close(ctx)

	if err := s.validateConfigSession(ctx, repo); err != nil {
		return domain.UserRole{}, err
	}

	// Protect the name check and partial update from concurrent role writes.
	if err := repo.LockUsersRoles(ctx); err != nil {
		return domain.UserRole{}, err
	}

	role, err := repo.ReadUsersRole(ctx, roleID)
	if err != nil {
		return domain.UserRole{}, err
	}

	if patch.Name != nil && *patch.Name != role.Name {
		exists, err := repo.ExistsUsersRole(ctx, *patch.Name)
		if err != nil {
			return domain.UserRole{}, err
		}
		if exists {
			return domain.UserRole{}, domain.ErrUsersRoleConflict
		}
		role.Name = *patch.Name
	}

	if patch.Description != nil {
		role.Description = *patch.Description
	}

	if err := repo.UpdateUsersRole(ctx, role); err != nil {
		return domain.UserRole{}, err
	}

	if err := repo.Commit(ctx); err != nil {
		return domain.UserRole{}, fmt.Errorf("commit user role patch: %w", err)
	}

	return role, nil
}

// DeleteUsersRole deletes a user role after validating the runtime configuration
// session.
//
// Errors:
//   - domain.ErrConfigSession: the runtime configuration session is invalid or
//     cannot be read, prolonged, or updated.
//   - domain.ErrUsersRoleNotFound: the role does not exist.
//   - Other repository errors: repository creation, role reading, role deletion,
//     or commit failed.
func (s Service[R]) DeleteUsersRole(ctx context.Context, roleID int32) error {
	repo, err := s.db.NewRepository(ctx)
	if err != nil {
		return err
	}

	defer repo.Close(ctx)

	if err := s.validateConfigSession(ctx, repo); err != nil {
		return err
	}

	if _, err := repo.ReadUsersRole(ctx, roleID); err != nil {
		return err
	}

	if err := repo.DeleteUsersRole(ctx, roleID); err != nil {
		return err
	}

	if err := repo.Commit(ctx); err != nil {
		return err
	}

	return nil
}
