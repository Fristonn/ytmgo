package handlers

import "net/http"

// Album GET /api/v1/albums/{id}
func (h *Handlers) Album(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	res, err := h.Svc.Album(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// Playlist GET /api/v1/playlists/{id}
func (h *Handlers) Playlist(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id != "" && !validID(id) {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	res, err := h.Svc.Playlist(r.Context(), id, r.URL.Query().Get("continuation"))
	if err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// Charts GET /api/v1/charts
func (h *Handlers) Charts(w http.ResponseWriter, r *http.Request) {
	res, err := h.Svc.Charts(r.Context())
	if err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sections": res})
}

// Moods GET /api/v1/moods
func (h *Handlers) Moods(w http.ResponseWriter, r *http.Request) {
	res, err := h.Svc.Moods(r.Context())
	if err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"moods": res})
}
