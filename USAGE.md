# Usage

Everything is `GET` and returns JSON. Default base url:

```text
http://localhost:8080
```

If you turned on API keys:

```http
X-API-Key: YOUR_KEY
```

or

```http
Authorization: Bearer YOUR_KEY
```

`/health` stays open either way.

---

## Table of contents

1. [Health](#1-health)
2. [Search](#2-search)
3. [Suggestions](#3-suggestions)
4. [Track](#4-track)
5. [Artist](#5-artist)
6. [Artist albums](#6-artist-albums)
7. [Artist songs](#7-artist-songs)
8. [Album](#8-album)
9. [Playlist](#9-playlist)
10. [Related](#10-related)
11. [Charts](#11-charts)
12. [Moods](#12-moods)
13. [Streams](#13-streams)
14. [Errors](#14-errors)

---

## 1. Health

Liveness check and optional client cooldown / PO-token status.

### Request

```http
GET /health
```

No path or query parameters.

### Example

```bash
curl -s "http://localhost:8080/health"
```

### Example response

```json
{
  "status": "ok",
  "version": "1.0.0",
  "clients": {
    "po_token": "disabled"
  }
}
```

When a client is cooling down after failures, `clients` may also include entries like `"WEB": "cooldown"`.

---

## 2. Search

Search YouTube Music (and fallback YouTube clients). Results are enriched across clients by entity ID.

### Request

```http
GET /api/v1/search
```

### Query parameters

| Name | Required | Description |
|------|----------|-------------|
| `q` | Yes\* | Search query (alias: `query`). Max length `200`. |
| `filter` | No | One of: `songs`, `videos`, `albums`, `artists`, `playlists`, `community_playlists`. Empty = mixed / best effort. |
| `continuation` | No | Opaque token from a previous response for the next page. |

\* `q` may be omitted only when `continuation` is set.

### Example

```bash
curl -s "http://localhost:8080/api/v1/search?q=blinding%20lights&filter=songs"
```

```bash
curl -s "http://localhost:8080/api/v1/search?q=the%20weeknd&filter=artists"
```

```bash
curl -s "http://localhost:8080/api/v1/search?continuation=TOKEN_FROM_PREVIOUS_RESPONSE"
```

### Example response (`filter=songs`)

```json
{
  "query": "blinding lights",
  "filter": "songs",
  "tracks": [
    {
      "id": "4NRXx6U8ABQ",
      "title": "Blinding Lights",
      "artists": [
        {
          "id": "UClYV6hHlupm_S_ObS1W-DYw",
          "name": "The Weeknd"
        }
      ],
      "album": {
        "id": "MPREb_…",
        "name": "After Hours"
      },
      "duration": 200,
      "durationText": "3:20",
      "artwork": [
        {
          "url": "https://lh3.googleusercontent.com/…",
          "width": 60,
          "height": 60
        },
        {
          "url": "https://lh3.googleusercontent.com/…",
          "width": 120,
          "height": 120
        }
      ],
      "explicit": false,
      "videoType": "MUSIC_VIDEO_TYPE_ATV"
    }
  ],
  "albums": [],
  "artists": [],
  "playlists": [],
  "continuation": "…optional…"
}
```

Empty arrays may be omitted depending on the filter and upstream payload.

---

## 3. Suggestions

Autocomplete-style search suggestions, plus any track chips the clients return.

### Request

```http
GET /api/v1/suggestions
```

### Query parameters

| Name | Required | Description |
|------|----------|-------------|
| `q` | Yes | Partial query. Max length `200`. |

### Example

```bash
curl -s "http://localhost:8080/api/v1/suggestions?q=blinding"
```

### Example response

```json
{
  "suggestions": [
    { "text": "blinding lights" },
    { "text": "blinding lights the weeknd" },
    { "text": "blinding lights slowed" }
  ],
  "tracks": [
    {
      "id": "4NRXx6U8ABQ",
      "title": "Blinding Lights",
      "artists": [
        { "name": "The Weeknd" }
      ],
      "artwork": [
        {
          "url": "https://lh3.googleusercontent.com/…",
          "width": 60,
          "height": 60
        }
      ]
    }
  ]
}
```

---

## 4. Track

Single track / video metadata (enriched across music + player clients).

### Request

```http
GET /api/v1/tracks/{id}
```

### Path parameters

| Name | Required | Description |
|------|----------|-------------|
| `id` | Yes | YouTube video / track ID (e.g. `4NRXx6U8ABQ`). |

### Example

```bash
curl -s "http://localhost:8080/api/v1/tracks/4NRXx6U8ABQ"
```

### Example response

```json
{
  "id": "4NRXx6U8ABQ",
  "title": "Blinding Lights",
  "artists": [
    {
      "id": "UClYV6hHlupm_S_ObS1W-DYw",
      "name": "The Weeknd"
    }
  ],
  "album": {
    "id": "MPREb_…",
    "name": "After Hours"
  },
  "duration": 200,
  "durationText": "3:20",
  "artwork": [
    {
      "url": "https://i.ytimg.com/vi/4NRXx6U8ABQ/hqdefault.jpg",
      "width": 480,
      "height": 360
    },
    {
      "url": "https://lh3.googleusercontent.com/…",
      "width": 544,
      "height": 544
    }
  ],
  "videoType": "MUSIC_VIDEO_TYPE_ATV"
}
```

---

## 5. Artist

Full artist profile: avatar, banner, top songs, albums, related artists, shelves.

### Request

```http
GET /api/v1/artists/{id}
```

### Path parameters

| Name | Required | Description |
|------|----------|-------------|
| `id` | Yes | Channel / browse ID (often starts with `UC…`). |

### Example

```bash
curl -s "http://localhost:8080/api/v1/artists/UClYV6hHlupm_S_ObS1W-DYw"
```

### Example response

```json
{
  "id": "UClYV6hHlupm_S_ObS1W-DYw",
  "name": "The Weeknd",
  "description": "…",
  "subscriberCount": "40.1M subscribers",
  "avatar": [
    {
      "url": "https://lh3.googleusercontent.com/…",
      "width": 150,
      "height": 150
    }
  ],
  "banner": [
    {
      "url": "https://lh3.googleusercontent.com/…",
      "width": 1440,
      "height": 424
    }
  ],
  "topSongs": [
    {
      "id": "4NRXx6U8ABQ",
      "title": "Blinding Lights",
      "artists": [{ "name": "The Weeknd" }],
      "duration": 200,
      "artwork": [{ "url": "https://lh3.googleusercontent.com/…", "width": 60, "height": 60 }]
    }
  ],
  "albums": [
    {
      "id": "MPREb_…",
      "title": "After Hours",
      "year": 2020,
      "type": "Album",
      "artwork": [{ "url": "https://lh3.googleusercontent.com/…", "width": 226, "height": 226 }]
    }
  ],
  "singles": [],
  "videos": [],
  "featured": [],
  "related": [
    { "id": "UC…", "name": "Related Artist" }
  ],
  "sections": [
    {
      "title": "Songs",
      "browseId": "…",
      "params": "…",
      "tracks": []
    }
  ]
}
```

Use `sections[].browseId` / `params` with the albums / songs endpoints below when you want a full shelf listing.

---

## 6. Artist albums

Albums / singles for an artist (shelf browse or values from the artist page).

### Request

```http
GET /api/v1/artists/{id}/albums
```

### Path parameters

| Name | Required | Description |
|------|----------|-------------|
| `id` | Yes | Artist browse / channel ID. |

### Query parameters

| Name | Required | Description |
|------|----------|-------------|
| `params` | No | Shelf params from `artist.sections[]`. |
| `continuation` | No | Next-page token. |

If `params` is empty, ytmgo tries the artist page’s albums / singles section automatically.

### Example

```bash
curl -s "http://localhost:8080/api/v1/artists/UClYV6hHlupm_S_ObS1W-DYw/albums"
```

```bash
curl -s "http://localhost:8080/api/v1/artists/UClYV6hHlupm_S_ObS1W-DYw/albums?params=PARAMS&continuation=TOKEN"
```

### Example response

```json
{
  "title": "Albums",
  "albums": [
    {
      "id": "MPREb_…",
      "playlistId": "OLAK5uy_…",
      "title": "After Hours",
      "artists": [{ "id": "UClYV6hHlupm_S_ObS1W-DYw", "name": "The Weeknd" }],
      "year": 2020,
      "type": "Album",
      "artwork": [
        {
          "url": "https://lh3.googleusercontent.com/…",
          "width": 226,
          "height": 226
        }
      ]
    }
  ],
  "continuation": "…optional…"
}
```

When falling back to the artist page without a dedicated shelf, the body may look like:

```json
{
  "albums": [],
  "singles": [],
  "continuation": ""
}
```

---

## 7. Artist songs

Top / all songs for an artist.

### Request

```http
GET /api/v1/artists/{id}/songs
```

### Path parameters

| Name | Required | Description |
|------|----------|-------------|
| `id` | Yes | Artist browse / channel ID. |

### Query parameters

| Name | Required | Description |
|------|----------|-------------|
| `params` | No | Shelf params from `artist.sections[]`. |
| `continuation` | No | Next-page token. |

### Example

```bash
curl -s "http://localhost:8080/api/v1/artists/UClYV6hHlupm_S_ObS1W-DYw/songs"
```

### Example response

```json
{
  "title": "Songs",
  "tracks": [
    {
      "id": "4NRXx6U8ABQ",
      "title": "Blinding Lights",
      "artists": [{ "name": "The Weeknd" }],
      "duration": 200,
      "durationText": "3:20",
      "artwork": [
        {
          "url": "https://lh3.googleusercontent.com/…",
          "width": 60,
          "height": 60
        }
      ]
    }
  ],
  "continuation": "…optional…"
}
```

---

## 8. Album

Album / EP / single details including track list when available.

### Request

```http
GET /api/v1/albums/{id}
```

### Path parameters

| Name | Required | Description |
|------|----------|-------------|
| `id` | Yes | Album browse ID (often `MPREb_…`). |

### Example

```bash
curl -s "http://localhost:8080/api/v1/albums/MPREb_…"
```

### Example response

```json
{
  "id": "MPREb_…",
  "playlistId": "OLAK5uy_…",
  "title": "After Hours",
  "artists": [
    {
      "id": "UClYV6hHlupm_S_ObS1W-DYw",
      "name": "The Weeknd"
    }
  ],
  "year": 2020,
  "type": "Album",
  "description": "…",
  "trackCount": 14,
  "duration": 3361,
  "artwork": [
    {
      "url": "https://lh3.googleusercontent.com/…",
      "width": 544,
      "height": 544
    }
  ],
  "tracks": [
    {
      "id": "4NRXx6U8ABQ",
      "title": "Blinding Lights",
      "artists": [{ "name": "The Weeknd" }],
      "duration": 200,
      "durationText": "3:20",
      "artwork": [
        {
          "url": "https://lh3.googleusercontent.com/…",
          "width": 60,
          "height": 60
        }
      ]
    }
  ]
}
```

---

## 9. Playlist

Playlist metadata and tracks, with optional continuation.

### Request

```http
GET /api/v1/playlists/{id}
```

### Path parameters

| Name | Required | Description |
|------|----------|-------------|
| `id` | Yes | Playlist ID. With or without `VL` prefix (`PL…`, `OLAK5uy_…`, etc.). |

### Query parameters

| Name | Required | Description |
|------|----------|-------------|
| `continuation` | No | Next-page token from a previous playlist response. |

### Example

```bash
curl -s "http://localhost:8080/api/v1/playlists/PL4fGSiJidcJb0X7mH9…"
```

```bash
curl -s "http://localhost:8080/api/v1/playlists/PL4fGSiJidcJb0X7mH9…?continuation=TOKEN"
```

### Example response

```json
{
  "id": "PL4fGSiJidcJb0X7mH9…",
  "title": "Today's Top Hits",
  "description": "…",
  "author": {
    "id": "UC…",
    "name": "YouTube Music"
  },
  "trackCount": 50,
  "trackCountText": "50 songs",
  "artwork": [
    {
      "url": "https://lh3.googleusercontent.com/…",
      "width": 544,
      "height": 544
    }
  ],
  "tracks": [
    {
      "id": "4NRXx6U8ABQ",
      "title": "Blinding Lights",
      "artists": [{ "name": "The Weeknd" }],
      "duration": 200,
      "artwork": [
        {
          "url": "https://lh3.googleusercontent.com/…",
          "width": 60,
          "height": 60
        }
      ]
    }
  ],
  "continuation": "…optional…"
}
```

When only paging with `continuation`, the response focuses on `tracks` + the next `continuation`.

---

## 10. Related

Related / recommended content for a track or video.

### Request

```http
GET /api/v1/related/{id}
```

### Path parameters

| Name | Required | Description |
|------|----------|-------------|
| `id` | Yes | Video / track ID. |

### Example

```bash
curl -s "http://localhost:8080/api/v1/related/4NRXx6U8ABQ"
```

### Example response

```json
{
  "title": "Related",
  "tracks": [
    {
      "id": "yzTuBuRdAyA",
      "title": "Save Your Tears",
      "artists": [{ "name": "The Weeknd" }],
      "duration": 215,
      "artwork": [
        {
          "url": "https://i.ytimg.com/vi/yzTuBuRdAyA/hqdefault.jpg",
          "width": 480,
          "height": 360
        }
      ]
    }
  ],
  "albums": [],
  "playlists": [],
  "artists": []
}
```

---

## 11. Charts

YouTube Music charts shelves.

### Request

```http
GET /api/v1/charts
```

No parameters.

### Example

```bash
curl -s "http://localhost:8080/api/v1/charts"
```

### Example response

```json
{
  "sections": [
    {
      "title": "Top songs",
      "tracks": [
        {
          "id": "…",
          "title": "…",
          "artists": [{ "name": "…" }],
          "artwork": [
            {
              "url": "https://lh3.googleusercontent.com/…",
              "width": 60,
              "height": 60
            }
          ]
        }
      ]
    },
    {
      "title": "Top albums",
      "albums": [
        {
          "id": "MPREb_…",
          "title": "…",
          "artwork": [
            {
              "url": "https://lh3.googleusercontent.com/…",
              "width": 226,
              "height": 226
            }
          ]
        }
      ]
    }
  ]
}
```

---

## 12. Moods

Moods & genres categories / chips.

### Request

```http
GET /api/v1/moods
```

No parameters.

### Example

```bash
curl -s "http://localhost:8080/api/v1/moods"
```

### Example response

```json
{
  "moods": [
    {
      "title": "Moods",
      "items": [
        {
          "title": "Chill",
          "browseId": "FEmusic_moods_and_genres_category",
          "params": "…",
          "color": "#…"
        },
        {
          "title": "Workout",
          "browseId": "FEmusic_moods_and_genres_category",
          "params": "…"
        }
      ]
    },
    {
      "title": "Genres",
      "items": [
        {
          "title": "Pop",
          "browseId": "FEmusic_moods_and_genres_category",
          "params": "…"
        }
      ]
    }
  ]
}
```

`browseId` / `params` can be used with browse-style flows on the client side; ytmgo exposes them as returned by Music.

---

## 13. Streams

Resolve audio (and related) formats for a video / track. Prefers clients that return direct URLs (`ANDROID_VR`, `IOS`, `ANDROID`, …).

### Request

```http
GET /api/v1/streams/{id}
```

### Path parameters

| Name | Required | Description |
|------|----------|-------------|
| `id` | Yes | Video / track ID. |

### Example

```bash
curl -s "http://localhost:8080/api/v1/streams/4NRXx6U8ABQ"
```

### Example response

```json
{
  "id": "4NRXx6U8ABQ",
  "title": "Blinding Lights",
  "author": "The Weeknd",
  "channelId": "UClYV6hHlupm_S_ObS1W-DYw",
  "duration": 200,
  "client": "ANDROID_VR",
  "expiresInSeconds": 21540,
  "artwork": [
    {
      "url": "https://i.ytimg.com/vi/4NRXx6U8ABQ/hqdefault.jpg",
      "width": 480,
      "height": 360
    }
  ],
  "audioFormats": [
    {
      "itag": 251,
      "url": "https://…googlevideo.com/videoplayback?…",
      "mimeType": "audio/webm; codecs=\"opus\"",
      "bitrate": 160001,
      "averageBitrate": 143556,
      "contentLength": 3588891,
      "audioQuality": "AUDIO_QUALITY_MEDIUM",
      "audioSampleRate": "48000",
      "audioChannels": 2,
      "codec": "opus",
      "container": "webm",
      "isAudioOnly": true,
      "expiresInSeconds": 21540
    },
    {
      "itag": 140,
      "url": "https://…googlevideo.com/videoplayback?…",
      "mimeType": "audio/mp4; codecs=\"mp4a.40.2\"",
      "bitrate": 131007,
      "codec": "mp4a.40.2",
      "container": "mp4",
      "isAudioOnly": true,
      "expiresInSeconds": 21540
    }
  ],
  "formats": []
}
```

Notes:

- Stream URLs expire. Re-call this endpoint when `expiresInSeconds` is close to zero.
- Set `EXPOSE_STREAM_URLS="false"` to omit `url` fields from the response.
- WEB-family clients may need a cipher sidecar and/or PO tokens; see [README](README.md) and [examples/README.md](examples/README.md).

---

## 14. Errors

Error bodies share this shape:

```json
{
  "error": "bad_request",
  "message": "query is required",
  "code": 400
}
```

### Common status codes

| HTTP | `error` | When |
|------|---------|------|
| `400` | `bad_request` | Missing / invalid params or IDs |
| `401` | `unauthorized` | API key required or invalid |
| `404` | `not_found` | Entity missing / unavailable |
| `429` | `rate_limited` or `upstream_rate_limited` | Local or upstream rate limit |
| `502` | `upstream_error` | Innertube / sidecar failure |
| `504` | `timeout` | Request deadline exceeded |
| `500` | `internal_error` | Unexpected server failure |

### Example error request

```bash
curl -s "http://localhost:8080/api/v1/search"
```

```json
{
  "error": "bad_request",
  "message": "query is required",
  "code": 400
}
```

```bash
curl -s "http://localhost:8080/api/v1/tracks/not%20valid"
```

```json
{
  "error": "bad_request",
  "message": "invalid id",
  "code": 400
}
```

---

## Quick reference

| Method | Endpoint |
|--------|----------|
| `GET` | `/health` |
| `GET` | `/api/v1/search` |
| `GET` | `/api/v1/suggestions` |
| `GET` | `/api/v1/tracks/{id}` |
| `GET` | `/api/v1/artists/{id}` |
| `GET` | `/api/v1/artists/{id}/albums` |
| `GET` | `/api/v1/artists/{id}/songs` |
| `GET` | `/api/v1/albums/{id}` |
| `GET` | `/api/v1/playlists/{id}` |
| `GET` | `/api/v1/related/{id}` |
| `GET` | `/api/v1/charts` |
| `GET` | `/api/v1/moods` |
| `GET` | `/api/v1/streams/{id}` |

Responses vary a bit by region and which client answered. Fields shown above are typical.
