# 🍝 Copy-Pasta

> Turn memes into ASCII art — paste, convert, copy, share!

A fun web app that converts meme images into ASCII art that you can copy-paste anywhere. Built with Go and React as a learning playground for backend development, frontend polish, and progressive productionization.

## ✨ Features

- **Image to ASCII**: Upload or paste any image (JPEG, PNG, GIF) and get ASCII art
- **Configurable output**: Adjust width, invert colors, choose character sets
- **One-click copy**: Copy your ASCII masterpiece to clipboard instantly
- **Fast**: Server-side Go conversion is blazing fast

## 🏗️ Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go + Chi router |
| Frontend | React + TypeScript + Vite |
| Infra | Docker Compose |
| ASCII Engine | Go `image` + `golang.org/x/image` |

## 🚀 Quick Start

### Prerequisites
- Go 1.22+
- Node.js 20+
- Docker & Docker Compose (optional)

### Local Development

```bash
# Run both services (requires make -j support)
make dev

# Or run individually:
make dev-backend   # Go server on :8080
make dev-frontend  # Vite dev server on :5173 (proxies /api to :8080)
```

### Docker

```bash
make docker-up     # Build and start all services
make docker-down   # Stop all services
```

Then open http://localhost:3000

## 📁 Project Structure

```
Copy-Pasta/
├── backend/
│   ├── cmd/server/         # Entry point
│   ├── internal/
│   │   ├── handler/        # HTTP handlers
│   │   ├── converter/      # Image → ASCII logic
│   │   └── middleware/     # CORS, logging
│   ├── go.mod
│   └── Dockerfile
├── frontend/
│   ├── src/
│   │   ├── components/     # React components
│   │   ├── api/            # API client
│   │   └── App.tsx
│   ├── package.json
│   └── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

## 🗺️ Roadmap

- [x] **Phase 1**: Foundation — end-to-end image → ASCII flow
- [ ] **Phase 2**: Polish — clipboard paste, drag-drop, controls, themes
- [ ] **Phase 3**: Persistence — PostgreSQL, history, shareable URLs
- [ ] **Phase 4**: Observability — logging, tracing, metrics, Grafana
- [ ] **Phase 5**: Social — gallery, likes, leaderboard
- [ ] **Phase 6**: Production — CI/CD, cloud deployment, hardening

## 🤝 Contributing

This is a personal learning project, but PRs and ideas are welcome!

## 📝 License

MIT
