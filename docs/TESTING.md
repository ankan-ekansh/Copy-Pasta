# Testing Plan

## Overview

Testing is introduced incrementally, starting with the highest-value, easiest-to-test code (pure functions in the converter) and expanding outward to HTTP handlers and eventually the frontend.

**Guiding principles:**
- Test behavior, not implementation details
- Prefer table-driven tests (Go convention)
- No mocking frameworks — use Go interfaces and `httptest`
- Keep tests fast (< 2s total)
- CI already runs `go test ./...` — adding test files is immediately enforced

---

## Step 1: Converter Unit Tests (highest value)

**Package:** `backend/internal/converter`

The converter has pure functions with deterministic output — ideal for unit testing.

### Tests to write:

| Test | What it verifies |
|------|-----------------|
| `TestConvert_NilImage` | Returns empty string for nil input |
| `TestConvert_EmptyImage` | Returns empty string for 0×0 image |
| `TestConvert_DefaultWidth` | Uses `DefaultWidth` (150) when width ≤ 0 |
| `TestConvert_OutputDimensions` | Output has expected rows/cols for known input size |
| `TestConvert_AllWhite` | All-white image produces lightest chars (spaces or dots) |
| `TestConvert_AllBlack` | All-black image produces densest chars (@, #, etc.) |
| `TestConvert_Invert` | Inverted output is inverse of non-inverted |
| `TestConvert_CustomRamp` | Custom char ramp is respected |
| `TestConvert_WidthClamping` | Output line length equals requested width |
| `TestConvertBraille_NilImage` | Returns empty string for nil |
| `TestConvertBraille_DefaultWidth` | Uses 80 when width ≤ 0 |
| `TestConvertBraille_OutputIsBraille` | All output chars are in U+2800–U+28FF range |
| `TestConvertBraille_Dithering` | Dithered output differs from non-dithered |
| `TestConvertBraille_Invert` | Invert flag produces different output |
| `TestOtsuThreshold` | Returns ~0.5 for evenly-split bimodal grid |
| `TestFloydSteinbergDither` | Output pixels are pushed to 0 or 1 |
| `TestLuminance_White` | White → ~1.0 |
| `TestLuminance_Black` | Black → ~0.0 |
| `TestImageStats` | Correct min/max/mean/stddev for known grid |
| `TestSobelEdgeDetect` | Detects vertical edge in synthetic image |

### Approach:
- Create synthetic `image.RGBA` images in tests (solid colors, gradients, half-black/half-white)
- Use table-driven tests with `t.Run()` subtests
- Test file: `backend/internal/converter/converter_test.go` and `braille_test.go`

---

## Step 2: Handler Integration Tests

**Package:** `backend/internal/handler`

Test the HTTP layer using `net/http/httptest` — no external dependencies needed.

### Tests to write:

| Test | What it verifies |
|------|-----------------|
| `TestConvert_ValidImage` | 200 + JSON response with ascii/width/height fields |
| `TestConvert_NoImage` | 400 + error "image file is required" |
| `TestConvert_InvalidImage` | 400 + error "invalid or unsupported image" (send random bytes) |
| `TestConvert_InvalidWidth` | 400 + error for width=-1, width=abc |
| `TestConvert_BrailleMode` | 200 + braille characters in response |
| `TestConvert_OversizeBody` | Body > 20MB triggers `MaxBytesReader`; expect 400 "invalid multipart payload" (current behavior — upgrade to 413 as a future improvement) |
| `TestConvert_InvalidInvert` | 400 + error for invert=banana |
| `TestHealth` | 200 + `{"status":"ok"}` |

### Approach:
- Build `multipart/form-data` requests programmatically
- Use a tiny 2×2 PNG created with `image/png` encoder in test setup
- Test file: `backend/internal/handler/handler_test.go`

---

## Step 3: Helper/Utility Tests

| Function | Test |
|----------|------|
| `luminance()` | Known colors (red, green, blue, white, black) → expected values |
| `clamp()` | Values <0 clamped to 0, >1 clamped to 1, in-range unchanged |
| `reverseString()` | "abc" → "cba", empty → empty |
| `imageStats()` | Known grid → correct min/max/mean/stddev |
| `applyContrast()` | Factor 2.0 pushes values away from 0.5 |

These are trivially fast and protect against regressions in core math.

---

## Step 4: Frontend Tests (later, lower priority)

**Framework:** Vitest + React Testing Library

| Test | What it verifies |
|------|-----------------|
| `ThemeToggle_Toggle` | Clicking toggles `data-theme` attribute |
| `ThemeToggle_InitFromStorage` | Reads from localStorage on mount |
| `ImageUploader` | Renders file input and accepts images |
| `AsciiOutput` | Renders pre-formatted text, copy button works |
| `convert API` | Calls fetch with correct multipart body |

### Setup required:
```bash
npm install -D vitest @testing-library/react @testing-library/jest-dom jsdom
```

Add to `frontend/package.json`:
```json
"scripts": {
  "test": "vitest run",
  "test:watch": "vitest"
}
```

**Why last:** Frontend tests require more tooling setup and the UI is changing frequently. Backend converter logic is more stable and higher risk.

---

## Implementation Order

```
Step 1 → converter unit tests     (PR: feat/converter-tests)
Step 2 → handler integration tests (PR: feat/handler-tests)
Step 3 → helper/utility tests      (can be in same PR as Step 1)
Step 4 → frontend tests            (PR: feat/frontend-tests, deferred)
```

## CI Integration

Already configured — `go test ./...` runs on every PR. Once test files exist, they're automatically enforced. No CI changes needed for Steps 1–3.

For Step 4, add to `.github/workflows/deploy.yml`:
```yaml
- name: Frontend tests
  working-directory: frontend
  run: npm test
```

## Coverage Target

Start with **no coverage threshold** — the goal is to have meaningful tests, not chase a number. Once we have 20+ tests, consider adding a coverage gate (~60% for converter package).

Run locally (from `backend/` directory):
```bash
cd backend
go test -cover ./internal/converter/
```
