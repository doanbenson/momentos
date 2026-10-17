package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/doanbenson/momentos/services/render-service/internal/business"
	"github.com/doanbenson/momentos/services/render-service/internal/database"
	"github.com/doanbenson/momentos/services/render-service/internal/ffmpeg"
)

// Deps wires HTTP handlers to business + ffmpeg readiness.
type Deps struct {
	Service *business.Service
	Runner  *ffmpeg.Runner
}

func NewMux(d Deps) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		if d.Runner == nil || !d.Runner.Available() {
			http.Error(w, "ffmpeg unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	mux.HandleFunc("GET /v1/internal/render-jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		job, ok := d.Service.Store.Get(id)
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, job)
	})

	// Local/dev enqueue until moment-service broker HTTP is ready.
	mux.HandleFunc("POST /v1/internal/render-jobs", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
		if err != nil {
			http.Error(w, "read body", http.StatusBadRequest)
			return
		}
		var req enqueueRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		cutJSON := req.Cut
		if len(cutJSON) == 0 {
			cutJSON = req.CutJSON
		}
		job := &database.Job{
			ID:           req.ID,
			MomentID:     req.MomentID,
			CutID:        req.CutID,
			CutVersion:   req.CutVersion,
			SongPickID:   req.SongPickID,
			EditAudioURL: req.EditAudioURL,
			Assets:       req.Assets,
			CutJSON:      cutJSON,
		}
		created, err := d.Service.Enqueue(job)
		if err != nil {
			if business.IsTerminal(err) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		writeJSON(w, http.StatusAccepted, created)
	})

	return mux
}

type enqueueRequest struct {
	ID           string               `json:"id"`
	MomentID     string               `json:"momentId"`
	CutID        string               `json:"cutId"`
	CutVersion   int                  `json:"cutVersion"`
	SongPickID   string               `json:"songPickId"`
	EditAudioURL string               `json:"editAudioUrl"`
	Assets       []database.AssetRef  `json:"assets"`
	Cut     json.RawMessage `json:"cut"`     // Cut object with clips[]
	CutJSON json.RawMessage `json:"cutJson"` // alias
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
