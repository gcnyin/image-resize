package encode

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"image-resize/cli/internal/format"
	"image-resize/cli/internal/imgmath"
	"image-resize/cli/internal/ops"
)

func makePatternImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r := uint8((x * 255) / max(1, w-1))
			g := uint8((y * 255) / max(1, h-1))
			var b uint8
			if (x+y)%16 < 8 {
				b = 200
			} else {
				b = 40
			}
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

func TestEncodeJPEGConstantRate(t *testing.T) {
	src := makePatternImage(512, 512)
	scaled, err := ops.ScaleByPercent(src, 50)
	if err != nil {
		t.Fatalf("ScaleByPercent failed: %v", err)
	}

	sourceBytes := int64(200_000)
	sourcePixels := int64(512 * 512)
	res, err := Encode(scaled, format.JPEG, sourceBytes, sourcePixels)
	if err != nil {
		t.Fatalf("Encode JPEG failed: %v", err)
	}
	if res.Ext != "jpg" {
		t.Fatalf("Ext = %q, want jpg", res.Ext)
	}
	if res.Quality < MinQuality || res.Quality > MaxQuality {
		t.Fatalf("Quality = %d, outside [%d, %d]", res.Quality, MinQuality, MaxQuality)
	}
	if int64(len(res.Data)) > res.TargetBytes {
		t.Fatalf("encoded %d bytes > target %d", len(res.Data), res.TargetBytes)
	}
	decoded, err := jpeg.Decode(bytes.NewReader(res.Data))
	if err != nil {
		t.Fatalf("output JPEG cannot decode: %v", err)
	}
	if decoded.Bounds().Dx() != 256 || decoded.Bounds().Dy() != 256 {
		t.Fatalf("decoded bounds = %v, want 256x256", decoded.Bounds())
	}
}

func TestEncodeLowQualityJPEGDoesNotBloat(t *testing.T) {
	src := makePatternImage(512, 512)
	var lowQ bytes.Buffer
	if err := jpeg.Encode(&lowQ, src, &jpeg.Options{Quality: 30}); err != nil {
		t.Fatalf("seed jpeg encode failed: %v", err)
	}
	scaled, err := ops.ScaleByPercent(src, 50)
	if err != nil {
		t.Fatalf("ScaleByPercent failed: %v", err)
	}

	res, err := Encode(scaled, format.JPEG, int64(lowQ.Len()), int64(512*512))
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	if int64(len(res.Data)) >= int64(lowQ.Len()) {
		t.Fatalf("50%% scaled low-q JPEG %d >= source %d", len(res.Data), lowQ.Len())
	}
}

func TestEncodeTinyImageTargetFloor(t *testing.T) {
	src := makePatternImage(32, 32)
	res, err := Encode(src, format.JPEG, 800, 64*64)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	if res.TargetBytes != imgmath.MinTargetBytes {
		t.Fatalf("TargetBytes = %d, want %d", res.TargetBytes, imgmath.MinTargetBytes)
	}
}

func TestEncodePNGLosslessRoundtrip(t *testing.T) {
	src := makePatternImage(96, 72)
	res, err := Encode(src, format.PNG, 50_000, 96*72)
	if err != nil {
		t.Fatalf("Encode PNG failed: %v", err)
	}
	if res.Ext != "png" || res.Quality != 0 {
		t.Fatalf("got Ext=%q Quality=%d, want png / 0", res.Ext, res.Quality)
	}
	decoded, err := png.Decode(bytes.NewReader(res.Data))
	if err != nil {
		t.Fatalf("output PNG cannot decode: %v", err)
	}
	for y := 0; y < 72; y++ {
		for x := 0; x < 96; x++ {
			if decoded.At(x, y) != src.RGBAAt(x, y) {
				t.Fatalf("pixel mismatch at (%d,%d)", x, y)
			}
		}
	}
}

func TestEncodeUnsupportedFormat(t *testing.T) {
	src := makePatternImage(16, 16)
	for _, kind := range []format.Kind{format.WebP, format.HEIF, format.Unknown} {
		if _, err := Encode(src, kind, 1000, 256); err == nil {
			t.Fatalf("Encode(%v) should return error", kind)
		}
	}
}
