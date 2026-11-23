package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/Fristonn/ytmgo/internal/models"
	"github.com/Fristonn/ytmgo/internal/service"
)

var idPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

// Handlers groups HTTP handlers backed by the application service.
type Handlers struct {
	Svc *service.Service
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, models.APIError{Error: code, Message: msg, Code: status})
}

func mapError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	msg := err.Error()
	switch {
	case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusGatewayTimeout, "timeout", "request timed out")
	case strings.Contains(msg, "required"):
		writeError(w, http.StatusBadRequest, "bad_request", msg)
	case strings.Contains(msg, "not found") || strings.Contains(msg, "unavailable"):
		writeError(w, http.StatusNotFound, "not_found", msg)
	case strings.Contains(msg, "429"):
		writeError(w, http.StatusTooManyRequests, "upstream_rate_limited", msg)
	default:
		writeError(w, http.StatusBadGateway, "upstream_error", msg)
	}
}

func validID(id string) bool {
	return idPattern.MatchString(id)
}
