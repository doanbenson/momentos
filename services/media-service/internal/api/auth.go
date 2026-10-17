package api

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
)

type ctxKey int

const (
	ctxPrincipal ctxKey = iota
)

const (
	headerPrincipal     = "X-Momentos-Principal"
	headerInternalToken = "X-Momentos-Internal-Token"
)

// Auth matches moment-api trust model:
// gateway verifies Supabase JWT, then forwards
//   Authorization: Bearer <INTERNAL_SERVICE_TOKEN>
//   X-Momentos-Principal: <sub>
// Internal workers may use Bearer or X-Momentos-Internal-Token without a principal.
type Auth struct {
	InternalToken string
}

func (a *Auth) requireInternal(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.tokenOK(r) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid or missing internal token")
			return
		}
		next(w, r)
	}
}

func (a *Auth) requirePrincipal(next http.HandlerFunc) http.HandlerFunc {
	return a.requireInternal(func(w http.ResponseWriter, r *http.Request) {
		principal := strings.TrimSpace(r.Header.Get(headerPrincipal))
		if principal == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing X-Momentos-Principal")
			return
		}
		ctx := context.WithValue(r.Context(), ctxPrincipal, principal)
		next(w, r.WithContext(ctx))
	})
}

func (a *Auth) tokenOK(r *http.Request) bool {
	if a.InternalToken == "" {
		return false
	}
	want := a.InternalToken
	if tok := bearerToken(r.Header.Get("Authorization")); tok != "" {
		return secureEqual(tok, want)
	}
	if tok := strings.TrimSpace(r.Header.Get(headerInternalToken)); tok != "" {
		return secureEqual(tok, want)
	}
	return false
}

func secureEqual(got, want string) bool {
	if len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func bearerToken(h string) string {
	const p = "Bearer "
	if len(h) < len(p) || !strings.EqualFold(h[:len(p)], p) {
		return ""
	}
	return strings.TrimSpace(h[len(p):])
}

func principal(r *http.Request) string {
	v, _ := r.Context().Value(ctxPrincipal).(string)
	return v
}
