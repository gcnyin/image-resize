package exif

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"testing"
)

func buildSyntheticJPEG(order binary.ByteOrder, orientation uint16, includeAPP0 bool) []byte {
	var buf bytes.Buffer
	buf.Write([]byte{0xFF, 0xD8})
	if includeAPP0 {
		buf.Write([]byte{0xFF, 0xE0, 0x00, 0x06, 'J', 'F', 'I', 'F'})
	}

	var tiff bytes.Buffer
	if order == binary.LittleEndian {
		tiff.WriteString("II")
	} else {
		tiff.WriteString("MM")
	}
	var u16 [2]byte
	var u32 [4]byte
	order.PutUint16(u16[:], 42)
	tiff.Write(u16[:])
	order.PutUint32(u32[:], 8)
	tiff.Write(u32[:])

	// IFD0: 1 entry (tag 0x0112, type SHORT=3, count=1, value=orientation)
	order.PutUint16(u16[:], 1)
	tiff.Write(u16[:])
	order.PutUint16(u16[:], 0x0112)
	tiff.Write(u16[:])
	order.PutUint16(u16[:], 3)
	tiff.Write(u16[:])
	order.PutUint32(u32[:], 1)
	tiff.Write(u32[:])
	order.PutUint16(u16[:], orientation)
	tiff.Write(u16[:])
	tiff.Write([]byte{0, 0})
	order.PutUint32(u32[:], 0)
	tiff.Write(u32[:])

	app1Payload := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	buf.Write([]byte{0xFF, 0xE1})
	binary.BigEndian.PutUint16(u16[:], uint16(len(app1Payload)+2))
	buf.Write(u16[:])
	buf.Write(app1Payload)
	buf.Write([]byte{0xFF, 0xDA, 0x00, 0x02, 0xFF, 0xD9})
	return buf.Bytes()
}

func TestReadOrientation(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want Orientation
	}{
		{"little_endian_6", buildSyntheticJPEG(binary.LittleEndian, 6, false), Rotate90},
		{"big_endian_8_after_app0", buildSyntheticJPEG(binary.BigEndian, 8, true), Rotate270},
		{"out_of_range_returns_normal", buildSyntheticJPEG(binary.LittleEndian, 99, false), Normal},
		{"not_jpeg", []byte("PNG"), Normal},
		{"truncated_app1", []byte{0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x10, 'E', 'x', 'i', 'f'}, Normal},
		{"empty", nil, Normal},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ReadOrientation(tc.data); got != tc.want {
				t.Fatalf("ReadOrientation() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestApplyAllOrientations(t *testing.T) {
	// Asymmetric 3x2 image with unique color per pixel (1..6).
	src := image.NewRGBA(image.Rect(0, 0, 3, 2))
	pix := func(id uint8) color.RGBA { return color.RGBA{R: id, G: id * 10, B: 255 - id, A: 255} }
	src.SetRGBA(0, 0, pix(1))
	src.SetRGBA(1, 0, pix(2))
	src.SetRGBA(2, 0, pix(3))
	src.SetRGBA(0, 1, pix(4))
	src.SetRGBA(1, 1, pix(5))
	src.SetRGBA(2, 1, pix(6))

	if got := Apply(src, Normal); got != src {
		t.Fatal("Normal orientation should return src directly")
	}

	tests := []struct {
		o     Orientation
		wantW int
		wantH int
		topL  uint8
		topR  uint8
		botL  uint8
		botR  uint8
	}{
		{FlipHorizontal, 3, 2, 3, 1, 6, 4},
		{Rotate180, 3, 2, 6, 4, 3, 1},
		{FlipVertical, 3, 2, 4, 6, 1, 3},
		{Transpose, 2, 3, 1, 4, 3, 6},
		{Rotate90, 2, 3, 4, 1, 6, 3},
		{Transverse, 2, 3, 6, 3, 4, 1},
		{Rotate270, 2, 3, 3, 6, 1, 4},
	}

	for _, tc := range tests {
		out := Apply(src, tc.o).(*image.RGBA)
		b := out.Bounds()
		if b.Dx() != tc.wantW || b.Dy() != tc.wantH {
			t.Fatalf("o=%d size = %dx%d, want %dx%d", tc.o, b.Dx(), b.Dy(), tc.wantW, tc.wantH)
		}
		check := func(x, y int, want uint8, corner string) {
			if got := out.RGBAAt(x, y).R; got != want {
				t.Fatalf("o=%d %s (%d,%d) = %d, want %d", tc.o, corner, x, y, got, want)
			}
		}
		check(0, 0, tc.topL, "topL")
		check(tc.wantW-1, 0, tc.topR, "topR")
		check(0, tc.wantH-1, tc.botL, "botL")
		check(tc.wantW-1, tc.wantH-1, tc.botR, "botR")
	}
}
