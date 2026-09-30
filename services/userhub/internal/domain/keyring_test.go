package domain

import (
	"encoding/base64"
	"errors"
	"testing"
)

func TestNewKeyring(t *testing.T) {
	t.Parallel()

	activeMaterial := keyringTestMaterial(1)
	previousMaterial := keyringTestMaterial(101)
	value := "-3:" + encodeKeyringTestMaterial(previousMaterial) +
		",7:" + encodeKeyringTestMaterial(activeMaterial)

	ring, err := NewKeyring(value)
	if err != nil {
		t.Fatalf("NewKeyring() error = %v", err)
	}

	if ring.ActiveID != 7 {
		t.Errorf("ActiveID = %d, want 7", ring.ActiveID)
	}
	if len(ring.Keys) != 2 {
		t.Fatalf("len(Keys) = %d, want 2", len(ring.Keys))
	}
	if got := ring.Keys[7]; got != activeMaterial {
		t.Errorf("Keys[7] = %v, want %v", got, activeMaterial)
	}
	if got := ring.Keys[3]; got != previousMaterial {
		t.Errorf("Keys[3] = %v, want %v", got, previousMaterial)
	}
	if ring.Empty() {
		t.Error("Empty() = true, want false")
	}
}

func TestNewKeyringWithSingleActiveKey(t *testing.T) {
	t.Parallel()

	material := keyringTestMaterial(1)
	value := "1:" + encodeKeyringTestMaterial(material)

	ring, err := NewKeyring(value)
	if err != nil {
		t.Fatalf("NewKeyring() error = %v", err)
	}

	if ring.ActiveID != 1 {
		t.Errorf("ActiveID = %d, want 1", ring.ActiveID)
	}
	if len(ring.Keys) != 1 {
		t.Fatalf("len(Keys) = %d, want 1", len(ring.Keys))
	}
	if got := ring.Keys[1]; got != material {
		t.Errorf("Keys[1] = %v, want %v", got, material)
	}
}

func TestNewKeyringErrors(t *testing.T) {
	t.Parallel()

	validMaterial := encodeKeyringTestMaterial(keyringTestMaterial(1))
	shortMaterial := encodeKeyringTestMaterial([KeyringKeySize]byte{})
	shortMaterial = shortMaterial[:len(shortMaterial)-4]

	tests := []struct {
		name        string
		value       string
		wantMessage string
	}{
		{
			name:        "empty value",
			value:       "",
			wantMessage: "empty keyring value",
		},
		{
			name:        "missing separator",
			value:       "1" + validMaterial,
			wantMessage: "invalid keyring format",
		},
		{
			name:        "empty key ID",
			value:       ":" + validMaterial,
			wantMessage: "invalid keyring format",
		},
		{
			name:        "empty key material",
			value:       "1:",
			wantMessage: "invalid keyring format",
		},
		{
			name:        "non-numeric key ID",
			value:       "invalid:" + validMaterial,
			wantMessage: "parse key ID",
		},
		{
			name:        "key ID overflow",
			value:       "32768:" + validMaterial,
			wantMessage: "parse key ID",
		},
		{
			name:        "zero key ID",
			value:       "0:" + validMaterial,
			wantMessage: "invalid key ID",
		},
		{
			name:        "minimum key ID",
			value:       "-32768:" + validMaterial,
			wantMessage: "invalid key ID",
		},
		{
			name:        "multiple active keys",
			value:       "1:" + validMaterial + ",2:" + validMaterial,
			wantMessage: "multiple active keys",
		},
		{
			name:        "duplicate normalized key ID",
			value:       "-1:" + validMaterial + ",1:" + validMaterial,
			wantMessage: "duplicate key ID 1",
		},
		{
			name:        "invalid base64",
			value:       "1:not-base64!",
			wantMessage: "key decode not-base64!",
		},
		{
			name:        "invalid key material size",
			value:       "1:" + shortMaterial,
			wantMessage: "invalid key material",
		},
		{
			name:        "no active key",
			value:       "-1:" + validMaterial,
			wantMessage: "no active key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ring, err := NewKeyring(tt.value)

			if !ring.Empty() {
				t.Errorf("NewKeyring() keyring = %#v, want empty", ring)
			}
			if !errors.Is(err, ErrKeyring) {
				t.Fatalf("NewKeyring() error = %v, want ErrKeyring", err)
			}

			var domainErr *Error
			if !errors.As(err, &domainErr) {
				t.Fatalf("NewKeyring() error type = %T, want *Error", err)
			}
			if domainErr.Message != tt.wantMessage {
				t.Errorf("error message = %q, want %q", domainErr.Message, tt.wantMessage)
			}
		})
	}
}

func keyringTestMaterial(seed byte) [KeyringKeySize]byte {
	var material [KeyringKeySize]byte
	for i := range material {
		material[i] = seed + byte(i)
	}

	return material
}

func encodeKeyringTestMaterial(material [KeyringKeySize]byte) string {
	return base64.URLEncoding.EncodeToString(material[:])
}
