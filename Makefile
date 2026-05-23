.PHONY: dev dev-backend dev-frontend build docker-up docker-up-obs docker-down clean

# Run both backend and frontend in dev mode
dev:
	@echo "Starting backend and frontend in dev mode..."
	@make -j2 dev-backend dev-frontend

dev-backend:
	cd backend && go run ./cmd/server

dev-frontend:
	cd frontend && npm run dev

# Build both projects
build:
	cd backend && go build -o bin/server ./cmd/server
	cd frontend && npm run build

# Docker commands
docker-up:
	docker compose up --build

docker-down:
	docker compose down

docker-up-obs:
	docker compose --profile observability up --build

docker-up-d:
	docker compose up --build -d

# Testing
test:
	cd backend && go test ./...

# Linting
lint:
	cd backend && go vet ./...
	cd frontend && npm run lint

# Clean build artifacts
clean:
	rm -rf backend/bin
	rm -rf frontend/dist
