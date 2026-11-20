using System;
using System.Diagnostics;
using System.Drawing;
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

    [STAThread]
    private static void Main()
    {
        // An application-owned native window fixes Edge's generic taskbar identity.
        // Development AppUserModelID is isolated from the installed production app.
        _ = SetCurrentProcessExplicitAppUserModelID("EssieuPilot.Beta");
        ApplicationConfiguration.Initialize();
        Application.Run(new MainWindow());
    }
}

internal sealed class MainWindow : Form
{
    private readonly WebView2 _webView;
    private Process? _backend;
    private Uri? _localUri;

    public MainWindow()
    {
        Text = "EssieuPilot";
        Width = 1320;
        Height = 850;
        MinimumSize = new Size(960, 640);
        StartPosition = FormStartPosition.CenterScreen;
        BackColor = Color.FromArgb(8, 15, 28);
        try
        {
            Icon = Icon.ExtractAssociatedIcon(Application.ExecutablePath) ?? SystemIcons.Application;
        }
        catch { Icon = SystemIcons.Application; }
        ShowIcon = true;
        _webView = new WebView2 { Dock = DockStyle.Fill, DefaultBackgroundColor = Color.FromArgb(8, 15, 28) };
        Controls.Add(_webView);
        Shown += async (_, _) => await InitializeAsync();
    }

    private async Task InitializeAsync()
    {
        try
        {
            string backendPath = System.IO.Path.Combine(AppContext.BaseDirectory, "EssieuPilotBackend.exe");
            if (!System.IO.File.Exists(backendPath))
                throw new System.IO.FileNotFoundException("Le composant EssieuPilotBackend.exe est introuvable.", backendPath);
            var info = new ProcessStartInfo(backendPath)
            {
                WorkingDirectory = AppContext.BaseDirectory,
                UseShellExecute = false,
                CreateNoWindow = true,
                WindowStyle = ProcessWindowStyle.Hidden,
                RedirectStandardOutput = true,
                RedirectStandardError = true,
            };
            _backend = Process.Start(info) ?? throw new InvalidOperationException("Démarrage du moteur impossible.");
            _backend.EnableRaisingEvents = true;
            _backend.Exited += (_, _) =>
            {
                if (!IsDisposed && IsHandleCreated)
                    BeginInvoke(new Action(() => Close()));
            };
            string? ready = await _backend.StandardOutput.ReadLineAsync().WaitAsync(TimeSpan.FromSeconds(15));
            if (ready is null || !ready.StartsWith("READY http://127.0.0.1:", StringComparison.Ordinal))
                throw new InvalidOperationException("Le moteur local n'a pas renvoyé son adresse de démarrage.");
            var address = new Uri(ready[6..]);
            if (address.Host != "127.0.0.1" || address.Scheme != "http" || address.Port == 0)
                throw new InvalidOperationException("Adresse locale du moteur invalide.");
            _localUri = address;
            await _webView.EnsureCoreWebView2Async();
            _webView.CoreWebView2.Settings.AreDevToolsEnabled = false;
            _webView.CoreWebView2.Settings.AreDefaultContextMenusEnabled = false;
            _webView.CoreWebView2.NavigationStarting += (_, e) =>
            {
                if (Uri.TryCreate(e.Uri, UriKind.Absolute, out var requested) &&
                    !string.Equals(requested.GetLeftPart(UriPartial.Authority), _localUri.GetLeftPart(UriPartial.Authority), StringComparison.OrdinalIgnoreCase))
                {
                    e.Cancel = true;
                    if (requested.Scheme == "https")
                        Process.Start(new ProcessStartInfo(requested.AbsoluteUri) { UseShellExecute = true });
                }
            };
            _webView.CoreWebView2.WindowCloseRequested += (_, _) => Close();
            _webView.Source = _localUri;
        }
        catch (Exception exception)
        {
            MessageBox.Show(this,
                "Impossible de démarrer la reconstruction d'EssieuPilot.\n\n" + exception.Message +
                "\n\nLa version 1.6.2 installée n'a pas été modifiée.",
                "EssieuPilot — lancement", MessageBoxButtons.OK, MessageBoxIcon.Error);
            Close();
        }
    }

    protected override void OnFormClosed(FormClosedEventArgs e)
    {
        try { if (_backend is { HasExited: false }) _backend.Kill(entireProcessTree: true); }
        catch { /* Backend may have already exited. */ }
        _backend?.Dispose();
        _webView.Dispose();
        base.OnFormClosed(e);
    }
}
