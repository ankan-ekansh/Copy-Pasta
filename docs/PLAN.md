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
**Status**: In Progress  
**Goal**: Improve UX, add styling, and make the output look good.

| Task | Status |
|------|--------|
| Clipboard paste support (Ctrl+V) | ✅ |
| Drag-and-drop upload | ✅ |
| ASCII preview with proper monospace sizing | ✅ |
| Controls: width slider, character set, invert | ✅ |
| Dark/light theme toggle | ✅ |
| Loading states & error handling polish | ✅ |
| Responsive design for mobile | ⬜ |
| Braille Unicode mode (A/B toggle) | ✅ |
| Floyd-Steinberg dithering | ✅ |
| Edge detection + contrast boost | ✅ |
| Mobile share presets (WhatsApp, Twitter, etc.) | ✅ |
| Auto-detect image type & pick best strategy (photo/sketch/logo) | ⬜ (deferred — build after observability data) |
| Multiple dithering algorithms (Atkinson, Stucki) | ⬜ |

---

## Phase 2.5: Deploy MVP — "Ship It Early" ✅
**Status**: Complete  
**Goal**: Get the app live on Azure so people can use it. Minimal viable deployment.

**Live URL**: https://copy-pasta.happyflower-831a5c58.eastus.azurecontainerapps.io

### Architecture (Azure)
```
┌─────────────┐       ┌────────────────────────┐
│   Browser   │──────▶│  Azure Container App    │
│   (Users)   │◀──────│  (single container)     │
└─────────────┘       │  Go serves API + static │
                      └────────────────────────┘
```

**Strategy**: Single-container deployment — Go backend serves both the API and the pre-built React static files. No separate frontend service needed.

### Tasks

| Task | Status |
|------|--------|
| Go serves static frontend files (embed or serve dir) | ✅ |
| Multi-stage Dockerfile (build Go + React in one image) | ✅ |
| Azure Container Registry (ACR) setup | ✅ |
| Azure Container App deployment | ✅ |
| Custom domain + HTTPS (optional, Azure auto-TLS) | ⬜ |
| GitHub Actions CI/CD (build → push → deploy) | ✅ |
| Environment config (PORT, CORS origin) | ✅ |
| Health check probe configured | ✅ |

### Azure Resources Needed
| Resource | SKU/Tier | Est. Cost |
|----------|----------|-----------|
| Container Registry | Basic | ~$5/mo |
| Container Apps Environment | Consumption | Pay-per-use (near-free at low traffic) |
| Custom domain (optional) | - | Free with Azure-managed cert |

### Deployment Steps (manual first, then automate)
1. Create resource group: `rg-copy-pasta`
2. Create ACR: `copypasta.azurecr.io`
3. Build & push unified Docker image
4. Create Container Apps Environment + App
5. Configure ingress (port 8080, external)
6. Verify health check works
7. Set up GitHub Actions for CI/CD

### Why Azure Container Apps?
- Scales to zero (no cost when idle)
- Built-in HTTPS/TLS
- Simple container deployment (no K8s complexity)
- Auto-scaling on traffic
- Perfect for a fun side project with company credits

---

## Phase 3: Persistence — "Remember the Pastas"
**Status**: Planned  
**Goal**: Save conversions so users can revisit, share, and (later) browse others' art.

### Design Decisions
- **No auth** — anonymous usage via session cookie (random UUID)
- **SQLite** — single-file DB, zero ops, ships in the container
- **No file storage** — only persist the generated ASCII text, not source images
- **Shareable links** — each conversion gets a short ID, viewable by anyone
- **Ownership via cookie** — creator can manage their art while cookie persists

### Data Model

```sql
CREATE TABLE pastas (
  id TEXT PRIMARY KEY,            -- nanoid, 10 chars (e.g. "V1StGXR8_Z")
  session_id TEXT NOT NULL,       -- links to creator's cookie
  ascii_art TEXT NOT NULL,        -- the generated output
  width INT NOT NULL,
  height INT NOT NULL,
  mode TEXT NOT NULL,             -- 'ascii' or 'braille'
  is_public BOOLEAN DEFAULT FALSE, -- opt-in for future gallery (Phase 5)
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_pastas_session ON pastas(session_id);
CREATE INDEX idx_pastas_public ON pastas(is_public, created_at);
```

### Implementation Steps (gradual)

#### Step 1: Database layer
- Add SQLite driver (`modernc.org/sqlite` — pure Go, no CGO)
- Create `internal/store` package with `Store` interface
- Auto-create table on startup (embedded migration)
- Wire into server startup

#### Step 2: Session cookie middleware
- Middleware checks for `copy-pasta-session` cookie
- If missing, generate UUID and set cookie (HttpOnly, SameSite=Lax, 1 year expiry)
- Attach session ID to request context

#### Step 3: Save on convert
- After successful conversion, auto-save to DB
- Return `id` in the API response alongside existing fields
- No behavior change for users — conversion still works the same

#### Step 4: View shared pasta
- `GET /api/pastas/:id` — returns pasta by ID (public, no auth needed)
- Frontend route `/pasta/:id` — renders the shared art (read-only view)
- OG meta tags for link previews (stretch)

#### Step 5: My History
- `GET /api/pastas` — returns pastas for current session (cookie-based)
- Frontend "My Pastas" page — list of recent conversions
- Delete button (soft-delete or hard-delete, session-owner only)

#### Step 6: Publish toggle
- `PATCH /api/pastas/:id` — toggle `is_public` (session-owner only)
- "Publish to gallery" button in UI
- Prepares data for Phase 5 gallery

### API Changes

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/convert` | - | Existing + now returns `id` field |
| GET | `/api/pastas` | cookie | List my pastas (paginated) |
| GET | `/api/pastas/:id` | - | View any pasta by ID |
| PATCH | `/api/pastas/:id` | cookie | Update is_public (owner only) |
| DELETE | `/api/pastas/:id` | cookie | Delete pasta (owner only) |

### Volume & Retention
- Each pasta is ~1-50KB of text (braille art at max width)
- SQLite handles millions of rows easily
- Future: add TTL cleanup for old unpublished pastas (e.g., 90 days)

### Migration Path to Production DB
- SQLite works for single-instance deployment (our current setup)
- If we need multi-instance later, swap to PostgreSQL with same `Store` interface
- The interface pattern makes this a config change, not a rewrite

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

## Phase 6: Production Hardening — "Bulletproof It"
**Status**: Planned  
**Goal**: Harden for real traffic and add operational maturity.

| Task | Status |
|------|--------|
| Rate limiting middleware | ⬜ |
| Image size/format validation hardening | ⬜ |
| Graceful shutdown | ⬜ |
| Database migrations (golang-migrate) | ⬜ |
| Auto-scaling rules (Azure Container Apps) | ⬜ |
| CDN for static assets (Azure Front Door) | ⬜ |
| Error alerting (Azure Monitor / PagerDuty) | ⬜ |

---

## Stretch Ideas
- Animated GIF → animated ASCII
- Webcam → live ASCII stream
- Color ASCII (ANSI escape codes)
- ~~Braille character mode for higher resolution~~ ✅ Done
- ~~Custom character ramp editor~~ (partially done — ramp selection exists)
- Copy as image (render ASCII to PNG for platforms that mangle Unicode)
- OpenGraph preview images for shared links
