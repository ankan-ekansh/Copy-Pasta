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
                      └──────────────────┘
```

---

## Backend (Go + Chi)

### Entry Point: `backend/cmd/server/main.go`
- Creates Chi router with middleware (CORS, logging, recoverer)
- Registers routes: `POST /api/convert`, `GET /api/health`
- Reads `PORT` from environment (default: 8080)

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

### Handler: `backend/internal/handler/handler.go`
- `POST /api/convert`: Accepts multipart form with `image` file and optional control fields
- Decodes JPEG/PNG/GIF, calls converter, returns JSON response
- `GET /api/health`: Returns `{"status": "ok"}`

### Middleware: `backend/internal/middleware/middleware.go`
- Chi's built-in Logger and Recoverer
- CORS configured to allow all origins (dev-friendly, tighten for production)

---

## Frontend (React + TypeScript + Vite)

### Components

| Component | File | Purpose |
|-----------|------|---------|
| App | `src/App.tsx` | Main layout, state management, conversion flow |
| ImageUploader | `src/components/ImageUploader.tsx` | File input, drag-drop, paste support |
| AsciiOutput | `src/components/AsciiOutput.tsx` | Displays result, copy button |

### API Client: `src/api/convert.ts`
- `convertImage(file, width?, invert?)` → `Promise<{ascii, width, height}>`
- Uses `FormData` with `fetch` POST to `/api/convert`

### Vite Config
- Proxies `/api` to `http://localhost:8080` in dev mode
- Production: nginx handles reverse proxy to backend

---

## Infrastructure

### Docker Compose (`docker-compose.yml`)
| Service | Port | Description |
|---------|------|-------------|
| backend | 8080 | Go API server |
| frontend | 3000 (nginx) | Static React app + API proxy |

### Makefile Commands
| Command | Description |
|---------|-------------|
| `make dev` | Run backend + frontend in dev mode (parallel) |
| `make dev-backend` | Go server only |
| `make dev-frontend` | Vite dev server only |
| `make build` | Build both projects |
| `make docker-up` | Build & start Docker Compose |
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
# Frontend at http://localhost:3000
# Backend at http://localhost:8080
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
| edgeMix | float | No | Edge detection blend 0-1 (default: auto based on image) |
| contrast | float | No | Contrast boost 0.1-3.0 (default: auto based on image) |
| charRamp | string | No | Custom character ramp string |

**Response**: `200 OK`
```json
{
  "ascii": "@@@###***...\n...",
  "width": 150,
  "height": 45
}
```

**Errors**: `400 Bad Request`
```json
{
  "error": "description of what went wrong"
}
```

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
| Image resize | CatmullRom | Best quality for downscaling |
| Height factor | 0.45 | Empirically tuned for monospace character aspect ratio |
| Character ramp | 70-level detailed | Eliminates banding, preserves subtle gradients |
| Edge detection | Sobel operator | Good balance of speed and edge quality |
| Contrast | Histogram stretch + 1.3x boost | Ensures full ramp usage regardless of input dynamic range |
| Luminance | Gamma-corrected BT.709 | Perceptually accurate brightness computation |
