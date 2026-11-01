package routes

import (
	"context"
	"net/http"
	"strings"

	"github.com/prolarps/urbanist/backend/auth"
	"github.com/prolarps/urbanist/backend/errcodes"
)

type ctxKey int

const claimsKey ctxKey = iota

// bearerOf extracts "Authorization: Bearer <token>".
func bearerOf(r *http.Request) string {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return ""
	}
	return token
}

// requireAuth turns a token into claims or answers 401; admin-role tokens
// are rejected here because player endpoints are player-only.
func requireAuth(authMan *auth.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := verifyBearer(authMan, r, auth.RolePlayer)
			if !ok {
				sendError(w, errcodes.Unauthorized, "Bearer player token required")
				return
			}
			next.ServeHTTP(w, r.WithContext(withClaims(r, claims)))
		})
	}
}

// requireAdmin guards the administrator panel (docs/API.md §2.7).
func requireAdmin(authMan *auth.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := verifyBearer(authMan, r, auth.RoleAdmin)
			if !ok {
				sendError(w, errcodes.Forbidden, "admin token required")
				return
			}
			next.ServeHTTP(w, r.WithContext(withClaims(r, claims)))
		})
	}
}

func verifyBearer(authMan *auth.Manager, r *http.Request, role string) (*auth.Claims, bool) {
	claims, err := authMan.Verify(bearerOf(r))
	return claims, err == nil && claims.Role == role
}

func withClaims(r *http.Request, claims *auth.Claims) context.Context {
	return context.WithValue(r.Context(), claimsKey, claims)
}

// claimsOf pulls the caller identity out of the request context.
func claimsOf(r *http.Request) *auth.Claims {
	claims, _ := r.Context().Value(claimsKey).(*auth.Claims)
	return claims
}
