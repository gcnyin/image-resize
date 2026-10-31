package com.gcnyin.imagetool

import kotlin.math.roundToInt

/** 纯尺寸计算逻辑，不依赖 Android 类，便于单元测试。 */
object ImageMath {

    /** 按百分比计算目标边长：四舍五入，且至少 1px。 */
    fun targetSize(original: Int, percent: Int): Int =
        maxOf(1, (original * percent / 100.0).roundToInt())

    /** 计算解码采样步长：最大的 2 的幂，使解码后最长边不超过 [maxDim]。 */
    fun inSampleSizeForCap(width: Int, height: Int, maxDim: Int): Int {
        var sample = 1
        val max = maxOf(width, height)
        while (max / sample > maxDim) sample *= 2
        return sample
    }

    /**
     * 在 [minQuality, maxQuality] 内二分查找不超出 [targetBytes] 的最大编码质量。
     * [encode] 返回指定质量的编码字节数（应随质量单调不减）。
     * 全部超出目标时返回 [minQuality]，全部满足时返回 [maxQuality]。
     */
    fun solveQuality(
        encode: (Int) -> Long,
        targetBytes: Long,
        minQuality: Int,
        maxQuality: Int,
    ): Int {
        require(minQuality <= maxQuality) { "质量区间不合法" }
        var lo = minQuality
        var hi = maxQuality
        var best = minQuality
        while (lo <= hi) {
            val mid = (lo + hi) / 2
            if (encode(mid) <= targetBytes) {
                best = mid
                lo = mid + 1
            } else {
                hi = mid - 1
            }
        }
        return best
    }
}
