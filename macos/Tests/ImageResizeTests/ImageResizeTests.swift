import XCTest
@testable import ImageResize

final class ImageResizeTests: XCTestCase {
    // ---------------------------------------------------------------- 工具

    /// 合成一张带渐变与条纹的小图（有足够的信息量让 JPEG 质量二分有意义）。
    private func makeImage(width: Int, height: Int) -> CGImage {
        let space = CGColorSpace(name: CGColorSpace.sRGB)!
        let ctx = CGContext(
            data: nil, width: width, height: height, bitsPerComponent: 8, bytesPerRow: 0,
            space: space, bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue
        )!
        for y in 0..<height {
            for x in 0..<width {
                let t = Double(x + y) / Double(width + height)
                let stripe = (x + y) % 16 < 8 ? 0.15 : 0.0
                ctx.setFillColor(
                    red: CGFloat(t), green: CGFloat(stripe + t * 0.5), blue: CGFloat(1 - t * 0.7), alpha: 1
                )
                ctx.fill(CGRect(x: x, y: y, width: 1, height: 1))
            }
        }
        return ctx.makeImage()!
    }

    private func tempFile(name: String) -> URL {
        let url = FileManager.default.temporaryDirectory.appendingPathComponent("imgresize-\(name)")
        try? FileManager.default.removeItem(at: url)
        return url
    }

    private func writePNG(_ image: CGImage, to url: URL) {
        let dest = CGImageDestinationCreateWithURL(url as CFURL, "public.png" as CFString, 1, nil)!
        CGImageDestinationAddImage(dest, image, nil)
        XCTAssertTrue(CGImageDestinationFinalize(dest))
    }

    // ---------------------------------------------------------------- ImageFormat

    func testSniffMagicNumbers() {
        XCTAssertEqual(ImageFormat.sniff([0xFF, 0xD8, 0xFF, 0xE0]), .jpeg)
        XCTAssertEqual(ImageFormat.sniff([0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0]), .png)
        XCTAssertEqual(
            ImageFormat.sniff([0x52, 0x49, 0x46, 0x46, 0, 0, 0, 0, 0x57, 0x45, 0x42, 0x50]),
            .webp
        )
        // ftyp 盒：brand heic → HEIF；brand mp42 → 未知
        XCTAssertEqual(
            ImageFormat.sniff(Array("    ftypheic....".utf8)),
            .heif
        )
        XCTAssertEqual(
            ImageFormat.sniff(Array("    ftypmp42 ....".utf8)),
            .unknown
        )
        XCTAssertEqual(ImageFormat.sniff([1, 2, 3]), .unknown)
        XCTAssertEqual(ImageFormat.sniff([]), .unknown)
    }

    func testWebpLosslessDetection() {
        func withChunk(_ offset: Int, bytes: [UInt8], in header: [UInt8]) -> [UInt8] {
            var h = header
            h.replaceSubrange(offset..<offset + 4, with: bytes)
            return h
        }
        let base = [UInt8](repeating: 0, count: 48)
        let vp8l: [UInt8] = [0x56, 0x50, 0x38, 0x4C]
        XCTAssertTrue(ImageFormat.isWebpLossless(withChunk(12, bytes: vp8l, in: base)))
        XCTAssertTrue(ImageFormat.isWebpLossless(withChunk(30, bytes: vp8l, in: base)))
        XCTAssertFalse(ImageFormat.isWebpLossless(base))
    }

    // ---------------------------------------------------------------- ImageMath

    func testTargetSize() {
        // .5 靠偶数（与 .NET MathF.Round 一致）：2.5→2、3.5→4
        XCTAssertEqual(ImageMath.targetSize(5, 50), 2)
        XCTAssertEqual(ImageMath.targetSize(7, 50), 4)
        XCTAssertEqual(ImageMath.targetSize(1000, 33), 330)
        XCTAssertEqual(ImageMath.targetSize(1000, 10), 100)
        XCTAssertEqual(ImageMath.targetSize(1, 1), 1)
        XCTAssertEqual(ImageMath.targetSize(99, 200), 198)
    }

    func testSolveQuality() {
        // 单调递增：字节 = 质量 × 10 + 50（encode(97) = 1020）
        let encode = { Int64($0 * 10 + 50) }
        // 全部达标 → max
        XCTAssertEqual(ImageMath.solveQuality(encode: encode, targetBytes: 2000, minQuality: 40, maxQuality: 97), 97)
        // 全部超目标 → min
        XCTAssertEqual(ImageMath.solveQuality(encode: encode, targetBytes: 100, minQuality: 40, maxQuality: 97), 40)
        // 边界：质量 55 产出 600 ≤ 600，56 产出 610 > 600
        XCTAssertEqual(ImageMath.solveQuality(encode: encode, targetBytes: 600, minQuality: 40, maxQuality: 97), 55)
    }

    // ---------------------------------------------------------------- ImageOps

    func testScaleByPercentSizes() {
        let image = makeImage(width: 100, height: 60)
        XCTAssertEqual(ImageOps.scaleByPercent(image, 50).width, 50)
        XCTAssertEqual(ImageOps.scaleByPercent(image, 50).height, 30)
        XCTAssertEqual(ImageOps.scaleByPercent(image, 10).width, 10)
        // 100% 原样返回
        XCTAssertTrue(ImageOps.scaleByPercent(image, 100) === image)
    }

    func testCropNormalized() {
        let image = makeImage(width: 100, height: 60)
        let left = ImageOps.crop(image, CGRect(x: 0, y: 0, width: 0.5, height: 1))
        XCTAssertEqual(left.width, 50)
        XCTAssertEqual(left.height, 60)
        let center = ImageOps.crop(image, CGRect(x: 0.25, y: 0.25, width: 0.5, height: 0.5))
        XCTAssertEqual(center.width, 50)
        XCTAssertEqual(center.height, 30)
        // 全图原样返回
        XCTAssertTrue(ImageOps.crop(image, CGRect(x: 0, y: 0, width: 1, height: 1)) === image)
    }

    // ---------------------------------------------------------------- ImageLoader

    func testLoadCapsLongestSide() throws {
        let big = makeImage(width: 6000, height: 3000)
        let url = tempFile(name: "big.png")
        writePNG(big, to: url)
        defer { try? FileManager.default.removeItem(at: url) }

        let loaded = try ImageLoader.load(path: url.path)
        XCTAssertEqual(loaded.cgImage.width, ImageLoader.maxWorkDim)
        XCTAssertEqual(loaded.cgImage.height, 2048)
        XCTAssertEqual(loaded.sourceWidth, 6000)
        XCTAssertEqual(loaded.sourceHeight, 3000)
        XCTAssertEqual(loaded.format, .png)
        XCTAssertGreaterThan(loaded.sourceFileBytes, 0)
    }

    func testLoadSmallImageKeepsSize() throws {
        let small = makeImage(width: 320, height: 200)
        let url = tempFile(name: "small.png")
        writePNG(small, to: url)
        defer { try? FileManager.default.removeItem(at: url) }

        let loaded = try ImageLoader.load(path: url.path)
        XCTAssertEqual(loaded.cgImage.width, 320)
        XCTAssertEqual(loaded.cgImage.height, 200)
    }

    // ---------------------------------------------------------------- ImageEncoder

    func testEncodeJpegConstantRate() throws {
        let image = makeImage(width: 512, height: 512)
        let sourceBytes: Int64 = 200_000
        // 缩到 1/2 边长 → 像素 1/4 → 目标字节数 ≈ 50 KB
        let half = ImageOps.scaleByPercent(image, 50)
        let encoded = try ImageEncoder.encode(
            half,
            sourceKind: .jpeg,
            sourceFileBytes: sourceBytes,
            sourcePixels: Int64(image.width) * Int64(image.height),
            sourceWebpLossless: false
        )
        let target = sourceBytes * Int64(half.width) * Int64(half.height)
            / (Int64(image.width) * Int64(image.height))
        XCTAssertEqual(encoded.ext, "jpg")
        XCTAssertLessThanOrEqual(Int64(encoded.data.count), target)
        XCTAssertGreaterThan(encoded.data.count, 0)
    }

    func testEncodePngLosslessRoundtrip() throws {
        let image = makeImage(width: 100, height: 80)
        let encoded = try ImageEncoder.encode(
            image, sourceKind: .png, sourceFileBytes: 50_000,
            sourcePixels: Int64(image.width) * Int64(image.height), sourceWebpLossless: false
        )
        XCTAssertEqual(encoded.ext, "png")
        XCTAssertEqual(encoded.mime, "image/png")

        // PNG 无损：解码逐像素一致
        let source = try ImageEncoder.pixelData(of: image)
        let decoded = CGImageSourceCreateWithData(encoded.data as CFData, nil).flatMap {
            CGImageSourceCreateImageAtIndex($0, 0, nil)
        }
        XCTAssertNotNil(decoded)
        XCTAssertEqual(try ImageEncoder.pixelData(of: decoded!), source)
    }

    func testEncodeHeicKeepsFormat() throws {
        let image = makeImage(width: 256, height: 256)
        let encoded = try ImageEncoder.encode(
            image, sourceKind: .heif, sourceFileBytes: 100_000,
            sourcePixels: Int64(image.width) * Int64(image.height), sourceWebpLossless: false
        )
        XCTAssertEqual(encoded.ext, "heic")
        XCTAssertEqual(encoded.mime, "image/heic")
        let source = CGImageSourceCreateWithData(encoded.data as CFData, nil)
        XCTAssertNotNil(source)
        XCTAssertEqual(CGImageSourceGetCount(source!), 1)
    }

    func testEncodeLosslessWebpPixelFidelity() throws {
        // libwebp 无损编码：输出解码后逐像素一致
        let image = makeImage(width: 96, height: 96)
        let encoded = try ImageEncoder.encode(
            image, sourceKind: .webp, sourceFileBytes: 30_000,
            sourcePixels: Int64(image.width) * Int64(image.height), sourceWebpLossless: true
        )
        XCTAssertEqual(encoded.ext, "webp")
        let decoded = CGImageSourceCreateWithData(encoded.data as CFData, nil).flatMap {
            CGImageSourceCreateImageAtIndex($0, 0, nil)
        }
        XCTAssertNotNil(decoded)
        XCTAssertEqual(try ImageEncoder.pixelData(of: decoded!), try ImageEncoder.pixelData(of: image))
    }

    func testEncodeLossyWebpConstantRate() throws {
        let image = makeImage(width: 512, height: 512)
        let sourceBytes: Int64 = 200_000
        let half = ImageOps.scaleByPercent(image, 50)
        let encoded = try ImageEncoder.encode(
            half, sourceKind: .webp, sourceFileBytes: sourceBytes,
            sourcePixels: Int64(image.width) * Int64(image.height), sourceWebpLossless: false
        )
        XCTAssertEqual(encoded.ext, "webp")
        let target = sourceBytes * Int64(half.width) * Int64(half.height)
            / (Int64(image.width) * Int64(image.height))
        XCTAssertLessThanOrEqual(Int64(encoded.data.count), target)
        XCTAssertGreaterThan(encoded.data.count, 0)
    }
}
