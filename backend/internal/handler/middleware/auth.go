package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	jwtpkg "github.com/Shah-Aayush/task-flow/backend/internal/auth"
	"github.com/google/uuid"
)

// contextKey is an unexported type for context keys in this package.
// Using a named type prevents key collisions with other packages.
type contextKey string

const claimsKey contextKey = "user_claims"

// writeUnauthorized writes a 401 JSON response directly — avoids importing
// the handler package (which would create an import cycle since handler imports middleware).
func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}

// Auth returns JWT validation middleware. It:
//  1. Extracts the Bearer token from the Authorization header
//  2. Validates the token using the JWT secret
//  3. Injects the typed Claims into the request context
//  4. Returns 401 for any failure — no information about WHY the token is invalid
//
// The secret is captured via closure — no global state.
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				writeUnauthorized(w)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := jwtpkg.ValidateToken(tokenStr, jwtSecret)
			if err != nil {
				writeUnauthorized(w)
				return
			}

			// Inject claims into context for downstream handlers
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaims retrieves the JWT claims injected by the Auth middleware.
// Returns nil if the claims are not in the context (should not happen in
// protected routes, but safe to handle).
func GetClaims(ctx context.Context) *jwtpkg.Claims {
	claims, _ := ctx.Value(claimsKey).(*jwtpkg.Claims)
	return claims
}

// RequireAuth is a helper used in handlers to extract claims and fail fast.
// Returns (claims, true) on success, writes 401 and returns (nil, false) on failure.
func RequireAuth(w http.ResponseWriter, r *http.Request) (*jwtpkg.Claims, bool) {
	claims := GetClaims(r.Context())
	if claims == nil || claims.UserID == uuid.Nil {
		writeUnauthorized(w)
		return nil, false
	}
	return claims, true
}
