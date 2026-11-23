package handlers

import "net/http"

// Health GET /health
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.Svc.Health())
}
