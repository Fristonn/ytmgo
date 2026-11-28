package search

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/Fristonn/ytmgo/internal/innertube/parser"
	"github.com/Fristonn/ytmgo/internal/models"
)

// ParseMusicSearch parses a WEB_REMIX search response.
// Handles musicCardShelfRenderer, musicShelfRenderer, and itemSectionRenderer
// (flat mixed results on unfiltered search).
func ParseMusicSearch(raw []byte, query, filter string) (*models.SearchResult, error) {
	root := gjson.ParseBytes(raw)
	if !root.IsObject() {
		return nil, fmt.Errorf("invalid search response")
	}

	result := &models.SearchResult{Query: query, Filter: filter}

	// Continuation page
	if cont := root.Get("continuationContents.musicShelfContinuation"); cont.Exists() {
		parseShelfContents(cont.Get("contents"), result, filter)
		result.Continuation = parser.ContinuationToken(cont)
		return result, nil
	}

	tabs := root.Get("contents.tabbedSearchResultsRenderer.tabs")
	if !tabs.Exists() {
		return result, nil
	}

	sectionContents := tabs.Get("0.tabRenderer.content.sectionListRenderer.contents")
	sectionContents.ForEach(func(_, section gjson.Result) bool {
		if shelf := section.Get("musicShelfRenderer"); shelf.Exists() {
			category := strings.ToLower(parser.RunsText(shelf.Get("title")))
			parseShelfContents(shelf.Get("contents"), result, category)
			if c := parser.ContinuationToken(shelf); c != "" {
				result.Continuation = c
			}
			return true
		}
		if card := section.Get("musicCardShelfRenderer"); card.Exists() {
			parseTopResultCard(card, result)
			return true
		}
		// Unfiltered search: each hit is wrapped in itemSectionRenderer.
		if isr := section.Get("itemSectionRenderer"); isr.Exists() {
			parseShelfContents(isr.Get("contents"), result, "")
		}
		return true
	})

	return result, nil
}

// parseTopResultCard handles musicCardShelfRenderer top-result cards.
func parseTopResultCard(card gjson.Result, result *models.SearchResult) {
	subtitle := parser.RunsText(card.Get("subtitle"))
	subtitleLower := strings.ToLower(subtitle)
	title := parser.RunsText(card.Get("title"))
	browseID := card.Get("title.runs.0.navigationEndpoint.browseEndpoint.browseId").String()
	if browseID == "" {
		browseID = card.Get("navigationEndpoint.browseEndpoint.browseId").String()
	}
	artwork := parser.PreferMusicArtwork(parser.MusicThumbnail(card.Get("thumbnail")))

	var cardArtist *models.ArtistRef
	switch {
	case strings.Contains(subtitleLower, "artist") || strings.HasPrefix(browseID, "UC"):
		if browseID != "" && title != "" {
			cardArtist = &models.ArtistRef{ID: browseID, Name: title}
			result.Artists = append(result.Artists, *cardArtist)
		}
	case strings.Contains(subtitleLower, "album") || strings.Contains(subtitleLower, "ep") || strings.Contains(subtitleLower, "single") || strings.HasPrefix(browseID, "MPRE"):
		if browseID != "" && title != "" {
			result.Albums = append(result.Albums, models.Album{ID: browseID, Title: title, Artwork: artwork})
		}
	case strings.Contains(subtitleLower, "playlist"):
		if browseID != "" && title != "" {
			result.Playlists = append(result.Playlists, models.Playlist{ID: strings.TrimPrefix(browseID, "VL"), Title: title, Artwork: artwork})
		}
	default:
		if vid := card.Get("onTap.watchEndpoint.videoId").String(); vid != "" {
			meta := parser.ParseSongRuns(card.Get("subtitle.runs"), true)
			result.Tracks = append(result.Tracks, models.Track{
				ID:           vid,
				Title:        title,
				Artists:      meta.Artists,
				Album:        meta.Album,
				Duration:     parser.ParseDuration(meta.DurationText),
				DurationText: meta.DurationText,
				Artwork:      artwork,
			})
		}
	}

	// Nested songs under a top-result card (e.g. artist top result).
	card.Get("contents").ForEach(func(_, item gjson.Result) bool {
		// Skip "More from YouTube" message rows.
		if item.Get("messageRenderer").Exists() {
			return true
		}
		if t := parser.TrackFromResponsiveList(item.Get("musicResponsiveListItemRenderer")); t != nil {
			if len(t.Artists) == 0 && cardArtist != nil {
				t.Artists = []models.ArtistRef{*cardArtist}
			}
			result.Tracks = append(result.Tracks, *t)
		}
		return true
	})
}

func parseShelfContents(contents gjson.Result, result *models.SearchResult, category string) {
	contents.ForEach(func(_, item gjson.Result) bool {
		r := item.Get("musicResponsiveListItemRenderer")
		if !r.Exists() {
			return true
		}
		pageType := r.Get("navigationEndpoint.browseEndpoint.browseEndpointContextSupportedConfigs.browseEndpointContextMusicConfig.pageType").String()
		browseID := r.Get("navigationEndpoint.browseEndpoint.browseId").String()
		hasVideo := r.Get("playlistItemData.videoId").Exists() || r.Get("navigationEndpoint.watchEndpoint.videoId").Exists() ||
			r.Get("overlay.musicItemThumbnailOverlayRenderer.content.musicPlayButtonRenderer.playNavigationEndpoint.watchEndpoint.videoId").Exists()

		// Prefer shelf category when present (filtered / titled shelves).
		switch {
		case strings.Contains(category, "song") || strings.Contains(category, "video"):
			if t := parser.TrackFromResponsiveList(r); t != nil {
				result.Tracks = append(result.Tracks, *t)
			}
			return true
		case strings.Contains(category, "artist"):
			if a := parser.ArtistFromResponsiveList(r); a != nil {
				result.Artists = append(result.Artists, *a)
			}
			return true
		case strings.Contains(category, "album"):
			if a := parser.AlbumFromResponsiveList(r); a != nil {
				result.Albums = append(result.Albums, *a)
			}
			return true
		case strings.Contains(category, "playlist"):
			if p := parser.PlaylistFromResponsiveList(r); p != nil {
				result.Playlists = append(result.Playlists, *p)
			}
			return true
		}

		switch {
		case hasVideo:
			if t := parser.TrackFromResponsiveList(r); t != nil {
				result.Tracks = append(result.Tracks, *t)
			}
		case pageType == "MUSIC_PAGE_TYPE_ARTIST" || strings.HasPrefix(browseID, "UC"):
			if a := parser.ArtistFromResponsiveList(r); a != nil {
				result.Artists = append(result.Artists, *a)
			}
		case pageType == "MUSIC_PAGE_TYPE_ALBUM" || strings.HasPrefix(browseID, "MPREb_"):
			if a := parser.AlbumFromResponsiveList(r); a != nil {
				result.Albums = append(result.Albums, *a)
			}
		case pageType == "MUSIC_PAGE_TYPE_PLAYLIST" || strings.HasPrefix(browseID, "VL") || strings.HasPrefix(browseID, "PL") || strings.HasPrefix(browseID, "RD"):
			if p := parser.PlaylistFromResponsiveList(r); p != nil {
				result.Playlists = append(result.Playlists, *p)
			}
		default:
			if t := parser.TrackFromResponsiveList(r); t != nil {
				result.Tracks = append(result.Tracks, *t)
			} else if a := parser.AlbumFromResponsiveList(r); a != nil {
				result.Albums = append(result.Albums, *a)
			} else if ar := parser.ArtistFromResponsiveList(r); ar != nil {
				result.Artists = append(result.Artists, *ar)
			} else if p := parser.PlaylistFromResponsiveList(r); p != nil {
				result.Playlists = append(result.Playlists, *p)
			}
		}
		return true
	})
}

// ParseSuggestions parses get_search_suggestions response.
func ParseSuggestions(raw []byte) ([]models.Suggestion, []models.Track, error) {
	root := gjson.ParseBytes(raw)
	var suggestions []models.Suggestion
	var tracks []models.Track

	root.Get("contents").ForEach(func(i, section gjson.Result) bool {
		section.Get("searchSuggestionsSectionRenderer.contents").ForEach(func(_, item gjson.Result) bool {
			if s := item.Get("searchSuggestionRenderer.suggestion"); s.Exists() {
				text := parser.RunsText(s)
				if text != "" {
					suggestions = append(suggestions, models.Suggestion{Text: text})
				}
			}
			if r := item.Get("musicResponsiveListItemRenderer"); r.Exists() {
				if t := parser.TrackFromResponsiveList(r); t != nil {
					tracks = append(tracks, *t)
				}
			}
			return true
		})
		return true
	})
	return suggestions, tracks, nil
}
