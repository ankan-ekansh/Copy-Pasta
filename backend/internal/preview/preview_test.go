package preview

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

// testRenderer creates a Renderer from the first available system font.
// Returns nil if no font is available (e.g., CI without fonts installed).
func testRenderer(t *testing.T) *Renderer {
	t.Helper()
	r, err := NewRenderer("")
	if err != nil {
		t.Skip("skipping: no font available on this system")
	}
	return r
}

func TestRender_ProducesValidPNG(t *testing.T) {
	r := testRenderer(t)

	var buf bytes.Buffer
	err := r.Render(&buf, "Hello, World!\nLine 2")
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("output is not valid PNG: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != imgWidth || bounds.Dy() != imgHeight {
		t.Errorf("dimensions = %dx%d, want %dx%d", bounds.Dx(), bounds.Dy(), imgWidth, imgHeight)
	}
}

func TestRender_EmptyArt(t *testing.T) {
	r := testRenderer(t)

	var buf bytes.Buffer
	err := r.Render(&buf, "")
	if err != nil {
		t.Fatalf("Render() with empty art: %v", err)
	}

	_, err = png.Decode(&buf)
	if err != nil {
		t.Fatalf("output is not valid PNG: %v", err)
	}
}

func TestRender_LargeArt(t *testing.T) {
	r := testRenderer(t)

	// Generate art with many lines and wide content
	var lines string
	for i := 0; i < 200; i++ {
		lines += "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefghijklmnopqrstuvwxyz!@#$\n"
	}

	var buf bytes.Buffer
	err := r.Render(&buf, lines)
	if err != nil {
		t.Fatalf("Render() with large art: %v", err)
	}

	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("output is not valid PNG: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != imgWidth || bounds.Dy() != imgHeight {
		t.Errorf("dimensions = %dx%d, want %dx%d", bounds.Dx(), bounds.Dy(), imgWidth, imgHeight)
	}
}

func TestRender_ImageDimensions(t *testing.T) {
	r := testRenderer(t)

	var buf bytes.Buffer
	err := r.Render(&buf, "test")
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("not a valid PNG: %v", err)
	}

	// Must always be exactly OG dimensions
	want := image.Rect(0, 0, 1200, 630)
	if img.Bounds() != want {
		t.Errorf("bounds = %v, want %v", img.Bounds(), want)
	}
}

func TestNewRenderer_NoFontsAvailable(t *testing.T) {
	// Save and restore default paths
	origPaths := DefaultFontPaths
	DefaultFontPaths = []string{"/nonexistent/a.ttf", "/nonexistent/b.ttf"}
	defer func() { DefaultFontPaths = origPaths }()

	_, err := NewRenderer("/also/nonexistent/font.ttf")
	if err == nil {
		t.Error("NewRenderer() should return error when no fonts are reachable")
	}
}
