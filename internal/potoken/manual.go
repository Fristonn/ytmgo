package potoken

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ManualProvider serves developer-supplied tokens.
type ManualProvider struct {
	mu        sync.RWMutex
	player    string
	session   string
	expiresAt time.Time
	ttl       time.Duration
}

// ManualConfig configures static tokens.
type ManualConfig struct {
	PlayerToken  string
	SessionToken string
	TTL          time.Duration // token lifetime from construction / Refresh
}

// NewManualProvider returns nil when both tokens are empty.
func NewManualProvider(cfg ManualConfig) *ManualProvider {
	player := trim(cfg.PlayerToken)
	session := trim(cfg.SessionToken)
	if player == "" && session == "" {
		return nil
	}
	if player == "" {
		player = session
	}
	if session == "" {
		session = player
	}
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = 6 * time.Hour
	}
	return &ManualProvider{
		player:    player,
		session:   session,
		expiresAt: time.Now().Add(ttl),
		ttl:       ttl,
	}
}

func (p *ManualProvider) Name() string { return "manual" }

func (p *ManualProvider) Enabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.player != "" || p.session != ""
}

// Tokens returns the configured tokens if still within TTL.
func (p *ManualProvider) Tokens(ctx context.Context, req Request) (Tokens, error) {
	if err := ctx.Err(); err != nil {
		return Tokens{}, err
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.player == "" && p.session == "" {
		return Tokens{}, ErrNotConfigured
	}
	now := time.Now()
	if !p.expiresAt.IsZero() && !now.Before(p.expiresAt) {
		return Tokens{}, fmt.Errorf("%w: manual tokens expired (refresh via config/restart)", ErrUnavailable)
	}
	return Tokens{
		Player:    p.player,
		Session:   p.session,
		ExpiresAt: p.expiresAt,
		Client:    req.ClientName,
		Visitor:   req.VisitorData,
		VideoID:   req.VideoID,
	}, nil
}

// Invalidate forces manual tokens to expire.
func (p *ManualProvider) Invalidate(videoID, visitorData string) {
	_ = videoID
	_ = visitorData
	p.mu.Lock()
	p.expiresAt = time.Now().Add(-time.Second)
	p.mu.Unlock()
}

// Refresh replaces manual tokens.
func (p *ManualProvider) Refresh(player, session string, ttl time.Duration) error {
	player, session = trim(player), trim(session)
	if player == "" && session == "" {
		return fmt.Errorf("%w: empty manual tokens", ErrUnavailable)
	}
	if player == "" {
		player = session
	}
	if session == "" {
		session = player
	}
	if ttl <= 0 {
		ttl = p.ttl
	}
	p.mu.Lock()
	p.player = player
	p.session = session
	p.ttl = ttl
	p.expiresAt = time.Now().Add(ttl)
	p.mu.Unlock()
	return nil
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 {
		c := s[len(s)-1]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		s = s[:len(s)-1]
	}
	return s
}
