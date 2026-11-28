package parser

import (
	"strconv"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/Fristonn/ytmgo/internal/models"
)

// TrackFromResponsiveList parses musicResponsiveListItemRenderer into a Track.
// Flex column 1 (and optionally 2) hold artists / album / duration / views runs;
// fixed column 0 often holds duration for playlists.
func TrackFromResponsiveList(r gjson.Result) *models.Track {
	if !r.Exists() {
		return nil
	}
	id := r.Get("playlistItemData.videoId").String()
	if id == "" {
		id = r.Get("navigationEndpoint.watchEndpoint.videoId").String()
	}
	if id == "" {
		id = r.Get("overlay.musicItemThumbnailOverlayRenderer.content.musicPlayButtonRenderer.playNavigationEndpoint.watchEndpoint.videoId").String()
	}
	if id == "" {
		return nil
	}
	titleCol := FlexColumn(r, 0)
	title := RunsText(titleCol.Get("text"))
	if title == "" || title == "Song deleted" {
		return nil
	}

	// Collect metadata runs from flex columns 1 and 2 (plays may sit in col 2).
	var runObjs []gjson.Result
	if col1 := FlexColumn(r, 1); col1.Exists() {
		runObjs = append(runObjs, col1.Get("text.runs").Array()...)
	}
	if col2 := FlexColumn(r, 2); col2.Exists() {
		if len(runObjs) > 0 {
			runObjs = append(runObjs, gjson.Parse(`{"text":" • "}`))
		}
		runObjs = append(runObjs, col2.Get("text.runs").Array()...)
	}
	meta := SongRuns{}
	if len(runObjs) > 0 {
		// Rebuild a runs array for ParseSongRuns.
		parts := make([]string, 0, len(runObjs))
		for _, run := range runObjs {
			parts = append(parts, run.Raw)
		}
		meta = ParseSongRuns(gjson.Parse("["+strings.Join(parts, ",")+"]"), true)
	}

	// Dedicated album flex column (playlist / album rows).
	if meta.Album == nil {
		for _, idx := range []int{2, 3} {
			col := FlexColumn(r, idx)
			if !col.Exists() {
				continue
			}
			run := col.Get("text.runs.0")
			name := run.Get("text").String()
			aid := run.Get("navigationEndpoint.browseEndpoint.browseId").String()
			if name != "" && (strings.HasPrefix(aid, "MPRE") || strings.Contains(aid, "release_detail")) {
				meta.Album = &models.AlbumRef{Name: name, ID: aid}
				break
			}
		}
	}

	durText := meta.DurationText
	if fixed := FixedColumn(r, 0); fixed.Exists() {
		if t := RunsText(fixed.Get("text")); t != "" {
			durText = t
		}
	}

	return &models.Track{
		ID:           id,
		Title:        title,
		Artists:      meta.Artists,
		Album:        meta.Album,
		Duration:     ParseDuration(durText),
		DurationText: durText,
		Artwork:      PreferMusicArtwork(MusicThumbnail(r.Get("thumbnail"))),
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
	meta := ParseSongRuns(r.Get("subtitle.runs"), true)
	return &models.Track{
		ID:           id,
		Title:        title,
		Artists:      meta.Artists,
		Album:        meta.Album,
		Duration:     ParseDuration(meta.DurationText),
		DurationText: meta.DurationText,
		Artwork:      PreferMusicArtwork(MusicThumbnail(r.Get("thumbnailRenderer"))),
		Explicit:     IsExplicit(r.Get("subtitleBadges")),
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
	meta := ParseSongRuns(r.Get("subtitle.runs"), true)
	year := 0
	if meta.Year != "" {
		year, _ = strconv.Atoi(meta.Year)
	}
	albumType := ""
	if runs := r.Get("subtitle.runs").Array(); len(runs) > 0 {
		first := runs[0].Get("text").String()
		if first == "Album" || first == "EP" || first == "Single" || first == "Audiobook" {
			albumType = first
		}
	}

	return &models.Album{
		ID:         browseID,
		PlaylistID: playlistID,
		Title:      title,
		Artists:    meta.Artists,
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
	if !strings.HasPrefix(browseID, "MPRE") {
		return nil
	}
	title := RunsText(r.Get("flexColumns.0.musicResponsiveListItemFlexColumnRenderer.text"))
	if title == "" {
		return nil
	}
	meta := ParseSongRuns(r.Get("flexColumns.1.musicResponsiveListItemFlexColumnRenderer.text.runs"), true)
	albumType := ""
	if runs := r.Get("flexColumns.1.musicResponsiveListItemFlexColumnRenderer.text.runs").Array(); len(runs) > 0 {
		first := runs[0].Get("text").String()
		if first == "Album" || first == "EP" || first == "Single" || first == "Audiobook" {
			albumType = first
		}
	}
	year := 0
	if meta.Year != "" {
		year, _ = strconv.Atoi(meta.Year)
	}
	return &models.Album{
		ID:       browseID,
		Title:    title,
		Artists:  meta.Artists,
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
	if title == "" {
		return nil
	}
	var author *models.ArtistRef
	meta := ParseSongRuns(r.Get("flexColumns.1.musicResponsiveListItemFlexColumnRenderer.text.runs"), true)
	if len(meta.Artists) > 0 {
		a := meta.Artists[0]
		author = &a
	}
	return &models.Playlist{
		ID:      id,
		Title:   title,
		Author:  author,
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
