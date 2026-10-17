package api

import "net/http"

func (h *handlers) getSongPick(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "id required")
		return
	}
	sign := queryBool(r, "sign", false)
	v, err := h.svc.GetSongPick(r.Context(), id, sign)
	if err != nil {
		mapBizError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"songPick": toSongPickDTO(v)})
}

func (h *handlers) listSongPicks(w http.ResponseWriter, r *http.Request) {
	sign := queryBool(r, "sign", false)
	list, err := h.svc.ListSongPicks(r.Context(), sign)
	if err != nil {
		mapBizError(w, err)
		return
	}
	out := make([]songPickDTO, 0, len(list))
	for i := range list {
		out = append(out, toSongPickDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"songPicks": out})
}
