// Command genassets draws the pixel-art agent sprites used by the dashboard.
// Run with: go run ./tools/genassets
//
// Each sprite is a small "agent at a workstation" scene rendered in blocky
// pixels; the browser scales them with image-rendering: pixelated.
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
)

const dim = 160 // sprite size in px

type agent struct {
	slug   string
	shirt  color.RGBA
	screen color.RGBA
	skin   color.RGBA
	hair   color.RGBA
}

var (
	navy   = color.RGBA{0x0a, 0x14, 0x24, 0xff}
	panel  = color.RGBA{0x10, 0x1d, 0x3e, 0xff}
	desk   = color.RGBA{0x24, 0x1a, 0x12, 0xff}
	deskHi = color.RGBA{0x3a, 0x2a, 0x1c, 0xff}
	gray   = color.RGBA{0x2a, 0x3a, 0x60, 0xff}
)

func main() {
	skinA := color.RGBA{0xe8, 0xb8, 0x90, 0xff}
	skinB := color.RGBA{0xd9, 0xa6, 0x7a, 0xff}
	agents := []agent{
		{"benefit", rgb(0x3a, 0x7b, 0xd5), rgb(0x39, 0xff, 0x88), skinA, rgb(0x3a, 0x2a, 0x18)},
		{"promo", rgb(0xee, 0x4d, 0x2d), rgb(0xff, 0xd2, 0x3f), skinB, rgb(0x5a, 0x32, 0x1a)},
		{"design", rgb(0x1f, 0xa8, 0xa8), rgb(0x36, 0xe0, 0xff), skinA, rgb(0x20, 0x20, 0x28)},
		{"prompt", rgb(0x7a, 0x5c, 0xff), rgb(0xcf, 0xe0, 0xff), skinB, rgb(0x2a, 0x1c, 0x40)},
		{"studio", rgb(0xff, 0x9f, 0x43), rgb(0xff, 0x6a, 0x2d), skinA, rgb(0x3a, 0x2a, 0x18)},
		{"qc", rgb(0x36, 0xe0, 0xff), rgb(0x39, 0xff, 0x88), skinB, rgb(0x20, 0x28, 0x3a)},
	}

	outDir := filepath.Join("internal", "web", "static", "img")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatal(err)
	}
	for _, a := range agents {
		img := draw(a)
		path := filepath.Join(outDir, "agent-"+a.slug+".png")
		f, err := os.Create(path)
		if err != nil {
			log.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			log.Fatal(err)
		}
		f.Close()
		log.Printf("wrote %s", path)
	}
}

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 0xff} }

func draw(a agent) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, dim, dim))
	fill(img, image.Rect(0, 0, dim, dim), panel)

	// Subtle scanline / circuit background.
	for y := 0; y < dim; y += 8 {
		fill(img, image.Rect(0, y, dim, y+1), navy)
	}

	// Desk across the bottom.
	fill(img, image.Rect(0, 118, dim, dim), desk)
	fill(img, image.Rect(0, 118, dim, 122), deskHi)

	// Monitor on the right.
	fill(img, image.Rect(86, 40, 150, 92), gray)         // bezel
	fill(img, image.Rect(90, 44, 146, 88), a.screen)     // screen
	fill(img, image.Rect(96, 52, 140, 56), shade(a.screen)) // content lines
	fill(img, image.Rect(96, 62, 128, 66), shade(a.screen))
	fill(img, image.Rect(96, 72, 134, 76), shade(a.screen))
	fill(img, image.Rect(110, 92, 126, 118), gray)       // stand

	// Character (seated, left).
	// Body / shirt.
	fill(img, image.Rect(24, 84, 70, 122), a.shirt)
	fill(img, image.Rect(24, 84, 70, 90), shade(a.shirt)) // shoulders shade
	// Arm reaching toward desk.
	fill(img, image.Rect(58, 96, 86, 108), a.shirt)
	// Neck.
	fill(img, image.Rect(40, 76, 54, 86), a.skin)
	// Head.
	fill(img, image.Rect(34, 50, 60, 80), a.skin)
	// Hair.
	fill(img, image.Rect(32, 46, 62, 58), a.hair)
	fill(img, image.Rect(32, 50, 38, 70), a.hair)
	// Eyes.
	fill(img, image.Rect(40, 64, 44, 68), navy)
	fill(img, image.Rect(50, 64, 54, 68), navy)

	// Status LED top-left.
	fill(img, image.Rect(8, 8, 18, 18), a.screen)

	// Border frame.
	frame(img, rgb(0x1f, 0x3b, 0x6e))
	return img
}

func fill(img *image.RGBA, r image.Rectangle, c color.RGBA) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if x >= 0 && y >= 0 && x < dim && y < dim {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func frame(img *image.RGBA, c color.RGBA) {
	fill(img, image.Rect(0, 0, dim, 3), c)
	fill(img, image.Rect(0, dim-3, dim, dim), c)
	fill(img, image.Rect(0, 0, 3, dim), c)
	fill(img, image.Rect(dim-3, 0, dim, dim), c)
}

func shade(c color.RGBA) color.RGBA {
	return color.RGBA{c.R / 2, c.G / 2, c.B / 2, 0xff}
}
