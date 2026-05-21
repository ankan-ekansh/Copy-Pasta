# 🍝 Copy-Pasta

> Turn memes into ASCII art — paste, convert, copy, share!

**Live**: https://copy-pasta.happyflower-831a5c58.eastus.azurecontainerapps.io

A fun web app that converts meme images into ASCII art that you can copy-paste anywhere. Built with Go and React as a learning playground for backend development, frontend polish, and progressive productionization.

## ✨ Features

- **Image to ASCII**: Upload, paste (Ctrl+V), or drag-and-drop any image (JPEG, PNG, GIF)
- **Braille Unicode mode**: High-resolution conversion using Unicode Braille characters (8 pixels per char)
- **Floyd-Steinberg dithering**: Simulates grayscale through dot density patterns
- **Edge detection**: Sobel operator preserves outlines at narrow widths
- **Mobile share presets**: One-tap widths for WhatsApp, iMessage, Twitter, Telegram, Discord, Reddit
- **Dark/light theme**: Toggle with system preference detection and localStorage persistence
- **Configurable output**: Width slider, invert, A/B mode toggle (ASCII vs Braille)
- **One-click copy**: Copy your ASCII masterpiece to clipboard instantly
- **Fast**: Server-side Go conversion is blazing fast

## 🏗️ Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go + Chi router |
| Frontend | React + TypeScript + Vite |
| Database | Azure Database for PostgreSQL (Phase 3, planned) |
| Infra | Docker Compose (dev), Azure Container Apps (prod) |
| CI/CD | GitHub Actions (test on PR, deploy on merge to main) |
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
# Single unified container (same as production)
docker build -t copy-pasta .
docker run -p 8080:8080 copy-pasta
```

Then open http://localhost:8080

## 📁 Project Structure

```
Copy-Pasta/
├── backend/
│   ├── cmd/server/         # Entry point (serves API + static frontend)
│   └── internal/
│       ├── handler/        # HTTP handlers
│       ├── converter/      # Image → ASCII/Braille engine
│       └── middleware/     # CORS, logging
├── frontend/
│   ├── src/
│   │   ├── components/     # React components (ImageUploader, AsciiOutput, ThemeToggle)
│   │   ├── api/            # API client
│   │   └── App.tsx         # Main app with controls, presets, mode toggle
│   └── index.html
├── infra/
│   └── setup-azure.sh     # Azure infrastructure provisioning (idempotent)
├── .github/workflows/
│   └── deploy.yml          # CI/CD: test → build → deploy
├── docs/
│   ├── PLAN.md             # Phase-by-phase roadmap
│   ├── IMPLEMENTATION.md   # Architecture & API reference
│   └── DEPLOYMENT.md       # Azure deployment guide
├── Dockerfile              # Multi-stage: Node build → Go build → Alpine runtime
├── docker-compose.yml      # Local dev with hot reload
├── Makefile
├── CONTRIBUTING.md         # Branch/PR workflow rules
└── README.md
```

## 🗺️ Roadmap

- [x] **Phase 1**: Foundation — end-to-end image → ASCII flow
- [x] **Phase 2**: Polish — paste, drag-drop, braille mode, dithering, presets, dark/light theme
- [x] **Phase 2.5**: Deploy — Azure Container Apps, CI/CD, GitHub Actions
- [ ] **Phase 3**: Persistence — PostgreSQL, history, shareable URLs
- [ ] **Phase 4**: Observability — logging, tracing, metrics
- [ ] **Phase 5**: Social — gallery, likes, leaderboard
- [ ] **Phase 6**: Production hardening — rate limiting, scaling, CDN

See **[docs/PLAN.md](docs/PLAN.md)** for detailed task tracking.

## 📚 Documentation

- **[Project Plan](docs/PLAN.md)** — Phase-by-phase roadmap with task tracking
- **[Implementation Guide](docs/IMPLEMENTATION.md)** — Architecture, API reference, design decisions
- **[Deployment Guide](docs/DEPLOYMENT.md)** — Azure setup, CI/CD, monitoring
- **[Contributing](CONTRIBUTING.md)** — Branch strategy, PR workflow, commit conventions

## 📝 License

MIT
