package compress

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"image-resize/cli/internal/format"
)

// genJPEG 生成一张带渐变的 JPEG（quality 拉低保证有可压缩空间）。
func genJPEG(t *testing.T, w, h int, quality int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 255 / w), G: uint8(y * 255 / h), B: uint8((x + y) * 255 / (w + h)), A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func genPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestCompressJPEG(t *testing.T) {
	src := genJPEG(t, 512, 512, 98) // 高质量源，压缩空间大
	if len(src) < minTargetBytes {
		t.Fatalf("测试源图太小: %d 字节", len(src))
	}
	decoded, err := jpeg.Decode(bytes.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	res, err := Compress(decoded, format.JPEG, int64(len(src)))
	if err != nil {
		t.Fatal(err)
	}
	if res.Quality < minQuality || res.Quality > maxQuality {
		t.Errorf("质量越界: %d", res.Quality)
	}
	// 输出是合法 JPEG，且尺寸不变。
	out, err := jpeg.Decode(bytes.NewReader(res.Data))
	if err != nil {
		t.Fatalf("输出不是合法 JPEG: %v", err)
	}
	if out.Bounds() != decoded.Bounds() {
		t.Errorf("尺寸变了: %v -> %v", decoded.Bounds(), out.Bounds())
	}
	if int64(len(res.Data)) > targetBytes(int64(len(src))) {
		t.Errorf("超出目标字节: %d > %d", len(res.Data), len(src))
	}
}

func TestCompressJPEGTinySource(t *testing.T) {
	// 小于 4096 字节的源：目标抬到 4096，输出允许比源大，但必须是合法 JPEG。
	src := genJPEG(t, 32, 32, 50)
	decoded, err := jpeg.Decode(bytes.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	res, err := Compress(decoded, format.JPEG, int64(len(src)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jpeg.Decode(bytes.NewReader(res.Data)); err != nil {
		t.Fatalf("输出不是合法 JPEG: %v", err)
	}
}

func TestCompressPNG(t *testing.T) {
	src := genPNG(t, 256, 256)
	decoded, err := png.Decode(bytes.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	res, err := Compress(decoded, format.PNG, int64(len(src)))
	if err != nil {
		t.Fatal(err)
	}
	out, err := png.Decode(bytes.NewReader(res.Data))
	if err != nil {
		t.Fatalf("输出不是合法 PNG: %v", err)
	}
	if res.Quality != 0 {
		t.Errorf("PNG 不应有质量值: %d", res.Quality)
	}
	if out.Bounds() != decoded.Bounds() {
		t.Errorf("尺寸变了: %v -> %v", decoded.Bounds(), out.Bounds())
	}
}

func TestCompressUnsupported(t *testing.T) {
	_, err := Compress(image.NewRGBA(image.Rect(0, 0, 1, 1)), format.WebP, 100)
	if err != ErrUnsupported {
		t.Errorf("want ErrUnsupported, got %v", err)
	}
}

func TestTargetBytes(t *testing.T) {
	if got := targetBytes(100); got != minTargetBytes {
		t.Errorf("小文件: got %d want %d", got, minTargetBytes)
	}
	if got := targetBytes(100_000); got != 100_000 {
		t.Errorf("大文件: got %d want 100000", got)
	}
}
