package converter

import (
	"image"
	"image/color"
	"math"
	"strings"

	"golang.org/x/image/draw"
)

const DefaultWidth = 150

// Extended ramps with more grayscale levels for better detail
const (
	RampDetailed = " .'`^\",:;Il!i><~+_-?][}{1)(|\\/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$"
	RampStandard = " .:-=+*#%@"
	RampBlocks   = " ░▒▓█"
	RampSimple   = " .oO@"
)

type Options struct {
	Width       int
	Invert      bool
	EdgeMix     float64 // 0.0 = pure brightness, 1.0 = pure edges, 0.3 is a good default
	Contrast    float64 // contrast boost factor, 1.0 = no change, 1.5 = 50% more
	CharRamp    string  // character ramp to use (empty = auto-select detailed)
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

	// Aspect ratio correction: chars are ~2.2x taller than wide in most monospace fonts
	targetHeight := int(float64(srcHeight) * (float64(width) / float64(srcWidth)) * 0.45)
	if targetHeight < 1 {
		targetHeight = 1
	}

	// High-quality resize
	resized := image.NewRGBA(image.Rect(0, 0, width, targetHeight))
	draw.CatmullRom.Scale(resized, resized.Bounds(), img, bounds, draw.Over, nil)

	// Compute brightness grid
	brightness := make([][]float64, targetHeight)
	for y := 0; y < targetHeight; y++ {
		brightness[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			brightness[y][x] = grayscaleBrightness(resized.At(x, y))
		}
	}

	// Normalize contrast (histogram stretching)
	brightness = normalizeContrast(brightness, targetHeight, width)

	// Apply contrast boost if requested
	contrastFactor := opts.Contrast
	if contrastFactor <= 0 {
		contrastFactor = 1.3 // default slight boost
	}
	if contrastFactor != 1.0 {
		brightness = applyContrast(brightness, targetHeight, width, contrastFactor)
	}

	// Compute edge map if edge mixing is enabled
	edgeMix := opts.EdgeMix
	if edgeMix < 0 {
		edgeMix = 0
	}
	if edgeMix > 1 {
		edgeMix = 1
	}
	// Default: blend 30% edge detection for better structural detail
	if opts.EdgeMix == 0 && opts.Contrast == 0 {
		edgeMix = 0.3
	}

	var edges [][]float64
	if edgeMix > 0 {
		edges = sobelEdgeDetect(brightness, targetHeight, width)
	}

	// Select character ramp
	ramp := opts.CharRamp
	if ramp == "" {
		ramp = RampDetailed
	}
	if opts.Invert {
		ramp = reverseString(ramp)
	}

	// Build ASCII output
	var builder strings.Builder
	builder.Grow((width + 1) * targetHeight)

	maxIndex := len(ramp) - 1
	for y := 0; y < targetHeight; y++ {
		for x := 0; x < width; x++ {
			val := brightness[y][x]

			// Blend with edge detection
			if edgeMix > 0 && edges != nil {
				edgeVal := edges[y][x]
				// Edges darken the output (make structural lines visible)
				val = val*(1.0-edgeMix) + (1.0-edgeVal)*edgeMix
			}

			// Map to character index (dark pixels = dense chars)
			index := int((1.0 - val) * float64(maxIndex))
			if index < 0 {
				index = 0
			}
			if index > maxIndex {
				index = maxIndex
			}
			builder.WriteByte(ramp[index])
		}
		if y < targetHeight-1 {
			builder.WriteByte('\n')
		}
	}

	return builder.String()
}

// normalizeContrast stretches the histogram so that the darkest pixel maps to 0
// and the brightest maps to 1, using the full character ramp.
func normalizeContrast(grid [][]float64, height, width int) [][]float64 {
	minVal := 1.0
	maxVal := 0.0

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			v := grid[y][x]
			if v < minVal {
				minVal = v
			}
			if v > maxVal {
				maxVal = v
			}
		}
	}

	span := maxVal - minVal
	if span < 0.01 {
		return grid // image is essentially flat
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			grid[y][x] = (grid[y][x] - minVal) / span
		}
	}
	return grid
}

// applyContrast boosts contrast by shifting values away from 0.5
func applyContrast(grid [][]float64, height, width int, factor float64) [][]float64 {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			v := grid[y][x]
			// Sigmoid-like contrast: shift from midpoint
			v = (v-0.5)*factor + 0.5
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			grid[y][x] = v
		}
	}
	return grid
}

// sobelEdgeDetect computes edge magnitude using Sobel operator
func sobelEdgeDetect(brightness [][]float64, height, width int) [][]float64 {
	edges := make([][]float64, height)
	for y := range edges {
		edges[y] = make([]float64, width)
	}

	maxEdge := 0.0
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			// Sobel kernels
			gx := -brightness[y-1][x-1] + brightness[y-1][x+1] +
				-2*brightness[y][x-1] + 2*brightness[y][x+1] +
				-brightness[y+1][x-1] + brightness[y+1][x+1]

			gy := -brightness[y-1][x-1] - 2*brightness[y-1][x] - brightness[y-1][x+1] +
				brightness[y+1][x-1] + 2*brightness[y+1][x] + brightness[y+1][x+1]

			mag := math.Sqrt(gx*gx + gy*gy)
			edges[y][x] = mag
			if mag > maxEdge {
				maxEdge = mag
			}
		}
	}

	// Normalize edges to 0-1
	if maxEdge > 0 {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				edges[y][x] /= maxEdge
			}
		}
	}

	return edges
}

func grayscaleBrightness(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	// Apply gamma correction for perceptual accuracy
	rf := math.Pow(float64(r)/65535.0, 2.2)
	gf := math.Pow(float64(g)/65535.0, 2.2)
	bf := math.Pow(float64(b)/65535.0, 2.2)
	// Luminance weights (BT.709)
	luminance := 0.2126*rf + 0.7152*gf + 0.0722*bf
	// Convert back to perceptual space
	return math.Pow(luminance, 1.0/2.2)
}

func reverseString(s string) string {
	bytes := []byte(s)
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}
	return string(bytes)
}
