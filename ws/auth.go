package ws

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Token payload, signed with an HMAC-SHA256 server secret; see docs/API.md §2.1.
type tokenPayload struct {
	PID  string `json:"pid"`
	Name string `json:"name"`
	Room string `json:"room"`
	Exp  int64  `json:"exp"`
}

// Authenticator issues and verifies opaque player tokens.
type Authenticator struct {
	secret []byte
	ttl    int64
}

func NewAuthenticator(secretPath string, ttl int64) (*Authenticator, error) {
	secret, err := loadOrCreateSecret(secretPath)
	if err != nil {
		return nil, err
	}
	return &Authenticator{secret: secret, ttl: ttl}, nil
}

// loadOrCreateSecret reads the HMAC secret from file; tokens therefore
// survive restarts. Missing file -> a fresh random secret is written.
func loadOrCreateSecret(path string) ([]byte, error) {
	if data, err := os.ReadFile(path); err == nil && len(data) >= 32 {
		return data, nil
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return nil, err
	}
	if err := os.WriteFile(path, secret, 0o600); err != nil {
		return nil, err
	}
	return secret, nil
}

// Issue returns "b64url(payload).b64url(hmac)" safe for URLs and headers.
func (a *Authenticator) Issue(pid, name, room string) (string, error) {
	payload, err := json.Marshal(tokenPayload{
		PID:  pid,
		Name: name,
		Room: room,
		Exp:  time.Now().Unix() + a.ttl,
	})
	if err != nil {
		return "", err
	}
	sig := a.sign(payload)
	return base64.RawURLEncoding.EncodeToString(payload) + "." +
		base64.RawURLEncoding.EncodeToString(sig), nil
}

func (a *Authenticator) sign(payload []byte) []byte {
	mac := hmac.New(sha256.New, a.secret)
	mac.Write(payload)
	return mac.Sum(nil)
}

// Verify parses, checks the deadline and confirms the signature. Errors are
// deliberately undistinguishable from outside: any -> "bad token".
func (a *Authenticator) Verify(token string) (*tokenPayload, error) {
	payloadB64, sigB64, ok := strings.Cut(token, ".")
	if !ok {
		return nil, errors.New("missing signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, errors.New("malformed payload")
	}
	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, errors.New("malformed signature")
	}
	want := a.sign(payload)
	if subtle.ConstantTimeCompare(sig, want) != 1 {
		return nil, errors.New("bad signature")
	}
	var p tokenPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, errors.New("malformed payload")
	}
	if time.Now().Unix() >= p.Exp {
		return nil, errors.New("expired")
	}
	return &p, nil
}
