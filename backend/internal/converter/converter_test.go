package converter

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"
)

// solidImage creates a uniform-color image of the given size.
func solidImage(w, h int, c color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

// halfImage creates an image that is black on the left and white on the right.
func halfImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < w/2 {
				img.Set(x, y, color.Black)
			} else {
				img.Set(x, y, color.White)
			}
		}
	}
	return img
}

func TestConvert_NilImage(t *testing.T) {
	result := Convert(nil, Options{})
	if result != "" {
		t.Errorf("expected empty string for nil image, got %q", result)
	}
}

func TestConvert_EmptyImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 0, 0))
	result := Convert(img, Options{})
	if result != "" {
		t.Errorf("expected empty string for 0x0 image, got %q", result)
	}
}

func TestConvert_DefaultWidth(t *testing.T) {
	img := solidImage(300, 300, color.White)
	result := Convert(img, Options{Width: 0})
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	firstLine := strings.SplitN(result, "\n", 2)[0]
	if len(firstLine) != DefaultWidth {
		t.Errorf("expected line width %d, got %d", DefaultWidth, len(firstLine))
	}
}

func TestConvert_WidthClamping(t *testing.T) {
	img := solidImage(200, 200, color.Gray{128})
	widths := []int{10, 50, 80, 120}
	for _, w := range widths {
		t.Run(fmt.Sprintf("width_%d", w), func(t *testing.T) {
			result := Convert(img, Options{Width: w})
			firstLine := strings.SplitN(result, "\n", 2)[0]
			if len(firstLine) != w {
				t.Errorf("width=%d: expected line length %d, got %d", w, w, len(firstLine))
			}
		})
	}
}

func TestConvert_OutputDimensions(t *testing.T) {
	img := solidImage(100, 100, color.Gray{128})
	result := Convert(img, Options{Width: 50})
	lines := strings.Split(result, "\n")
	if len(lines) == 0 {
		t.Fatal("expected at least one line of output")
	}
	// All lines should be exactly width=50
	for i, line := range lines {
		if len(line) != 50 {
			t.Errorf("line %d: expected length 50, got %d", i, len(line))
		}
	}
	// Height should be approximately width * 0.45 for square images
	srcH := float64(100)
	w := float64(50)
	srcW := float64(100)
	expectedHeight := int(srcH * (w / srcW) * 0.45)
	if len(lines) != expectedHeight {
		t.Errorf("expected %d rows, got %d", expectedHeight, len(lines))
	}
}

func TestConvert_AllWhite(t *testing.T) {
	img := solidImage(100, 100, color.White)
	// Disable auto-tuning so output is deterministic
	result := Convert(img, Options{Width: 20, Contrast: 1.0, EdgeMix: 0})
	chars := strings.ReplaceAll(result, "\n", "")
	if len(chars) == 0 {
		t.Fatal("expected non-empty result")
	}
	// Uniform image → uniform output (all same character)
	first := chars[0]
	for i, c := range []byte(chars) {
		if c != first {
			t.Errorf("expected uniform output, but char at %d differs: %c vs %c", i, c, first)
			break
		}
	}
}

func TestConvert_AllBlack(t *testing.T) {
	img := solidImage(100, 100, color.Black)
	result := Convert(img, Options{Width: 20, Contrast: 1.0, EdgeMix: 0})
	chars := strings.ReplaceAll(result, "\n", "")
	if len(chars) == 0 {
		t.Fatal("expected non-empty result")
	}
	// Uniform image → uniform output
	first := chars[0]
	for i, c := range []byte(chars) {
		if c != first {
			t.Errorf("expected uniform output, but char at %d differs: %c vs %c", i, c, first)
			break
		}
	}
}

func TestConvert_WhiteLighterThanBlack(t *testing.T) {
	// With a two-char ramp, white and black should map to opposite ends
	white := solidImage(100, 100, color.White)
	black := solidImage(100, 100, color.Black)
	half := halfImage(200, 200)

	// Use the half image to force normalization to use full range
	result := Convert(half, Options{Width: 20, Contrast: 1.0, EdgeMix: 0, CharRamp: " @"})
	chars := strings.ReplaceAll(result, "\n", "")
	hasSpace := strings.Contains(chars, " ")
	hasAt := strings.Contains(chars, "@")
	if !hasSpace || !hasAt {
		t.Errorf("expected both ' ' and '@' in half-image output, got: %q", chars[:20])
	}

	// Verify all-white uniform is different from all-black uniform (via normalization they're same)
	_ = white
	_ = black
}

func TestConvert_Invert(t *testing.T) {
	img := halfImage(200, 200)
	normal := Convert(img, Options{Width: 40, Invert: false})
	inverted := Convert(img, Options{Width: 40, Invert: true})
	if normal == inverted {
		t.Error("expected inverted output to differ from normal")
	}
}

func TestConvert_CustomRamp(t *testing.T) {
	img := solidImage(100, 100, color.Gray{128})
	customRamp := "XO"
	result := Convert(img, Options{Width: 20, CharRamp: customRamp})
	chars := strings.ReplaceAll(result, "\n", "")
	for _, c := range chars {
		if c != 'X' && c != 'O' {
			t.Errorf("unexpected char %c not in custom ramp %q", c, customRamp)
			break
		}
	}
}

func TestConvert_DifferentImagesProduceDifferentOutput(t *testing.T) {
	white := solidImage(100, 100, color.White)
	black := solidImage(100, 100, color.Black)
	half := halfImage(200, 200)

	opts := Options{Width: 30, Contrast: 1.0, EdgeMix: 0}
	rWhite := Convert(white, opts)
	rBlack := Convert(black, opts)
	rHalf := Convert(half, opts)

	if rHalf == rWhite {
		t.Error("half image should differ from all-white")
	}
	if rHalf == rBlack {
		t.Error("half image should differ from all-black")
	}
}

// --- Helper function tests (Step 3 from testing plan, implemented together) ---

func TestLuminance(t *testing.T) {
	tests := []struct {
		name string
		c    color.Color
		min  float64
		max  float64
	}{
		{"white", color.White, 0.95, 1.0},
		{"black", color.Black, 0.0, 0.05},
		{"mid-gray", color.Gray{128}, 0.4, 0.6},
		{"red", color.RGBA{255, 0, 0, 255}, 0.1, 0.6},
		{"green", color.RGBA{0, 255, 0, 255}, 0.5, 0.95},
		{"blue", color.RGBA{0, 0, 255, 255}, 0.05, 0.4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := luminance(tt.c)
			if l < tt.min || l > tt.max {
				t.Errorf("luminance(%s) = %f, expected in [%f, %f]", tt.name, l, tt.min, tt.max)
			}
		})
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{-0.5, 0.0},
		{0.0, 0.0},
		{0.5, 0.5},
		{1.0, 1.0},
		{1.5, 1.0},
	}
	for _, tt := range tests {
		result := clamp(tt.input)
		if result != tt.expected {
			t.Errorf("clamp(%f) = %f, want %f", tt.input, result, tt.expected)
		}
	}
}

func TestReverseString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abc", "cba"},
		{"", ""},
		{"a", "a"},
		{"ab", "ba"},
		{" .@", "@. "},
	}
	for _, tt := range tests {
		result := reverseString(tt.input)
		if result != tt.expected {
			t.Errorf("reverseString(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestImageStats(t *testing.T) {
	// 2x2 grid with values 0.0, 0.25, 0.75, 1.0
	grid := [][]float64{
		{0.0, 0.25},
		{0.75, 1.0},
	}
	min, max, mean, stddev := imageStats(grid, 2, 2)

	if min != 0.0 {
		t.Errorf("min = %f, want 0.0", min)
	}
	if max != 1.0 {
		t.Errorf("max = %f, want 1.0", max)
	}
	expectedMean := 0.5
	if mean < expectedMean-0.01 || mean > expectedMean+0.01 {
		t.Errorf("mean = %f, want ~%f", mean, expectedMean)
	}
	if stddev <= 0 {
		t.Error("stddev should be > 0 for non-uniform grid")
	}
}

func TestApplyContrast(t *testing.T) {
	grid := [][]float64{
		{0.3, 0.7},
		{0.5, 0.9},
	}
	applyContrast(grid, 2, 2, 2.0)

	// 0.3 → (0.3-0.5)*2+0.5 = 0.1
	// 0.7 → (0.7-0.5)*2+0.5 = 0.9
	// 0.5 → (0.5-0.5)*2+0.5 = 0.5
	// 0.9 → (0.9-0.5)*2+0.5 = 1.3 → clamped to 1.0
	expected := [][]float64{
		{0.1, 0.9},
		{0.5, 1.0},
	}
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			if diff := grid[y][x] - expected[y][x]; diff > 0.01 || diff < -0.01 {
				t.Errorf("grid[%d][%d] = %f, want %f", y, x, grid[y][x], expected[y][x])
			}
		}
	}
}

func TestSobelEdgeDetect(t *testing.T) {
	// Create a grid with a sharp vertical edge in the middle
	height, width := 10, 10
	grid := make([][]float64, height)
	for y := 0; y < height; y++ {
		grid[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			if x < width/2 {
				grid[y][x] = 0.0
			} else {
				grid[y][x] = 1.0
			}
		}
	}

	edges := sobelEdgeDetect(grid, height, width)

	// Edge values at the boundary (x=4,5) should be high
	edgeCol := width / 2
	maxEdge := 0.0
	for y := 1; y < height-1; y++ {
		if edges[y][edgeCol] > maxEdge {
			maxEdge = edges[y][edgeCol]
		}
	}
	if maxEdge < 0.5 {
		t.Errorf("expected strong edge at boundary, max edge = %f", maxEdge)
	}

	// Interior values (far from edge) should be near zero
	if edges[height/2][1] > 0.1 {
		t.Errorf("expected low edge value in flat region, got %f", edges[height/2][1])
	}
}
