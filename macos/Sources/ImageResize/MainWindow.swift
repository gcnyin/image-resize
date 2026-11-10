import SwiftUI

/// 界面与编辑流程编排（对应 Win 版 MainWindow）。
struct MainWindowView: View {
    /// 命令行直接打开的图片路径（如 `ImageResize 图片路径`）。
    let initialPath: String?

    /// 当前编辑中的工作区图像（加载 → 剪裁/缩放循环替换）。
    @State private var current: CGImage?
    /// 本次加载的源图片信息。
    @State private var source: LoadedImage?
    @State private var sourcePath: String?
    @State private var busy = false
    @State private var cropMode = true
    @State private var percent: Double = 50
    @State private var cropNorm = CGRect(x: 0, y: 0, width: 1, height: 1)
    @State private var feedback: Feedback?

    private let fullRect = CGRect(x: 0, y: 0, width: 1, height: 1)

    enum Severity { case info, success, error }

    struct Feedback: Equatable {
        let severity: Severity
        let message: String
    }

    var body: some View {
        VStack(spacing: 0) {
            header
            canvas
            bottomPanel
        }
        .onAppear {
            if let path = initialPath, current == nil {
                loadAsync(path)
            }
        }
    }

    // ---------------------------------------------------------------- 顶栏

    private var header: some View {
        HStack(spacing: 8) {
            Text("图片工具").font(.system(size: 18, weight: .semibold))
            Spacer()
            Button("选择图片") { openPanel() }
                .disabled(busy)
            Button("保存") { saveAsync() }
                .buttonStyle(.borderedProminent)
                .disabled(current == nil || busy)
        }
        .padding(EdgeInsets(top: 10, leading: 16, bottom: 6, trailing: 16))
    }

    // ---------------------------------------------------------------- 画布

    private var canvas: some View {
        GeometryReader { geo in
            let fit = fittedRect(canvas: geo.size)
            ZStack {
                Rectangle().fill(Color(nsColor: .canvasColor))
                if let cg = current {
                    Image(decorative: cg, scale: 1)
                        .resizable()
                        .frame(width: fit.width, height: fit.height)
                        .id(cg)
                } else {
                    Text("请选择一张图片开始").foregroundStyle(.secondary)
                }
                if current != nil && cropMode && !busy {
                    CropOverlay(imageRect: fit, cropNorm: $cropNorm)
                }
                if busy {
                    ProgressView().controlSize(.large)
                }
            }
            .dropDestination(for: URL.self) { urls, _ in
                guard let url = urls.first, url.isFileURL else { return false }
                loadAsync(url.path)
                return true
            }
        }
        .padding(EdgeInsets(top: 8, leading: 16, bottom: 8, trailing: 16))
    }

    // ---------------------------------------------------------------- 底部面板

    private var bottomPanel: some View {
        VStack(alignment: .leading, spacing: 8) {
            feedbackBanner
            Text(infoText)
                .font(.system(size: 12))
                .foregroundStyle(.secondary)
                .lineLimit(1)
                .truncationMode(.tail)
            HStack(spacing: 12) {
                Picker("", selection: $cropMode) {
                    Text("剪裁").tag(true)
                    Text("缩放").tag(false)
                }
                .pickerStyle(.segmented)
                .fixedSize()
                .disabled(busy)
                .labelsHidden()

                Divider().frame(height: 26)

                if cropMode {
                    cropControls
                } else {
                    resizeControls
                }
                Spacer()
            }
        }
        .padding(EdgeInsets(top: 10, leading: 16, bottom: 12, trailing: 16))
        .background(Color(nsColor: .separatorColor).opacity(0.25))
    }

    private var cropControls: some View {
        HStack(spacing: 8) {
            Button("全图") { cropNorm = fullRect }
                .disabled(current == nil || busy)
            Button("应用剪裁") { applyCrop() }
                .buttonStyle(.borderedProminent)
                .disabled(current == nil || busy)
        }
    }

    private var resizeControls: some View {
        HStack(spacing: 8) {
            Slider(value: $percent, in: 10...100)
                .frame(width: 220)
                .disabled(current == nil || busy)
            Text("\(Int(percent))%").font(.system(size: 16)).frame(minWidth: 46)
            ForEach([25, 50, 75], id: \.self) { p in
                Button("\(p)%") { percent = Double(p) }
                    .disabled(current == nil || busy)
            }
            Button("应用缩放") { applyResize() }
                .buttonStyle(.borderedProminent)
                .disabled(current == nil || busy)
        }
    }

    @ViewBuilder
    private var feedbackBanner: some View {
        if let fb = feedback {
            HStack(spacing: 6) {
                Circle().fill(fb.severity.color).frame(width: 8, height: 8)
                Text(fb.message).font(.system(size: 12))
                Spacer()
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 5)
            .background(fb.severity.color.opacity(0.12), in: RoundedRectangle(cornerRadius: 5))
            .transition(.opacity)
        }
    }

    // ---------------------------------------------------------------- 信息行

    private var infoText: String {
        guard let bmp = current, let img = source else { return "请选择一张图片开始" }
        let label = Self.formatLabel(img)
        if cropMode {
            let cw = max(1, Int((cropNorm.width * CGFloat(bmp.width)).rounded()))
            let ch = max(1, Int((cropNorm.height * CGFloat(bmp.height)).rounded()))
            return "当前 \(bmp.width)×\(bmp.height) · 剪裁区域 \(cw)×\(ch) · 原图 \(img.sourceWidth)×\(img.sourceHeight) \(label)"
        }
        let p = Int(percent)
        let tw = ImageMath.targetSize(bmp.width, p)
        let th = ImageMath.targetSize(bmp.height, p)
        return "比例 \(p)%：\(bmp.width)×\(bmp.height) → \(tw)×\(th) · 原图 \(img.sourceWidth)×\(img.sourceHeight) \(label)"
    }

    private static func formatLabel(_ img: LoadedImage) -> String {
        switch img.format {
        case .jpeg: return "JPEG"
        case .png: return "PNG"
        case .webp: return img.webpLossless ? "WebP(无损)" : "WebP"
        case .heif: return "HEIC"  // macOS 可保留原格式（Win 版回退 JPEG）
        case .unknown: return "JPEG"
        }
    }

    // ---------------------------------------------------------------- 加载

    private func openPanel() {
        guard let path = Panels.pickOpenImage() else { return }
        loadAsync(path)
    }

    private func loadAsync(_ path: String) {
        guard !busy else { return }
        busy = true
        feedback = nil
        Task {
            do {
                let img = try await Task.detached(priority: .userInitiated) {
                    try ImageLoader.load(path: path)
                }.value
                current = img.cgImage
                source = img
                sourcePath = path
                cropNorm = fullRect
            } catch {
                showFeedback("加载失败：\(error.localizedDescription)", .error, autoClose: false)
            }
            busy = false
        }
    }

    // ---------------------------------------------------------------- 剪裁 / 缩放

    private func applyCrop() {
        guard current != nil else { return }
        if cropNorm.width >= 0.999 && cropNorm.height >= 0.999 {
            showFeedback("选框已覆盖全图，无需剪裁", .info)
            return
        }
        let rect = cropNorm
        runImageOp("剪裁") { ImageOps.crop($0, rect) }
    }

    private func applyResize() {
        guard current != nil else { return }
        let p = Int(percent)
        if p >= 100 {
            showFeedback("比例已是 100%，无需缩放", .info)
            return
        }
        runImageOp("缩放") { ImageOps.scaleByPercent($0, p) }
    }

    private func runImageOp(_ name: String, _ op: @escaping (CGImage) -> CGImage) {
        guard let bmp = current, !busy else { return }
        busy = true
        Task {
            let next = await Task.detached(priority: .userInitiated) { op(bmp) }.value
            current = next
            cropNorm = fullRect
            showFeedback("\(name)完成：\(next.width)×\(next.height)", .success)
            busy = false
        }
    }

    // ---------------------------------------------------------------- 保存

    private func saveAsync() {
        guard let bmp = current, let img = source, !busy else { return }

        // 保存保持原格式：HEIC 源直接保留（Win 版回退 JPEG）
        let (choiceLabel, ext): (String, String)
        switch img.format {
        case .png: choiceLabel = "PNG 图片"; ext = "png"
        case .webp: choiceLabel = "WebP 图片"; ext = "webp"
        case .heif: choiceLabel = "HEIC 图片"; ext = "heic"
        case .jpeg, .unknown: choiceLabel = "JPEG 图片"; ext = "jpg"
        }

        let stem = sourcePath.map { ($0 as NSString).deletingPathExtension } ?? "image"
        guard let path = Panels.pickSavePath(suggestedStem: stem, ext: ext, label: choiceLabel) else { return }

        busy = true
        Task {
            do {
                // 恒定压缩率编码：与源图保持相同的每像素字节
                let encoded = try await Task.detached(priority: .userInitiated) {
                    try ImageEncoder.encode(
                        bmp,
                        sourceKind: img.format,
                        sourceFileBytes: img.sourceFileBytes,
                        sourcePixels: Int64(img.sourceWidth) * Int64(img.sourceHeight),
                        sourceWebpLossless: img.webpLossless
                    )
                }.value
                try encoded.data.write(to: URL(fileURLWithPath: path))
                let kb = (encoded.data.count + 1023) / 1024
                showFeedback("已保存：\(path)（\(kb) KB）", .success)
            } catch {
                showFeedback("保存失败：\(error.localizedDescription)", .error, autoClose: false)
            }
            busy = false
        }
    }

    // ---------------------------------------------------------------- 反馈与布局

    private func showFeedback(_ message: String, _ severity: Severity, autoClose: Bool = true) {
        let fb = Feedback(severity: severity, message: message)
        feedback = fb
        guard autoClose else { return }
        DispatchQueue.main.asyncAfter(deadline: .now() + 4) {
            if feedback == fb { feedback = nil }
        }
    }

    /// 画布四周留边：边框白线 + 描边会向选框外扩约 3pt，若图像铺满画布，
    /// 外扩部分会恰好落在覆盖层边界上被裁切（表现为左右或上下边框"缺半条"）。
    private static let canvasInset: CGFloat = 12

    /// 计算位图在画布中的实际显示区域（fitCenter + 四周留边），同步给剪裁覆盖层。
    private func fittedRect(canvas size: CGSize) -> CGRect {
        guard let bmp = current, size.width > 0, size.height > 0 else { return .zero }
        let availW = max(1, size.width - 2 * Self.canvasInset)
        let availH = max(1, size.height - 2 * Self.canvasInset)
        let scale = min(availW / CGFloat(bmp.width), availH / CGFloat(bmp.height))
        let w = CGFloat(bmp.width) * scale
        let h = CGFloat(bmp.height) * scale
        return CGRect(x: (size.width - w) / 2, y: (size.height - h) / 2, width: w, height: h)
    }
}

extension MainWindowView.Severity {
    var color: Color {
        switch self {
        case .info: return .blue
        case .success: return .green
        case .error: return .red
        }
    }
}

extension NSColor {
    /// 画布底色：浅色 #F2F3F7，深色 #141414（保持参考应用的观感）。
    static let canvasColor = NSColor(name: nil) { appearance in
        let dark = appearance.bestMatch(from: [.aqua, .darkAqua]) == .darkAqua
        if dark {
            return NSColor(srgbRed: 0x14 / 255, green: 0x14 / 255, blue: 0x14 / 255, alpha: 1)
        }
        return NSColor(srgbRed: 0xF2 / 255, green: 0xF3 / 255, blue: 0xF7 / 255, alpha: 1)
    }
}
