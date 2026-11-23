package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	Addr              string
	APIKeys           []string
	RequireAPIKey     bool
	CORSOrigins       []string
	HL                string
	GL                string
	RequestTimeout    time.Duration
	UpstreamTimeout   time.Duration
	CacheTTL          time.Duration
	CacheMaxEntries   int
	RateLimitRPS      float64
	RateLimitBurst    int
	MaxRetries        int
	CipherServiceURL  string
	ExposeStreamURLs  bool
	LogLevel          string

	// PO token settings
	PoTokenProvider       string // off|manual|http|"" (auto)
	PoTokenPlayer         string
	PoTokenSession        string
	PoTokenManualTTL      time.Duration
	PoTokenServiceURL     string
	PoTokenServicePath    string
	PoTokenServiceGetPot  string
	PoTokenServiceAuth    string
	PoTokenServiceAPIKey  string
	PoTokenServiceTimeout time.Duration
	PoTokenCacheTTL       time.Duration
	PoTokenCacheMax       int
	PoTokenRequiredForWeb bool // if true, WEB player fails when tokens unavailable
}

// Load reads configuration from the environment with sensible defaults.
func Load() Config {
	cfg := Config{
		Addr:              env("ADDR", ":8080"),
		HL:                env("YT_HL", "en"),
		GL:                env("YT_GL", "US"),
		RequestTimeout:    envDuration("REQUEST_TIMEOUT", 30*time.Second),
		UpstreamTimeout:   envDuration("UPSTREAM_TIMEOUT", 15*time.Second),
		CacheTTL:          envDuration("CACHE_TTL", 5*time.Minute),
		CacheMaxEntries:   envInt("CACHE_MAX_ENTRIES", 2048),
		RateLimitRPS:      envFloat("RATE_LIMIT_RPS", 20),
		RateLimitBurst:    envInt("RATE_LIMIT_BURST", 40),
		MaxRetries:        envInt("MAX_RETRIES", 2),
		CipherServiceURL:  env("CIPHER_SERVICE_URL", ""),
		ExposeStreamURLs:  envBool("EXPOSE_STREAM_URLS", true),
		LogLevel:          env("LOG_LEVEL", "info"),
		RequireAPIKey:     envBool("REQUIRE_API_KEY", false),

		PoTokenProvider:       env("PO_TOKEN_PROVIDER", ""),
		PoTokenPlayer:         env("PO_TOKEN_PLAYER", ""),
		PoTokenSession:        env("PO_TOKEN_SESSION", ""),
		PoTokenManualTTL:      envDuration("PO_TOKEN_MANUAL_TTL", 6*time.Hour),
		PoTokenServiceURL:     env("PO_TOKEN_SERVICE_URL", ""),
		PoTokenServicePath:    env("PO_TOKEN_SERVICE_PATH", "/tokens"),
		PoTokenServiceGetPot:  env("PO_TOKEN_SERVICE_GET_POT", "/get_pot"),
		PoTokenServiceAuth:    env("PO_TOKEN_SERVICE_AUTH", ""),
		PoTokenServiceAPIKey:  env("PO_TOKEN_SERVICE_API_KEY", ""),
		PoTokenServiceTimeout: envDuration("PO_TOKEN_SERVICE_TIMEOUT", 10*time.Second),
		PoTokenCacheTTL:       envDuration("PO_TOKEN_CACHE_TTL", 4*time.Minute),
		PoTokenCacheMax:       envInt("PO_TOKEN_CACHE_MAX", 256),
		PoTokenRequiredForWeb: envBool("PO_TOKEN_REQUIRED_FOR_WEB", false),
	}

	if keys := env("API_KEYS", ""); keys != "" {
		for _, k := range strings.Split(keys, ",") {
			k = strings.TrimSpace(k)
			if k != "" {
				cfg.APIKeys = append(cfg.APIKeys, k)
			}
		}
	}
	if origins := env("CORS_ORIGINS", "*"); origins != "" {
		cfg.CORSOrigins = strings.Split(origins, ",")
		for i := range cfg.CORSOrigins {
			cfg.CORSOrigins[i] = strings.TrimSpace(cfg.CORSOrigins[i])
		}
	}
	return cfg
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return unquote(v)
	}
	return def
}

// unquote strips optional surrounding double or single quotes from env values.
func unquote(v string) string {
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			return v[1 : len(v)-1]
		}
	}
	return v
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
