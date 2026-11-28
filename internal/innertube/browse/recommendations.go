package browse

import (
	"fmt"

	"github.com/tidwall/gjson"

	"github.com/Fristonn/ytmgo/internal/innertube/parser"
	"github.com/Fristonn/ytmgo/internal/models"
)

// ParseRelated parses related content from a next/browse response.
func ParseRelated(raw []byte) (*models.Section, error) {
	root := gjson.ParseBytes(raw)
	sec := &models.Section{Title: "Related"}

	// YouTube Music next response: playlistPanelRenderer contents
	panel := root.Get("contents.singleColumnMusicWatchNextResultsRenderer.tabbedRenderer.watchNextTabbedResultsRenderer.tabs.0.tabRenderer.content.musicQueueRenderer.content.playlistPanelRenderer")
	if panel.Exists() {
		panel.Get("contents").ForEach(func(_, item gjson.Result) bool {
			r := item.Get("playlistPanelVideoRenderer")
			if !r.Exists() {
				return true
			}
			id := r.Get("videoId").String()
			title := parser.RunsText(r.Get("title"))
			if id == "" || title == "" {
				return true
			}
			meta := parser.ParseSongRuns(r.Get("longBylineText.runs"), false)
			if len(meta.Artists) == 0 && meta.Album == nil {
				meta = parser.ParseSongRuns(r.Get("shortBylineText.runs"), false)
			}
			durText := r.Get("lengthText.runs.0.text").String()
			if durText == "" {
				durText = r.Get("lengthText.simpleText").String()
			}
			if durText == "" {
				durText = meta.DurationText
			}
			sec.Tracks = append(sec.Tracks, models.Track{
				ID:           id,
				Title:        title,
				Artists:      meta.Artists,
				Album:        meta.Album,
				Duration:     parser.ParseDuration(durText),
				DurationText: durText,
				Artwork:      parser.PreferMusicArtwork(parser.Thumbnails(r.Get("thumbnail"))),
			})
			return true
		})
	}

	// Related tab browse id is consumed by the service layer; keep queue items if present.

	// Related browse tabs (albums / artists carousels)
	root.Get("contents.sectionListRenderer.contents").ForEach(func(_, section gjson.Result) bool {
		carousel := section.Get("musicCarouselShelfRenderer")
		if !carousel.Exists() {
			return true
		}
		carousel.Get("contents").ForEach(func(_, item gjson.Result) bool {
			r := item.Get("musicTwoRowItemRenderer")
			if !r.Exists() {
				return true
			}
			switch parser.ClassifyTwoRow(r) {
			case "track":
				if t := parser.TrackFromTwoRow(r); t != nil {
					sec.Tracks = append(sec.Tracks, *t)
				}
			case "album":
				if a := parser.AlbumFromTwoRow(r); a != nil {
					sec.Albums = append(sec.Albums, *a)
				}
			case "artist":
				if a := parser.ArtistFromTwoRow(r); a != nil {
					sec.Artists = append(sec.Artists, *a)
				}
			case "playlist":
				if p := parser.PlaylistFromTwoRow(r); p != nil {
					sec.Playlists = append(sec.Playlists, *p)
				}
			}
			return true
		})
		return true
	})

	if len(sec.Tracks) == 0 && len(sec.Albums) == 0 && len(sec.Artists) == 0 {
		return nil, fmt.Errorf("no related content found")
	}
	return sec, nil
}

// ParseRelatedTab parses the dedicated related tab browse response.
func ParseRelatedTab(raw []byte) (*models.Section, error) {
	root := gjson.ParseBytes(raw)
	sec := &models.Section{Title: "Related"}

	contents := root.Get("contents.sectionListRenderer.contents")
	if !contents.Exists() {
		contents = root.Get("contents.singleColumnBrowseResultsRenderer.tabs.0.tabRenderer.content.sectionListRenderer.contents")
	}

	contents.ForEach(func(_, section gjson.Result) bool {
		if shelf := section.Get("musicShelfRenderer"); shelf.Exists() {
			shelf.Get("contents").ForEach(func(_, item gjson.Result) bool {
				if t := parser.TrackFromResponsiveList(item.Get("musicResponsiveListItemRenderer")); t != nil {
					sec.Tracks = append(sec.Tracks, *t)
				}
				return true
			})
		}
		if carousel := section.Get("musicCarouselShelfRenderer"); carousel.Exists() {
			carousel.Get("contents").ForEach(func(_, item gjson.Result) bool {
				r := item.Get("musicTwoRowItemRenderer")
				switch parser.ClassifyTwoRow(r) {
				case "album":
					if a := parser.AlbumFromTwoRow(r); a != nil {
						sec.Albums = append(sec.Albums, *a)
					}
				case "artist":
					if a := parser.ArtistFromTwoRow(r); a != nil {
						sec.Artists = append(sec.Artists, *a)
					}
				case "playlist":
					if p := parser.PlaylistFromTwoRow(r); p != nil {
						sec.Playlists = append(sec.Playlists, *p)
					}
				case "track":
					if t := parser.TrackFromTwoRow(r); t != nil {
						sec.Tracks = append(sec.Tracks, *t)
					}
				}
				return true
			})
		}
		return true
	})

	return sec, nil
}
