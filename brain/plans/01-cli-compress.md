# CLI 图片压缩（Go）方案

Back to [[plans/index]]

## Context

image-resize 已有 Android / Windows / macOS 三端，共同算法是"恒定压缩率"编码：
目标字节数 = max(4096, 源文件字节数 × 输出像素 / 源像素)。
现新增第四个模块 `cli/`：命令行版，用 Go 实现，**只做压缩**（无裁剪、无缩放）。

由于不做缩放，输出像素 == 源像素，目标字节数退化为 max(4096, 源文件字节数)，
即"重编码后不比原文件大，且信息密度不降"。压缩输出尺寸小于原文件即为成功。

## Scope

**包含**：JPEG / PNG 压缩、批量文件参数、输出目录 flag、单元测试、README。

**排除**：裁剪、缩放、HEIF、动图。WebP 见下方取舍。

## 方案取舍（2 个候选）

1. **纯标准库，仅 JPEG + PNG**（选定）：Go 标准库自带两种编解码；
   WebP/HEIF 无纯 Go 编码器，遇到时报"暂不支持"并跳过（exit code 非 0 但继续处理其余文件）。
   零依赖、单二进制、跨平台，符合"各端技术栈独立"的仓库结构。
2. 引 `golang.org/x/image/webp` 解码 + libwebp cgo 编码：能保格式，但破坏零依赖，
   Windows 交叉编译复杂。CLI 版定位轻量，不值得。

## 恒定压缩率（对齐三端）

- JPEG：质量区间 [40, 97]，二分搜"不超过目标字节数的最大质量"
  （对齐 `ImageMath.solveQuality`，macOS/Windows 同名实现）。
- PNG：无损直通重编码（`png.Encode`），无质量参数。
- 目标字节 = max(4096, 源文件字节)。若整个质量区间都超过目标字节，仍以 minQuality=40 的结果写出（允许略超目标），与 `ImageEncoder.swift` / `ImageMath.solveQuality` 行为一致。
- 输出 ≥ 源文件时照写并在结果行标注 `未变小`（如已高度优化的 PNG、< 4096 字节的小图），不报错。
- 不处理 EXIF 方向：Go `image/jpeg` 解码丢弃 EXIF，orientation ≠ 1 的照片重编码后方向以像素为准（README 声明）。

## 结构

```
cli/
├── go.mod            # module image-resize/cli，go 1.27
├── main.go           # flag 解析、遍历文件、汇总输出
├── internal/
│   ├── format/format.go      # 魔数嗅探（对齐 ImageFormat.swift：JPEG/PNG/WebP/HEIF 头）
│   ├── imgmath/imgmath.go    # SolveQuality 二分（对齐 ImageMath.swift）
│   └── compress/compress.go  # 恒定压缩率编码（对齐 ImageEncoder.swift）
│   └── compress/compress.go  # 恒定压缩率编码（对齐 ImageEncoder.swift）
├── internal/format/format_test.go、internal/imgmath/imgmath_test.go、internal/compress/compress_test.go
└── README.md
```

## CLI 契约

```
imgcompr [-o 输出目录(默认原目录)] 文件1 [文件2 ...]
```

- 输出文件名 = 原名（写到别的目录），或同目录时加 `_min` 后缀，不覆盖源文件。
- 每个文件一行结果：`输入 → 输出 (1.2 MB → 340 KB, -72%)`。
- 退出码：全部成功 → 0；存在跳过（WebP/HEIF/非 JPEG/PNG 魔数或扩展名）或失败（0 字节、损坏、无法解码）→ 1；无参数 → 2。
- 0 字节 / 解码失败：单独报错行，继续处理其余文件。
- 输出已存在时直接覆盖（重跑幂等）；源文件永不覆盖。

## Phases

1. `internal/format`：魔数嗅探 + 表驱动测试（JPEG/PNG/WebP 头、10 个 HEIF 品牌、空数据、扩展名兜底对齐 ImageFormat.swift 的 detect）。
2. `internal/imgmath`：`SolveQuality` 二分 + 单调性测试。
3. `internal/compress`：JPEG 目标字节二分、PNG 直通 + 集成测试（生成 JPEG/PNG→压缩→断言格式保持；< 4096 字节小图边界；损坏 JPEG 解码失败路径）。
4. `main.go`：flag、批量处理、汇总、退出码。
5. README（cli/ 与根 README 表格加一行）、.gitignore 加 `cli/imgcompr`。

## Verification

- 静态：`cd cli && go vet ./... && go test ./...`
- 运行：`go build` 后对仓库内真实图片（无则用测试生成）跑
  `./imgcompr test.jpg`，确认输出更小、可再次打开。
- 退出码：无参数 → usage + exit 2；全部跳过 → exit 1；成功 → 0。
