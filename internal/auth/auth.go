package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

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

// jwk represents a single JSON Web Key from the JWKS endpoint.
type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	// EC fields
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

type jwksResponse struct {
	Keys []jwk `json:"keys"`
}

// keystore holds resolved public keys indexed by kid.
// Values are *ecdsa.PublicKey (ES256) or []byte (HS256).
var keystore = map[string]interface{}{}

// InitAuth loads JWT verification keys from the Supabase JWKS endpoint.
//   - supabaseURL: e.g. https://<ref>.supabase.co  (empty = skip JWKS fetch)
//   - legacySecret: HS256 shared secret fallback (used for tokens signed before ECC migration)
func InitAuth(supabaseURL, legacySecret string) error {
	// Register the legacy HS256 secret so old tokens still work.
	if legacySecret != "" {
		keystore["legacy"] = []byte(legacySecret)
	}

	if supabaseURL == "" {
		if legacySecret == "" {
			return errors.New("auth: neither SUPABASE_URL nor JWT_SECRET configured")
		}
		return nil
	}

	url := strings.TrimRight(supabaseURL, "/") + "/auth/v1/.well-known/jwks.json"
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("auth: JWKS fetch failed (%s): %w", url, err)
	}
	defer resp.Body.Close()

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("auth: JWKS decode failed: %w", err)
	}

	for _, key := range jwks.Keys {
		if key.Kty == "EC" && key.Crv == "P-256" {
			pub, err := ecPublicKeyFromJWK(key)
			if err != nil {
				return fmt.Errorf("auth: bad EC key %q: %w", key.Kid, err)
			}
			id := key.Kid
			if id == "" {
				id = "ec-default"
			}
			keystore[id] = pub
		}
	}

	return nil
}

// ecPublicKeyFromJWK converts a P-256 JWK to an *ecdsa.PublicKey.
func ecPublicKeyFromJWK(key jwk) (*ecdsa.PublicKey, error) {
	xBytes, err := base64.RawURLEncoding.DecodeString(key.X)
	if err != nil {
		return nil, fmt.Errorf("X decode: %w", err)
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(key.Y)
	if err != nil {
		return nil, fmt.Errorf("Y decode: %w", err)
	}
	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}

// ValidateToken validates a Supabase JWT (ES256 or HS256 fallback) and returns claims.
func ValidateToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		kid, _ := token.Header["kid"].(string)

		switch token.Method.(type) {
		case *jwt.SigningMethodECDSA:
			// Try kid-specific key first
			if kid != "" {
				if key, ok := keystore[kid]; ok {
					return key, nil
				}
			}
			// Fallback: first EC key in store
			for _, v := range keystore {
				if ecKey, ok := v.(*ecdsa.PublicKey); ok {
					return ecKey, nil
				}
			}
			return nil, errors.New("auth: no EC public key for ES256 token")

		case *jwt.SigningMethodHMAC:
			if kid != "" {
				if key, ok := keystore[kid]; ok {
					return key, nil
				}
			}
			if key, ok := keystore["legacy"]; ok {
				return key, nil
			}
			return nil, errors.New("auth: no HMAC secret for HS256 token")

		default:
			return nil, fmt.Errorf("auth: unexpected signing method %v", token.Header["alg"])
		}
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("auth: token invalid")
	}
	return claims, nil
}

// Middleware verifies Supabase JWT access tokens and injects client_id into context.
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
		ctx := context.WithValue(r.Context(), ClientIDKey, claims.Sub)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetClientID extracts the Supabase user UUID from request context.
func GetClientID(ctx context.Context) (string, bool) {
	clientID, ok := ctx.Value(ClientIDKey).(string)
	return clientID, ok
}
