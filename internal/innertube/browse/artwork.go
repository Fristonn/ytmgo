package browse

import (
	"github.com/Fristonn/ytmgo/internal/innertube/parser"
	"github.com/Fristonn/ytmgo/internal/models"

	"github.com/tidwall/gjson"
)

// ResolveArtwork classifies and cleans artwork from a thumbnail node.
func ResolveArtwork(node gjson.Result, kind string) []models.Artwork {
	arts := parser.MusicThumbnail(node)
	if len(arts) == 0 {
		arts = parser.Thumbnails(node)
	}
	switch kind {
	case "album", "track", "playlist":
		return parser.PreferMusicArtwork(arts)
	case "avatar":
		return parser.PreferMusicArtwork(arts)
	case "banner":
		// Keep all banner variants including wide images.
		return arts
	case "video":
		return arts
	default:
		return arts
	}
}

// BannerFromHeader extracts artist page banners from immersive/visual headers.
func BannerFromHeader(header gjson.Result) []models.Artwork {
	paths := []string{
		"musicImmersiveHeaderRenderer.thumbnail.musicThumbnailRenderer.thumbnail",
		"musicVisualHeaderRenderer.foregroundThumbnail.musicThumbnailRenderer.thumbnail",
		"musicVisualHeaderRenderer.thumbnail.musicThumbnailRenderer.thumbnail",
	}
	for _, p := range paths {
		if arts := parser.Thumbnails(header.Get(p)); len(arts) > 0 {
			// Immersive thumbnail is often the avatar; visual header has banner-like images.
			return arts
		}
	}
	return nil
}

// AvatarFromHeader extracts artist avatar images.
func AvatarFromHeader(header gjson.Result) []models.Artwork {
	if arts := parser.MusicThumbnail(header.Get("musicImmersiveHeaderRenderer.thumbnail")); len(arts) > 0 {
		return parser.PreferMusicArtwork(arts)
	}
	if arts := parser.MusicThumbnail(header.Get("musicVisualHeaderRenderer.foregroundThumbnail")); len(arts) > 0 {
		return parser.PreferMusicArtwork(arts)
	}
	return nil
}
