package domain

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"time"
)

const (
	apiKeySize                = 32
	runtimeSessionHMACPurpose = "userhub:config-session\x00"
)

// RuntimeSession represents a session key that a service administrator can use
// via the X-UserHub-Key header to manage certain service settings and aspects
// of its state.
type RuntimeSession struct {
	Token      string
	TokenHMAC  string
	TokenKeyID int16
	ExpiresAt  time.Time
	ValidUntil time.Time
}

// Validate checks that the runtime session is unexpired and its token HMAC is valid.
func (s *RuntimeSession) Validate(keyringHMAC Keyring) error {
	if s.Token == "" {
		return ErrConfigSession.NewError("token is empty")
	}

	if s.TokenHMAC == "" {
		return ErrConfigSession.NewError("token HMAC is empty")
	}

	if time.Now().After(s.ExpiresAt) {
		return ErrConfigSession.NewError("expired")
	}

	if time.Now().After(s.ValidUntil) {
		return ErrConfigSession.NewError("valid until expired")
	}

	key, err := keyringHMAC.Key(s.TokenKeyID)
	if err != nil {
		return ErrConfigSession.WrapError(err, "invalid key")
	}

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(runtimeSessionHMACPurpose))
	mac.Write([]byte(s.Token))

	if s.TokenHMAC != base64.RawURLEncoding.EncodeToString(mac.Sum(nil)) {
		return ErrConfigSession.NewError("token mismatch")
	}

	return nil
}

// Prolong extends the session's expiration by the idle timeout, without
// allowing it to expire after its ValidUntil time.
func (s *RuntimeSession) Prolong(idleTimeout time.Duration) error {
	s.ExpiresAt = time.Now().Add(idleTimeout)

	if s.ExpiresAt.After(s.ValidUntil) {
		return ErrConfigSession.NewError("prolongation expired")
	}

	return nil
}

func generateAPIKey(random io.Reader) (string, error) {
	material := make([]byte, apiKeySize)
	if _, err := io.ReadFull(random, material); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(material), nil
}

// NewRuntimeSession creates a runtime session with the provided idle timeout and TTL.
func NewRuntimeSession(keyringHMAC Keyring, idleTimeout, ttl time.Duration) (*RuntimeSession, error) {
	token, err := generateAPIKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	key, err := keyringHMAC.Active()
	if err != nil {
		return nil, err
	}

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(runtimeSessionHMACPurpose))
	mac.Write([]byte(token))

	now := time.Now()

	s := &RuntimeSession{
		Token:      token,
		TokenHMAC:  base64.RawURLEncoding.EncodeToString(mac.Sum(nil)),
		TokenKeyID: keyringHMAC.ActiveID,
		ExpiresAt:  now.Add(idleTimeout),
		ValidUntil: now.Add(ttl),
	}

	return s, nil
}
