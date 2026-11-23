package models

// Artwork is a single image variant with optional dimensions.
type Artwork struct {
	URL    string `json:"url"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// ArtistRef is a lightweight artist reference.
type ArtistRef struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

// AlbumRef is a lightweight album reference.
type AlbumRef struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

// Track is a YouTube Music track / song.
type Track struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Artists     []ArtistRef `json:"artists,omitempty"`
	Album       *AlbumRef  `json:"album,omitempty"`
	Duration    int        `json:"duration,omitempty"` // seconds
	DurationText string    `json:"durationText,omitempty"`
	Artwork     []Artwork  `json:"artwork,omitempty"`
	Explicit    *bool      `json:"explicit,omitempty"`
	VideoType   string     `json:"videoType,omitempty"`
	IsAvailable *bool      `json:"isAvailable,omitempty"`
}

// Artist is a full artist profile.
type Artist struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Description     string      `json:"description,omitempty"`
	SubscriberCount string      `json:"subscriberCount,omitempty"`
	Avatar          []Artwork   `json:"avatar,omitempty"`
	Banner          []Artwork   `json:"banner,omitempty"`
	TopSongs        []Track     `json:"topSongs,omitempty"`
	Albums          []Album     `json:"albums,omitempty"`
	Singles         []Album     `json:"singles,omitempty"`
	Videos          []Track     `json:"videos,omitempty"`
	Featured        []Album     `json:"featured,omitempty"`
	Related         []ArtistRef `json:"related,omitempty"`
	Sections        []Section   `json:"sections,omitempty"`
}

// Album is an album / EP / single.
type Album struct {
	ID          string      `json:"id"`
	PlaylistID  string      `json:"playlistId,omitempty"`
	Title       string      `json:"title"`
	Artists     []ArtistRef `json:"artists,omitempty"`
	Year        int         `json:"year,omitempty"`
	Type        string      `json:"type,omitempty"` // Album, EP, Single
	Artwork     []Artwork   `json:"artwork,omitempty"`
	TrackCount  int         `json:"trackCount,omitempty"`
	Duration    int         `json:"duration,omitempty"`
	Explicit    *bool       `json:"explicit,omitempty"`
	Tracks      []Track     `json:"tracks,omitempty"`
	Description string      `json:"description,omitempty"`
}

// Playlist is a YouTube Music playlist.
type Playlist struct {
	ID           string      `json:"id"`
	Title        string      `json:"title"`
	Description  string      `json:"description,omitempty"`
	Author       *ArtistRef  `json:"author,omitempty"`
	Artwork      []Artwork   `json:"artwork,omitempty"`
	TrackCount   int         `json:"trackCount,omitempty"`
	TrackCountText string    `json:"trackCountText,omitempty"`
	Tracks       []Track     `json:"tracks,omitempty"`
	Continuation string      `json:"continuation,omitempty"`
}

// Section is a titled shelf of mixed items.
type Section struct {
	Title        string `json:"title"`
	BrowseID     string `json:"browseId,omitempty"`
	Params       string `json:"params,omitempty"`
	Tracks       []Track `json:"tracks,omitempty"`
	Albums       []Album `json:"albums,omitempty"`
	Playlists    []Playlist `json:"playlists,omitempty"`
	Artists      []ArtistRef `json:"artists,omitempty"`
}

// SearchResult holds search hits with optional continuation.
type SearchResult struct {
	Query        string     `json:"query,omitempty"`
	Filter       string     `json:"filter,omitempty"`
	Tracks       []Track    `json:"tracks,omitempty"`
	Albums       []Album    `json:"albums,omitempty"`
	Artists      []ArtistRef `json:"artists,omitempty"`
	Playlists    []Playlist `json:"playlists,omitempty"`
	Continuation string     `json:"continuation,omitempty"`
}

// Suggestion is a search suggestion.
type Suggestion struct {
	Text string `json:"text"`
}

// MoodGenre is a moods & genres category.
type MoodGenre struct {
	Title  string         `json:"title"`
	Items  []MoodGenreItem `json:"items"`
}

// MoodGenreItem is a single mood/genre chip.
type MoodGenreItem struct {
	Title    string `json:"title"`
	BrowseID string `json:"browseId,omitempty"`
	Params   string `json:"params,omitempty"`
	Color    string `json:"color,omitempty"`
}

// ChartSection is a charts shelf.
type ChartSection struct {
	Title     string      `json:"title"`
	Tracks    []Track     `json:"tracks,omitempty"`
	Albums    []Album     `json:"albums,omitempty"`
	Playlists []Playlist  `json:"playlists,omitempty"`
	Artists   []ArtistRef `json:"artists,omitempty"`
}

// StreamFormat describes an audio (or video) format.
type StreamFormat struct {
	Itag           int    `json:"itag"`
	URL            string `json:"url,omitempty"`
	MimeType       string `json:"mimeType"`
	Bitrate        int    `json:"bitrate,omitempty"`
	AverageBitrate int    `json:"averageBitrate,omitempty"`
	ContentLength  int64  `json:"contentLength,omitempty"`
	AudioQuality   string `json:"audioQuality,omitempty"`
	AudioSampleRate string `json:"audioSampleRate,omitempty"`
	AudioChannels  int    `json:"audioChannels,omitempty"`
	Codec          string `json:"codec,omitempty"`
	Container      string `json:"container,omitempty"`
	IsAudioOnly    bool   `json:"isAudioOnly"`
	ExpiresInSeconds int  `json:"expiresInSeconds,omitempty"`
}

// StreamResponse is the streams endpoint payload.
type StreamResponse struct {
	ID               string         `json:"id"`
	Title            string         `json:"title,omitempty"`
	Author           string         `json:"author,omitempty"`
	ChannelID        string         `json:"channelId,omitempty"`
	Duration         int            `json:"duration,omitempty"`
	Artwork          []Artwork      `json:"artwork,omitempty"`
	Client           string         `json:"client,omitempty"`
	ExpiresInSeconds int            `json:"expiresInSeconds,omitempty"`
	AudioFormats     []StreamFormat `json:"audioFormats,omitempty"`
	Formats          []StreamFormat `json:"formats,omitempty"`
}

// APIError is a structured API error body.
type APIError struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

// HealthResponse is the /health payload.
type HealthResponse struct {
	Status  string            `json:"status"`
	Version string            `json:"version"`
	Clients map[string]string `json:"clients,omitempty"`
}
