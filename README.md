# 图片工具（image-resize）

跨平台图片小工具合集：支持**裁剪**与**按百分比缩放**，三端算法与交互一致。
原先是三个独立仓库，现合并到本仓库的三个目录中，各自工程结构、构建方式保持不变。

| 目录 | 平台 | 技术栈 |
| --- | --- | --- |
| [`android/`](android/) | Android 10+ | Kotlin + AppCompat（Gradle） |
| [`windows/`](windows/) | Windows 10/11 | WinUI 3 + .NET 10 + SkiaSharp |
| [`macos/`](macos/) | macOS 13+ | SwiftUI + Image I/O + vendor libwebp |

## 共同功能

- **裁剪**：拖动四角 / 整体拖动 / 空白处重新框选（三分网格、白框深描边、选区外压暗）。
- **缩放**：10%~100% 滑杆 + 25%/50%/75% 快捷键，实时显示目标像素尺寸。
- **编辑循环**：裁剪 / 缩放可反复在上一步结果上操作，满意后保存。
- **恒定压缩率**：目标字节数 = 源图每像素字节数 × 输出像素数；JPEG / 有损 WebP 在 40~97
  内二分搜索最大质量，PNG / 无损 WebP 无损直通。50% 缩放时体积约为原图 1/4，不会因重新编码而变大。
- **加载**：自动按 EXIF 方向摆正；最长边超过 4096px 的工作区自动降采样。
- **格式保持**：按文件头魔数嗅探 JPEG / PNG / WebP / HEIF，保存保持原格式。

## 平台差异

| 项 | Android | Windows | macOS |
| --- | --- | --- | --- |
| 编解码 | Android skia | SkiaSharp | Image I/O + libwebp 1.5.0 |
| HEIF 源 | 回退 JPEG | 回退 JPEG | 原生 HEIC 编码，保留 HEIC |
| 保存位置 | 相册 `Pictures/图片工具/` | 另存为对话框 | 另存为对话框 |
| 桌面能力 | — | 拖放 / 命令行打开 / 深浅色跟随 | 拖放 / 命令行打开 / 深浅色跟随 |
| 深浅色 | DayNight 主题自动切换 | 跟随系统 | 跟随系统 |

## 构建

```sh
# Android
cd android && ./gradlew assembleDebug

# Windows（Windows 上执行）
cd windows && dotnet build ImageResize.csproj -c Debug -p Platform=x64

# macOS
cd macos && swift build
```

各平台的详细说明（测试、打包、签名、发布）见对应目录的 README。

## 目录结构

```
image-resize/
├── android/     # Android 工程（Gradle 根目录，含 app 模块与 test_img 样例）
├── windows/     # Windows 工程（单 csproj，含 samples 样例）
├── macos/       # macOS 工程（Swift Package，含 Vendor/libwebp 与测试）
└── README.md
```
