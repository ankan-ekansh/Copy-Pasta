package converter

import (
	"image"
	"image/color"
	"math"
	"strings"

	"golang.org/x/image/draw"
)

const DefaultWidth = 150

// Character ramps sorted by measured pixel density (dark to light).
// These are based on actual character cell fill ratios in monospace fonts.
const (
	// 32 visually-distinct characters, density-sorted — best general-purpose ramp.
	// Avoids directional chars (|/\) that create visual noise.
	RampDefault = " .,:;+*?%S#@"

	// 16 clean chars — good for medium detail with very clean output.
	RampClean = " .:;+*%#@"

	// 70 chars (Bourke's density-sorted ramp) — maximum grayscale levels.
	// Best for large widths (200+) and photographic images.
	RampFull = " .'`^\",:;Il!i><~+_-?][}{1)(|/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$"

	// Block characters for terminal display.
	RampBlocks = " ░▒▓█"
)

type Options struct {
	Width    int
	Invert   bool
	EdgeMix  float64 // 0=pure brightness, 1=pure edges. -1 means "auto-detect"
	Contrast float64 // contrast boost factor, 1.0=no change. 0 means "auto"
	CharRamp string  // character ramp (empty = RampDefault)
}

func Convert(img image.Image, opts Options) string {
	if img == nil {
		return ""
	}

	width := opts.Width
	if width <= 0 {
		width = DefaultWidth
	}

	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()
	if srcWidth == 0 || srcHeight == 0 {
		return ""
	}

	// Aspect ratio correction: monospace chars are ~2x taller than wide
	targetHeight := int(float64(srcHeight) * (float64(width) / float64(srcWidth)) * 0.45)
	if targetHeight < 1 {
		targetHeight = 1
	}

	// High-quality resize using CatmullRom (bicubic)
	resized := image.NewRGBA(image.Rect(0, 0, width, targetHeight))
	draw.CatmullRom.Scale(resized, resized.Bounds(), img, bounds, draw.Over, nil)

	// Step 1: Extract brightness grid with perceptual luminance
	grid := make([][]float64, targetHeight)
	for y := 0; y < targetHeight; y++ {
		grid[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			grid[y][x] = luminance(resized.At(x, y))
		}
	}

	// Step 2: Analyze image characteristics for auto-tuning
	minVal, maxVal, mean, stddev := imageStats(grid, targetHeight, width)

	// Step 3: Histogram normalization — use full brightness range
	span := maxVal - minVal
	if span > 0.01 {
		for y := 0; y < targetHeight; y++ {
			for x := 0; x < width; x++ {
				grid[y][x] = (grid[y][x] - minVal) / span
			}
		}
	}

	// Step 4: Contrast enhancement
	contrastFactor := opts.Contrast
	if contrastFactor == 0 {
		// Auto: boost more for low-contrast images
		if stddev < 0.15 {
			contrastFactor = 1.8
		} else if stddev < 0.25 {
			contrastFactor = 1.4
		} else {
			contrastFactor = 1.1
		}
	}
	if contrastFactor != 1.0 {
		applyContrast(grid, targetHeight, width, contrastFactor)
	}

	// Step 5: Edge detection with auto-tuning
	edgeMix := opts.EdgeMix
	if edgeMix < 0 || (opts.EdgeMix == 0 && opts.Contrast == 0) {
		// Auto-detect: high-contrast images (like line art) need less edge mixing,
		// low-contrast photos benefit from more
		if stddev > 0.35 {
			edgeMix = 0.15 // already high contrast (line art, etc.)
		} else if mean > 0.7 || mean < 0.3 {
			edgeMix = 0.4 // bright/dark image with details to pull out
		} else {
			edgeMix = 0.25 // balanced
		}
	}
	if edgeMix > 1 {
		edgeMix = 1
	}

	var edges [][]float64
	if edgeMix > 0 {
		edges = sobelEdgeDetect(grid, targetHeight, width)
	}

	// Step 6: Select and validate character ramp
	ramp := opts.CharRamp
	if ramp == "" {
		ramp = RampDefault
	}
	if opts.Invert {
		ramp = reverseString(ramp)
	}

	// Step 7: Map to characters
	var builder strings.Builder
	builder.Grow((width + 1) * targetHeight)

	maxIndex := len(ramp) - 1
	for y := 0; y < targetHeight; y++ {
		for x := 0; x < width; x++ {
			val := grid[y][x]

			// Blend edge information: edges darken (increase density)
			if edgeMix > 0 && edges != nil {
				edgeVal := edges[y][x]
				val = val*(1.0-edgeMix) + (1.0-edgeVal)*edgeMix
			}

			// Quantize to character index
			index := int((1.0 - clamp(val)) * float64(maxIndex) + 0.5)
			if index > maxIndex {
				index = maxIndex
			}
			if index < 0 {
				index = 0
			}
			builder.WriteByte(ramp[index])
		}
		if y < targetHeight-1 {
			builder.WriteByte('\n')
		}
	}

	return builder.String()
}

// imageStats computes min, max, mean, and standard deviation of the brightness grid.
func imageStats(grid [][]float64, height, width int) (min, max, mean, stddev float64) {
	min = 1.0
	max = 0.0
	sum := 0.0
	count := float64(height * width)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			v := grid[y][x]
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
			sum += v
		}
	}

	mean = sum / count

	// Standard deviation
	varSum := 0.0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			diff := grid[y][x] - mean
			varSum += diff * diff
		}
	}
	stddev = math.Sqrt(varSum / count)
	return
}

// applyContrast boosts contrast using midpoint-centered scaling.
func applyContrast(grid [][]float64, height, width int, factor float64) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			v := (grid[y][x]-0.5)*factor + 0.5
			grid[y][x] = clamp(v)
		}
	}
}

// sobelEdgeDetect computes normalized edge magnitude using the Sobel operator.
func sobelEdgeDetect(grid [][]float64, height, width int) [][]float64 {
	edges := make([][]float64, height)
	for y := range edges {
		edges[y] = make([]float64, width)
	}

	maxEdge := 0.0
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			gx := -grid[y-1][x-1] + grid[y-1][x+1] +
				-2*grid[y][x-1] + 2*grid[y][x+1] +
				-grid[y+1][x-1] + grid[y+1][x+1]

			gy := -grid[y-1][x-1] - 2*grid[y-1][x] - grid[y-1][x+1] +
				grid[y+1][x-1] + 2*grid[y+1][x] + grid[y+1][x+1]

			mag := math.Sqrt(gx*gx + gy*gy)
			edges[y][x] = mag
			if mag > maxEdge {
				maxEdge = mag
			}
		}
	}

	if maxEdge > 0 {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				edges[y][x] /= maxEdge
			}
		}
	}
	return edges
}

// luminance computes perceptual brightness using gamma-corrected BT.709 weights.
func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	// Linearize (gamma decode)
	rf := math.Pow(float64(r)/65535.0, 2.2)
	gf := math.Pow(float64(g)/65535.0, 2.2)
	bf := math.Pow(float64(b)/65535.0, 2.2)
	// BT.709 luminance
	lin := 0.2126*rf + 0.7152*gf + 0.0722*bf
	// Back to perceptual (gamma encode)
	return math.Pow(lin, 1.0/2.2)
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func reverseString(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}
