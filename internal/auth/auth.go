package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const ClientIDKey contextKey = "client_id"

// Claims defines the JWT claims structure returned by Supabase GoTrue.
type Claims struct {
	Sub   string `json:"sub"` // Supabase user UUID
	Email string `json:"email"`
	jwt.RegisteredClaims
}

var jwtSecret []byte

// InitAuth initializes the JWT secret key (Supabase JWT Secret).
func InitAuth(secret string) {
	jwtSecret = []byte(secret)
}

// ValidateToken validates a Supabase JWT token and returns claims.
func ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// Middleware verifies Supabase JWT access tokens and injects user sub (UUID) as client_id into context.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"authorization header required"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		claims, err := ValidateToken(parts[1])
		if err != nil {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Inject sub claim (UUID) as client_id context key
		ctx := context.WithValue(r.Context(), ClientIDKey, claims.Sub)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetClientID extracts client_id (Supabase sub UUID) from request context.
func GetClientID(ctx context.Context) (string, bool) {
	clientID, ok := ctx.Value(ClientIDKey).(string)
	return clientID, ok
}
