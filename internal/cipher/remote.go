package cipher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Manager resolves signed stream URLs via optional remote cipher.
type Manager struct {
	http       *http.Client
	remoteURL  string
	mu         sync.Mutex
	playerURL  string
	playerJS   string
	fetchedAt  time.Time
	playerTTL  time.Duration
}

// New creates a cipher manager. remoteURL may be empty.
func New(httpClient *http.Client, remoteURL string) *Manager {
	return &Manager{
		http:      httpClient,
		remoteURL: strings.TrimRight(remoteURL, "/"),
		playerTTL: 1 * time.Hour,
	}
}

// ResolveURL applies signature / n-parameter transforms when needed.
func (m *Manager) ResolveURL(ctx context.Context, streamURL, signature, sigKey, nParam, playerScript string) (string, error) {
	if streamURL == "" {
		return "", fmt.Errorf("empty stream url")
	}
	// Already usable absolute URL with no cipher fields.
	if signature == "" && nParam == "" {
		return streamURL, nil
	}
	if m.remoteURL != "" {
		return m.resolveRemote(ctx, streamURL, signature, sigKey, nParam, playerScript)
	}
	// Without a cipher service, return URL only if it has no signature requirement.
	if signature != "" {
		return "", fmt.Errorf("signature decipher required; configure CIPHER_SERVICE_URL")
	}
	// n-param without remote service: return as-is (may throttle).
	return streamURL, nil
}

type remoteResolveRequest struct {
	URL         string `json:"url"`
	Signature   string `json:"signature,omitempty"`
	SigKey      string `json:"signature_key,omitempty"`
	N           string `json:"n,omitempty"`
	PlayerURL   string `json:"player_url,omitempty"`
}

type remoteResolveResponse struct {
	ResolvedURL string `json:"resolved_url"`
	URL         string `json:"url"`
}

func (m *Manager) resolveRemote(ctx context.Context, streamURL, signature, sigKey, nParam, playerScript string) (string, error) {
	endpoint := m.remoteURL + "/resolve_url"
	body, _ := json.Marshal(remoteResolveRequest{
		URL:       streamURL,
		Signature: signature,
		SigKey:    sigKey,
		N:         nParam,
		PlayerURL: playerScript,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("cipher service HTTP %d: %s", resp.StatusCode, string(data))
	}
	var out remoteResolveResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return "", err
	}
	if out.ResolvedURL != "" {
		return out.ResolvedURL, nil
	}
	if out.URL != "" {
		return out.URL, nil
	}
	return "", fmt.Errorf("cipher service returned empty url")
}

// FetchPlayerScript downloads and caches the player JS URL from a watch page.
func (m *Manager) FetchPlayerScript(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.playerURL != "" && time.Since(m.fetchedAt) < m.playerTTL {
		return m.playerURL, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.youtube.com/iframe_api", nil)
	if err != nil {
		return "", err
	}
	resp, err := m.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	// Fallback: scrape embed page for player URL.
	_ = data
	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.youtube.com/embed/jNQXAC9IVRw", nil)
	if err != nil {
		return "", err
	}
	resp2, err := m.http.Do(req2)
	if err != nil {
		return "", err
	}
	defer resp2.Body.Close()
	html, err := io.ReadAll(io.LimitReader(resp2.Body, 2<<20))
	if err != nil {
		return "", err
	}
	s := string(html)
	const marker = `"jsUrl":"`
	i := strings.Index(s, marker)
	if i < 0 {
		const marker2 = `/s/player/`
		i = strings.Index(s, marker2)
		if i < 0 {
			return "", fmt.Errorf("player script url not found")
		}
		rest := s[i:]
		j := strings.Index(rest, `"`)
		if j < 0 {
			return "", fmt.Errorf("player script url not found")
		}
		path := rest[:j]
		if strings.HasPrefix(path, "/") {
			m.playerURL = "https://www.youtube.com" + path
		} else {
			m.playerURL = path
		}
		m.fetchedAt = time.Now()
		return m.playerURL, nil
	}
	rest := s[i+len(marker):]
	j := strings.Index(rest, `"`)
	if j < 0 {
		return "", fmt.Errorf("player script url not found")
	}
	path, err := url.QueryUnescape(strings.ReplaceAll(rest[:j], `\u002F`, "/"))
	if err != nil {
		path = rest[:j]
	}
	if strings.HasPrefix(path, "/") {
		m.playerURL = "https://www.youtube.com" + path
	} else {
		m.playerURL = path
	}
	m.fetchedAt = time.Now()
	return m.playerURL, nil
}
