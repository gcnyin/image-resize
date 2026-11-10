import CoreGraphics

/// 缩放与按百分比缩小操作。
enum ImageOps {
    /// 逐级缩放用中等插值（≈线性），最后精确到目标尺寸用高质量插值。
    ///
    /// 按百分比缩小图片。为避免一次缩小带来的锯齿，先按 1/2 逐级下采样；
    /// 目标尺寸到达后，再做一次高质量插值缩放到精确目标尺寸。
    /// 缩放不变形，只改变边长尺寸。
    static func scaleByPercent(_ src: CGImage, _ percent: Int) -> CGImage {
        guard (1...100).contains(percent) else {
            fatalError("百分比须在 1~100 之间")
        }
        let targetW = ImageMath.targetSize(src.width, percent)
        let targetH = ImageMath.targetSize(src.height, percent)
        if targetW == src.width && targetH == src.height { return src }

        var cur = src
        while cur.width / 2 >= targetW && cur.height / 2 >= targetH {
            cur = scale(cur, cur.width / 2, cur.height / 2, quality: .medium)
        }
        if cur.width != targetW || cur.height != targetH {
            cur = scale(cur, targetW, targetH, quality: .high)
        }
        return cur
    }

    /// 裁剪一个 [0,1] 归一化矩形内的位图。若覆盖全图则原样返回。
    static func crop(_ src: CGImage, _ norm: CGRect) -> CGImage {
        let x = clamp(Int((norm.minX * CGFloat(src.width)).rounded()), 0, src.width - 1)
        let y = clamp(Int((norm.minY * CGFloat(src.height)).rounded()), 0, src.height - 1)
        let w = clamp(Int((norm.width * CGFloat(src.width)).rounded()), 1, src.width - x)
        let h = clamp(Int((norm.height * CGFloat(src.height)).rounded()), 1, src.height - y)
        if x == 0 && y == 0 && w == src.width && h == src.height { return src }

        // CGImage.cropping(to:) 使用左上原点，与归一化坐标方向一致
        guard let sub = src.cropping(to: CGRect(x: x, y: y, width: w, height: h)) else {
            fatalError("裁剪失败")
        }
        return sub
    }

    /// 逐级 1/2 下采样，直到最长边不超过 maxDim。（加载主路径用 ImageIO 缩略图，此函数备用/测试）
    static func downsampleToCap(_ src: CGImage, _ maxDim: Int) -> CGImage {
        var cur = src
        while max(cur.width, cur.height) > maxDim {
            cur = scale(cur, max(1, cur.width / 2), max(1, cur.height / 2), quality: .medium)
        }
        return cur
    }

    private static func scale(_ src: CGImage, _ width: Int, _ height: Int, quality: CGInterpolationQuality) -> CGImage {
        let space = CGColorSpace(name: CGColorSpace.sRGB)!
        let ctx = CGContext(
            data: nil, width: width, height: height, bitsPerComponent: 8, bytesPerRow: 0,
            space: space, bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue
        )!
        ctx.interpolationQuality = quality
        ctx.draw(src, in: CGRect(x: 0, y: 0, width: width, height: height))
        return ctx.makeImage()!
    }

    private static func clamp(_ value: Int, _ lower: Int, _ upper: Int) -> Int {
        Swift.min(Swift.max(value, lower), upper)
    }
}
