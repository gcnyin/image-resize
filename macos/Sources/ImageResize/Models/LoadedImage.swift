import CoreGraphics

/// 本次加载的源图片信息与当前工作区图像。
struct LoadedImage {
    /// 工作区图像：已按 EXIF 摆正，最长边不超过 `ImageLoader.maxWorkDim`。
    let cgImage: CGImage
    let format: ImageFormatKind
    let webpLossless: Bool
    /// 摆正后的原始尺寸（EXIF 90°/270° 已互换宽高）。
    let sourceWidth: Int
    let sourceHeight: Int
    let sourceFileBytes: Int64
}
