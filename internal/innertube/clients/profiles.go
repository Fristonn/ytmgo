package clients

import "time"

// Capability describes what a client profile supports.
type Capability int

const (
	CapSearch Capability = 1 << iota
	CapBrowse
	CapPlayer
	CapSuggestions
	CapMusicMetadata
)

// Profile is a verified Innertube client identity.
type Profile struct {
	Name            string
	ClientName      string
	ClientVersion   string
	ClientID        string
	APIKey          string
	UserAgent       string
	Referer         string
	Origin          string
	HL              string
	GL              string
	DeviceMake      string
	DeviceModel     string
	OSName          string
	OSVersion       string
	AndroidSDK      int
	Platform        string
	RequireCipher   bool
	PreferAudioOnly bool
	Capabilities    Capability
	BaseURL         string
}

// Well-known public Innertube API keys (embedded in official clients; not secrets).
const (
	APIKeyWeb     = "AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"
	APIKeyAndroid = "AIzaSyA8eiZmM1FaDVjRy-df2KTyQ_vz_yYM39w"
	APIKeyIOS     = "AIzaSyB-63vPrdThhKuerbB2N_l7Kwwcxj6yUAc"
	APIKeyMusic   = "AIzaSyC9XL3ZjWddXya6X74dJoCTL-WEYFDNX30"
	APIKeyTVEmbed = "AIzaSyDCU8hByM-4DrUqRUYnGn-3llEO78bcxq8"
)

// Profiles returns the built-in client set with locale applied.
func Profiles(hl, gl string) map[string]Profile {
	webUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

	all := []Profile{
		{
			Name: "WEB_REMIX", ClientName: "WEB_REMIX", ClientVersion: "1.20250804.16.00",
			ClientID: "67", APIKey: APIKeyMusic, UserAgent: webUA,
			Referer: "https://music.youtube.com/", Origin: "https://music.youtube.com",
			Platform: "DESKTOP", BaseURL: "https://music.youtube.com/youtubei/v1",
			RequireCipher: true, PreferAudioOnly: true,
			Capabilities: CapSearch | CapBrowse | CapPlayer | CapSuggestions | CapMusicMetadata,
		},
		{
			Name: "ANDROID_MUSIC", ClientName: "ANDROID_MUSIC", ClientVersion: "7.29.51",
			ClientID: "21", APIKey: "AIzaSyAOghZGza2MQSZkY_zfZ370N-PUdXEo8AI",
			UserAgent: "com.google.android.apps.youtube.music/7.29.51 (Linux; U; Android 14) gzip",
			OSName: "Android", OSVersion: "14", AndroidSDK: 34, Platform: "MOBILE",
			BaseURL: "https://music.youtube.com/youtubei/v1",
			RequireCipher: false, PreferAudioOnly: true,
			Capabilities: CapSearch | CapBrowse | CapPlayer | CapSuggestions | CapMusicMetadata,
		},
		{
			Name: "WEB", ClientName: "WEB", ClientVersion: "2.20250805.01.00",
			ClientID: "1", APIKey: APIKeyWeb, UserAgent: webUA,
			Referer: "https://www.youtube.com/", Origin: "https://www.youtube.com",
			Platform: "DESKTOP", BaseURL: "https://www.youtube.com/youtubei/v1",
			RequireCipher: true,
			Capabilities: CapSearch | CapBrowse | CapPlayer | CapSuggestions,
		},
		{
			Name: "MWEB", ClientName: "MWEB", ClientVersion: "2.20250805.01.00",
			ClientID: "2", APIKey: APIKeyWeb,
			UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1",
			Referer: "https://m.youtube.com/", Origin: "https://m.youtube.com",
			Platform: "MOBILE", BaseURL: "https://www.youtube.com/youtubei/v1",
			RequireCipher: true,
			Capabilities: CapSearch | CapBrowse | CapPlayer,
		},
		{
			Name: "ANDROID", ClientName: "ANDROID", ClientVersion: "21.03.36",
			ClientID: "3", APIKey: APIKeyAndroid,
			UserAgent: "com.google.android.youtube/21.03.36 (Linux; U; Android 16) gzip",
			OSName: "Android", OSVersion: "16", AndroidSDK: 36, Platform: "MOBILE",
			BaseURL: "https://www.youtube.com/youtubei/v1",
			RequireCipher: false, PreferAudioOnly: true,
			Capabilities: CapPlayer | CapBrowse | CapSearch,
		},
		{
			Name: "ANDROID_VR", ClientName: "ANDROID_VR", ClientVersion: "1.65.10",
			ClientID: "28", APIKey: APIKeyAndroid,
			UserAgent: "com.google.android.apps.youtube.vr.oculus/1.65.10 (Linux; U; Android 12L; eureka-user Build/SQ3A.220605.009.A1) gzip",
			DeviceMake: "Oculus", DeviceModel: "Quest 3", OSName: "Android", OSVersion: "12",
			AndroidSDK: 32, Platform: "MOBILE",
			BaseURL: "https://www.youtube.com/youtubei/v1",
			RequireCipher: false, PreferAudioOnly: true,
			Capabilities: CapPlayer | CapBrowse,
		},
		{
			Name: "IOS", ClientName: "IOS", ClientVersion: "20.11.6",
			ClientID: "5", APIKey: APIKeyIOS,
			UserAgent: "com.google.ios.youtube/20.11.6 (iPhone16,2; U; CPU iOS 17_5_1 like Mac OS X)",
			DeviceMake: "Apple", DeviceModel: "iPhone16,2", OSName: "iOS", OSVersion: "17.5.1.21F90",
			Platform: "MOBILE", BaseURL: "https://www.youtube.com/youtubei/v1",
			RequireCipher: false, PreferAudioOnly: true,
			Capabilities: CapPlayer | CapBrowse,
		},
		{
			Name: "WEB_EMBEDDED_PLAYER", ClientName: "WEB_EMBEDDED_PLAYER", ClientVersion: "1.20250401.01.00",
			ClientID: "56", APIKey: APIKeyWeb, UserAgent: webUA,
			Referer: "https://www.youtube.com/", Origin: "https://www.youtube.com",
			Platform: "DESKTOP", BaseURL: "https://www.youtube.com/youtubei/v1",
			RequireCipher: true, PreferAudioOnly: true,
			Capabilities: CapPlayer,
		},
		{
			Name: "TVHTML5", ClientName: "TVHTML5", ClientVersion: "7.20250319.10.00",
			ClientID: "7", APIKey: APIKeyWeb,
			UserAgent: "Mozilla/5.0 (ChromiumStylePlatform) Cobalt/Version",
			Platform: "TV", BaseURL: "https://www.youtube.com/youtubei/v1",
			RequireCipher: true, PreferAudioOnly: true,
			Capabilities: CapPlayer,
		},
		{
			Name: "TVHTML5_SIMPLY", ClientName: "TVHTML5_SIMPLY", ClientVersion: "1.0",
			ClientID: "75", APIKey: APIKeyWeb,
			UserAgent: "Mozilla/5.0 (ChromiumStylePlatform) Cobalt/Version",
			Platform: "TV", BaseURL: "https://www.youtube.com/youtubei/v1",
			RequireCipher: true, PreferAudioOnly: true,
			Capabilities: CapPlayer,
		},
		{
			Name: "TVHTML5_SIMPLY_EMBEDDED_PLAYER", ClientName: "TVHTML5_SIMPLY_EMBEDDED_PLAYER",
			ClientVersion: "2.0", ClientID: "85", APIKey: APIKeyTVEmbed,
			UserAgent: "Mozilla/5.0 (PlayStation 4 5.55) AppleWebKit/601.2 (KHTML, like Gecko)",
			Platform: "TV", BaseURL: "https://www.youtube.com/youtubei/v1",
			RequireCipher: true, PreferAudioOnly: true,
			Capabilities: CapPlayer,
		},
		{
			Name: "MEDIA_CONNECT_FRONTEND", ClientName: "MEDIA_CONNECT_FRONTEND", ClientVersion: "0.2",
			ClientID: "95", APIKey: APIKeyWeb, UserAgent: webUA,
			Platform: "DESKTOP", BaseURL: "https://www.youtube.com/youtubei/v1",
			RequireCipher: false, PreferAudioOnly: true,
			Capabilities: CapPlayer | CapBrowse,
		},
	}

	out := make(map[string]Profile, len(all))
	for _, p := range all {
		p.HL = hl
		p.GL = gl
		out[p.Name] = p
	}
	return out
}

// PlayerFallbackOrder is the preferred client order for stream resolution.
var PlayerFallbackOrder = []string{
	"ANDROID_VR",
	"ANDROID_MUSIC",
	"IOS",
	"ANDROID",
	"TVHTML5",
	"TVHTML5_SIMPLY",
	"WEB_EMBEDDED_PLAYER",
	"MEDIA_CONNECT_FRONTEND",
	"WEB",
	"WEB_REMIX",
	"MWEB",
}

// MetadataBrowseOrder prefers Music-capable clients, then general browse clients.
var MetadataBrowseOrder = []string{
	"WEB_REMIX",
	"ANDROID_MUSIC",
	"WEB",
	"MWEB",
	"ANDROID",
	"ANDROID_VR",
	"IOS",
	"MEDIA_CONNECT_FRONTEND",
}

// MetadataSearchOrder is used for search enrichment.
var MetadataSearchOrder = []string{
	"WEB_REMIX",
	"ANDROID_MUSIC",
	"WEB",
	"MWEB",
	"ANDROID",
}

// MetadataTrackOrder combines music next/browse clients and player detail clients.
var MetadataTrackOrder = []string{
	"WEB_REMIX",
	"ANDROID_MUSIC",
	"ANDROID_VR",
	"IOS",
	"ANDROID",
	"WEB",
	"MWEB",
}

// Health tracks per-client availability with cooldown.
type Health struct {
	failures  int
	cooldown  time.Time
	lastError string
}

// Tracker manages client cooldowns after failures / rate limits.
type Tracker struct {
	states map[string]*Health
}

// NewTracker creates a client health tracker.
func NewTracker() *Tracker {
	return &Tracker{states: make(map[string]*Health)}
}

// Available reports whether a client may be used now.
func (t *Tracker) Available(name string) bool {
	h, ok := t.states[name]
	if !ok {
		return true
	}
	return time.Now().After(h.cooldown)
}

// MarkSuccess clears failure state.
func (t *Tracker) MarkSuccess(name string) {
	delete(t.states, name)
}

// MarkFailure records a failure and applies exponential cooldown.
func (t *Tracker) MarkFailure(name string, rateLimited bool, errMsg string) {
	h, ok := t.states[name]
	if !ok {
		h = &Health{}
		t.states[name] = h
	}
	h.failures++
	h.lastError = errMsg
	base := time.Duration(h.failures) * 2 * time.Second
	if rateLimited {
		base = 30 * time.Second * time.Duration(h.failures)
	}
	if base > 5*time.Minute {
		base = 5 * time.Minute
	}
	h.cooldown = time.Now().Add(base)
}

// Status returns a snapshot of client health strings.
func (t *Tracker) Status() map[string]string {
	out := make(map[string]string)
	now := time.Now()
	for name, h := range t.states {
		if now.Before(h.cooldown) {
			out[name] = "cooldown"
		} else if h.failures > 0 {
			out[name] = "degraded"
		}
	}
	return out
}

// Has reports if the profile supports a capability.
func (p Profile) Has(c Capability) bool {
	return p.Capabilities&c != 0
}
