package playback

import (
	"strconv"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/Fristonn/ytmgo/internal/models"
)

// ParseFormats extracts stream formats from a player response.
func ParseFormats(raw []byte) (title, author, channelID string, duration, expires int, artwork []models.Artwork, formats []models.StreamFormat, status, reason string) {
	root := gjson.ParseBytes(raw)
	status = root.Get("playabilityStatus.status").String()
	reason = root.Get("playabilityStatus.reason").String()

	vd := root.Get("videoDetails")
	title = vd.Get("title").String()
	author = vd.Get("author").String()
	channelID = vd.Get("channelId").String()
	duration, _ = strconv.Atoi(vd.Get("lengthSeconds").String())
	artwork = thumbnails(vd.Get("thumbnail.thumbnails"))

	expires = int(root.Get("streamingData.expiresInSeconds").Int())

	collect := func(arr gjson.Result) {
		arr.ForEach(func(_, f gjson.Result) bool {
			mime := f.Get("mimeType").String()
			itag := int(f.Get("itag").Int())
			url := f.Get("url").String()
			sf := models.StreamFormat{
				Itag:             itag,
				URL:              url,
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
	}

	collect(root.Get("streamingData.adaptiveFormats"))
	collect(root.Get("streamingData.formats"))
	return
}

// AudioOnly filters audio-only formats (Opus / AAC).
func AudioOnly(formats []models.StreamFormat) []models.StreamFormat {
	var out []models.StreamFormat
	for _, f := range formats {
		if f.IsAudioOnly || strings.HasPrefix(f.MimeType, "audio/") {
			f.IsAudioOnly = true
			out = append(out, f)
		}
	}
	return out
}

func codecFromMime(mime string) string {
	if i := strings.Index(mime, "codecs="); i >= 0 {
		c := mime[i+7:]
		c = strings.Trim(c, `"`)
		if j := strings.IndexByte(c, '"'); j >= 0 {
			c = c[:j]
		}
		return c
	}
	return ""
}

func containerFromMime(mime string) string {
	if strings.HasPrefix(mime, "audio/webm") || strings.HasPrefix(mime, "video/webm") {
		return "webm"
	}
	if strings.HasPrefix(mime, "audio/mp4") || strings.HasPrefix(mime, "video/mp4") {
		return "mp4"
	}
	if i := strings.IndexByte(mime, '/'); i >= 0 {
		rest := mime[i+1:]
		if j := strings.IndexByte(rest, ';'); j >= 0 {
			return rest[:j]
		}
		return rest
	}
	return ""
}

func thumbnails(arr gjson.Result) []models.Artwork {
	var out []models.Artwork
	arr.ForEach(func(_, t gjson.Result) bool {
		out = append(out, models.Artwork{
			URL:    t.Get("url").String(),
			Width:  int(t.Get("width").Int()),
			Height: int(t.Get("height").Int()),
		})
		return true
	})
	return PreferSquareThumbs(out)
}

// PreferSquareThumbs keeps near-square / Music CDN art when mixed with 16:9 video thumbs.
func PreferSquareThumbs(arts []models.Artwork) []models.Artwork {
	if len(arts) == 0 {
		return arts
	}
	var squares []models.Artwork
	var musicCDN []models.Artwork
	for _, a := range arts {
		if a.Width > 0 && a.Height > 0 {
			ratio := float64(a.Width) / float64(a.Height)
			if ratio >= 0.9 && ratio <= 1.1 {
				squares = append(squares, a)
				continue
			}
		}
		u := a.URL
		if strings.Contains(u, "googleusercontent") || strings.Contains(u, "ggpht.com") {
			musicCDN = append(musicCDN, a)
		}
	}
	if len(squares) > 0 {
		return squares
	}
	if len(musicCDN) > 0 {
		return musicCDN
	}
	return arts
}
