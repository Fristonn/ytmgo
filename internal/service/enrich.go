package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/Fristonn/ytmgo/internal/innertube/browse"
	"github.com/Fristonn/ytmgo/internal/innertube/clients"
	"github.com/Fristonn/ytmgo/internal/innertube/search"
	"github.com/Fristonn/ytmgo/internal/merge"
	"github.com/Fristonn/ytmgo/internal/models"
	"github.com/Fristonn/ytmgo/internal/playback"
)

func (s *Service) enrichSuggestions(ctx context.Context, q string) (map[string]any, error) {
	var suggestions []models.Suggestion
	var tracks []models.Track
	var lastErr error
	for _, name := range clients.MetadataSearchOrder {
		if !s.yt.Tracker().Available(name) {
			continue
		}
		p, ok := s.yt.Profile(name)
		if !ok || !p.Has(clients.CapSuggestions) {
			continue
		}
		raw, err := s.yt.SuggestionsAs(ctx, name, q)
		if err != nil {
			lastErr = err
			continue
		}
		sugs, tr, err := search.ParseSuggestions(raw)
		if err != nil {
			lastErr = err
			continue
		}
		suggestions = appendUniqueSuggestions(suggestions, sugs)
		tracks = merge.Tracks(tracks, tr)
		s.yt.Tracker().MarkSuccess(name)
	}
	if len(suggestions) == 0 && len(tracks) == 0 && lastErr != nil {
		return nil, lastErr
	}
	return map[string]any{"suggestions": suggestions, "tracks": tracks}, nil
}

func appendUniqueSuggestions(dst, src []models.Suggestion) []models.Suggestion {
	seen := map[string]struct{}{}
	for _, s := range dst {
		seen[strings.ToLower(s.Text)] = struct{}{}
	}
	out := append([]models.Suggestion(nil), dst...)
	for _, s := range src {
		key := strings.ToLower(s.Text)
		if _, ok := seen[key]; ok || s.Text == "" {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}
	return out
}

func (s *Service) enrichSearch(ctx context.Context, query, filter, continuation string) (*models.SearchResult, error) {
	var out *models.SearchResult
	var lastErr error
	for _, name := range clients.MetadataSearchOrder {
		if !s.yt.Tracker().Available(name) {
			continue
		}
		p, ok := s.yt.Profile(name)
		if !ok || !p.Has(clients.CapSearch) {
			continue
		}
		raw, err := s.yt.SearchAs(ctx, name, query, filter, continuation)
		if err != nil {
			lastErr = err
			continue
		}
		parsed, err := search.ParseMusicSearch(raw, query, filter)
		if err != nil {
			// WEB search shape differs; try video parser for WEB/MWEB.
			if name == "WEB" || name == "MWEB" || name == "ANDROID" {
				if vp, verr := search.ParseVideoSearch(raw, query); verr == nil {
					parsed = vp
					err = nil
				}
			}
			if err != nil {
				lastErr = err
				continue
			}
		}
		out = merge.SearchResult(out, parsed)
		s.yt.Tracker().MarkSuccess(name)
		// Continuations are client-specific; stop after the first successful page.
		if continuation != "" && out != nil {
			break
		}
		if out != nil && !searchIncomplete(out) {
			break
		}
	}
	if out == nil {
		if lastErr != nil {
			return nil, lastErr
		}
		return &models.SearchResult{Query: query, Filter: filter}, nil
	}
	return out, nil
}

func (s *Service) enrichTrack(ctx context.Context, id string) (*models.Track, error) {
	var out models.Track
	out.ID = id
	var lastErr error
	got := false

	for _, name := range clients.MetadataTrackOrder {
		if !s.yt.Tracker().Available(name) {
			continue
		}
		p, ok := s.yt.Profile(name)
		if !ok {
			continue
		}

		if p.Has(clients.CapMusicMetadata) || p.Has(clients.CapBrowse) {
			if raw, err := s.yt.NextAs(ctx, name, id, "", ""); err == nil {
				if sec, err2 := browse.ParseRelated(raw); err2 == nil {
					for _, cand := range sec.Tracks {
						if cand.ID == id {
							out = merge.Track(out, cand)
							got = true
							break
						}
					}
					if !got && len(sec.Tracks) > 0 && sec.Tracks[0].ID == id {
						out = merge.Track(out, sec.Tracks[0])
						got = true
					}
				}
			} else {
				lastErr = err
			}
		}

		if p.Has(clients.CapPlayer) && merge.TrackIncomplete(out) {
			if raw, _, err := s.yt.Player(ctx, name, id, ""); err == nil {
				title, author, _, duration, _, artwork, _, status, _ := playback.ParseFormats(raw)
				if status == "OK" || title != "" {
					t := models.Track{
						ID:       id,
						Title:    title,
						Duration: duration,
						Artwork:  artwork,
					}
					if author != "" {
						t.Artists = []models.ArtistRef{{Name: author}}
					}
					out = merge.Track(out, t)
					got = true
				}
			} else {
				lastErr = err
			}
		}

		if got && !merge.TrackIncomplete(out) {
			break
		}
	}

	if out.Title == "" {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("track not found")
	}
	return &out, nil
}

func (s *Service) enrichArtist(ctx context.Context, id string) (*models.Artist, error) {
	var out *models.Artist
	var lastErr error
	for _, name := range clients.MetadataBrowseOrder {
		if !s.yt.Tracker().Available(name) {
			continue
		}
		p, ok := s.yt.Profile(name)
		if !ok || !p.Has(clients.CapBrowse) {
			continue
		}
		raw, err := s.yt.BrowseAs(ctx, name, id, "", "")
		if err != nil {
			lastErr = err
			continue
		}
		parsed, err := browse.ParseArtist(raw, id)
		if err != nil {
			lastErr = err
			continue
		}
		if out == nil {
			out = parsed
		} else {
			merged := merge.Artist(*out, *parsed)
			out = &merged
		}
		s.yt.Tracker().MarkSuccess(name)
		if out != nil && !merge.ArtistIncomplete(*out) {
			break
		}
	}
	if out == nil {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("artist not found")
	}
	return out, nil
}

func (s *Service) enrichAlbum(ctx context.Context, id string) (*models.Album, error) {
	var out *models.Album
	var lastErr error
	for _, name := range clients.MetadataBrowseOrder {
		if !s.yt.Tracker().Available(name) {
			continue
		}
		p, ok := s.yt.Profile(name)
		if !ok || !p.Has(clients.CapBrowse) {
			continue
		}
		raw, err := s.yt.BrowseAs(ctx, name, id, "", "")
		if err != nil {
			lastErr = err
			continue
		}
		parsed, err := browse.ParseAlbum(raw, id)
		if err != nil {
			lastErr = err
			continue
		}
		if out == nil {
			out = parsed
		} else {
			merged := merge.Album(*out, *parsed)
			out = &merged
		}
		s.yt.Tracker().MarkSuccess(name)
		if out != nil && !merge.AlbumIncomplete(*out) {
			break
		}
	}
	if out == nil {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("album not found")
	}
	// Fill tracks via playlist shelf when still empty.
	if len(out.Tracks) == 0 && out.PlaylistID != "" {
		for _, name := range clients.MetadataBrowseOrder {
			if !s.yt.Tracker().Available(name) {
				continue
			}
			plRaw, err := s.yt.BrowseAs(ctx, name, "VL"+out.PlaylistID, "", "")
			if err != nil {
				continue
			}
			if pl, err := browse.ParsePlaylist(plRaw, out.PlaylistID); err == nil {
				out.Tracks = merge.Tracks(out.Tracks, pl.Tracks)
				out.TrackCount = len(out.Tracks)
				if len(out.Artwork) == 0 {
					out.Artwork = pl.Artwork
				}
				break
			}
		}
	}
	return out, nil
}

func (s *Service) enrichPlaylist(ctx context.Context, id, continuation string) (*models.Playlist, error) {
	if continuation != "" {
		raw, err := s.yt.Browse(ctx, "", "", continuation)
		if err != nil {
			return nil, err
		}
		tracks, cont, err := browse.ParsePlaylistContinuation(raw)
		if err != nil {
			return nil, err
		}
		return &models.Playlist{ID: id, Tracks: tracks, Continuation: cont}, nil
	}

	browseID := id
	if len(id) >= 2 && id[:2] != "VL" {
		browseID = "VL" + id
	}

	var out *models.Playlist
	var lastErr error
	for _, name := range clients.MetadataBrowseOrder {
		if !s.yt.Tracker().Available(name) {
			continue
		}
		p, ok := s.yt.Profile(name)
		if !ok || !p.Has(clients.CapBrowse) {
			continue
		}
		raw, err := s.yt.BrowseAs(ctx, name, browseID, "", "")
		if err != nil {
			lastErr = err
			continue
		}
		parsed, err := browse.ParsePlaylist(raw, id)
		if err != nil {
			lastErr = err
			continue
		}
		if out == nil {
			out = parsed
		} else {
			merged := merge.Playlist(*out, *parsed)
			out = &merged
		}
		s.yt.Tracker().MarkSuccess(name)
		if out != nil && !merge.PlaylistIncomplete(*out) {
			break
		}
	}
	if out == nil {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("playlist not found")
	}
	return out, nil
}

func (s *Service) enrichRelated(ctx context.Context, id string) (*models.Section, error) {
	var out *models.Section
	var lastErr error
	for _, name := range clients.MetadataTrackOrder {
		if !s.yt.Tracker().Available(name) {
			continue
		}
		raw, err := s.yt.NextAs(ctx, name, id, "", "")
		if err != nil {
			lastErr = err
			continue
		}
		sec, err := browse.ParseRelated(raw)
		if err != nil || (sec != nil && len(sec.Tracks) <= 1 && len(sec.Albums) == 0) {
			if browseID := relatedBrowseID(raw); browseID != "" {
				if braw, berr := s.yt.BrowseAs(ctx, name, browseID, "", ""); berr == nil {
					if tab, terr := browse.ParseRelatedTab(braw); terr == nil {
						sec = tab
						err = nil
					}
				}
			}
		}
		if err != nil || sec == nil {
			if err != nil {
				lastErr = err
			}
			continue
		}
		if out == nil {
			cp := *sec
			out = &cp
		} else {
			out.Tracks = merge.Tracks(out.Tracks, sec.Tracks)
			out.Albums = merge.Albums(out.Albums, sec.Albums)
			out.Playlists = merge.Playlists(out.Playlists, sec.Playlists)
			out.Artists = merge.ArtistRefs(out.Artists, sec.Artists)
			out.Title = pickNonEmpty(out.Title, sec.Title)
		}
		s.yt.Tracker().MarkSuccess(name)
		if out != nil && !relatedIncomplete(out) {
			break
		}
	}
	if out == nil {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("no related content found")
	}
	if len(out.Tracks) > 0 && out.Tracks[0].ID == id {
		out.Tracks = out.Tracks[1:]
	}
	return out, nil
}

func pickNonEmpty(dst, src string) string {
	if strings.TrimSpace(dst) == "" {
		return src
	}
	return dst
}

func searchIncomplete(r *models.SearchResult) bool {
	if r == nil {
		return true
	}
	if len(r.Tracks) == 0 && len(r.Albums) == 0 && len(r.Artists) == 0 && len(r.Playlists) == 0 {
		return true
	}
	missing := 0
	n := 0
	for _, t := range r.Tracks {
		if n >= 12 {
			break
		}
		n++
		if merge.TrackIncomplete(t) {
			missing++
		}
	}
	for _, a := range r.Albums {
		if n >= 16 {
			break
		}
		n++
		if len(a.Artwork) == 0 || a.Title == "" {
			missing++
		}
	}
	return missing > 0
}

func relatedIncomplete(s *models.Section) bool {
	if s == nil || len(s.Tracks) < 2 {
		return true
	}
	missingArt := 0
	limit := len(s.Tracks)
	if limit > 10 {
		limit = 10
	}
	for i := 0; i < limit; i++ {
		if len(s.Tracks[i].Artwork) == 0 || s.Tracks[i].Title == "" {
			missingArt++
		}
	}
	return missingArt > 2
}
