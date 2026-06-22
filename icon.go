// icon.go generates the menu bar status icon as a monochrome template image.
//
// Template images use only alpha to encode shape: AppKit recolours them to
// match the menu bar (white on dark, black on light). We draw a ring whose
// interior fills proportionally to usage — a single-colour, linear gauge.
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// usageIconBytes generates a monochrome template icon for the menu bar.
// A ring outlines the gauge; a solid sector inside fills clockwise to the
// usage percentage. All pixels are opaque black with varying alpha — AppKit
// applies the correct tint at render time.
func usageIconBytes(monthlyPct int) []byte {
	return gaugeIconBytes(clampPercent(monthlyPct))
}

// neutralIconBytes returns an empty ring for unconfigured / error states.
func neutralIconBytes() []byte {
	return gaugeIconBytes(-1) // negative => no fill
}

// gaugeIconBytes draws the template gauge. fillPct < 0 means no interior fill.
func gaugeIconBytes(fillPct int) []byte {
	const size = 22
	const center = float64(size)/2 - 0.5
	outerR := float64(size)/2 - 1
	innerR := outerR - 2.5

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	black := color.RGBA{R: 0, G: 0, B: 0, A: 255}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - center
			dy := float64(y) - center
			dist := math.Sqrt(dx*dx + dy*dy)

			// Ring: solid between innerR and outerR.
			if dist >= innerR && dist <= outerR {
				// Anti-alias the outer/inner edges for a crisp 1px ring.
				alpha := 255.0
				if dist > outerR-1 {
					alpha *= outerR - dist // fade out at outer edge
				} else if dist < innerR+1 {
					alpha *= dist - innerR // fade in at inner edge
				}
				if alpha < 0 {
					alpha = 0
				}
				if alpha > 255 {
					alpha = 255
				}
				img.Set(x, y, color.RGBA{A: uint8(alpha)})
				continue
			}

			// Interior fill: solid sector clockwise from top, proportional to fillPct.
			if dist < innerR && fillPct > 0 {
				angle := math.Atan2(dy, dx) + math.Pi/2 // 0 at top
				if angle < 0 {
					angle += 2 * math.Pi
				}
				threshold := 2 * math.Pi * float64(fillPct) / 100
				if angle <= threshold {
					img.Set(x, y, black)
				}
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// clampPercent bounds a percentage to [0, 100].
func clampPercent(pct int) int {
	switch {
	case pct < 0:
		return 0
	case pct > 100:
		return 100
	default:
		return pct
	}
}
