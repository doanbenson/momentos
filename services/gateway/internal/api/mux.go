package api

import "net/http"

// NewMux wires HTTP routes. Business orchestration lives in internal/business.
func NewMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	// TODO: JWT validation (Supabase) + BFF routes to moment/media/render
	return mux
}
