package api

import (
	"net/http"

	"github.com/doanbenson/momentos/services/media-service/internal/business"
)

type handlers struct {
	svc  *business.Service
	auth *Auth
}

type Deps struct {
	Service *business.Service
	Auth    *Auth
}

func NewMux(d Deps) http.Handler {
	h := &handlers{svc: d.Service, auth: d.Auth}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Principal-scoped (gateway forwards internal token + X-Momentos-Principal).
	mux.HandleFunc("POST /v1/assets", h.auth.requirePrincipal(h.createAsset))
	mux.HandleFunc("POST /v1/assets/{assetId}/complete", h.auth.requirePrincipal(h.completeAsset))
	mux.HandleFunc("GET /v1/assets/{assetId}", h.auth.requirePrincipal(h.getAsset))
	mux.HandleFunc("GET /v1/assets", h.auth.requirePrincipal(h.listAssets))
	mux.HandleFunc("DELETE /v1/assets/{assetId}", h.auth.requirePrincipal(h.deleteAsset))
	mux.HandleFunc("GET /v1/song-picks/{id}", h.auth.requirePrincipal(h.getSongPick))
	mux.HandleFunc("GET /v1/song-picks", h.auth.requirePrincipal(h.listSongPicks))

	// Internal / worker helpers (token only; ml + render + moment-service).
	mux.HandleFunc("POST /v1/internal/assets/sign-download", h.auth.requireInternal(h.signDownload))
	mux.HandleFunc("POST /v1/internal/objects/sign-download", h.auth.requireInternal(h.signObject))

	return mux
}
