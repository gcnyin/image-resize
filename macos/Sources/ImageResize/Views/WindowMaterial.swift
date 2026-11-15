import AppKit
import SwiftUI

/// 全窗背景材质：对应 Windows 版的 Mica —— 桌面在窗口后方模糊透出，图片直接铺在材质上，
/// 不再有画布自己的纯色底。
///
/// `NSVisualEffectView` 只有在窗口本身非不透明时才会透出窗口背后的内容，所以附带一个
/// 零尺寸探针拿到宿主 `NSWindow`：去掉不透明背景，并让内容延伸到标题栏，标题栏区域与
/// 内容区共用同一块材质（无分隔线、无额外色带）。
struct WindowMaterialBackground: View {
    var body: some View {
        VisualEffectView()
            .background(WindowProbe())
            .ignoresSafeArea()
    }
}

private struct VisualEffectView: NSViewRepresentable {
    func makeNSView(context: Context) -> NSVisualEffectView {
        let view = NSVisualEffectView()
        view.material = .underWindowBackground
        view.blendingMode = .behindWindow
        view.state = .followsWindowActiveState
        return view
    }

    func updateNSView(_ view: NSVisualEffectView, context: Context) {}
}

private struct WindowProbe: NSViewRepresentable {
    func makeNSView(context: Context) -> NSView { ProbeView() }

    func updateNSView(_ view: NSView, context: Context) {}

    /// 只用于在挂上窗口时改窗口配置，本身不参与布局与绘制。
    private final class ProbeView: NSView {
        override func viewDidMoveToWindow() {
            super.viewDidMoveToWindow()
            guard let window else { return }
            window.isOpaque = false
            window.backgroundColor = .clear
            window.titlebarAppearsTransparent = true
            window.titleVisibility = .hidden
            window.titlebarSeparatorStyle = .none
            window.styleMask.insert(.fullSizeContentView)
        }
    }
}
