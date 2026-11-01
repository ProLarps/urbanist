// Package auth issues and verifies player/admin tokens. Tokens are JWTs
// signed with HMAC-SHA256; the signing key lives in a file so tokens survive
// server restarts, as required by docs/API.md §2.1.
package auth

import (
	"crypto/rand"
	"os"
	"path/filepath"
)

// Role distinguishes the two token kinds the server issues.
const (
	RolePlayer = "player"
	RoleAdmin  = "admin"
)

// loadOrCreateSecret reads the signing key from path. A missing or too-short
// file is replaced with 32 fresh random bytes (created with 0600 on purpose).
func loadOrCreateSecret(path string) ([]byte, error) {
	if data, err := os.ReadFile(path); err == nil && len(data) >= 32 {
		return data, nil
	}

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	if err := os.WriteFile(path, secret, 0o600); err != nil {
		return nil, err
	}
	return secret, nil
}
