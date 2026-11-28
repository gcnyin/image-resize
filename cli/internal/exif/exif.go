// Package exif reads JPEG EXIF orientation tags and applies pixel transforms.
package exif

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/draw"
)

type Orientation uint8

const (
	Normal         Orientation = 1
	FlipHorizontal Orientation = 2
	Rotate180      Orientation = 3
	FlipVertical   Orientation = 4
	Transpose      Orientation = 5
	Rotate90       Orientation = 6
	Transverse     Orientation = 7
	Rotate270      Orientation = 8
)

// ReadOrientation parses JPEG APP1 TIFF IFD0 tag 0x0112. Missing or invalid metadata returns Normal.
func ReadOrientation(jpegData []byte) Orientation {
	if len(jpegData) < 4 || jpegData[0] != 0xFF || jpegData[1] != 0xD8 {
		return Normal
	}
	pos := 2
	for pos < len(jpegData) {
		for pos < len(jpegData) && jpegData[pos] != 0xFF {
			pos++
		}
		for pos < len(jpegData) && jpegData[pos] == 0xFF {
			pos++
		}
		if pos >= len(jpegData) {
			return Normal
		}
		marker := jpegData[pos]
		pos++

		// EOI or SOS marks the end of metadata segments.
		if marker == 0xD9 || marker == 0xDA {
			return Normal
		}
		if marker == 0x00 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD8) {
			continue
		}
		if pos+2 > len(jpegData) {
			return Normal
		}
		segLen := int(binary.BigEndian.Uint16(jpegData[pos : pos+2]))
		if segLen < 2 || pos+segLen > len(jpegData) {
			return Normal
		}
		payload := jpegData[pos+2 : pos+segLen]
		pos += segLen

		if marker == 0xE1 && bytes.HasPrefix(payload, []byte("Exif\x00\x00")) {
			return parseTIFFOrientation(payload[6:])
		}
	}
	return Normal
}

func parseTIFFOrientation(tiff []byte) Orientation {
	if len(tiff) < 8 {
		return Normal
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return Normal
	}
	if order.Uint16(tiff[2:4]) != 42 {
		return Normal
	}
	ifd0 := int(order.Uint32(tiff[4:8]))
	if ifd0 < 8 || ifd0+2 > len(tiff) {
		return Normal
	}
	count := int(order.Uint16(tiff[ifd0 : ifd0+2]))
	for i := 0; i < count; i++ {
		off := ifd0 + 2 + i*12
		if off+12 > len(tiff) {
			return Normal
		}
		tag := order.Uint16(tiff[off : off+2])
		typ := order.Uint16(tiff[off+2 : off+4])
		cnt := order.Uint32(tiff[off+4 : off+8])
		if tag == 0x0112 && typ == 3 && cnt == 1 {
			val := order.Uint16(tiff[off+8 : off+10])
			if val >= uint16(Normal) && val <= uint16(Rotate270) {
				return Orientation(val)
			}
			return Normal
		}
	}
	return Normal
}

// Apply returns an upright copy of src per o. Normal or unknown values return src unchanged.
func Apply(src image.Image, o Orientation) image.Image {
	if src == nil || o <= Normal || o > Rotate270 {
		return src
	}
	rgba := toZeroOriginRGBA(src)
	w, h := rgba.Rect.Dx(), rgba.Rect.Dy()

	dstW, dstH := w, h
	if o >= Transpose {
		dstW, dstH = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch o {
			case FlipHorizontal:
				dx, dy = w-1-x, y
			case Rotate180:
				dx, dy = w-1-x, h-1-y
			case FlipVertical:
				dx, dy = x, h-1-y
			case Transpose:
				dx, dy = y, x
			case Rotate90:
				dx, dy = h-1-y, x
			case Transverse:
				dx, dy = h-1-y, w-1-x
			case Rotate270:
				dx, dy = y, w-1-x
			}
			si := y*rgba.Stride + x*4
			di := dy*dst.Stride + dx*4
			copy(dst.Pix[di:di+4], rgba.Pix[si:si+4])
		}
	}
	return dst
}

func toZeroOriginRGBA(src image.Image) *image.RGBA {
	b := src.Bounds()
	if rgba, ok := src.(*image.RGBA); ok && b.Min == (image.Point{}) {
		return rgba
	}
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}
