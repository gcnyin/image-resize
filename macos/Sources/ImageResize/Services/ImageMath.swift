import Foundation

/// 目标尺寸与质量搜索的纯数学逻辑，独立于 ImageIO，便于单元测试。
enum ImageMath {
    /// 按百分比计算目标边长。四舍五入且 .5 靠偶数（与 .NET `MathF.Round` 默认行为一致），至少 1px。
    static func targetSize(_ original: Int, _ percent: Int) -> Int {
        let value = Double(original) * Double(percent) / 100.0
        return max(1, Int(value.rounded(.toNearestOrEven)))
    }

    /// 在 [minQuality, maxQuality] 内二分搜索不超过 `targetBytes` 的最大质量。
    /// `encode` 必须是关于质量的单调（非严格）递增的字节数函数。
    /// 全部超目标时返回 minQuality，全部达标时返回 maxQuality。
    static func solveQuality(
        encode: (Int) -> Int64,
        targetBytes: Int64,
        minQuality: Int,
        maxQuality: Int
    ) -> Int {
        precondition(minQuality <= maxQuality, "质量区间不合法")
        var lo = minQuality
        var hi = maxQuality
        var best = minQuality
        while lo <= hi {
            let mid = (lo + hi) / 2
            if encode(mid) <= targetBytes {
                best = mid
                lo = mid + 1
            } else {
                hi = mid - 1
            }
        }
        return best
    }
}
