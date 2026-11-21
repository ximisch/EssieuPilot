using System.Buffers.Binary;
using System.Diagnostics;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using System.Text.RegularExpressions;

namespace EssieuPilot.ATSInspector;

// Diagnostic R&D en lecture seule : aucun hook, aucune injection, aucun accès
// en écriture à ATS. Les offsets physiques historiques sont NON VALIDÉS.
internal static class Program
{
    private const string SteeringPattern =
        "48 89 5C 24 08 48 89 74 24 10 57 48 83 EC ?? 8B 41 ?? 48 8B D9 0F 29 74";
    private const int MaxExeBytes = 1024 * 1024 * 1024;
    private static readonly JsonSerializerOptions JsonOptions = new() { WriteIndented = true };

    private static int Main(string[] args)
    {
        try
        {
            Console.OutputEncoding = Encoding.UTF8;
            Console.Title = "EssieuPilot";
            Console.WriteLine("EssieuPilot — contrôle de compatibilité ATS (lecture seule)");
            if (args.Length > 0 && args[0] == "--self-test")
                return SelfTest();

            string? executable = args.Length > 0 ? NormalizePath(args[0]) : FindAtsExe();
            if (executable is null)
            {
                Console.Error.WriteLine(
                    "ATS non localisé. Dépose amtrucks.exe sur cet outil, ou exécute : " +
                    "EssieuPilot-ControleATS.exe \"C:\\...\\American Truck Simulator\\bin\\win_x64\\amtrucks.exe\"");
                Console.WriteLine("Aucun fichier ATS modifié.");
                return 2;
            }
            executable = Path.GetFullPath(executable);
            if (!File.Exists(executable) ||
                !Path.GetFileName(executable).Equals("amtrucks.exe", StringComparison.OrdinalIgnoreCase))
                throw new InvalidDataException("Sélectionne uniquement l'exécutable du jeu : amtrucks.exe.");

            var info = new FileInfo(executable);
            if (info.Length < 1024 * 1024 || info.Length > MaxExeBytes)
                throw new InvalidDataException("Taille de l'exécutable ATS inattendue.");
            byte[] raw = File.ReadAllBytes(executable);
            if (!IsPe64(raw))
                throw new InvalidDataException("Le fichier n'est pas un exécutable Windows PE x64.");
            string version = FileVersionInfo.GetVersionInfo(executable).FileVersion ?? "Non renseignée";
            string sha256 = Convert.ToHexString(SHA256.HashData(raw)).ToLowerInvariant();
            var findings = new List<SignatureFinding>
            {
                FindSignature(raw, "Fonction ancienne de direction individuelle",
                    SteeringPattern, 12)
            };
            var report = new InspectionReport(
                "ATS physique — vérification préliminaire sans commande",
                "ats", version, info.Length, sha256,
                DateTimeOffset.UtcNow, findings,
                "NON_VALIDÉ : la signature seule ne vérifie ni le vtable[73], " +
                "ni le champ steering (offset ancien 0x04E8), ni wheel_steering_stuff " +
                "(ancien 0x04F0), ni le mouvement physique/visuel.",
                "AUCUN HOOK INSTALLÉ. AUCUN FICHIER ATS MODIFIÉ. " +
                "NE PAS UTILISER LA DLL EXPÉRIMENTALE AVANT VALIDATION.");
            string documents = Environment.GetFolderPath(Environment.SpecialFolder.MyDocuments);
            if (string.IsNullOrWhiteSpace(documents))
                throw new DirectoryNotFoundException("Dossier Documents introuvable.");
            string folder = Path.Combine(documents, "EssieuPilot", "Diagnostics");
            Directory.CreateDirectory(folder);
            string target = Path.Combine(folder, "Rapport_Compatibilite_ATS.json");
            WriteAtomically(target, JsonSerializer.Serialize(report, JsonOptions));

            Console.WriteLine("Fichier analysé : amtrucks.exe");
            Console.WriteLine("Version Windows : " + report.FileVersion);
            Console.WriteLine("Taille : " + report.SizeBytes + " octets");
            Console.WriteLine("SHA-256 : " + report.Sha256);
            foreach (var finding in findings)
                Console.WriteLine("Signature ancienne : " + finding.Count +
                    " occurrence(s). Offsets : " + string.Join(", ", finding.Offsets));
            Console.WriteLine("Compatibilité physique : NON VALIDÉE.");
            Console.WriteLine("Rapport créé : " + target);
            Console.WriteLine("Ce rapport peut être transmis pour vérifier les hypothèses avant tout essai physique.");
            return 0;
        }
        catch (Exception ex)
        {
            Console.Error.WriteLine("Contrôle impossible : " + ex.Message);
            return 1;
        }
    }

    private static string? NormalizePath(string path)
    {
        string candidate = path.Trim().Trim('"');
        if (Directory.Exists(candidate))
            return Path.Combine(candidate, "bin", "win_x64", "amtrucks.exe");
        return candidate;
    }

    private static string? FindAtsExe()
    {
        var roots = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        void AddRoot(string? path)
        {
            if (!string.IsNullOrWhiteSpace(path) && Directory.Exists(path))
                roots.Add(Path.GetFullPath(path));
        }
        AddRoot(Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ProgramFilesX86), "Steam"));
        AddRoot(Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ProgramFiles), "Steam"));
        foreach (string drive in DriveInfo.GetDrives()
            .Where(d => d.DriveType == DriveType.Fixed && d.IsReady).Select(d => d.RootDirectory.FullName))
        {
            AddRoot(Path.Combine(drive, "Steam"));
            AddRoot(Path.Combine(drive, "SteamLibrary"));
            AddRoot(Path.Combine(drive, "Program Files (x86)", "Steam"));
        }
        try
        {
            using var key = Microsoft.Win32.Registry.CurrentUser.OpenSubKey(@"Software\Valve\Steam");
            AddRoot(key?.GetValue("SteamPath")?.ToString()?.Replace('/', Path.DirectorySeparatorChar));
        }
        catch { /* no rights to Steam registry, fall back to default paths */ }

        var gameRoots = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (string root in roots)
        {
            gameRoots.Add(root);
            string vdf = Path.Combine(root, "steamapps", "libraryfolders.vdf");
            if (File.Exists(vdf))
            {
                foreach (Match match in Regex.Matches(File.ReadAllText(vdf),
                    "\"path\"\\s+\"([^\"]+)\"", RegexOptions.IgnoreCase))
                {
                    string library = match.Groups[1].Value.Replace(@"\\", @"\");
                    if (Directory.Exists(library))
                        gameRoots.Add(library);
                }
            }
        }
        return gameRoots
            .Select(root => Path.Combine(root, "steamapps", "common",
                "American Truck Simulator", "bin", "win_x64", "amtrucks.exe"))
            .Where(File.Exists)
            .OrderByDescending(File.GetLastWriteTimeUtc)
            .FirstOrDefault();
    }

    internal static bool IsPe64(byte[] data)
    {
        if (data.Length < 0x100 || data[0] != 'M' || data[1] != 'Z')
            return false;
        int offset = BinaryPrimitives.ReadInt32LittleEndian(data.AsSpan(0x3C, 4));
        return offset >= 0x40 && offset + 24 <= data.Length &&
            data[offset] == 'P' && data[offset + 1] == 'E' &&
            data[offset + 2] == 0 && data[offset + 3] == 0 &&
            BinaryPrimitives.ReadUInt16LittleEndian(data.AsSpan(offset + 4, 2)) == 0x8664;
    }

    internal static SignatureFinding FindSignature(byte[] buffer, string name,
        string pattern, int maxOffsets)
    {
        int[] bytes = pattern.Split(' ', StringSplitOptions.RemoveEmptyEntries)
            .Select(s => s is "?" or "??" ? -1 : Convert.ToInt32(s, 16)).ToArray();
        if (bytes.Length < 4 || bytes.Length > 128 || bytes[0] < 0 ||
            bytes.Any(b => b < -1 || b > 255))
            throw new ArgumentException("Signature invalide.");
        int count = 0;
        var offsets = new List<string>();
        int limit = buffer.Length - bytes.Length;
        int current = 0;
        while (current <= limit)
        {
            int located = buffer.AsSpan(current, limit - current + 1)
                .IndexOf((byte)bytes[0]);
            if (located < 0) break;
            current += located;
            bool match = true;
            for (int i = 1; i < bytes.Length; i++)
            {
                if (bytes[i] >= 0 && buffer[current + i] != bytes[i])
                {
                    match = false;
                    break;
                }
            }
            if (match)
            {
                count++;
                if (offsets.Count < maxOffsets)
                    offsets.Add("0x" + current.ToString("X"));
            }
            current++;
        }
        return new SignatureFinding(name, count, offsets);
    }

    private static void WriteAtomically(string path, string content)
    {
        string tmp = path + "." + Guid.NewGuid().ToString("N") + ".tmp";
        try
        {
            File.WriteAllText(tmp, content, new UTF8Encoding(false));
            File.Move(tmp, path, true);
        }
        finally
        {
            if (File.Exists(tmp)) File.Delete(tmp);
        }
    }

    private static int SelfTest()
    {
        byte[] fakePe = new byte[512];
        fakePe[0] = (byte)'M'; fakePe[1] = (byte)'Z';
        BinaryPrimitives.WriteInt32LittleEndian(fakePe.AsSpan(0x3C, 4), 0x80);
        fakePe[0x80] = (byte)'P'; fakePe[0x81] = (byte)'E';
        BinaryPrimitives.WriteUInt16LittleEndian(fakePe.AsSpan(0x84, 2), 0x8664);
        if (!IsPe64(fakePe)) throw new Exception("PE x64 valide rejeté.");
        fakePe[0x84] = 0x4c;
        if (IsPe64(fakePe)) throw new Exception("Architecture non x64 acceptée.");
        byte[] bytes = { 0x48, 0x89, 0x5C, 0x24, 0x08, 0x48, 0x89, 0x5C, 0x24, 0x09 };
        var signature = FindSignature(bytes, "test", "48 89 ?? 24 08", 5);
        if (signature.Count != 1 || signature.Offsets.Single() != "0x0")
            throw new Exception("Recherche de signature incorrecte.");
        var noMatch = FindSignature(bytes, "test", "48 89 5C 24 07", 5);
        if (noMatch.Count != 0) throw new Exception("Fausse correspondance de signature.");
        Console.WriteLine("Self-test réussi : validation PE et recherche de signatures.");
        return 0;
    }
}

internal sealed record SignatureFinding(string Name, int Count, List<string> Offsets);
internal sealed record InspectionReport(string Purpose, string Game,
    string FileVersion, long SizeBytes, string Sha256,
    DateTimeOffset InspectedAt, List<SignatureFinding> Signatures,
    string PhysicalCompatibility, string SafetyStatus);
