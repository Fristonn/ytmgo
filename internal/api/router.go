package api

import (
	"net/http"

	"github.com/Fristonn/ytmgo/internal/api/handlers"
	"github.com/Fristonn/ytmgo/internal/config"
	"github.com/Fristonn/ytmgo/internal/service"
)

// NewRouter builds the HTTP mux with middleware.
func NewRouter(cfg config.Config, svc *service.Service) http.Handler {
	h := &handlers.Handlers{Svc: svc}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /api/v1/search", h.Search)
	mux.HandleFunc("GET /api/v1/suggestions", h.Suggestions)
	mux.HandleFunc("GET /api/v1/tracks/{id}", h.Track)
	mux.HandleFunc("GET /api/v1/artists/{id}", h.Artist)
	mux.HandleFunc("GET /api/v1/artists/{id}/albums", h.ArtistAlbums)
	mux.HandleFunc("GET /api/v1/artists/{id}/songs", h.ArtistSongs)
	mux.HandleFunc("GET /api/v1/albums/{id}", h.Album)
	mux.HandleFunc("GET /api/v1/playlists/{id}", h.Playlist)
	mux.HandleFunc("GET /api/v1/related/{id}", h.Related)
	mux.HandleFunc("GET /api/v1/charts", h.Charts)
	mux.HandleFunc("GET /api/v1/moods", h.Moods)
	mux.HandleFunc("GET /api/v1/streams/{id}", h.Streams)

	var handler http.Handler = mux
	handler = withTimeout(cfg.RequestTimeout, handler)
	handler = withAPIKey(cfg, handler)
	handler = withRateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst, handler)
	handler = withCORS(cfg, handler)
	handler = withLogging(handler)
	handler = withRecovery(handler)
	return handler
}
