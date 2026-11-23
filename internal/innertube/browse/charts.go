package browse

import (
	"github.com/tidwall/gjson"

	"github.com/Fristonn/ytmgo/internal/innertube/parser"
	"github.com/Fristonn/ytmgo/internal/models"
)

// ParseCharts parses FEmusic_charts browse response.
func ParseCharts(raw []byte) ([]models.ChartSection, error) {
	root := gjson.ParseBytes(raw)
	var sections []models.ChartSection

	contents := root.Get("contents.singleColumnBrowseResultsRenderer.tabs.0.tabRenderer.content.sectionListRenderer.contents")
	contents.ForEach(func(_, section gjson.Result) bool {
		if carousel := section.Get("musicCarouselShelfRenderer"); carousel.Exists() {
			cs := models.ChartSection{
				Title: parser.RunsText(carousel.Get("header.musicCarouselShelfBasicHeaderRenderer.title")),
			}
			carousel.Get("contents").ForEach(func(_, item gjson.Result) bool {
				if r := item.Get("musicTwoRowItemRenderer"); r.Exists() {
					switch parser.ClassifyTwoRow(r) {
					case "track":
						if t := parser.TrackFromTwoRow(r); t != nil {
							cs.Tracks = append(cs.Tracks, *t)
						}
					case "album":
						if a := parser.AlbumFromTwoRow(r); a != nil {
							cs.Albums = append(cs.Albums, *a)
						}
					case "playlist":
						if p := parser.PlaylistFromTwoRow(r); p != nil {
							cs.Playlists = append(cs.Playlists, *p)
						}
					case "artist":
						if a := parser.ArtistFromTwoRow(r); a != nil {
							cs.Artists = append(cs.Artists, *a)
						}
					}
					return true
				}
				if r := item.Get("musicResponsiveListItemRenderer"); r.Exists() {
					if t := parser.TrackFromResponsiveList(r); t != nil {
						cs.Tracks = append(cs.Tracks, *t)
						return true
					}
					if a := parser.ArtistFromResponsiveList(r); a != nil {
						cs.Artists = append(cs.Artists, *a)
					}
				}
				return true
			})
			if cs.Title != "" || len(cs.Tracks) > 0 || len(cs.Albums) > 0 || len(cs.Playlists) > 0 || len(cs.Artists) > 0 {
				sections = append(sections, cs)
			}
		}
		if shelf := section.Get("musicShelfRenderer"); shelf.Exists() {
			cs := models.ChartSection{Title: parser.RunsText(shelf.Get("title"))}
			shelf.Get("contents").ForEach(func(_, item gjson.Result) bool {
				if t := parser.TrackFromResponsiveList(item.Get("musicResponsiveListItemRenderer")); t != nil {
					cs.Tracks = append(cs.Tracks, *t)
				}
				return true
			})
			if cs.Title != "" || len(cs.Tracks) > 0 {
				sections = append(sections, cs)
			}
		}
		return true
	})

	return sections, nil
}

// ParseMoods parses FEmusic_moods_and_genres.
func ParseMoods(raw []byte) ([]models.MoodGenre, error) {
	root := gjson.ParseBytes(raw)
	var out []models.MoodGenre

	contents := root.Get("contents.singleColumnBrowseResultsRenderer.tabs.0.tabRenderer.content.sectionListRenderer.contents")
	contents.ForEach(func(_, section gjson.Result) bool {
		grid := section.Get("gridRenderer")
		if !grid.Exists() {
			return true
		}
		mg := models.MoodGenre{
			Title: parser.RunsText(grid.Get("header.gridHeaderRenderer.title")),
		}
		grid.Get("items").ForEach(func(_, item gjson.Result) bool {
			btn := item.Get("musicNavigationButtonRenderer")
			if !btn.Exists() {
				return true
			}
			title := parser.RunsText(btn.Get("buttonText"))
			ep := btn.Get("clickCommand.browseEndpoint")
			color := ""
			if c := btn.Get("solid.leftStripeColor"); c.Exists() {
				color = c.Raw
			}
			if title != "" {
				mg.Items = append(mg.Items, models.MoodGenreItem{
					Title:    title,
					BrowseID: ep.Get("browseId").String(),
					Params:   ep.Get("params").String(),
					Color:    color,
				})
			}
			return true
		})
		if len(mg.Items) > 0 {
			out = append(out, mg)
		}
		return true
	})
	return out, nil
}
