package api

import (
	"net/http"

	"github.com/doanbenson/momentos/services/gateway/internal/business"
)

type Deps struct {
	BFF       *business.BFF
	JWTConfig JWTConfig
}

func NewMux(d Deps) http.Handler {
	h := &Handler{BFF: d.BFF}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	auth := supabaseJWT(d.JWTConfig)
	mux.Handle("GET /v1/moments", auth(http.HandlerFunc(h.listMoments)))
	mux.Handle("POST /v1/moments", auth(http.HandlerFunc(h.createMoment)))
	mux.Handle("GET /v1/moments/{momentId}", auth(http.HandlerFunc(h.getMoment)))
	mux.Handle("GET /v1/moments/{momentId}/cut", auth(http.HandlerFunc(h.getCut)))
	mux.Handle("PATCH /v1/moments/{momentId}/cut", auth(http.HandlerFunc(h.patchCut)))

	// media-service proxies for Week Intake
	mux.Handle("POST /v1/assets", auth(http.HandlerFunc(h.proxyMedia)))
	mux.Handle("GET /v1/assets", auth(http.HandlerFunc(h.proxyMedia)))
	mux.Handle("GET /v1/assets/{assetId}", auth(http.HandlerFunc(h.proxyMedia)))
	mux.Handle("POST /v1/assets/{assetId}/complete", auth(http.HandlerFunc(h.proxyMedia)))
	mux.Handle("DELETE /v1/assets/{assetId}", auth(http.HandlerFunc(h.proxyMedia)))
	mux.Handle("GET /v1/song-picks", auth(http.HandlerFunc(h.proxyMedia)))
	mux.Handle("GET /v1/song-picks/{songPickId}", auth(http.HandlerFunc(h.proxyMedia)))

	return mux
}
