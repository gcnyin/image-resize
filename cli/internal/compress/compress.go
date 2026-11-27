// Package compress 恒定压缩率编码（对齐 macos/Services/ImageEncoder.swift）。
// 只做压缩（无裁剪/缩放）：输出像素 == 源像素，目标字节数 = max(4096, 源文件字节)。
package compress

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"

	"image-resize/cli/internal/format"
	"image-resize/cli/internal/imgmath"
)

const (
	minQuality = 40 // 质量下限：避免画质崩坏出现明显色块
	maxQuality = 97 // 质量上限：编码器已能达到的极限清晰度
)

// minTargetBytes 与三端一致的字节下限。
const minTargetBytes = 4096

// ErrUnsupported 源格式不是 JPEG / PNG。
var ErrUnsupported = errors.New("仅支持 JPEG / PNG")

// Result 压缩输出与实际使用的质量（PNG 无损直通，Quality 为 0）。
type Result struct {
	Data    []byte
	Quality int
}

// Compress 按恒定压缩率重编码，保持源格式。
//
// JPEG：目标字节数 = max(4096, srcFileBytes)，在 [40, 97] 内二分搜
// 不超过目标的最大质量；全区间超目标时仍以 40 写出（允许略超目标）。
// PNG：无损直通重编码。
// 不处理 EXIF 方向：image/jpeg 解码即丢弃 EXIF。
func Compress(src image.Image, kind format.Kind, srcFileBytes int64) (Result, error) {
	switch kind {
	case format.JPEG:
		target := targetBytes(srcFileBytes)
		quality := imgmath.SolveQuality(
			func(q int) int64 {
				var buf bytes.Buffer
				if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: q}); err != nil {
					return 1 << 62 // 编码失败视为超大，二分走低质量方向
				}
				return int64(buf.Len())
			},
			target, minQuality, maxQuality,
		)
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: quality}); err != nil {
			return Result{}, err
		}
		return Result{Data: buf.Bytes(), Quality: quality}, nil
	case format.PNG:
		var buf bytes.Buffer
		if err := png.Encode(&buf, src); err != nil {
			return Result{}, err
		}
		return Result{Data: buf.Bytes()}, nil
	default:
		return Result{}, ErrUnsupported
	}
}

func targetBytes(srcFileBytes int64) int64 {
	if srcFileBytes < minTargetBytes {
		return minTargetBytes
	}
	return srcFileBytes
}
