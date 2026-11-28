// Package format detects image container formats from magic headers and file extensions.
package format

import (
	"bytes"
	"path/filepath"
	"strings"
)

type Kind uint8

const (
	Unknown Kind = iota
	JPEG
	PNG
	WebP
	HEIF
)

func (k Kind) String() string {
	switch k {
	case JPEG:
		return "JPEG"
	case PNG:
		return "PNG"
	case WebP:
		return "WebP"
	case HEIF:
		return "HEIF"
	default:
		return "未知格式"
	}
}

var heifBrands = map[string]struct{}{
	"heic": {},
	"heix": {},
	"hevc": {},
	"hevx": {},
	"heim": {},
	"heis": {},
	"hevm": {},
	"hevs": {},
	"mif1": {},
	"msf1": {},
}

// Sniff inspects magic bytes in header to identify the image format.
func Sniff(header []byte) Kind {
	if len(header) >= 3 && header[0] == 0xFF && header[1] == 0xD8 && header[2] == 0xFF {
		return JPEG
	}
	if len(header) >= 8 && bytes.Equal(header[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		return PNG
	}
	if len(header) >= 12 && bytes.Equal(header[:4], []byte("RIFF")) && bytes.Equal(header[8:12], []byte("WEBP")) {
		return WebP
	}
	if len(header) >= 12 && bytes.Equal(header[4:8], []byte("ftyp")) {
		if _, ok := heifBrands[string(header[8:12])]; ok {
			return HEIF
		}
		return Unknown
	}
	return Unknown
}

// Detect identifies the format by sniffing header first, falling back to path extension.
func Detect(path string, header []byte) Kind {
	if kind := Sniff(header); kind != Unknown {
		return kind
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return JPEG
	case ".png":
		return PNG
	case ".webp":
		return WebP
	case ".heic", ".heif":
		return HEIF
	default:
		return Unknown
	}
}
