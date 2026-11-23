package search

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/Fristonn/ytmgo/internal/innertube/parser"
	"github.com/Fristonn/ytmgo/internal/models"
)

// ParseMusicSearch parses a WEB_REMIX search response.
func ParseMusicSearch(raw []byte, query, filter string) (*models.SearchResult, error) {
	root := gjson.ParseBytes(raw)
	if !root.IsObject() {
		return nil, fmt.Errorf("invalid search response")
	}

	result := &models.SearchResult{Query: query, Filter: filter}

	// Continuation page
	if cont := root.Get("continuationContents.musicShelfContinuation"); cont.Exists() {
		parseShelfContents(cont.Get("contents"), result)
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
			parseShelfContents(shelf.Get("contents"), result)
			if c := parser.ContinuationToken(shelf); c != "" {
				result.Continuation = c
			}
		}
		if card := section.Get("musicCardShelfRenderer"); card.Exists() {
			if t := parser.TrackFromResponsiveList(card.Get("contents.0.musicResponsiveListItemRenderer")); t != nil {
				result.Tracks = append(result.Tracks, *t)
			}
			// Top result may itself be a track via onTap
			if vid := card.Get("onTap.watchEndpoint.videoId").String(); vid != "" {
				title := parser.RunsText(card.Get("title"))
				result.Tracks = append(result.Tracks, models.Track{
					ID:      vid,
					Title:   title,
					Artwork: parser.PreferMusicArtwork(parser.MusicThumbnail(card.Get("thumbnail"))),
				})
			}
		}
		return true
	})

	return result, nil
}

func parseShelfContents(contents gjson.Result, result *models.SearchResult) {
	contents.ForEach(func(_, item gjson.Result) bool {
		r := item.Get("musicResponsiveListItemRenderer")
		if !r.Exists() {
			return true
		}
		pageType := r.Get("navigationEndpoint.browseEndpoint.browseEndpointContextSupportedConfigs.browseEndpointContextMusicConfig.pageType").String()
		switch {
		case r.Get("playlistItemData.videoId").Exists() || r.Get("navigationEndpoint.watchEndpoint.videoId").Exists():
			if t := parser.TrackFromResponsiveList(r); t != nil {
				result.Tracks = append(result.Tracks, *t)
			}
		case pageType == "MUSIC_PAGE_TYPE_ARTIST" || strings.HasPrefix(r.Get("navigationEndpoint.browseEndpoint.browseId").String(), "UC"):
			if a := parser.ArtistFromResponsiveList(r); a != nil {
				result.Artists = append(result.Artists, *a)
			}
		case pageType == "MUSIC_PAGE_TYPE_ALBUM" || strings.HasPrefix(r.Get("navigationEndpoint.browseEndpoint.browseId").String(), "MPREb_"):
			if a := parser.AlbumFromResponsiveList(r); a != nil {
				result.Albums = append(result.Albums, *a)
			}
		case pageType == "MUSIC_PAGE_TYPE_PLAYLIST":
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
