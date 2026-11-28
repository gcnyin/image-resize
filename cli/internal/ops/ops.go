// Package ops implements step-down 1/2 bilinear image scaling aligned with the GUI apps.
package ops

import (
	"fmt"
	"image"
	"image/draw"
	"math"

	"image-resize/cli/internal/imgmath"
)

// ScaleByPercent scales src by percent in [1, 100] using step-down 1/2 bilinear downsampling.
func ScaleByPercent(src image.Image, percent int) (image.Image, error) {
	if percent < 1 || percent > 100 {
		return nil, fmt.Errorf("百分比须在 1~100 之间: %d", percent)
	}
	if src == nil {
		return nil, fmt.Errorf("空图像")
	}
	b := src.Bounds()
	targetW := imgmath.TargetSize(b.Dx(), percent)
	targetH := imgmath.TargetSize(b.Dy(), percent)
	return ScaleToSize(src, targetW, targetH)
}

// ScaleToMaxDim scales src so its longest side is at most maxDim without upscaling.
func ScaleToMaxDim(src image.Image, maxDim int) (image.Image, error) {
	if maxDim < 1 {
		return nil, fmt.Errorf("最大边长须大于 0: %d", maxDim)
	}
	if src == nil {
		return nil, fmt.Errorf("空图像")
	}
	b := src.Bounds()
	targetW, targetH := imgmath.TargetSizeByMaxDim(b.Dx(), b.Dy(), maxDim)
	return ScaleToSize(src, targetW, targetH)
}

// ScaleToSize resamples src to targetW x targetH via progressive 1/2 bilinear steps.
func ScaleToSize(src image.Image, targetW, targetH int) (image.Image, error) {
	if src == nil {
		return nil, fmt.Errorf("空图像")
	}
	b := src.Bounds()
	srcW, srcH := b.Dx(), b.Dy()
	if srcW <= 0 || srcH <= 0 {
		return nil, fmt.Errorf("无效源尺寸: %dx%d", srcW, srcH)
	}
	if targetW < 1 || targetH < 1 {
		return nil, fmt.Errorf("无效目标尺寸: %dx%d", targetW, targetH)
	}
	if targetW == srcW && targetH == srcH {
		return src, nil
	}

	cur := toZeroOriginRGBA(src)
	for cur.Rect.Dx()/2 >= targetW && cur.Rect.Dy()/2 >= targetH {
		cur = scaleBilinear(cur, cur.Rect.Dx()/2, cur.Rect.Dy()/2)
	}
	if cur.Rect.Dx() != targetW || cur.Rect.Dy() != targetH {
		cur = scaleBilinear(cur, targetW, targetH)
	}
	return cur, nil
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

type sampleTap struct {
	i0 int
	i1 int
	w  float64
}

func buildAxisTaps(srcLen, dstLen int) []sampleTap {
	taps := make([]sampleTap, dstLen)
	scale := float64(srcLen) / float64(dstLen)
	for i := 0; i < dstLen; i++ {
		pos := (float64(i)+0.5)*scale - 0.5
		if pos <= 0 {
			taps[i] = sampleTap{i0: 0, i1: 0, w: 0}
			continue
		}
		if pos >= float64(srcLen-1) {
			last := srcLen - 1
			taps[i] = sampleTap{i0: last, i1: last, w: 0}
			continue
		}
		i0 := int(math.Floor(pos))
		i1 := min(i0+1, srcLen-1)
		taps[i] = sampleTap{i0: i0, i1: i1, w: pos - float64(i0)}
	}
	return taps
}

func scaleBilinear(src *image.RGBA, dstW, dstH int) *image.RGBA {
	srcW, srcH := src.Rect.Dx(), src.Rect.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	xTaps := buildAxisTaps(srcW, dstW)
	yTaps := buildAxisTaps(srcH, dstH)

	for dy := 0; dy < dstH; dy++ {
		yt := yTaps[dy]
		wy0 := 1.0 - yt.w
		wy1 := yt.w
		row0 := yt.i0 * src.Stride
		row1 := yt.i1 * src.Stride
		dstRow := dy * dst.Stride

		for dx := 0; dx < dstW; dx++ {
			xt := xTaps[dx]
			wx0 := 1.0 - xt.w
			wx1 := xt.w

			p00 := row0 + xt.i0*4
			p10 := row0 + xt.i1*4
			p01 := row1 + xt.i0*4
			p11 := row1 + xt.i1*4

			w00 := wx0 * wy0
			w10 := wx1 * wy0
			w01 := wx0 * wy1
			w11 := wx1 * wy1

			a := clampByte(
				float64(src.Pix[p00+3])*w00 +
					float64(src.Pix[p10+3])*w10 +
					float64(src.Pix[p01+3])*w01 +
					float64(src.Pix[p11+3])*w11,
			)
			r := clampByte(
				float64(src.Pix[p00])*w00 +
					float64(src.Pix[p10])*w10 +
					float64(src.Pix[p01])*w01 +
					float64(src.Pix[p11])*w11,
			)
			g := clampByte(
				float64(src.Pix[p00+1])*w00 +
					float64(src.Pix[p10+1])*w10 +
					float64(src.Pix[p01+1])*w01 +
					float64(src.Pix[p11+1])*w11,
			)
			b := clampByte(
				float64(src.Pix[p00+2])*w00 +
					float64(src.Pix[p10+2])*w10 +
					float64(src.Pix[p01+2])*w01 +
					float64(src.Pix[p11+2])*w11,
			)

			// Preserve premultiplied RGBA invariant (R, G, B <= A) after rounding.
			if r > a {
				r = a
			}
			if g > a {
				g = a
			}
			if b > a {
				b = a
			}

			di := dstRow + dx*4
			dst.Pix[di] = r
			dst.Pix[di+1] = g
			dst.Pix[di+2] = b
			dst.Pix[di+3] = a
		}
	}
	return dst
}

func clampByte(v float64) uint8 {
	r := math.Round(v)
	if r <= 0 {
		return 0
	}
	if r >= 255 {
		return 255
	}
	return uint8(r)
}
