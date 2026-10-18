package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/doanbenson/momentos/services/gateway/internal/business"
)

func TestListMomentsRequiresAuth(t *testing.T) {
	mux := NewMux(Deps{
		BFF:       business.NewBFF("http://moment.invalid", "", "", "internal"),
		JWTConfig: JWTConfig{Secret: []byte("test-secret"), Audience: "authenticated"},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/moments", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestListMomentsProxiesAuthenticatedUser(t *testing.T) {
	const body = `[{"id":"11111111-1111-1111-1111-111111111111","status":"ready","createdAt":"2026-01-02T00:00:00Z","songPickId":"live-a-life-you-will-remember"}]`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/moments" {
			t.Errorf("upstream %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("X-Momentos-Principal") != "user-1" {
			t.Errorf("principal %q", r.Header.Get("X-Momentos-Principal"))
			http.Error(w, "principal", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(upstream.Close)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-1",
		"aud": "authenticated",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}

	mux := NewMux(Deps{
		BFF:       business.NewBFF(upstream.URL, "", "", "internal"),
		JWTConfig: JWTConfig{Secret: []byte("test-secret"), Audience: "authenticated"},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/moments", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != body {
		t.Fatalf("body = %s", rec.Body.String())
	}
}
