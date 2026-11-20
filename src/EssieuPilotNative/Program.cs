using System;
using System.IO;
using System.Windows.Forms;

namespace EssieuPilotNative;

internal static class Program
{
    [STAThread]
    private static int Main(string[] args)
    {
        if (args.Length > 0 && args[0] == "--self-test")
            return NativeSelfTests.Run();
        ApplicationConfiguration.Initialize();
        Application.ThreadException += (_, evt) =>
        {
            AppInfo.Log("Erreur interface: " + evt.Exception);
            MessageBox.Show(evt.Exception.Message, "EssieuPilot", MessageBoxButtons.OK,
                MessageBoxIcon.Error);
        };
        AppDomain.CurrentDomain.UnhandledException += (_, evt) =>
            AppInfo.Log("Erreur inattendue: " + evt.ExceptionObject);
        Application.Run(new EssieuPilotWindow());
        return 0;
    }
}

internal static class AppInfo
{
    public const string Name = "EssieuPilot";
    public const string TechnicalVersion = "2.0.0-native.2";
    public const int Build = 2;
    public const string GithubRepo = "ximisch/EssieuPilot";
    public static string SettingsDirectory =>
        Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), "EssieuPilot-Native");
    public static string ProfilesPath => Path.Combine(SettingsDirectory, "profiles.json");
    public static string DiagnosticPath => Path.Combine(SettingsDirectory, "diagnostics.log");
    public static void Log(string message)
    {
        try
        {
            Directory.CreateDirectory(SettingsDirectory);
            File.AppendAllText(DiagnosticPath, $"[{DateTimeOffset.Now:O}] {message}{Environment.NewLine}");
        }
        catch { }
    }
}

internal static class NativeSelfTests
{
    public static int Run()
    {
        try
        {
            var release = """
                [
                  {"tag_name":"v2.0.0-native.2","prerelease":true,"draft":false,
                   "assets":[{"name":"Installer_EssieuPilot.exe",
                    "browser_download_url":"https://github.com/ximisch/EssieuPilot/releases/download/v2.0.0-native.2/Installer_EssieuPilot.exe",
                    "digest":"sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}]},
                  {"tag_name":"v2.0.0-native.1","prerelease":true,"draft":false,"assets":[]}
                ]
                """;
            if (ReleaseCatalog.GetNewest(release, 1)?.Build != 2)
                throw new InvalidOperationException("Updater release selection failed");
            if (ReleaseCatalog.GetNewest(release, 2) != null)
                throw new InvalidOperationException("Updater did not reject installed version");
            var invalid = release.Replace("https://github.com/", "https://example.com/");
            if (ReleaseCatalog.GetNewest(invalid, 1) != null)
                throw new InvalidOperationException("Updater accepted untrusted host");
            if (AtsStatusReader.IsAcknowledgementValid(new byte[32], Environment.TickCount64))
                throw new InvalidOperationException("Invalid steering ACK accepted");
            var sample = new byte[32];
            System.Buffers.Binary.BinaryPrimitives.WriteUInt32LittleEndian(sample.AsSpan(0), 0x45504131);
            System.Buffers.Binary.BinaryPrimitives.WriteUInt32LittleEndian(sample.AsSpan(4), 1);
            System.Buffers.Binary.BinaryPrimitives.WriteUInt32LittleEndian(sample.AsSpan(8), 2);
            System.Buffers.Binary.BinaryPrimitives.WriteUInt32LittleEndian(sample.AsSpan(12), 2);
            System.Buffers.Binary.BinaryPrimitives.WriteUInt64LittleEndian(sample.AsSpan(16), unchecked((ulong)Environment.TickCount64));
            if (!AtsStatusReader.IsAcknowledgementValid(sample, Environment.TickCount64))
                throw new InvalidOperationException("Recent steering ACK rejected");
            if (AtsStatusReader.IsAcknowledgementValid(sample, Environment.TickCount64 + 3000))
                throw new InvalidOperationException("Expired steering ACK accepted");
            Console.WriteLine("PASS: Native updater trust, version selection and physical DLL ACK");
            return 0;
        }
        catch (Exception ex)
        {
            Console.Error.WriteLine(ex);
            return 1;
        }
    }
}
