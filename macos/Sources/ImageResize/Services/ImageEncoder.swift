import CoreGraphics
import Foundation
import ImageIO

/// 恒定压缩率编码：目标字节数 = max(4096, 源文件字节数 × 输出像素 / 源像素)。
/// 保证缩小后的文件不会比原图大，且信息密度与原图一致。
///
/// PNG 直通无损；JPEG / 有损 WebP / HEIC 按目标大小二分搜质量。
/// WebP 编码走 vendor 的 libwebp（ImageIO 只能解码不能编码），
/// 无损 WebP 源用 libwebp 无损直通（与 Win 版 Skia/libwebp 行为一致）。
enum ImageEncoder {
    /// 质量下限（百分比口径）：避免画质崩坏出现明显色块。
    static let minQuality = 40
    /// 质量上限：平台编码器已能达到的极限清晰度。
    static let maxQuality = 97

    struct Encoded {
        let data: Data
        let mime: String
        let ext: String
    }

    private static let jpegType = "public.jpeg" as CFString
    private static let pngType = "public.png" as CFString
    private static let heicType = "public.heic" as CFString

    static func encode(
        _ image: CGImage,
        sourceKind: ImageFormatKind,
        sourceFileBytes: Int64,
        sourcePixels: Int64,
        sourceWebpLossless: Bool
    ) throws -> Encoded {
        let outPixels = Int64(image.width) * Int64(image.height)
        // 与源图保持相同的每像素字节数，即相同压缩率 / 信息密度。
        let targetBytes = Swift.max(4096, sourceFileBytes * outPixels / sourcePixels)

        switch sourceKind {
        case .png:
            return Encoded(data: try compress(image, type: pngType, lossyQuality: nil), mime: "image/png", ext: "png")
        case .webp where sourceWebpLossless:
            return try encodeWebp(image) { WebPEncoder.encodeLossless(pixels: $0, width: $1, height: $2, stride: $3) }
        case .webp:
            return try encodeWebpByTarget(image, targetBytes: targetBytes)
        case .heif:
            // Win 版 HEIF 只能回退 JPEG；macOS ImageIO 原生支持 HEIC 编码，保留原格式。
            return try encodeByTarget(image, type: heicType, targetBytes: targetBytes, mime: "image/heic", ext: "heic")
        // JPEG、未知格式 → JPEG
        case .jpeg, .unknown:
            return try encodeByTarget(image, type: jpegType, targetBytes: targetBytes, mime: "image/jpeg", ext: "jpg")
        }
    }

    // MARK: - 栅格化（RGBA8 premul，WebP 编码输入）

    /// 栅格化为固定格式像素数据（RGBA8 premul），用于 WebP 编码与无损比对。
    static func pixelData(of image: CGImage) throws -> Data {
        guard let data = rasterize(image) else {
            throw ImageError.cannotEncode("栅格化失败")
        }
        return data
    }

    private static func rasterize(_ image: CGImage) -> Data? {
        let space = CGColorSpace(name: CGColorSpace.sRGB)!
        let w = image.width, h = image.height
        guard let ctx = CGContext(
            data: nil, width: w, height: h, bitsPerComponent: 8, bytesPerRow: w * 4,
            space: space, bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue
        ) else { return nil }
        ctx.draw(image, in: CGRect(x: 0, y: 0, width: w, height: h))
        guard let raw = ctx.data else { return nil }
        return Data(bytes: raw, count: w * h * 4)
    }

    // MARK: - WebP（libwebp）

    private static func encodeWebp(
        _ image: CGImage,
        encode: (UnsafePointer<UInt8>, Int, Int, Int) -> Data?
    ) throws -> Encoded {
        let data = try pixelData(of: image)
        guard data.count == image.width * image.height * 4 else {
            throw ImageError.cannotEncode("WebP 编码失败")
        }
        let encoded = data.withUnsafeBytes { raw -> Data? in
            guard let base = raw.bindMemory(to: UInt8.self).baseAddress else { return nil }
            return encode(base, image.width, image.height, image.width * 4)
        }
        guard let encoded = encoded else {
            throw ImageError.cannotEncode("WebP 编码失败")
        }
        return Encoded(data: encoded, mime: "image/webp", ext: "webp")
    }

    /// 有损 WebP：二分搜索不超过目标字节数的最高质量。
    private static func encodeWebpByTarget(_ image: CGImage, targetBytes: Int64) throws -> Encoded {
        let data = try pixelData(of: image)
        guard data.count == image.width * image.height * 4 else {
            throw ImageError.cannotEncode("WebP 编码失败")
        }
        let quality = ImageMath.solveQuality(
            encode: { q -> Int64 in
                data.withUnsafeBytes { raw -> Int64 in
                    guard let base = raw.bindMemory(to: UInt8.self).baseAddress else { return Int64.max }
                    let out = WebPEncoder.encodeLossy(
                        pixels: base, width: image.width, height: image.height,
                        stride: image.width * 4, quality: q
                    )
                    return Int64(out?.count ?? Int.max)
                }
            },
            targetBytes: targetBytes,
            minQuality: minQuality,
            maxQuality: maxQuality
        )
        guard let encoded = data.withUnsafeBytes({ raw -> Data? in
            raw.bindMemory(to: UInt8.self).baseAddress.flatMap {
                WebPEncoder.encodeLossy(
                    pixels: $0, width: image.width, height: image.height,
                    stride: image.width * 4, quality: quality
                )
            }
        }) else {
            throw ImageError.cannotEncode("WebP 编码失败")
        }
        return Encoded(data: encoded, mime: "image/webp", ext: "webp")
    }

    // MARK: - ImageIO（JPEG / PNG / HEIC）

    /// 二分搜索不超过目标字节数的最高质量。
    private static func encodeByTarget(
        _ image: CGImage, type: CFString, targetBytes: Int64, mime: String, ext: String
    ) throws -> Encoded {
        let quality = ImageMath.solveQuality(
            encode: { q in
                // 编码失败视为超大，二分自然走低质量方向
                guard let data = try? compress(image, type: type, lossyQuality: Double(q) / 100.0) else {
                    return Int64.max
                }
                return Int64(data.count)
            },
            targetBytes: targetBytes,
            minQuality: minQuality,
            maxQuality: maxQuality
        )
        let data = try compress(image, type: type, lossyQuality: Double(quality) / 100.0)
        return Encoded(data: data, mime: mime, ext: ext)
    }

    private static func compress(_ image: CGImage, type: CFString, lossyQuality: Double?) throws -> Data {
        let data = NSMutableData()
        guard let dest = CGImageDestinationCreateWithData(data, type, 1, nil) else {
            throw ImageError.cannotEncode("编码器不可用")
        }
        var props: [CFString: Any] = [:]
        if let q = lossyQuality { props[kCGImageDestinationLossyCompressionQuality] = q }
        CGImageDestinationAddImage(dest, image, props as CFDictionary)
        guard CGImageDestinationFinalize(dest) else {
            throw ImageError.cannotEncode("编码失败")
        }
        return data as Data
    }
}
