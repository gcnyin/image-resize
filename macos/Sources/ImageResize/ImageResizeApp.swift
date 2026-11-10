import SwiftUI

@main
struct ImageResizeApp: App {
    /// 支持命令行直接打开图片：`ImageResize 图片路径`
    private let initialPath: String?

    init() {
        let args = ProcessInfo.processInfo.arguments
        initialPath = args.count > 1 && FileManager.default.fileExists(atPath: args[1]) ? args[1] : nil
    }

    var body: some Scene {
        WindowGroup("图片工具") {
            MainWindowView(initialPath: initialPath)
                .frame(minWidth: 860, minHeight: 620)
        }
        .defaultSize(width: 1100, height: 760)
        .windowResizability(.contentMinSize)
    }
}
