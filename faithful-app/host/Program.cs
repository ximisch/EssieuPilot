using System;
using System.Diagnostics;
using System.Drawing;
using System.IO;
using System.Runtime.InteropServices;
using System.Threading.Tasks;
using System.Windows.Forms;
using Microsoft.Web.WebView2.Core;
using Microsoft.Web.WebView2.WinForms;

namespace EssieuPilot;

internal static class Program
{
    [DllImport("shell32.dll", CharSet = CharSet.Unicode)]
    private static extern int SetCurrentProcessExplicitAppUserModelID(string appID);

    internal static readonly string LogFile = Path.Combine(
        Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),
        "EssieuPilot", "diagnostics", "original-ui-startup.log");

    internal static void Log(string message)
    {
        try
        {
            Directory.CreateDirectory(Path.GetDirectoryName(LogFile)!);
            File.AppendAllText(LogFile, $"[{DateTimeOffset.Now:O}] {message}{Environment.NewLine}");
        }
        catch { }
    }

    [STAThread]
    private static int Main(string[] args)
    {
        _ = SetCurrentProcessExplicitAppUserModelID("EssieuPilot.OriginalTest");
        ApplicationConfiguration.Initialize();
        Application.ThreadException += (_, e) => Log("UI exception: " + e.Exception);
        AppDomain.CurrentDomain.UnhandledException += (_, e) => Log("Fatal: " + e.ExceptionObject);
        if (args.Length > 0 && args[0] == "--ui-smoke")
        {
            try
            {
                using var form = new MainWindow(smoke: true);
                form.Show();
                Application.DoEvents();
                if (!form.Visible || !form.IsHandleCreated || form.Controls.Count == 0)
                    throw new InvalidOperationException("Original interface host window not created");
                form.Close();
                Log("UI smoke test PASS");
                return 0;
            }
            catch (Exception e)
            {
                Log("UI smoke test FAILED: " + e);
                return 1;
            }
        }
        Log("Starting faithful original EssieuPilot UI host");
        Application.Run(new MainWindow());
        return 0;
    }
}

internal sealed class MainWindow : Form
{
    private readonly WebView2 _webView;
    private readonly bool _smoke;
    private Process? _backend;
    private Uri? _localUri;

    public MainWindow(bool smoke = false)
    {
        _smoke = smoke;
        Text = "EssieuPilot";
        Width = 1320;
        Height = 850;
        MinimumSize = new Size(960, 640);
        StartPosition = FormStartPosition.CenterScreen;
        BackColor = Color.FromArgb(8, 15, 28);
        try { Icon = Icon.ExtractAssociatedIcon(Application.ExecutablePath) ?? SystemIcons.Application; }
        catch { Icon = SystemIcons.Application; }
        ShowIcon = true;
        _webView = new WebView2 { Dock = DockStyle.Fill, DefaultBackgroundColor = Color.FromArgb(8, 15, 28) };
        Controls.Add(_webView);
        if (!_smoke) Shown += async (_, _) => await InitializeAsync();
    }

    private async Task InitializeAsync()
    {
        try
        {
            string backendPath = Path.Combine(AppContext.BaseDirectory, "EssieuPilotBackend.exe");
            if (!File.Exists(backendPath))
                throw new FileNotFoundException("Composant EssieuPilotBackend.exe introuvable.", backendPath);

            // Detect missing Evergreen runtime explicitly instead of silently failing.
            try
            {
                var runtime = CoreWebView2Environment.GetAvailableBrowserVersionString();
                Program.Log("WebView2 runtime: " + runtime);
            }
            catch (Exception ex)
            {
                throw new InvalidOperationException(
                    "Le composant Windows Microsoft Edge WebView2 Runtime est indisponible. " +
                    "Installe le WebView2 Runtime officiel Microsoft avant de relancer EssieuPilot.", ex);
            }

            var info = new ProcessStartInfo(backendPath) {
                WorkingDirectory = AppContext.BaseDirectory,
                UseShellExecute = false,
                CreateNoWindow = true,
                WindowStyle = ProcessWindowStyle.Hidden,
                RedirectStandardOutput = true,
                RedirectStandardError = true
            };
            _backend = Process.Start(info) ?? throw new InvalidOperationException("Démarrage du moteur impossible.");
            _backend.EnableRaisingEvents = true;
            _backend.ErrorDataReceived += (_, e) => {
                if (e.Data != null) Program.Log("Backend: " + e.Data);
            };
            _backend.BeginErrorReadLine();
            _backend.Exited += (_, _) => {
                Program.Log("Backend exited");
                if (!IsDisposed && IsHandleCreated)
                    try { BeginInvoke(new Action(() => Close())); } catch { }
            };
            string? ready = await _backend.StandardOutput.ReadLineAsync().WaitAsync(TimeSpan.FromSeconds(30));
            if (ready == null || !ready.StartsWith("READY http://127.0.0.1:", StringComparison.Ordinal))
                throw new InvalidOperationException("Le moteur n'a pas fourni son adresse de démarrage.");
            var address = new Uri(ready[6..]);
            if (address.Host != "127.0.0.1" || address.Scheme != "http" || address.Port == 0)
                throw new InvalidOperationException("Adresse locale du moteur invalide.");
            _localUri = address;
            Program.Log("Backend ready at localhost port " + address.Port);
            await _webView.EnsureCoreWebView2Async();
            _webView.CoreWebView2.Settings.AreDevToolsEnabled = false;
            _webView.CoreWebView2.Settings.AreDefaultContextMenusEnabled = false;
            _webView.CoreWebView2.NavigationStarting += (_, e) => {
                if (Uri.TryCreate(e.Uri, UriKind.Absolute, out var target) &&
                    target.GetLeftPart(UriPartial.Authority) != _localUri.GetLeftPart(UriPartial.Authority))
                {
                    e.Cancel = true;
                    if (target.Scheme == Uri.UriSchemeHttps)
                        Process.Start(new ProcessStartInfo(target.AbsoluteUri) { UseShellExecute = true });
                }
            };
            _webView.CoreWebView2.NavigationCompleted += (_, e) =>
                Program.Log(e.IsSuccess ? "Original UI loaded" :
                    "Original UI navigation failed: " + e.WebErrorStatus);
            _webView.CoreWebView2.WindowCloseRequested += (_, _) => Close();
            _webView.Source = _localUri;
        }
        catch (Exception exception)
        {
            Program.Log("Startup failed: " + exception);
            MessageBox.Show(this,
                "Impossible d'ouvrir EssieuPilot.\n\n" + exception.Message +
                "\n\nJournal : %LOCALAPPDATA%\\EssieuPilot\\diagnostics\\original-ui-startup.log" +
                "\nL'installation officielle 1.6.2 n'est pas modifiée.",
                "EssieuPilot — diagnostic de démarrage",
                MessageBoxButtons.OK, MessageBoxIcon.Error);
            Close();
        }
    }

    protected override void OnFormClosed(FormClosedEventArgs e)
    {
        try { if (_backend is { HasExited: false }) _backend.Kill(entireProcessTree: true); }
        catch { }
        _backend?.Dispose();
        _webView.Dispose();
        base.OnFormClosed(e);
    }
}
