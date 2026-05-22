package converter

import (
	"image"
	"image/color"
	"strings"
	"testing"
	"unicode"
)

func TestConvertBraille_NilImage(t *testing.T) {
	result := ConvertBraille(nil, BrailleOptions{})
	if result != "" {
		t.Errorf("expected empty string for nil image, got %q", result)
	}
}

func TestConvertBraille_EmptyImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 0, 0))
	result := ConvertBraille(img, BrailleOptions{})
	if result != "" {
		t.Errorf("expected empty string for 0x0 image, got %q", result)
	}
}

func TestConvertBraille_DefaultWidth(t *testing.T) {
	// Image with a dark stripe on the right edge ensures last braille cell is non-empty
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			if x > 190 {
				img.Set(x, y, color.Black) // dark right edge
			} else {
				img.Set(x, y, color.White)
			}
		}
	}
	result := ConvertBraille(img, BrailleOptions{Width: 0, Dither: true})
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	// Default braille width is 80; dark stripe at right ensures no full trim
	firstLine := strings.SplitN(result, "\n", 2)[0]
	lineRunes := []rune(firstLine)
	if len(lineRunes) != 80 {
		t.Errorf("expected default line width of exactly 80, got %d", len(lineRunes))
	}
}

func TestConvertBraille_OutputIsBraille(t *testing.T) {
	img := halfImage(200, 200)
	result := ConvertBraille(img, BrailleOptions{Width: 40})
	for i, r := range result {
		if r == '\n' {
			continue
		}
		// Braille block: U+2800 to U+28FF
		if r < 0x2800 || r > 0x28FF {
			t.Errorf("char at position %d is %U, not in braille range U+2800–U+28FF", i, r)
			break
		}
	}
}

func TestConvertBraille_Dithering(t *testing.T) {
	img := solidImage(100, 100, color.Gray{128})
	withDither := ConvertBraille(img, BrailleOptions{Width: 30, Dither: true})
	withoutDither := ConvertBraille(img, BrailleOptions{Width: 30, Dither: false})
	if withDither == withoutDither {
		t.Error("expected dithered output to differ from non-dithered")
	}
}

func TestConvertBraille_Invert(t *testing.T) {
	img := halfImage(200, 200)
	normal := ConvertBraille(img, BrailleOptions{Width: 30, Invert: false})
	inverted := ConvertBraille(img, BrailleOptions{Width: 30, Invert: true})
	if normal == inverted {
		t.Error("expected inverted output to differ from normal")
	}
}

func TestConvertBraille_NonEmptyForRealImage(t *testing.T) {
	// A gradient image should produce varied braille output
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			gray := uint8(x * 255 / 100)
			img.Set(x, y, color.Gray{gray})
		}
	}
	result := ConvertBraille(img, BrailleOptions{Width: 30, Dither: true})
	if result == "" {
		t.Fatal("expected non-empty braille output for gradient image")
	}
	// Should have multiple distinct characters (not all the same)
	unique := make(map[rune]bool)
	for _, r := range result {
		if r != '\n' {
			unique[r] = true
		}
	}
	if len(unique) < 3 {
		t.Errorf("expected varied braille output, got only %d unique chars", len(unique))
	}
}

func TestOtsuThreshold(t *testing.T) {
	// Bimodal distribution: half dark (0.1-0.3), half bright (0.7-0.9)
	height, width := 20, 20
	grid := make([][]float64, height)
	for y := 0; y < height; y++ {
		grid[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			if x < width/2 {
				grid[y][x] = 0.1 + float64(y)*0.01 // dark cluster: 0.1–0.29
			} else {
				grid[y][x] = 0.7 + float64(y)*0.01 // bright cluster: 0.7–0.89
			}
		}
	}
	threshold := otsuThreshold(grid, height, width)
	// Otsu should find a threshold that separates the two clusters
	// It should be > the dark cluster max and < the bright cluster min
	if threshold < 0.2 || threshold > 0.8 {
		t.Errorf("otsuThreshold = %f, expected between 0.2 and 0.8 for bimodal distribution", threshold)
	}
}

func TestFloydSteinbergDither(t *testing.T) {
	height, width := 10, 10
	grid := make([][]float64, height)
	for y := 0; y < height; y++ {
		grid[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			grid[y][x] = 0.5 // mid-gray
		}
	}

	floydSteinbergDither(grid, height, width)

	// After dithering, all values should be 0.0 or 1.0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			v := grid[y][x]
			if v != 0.0 && v != 1.0 {
				t.Errorf("grid[%d][%d] = %f, expected 0.0 or 1.0 after dithering", y, x, v)
				return
			}
		}
	}
}

func TestConvertBraille_AllBrailleUnicode(t *testing.T) {
	// Verify the output only contains valid braille or newline characters
	img := solidImage(80, 80, color.Gray{100})
	result := ConvertBraille(img, BrailleOptions{Width: 20, Dither: true})
	for i, r := range result {
		if r == '\n' {
			continue
		}
		if !unicode.In(r, unicode.Braille) {
			t.Errorf("position %d: char %U is not a braille character", i, r)
			break
		}
	}
}
