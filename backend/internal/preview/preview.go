// Package preview renders ASCII/Braille art as PNG images for OG previews.
package preview

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
	"strings"

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
	return nil, os.ErrNotExist
}

func newFace(fontData []byte, size float64) (font.Face, error) {
	f, err := opentype.Parse(fontData)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}

// Renderer holds a parsed font for reuse across requests.
type Renderer struct {
	fontData []byte
}

// NewRenderer creates a Renderer, loading the font from disk.
// fontPath overrides the default search paths if non-empty.
func NewRenderer(fontPath string) (*Renderer, error) {
	data, err := findFont(fontPath)
	if err != nil {
		return nil, err
	}
	return &Renderer{fontData: data}, nil
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
		if n := len([]rune(line)); n > maxLen {
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

	face, err := newFace(rr.fontData, fontSize)
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

	lineHeight := fixed.I(int(fontSize * 1.2))
	startY := fixed.I(padding) + face.Metrics().Ascent

	for i, line := range lines {
		drawer.Dot = fixed.Point26_6{
			X: fixed.I(padding),
			Y: startY + lineHeight*fixed.Int26_6(i),
		}
		drawer.DrawString(line)
	}

	// Draw branding
	brandFace, err := newFace(rr.fontData, 12)
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
