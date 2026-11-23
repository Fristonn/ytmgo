package parser

import (
	"strconv"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/Fristonn/ytmgo/internal/models"
)

// TrackFromResponsiveList parses musicResponsiveListItemRenderer into a Track.
func TrackFromResponsiveList(r gjson.Result) *models.Track {
	if !r.Exists() {
		return nil
	}
	id := r.Get("playlistItemData.videoId").String()
	if id == "" {
		id = r.Get("navigationEndpoint.watchEndpoint.videoId").String()
	}
	if id == "" {
		return nil
	}
	title := RunsText(r.Get("flexColumns.0.musicResponsiveListItemFlexColumnRenderer.text"))
	if title == "" {
		return nil
	}

	artists := ArtistRefsFromRuns(r.Get("flexColumns.1.musicResponsiveListItemFlexColumnRenderer.text.runs"))

	var album *models.AlbumRef
	for _, idx := range []int{2, 3} {
		col := r.Get("flexColumns." + strconv.Itoa(idx) + ".musicResponsiveListItemFlexColumnRenderer.text.runs.0")
		if name := col.Get("text").String(); name != "" {
			aid := col.Get("navigationEndpoint.browseEndpoint.browseId").String()
			if strings.HasPrefix(aid, "MPREb_") || aid != "" {
				album = &models.AlbumRef{Name: name, ID: aid}
				break
			}
		}
	}

	durText := RunsText(r.Get("fixedColumns.0.musicResponsiveListItemFlexColumnRenderer.text"))
	arts := PreferMusicArtwork(MusicThumbnail(r.Get("thumbnail")))

	return &models.Track{
		ID:           id,
		Title:        title,
		Artists:      artists,
		Album:        album,
		Duration:     ParseDuration(durText),
		DurationText: durText,
		Artwork:      arts,
		Explicit:     IsExplicit(r.Get("badges")),
	}
}

// TrackFromTwoRow parses musicTwoRowItemRenderer as a song.
func TrackFromTwoRow(r gjson.Result) *models.Track {
	id := r.Get("navigationEndpoint.watchEndpoint.videoId").String()
	if id == "" {
		return nil
	}
	title := RunsText(r.Get("title"))
	if title == "" {
		return nil
	}
	var artists []models.ArtistRef
	if run := r.Get("subtitle.runs.0"); run.Exists() {
		artists = append(artists, models.ArtistRef{
			Name: run.Get("text").String(),
			ID:   run.Get("navigationEndpoint.browseEndpoint.browseId").String(),
		})
	}
	return &models.Track{
		ID:       id,
		Title:    title,
		Artists:  artists,
		Artwork:  PreferMusicArtwork(MusicThumbnail(r.Get("thumbnailRenderer"))),
		Explicit: IsExplicit(r.Get("subtitleBadges")),
	}
}

// AlbumFromTwoRow parses musicTwoRowItemRenderer as an album.
func AlbumFromTwoRow(r gjson.Result) *models.Album {
	browseID := r.Get("navigationEndpoint.browseEndpoint.browseId").String()
	if browseID == "" {
		return nil
	}
	// Albums use MPREb_ browse IDs; playlists use VL / PL.
	pageType := r.Get("navigationEndpoint.browseEndpoint.browseEndpointContextSupportedConfigs.browseEndpointContextMusicConfig.pageType").String()
	if pageType != "" && pageType != "MUSIC_PAGE_TYPE_ALBUM" {
		return nil
	}
	if !strings.HasPrefix(browseID, "MPREb_") && pageType == "" {
		// Heuristic: require album-like id when pageType absent.
		if strings.HasPrefix(browseID, "VL") || strings.HasPrefix(browseID, "PL") || strings.HasPrefix(browseID, "UC") || strings.HasPrefix(browseID, "FEmusic_") {
			return nil
		}
	}

	title := RunsText(r.Get("title"))
	if title == "" {
		return nil
	}
	playlistID := r.Get("thumbnailOverlay.musicItemThumbnailOverlayRenderer.content.musicPlayButtonRenderer.playNavigationEndpoint.watchPlaylistEndpoint.playlistId").String()
	if playlistID == "" {
		playlistID = r.Get("thumbnailOverlay.musicItemThumbnailOverlayRenderer.content.musicPlayButtonRenderer.playNavigationEndpoint.watchEndpoint.playlistId").String()
	}
	year := 0
	subtitle := r.Get("subtitle.runs")
	if subtitle.Exists() {
		last := subtitle.Array()
		if n := len(last); n > 0 {
			year, _ = strconv.Atoi(last[n-1].Get("text").String())
		}
	}
	albumType := ""
	if subtitle.Exists() && len(subtitle.Array()) > 0 {
		albumType = subtitle.Array()[0].Get("text").String()
	}

	return &models.Album{
		ID:         browseID,
		PlaylistID: playlistID,
		Title:      title,
		Year:       year,
		Type:       albumType,
		Artwork:    PreferMusicArtwork(MusicThumbnail(r.Get("thumbnailRenderer"))),
		Explicit:   IsExplicit(r.Get("subtitleBadges")),
	}
}

// PlaylistFromTwoRow parses a playlist two-row item.
func PlaylistFromTwoRow(r gjson.Result) *models.Playlist {
	browseID := r.Get("navigationEndpoint.browseEndpoint.browseId").String()
	pageType := r.Get("navigationEndpoint.browseEndpoint.browseEndpointContextSupportedConfigs.browseEndpointContextMusicConfig.pageType").String()
	if pageType != "" && pageType != "MUSIC_PAGE_TYPE_PLAYLIST" {
		return nil
	}
	if browseID == "" {
		return nil
	}
	id := strings.TrimPrefix(browseID, "VL")
	title := RunsText(r.Get("title"))
	if title == "" {
		return nil
	}
	var author *models.ArtistRef
	runs := r.Get("subtitle.runs").Array()
	if len(runs) > 0 {
		// Often: "Playlist • Author • N songs"
		for _, run := range runs {
			if bid := run.Get("navigationEndpoint.browseEndpoint.browseId").String(); bid != "" {
				author = &models.ArtistRef{Name: run.Get("text").String(), ID: bid}
				break
			}
		}
		if author == nil && len(runs) >= 3 {
			author = &models.ArtistRef{Name: runs[2].Get("text").String()}
		}
	}
	return &models.Playlist{
		ID:      id,
		Title:   title,
		Author:  author,
		Artwork: PreferMusicArtwork(MusicThumbnail(r.Get("thumbnailRenderer"))),
	}
}

// ArtistFromTwoRow parses an artist two-row item.
func ArtistFromTwoRow(r gjson.Result) *models.ArtistRef {
	browseID := r.Get("navigationEndpoint.browseEndpoint.browseId").String()
	pageType := r.Get("navigationEndpoint.browseEndpoint.browseEndpointContextSupportedConfigs.browseEndpointContextMusicConfig.pageType").String()
	if pageType != "" && pageType != "MUSIC_PAGE_TYPE_ARTIST" {
		return nil
	}
	if browseID == "" || (!strings.HasPrefix(browseID, "UC") && pageType == "") {
		return nil
	}
	name := RunsText(r.Get("title"))
	if name == "" {
		return nil
	}
	return &models.ArtistRef{ID: browseID, Name: name}
}

// ArtistFromResponsiveList parses an artist list item.
func ArtistFromResponsiveList(r gjson.Result) *models.ArtistRef {
	browseID := r.Get("navigationEndpoint.browseEndpoint.browseId").String()
	if browseID == "" {
		browseID = r.Get("flexColumns.0.musicResponsiveListItemFlexColumnRenderer.text.runs.0.navigationEndpoint.browseEndpoint.browseId").String()
	}
	if browseID == "" {
		return nil
	}
	name := RunsText(r.Get("flexColumns.0.musicResponsiveListItemFlexColumnRenderer.text"))
	if name == "" {
		return nil
	}
	return &models.ArtistRef{ID: browseID, Name: name}
}

// AlbumFromResponsiveList parses album search results.
func AlbumFromResponsiveList(r gjson.Result) *models.Album {
	browseID := r.Get("navigationEndpoint.browseEndpoint.browseId").String()
	if browseID == "" {
		browseID = r.Get("flexColumns.0.musicResponsiveListItemFlexColumnRenderer.text.runs.0.navigationEndpoint.browseEndpoint.browseId").String()
	}
	if !strings.HasPrefix(browseID, "MPREb_") {
		return nil
	}
	title := RunsText(r.Get("flexColumns.0.musicResponsiveListItemFlexColumnRenderer.text"))
	var artists []models.ArtistRef
	year := 0
	albumType := ""
	runs := r.Get("flexColumns.1.musicResponsiveListItemFlexColumnRenderer.text.runs").Array()
	for i, run := range runs {
		text := run.Get("text").String()
		if text == "" || text == "•" || text == " · " {
			continue
		}
		if bid := run.Get("navigationEndpoint.browseEndpoint.browseId").String(); bid != "" {
			artists = append(artists, models.ArtistRef{Name: text, ID: bid})
			continue
		}
		if y, err := strconv.Atoi(text); err == nil && y > 1900 && y < 2100 {
			year = y
			continue
		}
		if i == 0 && (text == "Album" || text == "EP" || text == "Single") {
			albumType = text
		}
	}
	return &models.Album{
		ID:       browseID,
		Title:    title,
		Artists:  artists,
		Year:     year,
		Type:     albumType,
		Artwork:  PreferMusicArtwork(MusicThumbnail(r.Get("thumbnail"))),
		Explicit: IsExplicit(r.Get("badges")),
	}
}

// PlaylistFromResponsiveList parses playlist search results.
func PlaylistFromResponsiveList(r gjson.Result) *models.Playlist {
	browseID := r.Get("navigationEndpoint.browseEndpoint.browseId").String()
	if browseID == "" {
		browseID = r.Get("flexColumns.0.musicResponsiveListItemFlexColumnRenderer.text.runs.0.navigationEndpoint.browseEndpoint.browseId").String()
	}
	if browseID == "" {
		return nil
	}
	id := strings.TrimPrefix(browseID, "VL")
	title := RunsText(r.Get("flexColumns.0.musicResponsiveListItemFlexColumnRenderer.text"))
	return &models.Playlist{
		ID:      id,
		Title:   title,
		Artwork: PreferMusicArtwork(MusicThumbnail(r.Get("thumbnail"))),
	}
}

// ClassifyTwoRow returns album, playlist, artist, or track.
func ClassifyTwoRow(r gjson.Result) string {
	if r.Get("navigationEndpoint.watchEndpoint.videoId").Exists() {
		return "track"
	}
	pageType := r.Get("navigationEndpoint.browseEndpoint.browseEndpointContextSupportedConfigs.browseEndpointContextMusicConfig.pageType").String()
	switch pageType {
	case "MUSIC_PAGE_TYPE_ALBUM":
		return "album"
	case "MUSIC_PAGE_TYPE_PLAYLIST":
		return "playlist"
	case "MUSIC_PAGE_TYPE_ARTIST":
		return "artist"
	}
	bid := r.Get("navigationEndpoint.browseEndpoint.browseId").String()
	switch {
	case strings.HasPrefix(bid, "MPREb_"):
		return "album"
	case strings.HasPrefix(bid, "VL"), strings.HasPrefix(bid, "PL"):
		return "playlist"
	case strings.HasPrefix(bid, "UC"):
		return "artist"
	}
	return ""
}
