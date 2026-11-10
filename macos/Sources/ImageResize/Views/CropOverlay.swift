import SwiftUI

/// 裁剪选框覆盖层（移植自 Win 版 CropOverlay）。
/// 在 imageRect（图片显示区，容器坐标）内维护归一化到 [0,1] 的选框，与位图实际尺寸解耦。
/// 手势：拖动四角缩放选框；在选框内拖动整体移动；在选框外按下并拖动重新框选。
/// 覆盖层画在照片上，配色固定不随深浅色主题（白框 + 深色描边 + 三分网格）。
struct CropOverlay: View {
    /// 图片显示区（容器坐标）；nil 或空表示没有图片。
    let imageRect: CGRect?
    /// 归一化选框：origin = left/top。
    @Binding var cropNorm: CGRect

    private enum DragMode { case none, move, tl, tr, bl, br, newSelection }

    @State private var mode: DragMode = .none
    @State private var dragStartNorm: CGRect = CGRect(x: 0, y: 0, width: 1, height: 1)
    @State private var downPoint: CGPoint = .zero
    @State private var moveOffset: CGPoint = .zero

    private static let minSizePx: CGFloat = 48       // 最小选区（逻辑像素）
    private static let cornerLenPx: CGFloat = 22     // 角手柄长度
    private static let cornerHitRadius: CGFloat = 18 // 角命中半径
    // 角手柄臂直接压在边框线上，与边框融合为一条粗线：
    // 白臂 3pt 盖住边框白线 1.5pt，深描边 6pt 盖住边框描边 4pt，重叠区纯白/纯深色无痕。
    // 端帽用平头（.butt）：臂端与细线的过渡是整齐的直角切口，避免圆头渐缩在分数缩放下发糊。
    private static let dimColor = Color(white: 0, opacity: 0.70)
    private static let frameColor = Color.white
    private static let frameShadowColor = Color(white: 0, opacity: 0.35)
    private static let gridColor = Color(white: 1, opacity: 0.35)

    var body: some View {
        Canvas { context, _ in
            guard let r = imageRect, r.width > 0, r.height > 0 else { return }
            let c = pixelRect(in: r)

            // 压暗选框外区域（上、下、左、右四块）
            fill(&context, CGRect(x: r.minX, y: r.minY, width: r.width, height: max(0, c.minY - r.minY)), Self.dimColor)
            fill(&context, CGRect(x: r.minX, y: c.maxY, width: r.width, height: max(0, r.maxY - c.maxY)), Self.dimColor)
            fill(&context, CGRect(x: r.minX, y: c.minY, width: max(0, c.minX - r.minX), height: c.height), Self.dimColor)
            fill(&context, CGRect(x: c.maxX, y: c.minY, width: max(0, r.maxX - c.maxX), height: c.height), Self.dimColor)

            // 三分辅助线：竖×2 + 横×2
            stroke(
                &context,
                path: Path { p in
                    p.move(to: CGPoint(x: c.minX + c.width / 3, y: c.minY))
                    p.addLine(to: CGPoint(x: c.minX + c.width / 3, y: c.maxY))
                    p.move(to: CGPoint(x: c.minX + c.width * 2 / 3, y: c.minY))
                    p.addLine(to: CGPoint(x: c.minX + c.width * 2 / 3, y: c.maxY))
                    p.move(to: CGPoint(x: c.minX, y: c.minY + c.height / 3))
                    p.addLine(to: CGPoint(x: c.maxX, y: c.minY + c.height / 3))
                    p.move(to: CGPoint(x: c.minX, y: c.minY + c.height * 2 / 3))
                    p.addLine(to: CGPoint(x: c.maxX, y: c.minY + c.height * 2 / 3))
                },
                color: Self.gridColor, width: 0.75
            )

            // 边框与四角手柄（先深色描边，再叠白线）
            context.stroke(Path(CGPath(rect: c, transform: nil)),
                           with: .color(Self.frameShadowColor), lineWidth: 4)
            context.stroke(Path(CGPath(rect: c, transform: nil)),
                           with: .color(Self.frameColor), lineWidth: 1.5)

            let corners: [(Bool, Bool)] = [(true, true), (false, true), (true, false), (false, false)]
            for (left, top) in corners {
                let cornerPath = cornerPath(rect: c, left: left, top: top)
                // 角点必须在中间：横臂 → 角 → 竖臂，否则会连出斜边
                stroke(&context, path: cornerPath, color: Self.frameShadowColor, width: 6, cap: .butt)
                stroke(&context, path: cornerPath, color: Self.frameColor, width: 3, cap: .butt)
            }
        }
        .contentShape(Rectangle())
        .allowsHitTesting(imageRect != nil && (imageRect?.width ?? 0) > 0)
        .gesture(
            DragGesture(minimumDistance: 0)
                .onChanged { value in
                    if mode == .none { beginDrag(at: value.startLocation) }
                    dragTo(at: value.location)
                }
                .onEnded { _ in endDrag() }
        )
    }

    // MARK: - 手势

    private func beginDrag(at p: CGPoint) {
        guard let r = imageRect, r.width > 0, r.height > 0 else { return }
        downPoint = p
        dragStartNorm = cropNorm
        let c = pixelRect(in: r)

        let corner = hitCorner(p, c)
        if corner != .none {
            mode = corner
            return
        }
        if c.contains(p) {
            mode = .move
            moveOffset = CGPoint(x: p.x - c.minX, y: p.y - c.minY)
            return
        }
        mode = .newSelection
        let x = normX(p.x, r), y = normY(p.y, r)
        cropNorm = CGRect(x: x, y: y, width: 0, height: 0)
    }

    private func dragTo(at p: CGPoint) {
        guard let r = imageRect, r.width > 0, r.height > 0 else { return }
        let sMinW = min(max(Self.minSizePx / r.width, 0), 1)
        let sMinH = min(max(Self.minSizePx / r.height, 0), 1)
        let s = dragStartNorm

        switch mode {
        case .move:
            let dx = (p.x - downPoint.x) / r.width
            let dy = (p.y - downPoint.y) / r.height
            let left = min(max(s.minX + dx, 0), 1 - s.width)
            let top = min(max(s.minY + dy, 0), 1 - s.height)
            cropNorm = CGRect(x: left, y: top, width: s.width, height: s.height)
        case .tl:
            let l = min(max(normX(p.x, r), 0), s.maxX - sMinW)
            let t = min(max(normY(p.y, r), 0), s.maxY - sMinH)
            cropNorm = fromEdges(l, t, s.maxX, s.maxY)
        case .tr:
            let right = min(max(normX(p.x, r), s.minX + sMinW), 1)
            let t = min(max(normY(p.y, r), 0), s.maxY - sMinH)
            cropNorm = fromEdges(s.minX, t, right, s.maxY)
        case .bl:
            let l = min(max(normX(p.x, r), 0), s.maxX - sMinW)
            let b = min(max(normY(p.y, r), s.minY + sMinH), 1)
            cropNorm = fromEdges(l, s.minY, s.maxX, b)
        case .br:
            let right = min(max(normX(p.x, r), s.minX + sMinW), 1)
            let b = min(max(normY(p.y, r), s.minY + sMinH), 1)
            cropNorm = fromEdges(s.minX, s.minY, right, b)
        case .newSelection:
            let x0 = normX(downPoint.x, r), y0 = normY(downPoint.y, r)
            let x1 = normX(p.x, r), y1 = normY(p.y, r)
            cropNorm = fromEdges(min(x0, x1), min(y0, y1), max(x0, x1), max(y0, y1))
        case .none:
            break
        }
    }

    private func endDrag() {
        if mode == .newSelection, let r = imageRect, r.width > 0, r.height > 0 {
            let minW = min(max(Self.minSizePx / max(r.width, 1), 0), 1)
            let minH = min(max(Self.minSizePx / max(r.height, 1), 0), 1)
            // 点击空白或拖动距离过小：恢复原选框
            if cropNorm.width < minW || cropNorm.height < minH {
                cropNorm = dragStartNorm
            }
        }
        mode = .none
    }

    private func hitCorner(_ p: CGPoint, _ c: CGRect) -> DragMode {
        func near(_ px: CGFloat, _ py: CGFloat) -> Bool {
            (p.x - px) * (p.x - px) + (p.y - py) * (p.y - py) <= Self.cornerHitRadius * Self.cornerHitRadius
        }
        if near(c.minX, c.minY) { return .tl }
        if near(c.maxX, c.minY) { return .tr }
        if near(c.minX, c.maxY) { return .bl }
        if near(c.maxX, c.maxY) { return .br }
        return .none
    }

    // MARK: - 坐标换算与绘制辅助

    private func pixelRect(in r: CGRect) -> CGRect {
        CGRect(
            x: r.minX + cropNorm.minX * r.width,
            y: r.minY + cropNorm.minY * r.height,
            width: cropNorm.width * r.width,
            height: cropNorm.height * r.height
        )
    }

    private func normX(_ x: CGFloat, _ r: CGRect) -> CGFloat {
        min(max((x - r.minX) / r.width, 0), 1)
    }

    private func normY(_ y: CGFloat, _ r: CGRect) -> CGFloat {
        min(max((y - r.minY) / r.height, 0), 1)
    }

    private func fromEdges(_ left: CGFloat, _ top: CGFloat, _ right: CGFloat, _ bottom: CGFloat) -> CGRect {
        CGRect(x: left, y: top, width: right - left, height: bottom - top)
    }

    private func cornerPath(rect c: CGRect, left: Bool, top: Bool) -> Path {
        let x = left ? c.minX : c.maxX
        let y = top ? c.minY : c.maxY
        let dx = left ? Self.cornerLenPx : -Self.cornerLenPx
        let dy = top ? Self.cornerLenPx : -Self.cornerLenPx
        // 臂压在边框线上，角点在中间：横臂 → 角 → 竖臂，否则会连出斜边。
        return Path { p in
            p.move(to: CGPoint(x: x + dx, y: y))
            p.addLine(to: CGPoint(x: x, y: y))
            p.addLine(to: CGPoint(x: x, y: y + dy))
        }
    }

    private func fill(_ context: inout GraphicsContext, _ rect: CGRect, _ color: Color) {
        guard rect.width > 0, rect.height > 0 else { return }
        context.fill(Path(rect), with: .color(color))
    }

    private func stroke(
        _ context: inout GraphicsContext, path: Path, color: Color, width: CGFloat,
        cap: CGLineCap = .round
    ) {
        context.stroke(path, with: .color(color), style: StrokeStyle(lineWidth: width, lineCap: cap))
    }
}
