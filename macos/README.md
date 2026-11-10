# 图片工具 · macOS

Windows 桌面图片小工具 [image-resize-win](../windows) 的 macOS 版（SwiftUI + Image I/O + vendor libwebp），支持**裁剪**与**按百分比缩放**。
返回仓库总览：[../README.md](../README.md)

- **裁剪**：拖动四角 / 整体拖动 / 空白处重新框选（带三分网格、白框深描边、选区外压暗）。
- **缩放**：10%~100% 滑杆 + 25%/50%/75% 快捷键，实时预览目标像素尺寸。
- **编辑循环**：裁剪/缩放可在上一步结果上反复进行，满意后保存。
- **恒定压缩率**：目标字节数 = 源每像素字节 × 输出像素数；JPEG/有损 WebP 在 40~97 内二分搜最大质量，PNG/无损 WebP 保持无损直通（50% 缩放 ≈ 原体积 1/4，绝不因重新编码变大）。
- **加载**：自动按 EXIF 摆正方向；最长边超过 4096px 的工作区自动降采样。
- **格式保持**：按文件头魔数嗅探 JPEG/PNG/WebP/HEIF，保存保持原格式。
- **桌面化**：文件选择/另存为对话框（默认原格式单选）、拖放打开、命令行 `ImageResize 图片路径` 直接打开、深浅色跟随系统。

## 与 Windows 版的差异

| 项 | Windows（SkiaSharp） | macOS（本工程） |
|---|---|---|
| HEIF 源 | 无法编码，回退 JPEG | ImageIO 原生 HEIC 编码，**保留 HEIC** |
| WebP 编码 | Skia 内置 libwebp | ImageIO 只能解码，vendor libwebp 1.5.0 编码（`Vendor/libwebp`，Apache-2.0） |
| 无损 WebP | Skia 无损直通 | libwebp `lossless=1` 无损直通（有单元测试逐像素验证） |
| 大图降采样 | 逐级 1/2 手工下采样 | `CGImageSourceCreateThumbnailAtIndex` 一次完成摆正 + 高质量降采样（上限同为 4096） |
| 裁剪角手柄 | 臂画在边框线上（圆头端帽，HiDPI 下过渡发糊） | 臂同样压在边框线上融合为一条粗线，但用平头端帽，臂端与细线过渡整齐（四边像素级验证） |
| 元数据 | 重新编码不保留 EXIF | 同样不保留（行为一致） |

## 构建 / 运行

依赖：Xcode 15+（Swift 5.9+），macOS 13+。所有命令都在本目录（`macos/`）执行。

```sh
swift build
.build/debug/ImageResize [可选图片路径]
```

- 单工程无 Xcode 项目文件，SPM 直出可执行文件。
- GUI 开发迭代建议直接打 bundle（见下），`swift run` 裸进程窗口行为不完整。

## 测试

```sh
swift test
```

覆盖：魔数嗅探、WebP 无损位探测、目标尺寸（.5 靠偶数，与 .NET `MathF.Round` 一致）、质量二分、缩放/裁剪尺寸、大图上限降采样、JPEG 恒定压缩率、PNG 无损回路、HEIC 保留、无损 WebP 逐像素一致、有损 WebP 恒定压缩率。

## 打包发布

### 本地 .app bundle

```sh
Scripts/make_app.sh          # release（默认）
Scripts/make_app.sh debug    # debug
```

产物：`build/ImageResize.app`（含图标、ad-hoc 签名，双击即用）。

### 分发给别人

当前 bundle 是 **ad-hoc 签名**，拷给同一台机器没问题；分发到别的机器需要：

```sh
codesign --force --options runtime --sign "Developer ID Application: <你的证书>" build/ImageResize.app
xcrun notarytool submit <zip> --apple-id <id> --team-id <tid> --password <pwd>
xcrun stapler staple build/ImageResize.app
```

## 目录结构

```
macos/
├── Package.swift              # SPM 清单（Clibwebp + ImageResize + 测试）
├── Sources/ImageResize/
│   ├── ImageResizeApp.swift   # 入口：窗口尺寸、命令行参数
│   ├── MainWindow.swift       # 顶栏/画布/底部面板、加载与编辑流程编排
│   ├── Views/CropOverlay.swift# 裁剪覆盖层（手势 + Canvas 直绘）
│   ├── Models/LoadedImage.swift
│   └── Services/
│       ├── ImageFormat.swift  # 魔数嗅探 + WebP 无损位探测
│       ├── ImageLoader.swift  # EXIF 摆正 + 4096 上限
│       ├── ImageOps.swift     # 逐级下采样缩放、归一化裁剪
│       ├── ImageEncoder.swift # 恒定压缩率：二分搜质量（ImageIO: JPEG/PNG/HEIC）
│       ├── WebPEncoder.swift  # libwebp 封装（有损/无损）
│       ├── ImageMath.swift    # 纯数学，独立可测
│       └── Panels.swift       # 打开/保存对话框
├── Tests/ImageResizeTests/    # 单元测试
├── Resources/                 # Info.plist、AppIcon.icns
├── Scripts/make_app.sh        # 打包 .app bundle
└── Vendor/libwebp/            # libwebp 1.5.0（enc+dec+dsp+utils+sharpyuv，Apache-2.0）
```
