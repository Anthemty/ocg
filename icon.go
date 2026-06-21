package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// usageIconBytes generates a coloured circular status icon for the menu bar.
// Colour depends on monthly usage: green (<50%), yellow (50-84%), red (≥85%).
// A white arc inside visualises the fill level. Size 22×22 for retina.
func usageIconBytes(monthlyPct int) []byte {
	const size = 22
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	cx, cy := float64(size)/2, float64(size)/2
	outerR := float64(size)/2 - 1
	innerR := outerR - 3

	col := usageColorRGBA(monthlyPct)

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx + 0.5
			dy := float64(y) - cy + 0.5
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= outerR {
				if dist >= innerR {
					// ring: coloured
					img.Set(x, y, col)
				} else {
					// interior: fill proportional arc
					angle := math.Atan2(dy, dx) + math.Pi/2 // 0 at top
					if angle < 0 {
						angle += 2 * math.Pi
					}
					threshold := 2 * math.Pi * float64(monthlyPct) / 100
					if angle <= threshold && monthlyPct > 0 {
						img.Set(x, y, col)
					}
				}
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// neutralIconBytes returns a grey circle for unconfigured / error states.
func neutralIconBytes() []byte {
	const size = 22
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	cx, cy := float64(size)/2, float64(size)/2
	r := float64(size)/2 - 1
	grey := color.RGBA{R: 150, G: 150, B: 150, A: 255}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx + 0.5
			dy := float64(y) - cy + 0.5
			if dx*dx+dy*dy <= r*r {
				img.Set(x, y, grey)
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func usageColorRGBA(pct int) color.RGBA {
	switch {
	case pct < 50:
		return color.RGBA{R: 52, G: 199, B: 89, A: 255} // system green
	case pct < 85:
		return color.RGBA{R: 255, G: 159, B: 10, A: 255} // system yellow
	default:
		return color.RGBA{R: 255, G: 69, B: 58, A: 255} // system red
	}
}
