import CoreGraphics
import Foundation
import ImageIO

enum ImageError: LocalizedError {
    case cannotOpen(String)
    case cannotDecode(String)
    case cannotEncode(String)

    var errorDescription: String? {
        switch self {
        case .cannotOpen(let m): return m
        case .cannotDecode(let m): return m
        case .cannotEncode(let m): return m
        }
    }
}

/// 图片加载流程：解码 → EXIF 摆正 → 限制工作尺寸。
/// ImageIO 的缩略图通道一次完成摆正 + 降采样（内部带高质量重采样），
/// 行为与 Win 版「解码 → 手工摆正 → 逐级 1/2 下采样到最长边 ≤ maxDim」等价。
enum ImageLoader {
    /// 加载工作区的最长边上限，避免超大图片占用工作内存。
    static let maxWorkDim = 4096

    static func load(path: String, maxDim: Int = ImageLoader.maxWorkDim) throws -> LoadedImage {
        let (kind, header) = ImageFormat.detect(path: path)
        let fileBytes = (try? FileManager.default.attributesOfItem(atPath: path)[.size] as? Int64) ?? 0

        guard let url = CFURLCreateWithFileSystemPath(nil, path as CFString, .cfurlposixPathStyle, false),
              let source = CGImageSourceCreateWithURL(url, nil) else {
            throw ImageError.cannotOpen("无法解析图片")
        }

        // 摆正后的原始尺寸：EXIF 90°/270° 时宽高互换
        let props = (CGImageSourceCopyPropertiesAtIndex(source, 0, nil) as? [CFString: Any]) ?? [:]
        let pixelW = props[kCGImagePropertyPixelWidth] as? Int ?? 0
        let pixelH = props[kCGImagePropertyPixelHeight] as? Int ?? 0
        let orientationValue = props[kCGImagePropertyOrientation] as? UInt32 ?? 1
        let swap = (5...8).contains(Int(orientationValue))
        let sourceWidth = swap ? pixelH : pixelW
        let sourceHeight = swap ? pixelW : pixelH

        let options: [CFString: Any] = [
            kCGImageSourceCreateThumbnailFromImageAlways: true,
            kCGImageSourceCreateThumbnailWithTransform: true,
            kCGImageSourceShouldCacheImmediately: true,
            kCGImageSourceThumbnailMaxPixelSize: maxDim,
        ]
        guard let cgImage = CGImageSourceCreateThumbnailAtIndex(source, 0, options as CFDictionary) else {
            throw ImageError.cannotDecode("解码失败")
        }

        return LoadedImage(
            cgImage: cgImage,
            format: kind,
            webpLossless: kind == .webp && ImageFormat.isWebpLossless(header),
            sourceWidth: sourceWidth,
            sourceHeight: sourceHeight,
            sourceFileBytes: fileBytes
        )
    }
}
