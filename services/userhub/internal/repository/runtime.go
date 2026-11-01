package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/viczem/userhub/services/userhub/internal/domain"
)

const runtimeSessionKey = "config_session"

type runtimeSessionValue struct {
	TokenHMAC  string    `json:"token_hmac"`
	TokenKeyID int16     `json:"token_key_id"`
	ExpiresAt  time.Time `json:"expires_at"`
	ValidUntil time.Time `json:"valid_until"`
}

// CreateRuntimeSession replaces the current runtime session.
func (repo Repository) CreateRuntimeSession(ctx context.Context, session *domain.RuntimeSession) error {
	value, err := json.Marshal(runtimeSessionValue{
		TokenHMAC:  session.TokenHMAC,
		TokenKeyID: session.TokenKeyID,
		ExpiresAt:  session.ExpiresAt,
		ValidUntil: session.ValidUntil,
	})
	if err != nil {
		return fmt.Errorf("marshal runtime session: %w", err)
	}

	const query = `
		INSERT INTO runtime (key, value)
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value
	`

	if _, err := repo.tx.Exec(ctx, query, runtimeSessionKey, string(value)); err != nil {
		return fmt.Errorf("upsert runtime session: %w", err)
	}

	return nil
}

// ReadRuntimeSession returns the current runtime session, or nil if none exists.
func (repo Repository) ReadRuntimeSession(ctx context.Context) (*domain.RuntimeSession, error) {
	token, ok := ctx.Value(domain.ConfigSessionTokenKey{}).(string)
	if !ok {
		return nil, domain.ErrConfigSession.NewError("context have no token")
	}

	const query = `SELECT value FROM runtime WHERE key = $1`

	var value string
	if err := repo.tx.QueryRow(ctx, query, runtimeSessionKey).Scan(&value); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrConfigSession.NewError("not found")
		}
		return nil, fmt.Errorf("read runtime session: %w", err)
	}

	var stored runtimeSessionValue
	if err := json.Unmarshal([]byte(value), &stored); err != nil {
		return nil, fmt.Errorf("unmarshal runtime session: %w", err)
	}

	return &domain.RuntimeSession{
		Token:      token,
		TokenHMAC:  stored.TokenHMAC,
		TokenKeyID: stored.TokenKeyID,
		ExpiresAt:  stored.ExpiresAt,
		ValidUntil: stored.ValidUntil,
	}, nil
}

// UpdateRuntimeSession updates the current runtime session.
func (repo Repository) UpdateRuntimeSession(ctx context.Context, session *domain.RuntimeSession) error {
	value, err := json.Marshal(runtimeSessionValue{
		TokenHMAC:  session.TokenHMAC,
		TokenKeyID: session.TokenKeyID,
		ExpiresAt:  session.ExpiresAt,
		ValidUntil: session.ValidUntil,
	})
	if err != nil {
		return fmt.Errorf("marshal runtime session: %w", err)
	}

	const query = `UPDATE runtime SET value = $1 WHERE key = $2`

	if _, err := repo.tx.Exec(ctx, query, string(value), runtimeSessionKey); err != nil {
		return fmt.Errorf("update runtime session: %w", err)
	}

	return nil
}

// DeleteRuntimeSession removes the current runtime session.
func (repo Repository) DeleteRuntimeSession(ctx context.Context) error {
	const query = `DELETE FROM runtime WHERE key = $1`

	if _, err := repo.tx.Exec(ctx, query, runtimeSessionKey); err != nil {
		return fmt.Errorf("delete runtime session: %w", err)
	}

	return nil
}
