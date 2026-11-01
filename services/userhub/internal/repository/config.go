package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/viczem/userhub/services/userhub/internal/domain"
)

type userRoleRow struct {
	ID          int32  `db:"id"`
	Name        string `db:"name"`
	Description string `db:"description"`
}

// ReadUsersRoles reads the configured user roles from the database.
func (repo Repository) ReadUsersRoles(ctx context.Context) ([]domain.UserRole, error) {
	rows, err := repo.tx.Query(ctx, "SELECT id, name, description FROM roles")
	if err != nil {
		return nil, err
	}

	roleRows, err := pgx.CollectRows(
		rows,
		pgx.RowToStructByName[userRoleRow],
	)
	if err != nil {
		return nil, fmt.Errorf("collect user roles: %w", err)
	}

	defer rows.Close()

	var roles []domain.UserRole
	for _, roleRow := range roleRows {
		roles = append(roles, domain.UserRole{
			ID:          roleRow.ID,
			Name:        roleRow.Name,
			Description: roleRow.Description,
		})
	}

	return roles, nil
}

// ReadUsersRole reads a configured user role by ID.
func (repo Repository) ReadUsersRole(ctx context.Context, roleID int32) (domain.UserRole, error) {
	var role domain.UserRole

	err := repo.tx.QueryRow(ctx,
		"SELECT id, name, description FROM roles WHERE id = $1", roleID,
	).Scan(&role.ID, &role.Name, &role.Description)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.UserRole{}, domain.ErrUsersRoleNotFound
		}

		return domain.UserRole{}, fmt.Errorf("read user role: %w", err)
	}

	return role, nil
}

// LockUsersRoles serializes role checks and updates against concurrent role writes until the transaction ends.
func (repo Repository) LockUsersRoles(ctx context.Context) error {
	_, err := repo.tx.Exec(ctx, "LOCK TABLE roles IN SHARE ROW EXCLUSIVE MODE")
	if err != nil {
		return fmt.Errorf("lock user roles: %w", err)
	}

	return nil
}

// UpdateUsersRole replaces all mutable role fields, targeting the role by ID.
func (repo Repository) UpdateUsersRole(ctx context.Context, role domain.UserRole) error {
	const query = `
		UPDATE roles
		SET name = $1, description = $2
		WHERE id = $3
	`

	result, err := repo.tx.Exec(
		ctx,
		query,
		role.Name,
		role.Description,
		role.ID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "roles_name_key" {
			return fmt.Errorf("%w: %w", domain.ErrUsersRoleConflict, err)
		}

		return fmt.Errorf("update user role: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrUsersRoleNotFound
	}

	return nil
}

// DeleteUsersRole removes a user role from the database.
func (repo Repository) DeleteUsersRole(ctx context.Context, roleID int32) error {
	_, err := repo.tx.Exec(ctx, "DELETE FROM roles WHERE id = $1", roleID)
	if err != nil {
		return fmt.Errorf("delete user role: %w", err)
	}

	return nil
}

// ExistsUsersRole checks if a user role exists in the database.
func (repo Repository) ExistsUsersRole(ctx context.Context, name string) (bool, error) {
	var exists bool

	err := repo.tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM roles WHERE name = $1)", name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user role existence: %w", err)
	}

	return exists, nil
}

// CreateUsersRole creates a new user role in the database.
func (repo Repository) CreateUsersRole(ctx context.Context, role domain.UserRole) (domain.UserRole, error) {
	err := repo.tx.QueryRow(
		ctx,
		"INSERT INTO roles (name, description) VALUES ($1, $2) RETURNING id",
		role.Name,
		role.Description,
	).Scan(&role.ID)
	if err != nil {
		return domain.UserRole{}, fmt.Errorf("create user role: %w", err)
	}

	return role, nil
}
