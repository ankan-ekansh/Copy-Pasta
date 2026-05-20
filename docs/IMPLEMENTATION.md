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
The core ASCII art engine.

**Algorithm:**
1. Resize input image to target width, maintaining aspect ratio
2. Apply 0.5 height factor (characters are ~2x taller than wide)
3. Use CatmullRom interpolation for high-quality downscaling
4. For each pixel, compute luminance: `0.299R + 0.587G + 0.114B`
5. Map brightness to ASCII ramp: ` .:-=+*#%@`
6. Build output string with newlines between rows

**Options:**
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| Width | int | 120 | Output width in characters |
| Invert | bool | false | Reverse light/dark mapping |

### Handler: `backend/internal/handler/handler.go`
- `POST /api/convert`: Accepts multipart form with `image` file, optional `width` and `invert` fields
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
| width | int | No | Output width (default: 120) |
| invert | bool | No | Invert brightness mapping |

**Response**: `200 OK`
```json
{
  "ascii": "@@@###***...\n...",
  "width": 120,
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
| Height factor | 0.5 | Characters are ~2x taller than wide in monospace |
| Character ramp | ` .:-=+*#%@` | Good contrast spread, 10 levels |
