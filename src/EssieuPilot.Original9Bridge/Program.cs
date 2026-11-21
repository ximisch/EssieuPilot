using System.Diagnostics;
using System.Security.Cryptography;
using System.Text.Json;
using System.Text.RegularExpressions;

namespace EssieuPilot.Original9;

// Pont externe, sans injection mémoire ni commande physique non vérifiée.
// Ne remplace pas l'interface officielle Original.8.
internal static class Program
{
    internal static readonly JsonSerializerOptions JsonOptions = new()
    {
        WriteIndented = true,
        PropertyNameCaseInsensitive = true
    };
    internal static readonly string StateFolder = Path.Combine(
        Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),
        "EssieuPilot", "Original9");
    internal static readonly string ProfileFolder = Path.Combine(
        Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData),
        "EssieuPilot", "Original9");
    private static readonly Uri HubUri = new("http://127.0.0.1:38471/snapshot");

    private static async Task<int> Main(string[] args)
    {
        try
        {
            if (args.Length == 0)
            {
                Console.WriteLine("EssieuPilot Original.9 — moteur de liaison ATS (préversion de développement)");
                Console.WriteLine("Commandes : status | watch | profiles add <id> <profil> <essieux> | install-modules <ats-root> <bundle-dir> <manifest.json> | self-test");
                return 0;
            }
            switch (args[0].ToLowerInvariant())
            {
                case "status":
                    await MonitorOnce(writeStatus: true);
                    return 0;
                case "watch":
                    Console.WriteLine("Surveillance en cours. Ctrl+C pour arrêter.");
                    while (true)
                    {
                        await MonitorOnce(writeStatus: true);
                        await Task.Delay(800);
                    }
                case "profiles" when args.Length == 5 &&
                    args[1].Equals("add", StringComparison.OrdinalIgnoreCase):
                    if (!int.TryParse(args[4], out int axleCount) || axleCount is < 1 or > 30)
                        throw new ArgumentException("Le nombre d'essieux doit être entre 1 et 30.");
                    ProfileStore.AddOrUpdate(args[2], args[3], axleCount);
                    Console.WriteLine("Association du profil enregistrée.");
                    return 0;
                case "install-modules" when args.Length == 4:
                    ModuleInstaller.Install(args[1], args[2], args[3]);
                    return 0;
                case "self-test":
                    SelfTest.Run();
                    return 0;
                default:
                    throw new ArgumentException("Commande inconnue ou arguments incomplets.");
            }
        }
        catch (OperationCanceledException) { return 0; }
        catch (Exception e)
        {
            Console.Error.WriteLine("EssieuPilot : " + e.Message);
            return 1;
        }
    }

    private static bool IsAtsRunning()
    {
        try
        {
            using var process = Process.GetProcessesByName("amtrucks").FirstOrDefault();
            return process is not null;
        }
        catch { return false; }
    }

    private static async Task MonitorOnce(bool writeStatus)
    {
        bool gameRunning = IsAtsRunning();
        bool hubConnected = false;
        TrailerSnapshot? snapshot = null;
        string detail = gameRunning ? "Hub télémétrique non accessible." : "ATS n'est pas lancé.";
        if (gameRunning)
        {
            try
            {
                using var client = new HttpClient { Timeout = TimeSpan.FromSeconds(2) };
                using var response = await client.GetAsync(HubUri);
                response.EnsureSuccessStatusCode();
                snapshot = SnapshotReader.Parse(await response.Content.ReadAsStringAsync());
                hubConnected = snapshot is not null;
                detail = hubConnected ? "Lecture télémétrique ATS." : "Télémétrie non ATS ou format incomplet.";
            }
            catch (Exception e) when (e is HttpRequestException or TaskCanceledException or JsonException)
            {
                detail = "Télémétrie indisponible : " + e.GetType().Name;
            }
        }
        var profiles = ProfileStore.Load();
        var match = snapshot is { Connected: true }
            ? profiles.FirstOrDefault(p => string.Equals(p.TrailerId, snapshot.TrailerId,
                StringComparison.Ordinal))
            : null;
        var status = new BridgeStatus(
            DateTimeOffset.UtcNow, gameRunning, hubConnected,
            snapshot?.Connected ?? false,
            snapshot?.TrailerId, snapshot?.TrailerName, snapshot?.EstimatedAxles,
            snapshot?.SpeedMps, match?.ProfileName,
            detail + (match is null && snapshot?.Connected == true
                ? " Remorque sans profil associé : aucune commande autorisée." : ""),
            PhysicalSteeringEnabled: false);
        Console.WriteLine(JsonSerializer.Serialize(status, JsonOptions));
        if (writeStatus)
        {
            Directory.CreateDirectory(StateFolder);
            AtomicFile.Write(Path.Combine(StateFolder, "bridge-status.json"),
                JsonSerializer.Serialize(status, JsonOptions));
        }
    }
}

public sealed record TrailerSnapshot(bool Connected, string? TrailerId,
    string? TrailerName, int? EstimatedAxles, double? SpeedMps);

public sealed record BridgeStatus(DateTimeOffset Timestamp, bool AtsRunning,
    bool TelemetryConnected, bool TrailerConnected, string? TrailerId,
    string? TrailerName, int? EstimatedAxles, double? SpeedMps,
    string? SelectedProfile, string Message, bool PhysicalSteeringEnabled);

public static class SnapshotReader
{
    // L'identifiant stable provient de trailer_spec.id, pas du nom commercial.
    // Le nombre d'essieux est une estimation géométrique à partir des positions Z.
    public static TrailerSnapshot? Parse(string json)
    {
        using var doc = JsonDocument.Parse(json);
        var root = doc.RootElement;
        if (!TryGet(root, "game", out var game) ||
            game.ValueKind != JsonValueKind.String ||
            !string.Equals(game.GetString(), "ats", StringComparison.OrdinalIgnoreCase))
            return null;
        if (!TryGet(root, "data", out var data) || data.ValueKind != JsonValueKind.Object)
            return null;
        bool attached = TryGet(data, "trailer", out var trailer) &&
            TryGet(trailer, "connected", out var flag) && flag.ValueKind == JsonValueKind.True;
        double? speed = TryGet(data, "speed_mps", out var sp) && sp.ValueKind == JsonValueKind.Number && sp.TryGetDouble(out var s)
            && double.IsFinite(s) ? s : null;
        if (!attached) return new(false, null, null, null, speed);

        if (!TryGet(root, "trailer_spec", out var spec)) return new(true, null, null, null, speed);
        string? id = GetString(spec, "id");
        if (string.IsNullOrWhiteSpace(id)) return new(true, null, null, null, speed);
        string? name = GetString(spec, "name");
        int? axles = null;
        if (TryGet(spec, "wheel_positions", out var wheels) &&
            wheels.ValueKind == JsonValueKind.Array)
        {
            var positions = new List<double>();
            foreach (var wheel in wheels.EnumerateArray())
            {
                if (TryGet(wheel, "z", out var z) && z.ValueKind == JsonValueKind.Number && z.TryGetDouble(out double position)
                    && double.IsFinite(position)) positions.Add(position);
            }
            if (positions.Count > 0)
            {
                positions.Sort();
                int count = 0;
                double? last = null;
                foreach (var z in positions)
                {
                    if (last is null || Math.Abs(z - last.Value) > 0.15)
                    {
                        count++;
                        last = z;
                    }
                }
                axles = count;
            }
        }
        return new(true, id.Trim(), name, axles, speed);
    }

    private static string? GetString(JsonElement e, string key) =>
        TryGet(e, key, out var value) && value.ValueKind == JsonValueKind.String
            ? value.GetString() : null;

    private static bool TryGet(JsonElement e, string key, out JsonElement value)
    {
        if (e.ValueKind == JsonValueKind.Object && e.TryGetProperty(key, out value))
            return true;
        value = default;
        return false;
    }
}

public sealed class TrailerProfileMap
{
    public string TrailerId { get; set; } = "";
    public string ProfileName { get; set; } = "";
    public int Axles { get; set; }
}

public static class ProfileStore
{
    private static string PathToFile => Path.Combine(Program.ProfileFolder, "profile-mappings.json");

    public static List<TrailerProfileMap> Load()
    {
        if (!File.Exists(PathToFile)) return new();
        return JsonSerializer.Deserialize<List<TrailerProfileMap>>(
            File.ReadAllText(PathToFile), Program.JsonOptions)
            ?? throw new InvalidDataException("Fichier de profils invalide.");
    }

    public static void AddOrUpdate(string trailerId, string profileName, int axles)
    {
        if (string.IsNullOrWhiteSpace(trailerId) || trailerId.Length > 256 ||
            string.IsNullOrWhiteSpace(profileName) || profileName.Length > 120)
            throw new ArgumentException("Identifiant et nom de profil obligatoires.");
        var items = Load();
        items.RemoveAll(x => x.TrailerId == trailerId);
        items.Add(new() { TrailerId = trailerId, ProfileName = profileName, Axles = axles });
        Directory.CreateDirectory(Program.ProfileFolder);
        AtomicFile.Write(PathToFile, JsonSerializer.Serialize(items, Program.JsonOptions));
    }
}

public static class AtomicFile
{
    public static void Write(string target, string content)
    {
        string temp = target + "." + Guid.NewGuid().ToString("N") + ".tmp";
        try
        {
            File.WriteAllText(temp, content);
            File.Move(temp, target, overwrite: true);
        }
        finally { if (File.Exists(temp)) File.Delete(temp); }
    }
}

public sealed class ModuleManifest
{
    public List<ModuleEntry> Modules { get; set; } = new();
}
public sealed class ModuleEntry
{
    public string File { get; set; } = "";
    public string Target { get; set; } = "";
    public string Sha256 { get; set; } = "";
}

public static class ModuleInstaller
{
    private static readonly Regex HashPattern = new("^[a-fA-F0-9]{64}$", RegexOptions.CultureInvariant);
    private static string OwnershipFile => Path.Combine(Program.StateFolder, "module-ownership.json");

    // Aucun module n'est téléchargé implicitement : seuls les binaires signés/hachés
    // par un manifeste explicite pourront être installés. SHA-256 = intégrité, pas signature.
    public static void Install(string gameRoot, string bundleDirectory, string manifestFile)
    {
        if (Process.GetProcessesByName("amtrucks").Length != 0)
            throw new InvalidOperationException("Fermer ATS avant toute installation de DLL ou de mods.");
        string root = Path.GetFullPath(gameRoot);
        string exe = Path.Combine(root, "bin", "win_x64", "amtrucks.exe");
        if (!File.Exists(exe))
            throw new DirectoryNotFoundException("Installation ATS invalide : " + exe);
        string bundle = Path.GetFullPath(bundleDirectory);
        var manifest = JsonSerializer.Deserialize<ModuleManifest>(
            File.ReadAllText(manifestFile), Program.JsonOptions)
            ?? throw new InvalidDataException("Manifeste vide.");
        if (manifest.Modules.Count == 0)
            throw new InvalidDataException("Aucun module autorisé dans le manifeste.");

        var owner = File.Exists(OwnershipFile)
            ? JsonSerializer.Deserialize<Dictionary<string, string>>(File.ReadAllText(OwnershipFile),
                Program.JsonOptions) ?? new(StringComparer.OrdinalIgnoreCase)
            : new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        owner = new Dictionary<string, string>(owner, StringComparer.OrdinalIgnoreCase);

        string modDir = Path.Combine(
            Environment.GetFolderPath(Environment.SpecialFolder.MyDocuments),
            "American Truck Simulator", "mod");
        var operations = new List<(string Source, string Destination, string Digest)>();
        var seen = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (var item in manifest.Modules)
        {
            if (string.IsNullOrWhiteSpace(item.File) ||
                item.File != Path.GetFileName(item.File) ||
                item.File.Contains('/') || item.File.Contains('\\') ||
                !HashPattern.IsMatch(item.Sha256))
                throw new InvalidDataException("Nom de fichier ou SHA-256 invalide.");
            bool plugin = item.Target.Equals("plugin", StringComparison.OrdinalIgnoreCase);
            bool mod = item.Target.Equals("mod", StringComparison.OrdinalIgnoreCase);
            if ((!plugin && !mod) ||
                (plugin && item.File is not ("EssieuPilotATS.dll" or "scs_telemetry_hub.dll")) ||
                (mod && !(item.File.StartsWith("EssieuPilot_", StringComparison.Ordinal) &&
                    item.File.EndsWith(".scs", StringComparison.OrdinalIgnoreCase))))
                throw new InvalidDataException("Module interdit : " + item.File);

            string src = Path.Combine(bundle, item.File);
            string dest = Path.Combine(plugin ? Path.Combine(root, "bin", "win_x64", "plugins") : modDir,
                item.File);
            if (!seen.Add(dest))
                throw new InvalidDataException("Module déclaré en double : " + item.File);
            if (!File.Exists(src))
                throw new FileNotFoundException("Module absent du paquet.", src);
            string actual = Convert.ToHexString(SHA256.HashData(File.ReadAllBytes(src)));
            if (!actual.Equals(item.Sha256, StringComparison.OrdinalIgnoreCase))
                throw new InvalidDataException("SHA-256 incorrect : " + item.File);
            if (File.Exists(dest))
            {
                string current = Convert.ToHexString(SHA256.HashData(File.ReadAllBytes(dest)));
                if (!current.Equals(actual, StringComparison.OrdinalIgnoreCase) &&
                    (!owner.TryGetValue(dest, out string? known) ||
                     !known.Equals(current, StringComparison.OrdinalIgnoreCase)))
                    throw new InvalidOperationException("Fichier existant non géré par EssieuPilot : " + dest);
            }
            operations.Add((src, dest, actual));
        }

        Directory.CreateDirectory(Program.StateFolder);
        string backup = Path.Combine(Program.StateFolder, "backups",
            DateTime.UtcNow.ToString("yyyyMMddHHmmss") + "-" + Guid.NewGuid().ToString("N"));
        Directory.CreateDirectory(backup);
        var undo = new List<Action>();
        try
        {
            for (int i = 0; i < operations.Count; i++)
            {
                var item = operations[i];
                Directory.CreateDirectory(Path.GetDirectoryName(item.Destination)!);
                string original = Path.Combine(backup, i + "-" + Path.GetFileName(item.Destination));
                if (File.Exists(item.Destination))
                {
                    File.Copy(item.Destination, original);
                    string restoreTarget = item.Destination;
                    string restoreSource = original;
                    undo.Add(() => File.Copy(restoreSource, restoreTarget, true));
                }
                else
                {
                    string deleteTarget = item.Destination;
                    undo.Add(() => { if (File.Exists(deleteTarget)) File.Delete(deleteTarget); });
                }
                string temp = item.Destination + "." + Guid.NewGuid().ToString("N") + ".tmp";
                try
                {
                    File.Copy(item.Source, temp, true);
                    File.Move(temp, item.Destination, true);
                }
                finally { if (File.Exists(temp)) File.Delete(temp); }
                owner[item.Destination] = item.Digest;
            }
            AtomicFile.Write(OwnershipFile, JsonSerializer.Serialize(owner, Program.JsonOptions));
            Console.WriteLine("Modules installés (fichiers vérifiés) : " + operations.Count);
            Console.WriteLine("Sauvegarde : " + backup);
            Console.WriteLine("Les mods .scs doivent être activés dans le gestionnaire de mods ATS.");
            Console.WriteLine("Installation physique des DLL != preuve de braquage des essieux.");
        }
        catch
        {
            for (int i = undo.Count - 1; i >= 0; i--)
            {
                try { undo[i](); }
                catch (Exception ex) { Console.Error.WriteLine("Restauration incomplète : " + ex.Message); }
            }
            throw;
        }
    }
}

internal static class SelfTest
{
    public static void Run()
    {
        const string attached = """
        {"type":"snapshot","game":"ats","data":{"speed_mps":1.25,"trailer":{"connected":true}},
         "trailer_spec":{"id":"scs.lowboy","name":"Lowboy","wheel_positions":[
          {"z":1.0},{"z":1.01},{"z":2.0},{"z":2.01},{"z":3.0},{"z":3.01}]}}
        """;
        var t = SnapshotReader.Parse(attached);
        Check(t is { Connected: true, TrailerId: "scs.lowboy", EstimatedAxles: 3 });
        Check(t.SpeedMps == 1.25);
        Check(SnapshotReader.Parse(attached.Replace("\"ats\"", "\"ets2\"")) is null);
        const string detached = """{"game":"ats","data":{"speed_mps":0,"trailer":{"connected":false}},"trailer_spec":{"id":"stale"}}""";
        Check(SnapshotReader.Parse(detached) is { Connected: false, TrailerId: null });
        Check(SnapshotReader.Parse("""{"game":"ats","data":{"trailer":{"connected":true}}}""") is { Connected: true, TrailerId: null });
        Check(SnapshotReader.Parse("""{"game":"ats","data":{"speed_mps":"bad","trailer":{"connected":false}}}""") is { Connected: false, SpeedMps: null });
        Console.WriteLine("Self-test : 6 assertions validées. Aucun essai physique ATS.");
    }
    private static void Check(bool result)
    {
        if (!result) throw new Exception("Échec des tests d'analyse télémétrique.");
    }
}
