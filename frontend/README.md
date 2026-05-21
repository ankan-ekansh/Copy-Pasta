# Copy-Pasta Frontend

React + TypeScript + Vite frontend for the Copy-Pasta meme-to-ASCII converter.

## Development

```bash
npm install
npm run dev      # Starts Vite dev server on :5173 (proxies /api to :8080)
```

## Build

```bash
npm run build    # Outputs to dist/ (served by Go backend in production)
```

## Key Components

- **`App.tsx`** — Main layout, mode toggle, share presets, width controls
- **`components/ImageUploader.tsx`** — Drag-and-drop, file picker, clipboard paste (Ctrl+V)
- **`components/AsciiOutput.tsx`** — Rendered output with copy-to-clipboard
- **`components/ThemeToggle.tsx`** — Dark/light mode switch with localStorage persistence
- **`api/convert.ts`** — API client for `POST /api/convert`

## Tech

- React 19
- TypeScript
- Vite (with `/api` proxy to backend in dev)
- No CSS framework — custom CSS with CSS variables for theming

