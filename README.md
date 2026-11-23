# ytmgo

REST API for YouTube Music metadata and audio streams. Pure Go.

Module: [`github.com/Fristonn/ytmgo`](https://github.com/Fristonn/ytmgo)

## Run

```bash
cp .env.example .env
go run .
```

```bash
go build -o bin/ytmgo .
./bin/ytmgo
```

Docker:

```bash
docker build -t ytmgo .
docker run --rm -p 8080:8080 --env-file .env ytmgo
```

Listens on `:8080` by default.

## What it does

- search, suggestions, tracks, artists, albums, playlists
- related, charts, moods
- stream URL resolution
- pulls from several Innertube clients and merges whatever each one returns (artwork, artists, duration, etc.)

API details and sample responses are in [USAGE.md](USAGE.md).

## Endpoints

| | |
|--|--|
| `GET /health` | health |
| `GET /api/v1/search` | `q`, `filter`, `continuation` |
| `GET /api/v1/suggestions` | `q` |
| `GET /api/v1/tracks/{id}` | track |
| `GET /api/v1/artists/{id}` | artist |
| `GET /api/v1/artists/{id}/albums` | albums |
| `GET /api/v1/artists/{id}/songs` | songs |
| `GET /api/v1/albums/{id}` | album |
| `GET /api/v1/playlists/{id}` | playlist |
| `GET /api/v1/related/{id}` | related |
| `GET /api/v1/charts` | charts |
| `GET /api/v1/moods` | moods / genres |
| `GET /api/v1/streams/{id}` | formats + urls |

```bash
curl "http://localhost:8080/api/v1/search?q=blinding%20lights&filter=songs"
curl "http://localhost:8080/api/v1/streams/4NRXx6U8ABQ"
```

## Clients

No OAuth. Optional PO tokens for WEB if you set them up ([examples](examples/README.md)). Streams usually come from mobile/VR clients with direct urls.

Tested against `4NRXx6U8ABQ` (things break when YouTube changes stuff):

| Client | Opus | Direct urls | PO | Playback | Search | Browse | Suggest | Related | Notes |
|--------|------|-------------|----|----------|--------|--------|---------|---------|-------|
| `WEB_REMIX` | | | helps | no* | yes | yes | yes | yes | main music metadata |
| `ANDROID_MUSIC` | | | | no* | yes | yes | yes | yes | music metadata, player often wants login |
| `WEB` | | | helps | no* | yes | yes | | yes | search/browse |
| `MWEB` | | | helps | flaky | flaky | flaky | | flaky | lots of 400s |
| `ANDROID` | yes | yes | | yes | yes | yes | | yes | streams + some metadata |
| `ANDROID_VR` | yes | yes | | yes | | yes | | yes | best for streams right now |
| `IOS` | yes | yes | | yes | | yes | | kinda | stream fallback |
| `WEB_EMBEDDED_PLAYER` | | | helps | sometimes | | | | yes | |
| `TVHTML5` | | | | no* | | | | yes | fallback |
| `TVHTML5_SIMPLY` | | | | no* | | | | yes | |
| `TVHTML5_SIMPLY_EMBEDDED_PLAYER` | | | | no | | | | yes | dead upstream |
| `MEDIA_CONNECT_FRONTEND` | | | | no | | no | | no | 404 lately |

\* no usable formats when we tested; metadata can still work.

Order we try:

- metadata: `WEB_REMIX`, `ANDROID_MUSIC`, `WEB`, `MWEB`, `ANDROID`, …
- streams: `ANDROID_VR`, `ANDROID_MUSIC`, `IOS`, `ANDROID`, then the rest

## Config

Copy [`.env.example`](.env.example). Useful knobs:

- `YT_HL` / `YT_GL`
- `API_KEYS`, `CORS_ORIGINS`
- `CACHE_TTL`
- `CIPHER_SERVICE_URL` if you need signature decryption
- PO token vars — [examples/README.md](examples/README.md)
- `EXPOSE_STREAM_URLS=false` to strip urls from stream responses

## Layout

```
main.go
internal/api          router, middleware, handlers
internal/innertube    clients + parsers
internal/playback     stream resolution
internal/cipher       optional remote cipher
internal/potoken      optional PO tokens
internal/service      glue
internal/models       json types
…
```

```bash
go vet ./...
go build -o bin/ytmgo .
```

## License

MIT
