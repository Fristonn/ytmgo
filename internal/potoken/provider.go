package potoken

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrUnavailable means a provider is configured but cannot supply tokens right now.
var ErrUnavailable = errors.New("po token unavailable")

// ErrNotConfigured means no provider was set up (feature is off).
var ErrNotConfigured = errors.New("po token provider not configured")

// Tokens holds player and session PO tokens.
type Tokens struct {
	Player    string
	Session   string
	ExpiresAt time.Time // zero = unknown / use provider default TTL
	Client    string    // optional client binding (WEB, WEB_REMIX, …)
	Visitor   string    // visitorData binding
	VideoID   string    // content binding
}

// Valid reports whether tokens are usable for the given binding.
func (t Tokens) Valid(videoID, visitorData string, now time.Time) bool {
	if t.Player == "" && t.Session == "" {
		return false
	}
	if !t.ExpiresAt.IsZero() && !now.Before(t.ExpiresAt) {
		return false
	}
	if t.VideoID != "" && videoID != "" && t.VideoID != videoID {
		return false
	}
	if t.Visitor != "" && visitorData != "" && t.Visitor != visitorData {
		return false
	}
	return true
}

// Request describes a token mint/lookup binding.
type Request struct {
	VideoID     string
	VisitorData string
	ClientName  string
}

// Provider is a pluggable PO-token source.
type Provider interface {
	Name() string
	Enabled() bool
	Tokens(ctx context.Context, req Request) (Tokens, error)
	Invalidate(videoID, visitorData string)
}

// Status is a safe, non-secret health snapshot.
type Status struct {
	Configured bool   `json:"configured"`
	Provider   string `json:"provider,omitempty"`
	Available  bool   `json:"available"`
	Message    string `json:"message,omitempty"`
}

// Snapshot returns a logging-safe status for a provider (nil-safe).
func Snapshot(p Provider) Status {
	if !Configured(p) {
		return Status{Configured: false, Provider: "none", Available: false, Message: "disabled"}
	}
	return Status{Configured: true, Provider: p.Name(), Available: true, Message: "ready"}
}

// Configured reports whether a usable provider is present (handles typed nils).
func Configured(p Provider) bool {
	if p == nil {
		return false
	}
	switch v := p.(type) {
	case *ManualProvider:
		return v != nil && v.Enabled()
	case *HTTPProvider:
		return v != nil && v.Enabled()
	default:
		return p.Enabled()
	}
}

// Redact returns a string safe for logs (never includes raw tokens).
func Redact(t Tokens) string {
	return fmt.Sprintf("poToken{player:%s session:%s client:%q video:%q expires:%v}",
		presence(t.Player), presence(t.Session), t.Client, t.VideoID, t.ExpiresAt.IsZero())
}

func presence(s string) string {
	if s == "" {
		return "absent"
	}
	return "present"
}
