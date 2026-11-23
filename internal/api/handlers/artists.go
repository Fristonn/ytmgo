package handlers

import "net/http"

// Artist GET /api/v1/artists/{id}
func (h *Handlers) Artist(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	res, err := h.Svc.Artist(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ArtistAlbums GET /api/v1/artists/{id}/albums
func (h *Handlers) ArtistAlbums(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	res, err := h.Svc.ArtistAlbums(r.Context(), id, r.URL.Query().Get("params"), r.URL.Query().Get("continuation"))
	if err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ArtistSongs GET /api/v1/artists/{id}/songs
func (h *Handlers) ArtistSongs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	res, err := h.Svc.ArtistSongs(r.Context(), id, r.URL.Query().Get("params"), r.URL.Query().Get("continuation"))
	if err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
