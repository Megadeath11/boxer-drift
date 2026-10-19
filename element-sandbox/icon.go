package main

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"

	"golang.org/x/image/draw"
)

//go:embed assets/icon.png
var iconPNG []byte

// windowIcons returns the window/taskbar icon at a few sizes.
func windowIcons() []image.Image {
	src, err := png.Decode(bytes.NewReader(iconPNG))
	if err != nil {
		return nil
	}
	var out []image.Image
	for _, n := range []int{16, 32, 64} {
		dst := image.NewRGBA(image.Rect(0, 0, n, n))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
		out = append(out, dst)
	}
	return out
}
