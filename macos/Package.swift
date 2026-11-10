// swift-tools-version:5.9
import PackageDescription

let package = Package(
    name: "ImageResize",
    platforms: [.macOS(.v13)],
    targets: [
        // Vendor 的 libwebp 编码器（ImageIO 只能解码 WebP，不能编码）
        .target(
            name: "Clibwebp",
            path: "Vendor/libwebp",
            publicHeadersPath: "src/webp",
            cSettings: [.headerSearchPath(".")]
        ),
        .executableTarget(
            name: "ImageResize",
            dependencies: ["Clibwebp"],
            path: "Sources/ImageResize"
        ),
        .testTarget(
            name: "ImageResizeTests",
            dependencies: ["ImageResize"],
            path: "Tests/ImageResizeTests"
        ),
    ]
)
