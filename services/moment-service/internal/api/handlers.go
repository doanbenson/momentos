package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/doanbenson/momentos/services/moment-service/internal/business"
	"github.com/doanbenson/momentos/services/moment-service/internal/database"
)

type Handler struct {
	Svc *business.Service
}

func (h *Handler) createMoment(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "read body")
		return
	}
	var body struct {
		AssetIDs   []uuid.UUID `json:"assetIds"`
		SongPickID string      `json:"songPickId"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	dto, err := h.Svc.CreateMoment(r.Context(), business.CreateMomentInput{
		Principal:      principalFrom(r.Context()),
		AssetIDs:       body.AssetIDs,
		SongPickID:     body.SongPickID,
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
		RawBody:        raw,
	})
	if err != nil {
		mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto)
}

func (h *Handler) getMoment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("momentId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid momentId")
		return
	}
	dto, err := h.Svc.GetMoment(r.Context(), principalFrom(r.Context()), id)
	if err != nil {
		mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) getCut(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("momentId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid momentId")
		return
	}
	dto, err := h.Svc.GetCut(r.Context(), principalFrom(r.Context()), id)
	if err != nil {
		mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) patchCut(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("momentId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid momentId")
		return
	}
	var patch business.EditorPatch
	if err := decodeJSON(r, &patch); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	dto, err := h.Svc.ApplyEditorPatch(r.Context(), principalFrom(r.Context()), id, patch)
	if err != nil {
		mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) claimJob(w http.ResponseWriter, r *http.Request) {
	var req business.ClaimRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	job, err := h.Svc.ClaimJob(r.Context(), req.JobType, req.WorkerID)
	if err != nil {
		mapErr(w, err)
		return
	}
	if job == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, jobToJSON(job))
}

func (h *Handler) completeJob(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("jobId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid jobId")
		return
	}
	var req business.CompleteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := h.Svc.CompleteJob(r.Context(), id, req); err != nil {
		mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) heartbeatJob(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("jobId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid jobId")
		return
	}
	var req business.HeartbeatRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := h.Svc.HeartbeatJob(r.Context(), id, req.WorkerID); err != nil {
		mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func jobToJSON(j *database.Job) map[string]any {
	payload := j.Payload
	// Empty RawMessage embeds as zero bytes and corrupts the envelope JSON.
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	out := map[string]any{
		"id":        j.ID,
		"momentId":  j.MomentID,
		"jobType":   j.JobType,
		"status":    j.Status,
		"attempt":   j.Attempt,
		"payload":   json.RawMessage(payload),
		"createdAt": j.CreatedAt,
		"updatedAt": j.UpdatedAt,
	}
	if j.CutVersion != nil {
		out["cutVersion"] = *j.CutVersion
	}
	if j.ClaimedBy != nil {
		out["claimedBy"] = *j.ClaimedBy
	}
	if j.LeaseUntil != nil {
		out["leaseUntil"] = *j.LeaseUntil
	}
	return out
}

func mapErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, business.ErrUnauthorized):
		writeErr(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, business.ErrForbidden):
		writeErr(w, http.StatusForbidden, err.Error())
	case errors.Is(err, business.ErrNotFound):
		writeErr(w, http.StatusNotFound, err.Error())
	case errors.Is(err, business.ErrConflict), errors.Is(err, business.ErrIdempotencyReuse):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, business.ErrInvalid):
		writeErr(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, business.ErrUpstream):
		writeErr(w, http.StatusBadGateway, err.Error())
	default:
		writeErr(w, http.StatusInternalServerError, err.Error())
	}
}
