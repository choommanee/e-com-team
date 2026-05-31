// Package promo builds a branded promotional graphic from a REAL product image
// (the affiliate doesn't invent products — this decorates the real photo with a
// price badge and headline ribbon for social posting).
package promo

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	_ "golang.org/x/image/webp" // decode Shopee webp images
)

//go:embed fonts/Sarabun-Bold.ttf
var sarabunBold []byte

const canvas = 1080

var (
	shopeeOrange = color.RGBA{0xEE, 0x4D, 0x2D, 0xFF}
	cream        = color.RGBA{0xFF, 0xF6, 0xEE, 0xFF}
	white        = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
	dark         = color.RGBA{0x20, 0x1A, 0x16, 0xFF}
)

// Builder renders promo graphics. It loads the embedded Thai font once.
type Builder struct {
	font   *opentype.Font
	client *http.Client
}

// NewBuilder parses the embedded font.
func NewBuilder() (*Builder, error) {
	f, err := opentype.Parse(sarabunBold)
	if err != nil {
		return nil, err
	}
	return &Builder{font: f, client: &http.Client{Timeout: 20 * time.Second}}, nil
}

// Build composites the real product image with a headline ribbon and price
// badge. If imageURL can't be fetched/decoded, a solid backdrop is used so the
// graphic still renders.
func (b *Builder) Build(imageURL, headline string, priceTHB, commissionPct float64) ([]byte, error) {
	dst := image.NewRGBA(image.Rect(0, 0, canvas, canvas))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{cream}, image.Point{}, draw.Src)

	// Product photo area (centered square with margin).
	const margin = 90
	box := image.Rect(margin, margin+90, canvas-margin, canvas-margin-40)
	if src := b.fetchImage(imageURL); src != nil {
		drawFitted(dst, box, src)
	} else {
		draw.Draw(dst, box, &image.Uniform{color.RGBA{0xE9, 0xDD, 0xD2, 0xFF}}, image.Point{}, draw.Src)
	}

	// Top headline ribbon.
	ribbon := image.Rect(0, 0, canvas, 120)
	draw.Draw(dst, ribbon, &image.Uniform{shopeeOrange}, image.Point{}, draw.Src)
	b.text(dst, clip(headline, 26), 40, 78, 44, white)

	// Price badge (bottom-left) — numerals render reliably.
	badge := image.Rect(40, canvas-150, 360, canvas-50)
	draw.Draw(dst, badge, &image.Uniform{shopeeOrange}, image.Point{}, draw.Src)
	b.text(dst, "฿"+formatInt(priceTHB), 60, canvas-85, 56, white)

	// Commission tag (bottom-right).
	if commissionPct > 0 {
		tag := image.Rect(canvas-360, canvas-150, canvas-40, canvas-50)
		draw.Draw(dst, tag, &image.Uniform{dark}, image.Point{}, draw.Src)
		b.text(dst, "คอม "+formatInt(commissionPct)+"%", canvas-340, canvas-85, 40, white)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (b *Builder) fetchImage(url string) image.Image {
	if url == "" {
		return nil
	}
	resp, err := b.client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 12<<20))
	if err != nil {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return img
}

// drawFitted scales src to fit inside box (preserving aspect) and centers it.
func drawFitted(dst *image.RGBA, box image.Rectangle, src image.Image) {
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw == 0 || sh == 0 {
		return
	}
	bw, bh := box.Dx(), box.Dy()
	scale := float64(bw) / float64(sw)
	if float64(sh)*scale > float64(bh) {
		scale = float64(bh) / float64(sh)
	}
	dw, dh := int(float64(sw)*scale), int(float64(sh)*scale)
	offX := box.Min.X + (bw-dw)/2
	offY := box.Min.Y + (bh-dh)/2
	// Nearest-neighbor scale (no external deps).
	for y := 0; y < dh; y++ {
		sy := sb.Min.Y + int(float64(y)/scale)
		for x := 0; x < dw; x++ {
			sx := sb.Min.X + int(float64(x)/scale)
			dst.Set(offX+x, offY+y, src.At(sx, sy))
		}
	}
}

// text draws a string at (x, baselineY) with the given pixel size.
func (b *Builder) text(dst *image.RGBA, s string, x, y int, size float64, col color.Color) {
	face, err := opentype.NewFace(b.font, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return
	}
	defer face.Close()
	d := &font.Drawer{Dst: dst, Src: &image.Uniform{col}, Face: face, Dot: fixed.P(x, y)}
	d.DrawString(sanitize(s))
}

// sanitize keeps only glyphs the Thai font can render (ASCII + Thai block +
// common spaces), dropping emoji/symbols that would show as missing-glyph boxes.
func sanitize(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r == ' ' || r == '\t':
			out = append(out, ' ')
		case r >= 0x20 && r <= 0x7E: // ASCII printable
			out = append(out, r)
		case r >= 0x0E00 && r <= 0x0E7F: // Thai
			out = append(out, r)
		}
	}
	return strings.TrimSpace(string(out))
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

func formatInt(f float64) string { return strconv.FormatInt(int64(f+0.5), 10) }

// ensure jpeg/png decoders are registered.
var _ = jpeg.Decode
var _ = png.Decode
