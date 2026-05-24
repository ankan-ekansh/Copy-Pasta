# =============================================================================
# Copy-Pasta — Unified Multi-Stage Dockerfile
# Builds both frontend and backend, serves from a single container.
# =============================================================================

# --- Stage 1: Build React Frontend ---
FROM node:20-alpine AS frontend-build
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --quiet
COPY frontend/ ./
RUN npm run build

# --- Stage 2: Build Go Backend ---
FROM golang:1.25-alpine AS backend-build
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# --- Stage 3: Production Runtime ---
FROM alpine:3.19 AS production
RUN apk --no-cache add ca-certificates font-dejavu
WORKDIR /app

# Copy the compiled binary
COPY --from=backend-build /server ./server

# Copy the built frontend into static/
COPY --from=frontend-build /app/frontend/dist ./static/

# Non-root user for security
RUN adduser -D -u 1000 appuser
USER appuser

EXPOSE 8080
ENV PORT=8080
ENV STATIC_DIR=static

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget -qO- http://localhost:8080/api/health || exit 1

ENTRYPOINT ["./server"]
