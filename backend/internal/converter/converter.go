package converter

import (
	"image"
	"image/color"
	"strings"

	"golang.org/x/image/draw"
)

const DefaultWidth = 120

const asciiRamp = " .:-=+*#%@"

type Options struct {
	Width  int
	Invert bool
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

	targetHeight := int(float64(srcHeight) * (float64(width) / float64(srcWidth)) * 0.5)
	if targetHeight < 1 {
		targetHeight = 1
	}

	resized := image.NewRGBA(image.Rect(0, 0, width, targetHeight))
	draw.CatmullRom.Scale(resized, resized.Bounds(), img, bounds, draw.Over, nil)

	ramp := asciiRamp
	if opts.Invert {
		ramp = reverseString(ramp)
	}

	var builder strings.Builder
	builder.Grow((width + 1) * targetHeight)

	maxIndex := len(ramp) - 1
	for y := 0; y < targetHeight; y++ {
		for x := 0; x < width; x++ {
			brightness := grayscaleBrightness(resized.At(x, y))
			index := int((1.0 - brightness) * float64(maxIndex))
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

func grayscaleBrightness(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	brightness := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
	return brightness / 65535.0
}

func reverseString(s string) string {
	bytes := []byte(s)
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}
	return string(bytes)
}
