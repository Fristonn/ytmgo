package potoken

import (
	"net/http"
	"strings"
	"time"
)

// Config selects and configures a PO-token provider. Empty/off → nil provider.
type Config struct {
	Provider string // off|manual|http|auto

	PlayerToken  string
	SessionToken string
	ManualTTL    time.Duration

	ServiceURL     string
	ServicePath    string
	ServiceGetPot  string
	ServiceAuth    string // Authorization header value
	ServiceAPIKey  string
	ServiceTimeout time.Duration
	CacheTTL       time.Duration
	CacheMax       int

	HTTPClient *http.Client
}

// NewFromConfig builds a Provider or returns nil when disabled.
func NewFromConfig(cfg Config) Provider {
	mode := strings.ToLower(strings.TrimSpace(cfg.Provider))
	switch mode {
	case "off", "none", "disabled":
		return nil
	case "manual":
		return asProvider(NewManualProvider(ManualConfig{
			PlayerToken:  cfg.PlayerToken,
			SessionToken: cfg.SessionToken,
			TTL:          cfg.ManualTTL,
		}))
	case "http", "external":
		return asProvider(NewHTTPProvider(HTTPConfig{
			BaseURL:    cfg.ServiceURL,
			TokensPath: cfg.ServicePath,
			GetPotPath: cfg.ServiceGetPot,
			AuthHeader: cfg.ServiceAuth,
			APIKey:     cfg.ServiceAPIKey,
			Timeout:    cfg.ServiceTimeout,
			CacheTTL:   cfg.CacheTTL,
			CacheMax:   cfg.CacheMax,
			HTTPClient: cfg.HTTPClient,
		}))
	default:
		if strings.TrimSpace(cfg.ServiceURL) != "" {
			return asProvider(NewHTTPProvider(HTTPConfig{
				BaseURL:    cfg.ServiceURL,
				TokensPath: cfg.ServicePath,
				GetPotPath: cfg.ServiceGetPot,
				AuthHeader: cfg.ServiceAuth,
				APIKey:     cfg.ServiceAPIKey,
				Timeout:    cfg.ServiceTimeout,
				CacheTTL:   cfg.CacheTTL,
				CacheMax:   cfg.CacheMax,
				HTTPClient: cfg.HTTPClient,
			}))
		}
		return asProvider(NewManualProvider(ManualConfig{
			PlayerToken:  cfg.PlayerToken,
			SessionToken: cfg.SessionToken,
			TTL:          cfg.ManualTTL,
		}))
	}
}

func asProvider[T Provider](p T) Provider {
	var anyProv Provider = p
	if anyProv == nil {
		return nil
	}
	switch v := any(p).(type) {
	case *ManualProvider:
		if v == nil {
			return nil
		}
	case *HTTPProvider:
		if v == nil {
			return nil
		}
	}
	return p
}
