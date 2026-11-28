package browse

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/Fristonn/ytmgo/internal/innertube/parser"
	"github.com/Fristonn/ytmgo/internal/models"
)

// ParseAlbum parses a WEB_REMIX album browse response.
func ParseAlbum(raw []byte, browseID string) (*models.Album, error) {
	root := gjson.ParseBytes(raw)

	header := root.Get("contents.twoColumnBrowseResultsRenderer.tabs.0.tabRenderer.content.sectionListRenderer.contents.0.musicResponsiveHeaderRenderer")
	if !header.Exists() {
		// Fallback older layout
		header = root.Get("header.musicDetailHeaderRenderer")
	}
	if !header.Exists() {
		return nil, fmt.Errorf("album header not found")
	}

	title := parser.RunsText(header.Get("title"))
	if title == "" {
		return nil, fmt.Errorf("album title not found")
	}

	artists := parser.ArtistRefsFromRuns(header.Get("straplineTextOne.runs"))
	if len(artists) == 0 {
		artists = parser.ArtistRefsFromRuns(header.Get("subtitle.runs"))
	}

	year := 0
	albumType := ""
	subtitleRuns := header.Get("subtitle.runs").Array()
	if len(subtitleRuns) > 0 {
		albumType = subtitleRuns[0].Get("text").String()
		year, _ = strconv.Atoi(subtitleRuns[len(subtitleRuns)-1].Get("text").String())
	}

	playlistID := ""
	if canon := root.Get("microformat.microformatDataRenderer.urlCanonical").String(); canon != "" {
		if i := strings.LastIndex(canon, "="); i >= 0 {
			playlistID = canon[i+1:]
		}
	}
	if playlistID == "" {
		playlistID = header.Get("buttons.0.musicPlayButtonRenderer.playNavigationEndpoint.watchPlaylistEndpoint.playlistId").String()
	}

	desc := parser.RunsText(header.Get("description.musicDescriptionShelfRenderer.description"))
	if desc == "" {
		desc = parser.RunsText(header.Get("description"))
		// Avoid returning raw nested JSON blobs.
		if strings.HasPrefix(strings.TrimSpace(desc), "{") {
			desc = ""
		}
	}

	album := &models.Album{
		ID:          browseID,
		PlaylistID:  playlistID,
		Title:       title,
		Artists:     artists,
		Year:        year,
		Type:        albumType,
		Artwork:     ResolveArtwork(header.Get("thumbnail"), "album"),
		Description: desc,
	}

	// Tracks live under musicShelfRenderer or musicPlaylistShelfRenderer.
	secondary := root.Get("contents.twoColumnBrowseResultsRenderer.secondaryContents.sectionListRenderer.contents")
	secondary.ForEach(func(_, section gjson.Result) bool {
		shelf := section.Get("musicPlaylistShelfRenderer")
		if !shelf.Exists() {
			shelf = section.Get("musicShelfRenderer")
		}
		if !shelf.Exists() {
			return true
		}
		shelf.Get("contents").ForEach(func(_, item gjson.Result) bool {
			if t := parser.TrackFromResponsiveList(item.Get("musicResponsiveListItemRenderer")); t != nil {
				fillAlbumTrack(t, browseID, title, artists)
				album.Tracks = append(album.Tracks, *t)
			}
			return true
		})
		return true
	})
	if len(album.Tracks) == 0 {
		shelf := root.Get("contents.singleColumnBrowseResultsRenderer.tabs.0.tabRenderer.content.sectionListRenderer.contents.0.musicShelfRenderer")
		shelf.Get("contents").ForEach(func(_, item gjson.Result) bool {
			if t := parser.TrackFromResponsiveList(item.Get("musicResponsiveListItemRenderer")); t != nil {
				fillAlbumTrack(t, browseID, title, artists)
				album.Tracks = append(album.Tracks, *t)
			}
			return true
		})
	}

	var totalDur int
	for _, t := range album.Tracks {
		totalDur += t.Duration
	}
	album.TrackCount = len(album.Tracks)
	album.Duration = totalDur

	return album, nil
}

// fillAlbumTrack sets album ref and inherits album artists when the track row omits them
// (common on album pages where the artist is only in the header).
func fillAlbumTrack(t *models.Track, browseID, title string, artists []models.ArtistRef) {
	if t.Album == nil {
		t.Album = &models.AlbumRef{ID: browseID, Name: title}
	}
	if len(t.Artists) == 0 && len(artists) > 0 {
		t.Artists = append([]models.ArtistRef(nil), artists...)
	}
}
