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

    /// WebP 是否为无损（首个图像数据块是 VP8L）。
    ///
    /// 按 RIFF 结构逐个 chunk 遍历，不能只看固定偏移：ICCP / EXIF / XMP 等元数据块长度不固定，
    /// 会把 VP8L 推到任意位置；动图还要看 ANMF 帧内的数据块。与 Windows 端实现一致
    /// （windows/Services/ImageFormat.cs → IsWebpLossless）。
    static func isWebpLossless(path: String) -> Bool {
        guard let handle = FileHandle(forReadingAtPath: path) else { return false }
        defer { try? handle.close() }

        func read(_ count: Int) -> [UInt8]? {
            guard let data = try? handle.read(upToCount: count), data.count == count else { return nil }
            return [UInt8](data)
        }

        guard let head = read(12),
              head[0..<4].elementsEqual(Array("RIFF".utf8)),
              head[8..<12].elementsEqual(Array("WEBP".utf8))
        else { return false }

        while let chunk = read(8) {
            let fourcc = String(decoding: chunk[0..<4], as: UTF8.self)
            let size = UInt32(chunk[4]) | UInt32(chunk[5]) << 8
                | UInt32(chunk[6]) << 16 | UInt32(chunk[7]) << 24

            switch fourcc {
            case "VP8L":
                return true // 无损图像数据
            case "VP8 ":
                return false // 有损图像数据
            case "ANMF":
                // 动图帧：帧内前 16 字节是帧头，之后才是该帧的图像数据块
                guard let frame = read(20) else { return false }
                return String(decoding: frame[16..<20], as: UTF8.self) == "VP8L"
            default:
                break
            }

            // 其余是元数据/容器块（VP8X / ICCP / EXIF / XMP / ALPH / ANIM …），整块跳过；RIFF 块按偶数字节对齐
            let skip = UInt64(size) + UInt64(size & 1)
            guard let position = try? handle.offset(),
                  (try? handle.seek(toOffset: position + skip)) != nil
            else { return false }
        }
        return false
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
