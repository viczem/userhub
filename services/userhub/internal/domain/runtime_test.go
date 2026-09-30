package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"testing"
	"time"
)

func TestNewRuntimeSession(t *testing.T) {
	t.Parallel()

	const (
		activeID    int16 = 7
		idleTimeout       = 5 * time.Minute
		ttl               = 30 * time.Minute
	)

	keyring, activeKey := newRuntimeSessionTestKeyring(activeID)
	before := time.Now()

	session, err := NewRuntimeSession(keyring, idleTimeout, ttl)
	if err != nil {
		t.Fatalf("NewRuntimeSession() error = %v", err)
	}

	after := time.Now()

	if len(session.Token) != 43 {
		t.Fatalf("len(NewRuntimeSession().Token) = %d, want 43", len(session.Token))
	}

	material, err := base64.RawURLEncoding.DecodeString(session.Token)
	if err != nil {
		t.Fatalf("decode generated API key: %v", err)
	}

	if len(material) != apiKeySize {
		t.Errorf("decoded key length = %d, want %d", len(material), apiKeySize)
	}

	mac := hmac.New(sha256.New, activeKey[:])
	_, _ = mac.Write([]byte(runtimeSessionHMACPurpose))
	_, _ = mac.Write([]byte(session.Token))
	wantHMAC := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if session.TokenHMAC != wantHMAC {
		t.Errorf("TokenHMAC = %q, want %q", session.TokenHMAC, wantHMAC)
	}
	if session.TokenKeyID != activeID {
		t.Errorf("TokenKeyID = %d, want %d", session.TokenKeyID, activeID)
	}
	if session.ExpiresAt.Before(before.Add(idleTimeout)) || session.ExpiresAt.After(after.Add(idleTimeout)) {
		t.Errorf(
			"ExpiresAt = %v, want between %v and %v",
			session.ExpiresAt,
			before.Add(idleTimeout),
			after.Add(idleTimeout),
		)
	}
	if session.ValidUntil.Before(before.Add(ttl)) || session.ValidUntil.After(after.Add(ttl)) {
		t.Errorf(
			"ValidUntil = %v, want between %v and %v",
			session.ValidUntil,
			before.Add(ttl),
			after.Add(ttl),
		)
	}
}

func TestNewRuntimeSessionReturnsActiveKeyError(t *testing.T) {
	t.Parallel()

	session, err := NewRuntimeSession(Keyring{ActiveID: 7}, 0, 0)

	if session != nil {
		t.Errorf("NewRuntimeSession() session = %#v, want nil", session)
	}
	if !errors.Is(err, ErrKeyring) {
		t.Errorf("NewRuntimeSession() error = %v, want ErrKeyring", err)
	}
}

func TestGenerateAPIKeyUsesProvidedRandomSource(t *testing.T) {
	t.Parallel()

	material := make([]byte, apiKeySize)
	for i := range material {
		material[i] = byte(i)
	}

	key, err := generateAPIKey(&fixedReader{data: material})
	if err != nil {
		t.Fatalf("generateAPIKey() error = %v", err)
	}

	want := base64.RawURLEncoding.EncodeToString(material)
	if key != want {
		t.Errorf("generateAPIKey() = %q, want %q", key, want)
	}
}

func TestGenerateAPIKeyReturnsRandomSourceError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("random source failed")
	key, err := generateAPIKey(errorReader{err: wantErr})

	if key != "" {
		t.Errorf("generateAPIKey() key = %q, want empty", key)
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("generateAPIKey() error = %v, want %v", err, wantErr)
	}
}

func newRuntimeSessionTestKeyring(activeID int16) (Keyring, [KeyringKeySize]byte) {
	var activeKey [KeyringKeySize]byte
	for i := range activeKey {
		activeKey[i] = byte(i + 1)
	}

	return Keyring{
		ActiveID: activeID,
		Keys: map[int16][KeyringKeySize]byte{
			activeID: activeKey,
		},
	}, activeKey
}

type fixedReader struct {
	data []byte
}

func (r *fixedReader) Read(p []byte) (int, error) {
	return copy(p, r.data), nil
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}
