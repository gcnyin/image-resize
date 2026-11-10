import AppKit
import UniformTypeIdentifiers

/// 打开 / 保存对话框（默认原格式单选，与 Win 版一致）。
enum Panels {
    static func pickOpenImage() -> String? {
        let panel = NSOpenPanel()
        panel.canChooseFiles = true
        panel.canChooseDirectories = false
        panel.allowsMultipleSelection = false
        panel.allowedContentTypes = [.image]
        panel.message = "选择一张图片"
        guard panel.runModal() == .OK else { return nil }
        return panel.urls.first?.path
    }

    /// 保存对话框：按原格式锁定单一扩展名。
    /// - Returns: 目标路径；用户取消返回 nil。
    static func pickSavePath(suggestedStem: String, ext: String, label: String) -> String? {
        let panel = NSSavePanel()
        panel.nameFieldStringValue = "\(suggestedStem)-edit"
        if let utType = UTType(filenameExtension: ext) {
            panel.allowedContentTypes = [utType]
        }
        panel.message = "保存为 \(label)"
        guard panel.runModal() == .OK, let url = panel.url else { return nil }
        return url.path
    }
}
