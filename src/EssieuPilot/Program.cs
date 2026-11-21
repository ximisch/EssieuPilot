using System.Diagnostics;
using System.Drawing;
using System.Net.Http.Headers;
using System.Security.Cryptography;
using System.Text.Json;
using System.Text.RegularExpressions;
using System.Windows.Forms;

namespace EssieuPilot;

internal static class Program
{
    [STAThread]
    private static void Main()
    {
        ApplicationConfiguration.Initialize();
        Application.Run(new MainWindow());
    }
}

public sealed class TrailerProfile
{
    public string Name { get; set; } = "Remorque de test";
    public int Axles { get; set; } = 3;
    public decimal MaxAngle { get; set; } = 35;
    public decimal SteeringRate { get; set; } = 14;
    public decimal ReturnRate { get; set; } = 7;
    public bool Inverted { get; set; }
}

internal sealed record UpdateCandidate(string Tag, string Url, string Sha256);

public sealed class MainWindow : Form
{
#if TEST_VERSION_4
    private const string TechnicalVersion = "1.7.0-test.4";
#elif TEST_VERSION_3
    private const string TechnicalVersion = "1.7.0-test.3";
#elif TEST_VERSION_2
    private const string TechnicalVersion = "1.7.0-test.2";
#else
    private const string TechnicalVersion = "1.7.0-test.1";
#endif
    private static readonly Color Night = Color.FromArgb(13, 23, 38);
    private static readonly Color Surface = Color.FromArgb(24, 38, 56);
    private static readonly Color PanelColor = Color.FromArgb(30, 49, 69);
    private static readonly Color Aqua = Color.FromArgb(45, 220, 212);
    private static readonly Color Fore = Color.FromArgb(233, 243, 247);
    private static readonly Color Muted = Color.FromArgb(168, 190, 202);
    private const string ReleaseList = "https://api.github.com/repos/ximisch/EssieuPilot/releases?per_page=30";
    private const string DownloadPrefix = "https://github.com/ximisch/EssieuPilot/releases/download/";
    private static readonly Regex TagExpression = new(@"^v(?<major>\d+)\.(?<minor>\d+)\.(?<patch>\d+)-test\.(?<build>\d+)$", RegexOptions.Compiled);

    private readonly string _profileFile = Path.Combine(
        Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), "EssieuPilot-Test", "profiles.json");
    private readonly HttpClient _client = new() { Timeout = TimeSpan.FromMinutes(10) };
    private readonly List<TrailerProfile> _profiles = new();
    private readonly Panel _body = new();
    private readonly Label _status = new();
    private readonly System.Windows.Forms.Timer _timer = new() { Interval = 50 };
    private TrailerProfile _profile = new();
    private DateTime _lastTick = DateTime.UtcNow;
    private Label? _angleText;
    private bool _left;
    private bool _right;
    private bool _forward;
    private double _angle;
    private UpdateCandidate? _update;
    private Button? _install;
    private Label? _updateMessage;

    public MainWindow()
    {
        Text = "EssieuPilot";
        ClientSize = new Size(1020, 675);
        MinimumSize = MaximumSize = Size;
        FormBorderStyle = FormBorderStyle.FixedSingle;
        MaximizeBox = false;
        StartPosition = FormStartPosition.CenterScreen;
        BackColor = Night;
        ForeColor = Fore;
        Font = new Font("Segoe UI", 10f);
        KeyPreview = true;
        _client.DefaultRequestHeaders.UserAgent.ParseAdd("EssieuPilot-Test/1.0");
        _client.DefaultRequestHeaders.Accept.Add(new MediaTypeWithQualityHeaderValue("application/vnd.github+json"));
        LoadProfiles();

        var header = Box(0, 0, 1020, 91, Surface, this);
        header.Controls.Add(LabelAt("EssieuPilot", 26, 10, 570, 48, 28, Aqua, true));
        header.Controls.Add(LabelAt("PILOTAGE DES ESSIEUX DIRECTEURS  •  AMERICAN TRUCK SIMULATOR", 29, 57, 748, 26, 9, Muted));
        header.Controls.Add(LabelAt("● MODE TEST", 843, 29, 163, 24, 10, Aqua, true));

        var side = Box(0, 91, 220, 541, Surface, this);
        side.Controls.Add(ActionButton("Tableau de bord", 14, 22, 192, 45, ShowDashboard));
        side.Controls.Add(ActionButton("Mes remorques", 14, 81, 192, 45, ShowProfiles));
        side.Controls.Add(ActionButton("Mises à jour", 14, 140, 192, 45, ShowUpdates));
        side.Controls.Add(LabelAt("ÉTAT DU JEU", 15, 346, 190, 26, 9, Muted, true));
        side.Controls.Add(LabelAt("● Hors ligne", 15, 374, 190, 28, 12, Color.Orange, true));
        side.Controls.Add(LabelAt("ESSIEUX DIRECTEURS", 15, 410, 190, 26, 9, Muted, true));
        side.Controls.Add(LabelAt("● Inactifs", 15, 438, 190, 28, 12, Color.Orange, true));
        side.Controls.Add(LabelAt("TEST : aucune DLL ATS", 15, 475, 200, 27, 9, Muted));
        side.Controls.Add(LabelAt(TechnicalVersion, 15, 503, 185, 25, 9, Muted));

        _body.SetBounds(239, 111, 757, 505);
        _body.BackColor = Night;
        Controls.Add(_body);

        _status.SetBounds(28, 640, 954, 28);
        _status.Text = "Aucune commande physique dans ATS ni compatibilité Convoi validée.";
        _status.ForeColor = Muted;
        Controls.Add(_status);

        KeyDown += (_, e) =>
        {
            if (e.KeyCode == Keys.A) _left = true;
            if (e.KeyCode == Keys.D) _right = true;
        };
        KeyUp += (_, e) =>
        {
            if (e.KeyCode == Keys.A) _left = false;
            if (e.KeyCode == Keys.D) _right = false;
        };
        Deactivate += (_, _) => { _left = false; _right = false; };
        _timer.Tick += (_, _) => Simulate();
        _timer.Start();
        ShowDashboard();
    }

    private static Panel Box(int x, int y, int w, int h, Color color, Control parent)
    {
        var p = new Panel { Bounds = new Rectangle(x, y, w, h), BackColor = color };
        parent.Controls.Add(p);
        return p;
    }

    private static Label LabelAt(string text, int x, int y, int w, int h, int size, Color color, bool strong = false)
    {
        return new Label
        {
            Text = text,
            Location = new Point(x, y),
            Size = new Size(w, h),
            ForeColor = color,
            BackColor = Color.Transparent,
            AutoEllipsis = false,
            Font = new Font("Segoe UI", size, strong ? FontStyle.Bold : FontStyle.Regular)
        };
    }

    private static Button ActionButton(string text, int x, int y, int w, int h, Action click, bool primary = false)
    {
        var b = new Button
        {
            Text = text,
            Bounds = new Rectangle(x, y, w, h),
            FlatStyle = FlatStyle.Flat,
            BackColor = primary ? Aqua : PanelColor,
            ForeColor = primary ? Night : Fore,
            Font = new Font("Segoe UI", 10, FontStyle.Bold),
            Cursor = Cursors.Hand
        };
        b.FlatAppearance.BorderColor = primary ? Aqua : Color.FromArgb(53, 83, 103);
        b.FlatAppearance.BorderSize = 1;
        b.Click += (_, _) => click();
        return b;
    }

    private void ResetPage(string title, string description)
    {
        _body.Controls.Clear();
        _angleText = null;
        _body.Controls.Add(LabelAt(title, 12, 7, 705, 45, 21, Aqua, true));
        _body.Controls.Add(LabelAt(description, 14, 52, 710, 36, 10, Muted));
    }

    private void ShowDashboard()
    {
        ResetPage("Tableau de bord", "Vue d'ensemble de la remorque et simulation locale des essieux.");
        var card = Box(13, 98, 722, 392, Surface, _body);
        card.Controls.Add(LabelAt("SIMULATION 3D  ·  REMORQUE", 20, 12, 440, 30, 12, Fore, true));
        card.Controls.Add(LabelAt("Angle des essieux", 515, 12, 185, 24, 10, Muted));
        _angleText = LabelAt("0,0°", 515, 36, 180, 45, 26, Aqua, true);
        card.Controls.Add(_angleText);
        var panorama = new Panel { Bounds = new Rectangle(18, 86, 686, 155), BackColor = Color.FromArgb(16, 34, 49) };
        panorama.Paint += (_, e) =>
        {
            var g = e.Graphics;
            g.SmoothingMode = System.Drawing.Drawing2D.SmoothingMode.AntiAlias;
            using var road = new Pen(Color.FromArgb(55, 108, 120), 2);
            using var highlight = new Pen(Aqua, 3);
            using var steel = new SolidBrush(Color.FromArgb(81, 136, 155));
            using var wheel = new SolidBrush(Color.FromArgb(41, 211, 205));
            using var cabin = new SolidBrush(Color.FromArgb(28, 151, 166));
            g.DrawLine(road, 20, 128, 661, 128);
            g.DrawLine(road, 95, 104, 596, 104);
            g.FillPolygon(steel, new [] { new Point(84, 73), new Point(533, 73), new Point(585, 98), new Point(124, 98) });
            g.DrawLine(highlight, 85, 72, 534, 72);
            g.FillRectangle(cabin, 36, 39, 80, 58);
            g.FillRectangle(cabin, 58, 20, 48, 28);
            g.FillRectangle(steel, 105, 77, 74, 16);
            for (var i = 0; i < Math.Clamp(_profile.Axles, 1, 8); i++)
            {
                int x = 430 + i * 27;
                g.FillEllipse(wheel, x, 94, 19, 19);
                g.FillEllipse(Brushes.Black, x + 5, 99, 9, 9);
            }
            using var titleFont = new Font("Segoe UI", 10, FontStyle.Bold);
            using var noteFont = new Font("Segoe UI", 8);
            g.DrawString(_profile.Name, titleFont, Brushes.White, 20, 7);
            g.DrawString("Aperçu schématique · non connecté à ATS", noteFont, Brushes.LightGray, 390, 132);
        };
        card.Controls.Add(panorama);
        var conn = Box(18, 254, 330, 58, PanelColor, card);
        conn.Controls.Add(LabelAt("CONNEXION ATS", 14, 7, 190, 22, 9, Muted, true));
        conn.Controls.Add(LabelAt("● Hors ligne", 14, 29, 255, 25, 11, Color.Orange, true));
        var active = Box(360, 254, 344, 58, PanelColor, card);
        active.Controls.Add(LabelAt("ESSIEUX DIRECTEURS", 14, 7, 260, 22, 9, Muted, true));
        active.Controls.Add(LabelAt("● Inactifs", 14, 29, 260, 25, 11, Color.Orange, true));
        var move = new CheckBox
        {
            Text = "Le véhicule avance (recentrage automatique simulé)",
            Bounds = new Rectangle(20, 318, 661, 30),
            Checked = _forward, ForeColor = Fore, BackColor = Surface
        };
        move.CheckedChanged += (_, _) => _forward = move.Checked;
        card.Controls.Add(move);
        card.Controls.Add(LabelAt("A / D : direction · Profil : " + _profile.Name + " · " + _profile.Axles + " essieux",
            20, 354, 675, 29, 10, Aqua));
        _status.Text = "Simulation uniquement : braquage ATS et Convoi non validés.";
    }

    private void Simulate()
    {
        var now = DateTime.UtcNow;
        double dt = Math.Min(0.15, Math.Max(0, (now - _lastTick).TotalSeconds));
        _lastTick = now;
        double max = (double)_profile.MaxAngle;
        if (_forward)
        {
            double move = (double)_profile.ReturnRate * dt;
            _angle = Math.Abs(_angle) <= move ? 0 : _angle - Math.Sign(_angle) * move;
        }
        else
        {
            int direction = (_right ? 1 : 0) - (_left ? 1 : 0);
            if (_profile.Inverted) direction = -direction;
            _angle = Math.Clamp(_angle + direction * (double)_profile.SteeringRate * dt, -max, max);
        }
        if (_angleText is not null) _angleText.Text = _angle.ToString("0.0") + "°";
    }

    private void ShowProfiles()
    {
        ResetPage("Mes remorques", "Créer, modifier et supprimer les profils de remorques.");
        var card = Box(13, 99, 722, 382, Surface, _body);
        var listing = new ListBox
        {
            Bounds = new Rectangle(18, 22, 231, 281),
            BackColor = PanelColor,
            ForeColor = Fore,
            BorderStyle = BorderStyle.None
        };
        listing.Items.AddRange(_profiles.Select(p => p.Name).Cast<object>().ToArray());
        card.Controls.Add(listing);

        var name = Input(267, 52, 409, _profile.Name, card);
        var axles = Number(267, 121, 80, _profile.Axles, 1, 30, card);
        var max = Number(370, 121, 90, _profile.MaxAngle, 1, 90, card);
        var turn = Number(479, 121, 90, _profile.SteeringRate, 1, 90, card);
        var center = Number(587, 121, 90, _profile.ReturnRate, 1, 90, card);
        var invert = new CheckBox
        {
            Text = "Inverser la direction",
            Bounds = new Rectangle(266, 180, 337, 35),
            Checked = _profile.Inverted,
            ForeColor = Fore
        };
        card.Controls.Add(invert);
        card.Controls.Add(LabelAt("Nom", 267, 22, 150, 26, 10, Muted));
        card.Controls.Add(LabelAt("Essieux", 267, 95, 93, 26, 10, Muted));
        card.Controls.Add(LabelAt("Max °", 370, 95, 88, 26, 10, Muted));
        card.Controls.Add(LabelAt("Braq °/s", 479, 95, 92, 26, 10, Muted));
        card.Controls.Add(LabelAt("Retour °/s", 587, 95, 102, 26, 10, Muted));

        listing.SelectedIndexChanged += (_, _) =>
        {
            if (listing.SelectedIndex < 0 || listing.SelectedIndex >= _profiles.Count) return;
            _profile = _profiles[listing.SelectedIndex];
            name.Text = _profile.Name;
            axles.Value = _profile.Axles;
            max.Value = _profile.MaxAngle;
            turn.Value = _profile.SteeringRate;
            center.Value = _profile.ReturnRate;
            invert.Checked = _profile.Inverted;
        };

        card.Controls.Add(ActionButton("Nouveau profil", 18, 318, 231, 40, () =>
        {
            var p = new TrailerProfile { Name = "Remorque " + (_profiles.Count + 1) };
            _profiles.Add(p);
            _profile = p;
            SaveProfiles();
            ShowProfiles();
        }));

        card.Controls.Add(ActionButton("Supprimer le profil", 267, 298, 409, 40, () =>
        {
            if (_profiles.Count <= 1)
            {
                MessageBox.Show("Le dernier profil ne peut pas être supprimé.", "EssieuPilot",
                    MessageBoxButtons.OK, MessageBoxIcon.Information);
                return;
            }
            var selected = _profile;
            if (MessageBox.Show("Supprimer définitivement le profil « " + selected.Name + " » ?",
                "EssieuPilot — suppression", MessageBoxButtons.YesNo,
                MessageBoxIcon.Warning) != DialogResult.Yes) return;
            _profiles.Remove(selected);
            _profile = _profiles[0];
            SaveProfiles();
            ShowProfiles();
            _status.Text = "Profil supprimé : " + selected.Name;
        }));

        card.Controls.Add(ActionButton("Enregistrer le profil", 267, 244, 409, 44, () =>
        {
            _profile.Name = string.IsNullOrWhiteSpace(name.Text) ? "Remorque de test" : name.Text.Trim();
            _profile.Axles = (int)axles.Value;
            _profile.MaxAngle = max.Value;
            _profile.SteeringRate = turn.Value;
            _profile.ReturnRate = center.Value;
            _profile.Inverted = invert.Checked;
            SaveProfiles();
            ShowProfiles();
            _status.Text = "Profil enregistré dans %APPDATA%\\EssieuPilot-Test.";
        }, true));
        card.Controls.Add(LabelAt("Ces réglages n'agissent pas sur les essieux du jeu.", 267, 349, 412, 26, 9, Muted));
        listing.SelectedIndex = Math.Max(0, _profiles.IndexOf(_profile));
    }

    private static TextBox Input(int x, int y, int w, string value, Control parent)
    {
        var box = new TextBox { Bounds = new Rectangle(x, y, w, 31), Text = value, BackColor = PanelColor, ForeColor = Fore, BorderStyle = BorderStyle.FixedSingle };
        parent.Controls.Add(box);
        return box;
    }

    private static NumericUpDown Number(int x, int y, int w, decimal value, int min, int max, Control parent)
    {
        var n = new NumericUpDown
        {
            Bounds = new Rectangle(x, y, w, 32),
            Minimum = min, Maximum = max, Value = Math.Clamp(value, min, max),
            BackColor = PanelColor, ForeColor = Fore,
            BorderStyle = BorderStyle.FixedSingle
        };
        parent.Controls.Add(n);
        return n;
    }

    private void LoadProfiles()
    {
        try
        {
            if (File.Exists(_profileFile))
            {
                var list = JsonSerializer.Deserialize<List<TrailerProfile>>(File.ReadAllText(_profileFile));
                if (list is not null) _profiles.AddRange(list.Where(p => p is not null && !string.IsNullOrWhiteSpace(p.Name)));
            }
        }
        catch (Exception e)
        {
            MessageBox.Show("Profils illisibles, aucun fichier supprimé : " + e.Message, "EssieuPilot");
        }
        if (_profiles.Count == 0) _profiles.Add(new TrailerProfile());
        _profile = _profiles[0];
    }

    private void SaveProfiles()
    {
        Directory.CreateDirectory(Path.GetDirectoryName(_profileFile)!);
        string temp = _profileFile + ".tmp";
        File.WriteAllText(temp, JsonSerializer.Serialize(_profiles, new JsonSerializerOptions { WriteIndented = true }));
        File.Move(temp, _profileFile, true);
    }

    private void ShowUpdates()
    {
        ResetPage("Mises à jour", "Recherche sur GitHub ; téléchargement et installation seulement après ton accord.");
        var card = Box(13, 101, 722, 307, Surface, _body);
        card.Controls.Add(LabelAt("Version technique installée : " + TechnicalVersion, 22, 22, 660, 35, 13, Fore, true));
        _updateMessage = LabelAt("Clique sur « Vérifier les mises à jour ».", 22, 81, 677, 63, 12, Muted);
        card.Controls.Add(_updateMessage);
        card.Controls.Add(ActionButton("Vérifier les mises à jour", 22, 171, 299, 48, async () => await CheckUpdates()));
        _install = ActionButton("Installer la mise à jour", 342, 171, 346, 48, async () => await InstallUpdate(), true);
        _install.Enabled = _update is not null;
        card.Controls.Add(_install);
        card.Controls.Add(LabelAt("Un programme non signé peut déclencher SmartScreen.", 22, 245, 674, 26, 10, Muted));
        card.Controls.Add(LabelAt("Profils chargés : " + _profiles.Count + " | Profil actif : " + _profile.Name, 22, 275, 674, 26, 10, Aqua));
        _status.Text = "Le test n'installe aucune DLL dans ATS.";
    }

    private static Version? ParseTag(string? tag)
    {
        var match = TagExpression.Match(tag ?? "");
        if (!match.Success) return null;
        return Version.TryParse(
            match.Groups["major"].Value + "." + match.Groups["minor"].Value + "." +
            match.Groups["patch"].Value + "." + match.Groups["build"].Value,
            out var v) ? v : null;
    }

    private async Task CheckUpdates()
    {
        _update = null;
        if (_install is not null) _install.Enabled = false;
        if (_updateMessage is not null) _updateMessage.Text = "Interrogation des publications GitHub…";
        try
        {
            using var response = await _client.GetAsync(ReleaseList);
            response.EnsureSuccessStatusCode();
            using var json = JsonDocument.Parse(await response.Content.ReadAsStringAsync());
            var installed = ParseTag("v" + TechnicalVersion) ?? throw new InvalidOperationException("Version locale incorrecte.");
            var available = new List<(Version Version, UpdateCandidate Candidate)>();
            foreach (var release in json.RootElement.EnumerateArray())
            {
                if (!release.TryGetProperty("tag_name", out var tagNode)) continue;
                string tag = tagNode.GetString() ?? "";
                var version = ParseTag(tag);
                if (version is null || version <= installed) continue;
                if (!release.TryGetProperty("prerelease", out var pre) || !pre.GetBoolean()) continue;
                if (!release.TryGetProperty("assets", out var assets)) continue;
                foreach (var asset in assets.EnumerateArray())
                {
                    if (asset.GetProperty("name").GetString() != "Installer_EssieuPilot.exe") continue;
                    string url = asset.GetProperty("browser_download_url").GetString() ?? "";
                    string digest = asset.TryGetProperty("digest", out var d) ? d.GetString() ?? "" : "";
                    if (!url.StartsWith(DownloadPrefix, StringComparison.Ordinal) ||
                        !Regex.IsMatch(digest, "^sha256:[0-9a-fA-F]{64}$")) continue;
                    available.Add((version, new UpdateCandidate(tag, url, digest[7..].ToLowerInvariant())));
                }
            }
            var latest = available.OrderByDescending(x => x.Version).FirstOrDefault();
            if (latest.Candidate is null)
            {
                if (_updateMessage is not null) _updateMessage.Text = "Aucune mise à jour de test plus récente n'est disponible.";
                return;
            }
            _update = latest.Candidate;
            if (_install is not null) _install.Enabled = true;
            if (_updateMessage is not null) _updateMessage.Text = "Mise à jour disponible : " + _update.Tag + "\nFichier : Installer_EssieuPilot.exe";
        }
        catch (Exception ex)
        {
            if (_updateMessage is not null) _updateMessage.Text = "Vérification impossible : " + ex.Message;
        }
    }

    private async Task InstallUpdate()
    {
        if (_update is null) return;
        if (MessageBox.Show("Télécharger puis lancer " + _update.Tag +
            " ? Les profils de test seront conservés. Confirmer l'installation ?",
            "EssieuPilot — mise à jour", MessageBoxButtons.YesNo, MessageBoxIcon.Question) != DialogResult.Yes) return;

        var candidate = _update;
        if (_install is not null) _install.Enabled = false;
        string target = Path.Combine(Path.GetTempPath(), "EssieuPilot-Update-" + Guid.NewGuid().ToString("N") + ".exe");
        try
        {
            if (_updateMessage is not null) _updateMessage.Text = "Téléchargement sécurisé en cours…";
            using (var response = await _client.GetAsync(candidate.Url, HttpCompletionOption.ResponseHeadersRead))
            {
                response.EnsureSuccessStatusCode();
                await using var source = await response.Content.ReadAsStreamAsync();
                await using var output = File.Create(target);
                await source.CopyToAsync(output);
            }
            if (_updateMessage is not null) _updateMessage.Text = "Vérification de l'intégrité SHA-256…";
            await using (var file = File.OpenRead(target))
            {
                string calculated = Convert.ToHexString(await SHA256.HashDataAsync(file)).ToLowerInvariant();
                if (!string.Equals(calculated, candidate.Sha256, StringComparison.Ordinal))
                    throw new InvalidDataException("Empreinte SHA-256 différente : installation annulée.");
            }
            Process.Start(new ProcessStartInfo(target) { UseShellExecute = true });
            Close();
        }
        catch (Exception ex)
        {
            try { if (File.Exists(target)) File.Delete(target); } catch { /* best effort */ }
            if (_updateMessage is not null) _updateMessage.Text = "Installation impossible : " + ex.Message;
            if (_install is not null) _install.Enabled = true;
        }
    }
}
