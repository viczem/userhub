package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

// DeleteRuntimeSession removes the current runtime session.
func (repo Repository) DeleteRuntimeSession(ctx context.Context) error {
	const query = `DELETE FROM runtime WHERE key = $1`

	if _, err := repo.tx.Exec(ctx, query, runtimeSessionKey); err != nil {
		return fmt.Errorf("delete runtime session: %w", err)
	}

	return nil
}
