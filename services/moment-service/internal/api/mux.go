package api

import (
	"net/http"

	"github.com/doanbenson/momentos/services/moment-service/internal/business"
)

type Deps struct {
	Svc           *business.Service
	InternalToken string
}

func NewMux(d Deps) http.Handler {
	h := &Handler{Svc: d.Svc}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	authed := chain(internalAuth(d.InternalToken), requirePrincipal)
	mux.Handle("GET /v1/moments", authed(http.HandlerFunc(h.listMoments)))
	mux.Handle("POST /v1/moments", authed(http.HandlerFunc(h.createMoment)))
	mux.Handle("GET /v1/moments/{momentId}", authed(http.HandlerFunc(h.getMoment)))
	mux.Handle("GET /v1/moments/{momentId}/cut", authed(http.HandlerFunc(h.getCut)))
	mux.Handle("PATCH /v1/moments/{momentId}/cut", authed(http.HandlerFunc(h.patchCut)))

	internalOnly := internalAuth(d.InternalToken)
	mux.Handle("POST /v1/internal/jobs/claim", internalOnly(http.HandlerFunc(h.claimJob)))
	mux.Handle("POST /v1/internal/jobs/{jobId}/complete", internalOnly(http.HandlerFunc(h.completeJob)))
	mux.Handle("POST /v1/internal/jobs/{jobId}/heartbeat", internalOnly(http.HandlerFunc(h.heartbeatJob)))

	return mux
}
