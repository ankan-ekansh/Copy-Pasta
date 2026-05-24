// Package preview renders ASCII/Braille art as PNG images for OG previews.
package preview

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	imgWidth  = 1200
	imgHeight = 630
	padding   = 40
)

var (
	bgColor    = color.RGBA{24, 24, 32, 255}
	textColor  = color.RGBA{200, 200, 210, 255}
	brandColor = color.RGBA{255, 107, 157, 255}
)

// DefaultFontPaths lists common locations for DejaVu Sans Mono.
var DefaultFontPaths = []string{
	"/usr/share/fonts/dejavu/DejaVuSansMono.ttf",           // Alpine
	"/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",  // Debian/Ubuntu
	"/usr/share/fonts/TTF/DejaVuSansMono.ttf",              // Arch
}

func findFont(extraPath string) ([]byte, error) {
	paths := DefaultFontPaths
	if extraPath != "" {
		paths = append([]string{extraPath}, paths...)
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err == nil {
			return data, nil
		}
	}
	return nil, fmt.Errorf("font not found in paths: %v", paths)
}

// Renderer holds a parsed font for reuse across requests.
type Renderer struct {
	font *opentype.Font
}

// NewRenderer creates a Renderer, loading and parsing the font from disk.
// If fontPath is non-empty it is searched first, then the default paths are tried.
func NewRenderer(fontPath string) (*Renderer, error) {
	data, err := findFont(fontPath)
	if err != nil {
		return nil, err
	}
	f, err := opentype.Parse(data)
	if err != nil {
		return nil, err
	}
	return &Renderer{font: f}, nil
}

func (rr *Renderer) newFace(size float64) (font.Face, error) {
	return opentype.NewFace(rr.font, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}

// Render draws the ASCII art onto a 1200x630 PNG and writes it to w.
func (rr *Renderer) Render(w io.Writer, asciiArt string) error {
	lines := strings.Split(asciiArt, "\n")

	// Trim trailing empty lines
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		lines = []string{"(empty)"}
	}

	// Find max line width in runes
	maxLen := 0
	for _, line := range lines {
		if n := utf8.RuneCountInString(line); n > maxLen {
			maxLen = n
		}
	}

	// Calculate font size to fit content
	availW := float64(imgWidth - 2*padding)
	availH := float64(imgHeight - 2*padding - 30)

	fontByWidth := availW / (float64(maxLen) * 0.6)
	fontByHeight := availH / (float64(len(lines)) * 1.2)

	fontSize := fontByWidth
	if fontByHeight < fontSize {
		fontSize = fontByHeight
	}
	if fontSize < 3 {
		fontSize = 3
	}
	if fontSize > 20 {
		fontSize = 20
	}

	face, err := rr.newFace(fontSize)
	if err != nil {
		return err
	}
	defer face.Close()

	// Create image with background
	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	draw.Draw(img, img.Bounds(), image.NewUniform(bgColor), image.Point{}, draw.Src)

	// Draw ASCII art
	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(textColor),
		Face: face,
	}

	lineHeight := face.Metrics().Height + face.Metrics().Height/10 // ~10% extra leading
	startY := fixed.I(padding) + face.Metrics().Ascent

	for i, line := range lines {
		drawer.Dot = fixed.Point26_6{
			X: fixed.I(padding),
			Y: startY + lineHeight*fixed.Int26_6(i),
		}
		drawer.DrawString(line)
	}

	// Draw branding
	brandFace, err := rr.newFace(12)
	if err == nil {
		defer brandFace.Close()
		bd := &font.Drawer{
			Dst:  img,
			Src:  image.NewUniform(brandColor),
			Face: brandFace,
		}
		text := "Copy-Pasta"
		tw := bd.MeasureString(text)
		bd.Dot = fixed.Point26_6{
			X: fixed.I(imgWidth-padding) - tw,
			Y: fixed.I(imgHeight - 15),
		}
		bd.DrawString(text)
	}

	return png.Encode(w, img)
}
