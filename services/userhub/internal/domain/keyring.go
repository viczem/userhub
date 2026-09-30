package domain

import (
	"encoding/base64"
	"strconv"
	"strings"
)

const (
	// KeyringKeySize is the required HMAC and AES-256 key size in bytes.
	KeyringKeySize = 32
	// ErrKeyring identifies keyring-related errors.
	ErrKeyring ErrorKind = "keyring"
)

// Keyring contains one active key and, during rotation, one previous key.
type Keyring struct {
	ActiveID int16
	Keys     map[int16][KeyringKeySize]byte
}

// Key returns the key material for the specified key ID.
// It returns an error if the key ID is not found.
func (ring Keyring) Key(id int16) ([]byte, error) {
	key, ok := ring.Keys[id]
	if !ok {
		return nil, ErrKeyring.NewErrorf("key %d is not found", id)
	}

	return key[:], nil
}

// Active returns the active key material.
// It returns an error if the active key is not found.
func (ring Keyring) Active() ([]byte, error) {
	return ring.Key(ring.ActiveID)
}

// Empty reports whether the keyring has no active key or key material.
func (ring Keyring) Empty() bool {
	return ring.ActiveID == 0 || len(ring.Keys) == 0
}

// NewKeyring parses a comma-separated keyring string in the format
// "id:base64url-key-material,id:base64url-key-material". A positive key ID
// marks the active key, while a negative key ID marks a previous key; exactly
// one active key is required.
func NewKeyring(value string) (Keyring, error) {
	if value == "" {
		return Keyring{}, ErrKeyring.NewError("empty keyring value")
	}

	ring := Keyring{
		Keys: make(map[int16][KeyringKeySize]byte),
	}

	entries := strings.SplitSeq(value, ",")
	for entry := range entries {
		idText, materialText, ok := strings.Cut(entry, ":")
		if !ok || idText == "" || materialText == "" {
			return Keyring{}, ErrKeyring.NewError("invalid keyring format")
		}

		// Parse as 16 bits because key IDs use the PostgreSQL SMALLINT range.
		rawID, err := strconv.ParseInt(idText, 10, 16)
		// -32768 has no corresponding positive SMALLINT value, so its absolute
		// value cannot be represented as an int16 key ID.
		if err != nil {
			return Keyring{}, ErrKeyring.WrapError(err, "parse key ID")
		}

		if rawID == 0 || rawID == -1<<15 {
			return Keyring{}, ErrKeyring.NewError("invalid key ID")
		}

		active := rawID > 0
		if active && ring.ActiveID != 0 {
			return Keyring{}, ErrKeyring.NewError("multiple active keys")
		}

		normalizedID := rawID
		if normalizedID < 0 {
			normalizedID = -normalizedID
		}

		id := int16(normalizedID)
		if _, duplicate := ring.Keys[id]; duplicate {
			return Keyring{}, ErrKeyring.NewErrorf("duplicate key ID %d", id)
		}

		decoded, err := base64.URLEncoding.Strict().DecodeString(materialText)
		if err != nil {
			return Keyring{}, ErrKeyring.WrapErrorf(err, "key decode %s", materialText)
		}

		if len(decoded) != KeyringKeySize {
			return Keyring{}, ErrKeyring.NewError("invalid key material")
		}

		var material [KeyringKeySize]byte
		copy(material[:], decoded)

		ring.Keys[id] = material

		if active {
			ring.ActiveID = id
		}
	}

	if ring.ActiveID == 0 {
		return Keyring{}, ErrKeyring.NewError("no active key")
	}

	return ring, nil
}
