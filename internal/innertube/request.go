package innertube

import (
	"encoding/json"

	"github.com/Fristonn/ytmgo/internal/innertube/clients"
)

// ContextPayload is the Innertube request context.
type ContextPayload struct {
	Client     map[string]any `json:"client"`
	User       map[string]any `json:"user,omitempty"`
	ThirdParty map[string]any `json:"thirdParty,omitempty"`
}

// buildContext creates an Innertube context for the given profile.
func buildContext(p clients.Profile, visitorData string, embedURL string) ContextPayload {
	client := map[string]any{
		"clientName":    p.ClientName,
		"clientVersion": p.ClientVersion,
		"hl":            p.HL,
		"gl":            p.GL,
	}
	if visitorData != "" {
		client["visitorData"] = visitorData
	}
	if p.Platform != "" {
		client["platform"] = p.Platform
	}
	if p.DeviceMake != "" {
		client["deviceMake"] = p.DeviceMake
	}
	if p.DeviceModel != "" {
		client["deviceModel"] = p.DeviceModel
	}
	if p.OSName != "" {
		client["osName"] = p.OSName
	}
	if p.OSVersion != "" {
		client["osVersion"] = p.OSVersion
	}
	if p.AndroidSDK > 0 {
		client["androidSdkVersion"] = p.AndroidSDK
	}
	if p.UserAgent != "" && (p.ClientName == "WEB" || p.ClientName == "WEB_REMIX" || p.ClientName == "MWEB") {
		client["userAgent"] = p.UserAgent
	}

	ctx := ContextPayload{
		Client: client,
		User:   map[string]any{"lockedSafetyMode": false},
	}
	if embedURL != "" {
		ctx.ThirdParty = map[string]any{"embedUrl": embedURL}
	}
	return ctx
}

func encodeBody(v any) ([]byte, error) {
	return json.Marshal(v)
}

// SearchBody is the /search request body.
type SearchBody struct {
	Context ContextPayload `json:"context"`
	Query   string         `json:"query,omitempty"`
	Params  string         `json:"params,omitempty"`
}

// BrowseBody is the /browse request body.
type BrowseBody struct {
	Context  ContextPayload `json:"context"`
	BrowseID string         `json:"browseId,omitempty"`
	Params   string         `json:"params,omitempty"`
}

// PlayerBody is the /player request body.
type PlayerBody struct {
	Context                     ContextPayload `json:"context"`
	VideoID                     string         `json:"videoId"`
	PlaylistID                  string         `json:"playlistId,omitempty"`
	ContentCheckOK              bool           `json:"contentCheckOk"`
	RacyCheckOK                 bool           `json:"racyCheckOk"`
	PlaybackContext             map[string]any `json:"playbackContext,omitempty"`
	ServiceIntegrityDimensions  map[string]any `json:"serviceIntegrityDimensions,omitempty"`
}

// NextBody is the /next request body.
type NextBody struct {
	Context      ContextPayload `json:"context"`
	VideoID      string         `json:"videoId,omitempty"`
	PlaylistID   string         `json:"playlistId,omitempty"`
	Continuation string         `json:"continuation,omitempty"`
}

// SuggestionsBody is the get_search_suggestions body.
type SuggestionsBody struct {
	Context ContextPayload `json:"context"`
	Input   string         `json:"input"`
}
