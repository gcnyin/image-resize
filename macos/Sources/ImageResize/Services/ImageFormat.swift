import Foundation

enum ImageFormatKind {
    case jpeg
    case png
    case webp
    case heif
    case unknown
}

/// 图片格式识别（按文件头魔数），用于保存时还原原格式。
enum ImageFormat {
    static let heifBrands: Set<String> = [
        "heic", "heix", "hevc", "hevx", "heim", "heis", "hevm", "hevs", "mif1", "msf1",
    ]

    /// 按文件头魔数判断格式。
    static func sniff(_ header: [UInt8]) -> ImageFormatKind {
        if header.count >= 3, header[0] == 0xFF, header[1] == 0xD8, header[2] == 0xFF {
            return .jpeg
        }
        if header.count >= 8,
           header[0] == 0x89, header[1] == 0x50, header[2] == 0x4E, header[3] == 0x47,
           header[4] == 0x0D, header[5] == 0x0A, header[6] == 0x1A, header[7] == 0x0A {
            return .png
        }
        if header.count >= 12,
           header[0] == 0x52, header[1] == 0x49, header[2] == 0x46, header[3] == 0x46,
           header[8] == 0x57, header[9] == 0x45, header[10] == 0x42, header[11] == 0x50 {
            return .webp
        }
        if header.count >= 12, header[4] == 0x66, header[5] == 0x74, header[6] == 0x79, header[7] == 0x70 {
            let brand = String(decoding: header[8..<12], as: UTF8.self)
            return heifBrands.contains(brand) ? .heif : .unknown
        }
        return .unknown
    }

    /// WebP 是否为无损（存在 VP8L chunk）：简单 WebP 的 chunk 在偏移 12，扩展 WebP（VP8X 头之后）约 30。
    ///
    /// TODO(webp-lossless): 这里只看固定偏移 12 / 30，且只读了 48 字节头。ICCP / EXIF / XMP 等元数据块
    /// 长度不固定，会把 VP8L 推到偏移之外，这类无损 WebP 会被误判为有损（信息栏只显示 WebP，
    /// 保存时按有损重编码）。Windows 端已改为按 RIFF 结构遍历 chunk（windows/Services/ImageFormat.cs），
    /// 这边需要同样处理（改为流式遍历 + 动图 ANMF 帧判定），Tests 里 12 / 30 的两条断言也要一起改。
    static func isWebpLossless(_ header: [UInt8]) -> Bool {
        func vp8l(_ h: [UInt8], at offset: Int) -> Bool {
            guard offset + 4 <= h.count else { return false }
            return h[offset] == 0x56 && h[offset + 1] == 0x50 && h[offset + 2] == 0x38 && h[offset + 3] == 0x4C
        }
        return vp8l(header, at: 12) || vp8l(header, at: 30)
    }

    /// 探测文件的格式：先魔数嗅探，UNKNOWN 时按扩展名兜底。返回格式与读取到的文件头。
    static func detect(path: String) -> (kind: ImageFormatKind, header: [UInt8]) {
        let header = readHeader(path: path, count: 48)
        let kind = sniff(header)
        if kind != .unknown { return (kind, header) }

        let ext = (path as NSString).pathExtension.lowercased()
        let fallback: ImageFormatKind
        switch ext {
        case "jpg", "jpeg": fallback = .jpeg
        case "png": fallback = .png
        case "webp": fallback = .webp
        case "heic", "heif": fallback = .heif
        default: fallback = .unknown
        }
        return (fallback, header)
    }

    private static func readHeader(path: String, count: Int) -> [UInt8] {
        guard let handle = FileHandle(forReadingAtPath: path) else { return [] }
        defer { try? handle.close() }
        let data = handle.readData(ofLength: count)
        return [UInt8](data)
    }
}
