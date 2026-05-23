# 🛠️ Copy-Pasta — Implementation Guide

## Architecture Overview

```
┌─────────────┐       ┌──────────────────┐
│   Browser   │──────▶│  React Frontend  │
│  (User)     │◀──────│  (Vite + TS)     │
└─────────────┘       └────────┬─────────┘
                               │ /api/*
                               ▼
                      ┌──────────────────┐
                      │   Go Backend     │
                      │   (Chi Router)   │
                      ├──────────────────┤
                      │ • handler/       │
                      │ • converter/     │
                      │ • middleware/    │
                      │ • store/         │
                      └────────┬─────────┘
                               │ DATABASE_URL
                               ▼
                      ┌──────────────────┐
                      │   PostgreSQL 16  │
                      └──────────────────┘
```

---

## Backend (Go + Chi)

### Entry Point: `backend/cmd/server/main.go`
- Creates Chi router with middleware (CORS, logging, recoverer, session cookies)
- Registers routes: `POST /api/convert`, `GET /api/health`, `/api/pastas` (list, get, delete, patch)
- Reads `PORT` from environment (default: 8080)
- Connects to PostgreSQL via `DATABASE_URL` (graceful degradation if unset/unavailable)
- Passes `Store` to handlers via functional options

### Converter: `backend/internal/converter/converter.go`
The core ASCII art engine — produces high-quality output using adaptive image processing.

**Algorithm (pipeline):**
1. **Resize** — CatmullRom (bicubic) scaling to target width, 0.45 height factor for monospace aspect ratio
2. **Luminance** — Gamma-corrected BT.709 perceptual brightness (`γ=2.2` decode → weighted sum → `γ=2.2` encode)
3. **Image analysis** — Compute min/max/mean/stddev to auto-tune processing parameters
4. **Histogram normalization** — Stretch to full 0–1 range so all character levels are utilized
5. **Adaptive contrast** — Auto-selects boost factor based on image stddev (low-contrast images get more boost)
6. **Sobel edge detection** — Compute gradient magnitude, auto-blend based on image characteristics
7. **Character mapping** — Quantize intensity to density-sorted character ramp with rounding

**Options:**
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| Width | int | 150 | Output width in characters |
| Invert | bool | false | Reverse light/dark mapping |
| EdgeMix | float64 | auto | Edge blend (0=none, 1=max). Auto-detects based on image stats |
| Contrast | float64 | auto | Contrast factor (1.0=none). Auto: 1.1–1.8 based on stddev |
| CharRamp | string | RampDefault | Character set for mapping |

**Available character ramps:**
| Name | Characters | Levels | Best for |
|------|-----------|--------|----------|
| RampDefault | ` .,:;+*?%S#@` | 12 | General purpose — clean, no visual noise |
| RampClean | ` .:;+*%#@` | 9 | Minimal, very clean output |
| RampFull | 70-char Bourke ramp | 70 | Large widths (200+), photographic detail |
| RampBlocks | ` ░▒▓█` | 5 | Terminal block-art style |

**Key design insight:** The default 12-character ramp uses only "isotropic" characters (those that look the same regardless of neighbors). Characters like `|`, `/`, `\`, `(`, `)` create directional visual noise and are excluded from the default. They're available in `RampFull` for specialized use at large widths.

### Braille Converter: `backend/internal/converter/braille.go`
A **higher-resolution** conversion mode using Unicode Braille characters (U+2800–U+28FF).

**How it works:**
Each Braille character encodes a 2×4 dot matrix (8 binary pixels per character cell). This gives dramatically higher effective resolution than brightness-to-char mapping — a 80-character-wide output has 160 effective horizontal pixels.

**Algorithm:**
1. **Resize** — Scale image so width in pixels = `Width × 2`, height adjusted with aspect ratio correction (÷4 rows per char)
2. **Grayscale** — Convert to gamma-correct luminance (same BT.709 formula as ASCII mode)
3. **Histogram normalization** — Stretch brightness to full 0–1 range
4. **Edge detection** — Sobel operator blended into brightness (stronger at narrow widths)
5. **Contrast boost** — Enhance detail (stronger at narrow widths)
6. **Dithering** — Floyd-Steinberg error diffusion (always enabled via API; pushes pixels toward 0 or 1, simulating grayscale through dot density)
7. **Thresholding** — Binary threshold at 0.5 (post-dithering). When dithering is disabled: Otsu's method auto-detects optimal threshold.
8. **Dot mapping** — Each 2×4 block maps to Braille dot positions: `[0,3 / 1,4 / 2,5 / 6,7]` → bit offset from U+2800
9. **Character assembly** — Each character is `rune(0x2800 + dotBits)`

**Options:**
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| Width | int | 80 | Output width in Braille characters (effective px = width×2) |
| Threshold | float64 | 0 | Binary threshold. With dithering (always on via API): 0 → 0.5. Without dithering: 0 → Otsu auto-detect |
| Invert | bool | false | Invert dot pattern |
| Dither | bool | false | Floyd-Steinberg dithering (always `true` via API) |

**When to use which mode (A/B comparison):**
| Criterion | ASCII Mode | Braille Mode |
|-----------|-----------|--------------|
| Resolution | Lower (1 pixel per char) | Higher (8 pixels per char) |
| Style | Classic retro terminal | Modern Unicode art |
| Best for | Artistic/stylized look | Recognizable meme reproduction |
| Compatibility | Works everywhere | Needs Unicode Braille font support |
| Grayscale | Yes (12+ levels) | Simulated via dithered dot density (binary dots, perceptual grayscale) |

### Handler: `backend/internal/handler/handler.go`
- `POST /api/convert`: Accepts multipart form with `image` file and optional control fields
  - `mode` field: `"ascii"` (default) or `"braille"` — selects conversion algorithm
  - `threshold` field: float 0–1 for braille mode. If omitted or 0, defaults to 0.5 (since the API always enables dithering). Otsu auto-threshold is only used when dithering is disabled (not exposed via API).
- Decodes JPEG/PNG/GIF, routes to appropriate converter, returns JSON response
- Auto-saves to DB on successful conversion (best-effort, never fails the request)
- Returns `id` field in response when persistence is available
- `GET /api/health`: Returns `{"status": "ok"}`

### Handler: `backend/internal/handler/pastas.go`
- `GET /api/pastas` — list pastas for current session (paginated via limit/offset)
- `GET /api/pastas/:id` — view any pasta by ID (unlisted-but-shareable)
- `DELETE /api/pastas/:id` — delete pasta (atomic ownership check)
- `PATCH /api/pastas/:id` — set is_public (atomic ownership check)

### Middleware: `backend/internal/middleware/middleware.go`
Global middleware chain (in order):
1. **CORS** — Origin validation, credentials support, wildcard+credentials guard
2. **RealIP** (`chi/middleware.RealIP`) — Conditional: only enabled when `TRUSTED_PROXY=true`. Extracts client IP from `X-Forwarded-For`/`X-Real-IP` headers.
3. **RequestID** — Generates/propagates `X-Request-ID` header
4. **Metrics** — Records HTTP request count and duration (Prometheus histograms)
5. **RequestLog** — Structured access logging via `slog`
6. **Recoverer** — Panic recovery
7. **Session** — Sets `copy-pasta-session` UUID cookie (HttpOnly, SameSite=Lax, Secure via TLS/X-Forwarded-Proto)

**RateLimitAPI** is applied only to `/api` routes (via chi `Route` group), not to static assets or `/metrics`.

### Rate Limiting: `backend/internal/middleware/ratelimit.go`
- **RateLimitConvert()** — Applied per-route on `POST /api/convert`. 10 req/min per IP (configurable via `RATE_LIMIT_CONVERT`).
- **RateLimitAPI()** — Applied to general `/api` routes (excludes `/api/convert` which has its own limiter). 100 req/min per IP (configurable via `RATE_LIMIT_API`).
- IP keying is conditional on `TRUSTED_PROXY`:
  - `TRUSTED_PROXY=true` → uses `httprate.WithKeyByRealIP()` (reads X-Real-IP/X-Forwarded-For)
  - `TRUSTED_PROXY` unset/false → uses `httprate.WithKeyByIP()` (reads RemoteAddr directly)
- `Retry-After` header is derived from `rateLimitWindow` (currently 60s).
- Returns 429 with JSON `{"error": "rate limit exceeded, try again later"}`.

**Trust model:**
- **Docker Compose** (`TRUSTED_PROXY=true`): Backend port is not published — only reachable via nginx (port 3000). Nginx overwrites `X-Real-IP` and `X-Forwarded-For` with `$remote_addr`, so client-supplied values are discarded. RealIP middleware + `WithKeyByRealIP()` are safe.
- **Azure** (`TRUSTED_PROXY=true`, set by `infra/setup-azure.sh`): Container Apps ingress overwrites `X-Forwarded-For` at the edge. The env var is set alongside `DATABASE_URL` during provisioning.
- **Local dev** (`make dev-backend`, `TRUSTED_PROXY` unset): RealIP middleware is skipped, rate limiting uses `RemoteAddr` directly. No spoofing possible.

### Metrics: `backend/internal/metrics/`
- Prometheus counters and histograms (no namespace prefix)
- Metric names: `conversions_total`, `conversion_duration_seconds`, `http_requests_total`, `http_request_duration_seconds`, `db_operation_duration_seconds`
- `/metrics` endpoint gated by `EXPOSE_METRICS=true` (opt-in)

### InstrumentedStore: `backend/internal/store/instrumented.go`
- Decorator wrapping any `Store` implementation to record `db_operation_duration_seconds`
- Exposes `Ping()` via type assertion on inner store (returns `ErrPingNotSupported` if inner lacks it)

### Store: `backend/internal/store/`
- `Store` interface: `Save`, `Get`, `ListBySession`, `DeleteByOwner`, `SetPublicByOwner`, `Close`
- `PostgresStore` implementation using `pgxpool` (connection pool)
- Runs migration on startup (CREATE TABLE IF NOT EXISTS, separate statements for pgx compatibility)
- Connection via `DATABASE_URL` env var with 10s timeout
- Uses crypto/rand for URL-safe IDs (10 chars)
- Atomic ownership checks: `WHERE id=$1 AND session_id=$2` (no TOCTOU races)
- **Graceful degradation**: If DB unavailable, app starts without persistence; pasta endpoints return 503

---

## Frontend (React + TypeScript + Vite)

### Components

| Component | File | Purpose |
|-----------|------|---------|
| App | `src/App.tsx` | Main layout, state management, conversion flow, share link |
| ImageUploader | `src/components/ImageUploader.tsx` | File input, drag-drop, paste support |
| AsciiOutput | `src/components/AsciiOutput.tsx` | Displays result, copy button |
| HistoryPanel | `src/components/HistoryPanel.tsx` | Recent conversions list with share/view/delete |
| PastaView | `src/components/PastaView.tsx` | Share page (`/pasta/:id`) with read-only ASCII view |

### Routing: `src/main.tsx`
- `BrowserRouter` with React Router v7
- Routes: `/` (App), `/pasta/:id` (PastaView), `*` (catch-all → redirect to `/`)

### API Client: `src/api/convert.ts`
- `convertImage(file, options?)` → `Promise<{ascii, width, height, id?}>`
- Options: `{ width?, invert?, mode?: 'ascii'|'braille', threshold? }`
- Uses `FormData` with `fetch` POST to `/api/convert`
- Includes `credentials: 'include'` for session cookie

### API Client: `src/api/pastas.ts`
- `getPasta(id)` → `Promise<Pasta>` — fetch a single pasta by ID
- `listPastas(limit?, offset?)` → `Promise<Pasta[]>` — list user's pastas (returns `[]` on 503)
- `deletePasta(id)` → `Promise<void>` — delete a pasta by ID
- All use `credentials: 'include'` and `encodeURIComponent(id)` in paths

### Vite Config
- Proxies `/api` to `http://localhost:8080` in dev mode
- Production: Go backend serves pre-built static files directly (no separate frontend service)
- Docker Compose (local): nginx container proxies API to backend

---

## Infrastructure

### Docker Compose (`docker-compose.yml`)
| Service | Port | Description |
|---------|------|-------------|
| backend | 8080 | Go API server (connects to postgres) |
| frontend | 3000 (nginx) | Static React app + API proxy |
| postgres | 5432 | PostgreSQL 16 for persistence |

### Makefile Commands
| Command | Description |
|---------|-------------|
| `make dev` | Run backend + frontend in dev mode (parallel) |
| `make dev-backend` | Go server only |
| `make dev-frontend` | Vite dev server only |
| `make build` | Build both projects |
| `make docker-up` | Build & start Docker Compose |
| `make docker-up-obs` | Build & start with observability (Prometheus + Grafana) |
| `make docker-down` | Stop Docker Compose |
| `make test` | Run Go tests |
| `make lint` | Run linters (go vet + eslint) |
| `make clean` | Remove build artifacts |

---

## Running Locally

### Option 1: Native (fastest feedback loop)
```bash
# Terminal 1 — Backend
cd backend && go run ./cmd/server
# Runs on :8080

# Terminal 2 — Frontend  
cd frontend && npm run dev
# Runs on :5173, proxies /api to :8080
```

Or use: `make dev` (runs both in parallel)

### Option 2: Docker Compose
```bash
make docker-up
# App at http://localhost:3000 (nginx proxies /api to backend internally)
```

---

## API Reference

### `POST /api/convert`
Convert an image to ASCII art.

**Request**: `multipart/form-data`
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| image | File | Yes | JPEG, PNG, or GIF image |
| width | int | No | Output width in chars (default: 150) |
| invert | bool | No | Invert brightness mapping |
| mode | string | No | `"ascii"` (default) or `"braille"` |
| threshold | float | No | Braille threshold 0-1. Omit or 0 → defaults to 0.5 (dithering always on) |
| edgeMix | float | No | Edge detection blend 0-1 (default: auto based on image) |
| contrast | float | No | Contrast boost 0.1-3.0 (default: auto based on image) |
| charRamp | string | No | Custom character ramp string |

**Response**: `200 OK`
```json
{
  "ascii": "@@@###***...\n...",
  "width": 150,
  "height": 45,
  "id": "aBcDeFgHiJ"
}
```

> **Note**: `id` is only present when persistence is enabled (`DATABASE_URL` configured). It is omitted otherwise.

**Errors**: `400 Bad Request`
```json
{
  "error": "description of what went wrong"
}
```

### `GET /api/pastas`
List current user's pastas (session-based).

**Query params**: `limit` (default 20), `offset` (default 0)

**Response**: `200 OK`
```json
{
  "pastas": [
    {
      "id": "aBcDeFgHiJ",
      "ascii_art": "...",
      "width": 150,
      "height": 45,
      "mode": "ascii",
      "is_public": false,
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

**Errors**: `503 Service Unavailable` (persistence disabled)

### `GET /api/pastas/:id`
Get a single pasta by ID (shareable, no auth required).

**Response**: `200 OK` — same shape as list item

**Errors**: `404 Not Found`, `503 Service Unavailable`

### `DELETE /api/pastas/:id`
Delete a pasta (owner only, atomic ownership check).

**Response**: `204 No Content`

**Errors**: `404 Not Found` (or not owner), `503 Service Unavailable`

### `PATCH /api/pastas/:id`
Set public/private visibility (owner only).

**Request**: `application/json`
```json
{ "is_public": true }
```

**Response**: `200 OK`
```json
{ "is_public": true }
```

**Errors**: `404 Not Found` (or not owner), `503 Service Unavailable`

### `GET /api/health`
Health check endpoint.

**Response**: `200 OK`
```json
{
  "status": "ok"
}
```

---

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| ASCII processing | Server-side Go | Learn Go image processing, consistent output |
| Router | Chi | Lightweight, idiomatic, great middleware |
| Frontend | Vite + React + TS | Fast dev experience, type safety |
| Frontend routing | React Router v7 | Declarative, standard SPA routing |
| Image resize | CatmullRom | Best quality for downscaling |
| Height factor | 0.45 | Empirically tuned for monospace character aspect ratio |
| Character ramp | 12-level isotropic default | Clean output, no directional noise; 70-level available for large widths |
| Edge detection | Sobel operator | Good balance of speed and edge quality |
| Contrast | Histogram stretch + adaptive boost (1.1–1.8×) | Auto-selects based on image stddev |
| Luminance | Gamma-corrected BT.709 | Perceptually accurate brightness computation |
| Braille mode | Unicode Braille (U+2800-28FF) | 2×4 dot patterns, 2× resolution vs ASCII |
| Persistence | PostgreSQL + pgx/v5 | Production-grade, Azure-compatible, graceful degradation |
| Session identity | UUID cookie (HttpOnly) | Simple, no login required, secure |
| ID generation | crypto/rand base62 (10 chars) | URL-safe, collision-resistant, no external deps |
