# 📊 Phase 4: Observability — Detailed Plan

## Current State

| What | Current | Target |
|------|---------|--------|
| Logging | `log.Printf` (unstructured, no levels) | JSON structured logs with slog |
| Access logs | `chimiddleware.Logger` (Apache-style) | Custom slog middleware with latency, status, request_id |
| Health check | `{"status":"ok"}` (no dependency checks) | DB ping, version, uptime, dependency status |
| Metrics | None | Prometheus counters + histograms |
| Dashboards | None | Grafana with pre-built panels |
| Request correlation | None | X-Request-ID propagation across logs (not distributed tracing) |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Azure Container Apps Environment (cae-copy-pasta)          │
│                                                             │
│  ┌──────────────┐  scrape /metrics       ┌──────────────┐   │
│  │  copy-pasta  │◄──────────────────────│  Prometheus  │   │
│  │  (Go app)    │                       │  :9090       │   │
│  │  :8080       │                       │  (internal)  │   │
│  └──────────────┘                       └──────┬───────┘   │
│                                                 │           │
│                                          ┌──────▼───────┐   │
│                                          │   Grafana    │   │
│                                          │   :3000      │   │
│                                          │  (external)  │   │
│                                          └──────────────┘   │
│                                                ▲            │
└────────────────────────────────────────────────┼────────────┘
                                                 │
                                           You (browser)
```

### Why Not Sidecars?

Azure Container Apps supports sidecar containers (multiple containers in one app). This is **not** suitable for Prometheus/Grafana because:

1. **Scaling conflict** — Sidecars share the main app's scaling rules. If `copy-pasta` scales to zero, a Prometheus sidecar stops scraping.
2. **Data fragmentation** — If the app scales to 3 replicas, you'd get 3 independent Prometheus instances with partial data.
3. **Resource contention** — Prometheus/Grafana would eat into the app's 0.25 CPU / 0.5Gi allocation.

Separate Container Apps give each service independent scaling, its own resources, and free internal DNS within the same environment.

---

## Production Hosting: Self-Hosted vs Azure Managed

| | Self-Hosted (Container Apps) | Azure Managed |
|---|---|---|
| **What** | Prometheus + Grafana as separate Container Apps in same environment | Azure Monitor Managed Prometheus + Azure Managed Grafana |
| **Cost** | ~$5-7/mo (0.25 CPU + 0.5Gi each, Prometheus always-on) | ~$10-12/mo (1 user at $9 + low ingestion at ~$1-3) |
| **For our usage** | ~$5-7/mo fixed | ~$10-12/mo |
| **Setup** | We deploy + configure ourselves | Azure provisions, patches, scales |
| **Learning value** | 🟢 High — understand scraping, PromQL, dashboards from scratch | 🟡 Medium — abstracted away |
| **Maintenance** | We handle upgrades, storage, config | Azure handles everything |
| **Data retention** | Limited by volume size (configurable) | 18 months included |
| **Custom dashboards** | Full control | Full control |
| **Persistence** | Azure Files volume mount | Managed (built-in) |
| **Scaling** | Manual (1 replica is fine for us) | Auto-scaled by Azure |
| **Network** | Same environment = internal communication, fast | Needs endpoint configuration |
| **Downside** | If Prometheus dies, you lose metrics until restart | Vendor lock-in, less educational |

### Decision: **Self-hosted** ✅

Reasons:
1. Primary goal is **learning** — we want to understand how the observability stack works end-to-end
2. Cost is comparable (~$5-7 vs ~$10-12/mo)
3. Everything stays in our Container Apps environment — simple networking
4. Can migrate to managed later if maintenance becomes a burden
5. Dashboard JSON is portable (works with both self-hosted and managed Grafana)

---

## Implementation Steps

### Step 1: Structured Logging with slog

Replace `log.Printf` with Go stdlib `log/slog` (available since Go 1.21, we're on 1.25).

**New files:**
- `backend/internal/logging/logging.go` — logger initialization (JSON in prod, colored text in dev)

**Modified files:**
- `backend/cmd/server/main.go` — init slog, replace all `log.Printf`
- `backend/internal/handler/handler.go` — replace `log.Printf` with `slog.Error`/`slog.Info`

**Design:**
- `LOG_FORMAT=json` (default in prod) or `LOG_FORMAT=text` (for local dev)
- `LOG_LEVEL=info` (default) | `debug` | `warn` | `error`
- Logger pulled from request context in handlers

### Step 2: Request ID Middleware

**New files:**
- `backend/internal/middleware/requestid.go` — generates UUID, sets `X-Request-ID` header, stores in context

**Behavior:**
- If incoming request has `X-Request-ID`, validate and reuse it (for correlation across services)
- **Validation:** max 64 characters, alphanumeric + hyphens only (`^[a-zA-Z0-9\-]{1,64}$`). Reject invalid values silently (generate a new ID instead).
- Otherwise generate a new UUID v4
- All log entries for that request include `request_id` field

### Step 3: Custom Access Log Middleware

**New files:**
- `backend/internal/middleware/requestlog.go` — replaces `chimiddleware.Logger`

**Modified files:**
- `backend/internal/middleware/middleware.go` — swap `chimiddleware.Logger` for custom

**Log output (JSON):**
```json
{
  "level": "INFO",
  "msg": "request completed",
  "request_id": "abc-123",
  "method": "POST",
  "path": "/api/convert",
  "status": 200,
  "latency_ms": 142,
  "bytes": 8423,
  "remote_addr": "10.0.0.1"
}
```

### Step 4: Rich Health Endpoint

**New files:**
- `backend/internal/handler/health.go` — extracted from handler.go

**Response:**
```json
{
  "status": "healthy",
  "version": "abc1234",
  "uptime_seconds": 8100,
  "checks": {
    "database": { "status": "up", "latency_ms": 3 }
  }
}
```

**Implementation:**
- DB ping with 2s timeout
- Version injected via `go build -ldflags "-X main.version=$(git rev-parse --short HEAD)"`; falls back to `APP_VERSION` env var (default: `dev`)
- Uptime tracked from `time.Now()` at startup
- Returns HTTP 503 if any critical check fails (useful for Container Apps health probes)

### Step 5: Prometheus Metrics

**New dependency:** `github.com/prometheus/client_golang` v1.23.2

**New files:**
- `backend/internal/metrics/metrics.go` — define and register all metrics
- `backend/internal/middleware/metrics.go` — HTTP metrics middleware
- `backend/internal/store/instrumented.go` — Store decorator for DB metrics

**Metrics exposed at `GET /metrics` (opt-in via `EXPOSE_METRICS=true`):**

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `http_requests_total` | Counter | method, route, status | Total HTTP requests |
| `http_request_duration_seconds` | Histogram | method, route | Request latency |
| `http_response_size_bytes` | Histogram | method, route | Response size |
| `conversions_total` | Counter | mode (ascii/braille) | Conversion count |
| `conversion_duration_seconds` | Histogram | mode | Conversion processing time |
| `db_operations_total` | Counter | operation, status | DB query count |
| `db_operation_duration_seconds` | Histogram | operation | DB query latency |

> **Note:** The `route` label uses the registered route pattern (e.g., `/api/pastas/{id}`), **not** the raw URL path. This prevents unbounded cardinality from dynamic path segments. Chi's `RouteContext` provides the pattern at middleware level.

**Endpoint security:**
- `/metrics` is **opt-in** — requires `EXPOSE_METRICS=true` env var to mount the handler
- Internal instrumentation (middleware, store decorator) runs regardless — the flag only controls HTTP endpoint exposure
- In production (Azure), Prometheus scrapes via internal DNS; the metrics endpoint is never exposed to the public internet
- Middleware skips recording metrics for the `/metrics` path itself (avoids inflating counts from Prometheus scrapes)

**DB instrumentation pattern:**
- `InstrumentedStore` wraps any `Store` implementation (decorator pattern)
- Records operation name, duration, and success/error status per call
- `ErrNotFound` is classified as "success" (expected business outcome, not an error)
- Exposes `Ping()` that delegates to inner store; returns `ErrPingNotSupported` sentinel if inner doesn't implement it (health check maps this to "unknown" status)

**Modified files:**
- `backend/cmd/server/main.go` — conditionally mount `/metrics` endpoint, wrap store with `NewInstrumented`
- `backend/internal/handler/handler.go` — instrument `Convert` endpoint with conversion timing
- `backend/internal/middleware/middleware.go` — add Metrics to middleware chain

### Step 6: Local Observability Stack (Docker Compose)

**New files:**
- `infra/prometheus/prometheus.yml` — scrape config
- `infra/grafana/provisioning/datasources/prometheus.yml` — auto-provision datasource
- `infra/grafana/provisioning/dashboards/dashboard.yml` — auto-load dashboards from disk
- `infra/grafana/dashboards/.gitkeep` — placeholder (dashboard JSON added in Step 8)

**Modified files:**
- `docker-compose.yml` — add `prometheus` + `grafana` services under `observability` profile

**Docker Compose additions:**
```yaml
prometheus:
  image: prom/prometheus:v2.53.0
  profiles: [observability]
  volumes:
    - ./infra/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
  ports:
    - "9090:9090"

grafana:
  image: grafana/grafana:11.1.0
  profiles: [observability]
  volumes:
    - ./infra/grafana/provisioning:/etc/grafana/provisioning
    - ./infra/grafana/dashboards:/var/lib/grafana/dashboards
  ports:
    - "3001:3000"
  environment:
    - GF_SECURITY_ADMIN_PASSWORD=${GF_SECURITY_ADMIN_PASSWORD:-changeme}
    - GF_AUTH_ANONYMOUS_ENABLED=false
    - GF_USERS_ALLOW_SIGN_UP=false
```

**Usage:**
```bash
# Base stack (no observability):
docker compose up

# With observability:
docker compose --profile observability up
```

> **Why profiles?** The base stack (`docker compose up`) works without any `.env` setup. Prometheus and Grafana are opt-in — users explicitly choose to start the observability stack. This avoids breaking the base dev workflow for contributors who don't need metrics.

> **Local access:** Grafana is mapped to port 3001 (avoids conflict with frontend's 3000). Open `http://localhost:3001`, login as `admin` with the password from your `.env` file (default: `changeme`).

### Step 7: Deploy to Azure (Self-Hosted)

**Modified files:**
- `infra/setup-azure.sh` — add Steps 7-8 for Prometheus + Grafana Container Apps

**Prometheus Container App:**
- Image: `prom/prometheus:v2.53.0`
- Ingress: **internal only** (not exposed to internet)
- Volume: Azure Files for data persistence
- Config: baked into a custom image (Dockerfile in `infra/prometheus/`)
- Min replicas: 1 (must always be running to scrape)
- CPU: 0.25, Memory: 0.5Gi

**Grafana Container App:**
- Image: custom (Dockerfile in `infra/grafana/`) with provisioning baked in
- Ingress: **external** (browser access)
- Min replicas: 0, Max: 1 (scale to zero when idle)
- CPU: 0.25, Memory: 0.5Gi
- Datasource: points to Prometheus internal URL (`http://prometheus:9090`)

**Grafana hardening:**
- `GF_SECURITY_ADMIN_PASSWORD` stored as Container Apps secret (40-char hex)
- `GF_AUTH_ANONYMOUS_ENABLED=false` (explicitly disable anonymous access)
- `GF_USERS_ALLOW_SIGN_UP=false` (no self-registration)
- Future upgrade path: Azure AD OAuth via `GF_AUTH_AZUREAD_*` or IP allowlist on ingress

**Grafana secret setup (in `infra/setup-azure.sh`):**
```bash
# Generate strong password
GRAFANA_PASSWORD=$(openssl rand -hex 20)

# Store as Container Apps secret
az containerapp secret set \
  --name grafana \
  --resource-group "$RESOURCE_GROUP" \
  --secrets gf-security-admin-password="$GRAFANA_PASSWORD"

# Map secret to Grafana's native env var
az containerapp update \
  --name grafana \
  --resource-group "$RESOURCE_GROUP" \
  --set-env-vars "GF_SECURITY_ADMIN_PASSWORD=secretref:gf-security-admin-password"
```

**Prometheus scrape config:**
```yaml
scrape_configs:
  - job_name: 'copy-pasta'
    scrape_interval: 15s
    static_configs:
      - targets: ['copy-pasta:8080']  # internal DNS in same Container Apps env
```

### Step 8: Pre-built Grafana Dashboard

**Panels:**
1. Request rate (req/s) by endpoint
2. Latency percentiles (p50, p95, p99) by endpoint
3. Error rate (4xx, 5xx)
4. Conversion count by mode (ascii vs braille)
5. Conversion latency histogram
6. DB operation latency
7. Health check status

---

## Implementation Order

```
Step 1 (slog) → Step 2 (request ID) → Step 3 (access log) → Step 4 (health)
    → Step 5 (metrics) → Step 6 (local stack) → Step 7 (Azure deploy) → Step 8 (dashboard)
```

**PR strategy:**
- PR 1: Steps 1-4 (logging + health) — no new deps
- PR 2: Steps 5-6 (metrics + local Prometheus/Grafana)
- PR 3: Steps 7-8 (Azure deployment + dashboard)

---

## Future: Distributed Tracing (Phase 4b)

When the app grows (async workers, multiple services), upgrade from request correlation to full distributed tracing:

- **OpenTelemetry Go SDK** — auto-instruments HTTP handlers and DB calls
- **W3C TraceContext** headers — propagates trace/span IDs across services
- **Jaeger or Grafana Tempo** — trace backend (Tempo integrates with our existing Grafana)
- **Request ID bridge** — our X-Request-ID becomes a correlation attribute on each trace

This is intentionally deferred: for a single-service app, Prometheus metrics + request correlation give 95% of the debugging value at 10% of the complexity.

---

## Dependencies

| Dependency | Purpose | Scope |
|------------|---------|-------|
| `log/slog` | Structured logging | stdlib (no new dep) |
| `github.com/prometheus/client_golang` | Metrics exposition | New Go dependency |
| `prom/prometheus` Docker image | Metrics collection | Dev + prod container |
| `grafana/grafana` Docker image | Dashboards | Dev + prod container |

---

## Environment Variables (New)

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_FORMAT` | `json` | `json` or `text` |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `APP_VERSION` | `dev` | Fallback if not injected via `-ldflags "-X ...handler.version=..."` at build time |
| `EXPOSE_METRICS` | (not set = disabled) | Set to `true` to mount the `/metrics` HTTP endpoint for Prometheus scraping |
| `GF_SECURITY_ADMIN_PASSWORD` | `changeme` (local) | Grafana admin login — in Azure, stored as Container Apps secret via `secretref:` |

---

## Cost Summary

| Component | Local | Azure (Self-Hosted) |
|-----------|-------|---------------------|
| Prometheus | Free (docker) | ~$3-5/mo (always-on, 0.25 CPU) |
| Grafana | Free (docker) | ~$1-2/mo (scales to zero) |
| App metrics code | Free | Free |
| **Total added** | **$0** | **~$5-7/mo** |

---

## Security Considerations

### Credential Management

| Environment | Where password lives | How it's set |
|-------------|---------------------|--------------|
| **Local dev** | `.env` file (git-ignored) | `GF_SECURITY_ADMIN_PASSWORD=yourpassword` — docker-compose reads `.env` automatically. Default is `changeme`. |
| **Azure prod** | Container Apps secret (encrypted at rest) | `az containerapp secret set` → mapped via `secretref:` to env var |

### Metrics Endpoint Exposure

| Environment | Strategy |
|-------------|----------|
| **Local dev** | Set `EXPOSE_METRICS=true` in `.env` when running with `--profile observability` |
| **Azure prod** | Backend exposes `/metrics` only on internal ingress; Prometheus scrapes internally. Never exposed to public internet. |

### Current vs Production-Grade

| Concern | Our approach (sufficient for learning) | Production-grade alternative |
|---------|---------------------------------------|------------------------------|
| Password storage | Container Apps secrets | Azure Key Vault with managed identity |
| Access control | Password-only login | Azure AD OAuth (Grafana supports `GF_AUTH_AZUREAD_*` natively) |
| Network exposure | External ingress + password | Internal-only ingress + VPN / Azure Front Door with IP allowlist |
| Password rotation | Manual (re-run script) | Key Vault rotation policy |
| Prometheus access | Internal ingress (not exposed) | ✅ Already correct |

> **Why this is fine for now:** Grafana only exposes operational metrics (request rates, latency) — no user PII. The attack surface is limited to someone guessing a 40-char hex password on an obscure URL. If we ever expose sensitive data, add Azure AD OAuth as the first upgrade.
