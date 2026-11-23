package browse

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/Fristonn/ytmgo/internal/innertube/parser"
	"github.com/Fristonn/ytmgo/internal/models"
)

// ParseArtist parses a WEB_REMIX artist browse response.
func ParseArtist(raw []byte, browseID string) (*models.Artist, error) {
	root := gjson.ParseBytes(raw)
	header := root.Get("header")

	name := parser.RunsText(header.Get("musicImmersiveHeaderRenderer.title"))
	if name == "" {
		name = parser.RunsText(header.Get("musicVisualHeaderRenderer.title"))
	}
	if name == "" {
		return nil, fmt.Errorf("artist name not found")
	}

	desc := parser.RunsText(header.Get("musicImmersiveHeaderRenderer.description"))
	subs := parser.RunsText(header.Get("musicImmersiveHeaderRenderer.subscriptionButton.subscribeButtonRenderer.subscriberCountText"))
	if subs == "" {
		subs = header.Get("musicImmersiveHeaderRenderer.monthlyListenerCount.runs.0.text").String()
	}

	artist := &models.Artist{
		ID:              browseID,
		Name:            name,
		Description:     desc,
		SubscriberCount: subs,
		Avatar:          AvatarFromHeader(header),
		Banner:          BannerFromHeader(header),
	}

	// Prefer visual-header banner art
	if visual := header.Get("musicVisualHeaderRenderer"); visual.Exists() {
		if arts := parser.Thumbnails(visual.Get("thumbnail.musicThumbnailRenderer.thumbnail")); len(arts) > 0 {
			artist.Banner = arts
		}
	}

	contents := root.Get("contents.singleColumnBrowseResultsRenderer.tabs.0.tabRenderer.content.sectionListRenderer.contents")
	contents.ForEach(func(_, section gjson.Result) bool {
		sec := parseArtistSection(section)
		if sec == nil {
			return true
		}
		artist.Sections = append(artist.Sections, *sec)
		titleLower := strings.ToLower(sec.Title)
		switch {
		case strings.Contains(titleLower, "song") || strings.Contains(titleLower, "top"):
			artist.TopSongs = append(artist.TopSongs, sec.Tracks...)
		case strings.Contains(titleLower, "album"):
			artist.Albums = append(artist.Albums, sec.Albums...)
		case strings.Contains(titleLower, "single") || strings.Contains(titleLower, "ep"):
			artist.Singles = append(artist.Singles, sec.Albums...)
		case strings.Contains(titleLower, "video"):
			artist.Videos = append(artist.Videos, sec.Tracks...)
		case strings.Contains(titleLower, "featuring") || strings.Contains(titleLower, "featured"):
			artist.Featured = append(artist.Featured, sec.Albums...)
		case strings.Contains(titleLower, "fans") || strings.Contains(titleLower, "similar") || strings.Contains(titleLower, "related"):
			artist.Related = append(artist.Related, sec.Artists...)
		}
		return true
	})

	return artist, nil
}

func parseArtistSection(section gjson.Result) *models.Section {
	if shelf := section.Get("musicShelfRenderer"); shelf.Exists() {
		sec := &models.Section{
			Title: parser.RunsText(shelf.Get("title")),
		}
		if ep := shelf.Get("title.runs.0.navigationEndpoint.browseEndpoint"); ep.Exists() {
			sec.BrowseID = ep.Get("browseId").String()
			sec.Params = ep.Get("params").String()
		}
		shelf.Get("contents").ForEach(func(_, item gjson.Result) bool {
			if t := parser.TrackFromResponsiveList(item.Get("musicResponsiveListItemRenderer")); t != nil {
				sec.Tracks = append(sec.Tracks, *t)
			}
			return true
		})
		if len(sec.Tracks) == 0 && sec.Title == "" {
			return nil
		}
		return sec
	}

	if carousel := section.Get("musicCarouselShelfRenderer"); carousel.Exists() {
		sec := &models.Section{
			Title: parser.RunsText(carousel.Get("header.musicCarouselShelfBasicHeaderRenderer.title")),
		}
		if ep := carousel.Get("header.musicCarouselShelfBasicHeaderRenderer.moreContentButton.buttonRenderer.navigationEndpoint.browseEndpoint"); ep.Exists() {
			sec.BrowseID = ep.Get("browseId").String()
			sec.Params = ep.Get("params").String()
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
			case "playlist":
				if p := parser.PlaylistFromTwoRow(r); p != nil {
					sec.Playlists = append(sec.Playlists, *p)
				}
			case "artist":
				if a := parser.ArtistFromTwoRow(r); a != nil {
					sec.Artists = append(sec.Artists, *a)
				}
			}
			return true
		})
		return sec
	}
	return nil
}

// ParseArtistItems parses artist discography / songs list pages.
func ParseArtistItems(raw []byte) (title string, albums []models.Album, tracks []models.Track, continuation string, err error) {
	root := gjson.ParseBytes(raw)

	if grid := root.Get("contents.singleColumnBrowseResultsRenderer.tabs.0.tabRenderer.content.sectionListRenderer.contents.0.gridRenderer"); grid.Exists() {
		title = parser.RunsText(grid.Get("header.gridHeaderRenderer.title"))
		grid.Get("items").ForEach(func(_, item gjson.Result) bool {
			if a := parser.AlbumFromTwoRow(item.Get("musicTwoRowItemRenderer")); a != nil {
				albums = append(albums, *a)
			}
			return true
		})
		continuation = parser.ContinuationToken(grid)
		return
	}

	if cont := root.Get("continuationContents.gridContinuation"); cont.Exists() {
		cont.Get("items").ForEach(func(_, item gjson.Result) bool {
			if a := parser.AlbumFromTwoRow(item.Get("musicTwoRowItemRenderer")); a != nil {
				albums = append(albums, *a)
			}
			return true
		})
		continuation = parser.ContinuationToken(cont)
		return
	}

	shelfPath := "contents.singleColumnBrowseResultsRenderer.tabs.0.tabRenderer.content.sectionListRenderer.contents.0.musicPlaylistShelfRenderer"
	if shelf := root.Get(shelfPath); shelf.Exists() {
		title = parser.RunsText(root.Get("header.musicHeaderRenderer.title"))
		shelf.Get("contents").ForEach(func(_, item gjson.Result) bool {
			if t := parser.TrackFromResponsiveList(item.Get("musicResponsiveListItemRenderer")); t != nil {
				tracks = append(tracks, *t)
			}
			return true
		})
		continuation = parser.ContinuationToken(shelf)
		return
	}

	if cont := root.Get("continuationContents.musicPlaylistShelfContinuation"); cont.Exists() {
		cont.Get("contents").ForEach(func(_, item gjson.Result) bool {
			if t := parser.TrackFromResponsiveList(item.Get("musicResponsiveListItemRenderer")); t != nil {
				tracks = append(tracks, *t)
			}
			return true
		})
		continuation = parser.ContinuationToken(cont)
		return
	}

	return
}
