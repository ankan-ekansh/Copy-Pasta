package converter

import (
	"image"
	"math"
	"strings"

	"golang.org/x/image/draw"
)

// Braille Unicode characters encode a 2x4 dot matrix per character cell.
// Each character maps 8 binary pixels, giving much higher effective resolution
// than traditional brightness-to-char mapping.
//
// Braille dot positions:
//   [0] [3]
//   [1] [4]
//   [2] [5]
//   [6] [7]
//
// Unicode offset: U+2800 + sum of 2^position for each "on" dot.

// BrailleOptions controls braille conversion.
type BrailleOptions struct {
	Width     int     // output width in characters (each char = 2 pixels wide)
	Threshold float64 // brightness threshold for dot on/off (0-1, 0 = auto via Otsu)
	Invert    bool    // invert (dark dots on light background vs light dots on dark)
	Dither    bool    // use Floyd-Steinberg dithering for simulated grayscale
}

// ConvertBraille converts an image to Unicode Braille art.
// This gives ~2x horizontal and ~4x vertical resolution vs standard ASCII.
func ConvertBraille(img image.Image, opts BrailleOptions) string {
	if img == nil {
		return ""
	}

	width := opts.Width
	if width <= 0 {
		width = 80 // default braille width (each char = 2px, so 160px effective)
	}

	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()
	if srcWidth == 0 || srcHeight == 0 {
		return ""
	}

	// Each braille character represents 2 columns × 4 rows of pixels
	pixelWidth := width * 2
	pixelHeight := int(float64(srcHeight) * (float64(pixelWidth) / float64(srcWidth)))
	// Round up to multiple of 4 for clean row boundaries
	if pixelHeight%4 != 0 {
		pixelHeight += 4 - (pixelHeight % 4)
	}

	// High-quality resize
	resized := image.NewRGBA(image.Rect(0, 0, pixelWidth, pixelHeight))
	draw.CatmullRom.Scale(resized, resized.Bounds(), img, bounds, draw.Over, nil)

	// Extract brightness grid
	grid := make([][]float64, pixelHeight)
	for y := 0; y < pixelHeight; y++ {
		grid[y] = make([]float64, pixelWidth)
		for x := 0; x < pixelWidth; x++ {
			grid[y][x] = luminance(resized.At(x, y))
		}
	}

	// Apply Floyd-Steinberg dithering or simple threshold
	if opts.Dither {
		floydSteinbergDither(grid, pixelHeight, pixelWidth, opts.Invert)
	}

	// Determine threshold (for non-dithered mode, or as reference)
	threshold := opts.Threshold
	if threshold <= 0 || threshold >= 1 {
		if opts.Dither {
			threshold = 0.5 // after dithering, pixels are pushed toward 0 or 1
		} else {
			threshold = otsuThreshold(grid, pixelHeight, pixelWidth)
		}
	}

	// Build braille output
	charRows := pixelHeight / 4
	var builder strings.Builder
	builder.Grow((width + 1) * charRows)

	for charY := 0; charY < charRows; charY++ {
		line := make([]rune, width)
		for charX := 0; charX < width; charX++ {
			px := charX * 2
			py := charY * 4

			// Map 2x4 pixel block to braille dot pattern
			var brailleOffset rune = 0x2800
			dots := [8]bool{
				isDot(grid, py+0, px+0, pixelHeight, pixelWidth, threshold, opts.Invert && !opts.Dither),
				isDot(grid, py+1, px+0, pixelHeight, pixelWidth, threshold, opts.Invert && !opts.Dither),
				isDot(grid, py+2, px+0, pixelHeight, pixelWidth, threshold, opts.Invert && !opts.Dither),
				isDot(grid, py+0, px+1, pixelHeight, pixelWidth, threshold, opts.Invert && !opts.Dither),
				isDot(grid, py+1, px+1, pixelHeight, pixelWidth, threshold, opts.Invert && !opts.Dither),
				isDot(grid, py+2, px+1, pixelHeight, pixelWidth, threshold, opts.Invert && !opts.Dither),
				isDot(grid, py+3, px+0, pixelHeight, pixelWidth, threshold, opts.Invert && !opts.Dither),
				isDot(grid, py+3, px+1, pixelHeight, pixelWidth, threshold, opts.Invert && !opts.Dither),
			}

			for i, on := range dots {
				if on {
					brailleOffset |= 1 << i
				}
			}

			line[charX] = brailleOffset
		}
		// Trim trailing empty braille chars (U+2800) to shorten lines
		trimmed := strings.TrimRight(string(line), "\u2800")
		builder.WriteString(trimmed)
		if charY < charRows-1 {
			builder.WriteByte('\n')
		}
	}

	return builder.String()
}

// floydSteinbergDither applies Floyd-Steinberg error diffusion dithering in-place.
// After dithering, pixels are pushed toward 0.0 or 1.0, simulating grayscale
// through dot density patterns.
func floydSteinbergDither(grid [][]float64, height, width int, invert bool) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			oldVal := grid[y][x]
			// Threshold at 0.5 for dithering
			var newVal float64
			if invert {
				if oldVal >= 0.5 {
					newVal = 1.0
				} else {
					newVal = 0.0
				}
			} else {
				if oldVal < 0.5 {
					newVal = 0.0
				} else {
					newVal = 1.0
				}
			}
			grid[y][x] = newVal
			err := oldVal - newVal

			// Distribute error to neighbors (Floyd-Steinberg kernel)
			if x+1 < width {
				grid[y][x+1] += err * 7.0 / 16.0
			}
			if y+1 < height {
				if x-1 >= 0 {
					grid[y+1][x-1] += err * 3.0 / 16.0
				}
				grid[y+1][x] += err * 5.0 / 16.0
				if x+1 < width {
					grid[y+1][x+1] += err * 1.0 / 16.0
				}
			}
		}
	}
}

// isDot returns true if the pixel at (row, col) should be "on" (a raised dot).
func isDot(grid [][]float64, row, col, maxRow, maxCol int, threshold float64, invert bool) bool {
	if row >= maxRow || col >= maxCol {
		return false
	}
	bright := grid[row][col]
	if invert {
		return bright >= threshold // bright pixels = dots (for dark backgrounds)
	}
	return bright < threshold // dark pixels = dots (for light backgrounds)
}

// otsuThreshold computes the optimal binary threshold using Otsu's method.
// This automatically finds the best split point between foreground and background.
func otsuThreshold(grid [][]float64, height, width int) float64 {
	// Build histogram with 256 bins
	const bins = 256
	var histogram [bins]int
	total := height * width

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			bin := int(grid[y][x] * float64(bins-1))
			if bin >= bins {
				bin = bins - 1
			}
			if bin < 0 {
				bin = 0
			}
			histogram[bin]++
		}
	}

	// Otsu's algorithm: find threshold that maximizes between-class variance
	var sumTotal float64
	for i := 0; i < bins; i++ {
		sumTotal += float64(i) * float64(histogram[i])
	}

	var sumBg float64
	var weightBg int
	maxVariance := 0.0
	bestThreshold := 0.5

	for t := 0; t < bins; t++ {
		weightBg += histogram[t]
		if weightBg == 0 {
			continue
		}
		weightFg := total - weightBg
		if weightFg == 0 {
			break
		}

		sumBg += float64(t) * float64(histogram[t])
		meanBg := sumBg / float64(weightBg)
		meanFg := (sumTotal - sumBg) / float64(weightFg)

		variance := float64(weightBg) * float64(weightFg) * math.Pow(meanBg-meanFg, 2)
		if variance > maxVariance {
			maxVariance = variance
			bestThreshold = float64(t) / float64(bins-1)
		}
	}

	return bestThreshold
}
