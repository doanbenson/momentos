package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const ctxPrincipal ctxKey = "principal"

const PrincipalHeader = "X-Momentos-Principal"

func withPrincipal(ctx context.Context, sub string) context.Context {
	return context.WithValue(ctx, ctxPrincipal, sub)
}

func Principal(ctx context.Context) string {
	v, _ := ctx.Value(ctxPrincipal).(string)
	return v
}

type JWTConfig struct {
	Secret   []byte
	Audience string
}

func JWTConfigFromEnv() (JWTConfig, error) {
	secret := os.Getenv("SUPABASE_JWT_SECRET")
	if secret == "" {
		return JWTConfig{}, fmt.Errorf("SUPABASE_JWT_SECRET is required")
	}
	aud := os.Getenv("SUPABASE_JWT_AUD")
	if aud == "" {
		aud = "authenticated"
	}
	return JWTConfig{Secret: []byte(secret), Audience: aud}, nil
}

func supabaseJWT(cfg JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := bearerToken(r)
			if raw == "" {
				writeErr(w, http.StatusUnauthorized, "missing bearer token")
				return
			}
			sub, err := verifySupabaseJWT(raw, cfg)
			if err != nil {
				writeErr(w, http.StatusUnauthorized, "invalid token")
				return
			}
			// Overwrite any inbound principal spoof.
			r.Header.Del(PrincipalHeader)
			r.Header.Set(PrincipalHeader, sub)
			next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), sub)))
		})
	}
}

func verifySupabaseJWT(raw string, cfg JWTConfig) (string, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithExpirationRequired(),
		jwt.WithAudience(cfg.Audience),
	)
	tok, err := parser.Parse(raw, func(t *jwt.Token) (any, error) {
		return cfg.Secret, nil
	})
	if err != nil || !tok.Valid {
		return "", fmt.Errorf("parse: %w", err)
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("claims")
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", fmt.Errorf("missing sub")
	}
	// Extra exp check for clarity (parser already validates).
	if exp, err := claims.GetExpirationTime(); err == nil && exp != nil && exp.Before(time.Now().Add(-30*time.Second)) {
		return "", fmt.Errorf("expired")
	}
	return sub, nil
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, prefix))
}
