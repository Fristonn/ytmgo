package browse

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/Fristonn/ytmgo/internal/innertube/parser"
	"github.com/Fristonn/ytmgo/internal/models"
)

// ParsePlaylist parses a WEB_REMIX playlist browse response.
func ParsePlaylist(raw []byte, playlistID string) (*models.Playlist, error) {
	root := gjson.ParseBytes(raw)

	base := root.Get("contents.twoColumnBrowseResultsRenderer.tabs.0.tabRenderer.content.sectionListRenderer.contents.0")
	header := base.Get("musicResponsiveHeaderRenderer")
	if !header.Exists() {
		header = base.Get("musicEditablePlaylistDetailHeaderRenderer.header.musicResponsiveHeaderRenderer")
	}
	if !header.Exists() {
		header = root.Get("header.musicDetailHeaderRenderer")
	}

	title := parser.RunsText(header.Get("title"))
	if title == "" {
		return nil, fmt.Errorf("playlist title not found")
	}

	author := playlistAuthor(header)

	desc := parser.RunsText(header.Get("description.musicDescriptionShelfRenderer.description"))
	if desc == "" {
		desc = parser.RunsText(header.Get("description"))
	}

	pl := &models.Playlist{
		ID:             strings.TrimPrefix(playlistID, "VL"),
		Title:          title,
		Description:    desc,
		Author:         author,
		Artwork:        ResolveArtwork(header.Get("thumbnail"), "playlist"),
		TrackCountText: parser.RunsText(header.Get("secondSubtitle")),
	}

	shelf := root.Get("contents.twoColumnBrowseResultsRenderer.secondaryContents.sectionListRenderer.contents.0.musicPlaylistShelfRenderer")
	if !shelf.Exists() {
		shelf = root.Get("contents.singleColumnBrowseResultsRenderer.tabs.0.tabRenderer.content.sectionListRenderer.contents.0.musicPlaylistShelfRenderer")
	}

	shelf.Get("contents").ForEach(func(_, item gjson.Result) bool {
		if t := parser.TrackFromResponsiveList(item.Get("musicResponsiveListItemRenderer")); t != nil {
			pl.Tracks = append(pl.Tracks, *t)
		}
		return true
	})
	pl.TrackCount = len(pl.Tracks)
	pl.Continuation = parser.ContinuationToken(shelf)

	return pl, nil
}

// playlistAuthor reads the owner from strapline text or the facepile avatar stack.
func playlistAuthor(header gjson.Result) *models.ArtistRef {
	if run := header.Get("straplineTextOne.runs.0"); run.Exists() {
		name := run.Get("text").String()
		if name != "" {
			return &models.ArtistRef{
				Name: name,
				ID:   run.Get("navigationEndpoint.browseEndpoint.browseId").String(),
			}
		}
	}
	face := header.Get("facepile.avatarStackViewModel")
	if !face.Exists() {
		return nil
	}
	name := face.Get("text.content").String()
	if name == "" {
		name = face.Get("rendererContext.accessibilityContext.label").String()
	}
	if name == "" {
		return nil
	}
	id := face.Get("rendererContext.commandContext.onTap.innertubeCommand.browseEndpoint.browseId").String()
	return &models.ArtistRef{Name: name, ID: id}
}

// ParsePlaylistContinuation parses continued playlist tracks.
func ParsePlaylistContinuation(raw []byte) (tracks []models.Track, continuation string, err error) {
	root := gjson.ParseBytes(raw)
	cont := root.Get("continuationContents.musicPlaylistShelfContinuation")
	if !cont.Exists() {
		return nil, "", fmt.Errorf("playlist continuation not found")
	}
	cont.Get("contents").ForEach(func(_, item gjson.Result) bool {
		if t := parser.TrackFromResponsiveList(item.Get("musicResponsiveListItemRenderer")); t != nil {
			tracks = append(tracks, *t)
		}
		return true
	})
	continuation = parser.ContinuationToken(cont)
	return tracks, continuation, nil
}
