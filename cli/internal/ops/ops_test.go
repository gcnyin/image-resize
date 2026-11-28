package ops

import (
	"image"
	"image/color"
	"testing"
)

func makeSolid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestScaleByPercentSizes(t *testing.T) {
	src := makeSolid(100, 60, color.RGBA{R: 120, G: 80, B: 40, A: 255})

	half, err := ScaleByPercent(src, 50)
	if err != nil {
		t.Fatalf("ScaleByPercent(50) failed: %v", err)
	}
	if half.Bounds().Dx() != 50 || half.Bounds().Dy() != 30 {
		t.Fatalf("50%% size = %dx%d, want 50x30", half.Bounds().Dx(), half.Bounds().Dy())
	}

	tenth, err := ScaleByPercent(src, 10)
	if err != nil {
		t.Fatalf("ScaleByPercent(10) failed: %v", err)
	}
	if tenth.Bounds().Dx() != 10 || tenth.Bounds().Dy() != 6 {
		t.Fatalf("10%% size = %dx%d, want 10x6", tenth.Bounds().Dx(), tenth.Bounds().Dy())
	}

	same, err := ScaleByPercent(src, 100)
	if err != nil {
		t.Fatalf("ScaleByPercent(100) failed: %v", err)
	}
	if same != src {
		t.Fatal("100% scale should return src unchanged")
	}
}

func TestScaleToMaxDim(t *testing.T) {
	landscape := makeSolid(400, 200, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	scaled, err := ScaleToMaxDim(landscape, 100)
	if err != nil {
		t.Fatalf("ScaleToMaxDim failed: %v", err)
	}
	if scaled.Bounds().Dx() != 100 || scaled.Bounds().Dy() != 50 {
		t.Fatalf("got %dx%d, want 100x50", scaled.Bounds().Dx(), scaled.Bounds().Dy())
	}

	noop, err := ScaleToMaxDim(landscape, 500)
	if err != nil {
		t.Fatalf("ScaleToMaxDim noop failed: %v", err)
	}
	if noop != landscape {
		t.Fatal("maxDim >= longest side should return src unchanged")
	}
}

func TestScalePreservesSolidColorAndPremulInvariant(t *testing.T) {
	// Left half opaque white, right half transparent (0,0,0,0).
	src := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 8; x++ {
			src.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	out, err := ScaleByPercent(src, 50)
	if err != nil {
		t.Fatalf("ScaleByPercent failed: %v", err)
	}
	rgba := out.(*image.RGBA)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			c := rgba.RGBAAt(x, y)
			if c.R > c.A || c.G > c.A || c.B > c.A {
				t.Fatalf("premultiplied invariant violated at (%d,%d): %+v", x, y, c)
			}
			// Every valid white/transparent blend keeps R==G==B==A.
			if c.R != c.A || c.G != c.A || c.B != c.A {
				t.Fatalf("unexpected dark fringe at (%d,%d): %+v", x, y, c)
			}
		}
	}
}

func TestScaleNonZeroOriginAndValidation(t *testing.T) {
	sub := image.NewRGBA(image.Rect(10, 20, 30, 40))
	for y := 20; y < 40; y++ {
		for x := 10; x < 30; x++ {
			sub.SetRGBA(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	out, err := ScaleByPercent(sub, 50)
	if err != nil {
		t.Fatalf("ScaleByPercent non-zero origin failed: %v", err)
	}
	if out.Bounds() != image.Rect(0, 0, 10, 10) {
		t.Fatalf("bounds = %v, want (0,0)-(10,10)", out.Bounds())
	}
	c := out.(*image.RGBA).RGBAAt(5, 5)
	if c != (color.RGBA{R: 200, G: 100, B: 50, A: 255}) {
		t.Fatalf("pixel = %+v, want {200 100 50 255}", c)
	}

	if _, err := ScaleByPercent(sub, 0); err == nil {
		t.Fatal("percent=0 should error")
	}
	if _, err := ScaleByPercent(sub, 101); err == nil {
		t.Fatal("percent=101 should error")
	}
	if _, err := ScaleToMaxDim(sub, 0); err == nil {
		t.Fatal("maxDim=0 should error")
	}
}
