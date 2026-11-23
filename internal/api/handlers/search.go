package handlers

import "net/http"

// Search GET /api/v1/search
func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		q = r.URL.Query().Get("query")
	}
	filter := r.URL.Query().Get("filter")
	cont := r.URL.Query().Get("continuation")
	if len(q) > 200 {
		writeError(w, http.StatusBadRequest, "bad_request", "query too long")
		return
	}
	res, err := h.Svc.Search(r.Context(), q, filter, cont)
	if err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// Suggestions GET /api/v1/suggestions
func (h *Handlers) Suggestions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if len(q) > 200 {
		writeError(w, http.StatusBadRequest, "bad_request", "q too long")
		return
	}
	res, err := h.Svc.Suggestions(r.Context(), q)
	if err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
