package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the payload carried by every token the server signs.
type Claims struct {
	PID  string // player id, empty for admin tokens
	Name string // player name, empty for admin tokens
	Role string // RolePlayer or RoleAdmin
}

// Manager signs and verifies tokens with one persistent HMAC key.
type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(secretPath string, ttl time.Duration) (*Manager, error) {
	secret, err := loadOrCreateSecret(secretPath)
	if err != nil {
		return nil, err
	}
	return &Manager{secret: secret, ttl: ttl}, nil
}

func (m *Manager) sign(claims Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"pid":  claims.PID,
		"name": claims.Name,
		"role": claims.Role,
		"exp":  time.Now().Add(m.ttl).Unix(),
	})
	return token.SignedString(m.secret)
}

// IssuePlayer returns a player token; profiles live in the db package.
func (m *Manager) IssuePlayer(pid, name string) (string, error) {
	return m.sign(Claims{PID: pid, Name: name, Role: RolePlayer})
}

// IssueAdmin signs an admin-panel token; no player identity is attached.
func (m *Manager) IssueAdmin() (string, error) {
	return m.sign(Claims{Role: RoleAdmin})
}

// Verify parses the token, checks the deadline and the signature. Any
// problem becomes one indistinguishable error: from outside, just "bad token".
func (m *Manager) Verify(token string) (*Claims, error) {
	parsed, err := jwt.Parse(token, keyFunc(m.secret),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !parsed.Valid {
		return nil, errors.New("bad token")
	}
	return claimsFrom(parsed)
}

// claimsFrom pulls our custom fields out of a validated token.
func claimsFrom(token *jwt.Token) (*Claims, error) {
	values, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("bad token")
	}
	role, _ := values["role"].(string)
	if role != RolePlayer && role != RoleAdmin {
		return nil, errors.New("bad token")
	}
	return &Claims{
		PID:  stringArg(values, "pid"),
		Name: stringArg(values, "name"),
		Role: role,
	}, nil
}

func keyFunc(secret []byte) jwt.Keyfunc {
	return func(t *jwt.Token) (any, error) { return secret, nil }
}

func stringArg(values jwt.MapClaims, key string) string {
	s, _ := values[key].(string)
	return s
}
