package search

import (
	"fmt"

	"github.com/tidwall/gjson"

	"github.com/Fristonn/ytmgo/internal/innertube/parser"
	"github.com/Fristonn/ytmgo/internal/models"
)

// ParseVideoSearch parses a classic WEB YouTube search response (optional path).
func ParseVideoSearch(raw []byte, query string) (*models.SearchResult, error) {
	root := gjson.ParseBytes(raw)
	if !root.IsObject() {
		return nil, fmt.Errorf("invalid video search response")
	}
	result := &models.SearchResult{Query: query, Filter: "videos"}

	contents := root.Get("contents.twoColumnSearchResultsRenderer.primaryContents.sectionListRenderer.contents")
	contents.ForEach(func(_, section gjson.Result) bool {
		section.Get("itemSectionRenderer.contents").ForEach(func(_, item gjson.Result) bool {
			vr := item.Get("videoRenderer")
			if !vr.Exists() {
				return true
			}
			id := vr.Get("videoId").String()
			title := parser.RunsText(vr.Get("title"))
			if id == "" || title == "" {
				return true
			}
			durText := vr.Get("lengthText.simpleText").String()
			var artists []models.ArtistRef
			if ch := vr.Get("ownerText.runs.0"); ch.Exists() {
				artists = append(artists, models.ArtistRef{
					Name: ch.Get("text").String(),
					ID:   ch.Get("navigationEndpoint.browseEndpoint.browseId").String(),
				})
			}
			result.Tracks = append(result.Tracks, models.Track{
				ID:           id,
				Title:        title,
				Artists:      artists,
				Duration:     parser.ParseDuration(durText),
				DurationText: durText,
				Artwork:      parser.Thumbnails(vr.Get("thumbnail")),
				VideoType:    "VIDEO",
			})
			return true
		})
		return true
	})
	return result, nil
}
