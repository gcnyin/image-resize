// Package imgmath 质量二分搜索（对齐 macos/Services/ImageMath.swift 的 solveQuality）。
package imgmath

// SolveQuality 在 [minQuality, maxQuality] 内二分搜索不超过 targetBytes 的最大质量。
// encode 必须是关于质量的单调（非严格）递增的字节数函数。
// 全部超目标时返回 minQuality，全部达标时返回 maxQuality。
func SolveQuality(encode func(int) int64, targetBytes int64, minQuality, maxQuality int) int {
	lo, hi := minQuality, maxQuality
	best := minQuality
	for lo <= hi {
		mid := (lo + hi) / 2
		if encode(mid) <= targetBytes {
			best = mid
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return best
}
