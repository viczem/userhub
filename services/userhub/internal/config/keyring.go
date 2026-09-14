package config

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

const (
	// KeyringKeySize is the required HMAC and AES-256 key size in bytes.
	KeyringKeySize = 32
)

// Keyring contains one active key and, during rotation, one previous key.
type Keyring struct {
	ActiveID int16
	Keys     map[int16][KeyringKeySize]byte
}

// Empty reports whether the keyring is not configured.
func (ring Keyring) Empty() bool {
	return len(ring.Keys) == 0
}

func parseKeyring(value string) (Keyring, error) {
	if value == "" {
		return Keyring{}, nil
	}

	ring := Keyring{
		Keys: make(map[int16][KeyringKeySize]byte),
	}

	entries := strings.SplitSeq(value, ",")
	for entry := range entries {
		idText, materialText, ok := strings.Cut(entry, ":")
		if !ok || idText == "" || materialText == "" {
			return Keyring{}, newError("has an invalid keyring format")
		}

		// Parse as 16 bits because key IDs use the PostgreSQL SMALLINT range.
		rawID, err := strconv.ParseInt(idText, 10, 16)
		// -32768 has no corresponding positive SMALLINT value, so its absolute
		// value cannot be represented as an int16 key ID.
		if err != nil {
			return Keyring{}, wrapError(err, "parse key ID")
		}

		if rawID == 0 || rawID == -1<<15 {
			return Keyring{}, newError("contains an invalid key ID")
		}

		active := rawID > 0
		if active && ring.ActiveID != 0 {
			return Keyring{}, newError("contains multiple active keys")
		}

		normalizedID := rawID
		if normalizedID < 0 {
			normalizedID = -normalizedID
		}

		id := int16(normalizedID)
		if _, duplicate := ring.Keys[id]; duplicate {
			return Keyring{}, newError("contains a duplicate key ID")
		}

		decoded, err := base64.URLEncoding.Strict().DecodeString(materialText)
		if err != nil {
			return Keyring{}, wrapError(err, fmt.Sprintf("key decode %s", materialText))
		}

		if len(decoded) != KeyringKeySize {
			return Keyring{}, newError("contains invalid key material")
		}

		var material [KeyringKeySize]byte
		copy(material[:], decoded)

		ring.Keys[id] = material

		if active {
			ring.ActiveID = id
		}
	}

	if ring.ActiveID == 0 {
		return Keyring{}, newError("does not contain an active key")
	}

	return ring, nil
}
