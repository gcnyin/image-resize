using Microsoft.UI.Xaml;

namespace ImageResize;

public partial class App : Application
{
    private Window? _window;

    public App()
    {
        InitializeComponent();
        UnhandledException += (_, e) => LogCrash(e.Exception);
    }

    /// <summary>把未处理异常写入 LocalAppData，避免依赖开发机路径；写入失败不掩盖原始异常。</summary>
    private static void LogCrash(Exception exception)
    {
        try
        {
            var dir = Path.Combine(
                Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),
                "ImageResize");
            Directory.CreateDirectory(dir);
            var msg = $"[{DateTime.Now:yyyy-MM-dd HH:mm:ss}] Unhandled: {exception.Message}\n{exception}\n\n";
            File.AppendAllText(Path.Combine(dir, "crash.log"), msg);
        }
        catch
        {
            // 日志写入失败时忽略，保留原始未处理异常
        }
    }

    protected override void OnLaunched(LaunchActivatedEventArgs args)
    {
        _window = new MainWindow();
        _window.Activate();
    }
}
