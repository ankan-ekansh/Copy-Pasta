# Architecture — Copy-Pasta

## System Overview

Copy-Pasta converts meme images into ASCII/Braille art. The stack:

```
┌─────────────┐     ┌─────────────────┐     ┌────────────┐
│  React SPA  │────▶│  Go API (Chi)   │────▶│ PostgreSQL │
│  (Vite/TS)  │     │  /api/* routes  │     │            │
└─────────────┘     └─────────────────┘     └────────────┘
      nginx              │
   (port 3000)           ▼
                    ┌──────────┐     ┌─────────┐
                    │Prometheus│────▶│ Grafana │
                    │ (9090)   │     │ (3001)  │
                    └──────────┘     └─────────┘
                    (profile: observability)
```

## Request Flow

1. User uploads image via frontend (`POST /api/convert`)
2. Backend validates size (20MB max), decodes image
3. Converter transforms pixels → Braille Unicode characters
4. Result stored in PostgreSQL (session-scoped ownership)
5. User can list, share (make public), or delete their pastas
6. Public pastas appear in the gallery (`GET /api/gallery`) with like counts
7. Users can like/unlike public pastas; likes are cleared atomically on unpublish

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Chi router | Lightweight, stdlib-compatible, context-based routing |
| Store interface | Enables swapping persistence (in-memory for tests, Postgres for prod) |
| InstrumentedStore decorator | Adds metrics without modifying Store implementations |
| Session via cookie (no auth) | MVP simplicity — no login required, ownership via browser session |
| Compose profiles | Keep base stack minimal; observability is opt-in |
| Braille conversion | Higher fidelity than traditional ASCII — each char encodes 2×4 pixel block |
| CTE pagination for gallery | Separates row selection from aggregation; cleaner query, predictable performance |
| Transactional unpublish | Atomic unpublish + clear likes ensures consistency — no window where likes exist on a non-public pasta |
| SQL-enforced public gate | `INSERT...SELECT WHERE is_public=TRUE` — impossible to like a non-public pasta even with race conditions |
| useRef for in-flight guards | Synchronous check prevents double-clicks without waiting for React re-render |

---

## Known Footguns & Constraints

These are architectural choices that can bite you if you're unaware. **Keep this table updated when changing related code.**

| Area | Constraint | Why it matters |
|------|-----------|----------------|
| Upload size | Hardcoded 20MB (`MaxBytesReader` in handler.go) | No env override — changing requires code change + deploy |
| CORS | Origins hardcoded + `CORS_ORIGINS` env override | New domains require updating the default list or setting the env var |
| Session cookie | 1-year expiry, `Secure` only when TLS detected | On plain HTTP (local dev), cookie is not Secure — fine for dev, not for prod without TLS |
| Store.Ping() | Not part of the `Store` interface — exposed via type assertion | Adding a new Store implementation? Must also add `Ping()` or health reports "unknown" |
| Rate limiting | In-memory, per-IP via `go-chi/httprate` | 10 req/min on `/api/convert`, 100 req/min on general API routes (separate groups — not double-applied). Resets on restart. Not shared across replicas. Requires `TRUSTED_PROXY=true` when behind a proxy; keys on `RemoteAddr` otherwise. |
| API prefix | Frontend assumes all backend routes are under `/api/` | Never mount handlers outside `/api/` (except `/metrics`) |
| SPA routing | nginx `try_files` falls back to `index.html` | Backend's `NotFound` handler also does SPA fallback — don't add catch-all routes |
| InstrumentedStore | Must wrap ALL Store methods | If `Store` interface gains a new method, `InstrumentedStore` must be updated or it won't compile |
| Likes & public gating | Only public pastas can be liked (`INSERT...SELECT WHERE is_public=TRUE`) | Unpublishing clears all likes in same transaction; unlike checks `IsPublicPasta` first |
| SetPublicByOwner | Wrapped in a transaction (UPDATE + DELETE likes on unpublish) | Adding logic between the two statements can break atomicity |
| Gallery pagination | CTE-based with LEFT JOIN on likes | Changing the GROUP BY or ORDER BY requires matching CTE columns |
| Docker Compose env | Interpolates ALL env vars regardless of active profiles | `${VAR:?}` syntax breaks `docker compose config` even for inactive-profile services |
| .env passthrough | Bare `- MY_VAR` in compose only works if var is set | Removing the mapping from docker-compose.yml breaks the chain even if .env has the value |

---

## Module Boundaries

```
backend/
├── cmd/server/          # Wiring & startup only — no business logic
├── internal/
│   ├── converter/       # Pure: image → art (no I/O, no state)
│   ├── handler/         # HTTP layer: parse request, call converter/store, write response
│   │                    #   handler.go (convert), pastas.go, gallery.go (gallery + likes)
│   ├── middleware/      # Cross-cutting: CORS, sessions, metrics, logging
│   ├── metrics/         # Prometheus metric definitions (counters, histograms)
│   └── store/           # Persistence interface + implementations
└── Dockerfile

frontend/
├── src/                 # React components, pages, API client
├── nginx.conf           # Production static file serving + SPA fallback
└── Dockerfile           # Multi-stage: build with Node, serve with nginx
```

### Rules

- `converter/` must remain pure — no database, no HTTP, no global state
- `handler/` may call `converter/` and `store/` but not `middleware/`
- `middleware/` must not import from `handler/` or `store/`
- `metrics/` defines metrics only — recording happens in `middleware/` and `store/`
