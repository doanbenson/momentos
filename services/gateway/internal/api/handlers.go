package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/doanbenson/momentos/services/gateway/internal/business"
)

type Handler struct {
	BFF *business.BFF
}

func (h *Handler) createMoment(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "read body")
		return
	}
	status, raw, err := h.BFF.ProxyCreateMoment(r.Context(), Principal(r.Context()), body, r.Header.Get("Idempotency-Key"))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeUpstream(w, status, raw)
}

func (h *Handler) listMoments(w http.ResponseWriter, r *http.Request) {
	status, raw, err := h.BFF.ProxyListMoments(r.Context(), Principal(r.Context()))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeUpstream(w, status, raw)
}

func (h *Handler) getMoment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("momentId")
	agg, err := h.BFF.GetMomentAggregate(r.Context(), Principal(r.Context()), id)
	if err != nil {
		var he *business.HTTPError
		if errors.As(err, &he) {
			writeUpstream(w, he.Status, he.Body)
			return
		}
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, agg)
}

func (h *Handler) getCut(w http.ResponseWriter, r *http.Request) {
	status, raw, err := h.BFF.ProxyGetCut(r.Context(), Principal(r.Context()), r.PathValue("momentId"))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeUpstream(w, status, raw)
}

func (h *Handler) patchCut(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "read body")
		return
	}
	status, raw, err := h.BFF.ProxyPatchCut(r.Context(), Principal(r.Context()), r.PathValue("momentId"), body)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeUpstream(w, status, raw)
}

func (h *Handler) proxyMedia(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "read body")
		return
	}
	var payload []byte
	if len(body) > 0 {
		payload = body
	}
	path := r.URL.Path
	status, raw, err := h.BFF.ProxyMedia(r.Context(), r.Method, path, Principal(r.Context()), payload, r.URL.RawQuery)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeUpstream(w, status, raw)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeUpstream(w http.ResponseWriter, status int, raw []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if len(raw) > 0 {
		_, _ = w.Write(raw)
	}
}
