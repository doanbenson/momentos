package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/doanbenson/momentos/services/moment-service/internal/business"
)

func TestListMomentsRouteRequiresPrincipal(t *testing.T) {
	mux := NewMux(Deps{Svc: &business.Service{}, InternalToken: "secret"})
	req := httptest.NewRequest(http.MethodGet, "/v1/moments", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}
