package parser

import (
	"strconv"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/Fristonn/ytmgo/internal/models"
)

// RunsText joins Innertube text runs into a single string.
func RunsText(v gjson.Result) string {
	if !v.Exists() {
		return ""
	}
	// Nested description shelves (playlist / album headers).
	if shelf := v.Get("musicDescriptionShelfRenderer.description"); shelf.Exists() {
		if t := RunsText(shelf); t != "" {
			return t
		}
	}
	if t := v.Get("runs.#.text"); t.Exists() {
		var b strings.Builder
		t.ForEach(func(_, r gjson.Result) bool {
			b.WriteString(r.String())
			return true
		})
		return b.String()
	}
	if s := v.Get("simpleText"); s.Exists() {
		return s.String()
	}
	// Avoid dumping raw nested JSON objects as "text".
	if v.IsObject() || v.IsArray() {
		return ""
	}
	return v.String()
}

// OddElements returns every other run starting at 0 (name separators are skipped).
func OddElements(runs gjson.Result) []gjson.Result {
	var out []gjson.Result
	runs.ForEach(func(i, r gjson.Result) bool {
		if i.Int()%2 == 0 {
			out = append(out, r)
		}
		return true
	})
	return out
}

// ParseDuration converts "M:SS" / "H:MM:SS" to seconds.
func ParseDuration(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	parts := strings.Split(s, ":")
	if len(parts) > 3 {
		return 0
	}
	var sec int
	mul := 1
	for i := len(parts) - 1; i >= 0; i-- {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return 0
		}
		sec += n * mul
		mul *= 60
	}
	return sec
}

// Thumbnails extracts artwork variants from a thumbnails array.
func Thumbnails(v gjson.Result) []models.Artwork {
	arr := v
	if v.Get("thumbnails").Exists() {
		arr = v.Get("thumbnails")
	}
	var out []models.Artwork
	arr.ForEach(func(_, t gjson.Result) bool {
		url := t.Get("url").String()
		if url == "" {
			return true
		}
		out = append(out, models.Artwork{
			URL:    url,
			Width:  int(t.Get("width").Int()),
			Height: int(t.Get("height").Int()),
		})
		return true
	})
	return out
}

// MusicThumbnail extracts from musicThumbnailRenderer.
func MusicThumbnail(v gjson.Result) []models.Artwork {
	path := v
	if v.Get("musicThumbnailRenderer").Exists() {
		path = v.Get("musicThumbnailRenderer.thumbnail")
	} else if v.Get("thumbnail.musicThumbnailRenderer").Exists() {
		path = v.Get("thumbnail.musicThumbnailRenderer.thumbnail")
	} else if v.Get("thumbnailRenderer.musicThumbnailRenderer").Exists() {
		path = v.Get("thumbnailRenderer.musicThumbnailRenderer.thumbnail")
	}
	return Thumbnails(path)
}

// IsExplicit checks for MUSIC_EXPLICIT_BADGE.
func IsExplicit(badges gjson.Result) *bool {
	found := false
	badges.ForEach(func(_, b gjson.Result) bool {
		icon := b.Get("musicInlineBadgeRenderer.icon.iconType").String()
		if icon == "MUSIC_EXPLICIT_BADGE" {
			found = true
			return false
		}
		return true
	})
	if !found {
		return nil
	}
	t := true
	return &t
}

// ContinuationToken extracts a next-continuation token from common locations.
func ContinuationToken(v gjson.Result) string {
	paths := []string{
		"continuations.0.nextContinuationData.continuation",
		"continuationItemRenderer.continuationEndpoint.continuationCommand.token",
		"continuationItemRenderer.continuationEndpoint.commandExecutorCommand.commands.1.continuationCommand.token",
	}
	for _, p := range paths {
		if t := v.Get(p); t.Exists() && t.String() != "" {
			return t.String()
		}
	}
	// Scan contents for continuationItemRenderer
	var token string
	v.Get("contents").ForEach(func(_, item gjson.Result) bool {
		if t := item.Get("continuationItemRenderer.continuationEndpoint.continuationCommand.token"); t.Exists() {
			token = t.String()
			return false
		}
		if t := item.Get("continuationItemRenderer.continuationEndpoint.commandExecutorCommand.commands.1.continuationCommand.token"); t.Exists() {
			token = t.String()
			return false
		}
		return true
	})
	return token
}

// ArtistRefsFromRuns parses artist refs from mixed song metadata runs.
// Album titles, durations, years, and view counts are filtered out.
func ArtistRefsFromRuns(runs gjson.Result) []models.ArtistRef {
	return ParseSongRuns(runs, false).Artists
}

// BestSquare picks the largest near-square artwork (aspect ~1).
func BestSquare(arts []models.Artwork) []models.Artwork {
	if len(arts) == 0 {
		return nil
	}
	var squares []models.Artwork
	for _, a := range arts {
		if a.Width > 0 && a.Height > 0 {
			ratio := float64(a.Width) / float64(a.Height)
			if ratio >= 0.9 && ratio <= 1.1 {
				squares = append(squares, a)
			}
		}
	}
	if len(squares) == 0 {
		// Prefer music.googleusercontent / lh3 square hosts over i.ytimg video thumbs.
		for _, a := range arts {
			if strings.Contains(a.URL, "googleusercontent") || strings.Contains(a.URL, "ggpht.com") {
				squares = append(squares, a)
			}
		}
	}
	if len(squares) == 0 {
		return arts
	}
	return squares
}

// PreferMusicArtwork filters out 16:9 YouTube video thumbnails when square art exists.
func PreferMusicArtwork(arts []models.Artwork) []models.Artwork {
	sq := BestSquare(arts)
	if len(sq) > 0 {
		return sq
	}
	return arts
}
