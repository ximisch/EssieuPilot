using System;
using System.Buffers.Binary;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.IO.MemoryMappedFiles;
using System.Linq;
using System.Net.Http;
using System.Net.Http.Headers;
using System.Security.Cryptography;
using System.Text.Json;
using System.Text.RegularExpressions;
using System.Threading.Tasks;

namespace EssieuPilotNative;

public sealed class TrailerProfile
{
    public Guid Id { get; set; } = Guid.NewGuid();
    public string Name { get; set; } = "Remorque de test";
    public int Axles { get; set; } = 3;
    public decimal MaxAngle { get; set; } = 35;
    public decimal SteeringRate { get; set; } = 14;
    public decimal ReturnRate { get; set; } = 7;
    public bool Inverted { get; set; }
    public string EnableKey { get; set; } = "F8";
    public string LeftKey { get; set; } = "F9";
    public string RightKey { get; set; } = "F10";
}

internal sealed class ProfilesRepository
{
    public List<TrailerProfile> Read()
    {
        try
        {
            if (File.Exists(AppInfo.ProfilesPath))
            {
                var data = JsonSerializer.Deserialize<List<TrailerProfile>>(
                    File.ReadAllText(AppInfo.ProfilesPath));
                if (data is { Count: > 0 })
                    return data.Where(p => p != null && !string.IsNullOrWhiteSpace(p.Name))
                               .ToList();
            }
        }
        catch (Exception ex) { AppInfo.Log("Profile load failed: " + ex); }
        return new List<TrailerProfile> { new() };
    }

    public void Save(IEnumerable<TrailerProfile> profiles)
    {
        Directory.CreateDirectory(AppInfo.SettingsDirectory);
        var temp = AppInfo.ProfilesPath + ".tmp";
        var json = JsonSerializer.Serialize(profiles, new JsonSerializerOptions { WriteIndented = true });
        File.WriteAllText(temp, json);
        File.Move(temp, AppInfo.ProfilesPath, true);
    }
}

internal readonly record struct AtsSnapshot(bool? GameRunning, bool SteeringAcknowledged);

internal static class AtsStatusReader
{
    private const string AckMap = @"Local\EssieuPilotAppliedV1";
    public static AtsSnapshot Read()
    {
        bool? running = null;
        try
        {
            var processes = Process.GetProcessesByName("amtrucks");
            try { running = processes.Length > 0; }
            finally { foreach (var process in processes) process.Dispose(); }
        }
        catch (Exception ex)
        {
            AppInfo.Log("ATS enumeration unavailable: " + ex.Message);
        }
        return new AtsSnapshot(running, running == true && HasFreshAcknowledgement());
    }

    private static bool HasFreshAcknowledgement()
    {
        try
        {
            using var mapping = MemoryMappedFile.OpenExisting(AckMap, MemoryMappedFileRights.Read);
            using var view = mapping.CreateViewAccessor(0, 32, MemoryMappedFileAccess.Read);
            uint sequenceBefore = view.ReadUInt32(8);
            var bytes = new byte[32];
            view.ReadArray(0, bytes, 0, bytes.Length);
            uint sequenceAfter = view.ReadUInt32(8);
            return sequenceBefore == sequenceAfter &&
                   IsAcknowledgementValid(bytes, Environment.TickCount64);
        }
        catch (FileNotFoundException) { return false; }
        catch (Exception) { return false; }
    }

    internal static bool IsAcknowledgementValid(ReadOnlySpan<byte> bytes, long nowTick)
    {
        if (bytes.Length != 32) return false;
        var magic = BinaryPrimitives.ReadUInt32LittleEndian(bytes.Slice(0, 4));
        var version = BinaryPrimitives.ReadUInt32LittleEndian(bytes.Slice(4, 4));
        var begin = BinaryPrimitives.ReadUInt32LittleEndian(bytes.Slice(8, 4));
        var end = BinaryPrimitives.ReadUInt32LittleEndian(bytes.Slice(12, 4));
        var tick = BinaryPrimitives.ReadUInt64LittleEndian(bytes.Slice(16, 8));
        if (magic != 0x45504131 || version != 1 || begin != end || (begin & 1) != 0)
            return false;
        if (nowTick < 0) return false;
        ulong now = (ulong)nowTick;
        return now >= tick && now - tick <= 1500;
    }
}

internal sealed record UpdateCandidate(int Build, string Tag, string Url, string Sha256);

internal static class ReleaseCatalog
{
    private static readonly Regex Tag = new(
        @"^v2\.0\.0-native\.(\d+)$", RegexOptions.Compiled | RegexOptions.CultureInvariant);
    private static readonly Regex Hash = new(
        "^sha256:[0-9a-fA-F]{64}$", RegexOptions.Compiled | RegexOptions.CultureInvariant);

    public static UpdateCandidate? GetNewest(string json, int currentBuild)
    {
        using var doc = JsonDocument.Parse(json);
        UpdateCandidate? latest = null;
        foreach (var release in doc.RootElement.EnumerateArray())
        {
            if (!release.TryGetProperty("prerelease", out var pre) || !pre.GetBoolean())
                continue;
            if (release.TryGetProperty("draft", out var draft) && draft.GetBoolean())
                continue;
            var tag = release.GetProperty("tag_name").GetString() ?? "";
            var match = Tag.Match(tag);
            if (!match.Success || !int.TryParse(match.Groups[1].Value, out var build) ||
                build <= currentBuild || (latest != null && latest.Build >= build))
                continue;
            if (!release.TryGetProperty("assets", out var assets) ||
                assets.ValueKind != JsonValueKind.Array) continue;
            foreach (var asset in assets.EnumerateArray())
            {
                if (asset.GetProperty("name").GetString() != "Installer_EssieuPilot.exe")
                    continue;
                var url = asset.GetProperty("browser_download_url").GetString() ?? "";
                var digest = asset.TryGetProperty("digest", out var d) ? d.GetString() ?? "" : "";
                if (!Hash.IsMatch(digest) ||
                    !Uri.TryCreate(url, UriKind.Absolute, out var uri) ||
                    uri.Scheme != Uri.UriSchemeHttps ||
                    uri.Host != "github.com" || uri.Query != "" || uri.Fragment != "" ||
                    uri.AbsolutePath != $"/{AppInfo.GithubRepo}/releases/download/{tag}/Installer_EssieuPilot.exe")
                    continue;
                latest = new UpdateCandidate(build, tag, url, digest[7..].ToLowerInvariant());
                break;
            }
        }
        return latest;
    }
}

internal sealed class NativeUpdateService : IDisposable
{
    private readonly HttpClient _client = new() { Timeout = TimeSpan.FromMinutes(15) };
    public NativeUpdateService()
    {
        _client.DefaultRequestHeaders.UserAgent.ParseAdd("EssieuPilot-Native/2.0");
        _client.DefaultRequestHeaders.Accept.Add(
            new MediaTypeWithQualityHeaderValue("application/vnd.github+json"));
    }

    public async Task<UpdateCandidate?> CheckAsync()
    {
        var url = $"https://api.github.com/repos/{AppInfo.GithubRepo}/releases?per_page=50";
        using var response = await _client.GetAsync(url);
        response.EnsureSuccessStatusCode();
        var json = await response.Content.ReadAsStringAsync();
        return ReleaseCatalog.GetNewest(json, AppInfo.Build);
    }

    public async Task<string> DownloadVerifiedInstallerAsync(UpdateCandidate release, IProgress<int>? progress)
    {
        var dest = Path.Combine(Path.GetTempPath(), "EssieuPilot-Native-" +
            Guid.NewGuid().ToString("N") + ".exe");
        try
        {
            using var response = await _client.GetAsync(release.Url, HttpCompletionOption.ResponseHeadersRead);
            response.EnsureSuccessStatusCode();
            if (response.Content.Headers.ContentLength is long len && (len < 500_000 || len > 300_000_000))
                throw new InvalidDataException("Taille de téléchargement suspecte.");
            await using (var input = await response.Content.ReadAsStreamAsync())
            await using (var output = new FileStream(dest, FileMode.CreateNew, FileAccess.Write, FileShare.None, 131072, true))
            {
                long received = 0;
                var buffer = new byte[131072];
                while (true)
                {
                    int read = await input.ReadAsync(buffer);
                    if (read == 0) break;
                    await output.WriteAsync(buffer.AsMemory(0, read));
                    received += read;
                    if (received > 300_000_000) throw new InvalidDataException("Fichier trop volumineux.");
                    if (response.Content.Headers.ContentLength is long total && total > 0)
                        progress?.Report((int)Math.Min(100, received * 100 / total));
                }
            }
            await using (var file = File.OpenRead(dest))
            {
                var hash = Convert.ToHexString(await SHA256.HashDataAsync(file)).ToLowerInvariant();
                if (!string.Equals(hash, release.Sha256, StringComparison.Ordinal))
                    throw new InvalidDataException("Empreinte SHA-256 invalide. Installation annulée.");
            }
            progress?.Report(100);
            return dest;
        }
        catch
        {
            try { if (File.Exists(dest)) File.Delete(dest); } catch { }
            throw;
        }
    }

    public void Dispose() => _client.Dispose();
}
