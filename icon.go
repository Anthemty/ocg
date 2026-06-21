package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

// iconBytes returns a 16×16 black-on-transparent PNG: three ascending bars
// suggesting usage monitoring. Suitable as a macOS template image.
func iconBytes() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	// three bars, bottom-aligned at y=13, each 3px wide
	bars := []struct{ x, top int }{
		{2, 9},   // short
		{7, 5},   // medium
		{12, 2},  // tall
	}
	for _, b := range bars {
		for x := b.x; x < b.x+3; x++ {
			for y := b.top; y <= 13; y++ {
				img.Set(x, y, color.Black)
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
