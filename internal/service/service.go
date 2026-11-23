package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"github.com/Fristonn/ytmgo/internal/cache"
	"github.com/Fristonn/ytmgo/internal/cipher"
	"github.com/Fristonn/ytmgo/internal/config"
	"github.com/Fristonn/ytmgo/internal/httpclient"
	"github.com/Fristonn/ytmgo/internal/innertube"
	"github.com/Fristonn/ytmgo/internal/innertube/browse"
	"github.com/Fristonn/ytmgo/internal/innertube/parser"
	"github.com/Fristonn/ytmgo/internal/merge"
	"github.com/Fristonn/ytmgo/internal/models"
	"github.com/Fristonn/ytmgo/internal/playback"
	"github.com/Fristonn/ytmgo/internal/potoken"
)

// Service is the application facade used by HTTP handlers.
type Service struct {
	cfg      config.Config
	yt       *innertube.Client
	cache    *cache.TTL
	resolver *playback.Resolver
	started  time.Time
	po       potoken.Provider
}

// New constructs the service graph.
func New(cfg config.Config) *Service {
	hc := httpclient.New(cfg.UpstreamTimeout)
	po := potoken.NewFromConfig(potoken.Config{
		Provider:       cfg.PoTokenProvider,
		PlayerToken:    cfg.PoTokenPlayer,
		SessionToken:   cfg.PoTokenSession,
		ManualTTL:      cfg.PoTokenManualTTL,
		ServiceURL:     cfg.PoTokenServiceURL,
		ServicePath:    cfg.PoTokenServicePath,
		ServiceGetPot:  cfg.PoTokenServiceGetPot,
		ServiceAuth:    cfg.PoTokenServiceAuth,
		ServiceAPIKey:  cfg.PoTokenServiceAPIKey,
		ServiceTimeout: cfg.PoTokenServiceTimeout,
		CacheTTL:       cfg.PoTokenCacheTTL,
		CacheMax:       cfg.PoTokenCacheMax,
		HTTPClient:     hc,
	})
	yt := innertube.New(innertube.Options{
		HTTPClient:         hc,
		HL:                 cfg.HL,
		GL:                 cfg.GL,
		MaxRetries:         cfg.MaxRetries,
		MaxConcurrent:      32,
		PoTokens:           po,
		PoTokenRequiredWeb: cfg.PoTokenRequiredForWeb,
	})
	cm := cipher.New(hc, cfg.CipherServiceURL)
	return &Service{
		cfg:      cfg,
		yt:       yt,
		cache:    cache.New(cfg.CacheTTL, cfg.CacheMaxEntries),
		resolver: playback.NewResolver(yt, cm),
		started:  time.Now(),
		po:       po,
	}
}

// Health returns service health (never includes token secrets).
func (s *Service) Health() models.HealthResponse {
	h := models.HealthResponse{
		Status:  "ok",
		Version: "1.0.0",
		Clients: s.yt.Tracker().Status(),
	}
	st := potoken.Snapshot(s.po)
	if h.Clients == nil {
		h.Clients = map[string]string{}
	}
	if st.Configured {
		h.Clients["po_token"] = st.Provider + ":ready"
	} else {
		h.Clients["po_token"] = "disabled"
	}
	return h
}

// Search performs a YouTube Music search.
func (s *Service) Search(ctx context.Context, query, filter, continuation string) (*models.SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" && continuation == "" {
		return nil, fmt.Errorf("query is required")
	}
	key := "search:" + filter + ":" + query + ":" + continuation
	v, err, _ := s.cache.Do(key, func() (any, error) {
		return s.enrichSearch(ctx, query, filter, continuation)
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.SearchResult), nil
}

// Suggestions returns search suggestions.
func (s *Service) Suggestions(ctx context.Context, q string) (map[string]any, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, fmt.Errorf("q is required")
	}
	key := "suggest:" + q
	v, err, _ := s.cache.Do(key, func() (any, error) {
		return s.enrichSuggestions(ctx, q)
	})
	if err != nil {
		return nil, err
	}
	return v.(map[string]any), nil
}

// Track returns track metadata via player + optional next.
func (s *Service) Track(ctx context.Context, id string) (*models.Track, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	key := "track:" + id
	v, err, _ := s.cache.Do(key, func() (any, error) {
		return s.enrichTrack(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.Track), nil
}

// Artist returns an artist profile.
func (s *Service) Artist(ctx context.Context, id string) (*models.Artist, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	key := "artist:" + id
	v, err, _ := s.cache.Do(key, func() (any, error) {
		return s.enrichArtist(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.Artist), nil
}

// ArtistAlbums returns albums / singles for an artist section browse.
func (s *Service) ArtistAlbums(ctx context.Context, id, params, continuation string) (map[string]any, error) {
	id = strings.TrimSpace(id)
	key := "artistAlbums:" + id + ":" + params + ":" + continuation
	v, err, _ := s.cache.Do(key, func() (any, error) {
		var raw []byte
		var err error
		if continuation != "" {
			raw, err = s.yt.Browse(ctx, "", "", continuation)
		} else {
			// Fetch artist page to discover albums section endpoint when params empty.
			if params == "" {
				artist, err2 := s.Artist(ctx, id)
				if err2 != nil {
					return nil, err2
				}
				for _, sec := range artist.Sections {
					title := strings.ToLower(sec.Title)
					if (strings.Contains(title, "album") || strings.Contains(title, "single")) && sec.BrowseID != "" {
						raw, err = s.yt.Browse(ctx, sec.BrowseID, sec.Params, "")
						break
					}
				}
				if raw == nil {
					return map[string]any{
						"albums":       artist.Albums,
						"singles":      artist.Singles,
						"continuation": "",
					}, nil
				}
			} else {
				raw, err = s.yt.Browse(ctx, id, params, "")
			}
		}
		if err != nil {
			return nil, err
		}
		title, albums, _, cont, err := browse.ParseArtistItems(raw)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"title":        title,
			"albums":       albums,
			"continuation": cont,
		}, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(map[string]any), nil
}

// ArtistSongs returns top / all songs for an artist.
func (s *Service) ArtistSongs(ctx context.Context, id, params, continuation string) (map[string]any, error) {
	id = strings.TrimSpace(id)
	key := "artistSongs:" + id + ":" + params + ":" + continuation
	v, err, _ := s.cache.Do(key, func() (any, error) {
		if continuation != "" {
			raw, err := s.yt.Browse(ctx, "", "", continuation)
			if err != nil {
				return nil, err
			}
			_, _, tracks, cont, err := browse.ParseArtistItems(raw)
			if err != nil {
				return nil, err
			}
			return map[string]any{"tracks": tracks, "continuation": cont}, nil
		}
		if params != "" {
			raw, err := s.yt.Browse(ctx, id, params, "")
			if err != nil {
				return nil, err
			}
			title, _, tracks, cont, err := browse.ParseArtistItems(raw)
			if err != nil {
				return nil, err
			}
			return map[string]any{"title": title, "tracks": tracks, "continuation": cont}, nil
		}
		artist, err := s.Artist(ctx, id)
		if err != nil {
			return nil, err
		}
		for _, sec := range artist.Sections {
			if strings.Contains(strings.ToLower(sec.Title), "song") && sec.BrowseID != "" {
				raw, err := s.yt.Browse(ctx, sec.BrowseID, sec.Params, "")
				if err != nil {
					break
				}
				title, _, tracks, cont, err := browse.ParseArtistItems(raw)
				if err == nil && len(tracks) > 0 {
					return map[string]any{"title": title, "tracks": tracks, "continuation": cont}, nil
				}
			}
		}
		return map[string]any{"tracks": artist.TopSongs, "continuation": ""}, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(map[string]any), nil
}

// Album returns album metadata.
func (s *Service) Album(ctx context.Context, id string) (*models.Album, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	key := "album:" + id
	v, err, _ := s.cache.Do(key, func() (any, error) {
		return s.enrichAlbum(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.Album), nil
}

// Playlist returns a playlist with optional continuation.
func (s *Service) Playlist(ctx context.Context, id, continuation string) (*models.Playlist, error) {
	id = strings.TrimSpace(id)
	if id == "" && continuation == "" {
		return nil, fmt.Errorf("id is required")
	}
	key := "playlist:" + id + ":" + continuation
	v, err, _ := s.cache.Do(key, func() (any, error) {
		return s.enrichPlaylist(ctx, id, continuation)
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.Playlist), nil
}

// Related returns related tracks for a video/track id.
func (s *Service) Related(ctx context.Context, id string) (*models.Section, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	key := "related:" + id
	v, err, _ := s.cache.Do(key, func() (any, error) {
		return s.enrichRelated(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.Section), nil
}

func relatedBrowseID(nextRaw []byte) string {
	var id string
	gjson.ParseBytes(nextRaw).Get("contents.singleColumnMusicWatchNextResultsRenderer.tabbedRenderer.watchNextTabbedResultsRenderer.tabs").ForEach(func(_, tab gjson.Result) bool {
		title := strings.ToLower(tab.Get("tabRenderer.title").String())
		if title == "related" {
			id = tab.Get("tabRenderer.endpoint.browseEndpoint.browseId").String()
			return false
		}
		return true
	})
	return id
}

// Charts returns music charts.
func (s *Service) Charts(ctx context.Context) ([]models.ChartSection, error) {
	v, err, _ := s.cache.Do("charts", func() (any, error) {
		raw, err := s.yt.Browse(ctx, "FEmusic_charts", "", "")
		if err != nil {
			return nil, err
		}
		return browse.ParseCharts(raw)
	})
	if err != nil {
		return nil, err
	}
	return v.([]models.ChartSection), nil
}

// Moods returns moods and genres.
func (s *Service) Moods(ctx context.Context) ([]models.MoodGenre, error) {
	v, err, _ := s.cache.Do("moods", func() (any, error) {
		raw, err := s.yt.Browse(ctx, "FEmusic_moods_and_genres", "", "")
		if err != nil {
			return nil, err
		}
		return browse.ParseMoods(raw)
	})
	if err != nil {
		return nil, err
	}
	return v.([]models.MoodGenre), nil
}

// Streams resolves playable audio formats.
func (s *Service) Streams(ctx context.Context, id string) (*models.StreamResponse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	key := "streams:" + id
	v, err, _ := s.cache.Do(key, func() (any, error) {
		resp, err := s.resolver.Resolve(ctx, id)
		if err != nil {
			return nil, err
		}
		// Player clients return 16:9 video thumbs; pull square Music art when available.
		if track, terr := s.enrichTrack(ctx, id); terr == nil && track != nil {
			resp.Artwork = parser.PreferMusicArtwork(merge.Artwork(track.Artwork, resp.Artwork))
			if resp.Title == "" {
				resp.Title = track.Title
			}
			if resp.Author == "" && len(track.Artists) > 0 {
				resp.Author = track.Artists[0].Name
			}
			if resp.Duration == 0 {
				resp.Duration = track.Duration
			}
		} else {
			resp.Artwork = parser.PreferMusicArtwork(resp.Artwork)
		}
		if !s.cfg.ExposeStreamURLs {
			for i := range resp.AudioFormats {
				resp.AudioFormats[i].URL = ""
			}
			for i := range resp.Formats {
				resp.Formats[i].URL = ""
			}
		}
		return resp, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.StreamResponse), nil
}

// CacheStats exposes cache size for health/debug.
func (s *Service) CacheStats() int {
	return s.cache.Len()
}
