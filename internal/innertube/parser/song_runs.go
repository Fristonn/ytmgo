package parser

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/tidwall/gjson"

	"github.com/Fristonn/ytmgo/internal/models"
)

// SongRuns is the classified metadata from Innertube text runs.
type SongRuns struct {
	Artists      []models.ArtistRef
	Album        *models.AlbumRef
	DurationText string
	Year         string
	Views        string
}

var (
	durationRunRe = regexp.MustCompile(`^(\d+:)*\d+:\d+$`)
	yearRunRe     = regexp.MustCompile(`^\d{4}$`)
	// Leading views word ("조회수 17억회", "播放次數：4505") or bidi mark, up to first digit.
	// Fullwidth colon U+FF1A; bidi marks U+200E–U+200F, U+202A–U+202E.
	viewsPrefixRe = regexp.MustCompile("^[^\x30-\x39]*?[\t :\uFF1A\u200E\u200F\u202A\u202B\u202C\u202D\u202E]")
)

// ParseSongRuns classifies odd-indexed runs into artists / album / duration / year / views.
// Separators (•) sit on even indices and are skipped.
// When skipTypeSpec is true, a leading unlinked type label ("Song", "Video", …) is dropped.
func ParseSongRuns(runs gjson.Result, skipTypeSpec bool) SongRuns {
	arr := runs.Array()
	// Only strip an *unlinked* leading type label ("Song" • …).
	// Linked artists (navigationEndpoint present) must never be skipped, even when
	// the next token is views/plays.
	if skipTypeSpec && len(arr) > 2 {
		first := arr[0]
		if !first.Get("navigationEndpoint").Exists() &&
			classifySongRun(first) == runArtist &&
			isDotSeparator(arr[1]) &&
			isMetaRun(classifySongRun(arr[2])) {
			arr = arr[2:]
		}
	}

	var out SongRuns
	for i, run := range arr {
		if i%2 == 1 {
			continue
		}
		switch kind, data := parseSongRun(run); kind {
		case runAlbum:
			out.Album = data.Album
		case runArtist:
			if data.Artist.Name != "" {
				out.Artists = append(out.Artists, data.Artist)
			}
		case runDuration:
			out.DurationText = data.Text
		case runYear:
			out.Year = data.Text
		case runViews:
			out.Views = data.Text
		}
	}
	return out
}

type runKind int

const (
	runArtist runKind = iota
	runAlbum
	runDuration
	runYear
	runViews
)

type runData struct {
	Artist models.ArtistRef
	Album  *models.AlbumRef
	Text   string
}

func isMetaRun(k runKind) bool {
	return k == runArtist || k == runDuration || k == runViews || k == runYear
}

func isDotSeparator(run gjson.Result) bool {
	text := run.Get("text").String()
	return text == " • " || text == "•" || text == " · " || text == "·"
}

func classifySongRun(run gjson.Result) runKind {
	kind, _ := parseSongRun(run)
	return kind
}

func parseSongRun(run gjson.Result) (runKind, runData) {
	text := run.Get("text").String()
	if text == "" {
		return runArtist, runData{}
	}

	if run.Get("navigationEndpoint").Exists() {
		id := run.Get("navigationEndpoint.browseEndpoint.browseId").String()
		if strings.HasPrefix(id, "MPRE") || strings.Contains(id, "release_detail") {
			return runAlbum, runData{Album: &models.AlbumRef{Name: text, ID: id}}
		}
		return runArtist, runData{Artist: models.ArtistRef{Name: text, ID: id}}
	}

	if durationRunRe.MatchString(text) {
		return runDuration, runData{Text: text}
	}
	if yearRunRe.MatchString(text) {
		return runYear, runData{Text: text}
	}
	if views := parseViews(text); views != "" {
		return runViews, runData{Text: views}
	}
	return runArtist, runData{Artist: models.ArtistRef{Name: text}}
}

// parseViews keeps view-count tokens and rejects bare artist names.
func parseViews(text string) string {
	prefixed := false
	if !containsLatinLetter(text) {
		replaced := viewsPrefixRe.ReplaceAllString(text, "")
		if replaced != text {
			prefixed = true
			text = replaced
		}
	}
	if text == "" || !unicode.IsDigit(rune(text[0])) {
		return ""
	}
	// Bare ASCII token like "2Pac" is an artist, not a view count.
	if !prefixed && isASCII(text) && !strings.Contains(text, " ") {
		return ""
	}
	// "1B views", "3.7B plays", "88 k vues", etc.
	head := strings.Split(text, " ")[0]
	parts := strings.Split(head, "\u00a0")
	if len(parts) > 2 {
		parts = parts[:2]
	}
	return strings.Join(parts, "\u00a0")
}

func containsLatinLetter(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

// FlexColumn returns musicResponsiveListItemFlexColumnRenderer at index, or empty.
func FlexColumn(item gjson.Result, index int) gjson.Result {
	col := item.Get("flexColumns." + strconv.Itoa(index) + ".musicResponsiveListItemFlexColumnRenderer")
	if !col.Get("text.runs").Exists() {
		return gjson.Result{}
	}
	return col
}

// FixedColumn returns musicResponsiveListItemFixedColumnRenderer at index, or empty.
func FixedColumn(item gjson.Result, index int) gjson.Result {
	col := item.Get("fixedColumns." + strconv.Itoa(index) + ".musicResponsiveListItemFixedColumnRenderer")
	if !col.Exists() {
		// Older mistaken key fallback.
		col = item.Get("fixedColumns." + strconv.Itoa(index) + ".musicResponsiveListItemFlexColumnRenderer")
	}
	if !col.Get("text").Exists() {
		return gjson.Result{}
	}
	return col
}
