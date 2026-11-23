package merge

import (
	"strings"

	"github.com/Fristonn/ytmgo/internal/models"
)

// Artwork merges variants, preferring larger and nearer-square images.
func Artwork(dst, src []models.Artwork) []models.Artwork {
	if len(src) == 0 {
		return dst
	}
	if len(dst) == 0 {
		return append([]models.Artwork(nil), src...)
	}
	seen := make(map[string]int, len(dst)+len(src))
	out := append([]models.Artwork(nil), dst...)
	for i, a := range out {
		seen[normalizeURL(a.URL)] = i
	}
	for _, a := range src {
		key := normalizeURL(a.URL)
		if i, ok := seen[key]; ok {
			if artworkScore(a) > artworkScore(out[i]) {
				out[i] = a
			}
			continue
		}
		seen[key] = len(out)
		out = append(out, a)
	}
	return out
}

func artworkScore(a models.Artwork) int {
	area := a.Width * a.Height
	if a.Width > 0 && a.Height > 0 {
		diff := a.Width - a.Height
		if diff < 0 {
			diff = -diff
		}
		// Prefer square-ish artwork for music entities.
		if diff*10 <= a.Width+a.Height {
			area *= 2
		}
	}
	if strings.Contains(a.URL, "googleusercontent") || strings.Contains(a.URL, "ggpht.com") {
		area += 1000
	}
	return area
}

func normalizeURL(u string) string {
	if i := strings.Index(u, "="); i > 0 && (strings.Contains(u, "googleusercontent") || strings.Contains(u, "ggpht")) {
		return u[:i]
	}
	return u
}

func pickStr(dst, src string) string {
	if strings.TrimSpace(dst) == "" {
		return src
	}
	if len(src) > len(dst)*2 && len(src) > len(dst)+40 {
		return src // richer description / text
	}
	return dst
}

func pickInt(dst, src int) int {
	if dst == 0 {
		return src
	}
	return dst
}

func pickBool(dst, src *bool) *bool {
	if dst == nil {
		return src
	}
	return dst
}

// ArtistRefs merges by ID (or name when ID missing).
func ArtistRefs(dst, src []models.ArtistRef) []models.ArtistRef {
	if len(src) == 0 {
		return dst
	}
	if len(dst) == 0 {
		return append([]models.ArtistRef(nil), src...)
	}
	idx := map[string]int{}
	out := append([]models.ArtistRef(nil), dst...)
	for i, a := range out {
		key := a.ID
		if key == "" {
			key = "n:" + strings.ToLower(a.Name)
		}
		idx[key] = i
	}
	for _, a := range src {
		key := a.ID
		if key == "" {
			key = "n:" + strings.ToLower(a.Name)
		}
		if i, ok := idx[key]; ok {
			if out[i].ID == "" {
				out[i].ID = a.ID
			}
			if out[i].Name == "" {
				out[i].Name = a.Name
			}
			continue
		}
		idx[key] = len(out)
		out = append(out, a)
	}
	return out
}

// Track fills missing fields from src matched by ID.
func Track(dst, src models.Track) models.Track {
	if dst.ID == "" {
		dst.ID = src.ID
	}
	dst.Title = pickStr(dst.Title, src.Title)
	dst.DurationText = pickStr(dst.DurationText, src.DurationText)
	dst.VideoType = pickStr(dst.VideoType, src.VideoType)
	dst.Duration = pickInt(dst.Duration, src.Duration)
	dst.Explicit = pickBool(dst.Explicit, src.Explicit)
	dst.IsAvailable = pickBool(dst.IsAvailable, src.IsAvailable)
	dst.Artists = ArtistRefs(dst.Artists, src.Artists)
	dst.Artwork = Artwork(dst.Artwork, src.Artwork)
	if dst.Album == nil {
		dst.Album = src.Album
	} else if src.Album != nil {
		if dst.Album.ID == "" {
			dst.Album.ID = src.Album.ID
		}
		if dst.Album.Name == "" {
			dst.Album.Name = src.Album.Name
		}
	}
	return dst
}

// Tracks merges lists by video/track ID.
func Tracks(dst, src []models.Track) []models.Track {
	if len(src) == 0 {
		return dst
	}
	if len(dst) == 0 {
		return append([]models.Track(nil), src...)
	}
	idx := map[string]int{}
	out := append([]models.Track(nil), dst...)
	for i, t := range out {
		if t.ID != "" {
			idx[t.ID] = i
		}
	}
	for _, t := range src {
		if t.ID == "" {
			continue
		}
		if i, ok := idx[t.ID]; ok {
			out[i] = Track(out[i], t)
			continue
		}
		idx[t.ID] = len(out)
		out = append(out, t)
	}
	return out
}

// Album merges album metadata and nested tracks by ID.
func Album(dst, src models.Album) models.Album {
	if dst.ID == "" {
		dst.ID = src.ID
	}
	dst.PlaylistID = pickStr(dst.PlaylistID, src.PlaylistID)
	dst.Title = pickStr(dst.Title, src.Title)
	dst.Type = pickStr(dst.Type, src.Type)
	dst.Description = pickStr(dst.Description, src.Description)
	dst.Year = pickInt(dst.Year, src.Year)
	dst.TrackCount = pickInt(dst.TrackCount, src.TrackCount)
	dst.Duration = pickInt(dst.Duration, src.Duration)
	dst.Explicit = pickBool(dst.Explicit, src.Explicit)
	dst.Artists = ArtistRefs(dst.Artists, src.Artists)
	dst.Artwork = Artwork(dst.Artwork, src.Artwork)
	dst.Tracks = Tracks(dst.Tracks, src.Tracks)
	if dst.TrackCount == 0 {
		dst.TrackCount = len(dst.Tracks)
	}
	return dst
}

// Albums merges album lists by browse/album ID.
func Albums(dst, src []models.Album) []models.Album {
	if len(src) == 0 {
		return dst
	}
	if len(dst) == 0 {
		return append([]models.Album(nil), src...)
	}
	idx := map[string]int{}
	out := append([]models.Album(nil), dst...)
	for i, a := range out {
		if a.ID != "" {
			idx[a.ID] = i
		}
	}
	for _, a := range src {
		if a.ID == "" {
			continue
		}
		if i, ok := idx[a.ID]; ok {
			out[i] = Album(out[i], a)
			continue
		}
		idx[a.ID] = len(out)
		out = append(out, a)
	}
	return out
}

// Playlist merges playlist fields and tracks.
func Playlist(dst, src models.Playlist) models.Playlist {
	if dst.ID == "" {
		dst.ID = src.ID
	}
	dst.Title = pickStr(dst.Title, src.Title)
	dst.Description = pickStr(dst.Description, src.Description)
	dst.TrackCountText = pickStr(dst.TrackCountText, src.TrackCountText)
	dst.Continuation = pickStr(dst.Continuation, src.Continuation)
	dst.TrackCount = pickInt(dst.TrackCount, src.TrackCount)
	dst.Artwork = Artwork(dst.Artwork, src.Artwork)
	dst.Tracks = Tracks(dst.Tracks, src.Tracks)
	if dst.Author == nil {
		dst.Author = src.Author
	} else if src.Author != nil {
		if dst.Author.ID == "" {
			dst.Author.ID = src.Author.ID
		}
		if dst.Author.Name == "" {
			dst.Author.Name = src.Author.Name
		}
	}
	if dst.TrackCount == 0 {
		dst.TrackCount = len(dst.Tracks)
	}
	return dst
}

// Playlists merges playlist lists by ID.
func Playlists(dst, src []models.Playlist) []models.Playlist {
	if len(src) == 0 {
		return dst
	}
	if len(dst) == 0 {
		return append([]models.Playlist(nil), src...)
	}
	idx := map[string]int{}
	out := append([]models.Playlist(nil), dst...)
	for i, p := range out {
		if p.ID != "" {
			idx[p.ID] = i
		}
	}
	for _, p := range src {
		if p.ID == "" {
			continue
		}
		if i, ok := idx[p.ID]; ok {
			out[i] = Playlist(out[i], p)
			continue
		}
		idx[p.ID] = len(out)
		out = append(out, p)
	}
	return out
}

// Artist merges full artist profiles.
func Artist(dst, src models.Artist) models.Artist {
	if dst.ID == "" {
		dst.ID = src.ID
	}
	dst.Name = pickStr(dst.Name, src.Name)
	dst.Description = pickStr(dst.Description, src.Description)
	dst.SubscriberCount = pickStr(dst.SubscriberCount, src.SubscriberCount)
	dst.Avatar = Artwork(dst.Avatar, src.Avatar)
	dst.Banner = Artwork(dst.Banner, src.Banner)
	dst.TopSongs = Tracks(dst.TopSongs, src.TopSongs)
	dst.Albums = Albums(dst.Albums, src.Albums)
	dst.Singles = Albums(dst.Singles, src.Singles)
	dst.Videos = Tracks(dst.Videos, src.Videos)
	dst.Featured = Albums(dst.Featured, src.Featured)
	dst.Related = ArtistRefs(dst.Related, src.Related)
	if len(dst.Sections) == 0 {
		dst.Sections = src.Sections
	} else if len(src.Sections) > 0 {
		dst.Sections = Sections(dst.Sections, src.Sections)
	}
	return dst
}

// Sections merges shelves by title.
func Sections(dst, src []models.Section) []models.Section {
	if len(src) == 0 {
		return dst
	}
	if len(dst) == 0 {
		return append([]models.Section(nil), src...)
	}
	idx := map[string]int{}
	out := append([]models.Section(nil), dst...)
	for i, s := range out {
		idx[strings.ToLower(s.Title)] = i
	}
	for _, s := range src {
		key := strings.ToLower(s.Title)
		if i, ok := idx[key]; ok {
			out[i].BrowseID = pickStr(out[i].BrowseID, s.BrowseID)
			out[i].Params = pickStr(out[i].Params, s.Params)
			out[i].Tracks = Tracks(out[i].Tracks, s.Tracks)
			out[i].Albums = Albums(out[i].Albums, s.Albums)
			out[i].Playlists = Playlists(out[i].Playlists, s.Playlists)
			out[i].Artists = ArtistRefs(out[i].Artists, s.Artists)
			continue
		}
		idx[key] = len(out)
		out = append(out, s)
	}
	return out
}

// SearchResult merges search hits across clients by entity ID.
func SearchResult(dst, src *models.SearchResult) *models.SearchResult {
	if dst == nil {
		return src
	}
	if src == nil {
		return dst
	}
	dst.Query = pickStr(dst.Query, src.Query)
	dst.Filter = pickStr(dst.Filter, src.Filter)
	dst.Continuation = pickStr(dst.Continuation, src.Continuation)
	dst.Tracks = Tracks(dst.Tracks, src.Tracks)
	dst.Albums = Albums(dst.Albums, src.Albums)
	dst.Artists = ArtistRefs(dst.Artists, src.Artists)
	dst.Playlists = Playlists(dst.Playlists, src.Playlists)
	return dst
}

// TrackIncomplete reports whether enrichment is still useful.
func TrackIncomplete(t models.Track) bool {
	return t.Title == "" || len(t.Artwork) == 0 || len(t.Artists) == 0 || t.Duration == 0
}

// ArtistIncomplete reports whether enrichment is still useful.
func ArtistIncomplete(a models.Artist) bool {
	return a.Name == "" || len(a.Avatar) == 0 || (a.Description == "" && len(a.TopSongs) == 0 && len(a.Albums) == 0)
}

// AlbumIncomplete reports whether enrichment is still useful.
func AlbumIncomplete(a models.Album) bool {
	return a.Title == "" || len(a.Artwork) == 0 || len(a.Tracks) == 0 || len(a.Artists) == 0
}

// PlaylistIncomplete reports whether enrichment is still useful.
func PlaylistIncomplete(p models.Playlist) bool {
	return p.Title == "" || len(p.Artwork) == 0 || len(p.Tracks) == 0
}
