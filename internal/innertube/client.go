package innertube

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Fristonn/ytmgo/internal/innertube/clients"
	"github.com/Fristonn/ytmgo/internal/potoken"
)

const maxResponseBytes = 8 << 20 // 8 MiB

// Client talks to YouTube Innertube endpoints.
type Client struct {
	http        *http.Client
	profiles    map[string]clients.Profile
	tracker     *clients.Tracker
	visitorData string
	maxRetries  int
	mu          sync.RWMutex
	sem         chan struct{}
	poTokens       potoken.Provider
	poRequiredWeb  bool
	lastSession    string // last stream pot from provider (never logged)
}

// Options configures the Innertube client.
type Options struct {
	HTTPClient    *http.Client
	HL            string
	GL            string
	MaxRetries    int
	MaxConcurrent int
	VisitorData        string
	PoTokens           potoken.Provider
	PoTokenRequiredWeb bool
}

// New creates an Innertube client.
func New(opts Options) *Client {
	if opts.HTTPClient == nil {
		panic("innertube: HTTPClient required")
	}
	if opts.HL == "" {
		opts.HL = "en"
	}
	if opts.GL == "" {
		opts.GL = "US"
	}
	if opts.MaxRetries < 0 {
		opts.MaxRetries = 0
	}
	if opts.MaxConcurrent <= 0 {
		opts.MaxConcurrent = 32
	}
	return &Client{
		http:        opts.HTTPClient,
		profiles:    clients.Profiles(opts.HL, opts.GL),
		tracker:     clients.NewTracker(),
		visitorData: opts.VisitorData,
		maxRetries:  opts.MaxRetries,
		sem:           make(chan struct{}, opts.MaxConcurrent),
		poTokens:      opts.PoTokens,
		poRequiredWeb: opts.PoTokenRequiredWeb,
	}
}

// Profile returns a named client profile.
func (c *Client) Profile(name string) (clients.Profile, bool) {
	p, ok := c.profiles[name]
	return p, ok
}

// Tracker exposes client health.
func (c *Client) Tracker() *clients.Tracker {
	return c.tracker
}

// SetVisitorData updates the visitor data token.
func (c *Client) SetVisitorData(vd string) {
	c.mu.Lock()
	c.visitorData = vd
	c.mu.Unlock()
}

func (c *Client) getVisitorData() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.visitorData
}

// SearchFilter known Music search param values.
var SearchFilter = map[string]string{
	"songs":               "EgWKAQIIAWoKEAkQBRAKEAMQBA%3D%3D",
	"videos":              "EgWKAQIQAWoKEAkQChAFEAMQBA%3D%3D",
	"albums":              "EgWKAQIYAWoKEAkQChAFEAMQBA%3D%3D",
	"artists":             "EgWKAQIgAWoKEAkQChAFEAMQBA%3D%3D",
	"playlists":           "EgeKAQQoADgBagwQDhAKEAMQBRAJEAQ%3D",
	"community_playlists": "EgeKAQQoAEABagoQAxAEEAoQCRAF",
}

// Search posts to youtubei/v1/search using WEB_REMIX.
func (c *Client) Search(ctx context.Context, query, filter, continuation string) ([]byte, error) {
	return c.SearchAs(ctx, "WEB_REMIX", query, filter, continuation)
}

// SearchAs posts search with a named client profile.
func (c *Client) SearchAs(ctx context.Context, profileName, query, filter, continuation string) ([]byte, error) {
	p, err := c.requireProfile(profileName)
	if err != nil {
		return nil, err
	}
	params := ""
	if filter != "" {
		if v, ok := SearchFilter[strings.ToLower(filter)]; ok {
			params = v
		} else {
			params = filter
		}
	}
	body := SearchBody{
		Context: buildContext(p, c.getVisitorData(), ""),
		Query:   query,
		Params:  params,
	}
	q := url.Values{}
	if continuation != "" {
		q.Set("continuation", continuation)
		q.Set("ctoken", continuation)
	}
	return c.post(ctx, p, "search", body, q)
}

// Browse posts to youtubei/v1/browse using WEB_REMIX.
func (c *Client) Browse(ctx context.Context, browseID, params, continuation string) ([]byte, error) {
	return c.BrowseAs(ctx, "WEB_REMIX", browseID, params, continuation)
}

// BrowseAs posts browse with a named client profile.
func (c *Client) BrowseAs(ctx context.Context, profileName, browseID, params, continuation string) ([]byte, error) {
	p, err := c.requireProfile(profileName)
	if err != nil {
		return nil, err
	}
	body := BrowseBody{
		Context:  buildContext(p, c.getVisitorData(), ""),
		BrowseID: browseID,
		Params:   params,
	}
	q := url.Values{}
	if continuation != "" {
		q.Set("continuation", continuation)
		q.Set("ctoken", continuation)
		q.Set("type", "next")
	}
	return c.post(ctx, p, "browse", body, q)
}

// Suggestions posts to music/get_search_suggestions using WEB_REMIX.
func (c *Client) Suggestions(ctx context.Context, input string) ([]byte, error) {
	return c.SuggestionsAs(ctx, "WEB_REMIX", input)
}

// SuggestionsAs posts suggestions with a named client profile.
func (c *Client) SuggestionsAs(ctx context.Context, profileName, input string) ([]byte, error) {
	p, err := c.requireProfile(profileName)
	if err != nil {
		return nil, err
	}
	endpoint := "music/get_search_suggestions"
	if p.ClientName != "WEB_REMIX" && p.ClientName != "ANDROID_MUSIC" {
		endpoint = "get_search_suggestions"
	}
	body := SuggestionsBody{
		Context: buildContext(p, c.getVisitorData(), ""),
		Input:   input,
	}
	return c.post(ctx, p, endpoint, body, nil)
}

// Next posts to youtubei/v1/next using WEB_REMIX.
func (c *Client) Next(ctx context.Context, videoID, playlistID, continuation string) ([]byte, error) {
	return c.NextAs(ctx, "WEB_REMIX", videoID, playlistID, continuation)
}

// NextAs posts next with a named client profile.
func (c *Client) NextAs(ctx context.Context, profileName, videoID, playlistID, continuation string) ([]byte, error) {
	p, err := c.requireProfile(profileName)
	if err != nil {
		return nil, err
	}
	body := NextBody{
		Context:      buildContext(p, c.getVisitorData(), ""),
		VideoID:      videoID,
		PlaylistID:   playlistID,
		Continuation: continuation,
	}
	return c.post(ctx, p, "next", body, nil)
}

func (c *Client) requireProfile(name string) (clients.Profile, error) {
	p, ok := c.profiles[name]
	if !ok {
		return clients.Profile{}, fmt.Errorf("unknown client profile: %s", name)
	}
	if !c.tracker.Available(name) {
		return clients.Profile{}, fmt.Errorf("client %s in cooldown", name)
	}
	return p, nil
}

// Player posts to youtubei/v1/player using a specific client profile.
func (c *Client) Player(ctx context.Context, profileName, videoID, playlistID string) ([]byte, string, error) {
	p, ok := c.profiles[profileName]
	if !ok {
		return nil, "", fmt.Errorf("unknown client profile: %s", profileName)
	}
	if !c.tracker.Available(profileName) {
		return nil, "", fmt.Errorf("client %s in cooldown", profileName)
	}

	embed := ""
	if strings.Contains(p.ClientName, "EMBEDDED") || p.ClientName == "TVHTML5_SIMPLY_EMBEDDED_PLAYER" {
		embed = "https://www.youtube.com/watch?v=" + videoID
	}

	visitor := c.getVisitorData()
	body := PlayerBody{
		Context:        buildContext(p, visitor, embed),
		VideoID:        videoID,
		PlaylistID:     playlistID,
		ContentCheckOK: true,
		RacyCheckOK:    true,
	}

	needsPo := potoken.Configured(c.poTokens) &&
		(p.ClientName == "WEB" || p.ClientName == "WEB_REMIX" || p.ClientName == "MWEB")
	if needsPo {
		if visitor == "" {
			if c.poRequiredWeb {
				return nil, profileName, fmt.Errorf("%w: visitorData missing", potoken.ErrUnavailable)
			}
		} else {
			tok, err := c.poTokens.Tokens(ctx, potoken.Request{
				VideoID:     videoID,
				VisitorData: visitor,
				ClientName:  p.ClientName,
			})
			if err != nil || (tok.Player == "" && tok.Session == "") {
				if c.poRequiredWeb {
					if err == nil {
						err = potoken.ErrUnavailable
					}
					return nil, profileName, fmt.Errorf("po token required for %s: %w", p.ClientName, err)
				}
				// Soft-skip: continue without PO tokens (other clients may still succeed).
			} else {
				playerTok := tok.Player
				if playerTok == "" {
					playerTok = tok.Session
				}
				body.ServiceIntegrityDimensions = map[string]any{"poToken": playerTok}
				c.mu.Lock()
				c.lastSession = tok.Session
				c.mu.Unlock()
			}
		}
	}

	raw, err := c.post(ctx, p, "player", body, nil)
	if err != nil {
		rateLimited := strings.Contains(err.Error(), "429")
		c.tracker.MarkFailure(profileName, rateLimited, err.Error())
		return nil, profileName, err
	}
	c.tracker.MarkSuccess(profileName)
	return raw, profileName, nil
}

// LastStreamPoToken returns the most recent session pot for media URLs.
func (c *Client) LastStreamPoToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastSession
}

// VisitorData returns the current visitor token.
func (c *Client) VisitorData() string {
	return c.getVisitorData()
}

func (c *Client) post(ctx context.Context, p clients.Profile, endpoint string, body any, query url.Values) ([]byte, error) {
	select {
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	payload, err := encodeBody(body)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * 200 * time.Millisecond
			jitter := time.Duration(rand.Intn(100)) * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff + jitter):
			}
		}

		raw, status, err := c.doOnce(ctx, p, endpoint, payload, query)
		if err != nil {
			lastErr = err
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			continue
		}
		if status == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("innertube 429 rate limited")
			c.tracker.MarkFailure(p.Name, true, lastErr.Error())
			continue
		}
		if status >= 500 {
			lastErr = fmt.Errorf("innertube HTTP %d", status)
			continue
		}
		if status >= 400 {
			return nil, fmt.Errorf("innertube HTTP %d: %s", status, truncate(string(raw), 200))
		}
		c.captureVisitorData(raw)
		return raw, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("innertube request failed")
	}
	return nil, lastErr
}

func (c *Client) doOnce(ctx context.Context, p clients.Profile, endpoint string, payload []byte, query url.Values) ([]byte, int, error) {
	u := strings.TrimRight(p.BaseURL, "/") + "/" + strings.TrimLeft(endpoint, "/")
	q := url.Values{}
	if query != nil {
		q = query
	}
	q.Set("prettyPrint", "false")
	if p.APIKey != "" {
		q.Set("key", p.APIKey)
	}
	u = u + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", p.UserAgent)
	req.Header.Set("X-Goog-Api-Format-Version", "1")
	req.Header.Set("X-YouTube-Client-Name", p.ClientID)
	req.Header.Set("X-YouTube-Client-Version", p.ClientVersion)
	if p.Origin != "" {
		req.Header.Set("Origin", p.Origin)
		req.Header.Set("X-Origin", p.Origin)
	}
	if p.Referer != "" {
		req.Header.Set("Referer", p.Referer)
	}
	req.Header.Set("Accept-Language", p.HL)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, maxResponseBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if len(data) > maxResponseBytes {
		return nil, resp.StatusCode, fmt.Errorf("response exceeds %d bytes", maxResponseBytes)
	}
	return data, resp.StatusCode, nil
}

func (c *Client) captureVisitorData(raw []byte) {
	// Lightweight scan without full JSON parse.
	const key = `"visitorData":"`
	s := string(raw)
	i := strings.Index(s, key)
	if i < 0 {
		return
	}
	s = s[i+len(key):]
	j := strings.IndexByte(s, '"')
	if j <= 0 {
		return
	}
	vd := s[:j]
	if vd != "" {
		c.SetVisitorData(vd)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
