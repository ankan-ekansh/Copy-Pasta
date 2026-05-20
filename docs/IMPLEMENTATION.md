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
The core ASCII art engine — produces high-quality output using multiple image processing techniques.

**Algorithm (pipeline):**
1. **Resize** — Scale to target width using CatmullRom interpolation, apply 0.45 height factor (compensates for character aspect ratio in monospace fonts)
2. **Brightness extraction** — Compute per-pixel luminance using gamma-corrected BT.709 weights: `0.2126R + 0.7152G + 0.0722B` (with gamma 2.2 correction for perceptual accuracy)
3. **Histogram normalization** — Stretch min/max brightness to fill the full 0-1 range, ensuring all character ramp levels are used
4. **Contrast boost** — Apply midpoint-centered contrast scaling (default 1.3x) to push values away from the middle, creating crisper output
5. **Edge detection** — Sobel operator computes gradient magnitude, normalized to 0-1
6. **Edge blending** — Mix brightness and edge maps (default 30% edges) so structural lines remain visible even in flat-brightness regions
7. **Character mapping** — Map final intensity value to ASCII character ramp

**Options:**
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| Width | int | 150 | Output width in characters |
| Invert | bool | false | Reverse light/dark mapping |
| EdgeMix | float64 | 0.3 | Edge detection blend (0=pure brightness, 1=pure edges) |
| Contrast | float64 | 1.3 | Contrast boost factor (1.0=no change, higher=more contrast) |
| CharRamp | string | RampDetailed | Character set for mapping brightness levels |

**Available character ramps:**
| Name | Characters | Levels | Best for |
|------|-----------|--------|----------|
| RampDetailed | ` .'^\`",:;Il!i><~+_-?][}{1)(|\\/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$` | 70 | High-detail images (default) |
| RampStandard | ` .:-=+*#%@` | 10 | Simple/retro look |
| RampBlocks | ` ░▒▓█` | 5 | Block-art style |
| RampSimple | ` .oO@` | 5 | Minimal, high-contrast |

**Why these techniques matter:**
- **Histogram normalization** prevents "washed out" output where most of the image maps to the same few characters (the main issue with the original converter)
- **Gamma-correct luminance** ensures mid-tones render faithfully instead of appearing too dark
- **Edge detection** preserves structural detail (outlines, facial features) that pure brightness mapping loses in flat-color regions
- **70-level character ramp** vs original 10-level eliminates banding artifacts

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
| edgeMix | float | No | Edge detection blend 0-1 (default: 0.3) |
| contrast | float | No | Contrast boost 0.1-3.0 (default: 1.3) |
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
