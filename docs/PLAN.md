# 🗺️ Copy-Pasta — Project Plan

## Vision
A fun web app where users paste/upload meme images and get ASCII art back that they can copy-paste anywhere. Built as a progressive learning playground.

---

## Phase 1: Foundation — "Hello ASCII" ✅
**Status**: Complete  
**Goal**: End-to-end flow: upload image → get ASCII art back.

| Task | Status |
|------|--------|
| Go module + Chi router setup | ✅ |
| ASCII converter (brightness → char mapping) | ✅ |
| Image upload endpoint (`POST /api/convert`) | ✅ |
| React + TypeScript frontend scaffold | ✅ |
| Upload UI + ASCII display + copy button | ✅ |
| Docker Compose setup | ✅ |
| Makefile | ✅ |

---

## Phase 2: Polish — "Make It Delightful"
**Status**: Upcoming  
**Goal**: Improve UX, add styling, and make the output look good.

| Task | Status |
|------|--------|
| Clipboard paste support (Ctrl+V) | ⬜ |
| Drag-and-drop upload | ⬜ |
| ASCII preview with proper monospace sizing | ⬜ |
| Controls: width slider, character set, invert | ⬜ |
| Dark/light theme toggle | ⬜ |
| Loading states & error handling polish | ⬜ |
| Responsive design for mobile | ⬜ |

---

## Phase 3: Persistence — "Remember the Pastas"
**Status**: Planned  
**Goal**: Store conversions in a database for revisiting.

| Task | Status |
|------|--------|
| Add PostgreSQL via Docker Compose | ⬜ |
| Schema: `conversions` table | ⬜ |
| Go database layer (pgx) | ⬜ |
| API: `GET /api/pastas`, `GET /api/pastas/:id` | ⬜ |
| Frontend: gallery/history page | ⬜ |
| Shareable URLs (`/pasta/:id`) | ⬜ |

---

## Phase 4: Observability — "See What's Happening"
**Status**: Planned  
**Goal**: Structured logging, metrics, and tracing.

| Task | Status |
|------|--------|
| Structured logging (slog/zerolog) | ⬜ |
| OpenTelemetry tracing | ⬜ |
| Prometheus metrics (`/metrics`) | ⬜ |
| Grafana + Prometheus in Docker Compose | ⬜ |
| Health check endpoint (`GET /api/health`) | ✅ |

---

## Phase 5: Sharing & Social — "Show Off Your Art"
**Status**: Planned  
**Goal**: Let users share creations and see what others made.

| Task | Status |
|------|--------|
| Public gallery (opt-in sharing) | ⬜ |
| Share links with OG meta tags | ⬜ |
| Like / upvote system | ⬜ |
| Leaderboard: most liked | ⬜ |
| Rate limiting middleware | ⬜ |

---

## Phase 6: Production Hardening — "Ship It"
**Status**: Planned  
**Goal**: Make it deployable to a real environment.

| Task | Status |
|------|--------|
| Multi-stage Docker build (optimized) | ⬜ |
| CI/CD pipeline (GitHub Actions) | ⬜ |
| Environment-based configuration | ⬜ |
| CORS, security headers, input validation | ⬜ |
| Graceful shutdown | ⬜ |
| Database migrations (golang-migrate) | ⬜ |
| Cloud deployment (Fly.io / Railway / GCP) | ⬜ |

---

## Stretch Ideas
- Animated GIF → animated ASCII
- Webcam → live ASCII stream
- Color ASCII (ANSI escape codes)
- Braille character mode for higher resolution
- Custom character ramp editor
