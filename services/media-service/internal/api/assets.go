package api

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/doanbenson/momentos/services/media-service/internal/business"
	"github.com/doanbenson/momentos/services/media-service/internal/database"
	"github.com/google/uuid"
)

type createAssetBody struct {
	Type                string `json:"type"`
	MimeType            string `json:"mimeType"`
	ByteSize            int64  `json:"byteSize"`
	DurationMs          *int   `json:"durationMs"`
	Width               *int   `json:"width"`
	Height              *int   `json:"height"`
	WantThumbnailUpload *bool  `json:"wantThumbnailUpload"`
}

func (h *handlers) createAsset(w http.ResponseWriter, r *http.Request) {
	var body createAssetBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	wantThumb := false
	if body.WantThumbnailUpload != nil {
		wantThumb = *body.WantThumbnailUpload
	}
	res, err := h.svc.CreateAsset(r.Context(), business.CreateAssetInput{
		UserID:              principal(r),
		Type:                database.AssetType(body.Type),
		MimeType:            body.MimeType,
		ByteSize:            body.ByteSize,
		DurationMs:          body.DurationMs,
		Width:               body.Width,
		Height:              body.Height,
		WantThumbnailUpload: wantThumb,
	})
	if err != nil {
		mapBizError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"asset":              toAssetDTOFromAsset(res.Asset),
		"uploadUrl":          res.UploadURL,
		"uploadHeaders":      res.UploadHeaders,
		"thumbnailUploadUrl": nullIfEmpty(res.ThumbnailUploadURL),
		"thumbnailHeaders":   res.ThumbnailHeaders,
	})
}

type completeAssetBody struct {
	Checksum          *string `json:"checksum"`
	ByteSize          *int64  `json:"byteSize"`
	DurationMs        *int    `json:"durationMs"`
	Width             *int    `json:"width"`
	Height            *int    `json:"height"`
	ThumbnailUploaded *bool   `json:"thumbnailUploaded"`
}

func (h *handlers) completeAsset(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("assetId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid assetId")
		return
	}
	var body completeAssetBody
	if err := decodeJSONOptional(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	asset, err := h.svc.CompleteAsset(r.Context(), business.CompleteAssetInput{
		UserID:            principal(r),
		AssetID:           id,
		Checksum:          body.Checksum,
		ByteSize:          body.ByteSize,
		DurationMs:        body.DurationMs,
		Width:             body.Width,
		Height:            body.Height,
		ThumbnailUploaded: body.ThumbnailUploaded,
	})
	if err != nil {
		mapBizError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"asset": toAssetDTOFromAsset(asset),
	})
}

func (h *handlers) getAsset(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("assetId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid assetId")
		return
	}
	sign := queryBool(r, "sign", true)
	v, err := h.svc.GetAsset(r.Context(), principal(r), id, sign)
	if err != nil {
		mapBizError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"asset": toAssetDTO(v)})
}

func (h *handlers) listAssets(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimSpace(r.URL.Query().Get("ids"))
	if raw == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "ids query required")
		return
	}
	parts := strings.Split(raw, ",")
	ids := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := uuid.Parse(p)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid id in ids")
			return
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "ids query required")
		return
	}
	sign := queryBool(r, "sign", false)
	views, err := h.svc.ListAssets(r.Context(), principal(r), ids, sign)
	if err != nil {
		mapBizError(w, err)
		return
	}
	assets := make([]assetDTO, 0, len(views))
	for i := range views {
		assets = append(assets, toAssetDTO(&views[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"assets": assets})
}

func (h *handlers) deleteAsset(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("assetId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid assetId")
		return
	}
	if err := h.svc.DeleteAsset(r.Context(), principal(r), id); err != nil {
		mapBizError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) signDownload(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AssetIDs   []string `json:"assetIds"`
		SongPickID string   `json:"songPickId"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	ids := make([]uuid.UUID, 0, len(body.AssetIDs))
	for _, s := range body.AssetIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid assetId")
			return
		}
		ids = append(ids, id)
	}
	res, err := h.svc.SignDownload(r.Context(), business.SignDownloadInput{
		AssetIDs:   ids,
		SongPickID: body.SongPickID,
	})
	if err != nil {
		mapBizError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *handlers) signObject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StorageKey string `json:"storageKey"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	url, err := h.svc.SignObject(r.Context(), body.StorageKey)
	if err != nil {
		mapBizError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"downloadUrl": url})
}

func mapBizError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, business.ErrInvalidInput),
		errors.Is(err, business.ErrSizeExceeded),
		errors.Is(err, business.ErrMimeMismatch),
		errors.Is(err, business.ErrInvalidStatus),
		errors.Is(err, business.ErrObjectMissing):
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
	case errors.Is(err, business.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, business.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, business.ErrRateLimited):
		writeError(w, http.StatusTooManyRequests, "rate_limited", "too many asset creates")
	case errors.Is(err, business.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", err.Error())
	default:
		log.Printf("media-service handler error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
	}
}

func queryBool(r *http.Request, key string, def bool) bool {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	default:
		return def
	}
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
