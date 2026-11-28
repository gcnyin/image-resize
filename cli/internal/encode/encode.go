// Package encode implements constant-compression-rate image encoding for JPEG and PNG.
package encode

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math"

	"image-resize/cli/internal/format"
	"image-resize/cli/internal/imgmath"
)

const (
	MinQuality = 40
	MaxQuality = 97
)

type Result struct {
	Data        []byte
	Ext         string
	Quality     int
	TargetBytes int64
}

// Encode writes img using constant-compression-rate target sizing for JPEG and lossless encoding for PNG.
func Encode(img image.Image, kind format.Kind, sourceFileBytes, sourcePixels int64) (Result, error) {
	if img == nil {
		return Result{}, fmt.Errorf("空图像")
	}
	b := img.Bounds()
	outPixels := int64(b.Dx()) * int64(b.Dy())
	targetBytes := imgmath.TargetBytes(sourceFileBytes, sourcePixels, outPixels)

	switch kind {
	case format.PNG:
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return Result{}, fmt.Errorf("PNG 编码失败: %w", err)
		}
		return Result{
			Data:        buf.Bytes(),
			Ext:         "png",
			Quality:     0,
			TargetBytes: targetBytes,
		}, nil

	case format.JPEG:
		return encodeJPEG(img, targetBytes)

	default:
		return Result{}, fmt.Errorf("%s 暂不支持编码，仅支持 JPEG/PNG", kind)
	}
}

func encodeJPEG(img image.Image, targetBytes int64) (Result, error) {
	cached := make(map[int][]byte, 8)
	var encodeErr error

	probe := func(q int) int64 {
		if data, ok := cached[q]; ok {
			return int64(len(data))
		}
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
			if encodeErr == nil {
				encodeErr = err
			}
			return math.MaxInt64
		}
		data := buf.Bytes()
		cached[q] = data
		return int64(len(data))
	}

	quality := imgmath.SolveQuality(probe, targetBytes, MinQuality, MaxQuality)
	if encodeErr != nil {
		return Result{}, fmt.Errorf("JPEG 编码失败: %w", encodeErr)
	}
	data, ok := cached[quality]
	if !ok {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return Result{}, fmt.Errorf("JPEG 编码失败: %w", err)
		}
		data = buf.Bytes()
	}

	return Result{
		Data:        data,
		Ext:         "jpg",
		Quality:     quality,
		TargetBytes: targetBytes,
	}, nil
}
