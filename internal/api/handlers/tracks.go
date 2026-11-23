package handlers

import "net/http"

// Track GET /api/v1/tracks/{id}
func (h *Handlers) Track(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	res, err := h.Svc.Track(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// Related GET /api/v1/related/{id}
func (h *Handlers) Related(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	res, err := h.Svc.Related(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// Streams GET /api/v1/streams/{id}
func (h *Handlers) Streams(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	res, err := h.Svc.Streams(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
