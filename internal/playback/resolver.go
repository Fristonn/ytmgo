package playback

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/Fristonn/ytmgo/internal/cipher"
	"github.com/Fristonn/ytmgo/internal/innertube"
	"github.com/Fristonn/ytmgo/internal/innertube/clients"
	"github.com/Fristonn/ytmgo/internal/models"
)

// Resolver resolves playable audio streams with client fallback.
type Resolver struct {
	innertube *innertube.Client
	cipher    *cipher.Manager
}

// NewResolver creates a stream resolver.
func NewResolver(ic *innertube.Client, cm *cipher.Manager) *Resolver {
	return &Resolver{innertube: ic, cipher: cm}
}

// Resolve returns audio formats for a video ID.
func (r *Resolver) Resolve(ctx context.Context, videoID string) (*models.StreamResponse, error) {
	if videoID == "" {
		return nil, fmt.Errorf("video id required")
	}
	var lastErr error
	for _, name := range clients.PlayerFallbackOrder {
		if !r.innertube.Tracker().Available(name) {
			continue
		}
		raw, used, err := r.innertube.Player(ctx, name, videoID, "")
		if err != nil {
			lastErr = err
			continue
		}
		title, author, channelID, duration, expires, artwork, _, status, reason := ParseFormats(raw)
		if status != "" && status != "OK" {
			lastErr = fmt.Errorf("playability %s: %s (client %s)", status, reason, used)
			r.innertube.Tracker().MarkFailure(used, false, lastErr.Error())
			continue
		}

		formats, err := r.resolveFormatURLs(ctx, raw, used)
		if err != nil {
			lastErr = err
			continue
		}
		audio := AudioOnly(formats)
		if len(audio) == 0 {
			lastErr = fmt.Errorf("no audio formats from client %s", used)
			continue
		}
		// Verify at least one format has a usable URL.
		usable := false
		for _, a := range audio {
			if a.URL != "" && strings.HasPrefix(a.URL, "http") {
				usable = true
				break
			}
		}
		if !usable {
			lastErr = fmt.Errorf("no usable stream urls from client %s", used)
			continue
		}

		return &models.StreamResponse{
			ID:               videoID,
			Title:            title,
			Author:           author,
			ChannelID:        channelID,
			Duration:         duration,
			Artwork:          artwork,
			Client:           used,
			ExpiresInSeconds: expires,
			AudioFormats:     audio,
			Formats:          formats,
		}, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("all player clients failed")
	}
	return nil, lastErr
}

func (r *Resolver) resolveFormatURLs(ctx context.Context, raw []byte, clientName string) ([]models.StreamFormat, error) {
	root := gjson.ParseBytes(raw)
	expires := int(root.Get("streamingData.expiresInSeconds").Int())
	p, _ := r.innertube.Profile(clientName)

	var playerScript string
	var formats []models.StreamFormat

	collect := func(arr gjson.Result) error {
		var err error
		arr.ForEach(func(_, f gjson.Result) bool {
			mime := f.Get("mimeType").String()
			cf := ExtractCipherFields(f)
			streamURL := cf.URL

			needsCipher := cf.Signature != "" || (cf.N != "" && p.RequireCipher)
			if needsCipher && r.cipher != nil {
				if playerScript == "" {
					playerScript, err = r.cipher.FetchPlayerScript(ctx)
					if err != nil {
						return false
					}
				}
				streamURL, err = r.cipher.ResolveURL(ctx, cf.URL, cf.Signature, cf.SigKey, cf.N, playerScript)
				if err != nil {
					// Keep undeciphered URL if present; may still work for some clients.
					if cf.URL == "" {
						return false
					}
					streamURL = cf.URL
					err = nil
				}
			}

			if pot := r.innertube.LastStreamPoToken(); pot != "" && streamURL != "" {
				streamURL = applyPot(streamURL, pot)
			}

			sf := models.StreamFormat{
				Itag:             int(f.Get("itag").Int()),
				URL:              streamURL,
				MimeType:         mime,
				Bitrate:          int(f.Get("bitrate").Int()),
				AverageBitrate:   int(f.Get("averageBitrate").Int()),
				ContentLength:    f.Get("contentLength").Int(),
				AudioQuality:     f.Get("audioQuality").String(),
				AudioSampleRate:  f.Get("audioSampleRate").String(),
				AudioChannels:    int(f.Get("audioChannels").Int()),
				IsAudioOnly:      !f.Get("width").Exists() && strings.HasPrefix(mime, "audio/"),
				ExpiresInSeconds: expires,
				Codec:            codecFromMime(mime),
				Container:        containerFromMime(mime),
			}
			formats = append(formats, sf)
			return true
		})
		return err
	}

	if err := collect(root.Get("streamingData.adaptiveFormats")); err != nil {
		return formats, err
	}
	_ = collect(root.Get("streamingData.formats"))
	return formats, nil
}

func applyPot(rawURL, pot string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		sep := "?"
		if strings.Contains(rawURL, "?") {
			sep = "&"
		}
		return rawURL + sep + "pot=" + url.QueryEscape(pot)
	}
	q := u.Query()
	q.Set("pot", pot)
	u.RawQuery = q.Encode()
	return u.String()
}
