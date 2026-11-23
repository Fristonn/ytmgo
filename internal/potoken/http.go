package potoken

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPConfig configures an external PO-token HTTP service.
type HTTPConfig struct {
	BaseURL    string
	TokensPath string // default /tokens
	GetPotPath string // default /get_pot (bgutil fallback)
	AuthHeader string // full Authorization value, e.g. "Bearer …"
	APIKey     string // sent as X-API-Key when set
	Timeout    time.Duration
	CacheTTL   time.Duration
	CacheMax   int
	HTTPClient *http.Client
}

// HTTPProvider fetches tokens from an external service. No generator is bundled.
type HTTPProvider struct {
	base       string
	tokensPath string
	getPotPath string
	auth       string
	apiKey     string
	client     *http.Client
	cache      *tokenCache
}

// NewHTTPProvider returns nil when BaseURL is empty.
func NewHTTPProvider(cfg HTTPConfig) *HTTPProvider {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		return nil
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: timeout}
	} else {
		hc = &http.Client{Timeout: timeout, Transport: hc.Transport}
	}
	tokensPath := cfg.TokensPath
	if tokensPath == "" {
		tokensPath = "/tokens"
	}
	getPotPath := cfg.GetPotPath
	if getPotPath == "" {
		getPotPath = "/get_pot"
	}
	return &HTTPProvider{
		base:       base,
		tokensPath: ensureSlash(tokensPath),
		getPotPath: ensureSlash(getPotPath),
		auth:       strings.TrimSpace(cfg.AuthHeader),
		apiKey:     strings.TrimSpace(cfg.APIKey),
		client:     hc,
		cache:      newTokenCache(cfg.CacheTTL, cfg.CacheMax),
	}
}

func (p *HTTPProvider) Name() string    { return "http" }
func (p *HTTPProvider) Enabled() bool   { return p != nil && p.base != "" }

func (p *HTTPProvider) Tokens(ctx context.Context, req Request) (Tokens, error) {
	if p == nil || p.base == "" {
		return Tokens{}, ErrNotConfigured
	}
	if req.VideoID == "" || req.VisitorData == "" {
		return Tokens{}, fmt.Errorf("%w: videoId and visitorData required", ErrUnavailable)
	}
	if tok, ok := p.cache.get(req); ok && tok.Valid(req.VideoID, req.VisitorData, time.Now()) {
		return tok, nil
	}
	tok, err := p.fetch(ctx, req)
	if err != nil {
		return Tokens{}, err
	}
	tok = normalize(tok)
	if tok.Player == "" && tok.Session == "" {
		return Tokens{}, fmt.Errorf("%w: empty response from provider", ErrUnavailable)
	}
	p.cache.set(req, tok)
	return tok, nil
}

func (p *HTTPProvider) Invalidate(videoID, visitorData string) {
	if p != nil && p.cache != nil {
		p.cache.invalidate(videoID, visitorData)
	}
}

func (p *HTTPProvider) fetch(ctx context.Context, req Request) (Tokens, error) {
	body := map[string]string{
		"videoId":     req.VideoID,
		"visitorData": req.VisitorData,
		"clientName":  req.ClientName,
	}
	if tok, err := p.postJSON(ctx, p.base+p.tokensPath, body); err == nil && (tok.Player != "" || tok.Session != "") {
		return tok, nil
	}

	tok, err := p.postJSON(ctx, p.base+p.getPotPath, map[string]string{
		"content_binding": req.VideoID,
		"visitor_data":    req.VisitorData,
		"client_name":     req.ClientName,
	})
	if err != nil {
		tok, err = p.postJSON(ctx, p.base+p.getPotPath, map[string]string{
			"content_binding": req.VisitorData,
		})
		if err != nil {
			return Tokens{}, fmt.Errorf("%w: external provider request failed", ErrUnavailable)
		}
	}
	return tok, nil
}

type wireTokens struct {
	PlayerToken      string  `json:"playerToken"`
	SessionToken     string  `json:"sessionToken"`
	PoToken          string  `json:"poToken"`
	PoTokenSnake     string  `json:"po_token"`
	Pot              string  `json:"pot"`
	ExpiresInSeconds float64 `json:"expiresInSeconds"`
	ExpiresIn        float64 `json:"expires_in"`
	TTLSeconds       float64 `json:"ttlSeconds"`
}

func (p *HTTPProvider) postJSON(ctx context.Context, endpoint string, body any) (Tokens, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return Tokens{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return Tokens{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if p.auth != "" {
		req.Header.Set("Authorization", p.auth)
	}
	if p.apiKey != "" {
		req.Header.Set("X-API-Key", p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return Tokens{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Tokens{}, err
	}
	if resp.StatusCode >= 400 {
		return Tokens{}, fmt.Errorf("po token service HTTP %d", resp.StatusCode)
	}
	var wire wireTokens
	if err := json.Unmarshal(data, &wire); err != nil {
		return Tokens{}, fmt.Errorf("invalid po token response")
	}
	tok := Tokens{
		Player:  firstNonEmpty(wire.PlayerToken, wire.PoToken, wire.PoTokenSnake),
		Session: firstNonEmpty(wire.SessionToken, wire.Pot, wire.PoToken, wire.PoTokenSnake),
	}
	sec := wire.ExpiresInSeconds
	if sec <= 0 {
		sec = wire.ExpiresIn
	}
	if sec <= 0 {
		sec = wire.TTLSeconds
	}
	if sec > 0 {
		tok.ExpiresAt = time.Now().Add(time.Duration(sec) * time.Second)
	}
	return tok, nil
}

func normalize(t Tokens) Tokens {
	if t.Player == "" {
		t.Player = t.Session
	}
	if t.Session == "" {
		t.Session = t.Player
	}
	return t
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func ensureSlash(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		return "/" + p
	}
	return p
}
