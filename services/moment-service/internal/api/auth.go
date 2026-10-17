package api

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey string

const (
	ctxPrincipal ctxKey = "principal"
)

func withPrincipal(ctx context.Context, principal string) context.Context {
	return context.WithValue(ctx, ctxPrincipal, principal)
}

func principalFrom(ctx context.Context) string {
	v, _ := ctx.Value(ctxPrincipal).(string)
	return v
}

func internalAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				writeErr(w, http.StatusInternalServerError, "INTERNAL_SERVICE_TOKEN not configured")
				return
			}
			got := bearerToken(r)
			if got == "" {
				got = r.Header.Get("X-Momentos-Internal-Token")
			}
			if got == "" || got != token {
				writeErr(w, http.StatusUnauthorized, "invalid internal token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func requirePrincipal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimSpace(r.Header.Get("X-Momentos-Principal"))
		if p == "" {
			writeErr(w, http.StatusUnauthorized, "missing X-Momentos-Principal")
			return
		}
		next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), p)))
	})
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, prefix))
}

func chain(mw ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		for i := len(mw) - 1; i >= 0; i-- {
			h = mw[i](h)
		}
		return h
	}
}
