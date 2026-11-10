import Foundation
import Clibwebp

/// libwebp 封装：ImageIO 只能解码 WebP（macOS 11+），编码交给 vendor 的 libwebp。
/// 输入统一为 RGBA8（premultiplied）像素缓冲。
enum WebPEncoder {
    /// 有损编码，quality 0~100。失败返回 nil。
    static func encodeLossy(pixels: UnsafePointer<UInt8>, width: Int, height: Int, stride: Int, quality: Int) -> Data? {
        var output: UnsafeMutablePointer<UInt8>?
        let size = WebPEncodeRGBA(
            pixels, Int32(width), Int32(height), Int32(stride), Float(quality), &output
        )
        guard size > 0, let output = output else { return nil }
        defer { WebPFree(output) }
        return Data(bytes: output, count: size)
    }

    /// 无损编码（quality 对无损用于控制编码努力程度，100 = 最佳压缩）。
    static func encodeLossless(pixels: UnsafePointer<UInt8>, width: Int, height: Int, stride: Int, quality: Int = 100) -> Data? {
        var config = WebPConfig()
        guard WebPConfigInit(&config) != 0 else { return nil }
        config.lossless = 1
        config.quality = Float(quality)
        config.method = 6  // 最高压缩努力（对文件体积有益，速度可接受）
        guard WebPValidateConfig(&config) != 0 else { return nil }

        var picture = WebPPicture()
        guard WebPPictureInit(&picture) != 0 else { return nil }
        defer { WebPPictureFree(&picture) }
        picture.width = Int32(width)
        picture.height = Int32(height)
        picture.use_argb = 1
        guard WebPPictureImportRGBA(&picture, pixels, Int32(stride)) != 0 else { return nil }

        var writer = WebPMemoryWriter()
        WebPMemoryWriterInit(&writer)
        var ok = false
        // & 只能用于调用参数，这里要长期持有地址，需显式取指针
        withUnsafeMutablePointer(to: &writer) { writerPtr in
            picture.writer = WebPMemoryWrite
            picture.custom_ptr = UnsafeMutableRawPointer(writerPtr)
            ok = WebPEncode(&config, &picture) != 0
        }
        defer { WebPMemoryWriterClear(&writer) }
        guard ok, writer.size > 0, let mem = writer.mem else { return nil }
        return Data(bytes: mem, count: writer.size)
    }
}
