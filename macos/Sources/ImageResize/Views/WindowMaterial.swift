import AppKit
import SwiftUI

/// 全窗背景材质：对应 Windows 版的 Mica —— 桌面在窗口后方模糊透出，图片直接铺在材质上，
/// 不再有画布自己的纯色底。
///
/// 关键点：SwiftUI 的 `.background` 只覆盖内容区，标题栏由 AppKit 自己绘制，所以材质条
/// 会停在标题栏下方。这里改为在宿主 `NSWindow` 的 `contentView` 最底层直接插入一块
/// `NSVisualEffectView` 并铺满窗口（配合 `fullSizeContentView`），标题栏与内容区因此共用
/// 同一块材质，没有分隔线或色带。
struct WindowMaterialBackground: View {
    var body: some View {
        WindowProbe()
    }
}

private struct WindowProbe: NSViewRepresentable {
    func makeNSView(context: Context) -> NSView { ProbeView() }

    func updateNSView(_ view: NSView, context: Context) {}

    /// 只用于在挂上窗口时改窗口配置并安装材质视图，本身不参与布局与绘制。
    private final class ProbeView: NSView {
        private weak var effectView: NSVisualEffectView?

        override func viewDidMoveToWindow() {
            super.viewDidMoveToWindow()
            guard let window, let container = window.contentView else { return }

            // 窗口必须非不透明，behindWindow 材质才能透出桌面。
            window.isOpaque = false
            window.backgroundColor = .clear
            window.titlebarAppearsTransparent = true
            window.titleVisibility = .hidden
            window.titlebarSeparatorStyle = .none
            window.styleMask.insert(.fullSizeContentView)

            guard effectView == nil else { return }

            let effect = NSVisualEffectView()
            effect.material = .underWindowBackground
            effect.blendingMode = .behindWindow
            effect.state = .followsWindowActiveState
            effect.autoresizingMask = [.width, .height]
            effect.frame = container.bounds
            // 插到最底层，SwiftUI 内容与标题栏都叠在它上面。
            container.addSubview(effect, positioned: .below, relativeTo: nil)
            effectView = effect
        }
    }
}
