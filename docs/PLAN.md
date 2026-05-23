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
**Status**: Complete (minor stretch items deferred)  
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

**Live URL**: https://copy-pasta.salmondesert-9297c655.centralindia.azurecontainerapps.io

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

## Phase 3: Persistence — "Remember the Pastas" ✅
**Status**: Complete  
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
  id TEXT PRIMARY KEY,            -- crypto/rand base62, 10 chars
  session_id TEXT NOT NULL,       -- links to creator's cookie
  ascii_art TEXT NOT NULL,        -- the generated output
  width INT NOT NULL,
  height INT NOT NULL,
  mode TEXT NOT NULL,             -- 'ascii' or 'braille'
  is_public BOOLEAN NOT NULL DEFAULT FALSE, -- opt-in for future gallery (Phase 5)
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pastas_session ON pastas(session_id);
CREATE INDEX idx_pastas_public ON pastas(is_public, created_at);
```

### Implementation Steps (gradual)

#### Step 1: Database layer ✅
- ~~Provision Azure Database for PostgreSQL Flexible Server (B1ms, 32GB)~~ (deferred to deploy time)
- Add `pgx` driver (Go PostgreSQL driver)
- Create `internal/store` package with `Store` interface
- Run migration on startup (create table if not exists)
- Wire into server startup via `DATABASE_URL` env var
- PostgreSQL 16 in docker-compose for local dev

#### Step 2: Session cookie middleware ✅
- Middleware checks for `copy-pasta-session` cookie
- If missing, generate UUID and set cookie (HttpOnly, Secure via X-Forwarded-Proto, SameSite=Lax, Path=/, 1 year expiry)
- Attach session ID to request context

#### Step 3: Save on convert ✅
- After successful conversion, auto-save to DB (best-effort, doesn't fail request)
- Return `id` in the API response alongside existing fields
- No behavior change for users — conversion still works the same

#### Step 4: View shared pasta ✅
- `GET /api/pastas/:id` — returns pasta by ID
- **Visibility model**: all pastas are "unlisted but shareable" — anyone with the link can view regardless of `is_public`. The `is_public` flag only controls whether the pasta appears in the Phase 5 gallery/browse feed.
- Frontend route `/pasta/:id` — renders the shared art (read-only view) ✅
- OG meta tags for link previews (stretch) ⬜

#### Step 5: My History ✅
- `GET /api/pastas` — returns pastas for current session (cookie-based, paginated)
- `DELETE /api/pastas/:id` — atomic ownership check (session_id in WHERE)
- Frontend "My Pastas" panel — list of recent conversions with share/delete ✅

#### Step 6: Set visibility ✅
- `PATCH /api/pastas/:id` — sets `is_public` to provided value, atomic ownership check (session_id in WHERE)
- "Publish to gallery" button in UI ⬜ (deferred to Phase 5)
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
**Status**: In Progress  
**Goal**: Structured logging, metrics, dashboards, and request correlation (X-Request-ID).

| Task | Status |
|------|--------|
| Structured logging (slog) | ✅ PR #15 |
| Request ID middleware | ✅ PR #15 |
| Rich health endpoint (DB check, version, uptime) | ✅ PR #15 |
| Prometheus metrics (`/metrics`) | ✅ PR #16 |
| Local observability stack (docker-compose) | ✅ PR #16 |
| Production Prometheus + Grafana | ⬜ |
| Pre-built Grafana dashboard | ⬜ |
| Documentation | ✅ PR #14 |

See **[OBSERVABILITY.md](OBSERVABILITY.md)** for the detailed implementation plan.

---

## Phase 4c: Rate Limiting — "Don't Get Spammed"
**Status**: Planned  
**Goal**: Protect CPU-intensive conversion endpoint from abuse without impeding legitimate users.

### Options Evaluated

| Option | Algorithm | Pros | Cons | Verdict |
|--------|-----------|------|------|---------|
| **`go-chi/httprate`** | Sliding window, in-memory | Chi-native, auto IP cleanup, rate-limit headers, per-route scoping | One dep; in-memory resets on restart | ✅ Chosen |
| `golang.org/x/time/rate` | Token bucket, in-memory | Zero deps, stdlib-adjacent | Manual IP map management, cleanup goroutine, no headers | ❌ More code for same result |
| Redis + sliding window | Centralized counter | Accurate across replicas, survives restarts | Adds Redis infra, overkill at max 3 replicas | ❌ Over-engineered |
| Azure Front Door / API Mgmt | Edge throttling | Zero code, handles DDoS | ~$35/mo min, vendor lock-in, out-of-repo config | ❌ Too costly for toy project |
| nginx `limit_req` | Leaky bucket at proxy | Battle-tested | No standalone proxy in prod (Container Apps ingress) | ❌ Infeasible in prod |

### Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Library | `go-chi/httprate` | First-party Chi middleware, handles all boilerplate |
| Scope | Per-IP | Simple, effective for single-origin abuse |
| Convert limit | 10 req/min | CPU-heavy; 10/min is generous for real use |
| General API limit | 100 req/min | Reads are cheap; protect against scraping |
| Configuration | Env vars (`RATE_LIMIT_CONVERT`, `RATE_LIMIT_API`) | Tunable per environment without redeploy |
| Multi-replica gap | Accepted | Max 3 replicas × 10 = 30 worst case; ceiling is low |
| Response | 429 + JSON + standard headers | `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `Retry-After` |

### Implementation Steps

| Step | Task | Status |
|------|------|--------|
| 1 | Add `go-chi/httprate` dependency to `backend/go.mod` | ⬜ |
| 2 | Create `backend/internal/middleware/ratelimit.go` — factory reading env vars | ⬜ |
| 3 | Create `backend/internal/middleware/ratelimit_test.go` — under/over limit, per-IP isolation | ⬜ |
| 4 | Wire in `backend/cmd/server/main.go` — stricter on convert, general on all API | ⬜ |
| 5 | Update `.env.example` with new vars | ⬜ |
| 6 | Update `docs/ARCHITECTURE.md` footguns table | ✅ |

### Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Shared IP (NAT/VPN) hits limit | 10/min is generous; tunable via env var |
| Attacker rotates IPs | Damage ceiling is low (max 3 replicas × CPU cap) |
| State lost on restart | Acceptable — no persistent abuse tracking needed |
| X-Forwarded-For spoofing | httprate uses rightmost non-private IP; Azure sets real client IP |

---

## Phase 4d: Distributed Tracing — "See the Waterfall" (Stretch)
**Status**: Future  
**Goal**: Add OpenTelemetry tracing for end-to-end request visibility (frontend → backend → DB).

| Task | Status |
|------|--------|
| OpenTelemetry SDK integration (Go) | ⬜ |
| Trace propagation (W3C TraceContext headers) | ⬜ |
| DB span instrumentation | ⬜ |
| Jaeger or Tempo as trace backend | ⬜ |
| Trace → Request ID correlation | ⬜ |

> **Note**: Phase 4 adds X-Request-ID correlation, which covers our current needs. Distributed tracing becomes valuable when we add async workers, multiple services, or need per-request latency breakdowns. The request IDs from Phase 4 will serve as correlation keys in traces.

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
