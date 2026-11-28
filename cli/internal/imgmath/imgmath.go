// Package imgmath provides pure dimension, target-size, and quality-search math.
package imgmath

import "math"

// MinTargetBytes prevents target byte budgets from collapsing on tiny images.
const MinTargetBytes int64 = 4096

// TargetSize scales original by percent using round-to-even with a 1px floor,
// matching .NET MathF.Round and Swift FloatingPointRoundingRule.toNearestOrEven.
func TargetSize(original, percent int) int {
	val := float64(original) * float64(percent) / 100.0
	return max(1, int(math.RoundToEven(val)))
}

// TargetSizeByMaxDim computes aspect-preserving dimensions capped at maxDim without upscaling.
func TargetSizeByMaxDim(width, height, maxDim int) (int, int) {
	if max(width, height) <= maxDim {
		return width, height
	}
	if width >= height {
		scaledH := max(1, int(math.RoundToEven(float64(height)*float64(maxDim)/float64(width))))
		return maxDim, scaledH
	}
	scaledW := max(1, int(math.RoundToEven(float64(width)*float64(maxDim)/float64(height))))
	return scaledW, maxDim
}

// TargetBytes returns the constant-compression-rate byte budget for outPixels.
func TargetBytes(sourceFileBytes, sourcePixels, outPixels int64) int64 {
	if sourcePixels <= 0 {
		return max(MinTargetBytes, sourceFileBytes)
	}
	return max(MinTargetBytes, sourceFileBytes*outPixels/sourcePixels)
}

// SolveQuality binary-searches [minQuality, maxQuality] for the highest quality whose
// encoded byte size does not exceed targetBytes. Returns minQuality if all exceed.
func SolveQuality(encode func(int) int64, targetBytes int64, minQuality, maxQuality int) int {
	if minQuality > maxQuality {
		panic("imgmath: minQuality must be <= maxQuality")
	}
	lo, hi := minQuality, maxQuality
	best := minQuality
	for lo <= hi {
		mid := lo + (hi-lo)/2
		if encode(mid) <= targetBytes {
			best = mid
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return best
}
