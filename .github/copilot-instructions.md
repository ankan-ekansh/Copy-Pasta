# Copilot Instructions — Copy-Pasta

## Project Overview

Copy-Pasta is a Go (backend) + React (frontend) web app that converts meme images into ASCII/Braille art. It's deployed on Azure Container Apps with PostgreSQL for persistence and Prometheus/Grafana for observability.

---

## Code Style & Conventions

### Go Backend (`backend/`)

- **Go version:** 1.25 (module-level)
- **Router:** Chi (`github.com/go-chi/chi/v5`)
- **Logging:** `log/slog` only — never `log.Printf` or `fmt.Printf` for operational output
- **Error handling:**
  - Use `errors.Is()` for sentinel checks (handles wrapped errors)
  - Never expose raw error messages in HTTP responses — log server-side, return generic messages
  - Return appropriate HTTP status codes (400 for bad input, 503 for degraded health)
- **Naming:** Go standard — `camelCase` locals, `PascalCase` exports, package names are short lowercase nouns
- **Testing:** Use stdlib `testing` package. No testify or third-party test frameworks.

### React Frontend (`frontend/`)

- **Framework:** React 19 + Vite
- **Language:** TypeScript
- **Styling:** Plain CSS (no framework)
- **State:** React hooks (no external state management)
- **Linting:** ESLint (`npm run lint`)

### General

- Only comment code that needs clarification — don't over-comment obvious logic
- Keep functions focused and small
- Prefer returning early over deep nesting

---

## Architecture Rules

- **Metrics labels:** Never use raw URL paths as Prometheus labels (unbounded cardinality). Use route patterns from `chi.RouteContext().RoutePattern()`.
- **Middleware:** Always nil-guard `chi.RouteContext()` — can be nil outside the router.
- **Store pattern:** Use interfaces for persistence. The `InstrumentedStore` decorator wraps any `Store` implementation to add metrics without modifying the original.
- **Health endpoint:** Report "unknown" (not "down") when a capability isn't available or testable.
- **Docker Compose:** 
  - Pin all image versions (no `:latest`)
  - Bind dev-only ports to `127.0.0.1`
  - Use `profiles` for optional services
  - Docker Compose interpolates ALL env vars regardless of profiles — `${VAR:?}` breaks the base stack
  - Bare env references (`- MY_VAR`) pass through from `.env` only when set

---

## Known Footguns & Constraints

These are architectural choices that can bite you if you're unaware:

| Area | Constraint | Why it matters |
|------|-----------|----------------|
| Upload size | Hardcoded 20MB (`MaxBytesReader` in handler.go) | No env override — changing requires code change + deploy |
| CORS | Origins hardcoded + `CORS_ORIGINS` env override | New domains require updating the default list or setting the env var |
| Session cookie | 1-year expiry, `Secure` only when TLS detected | On plain HTTP (local dev), cookie is not Secure — fine for dev, not for prod without TLS |
| Store.Ping() | Not part of the `Store` interface — exposed via type assertion | Adding a new Store implementation? Must also add `Ping()` or health reports "unknown" |
| Rate limiting | None | Conversion endpoint is CPU-intensive — no protection against abuse |
| API prefix | Frontend assumes all backend routes are under `/api/` | Never mount handlers outside `/api/` (except `/metrics`) |
| SPA routing | nginx `try_files` falls back to `index.html` | Backend's `NotFound` handler also does SPA fallback — don't add catch-all routes |
| InstrumentedStore | Must wrap ALL Store methods | If `Store` interface gains a new method, `InstrumentedStore` must be updated or it won't compile |

---

## Pre-Push Discipline

### 1. Trace the Full Data Flow

For every config value you add, modify, or remove, trace its complete path:

- **Source** → Where is it defined? (`.env`, code default, Azure secret)
- **Transport** → How does it reach the consumer? (docker-compose passthrough, build arg, volume mount)
- **Consumer** → What reads it? (Go `os.Getenv`, Grafana env, Prometheus config)
- **Verify:** "If I change/remove this line, does the value still arrive at the consumer?"

### 2. Think in Systems, Not Lines

When addressing a concern (from a reviewer or self-found):

1. Identify the **design constraint** (e.g., "opt-in means opt-in everywhere")
2. List **every file** where that constraint applies
3. Apply the change **consistently across ALL of them** in one commit
4. Mentally walk through the **end-to-end user scenario** to verify it works

### 3. State What's Infeasible

If a suggestion can't be implemented due to a technical limitation:

- **Reply with evidence** — error messages, tested behavior, documentation links
- **Document the limitation** in the relevant docs for future contributors
- **Explain the chosen alternative** and why it's the best option given the constraint

### 4. Documentation Must Match Code

Every code change should trigger a doc check:

- Do env var tables match actual env vars in code?
- Do file paths in docs point to files that exist?
- Do code snippets in docs match the actual implementation?
- Do usage instructions (commands, credentials, ports) reflect current reality?

### 5. Self-Review Before Pushing

Simulate a reviewer reading your diff:

- Read every changed file **top-to-bottom** (not just your edit)
- For each env var: trace source → transport → consumer
- For each file referenced in docs: verify it exists
- For each behavioral claim: verify with a test or mental walkthrough
- Ask: "What would break if someone followed these docs exactly?"

---

## Build & Run

```bash
# Backend (local dev)
cd backend && go run ./cmd/server

# Frontend (local dev)
cd frontend && npm run dev

# Full stack via Docker Compose
docker compose up

# With observability (requires EXPOSE_METRICS=true in .env)
docker compose --profile observability up

# Run backend tests
cd backend && go test ./...

# Lint frontend
cd frontend && npm run lint
```

---

## Key Files

| File | Purpose |
|------|---------|
| `backend/cmd/server/main.go` | Entry point, route registration |
| `backend/internal/handler/` | HTTP handlers |
| `backend/internal/middleware/` | Chi middleware chain |
| `backend/internal/store/` | Persistence interface + PostgreSQL + InstrumentedStore |
| `backend/internal/metrics/` | Prometheus metric definitions |
| `backend/internal/converter/` | Image → ASCII/Braille conversion |
| `frontend/src/` | React application |
| `docker-compose.yml` | Local dev stack |
| `infra/` | Prometheus, Grafana, Azure deployment configs |
| `docs/PLAN.md` | Phase-by-phase roadmap |
| `docs/OBSERVABILITY.md` | Observability architecture and decisions |

