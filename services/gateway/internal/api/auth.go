package api

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
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
	JWKSURL  string
	cache    *jwksCache
}

func JWTConfigFromEnv() (JWTConfig, error) {
	base := strings.TrimRight(os.Getenv("SUPABASE_URL"), "/")
	secret := os.Getenv("SUPABASE_JWT_SECRET")
	if base == "" && secret == "" {
		return JWTConfig{}, fmt.Errorf("SUPABASE_URL or SUPABASE_JWT_SECRET is required")
	}
	aud := os.Getenv("SUPABASE_JWT_AUD")
	if aud == "" {
		aud = "authenticated"
	}
	cfg := JWTConfig{Secret: []byte(secret), Audience: aud}
	if base != "" {
		cfg.JWKSURL = base + "/auth/v1/.well-known/jwks.json"
		cfg.cache = newJWKSCache(cfg.JWKSURL)
	}
	return cfg, nil
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
		jwt.WithValidMethods([]string{
			jwt.SigningMethodHS256.Name,
			jwt.SigningMethodES256.Name,
			jwt.SigningMethodRS256.Name,
		}),
		jwt.WithExpirationRequired(),
		jwt.WithAudience(cfg.Audience),
	)
	tok, err := parser.Parse(raw, func(t *jwt.Token) (any, error) {
		switch t.Method.Alg() {
		case jwt.SigningMethodHS256.Name:
			if len(cfg.Secret) == 0 {
				return nil, fmt.Errorf("HS256 token but SUPABASE_JWT_SECRET unset")
			}
			return cfg.Secret, nil
		case jwt.SigningMethodES256.Name, jwt.SigningMethodRS256.Name:
			if cfg.cache == nil {
				return nil, fmt.Errorf("asymmetric token but SUPABASE_URL unset")
			}
			kid, _ := t.Header["kid"].(string)
			return cfg.cache.key(kid)
		default:
			return nil, fmt.Errorf("unexpected alg %s", t.Method.Alg())
		}
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

// jwksCache fetches Supabase Auth public keys for ES256/RS256 verification.
type jwksCache struct {
	url    string
	client *http.Client

	mu        sync.Mutex
	keys      map[string]any
	fetchedAt time.Time
}

func newJWKSCache(url string) *jwksCache {
	return &jwksCache{
		url:    url,
		client: &http.Client{Timeout: 10 * time.Second},
		keys:   map[string]any{},
	}
}

func (c *jwksCache) key(kid string) (any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if k, ok := c.keys[kid]; ok && kid != "" && time.Since(c.fetchedAt) < 10*time.Minute {
		return k, nil
	}
	if err := c.refreshLocked(); err != nil {
		return nil, err
	}
	if kid == "" {
		for _, k := range c.keys {
			return k, nil
		}
		return nil, fmt.Errorf("jwks empty")
	}
	k, ok := c.keys[kid]
	if !ok {
		return nil, fmt.Errorf("unknown kid %q", kid)
	}
	return k, nil
}

func (c *jwksCache) refreshLocked() error {
	req, err := http.NewRequest(http.MethodGet, c.url, nil)
	if err != nil {
		return err
	}
	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("jwks fetch: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("jwks read: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks status %d", res.StatusCode)
	}
	var set struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Alg string `json:"alg"`
			Crv string `json:"crv"`
			X   string `json:"x"`
			Y   string `json:"y"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(body, &set); err != nil {
		return fmt.Errorf("jwks json: %w", err)
	}
	next := make(map[string]any, len(set.Keys))
	for _, k := range set.Keys {
		switch k.Kty {
		case "EC":
			pub, err := ecPublicKey(k.X, k.Y, k.Crv)
			if err != nil {
				return err
			}
			next[k.Kid] = pub
		default:
			// Skip unsupported key types (e.g. RSA until needed).
			continue
		}
	}
	if len(next) == 0 {
		return fmt.Errorf("jwks had no usable keys")
	}
	c.keys = next
	c.fetchedAt = time.Now()
	return nil
}

func ecPublicKey(xB64, yB64, crv string) (*ecdsa.PublicKey, error) {
	if crv != "P-256" {
		return nil, fmt.Errorf("unsupported EC curve %q", crv)
	}
	x, err := base64.RawURLEncoding.DecodeString(xB64)
	if err != nil {
		return nil, fmt.Errorf("jwk x: %w", err)
	}
	y, err := base64.RawURLEncoding.DecodeString(yB64)
	if err != nil {
		return nil, fmt.Errorf("jwk y: %w", err)
	}
	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(x),
		Y:     new(big.Int).SetBytes(y),
	}, nil
}
