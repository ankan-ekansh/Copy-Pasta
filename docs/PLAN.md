# 🗺️ Copy-Pasta — Project Plan

## Vision
A fun web app where users paste/upload meme images and get ASCII art back that they can copy-paste anywhere. Built as a progressive learning playground.

---

## Cross-cutting: Testing

**Status**: In progress  
**Goal**: Meaningful test coverage, introduced incrementally alongside features.

| Task | Status |
|------|--------|
| Converter unit tests (Step 1) | ✅ |
| Handler integration tests (Step 2) | ⬜ |
| Helper/utility tests (Step 3) | ✅ |
| Frontend tests with Vitest (Step 4) | ⬜ |

See **[TESTING.md](TESTING.md)** for the full testing plan.

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
- **PostgreSQL** — Azure Database for PostgreSQL Flexible Server (B1ms tier, free 12 months)
- **No file storage** — only persist the generated ASCII text, not source images
- **Shareable links** — each conversion gets a short ID, viewable by anyone
- **Ownership via cookie** — creator can manage their art while cookie persists
- **Multi-replica safe** — PostgreSQL supports concurrent connections from 0-3 replicas

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
- Provision Azure Database for PostgreSQL Flexible Server (B1ms, 32GB)
- Add `pgx` driver (Go PostgreSQL driver)
- Create `internal/store` package with `Store` interface
- Run migration on startup (create table if not exists)
- Wire into server startup via `DATABASE_URL` env var

#### Step 2: Session cookie middleware
- Middleware checks for `copy-pasta-session` cookie
- If missing, generate UUID and set cookie (HttpOnly, Secure, SameSite=Lax, Path=/, 1 year expiry)
- Attach session ID to request context

#### Step 3: Save on convert
- After successful conversion, auto-save to DB
- Return `id` in the API response alongside existing fields
- No behavior change for users — conversion still works the same

#### Step 4: View shared pasta
- `GET /api/pastas/:id` — returns pasta by ID
- **Visibility model**: all pastas are "unlisted but shareable" — anyone with the link can view regardless of `is_public`. The `is_public` flag only controls whether the pasta appears in the Phase 5 gallery/browse feed.
- Frontend route `/pasta/:id` — renders the shared art (read-only view)
- OG meta tags for link previews (stretch)

#### Step 5: My History
- `GET /api/pastas` — returns pastas for current session (cookie-based)
- Frontend "My Pastas" page — list of recent conversions
- Hard-delete only (no soft-delete complexity; deleted = gone from DB)

#### Step 6: Publish toggle
- `PATCH /api/pastas/:id` — toggle `is_public` (session-owner only)
- "Publish to gallery" button in UI
- Prepares data for Phase 5 gallery

### API Changes

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/convert` | - | Existing + now also returns `id` field (Phase 3 addition) |
| GET | `/api/pastas` | cookie | List my pastas (paginated) |
| GET | `/api/pastas/:id` | - | View any pasta by ID (unlisted-but-shareable) |
| PATCH | `/api/pastas/:id` | cookie | Update is_public (owner only) |
| DELETE | `/api/pastas/:id` | cookie | Delete pasta (owner only) |

### Volume & Retention
- Each pasta is ~1-50KB of text (braille art at max width)
- PostgreSQL 32GB storage handles millions of rows easily
- Future: add TTL cleanup for old unpublished pastas (e.g., 90 days)

### Infrastructure
- **Azure Database for PostgreSQL Flexible Server** (Burstable B1ms)
  - 1 vCore, 2GB RAM, 32GB storage
  - Free for 12 months with Azure account
  - ~$13/mo after free period (covered by company credits)
- Container App keeps 0-3 replicas (no scaling restriction)
- Connection via `DATABASE_URL` environment variable (set as Container App secret)
- Update `infra/setup-azure.sh` to provision PostgreSQL server
- Add `DATABASE_URL` to GitHub Actions secrets for CI/CD

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
