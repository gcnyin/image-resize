// Package format 按文件头魔数识别图片格式（对齐 macos/Services/ImageFormat.swift）。
package format

import "strings"

type Kind int

const (
	Unknown Kind = iota
	JPEG
	PNG
	WebP
	HEIF
)

var heifBrands = map[string]bool{
	"heic": true, "heix": true, "hevc": true, "hevx": true,
	"heim": true, "heis": true, "hevm": true, "hevs": true,
	"mif1": true, "msf1": true,
}

// Sniff 按文件头魔数判断格式。
func Sniff(header []byte) Kind {
	if len(header) >= 3 && header[0] == 0xFF && header[1] == 0xD8 && header[2] == 0xFF {
		return JPEG
	}
	if len(header) >= 8 &&
		header[0] == 0x89 && header[1] == 0x50 && header[2] == 0x4E && header[3] == 0x47 &&
		header[4] == 0x0D && header[5] == 0x0A && header[6] == 0x1A && header[7] == 0x0A {
		return PNG
	}
	if len(header) >= 12 &&
		header[0] == 'R' && header[1] == 'I' && header[2] == 'F' && header[3] == 'F' &&
		header[8] == 'W' && header[9] == 'E' && header[10] == 'B' && header[11] == 'P' {
		return WebP
	}
	if len(header) >= 12 &&
		header[4] == 'f' && header[5] == 't' && header[6] == 'y' && header[7] == 'p' {
		if heifBrands[string(header[8:12])] {
			return HEIF
		}
	}
	return Unknown
}

// Detect 探测文件格式：先魔数嗅探，Unknown 时按扩展名兜底。
func Detect(path string, header []byte) Kind {
	if k := Sniff(header); k != Unknown {
		return k
	}
	dot := strings.LastIndexByte(path, '.')
	if dot < 0 {
		return Unknown
	}
	switch strings.ToLower(path[dot+1:]) {
	case "jpg", "jpeg":
		return JPEG
	case "png":
		return PNG
	case "webp":
		return WebP
	case "heic", "heif":
		return HEIF
	}
	return Unknown
}
