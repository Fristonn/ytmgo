# PO tokens

ytmgo does not ship a PO token generator. You can plug one in if you need WEB playback to work better. Without it, metadata and most mobile/VR streams still work.

`po_token.html` is just a reference helper for people running their own sidecar. The API never loads it.

## Manual

```bash
export PO_TOKEN_PROVIDER="manual"
export PO_TOKEN_PLAYER="..."
export PO_TOKEN_SESSION="..."
export PO_TOKEN_MANUAL_TTL="6h"
```

## HTTP sidecar

```bash
export PO_TOKEN_PROVIDER="http"
export PO_TOKEN_SERVICE_URL="http://127.0.0.1:4416"
export PO_TOKEN_SERVICE_PATH="/tokens"
export PO_TOKEN_SERVICE_GET_POT="/get_pot"
export PO_TOKEN_SERVICE_AUTH="Bearer YOUR_TOKEN"
export PO_TOKEN_SERVICE_API_KEY="optional-key"
export PO_TOKEN_SERVICE_TIMEOUT="10s"
export PO_TOKEN_CACHE_TTL="4m"
export PO_TOKEN_CACHE_MAX="256"
```

### POST /tokens

```json
{"videoId":"VIDEO_ID","visitorData":"VISITOR","clientName":"WEB_REMIX"}
```

```json
{"playerToken":"...","sessionToken":"...","expiresInSeconds":300}
```

### POST /get_pot

```json
{"content_binding":"VIDEO_ID","visitor_data":"VISITOR"}
```

```json
{"po_token":"..."}
```

`playerToken` goes into Innertube as `serviceIntegrityDimensions.poToken`.  
`sessionToken` / `pot` is appended on media urls as `?pot=`.

Force WEB to require tokens:

```bash
export PO_TOKEN_REQUIRED_FOR_WEB="true"
```

If `PO_TOKEN_PROVIDER` is empty: use HTTP when a URL is set, else manual tokens, else off.
