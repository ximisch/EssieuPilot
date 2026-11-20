using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Drawing;
using System.Drawing.Drawing2D;
using System.IO;
using System.Linq;
using System.Windows.Forms;

namespace EssieuPilotNative;

internal sealed class StatusLamp : Control
{
    private Color _lampColor = Color.OrangeRed;
    public Color LampColor
    {
        get => _lampColor;
        set { _lampColor = value; Invalidate(); }
    }
    public StatusLamp()
    {
        Size = new Size(17, 17);
        DoubleBuffered = true;
        BackColor = Color.Transparent;
    }
    protected override void OnPaint(PaintEventArgs e)
    {
        base.OnPaint(e);
        e.Graphics.SmoothingMode = SmoothingMode.AntiAlias;
        using var outer = new SolidBrush(Color.FromArgb(50, LampColor));
        using var inner = new SolidBrush(LampColor);
        e.Graphics.FillEllipse(outer, 0, 0, 16, 16);
        e.Graphics.FillEllipse(inner, 4, 4, 8, 8);
    }
}

internal sealed class EssieuPilotWindow : Form
{
    private static readonly Color Night = Color.FromArgb(11, 22, 37);
    private static readonly Color Surface = Color.FromArgb(21, 37, 55);
    private static readonly Color SurfaceAlt = Color.FromArgb(26, 45, 63);
    private static readonly Color Border = Color.FromArgb(47, 77, 90);
    private static readonly Color Aqua = Color.FromArgb(48, 218, 208);
    private static readonly Color TextColor = Color.FromArgb(234, 244, 248);
    private static readonly Color SubText = Color.FromArgb(157, 185, 199);
    private static readonly Color Good = Color.FromArgb(49, 218, 111);
    private static readonly Color Bad = Color.FromArgb(244, 86, 85);

    private readonly Panel _body = new() { Dock = DockStyle.Fill, AutoScroll = true };
    private readonly ProfilesRepository _storage = new();
    private readonly NativeUpdateService _updates = new();
    private readonly List<TrailerProfile> _profiles;
    private TrailerProfile _active;
    private readonly StatusLamp _atsLamp = new();
    private readonly StatusLamp _steeringLamp = new();
    private readonly Label _atsLabel = new();
    private readonly Label _steeringLabel = new();
    private readonly Label _message = new();
    private readonly Label _updateHint = new();
    private Label? _dashboardAngle;
    private Label? _updatesMessage;
    private ProgressBar? _progress;
    private Button? _installButton;
    private UpdateCandidate? _candidate;
    private string _currentPage = "Tableau de bord";
    private readonly System.Windows.Forms.Timer _pollTimer = new() { Interval = 1000 };
    private readonly System.Windows.Forms.Timer _simTimer = new() { Interval = 50 };
    private DateTime _lastFrame = DateTime.UtcNow;
    private double _angle;
    private bool _simLeft, _simRight, _simMoving;
    private AtsSnapshot _lastSnapshot = new(null, false);

    public EssieuPilotWindow()
    {
        Text = "EssieuPilot";
        ClientSize = new Size(1200, 758);
        MinimumSize = new Size(1050, 690);
        StartPosition = FormStartPosition.CenterScreen;
        BackColor = Night;
        ForeColor = TextColor;
        Font = new Font("Segoe UI", 10);
        DoubleBuffered = true;
        KeyPreview = true;
        _profiles = _storage.Read();
        _active = _profiles[0];
        BuildShell();
        ShowPage("Tableau de bord");
        KeyDown += (_, e) => {
            if (e.KeyCode == Keys.A) _simLeft = true;
            if (e.KeyCode == Keys.D) _simRight = true;
        };
        KeyUp += (_, e) => {
            if (e.KeyCode == Keys.A) _simLeft = false;
            if (e.KeyCode == Keys.D) _simRight = false;
        };
        Deactivate += (_, _) => { _simLeft = false; _simRight = false; };
        _pollTimer.Tick += (_, _) => RefreshStatus();
        _pollTimer.Start();
        _simTimer.Tick += (_, _) => TickSimulation();
        _simTimer.Start();
        Shown += async (_, _) => {
            RefreshStatus();
            await CheckUpdates(silent: true);
        };
        FormClosed += (_, _) => {
            _pollTimer.Dispose();
            _simTimer.Dispose();
            _updates.Dispose();
        };
        AppInfo.Log("Native application opened: " + AppInfo.TechnicalVersion);
    }

    private void BuildShell()
    {
        var footer = new Panel { Dock = DockStyle.Bottom, Height = 36, BackColor = Surface };
        _message.AutoEllipsis = true;
        _message.ForeColor = SubText;
        _message.Dock = DockStyle.Fill;
        _message.TextAlign = ContentAlignment.MiddleLeft;
        _message.Padding = new Padding(24, 0, 0, 0);
        _message.Text = "Profils enregistrés localement. Le pilotage réel dans ATS doit encore être validé.";
        footer.Controls.Add(_message);

        var side = new Panel { Dock = DockStyle.Left, Width = 238, BackColor = Surface, Padding = new Padding(12) };
        var navTitle = Label("NAVIGATION", 12, Aqua, true);
        navTitle.SetBounds(19, 22, 195, 28);
        side.Controls.Add(navTitle);
        var nav = new[] {
            "Tableau de bord", "Profils de remorques", "Commandes",
            "Connexions", "Mises à jour", "Paramètres", "À propos"
        };
        for (int i = 0; i < nav.Length; i++)
        {
            string page = nav[i];
            var btn = Button(page, () => ShowPage(page));
            btn.SetBounds(8, 68 + i * 61, 208, 46);
            btn.TextAlign = ContentAlignment.MiddleLeft;
            btn.Padding = new Padding(12, 0, 0, 0);
            btn.BackColor = page == "Tableau de bord" ? Color.FromArgb(27, 84, 91) : SurfaceAlt;
            side.Controls.Add(btn);
        }
        var sideHelp = Label("ATS 1.62 · Edition d'essai native", 9, SubText);
        sideHelp.SetBounds(15, 536, 215, 45);
        sideHelp.Anchor = AnchorStyles.Bottom | AnchorStyles.Left;
        side.Controls.Add(sideHelp);
        _updateHint.ForeColor = Aqua;
        _updateHint.BackColor = Color.Transparent;
        _updateHint.Font = new Font("Segoe UI", 9, FontStyle.Bold);
        _updateHint.AutoEllipsis = true;
        _updateHint.SetBounds(15, 491, 211, 37);
        side.Controls.Add(_updateHint);

        var header = new Panel { Dock = DockStyle.Top, Height = 106, BackColor = Surface };
        var emblem = new Panel { Left = 23, Top = 27, Width = 53, Height = 46, BackColor = Color.FromArgb(21, 74, 81) };
        emblem.Paint += (_, e) => {
            e.Graphics.SmoothingMode = SmoothingMode.AntiAlias;
            using var pen = new Pen(Aqua, 3f);
            e.Graphics.DrawLine(pen, 5, 25, 23, 25);
            e.Graphics.DrawLine(pen, 22, 25, 30, 16);
            e.Graphics.DrawRectangle(pen, 30, 16, 17, 14);
            e.Graphics.DrawEllipse(pen, 9, 29, 8, 8);
            e.Graphics.DrawEllipse(pen, 35, 29, 8, 8);
        };
        header.Controls.Add(emblem);
        var name = Label("EssieuPilot", 27, Aqua, true);
        name.SetBounds(90, 16, 395, 45);
        header.Controls.Add(name);
        var tagline = Label("Pilotage et configuration des essieux directeurs", 10, SubText);
        tagline.SetBounds(93, 61, 610, 30);
        header.Controls.Add(tagline);

        var statusArea = new Panel { Width = 365, Height = 84, Top = 10, Anchor = AnchorStyles.Top | AnchorStyles.Right, BackColor = SurfaceAlt };
        statusArea.Left = ClientSize.Width - statusArea.Width - 19;
        var atsHeader = Label("CONNEXION ATS", 10, SubText, true);
        atsHeader.SetBounds(14, 6, 320, 22);
        statusArea.Controls.Add(atsHeader);
        _atsLamp.SetBounds(15, 30, 18, 18);
        statusArea.Controls.Add(_atsLamp);
        _atsLabel.SetBounds(39, 29, 315, 23);
        _atsLabel.ForeColor = TextColor;
        _atsLabel.Text = "ATS : vérification";
        statusArea.Controls.Add(_atsLabel);
        _steeringLamp.SetBounds(15, 55, 18, 18);
        statusArea.Controls.Add(_steeringLamp);
        _steeringLabel.SetBounds(39, 54, 315, 23);
        _steeringLabel.ForeColor = TextColor;
        _steeringLabel.Text = "Pilotage : non confirmé";
        statusArea.Controls.Add(_steeringLabel);
        header.Controls.Add(statusArea);
        header.Resize += (_, _) => statusArea.Left = header.ClientSize.Width - statusArea.Width - 19;

        _body.BackColor = Night;
        Controls.Add(_body);
        Controls.Add(side);
        Controls.Add(footer);
        Controls.Add(header);
    }

    private static Label Label(string text, float size = 11, Color? color = null, bool bold = false)
    {
        return new Label {
            Text = text,
            ForeColor = color ?? TextColor,
            BackColor = Color.Transparent,
            Font = new Font("Segoe UI", size, bold ? FontStyle.Bold : FontStyle.Regular),
            AutoEllipsis = false
        };
    }

    private static Button Button(string title, Action action, bool primary = false)
    {
        var b = new Button {
            Text = title,
            Font = new Font("Segoe UI", 10, FontStyle.Bold),
            FlatStyle = FlatStyle.Flat,
            ForeColor = primary ? Night : TextColor,
            BackColor = primary ? Aqua : SurfaceAlt,
            Cursor = Cursors.Hand
        };
        b.FlatAppearance.BorderColor = primary ? Aqua : Border;
        b.Click += (_, _) => action();
        return b;
    }

    private static Panel Card(Panel parent, int x, int y, int w, int h)
    {
        var p = new Panel {
            Bounds = new Rectangle(x, y, w, h),
            BackColor = Surface,
            BorderStyle = BorderStyle.FixedSingle
        };
        parent.Controls.Add(p);
        return p;
    }

    private static void Put(Control target, Control parent, int x, int y, int w, int h)
    {
        target.Bounds = new Rectangle(x, y, w, h);
        parent.Controls.Add(target);
    }

    private void ResetPage(string heading, string subtitle)
    {
        _body.SuspendLayout();
        _body.Controls.Clear();
        _dashboardAngle = null;
        _updatesMessage = null;
        _progress = null;
        _installButton = null;
        _body.AutoScrollMinSize = new Size(860, 585);
        Put(Label(heading, 22, Aqua, true), _body, 27, 24, 810, 47);
        Put(Label(subtitle, 10, SubText), _body, 29, 76, 820, 32);
        _body.ResumeLayout();
    }

    private void ShowPage(string page)
    {
        _currentPage = page;
        switch (page)
        {
            case "Tableau de bord": ShowDashboard(); break;
            case "Profils de remorques": ShowProfiles(); break;
            case "Commandes": ShowCommands(); break;
            case "Connexions": ShowConnections(); break;
            case "Mises à jour": ShowUpdates(); break;
            case "Paramètres": ShowSettings(); break;
            default: ShowAbout(); break;
        }
    }

    private void ShowDashboard()
    {
        ResetPage("Tableau de bord", "Vue d'ensemble des remorques, du pilotage et de la connexion à American Truck Simulator.");
        var status = Card(_body, 25, 121, 844, 133);
        Put(Label("ÉTAT DES CONNEXIONS", 13, Aqua, true), status, 24, 19, 550, 30);
        Put(Label("Les deux voyants restent visibles en haut à droite sur toutes les pages.", 11), status, 24, 60, 760, 31);
        Put(Label("Vert pilotage = retour récent de la DLL ; mouvement réel des roues non certifié.", 10, SubText), status, 24, 93, 770, 25);

        var profile = Card(_body, 25, 272, 400, 224);
        Put(Label("REMORQUE ACTIVE", 13, Aqua, true), profile, 19, 18, 355, 27);
        Put(Label(_active.Name, 17, TextColor, true), profile, 19, 55, 357, 35);
        Put(Label($"{_active.Axles} essieux  •  ±{_active.MaxAngle}° max", 12), profile, 19, 106, 358, 30);
        Put(Label($"Braquage {_active.SteeringRate}°/s · Retour {_active.ReturnRate}°/s", 11, SubText), profile, 19, 149, 362, 28);
        var edit = Button("Modifier mes profils", () => ShowPage("Profils de remorques"));
        Put(edit, profile, 19, 186, 359, 30);

        var sim = Card(_body, 441, 272, 428, 224);
        Put(Label("SIMULATION LOCALE", 13, Aqua, true), sim, 19, 18, 376, 26);
        _dashboardAngle = Label("0,0°", 32, Aqua, true);
        Put(_dashboardAngle, sim, 19, 56, 375, 71);
        Put(Label("Maintenir A ou D dans EssieuPilot pour simuler l'angle.", 10, SubText), sim, 19, 131, 391, 29);
        var moving = new CheckBox {
            Text = "Le véhicule avance (recentrage simulé)",
            Checked = _simMoving,
            ForeColor = TextColor,
            BackColor = Surface,
            AutoSize = false
        };
        moving.CheckedChanged += (_, _) => _simMoving = moving.Checked;
        Put(moving, sim, 19, 178, 391, 32);
        _message.Text = "La simulation n'agit pas sur les roues d'ATS.";
    }

    private void ShowConnections()
    {
        ResetPage("Connexions", "Surveillance automatique du jeu et des retours du module de direction, sans fenêtre annexe.");
        var c = Card(_body, 25, 123, 844, 310);
        Put(Label("ATS — PROCESSUS DU JEU", 13, Aqua, true), c, 23, 21, 680, 30);
        Put(Label("Voyant vert : amtrucks.exe détecté sous Windows.", 11), c, 23, 59, 760, 34);
        Put(Label("PILOTAGE — CONFIRMATION DE LA DLL", 13, Aqua, true), c, 23, 113, 760, 30);
        Put(Label("Voyant vert : commande acquittée récemment par EssieuPilotATS.dll.", 11), c, 23, 151, 785, 29);
        Put(Label("Cela ne prouve pas encore que les roues braquent physiquement dans ATS.", 11, SubText), c, 23, 192, 780, 28);
        Put(Label("Sans DLL, le deuxième voyant reste rouge. Aucun fichier ATS n'est installé par cette application.", 10, SubText), c, 23, 238, 795, 40);
        var refresh = Button("Actualiser le diagnostic", RefreshStatus);
        Put(refresh, _body, 25, 452, 304, 45);
        var diagnostic = Button("Ouvrir le journal", () => {
            if (File.Exists(AppInfo.DiagnosticPath))
                Process.Start(new ProcessStartInfo(AppInfo.DiagnosticPath) { UseShellExecute = true });
            else MessageBox.Show("Aucun journal disponible.", AppInfo.Name);
        });
        Put(diagnostic, _body, 347, 452, 289, 45);
    }

    private void ShowProfiles()
    {
        ResetPage("Profils de remorques", "Crée tes configurations ; chaque remorque possède ses propres réglages.");
        var c = Card(_body, 25, 121, 844, 427);
        var selection = new ListBox {
            BackColor = SurfaceAlt,
            ForeColor = TextColor,
            Font = new Font("Segoe UI", 10),
            BorderStyle = BorderStyle.FixedSingle
        };
        Put(selection, c, 17, 26, 230, 319);
        selection.Items.AddRange(_profiles.Select(p => p.Name).Cast<object>().ToArray());
        var name = new TextBox { BackColor = SurfaceAlt, ForeColor = TextColor, BorderStyle = BorderStyle.FixedSingle, Text = _active.Name };
        Put(Label("Nom de la remorque", 10, SubText), c, 267, 28, 500, 25);
        Put(name, c, 267, 56, 525, 34);
        Put(Label("Essieux", 10, SubText), c, 267, 108, 102, 25);
        Put(Label("Angle max.", 10, SubText), c, 397, 108, 110, 25);
        Put(Label("Braquage °/s", 10, SubText), c, 527, 108, 120, 25);
        Put(Label("Retour °/s", 10, SubText), c, 660, 108, 119, 25);
        var axles = Numeric(1, 30, _active.Axles);
        var max = Numeric(1, 90, _active.MaxAngle);
        var turn = Numeric(1, 90, _active.SteeringRate);
        var center = Numeric(1, 90, _active.ReturnRate);
        Put(axles, c, 267, 137, 101, 33);
        Put(max, c, 397, 137, 102, 33);
        Put(turn, c, 527, 137, 105, 33);
        Put(center, c, 660, 137, 106, 33);
        var inverse = new CheckBox {
            Text = "Inverser le sens",
            ForeColor = TextColor, BackColor = Surface,
            Checked = _active.Inverted
        };
        Put(inverse, c, 267, 199, 325, 28);
        selection.SelectedIndexChanged += (_, _) => {
            if (selection.SelectedIndex < 0 || selection.SelectedIndex >= _profiles.Count) return;
            _active = _profiles[selection.SelectedIndex];
            name.Text = _active.Name;
            axles.Value = Math.Clamp(_active.Axles, 1, 30);
            max.Value = Math.Clamp(_active.MaxAngle, 1, 90);
            turn.Value = Math.Clamp(_active.SteeringRate, 1, 90);
            center.Value = Math.Clamp(_active.ReturnRate, 1, 90);
            inverse.Checked = _active.Inverted;
        };
        var save = Button("Enregistrer", () => {
            _active.Name = string.IsNullOrWhiteSpace(name.Text) ? "Remorque" : name.Text.Trim();
            _active.Axles = (int)axles.Value;
            _active.MaxAngle = max.Value;
            _active.SteeringRate = turn.Value;
            _active.ReturnRate = center.Value;
            _active.Inverted = inverse.Checked;
            try { _storage.Save(_profiles); _message.Text = "Profil enregistré localement."; ShowProfiles(); }
            catch (Exception ex) { MessageBox.Show(ex.Message, AppInfo.Name); }
        }, true);
        Put(save, c, 267, 270, 242, 45);
        var add = Button("Ajouter", () => {
            _active = new TrailerProfile { Name = "Remorque " + (_profiles.Count + 1) };
            _profiles.Add(_active);
            try { _storage.Save(_profiles); ShowProfiles(); }
            catch (Exception ex) { MessageBox.Show(ex.Message, AppInfo.Name); }
        });
        Put(add, c, 524, 270, 268, 45);
        var remove = Button("Supprimer le profil", () => {
            if (_profiles.Count <= 1) { MessageBox.Show("Conserve au moins un profil.", AppInfo.Name); return; }
            if (MessageBox.Show("Supprimer ce profil ?", AppInfo.Name, MessageBoxButtons.YesNo,
                MessageBoxIcon.Question) != DialogResult.Yes) return;
            _profiles.Remove(_active);
            _active = _profiles[0];
            try { _storage.Save(_profiles); ShowProfiles(); }
            catch (Exception ex) { MessageBox.Show(ex.Message, AppInfo.Name); }
        });
        Put(remove, c, 17, 356, 230, 44);
        Put(Label("Les profils de cette application sont séparés des anciennes versions.", 10, SubText), c, 267, 362, 525, 32);
        selection.SelectedIndex = Math.Max(0, _profiles.IndexOf(_active));
    }

    private static NumericUpDown Numeric(int min, int max, decimal value) =>
        new() {
            Minimum = min, Maximum = max, Value = Math.Clamp(value, min, max),
            BackColor = SurfaceAlt, ForeColor = TextColor,
            BorderStyle = BorderStyle.FixedSingle, DecimalPlaces = 0
        };

    private void ShowCommands()
    {
        ResetPage("Commandes", "Préparation des commandes : configuration et simulation, sans braquage réel pour l'instant.");
        var c = Card(_body, 25, 122, 844, 351);
        Put(Label("COMMANDES DE RÉFÉRENCE", 14, Aqua, true), c, 25, 21, 780, 30);
        Put(Label("F8    Activer / désactiver le pilotage (module expérimental)", 12), c, 27, 78, 771, 31);
        Put(Label("F9    Braquer vers la gauche", 12), c, 27, 127, 771, 30);
        Put(Label("F10  Braquer vers la droite", 12), c, 27, 176, 771, 30);
        Put(Label("Simulation dans cette application : touches A et D.", 11, SubText), c, 27, 226, 772, 32);
        Put(Label("Support clavier/manette/joystick et commandes physiques ATS : intégration à valider séparément.",
            10, SubText), c, 27, 274, 770, 45);
        _message.Text = "Les raccourcis affichés ne garantissent pas le contrôle dans ATS.";
    }

    private async System.Threading.Tasks.Task CheckUpdates(bool silent)
    {
        try
        {
            if (!silent && _updatesMessage != null) _updatesMessage.Text = "Vérification sur GitHub…";
            var candidate = await _updates.CheckAsync();
            _candidate = candidate;
            _updateHint.Text = candidate == null ? "" : "Mise à jour disponible : " + candidate.Tag;
            if (_currentPage == "Mises à jour" && _updatesMessage != null)
                _updatesMessage.Text = candidate == null ?
                    "EssieuPilot est déjà à jour sur le canal d'essai natif." :
                    "Version disponible : " + candidate.Tag + " — téléchargement intégré prêt.";
            if (_installButton != null) _installButton.Enabled = candidate != null;
        }
        catch (Exception ex)
        {
            if (!silent && _updatesMessage != null)
                _updatesMessage.Text = "Vérification impossible : " + ex.Message;
            AppInfo.Log("Update check failed: " + ex);
        }
    }

    private void ShowUpdates()
    {
        ResetPage("Mises à jour", "Vérification, téléchargement et installation depuis cette fenêtre, avec ton accord.");
        var c = Card(_body, 25, 123, 844, 344);
        Put(Label("Version installée : " + AppInfo.TechnicalVersion, 14, Aqua, true), c, 24, 23, 795, 34);
        _updatesMessage = Label(_candidate == null ?
            "Clique sur « Vérifier » pour rechercher une nouvelle version." :
            "Disponible : " + _candidate.Tag, 12);
        Put(_updatesMessage, c, 24, 90, 785, 62);
        var check = Button("Vérifier les mises à jour", () => _ = CheckUpdates(silent: false));
        Put(check, c, 24, 174, 324, 46);
        _installButton = Button("Télécharger et installer", () => _ = InstallUpdate(), true);
        _installButton.Enabled = _candidate != null;
        Put(_installButton, c, 365, 174, 426, 46);
        _progress = new ProgressBar { Minimum = 0, Maximum = 100, Value = 0 };
        Put(_progress, c, 24, 238, 767, 20);
        Put(Label("Téléchargements vérifiés par SHA-256. Aucune installation sans confirmation.",
            10, SubText), c, 24, 281, 779, 44);
        var note = Label("Canal d'essai natif — les versions 1.6.3 ne sont pas remplacées automatiquement.", 10, SubText);
        Put(note, _body, 30, 497, 805, 36);
    }

    private async System.Threading.Tasks.Task InstallUpdate()
    {
        if (_candidate == null) return;
        var selected = _candidate;
        if (MessageBox.Show(
            "Installer " + selected.Tag + " ?\nLe nouvel installateur sera téléchargé et vérifié avant son ouverture.",
            AppInfo.Name + " — mise à jour",
            MessageBoxButtons.YesNo, MessageBoxIcon.Question) != DialogResult.Yes) return;
        if (_installButton != null) _installButton.Enabled = false;
        try
        {
            var progress = new Progress<int>(p => { if (_progress != null) _progress.Value = Math.Clamp(p, 0, 100); });
            if (_updatesMessage != null) _updatesMessage.Text = "Téléchargement sécurisé en cours…";
            string file = await _updates.DownloadVerifiedInstallerAsync(selected, progress);
            if (_updatesMessage != null) _updatesMessage.Text = "Fichier contrôlé. Ouverture de l'installateur…";
            Process.Start(new ProcessStartInfo(file) { UseShellExecute = true });
            Close();
        }
        catch (Exception ex)
        {
            AppInfo.Log("Update install failed: " + ex);
            if (_updatesMessage != null) _updatesMessage.Text = "Erreur : " + ex.Message;
            if (_installButton != null) _installButton.Enabled = true;
        }
    }

    private void ShowSettings()
    {
        ResetPage("Paramètres", "Stockage local, diagnostics et fonctionnement du logiciel.");
        var c = Card(_body, 25, 121, 844, 295);
        Put(Label("DOSSIER DES PROFILS", 13, Aqua, true), c, 23, 22, 730, 29);
        Put(Label(AppInfo.ProfilesPath, 11, SubText), c, 23, 72, 775, 36);
        Put(Label("Les anciens profils EssieuPilot ne sont ni effacés ni écrasés.", 11), c, 23, 117, 773, 34);
        var directory = Button("Ouvrir mon dossier EssieuPilot", () => {
            Directory.CreateDirectory(AppInfo.SettingsDirectory);
            Process.Start(new ProcessStartInfo(AppInfo.SettingsDirectory) { UseShellExecute = true });
        });
        Put(directory, c, 23, 187, 425, 48);
        Put(Label("Les mises à jour sont proposées, jamais installées sans ton accord.", 10, SubText),
            c, 23, 248, 780, 28);
    }

    private void ShowAbout()
    {
        ResetPage("À propos", "Informations techniques et limites connues.");
        var c = Card(_body, 25, 125, 844, 293);
        Put(Label("EssieuPilot", 23, Aqua, true), c, 26, 21, 730, 42);
        Put(Label("Application Windows native — version " + AppInfo.TechnicalVersion, 12), c, 26, 80, 770, 32);
        Put(Label("Détection d'ATS par amtrucks.exe ; accusé DLL sous 1,5 seconde.", 11), c, 26, 134, 775, 33);
        Put(Label("Aucun braquage réel dans ATS ni compatibilité Convoi confirmé pour cette édition.", 11, SubText),
            c, 26, 183, 781, 57);
        Put(Label("Code du projet : github.com/ximisch/EssieuPilot", 10, SubText), c, 26, 254, 770, 27);
    }

    private void RefreshStatus()
    {
        _lastSnapshot = AtsStatusReader.Read();
        _atsLamp.LampColor = _lastSnapshot.GameRunning == true ? Good : Bad;
        _atsLabel.Text = _lastSnapshot.GameRunning switch {
            true => "ATS : EN LIGNE",
            false => "ATS : HORS LIGNE",
            _ => "ATS : ÉTAT INDISPONIBLE"
        };
        _steeringLamp.LampColor = _lastSnapshot.SteeringAcknowledged ? Good : Bad;
        _steeringLabel.Text = _lastSnapshot.SteeringAcknowledged ?
            "Pilotage : retour DLL actif" : "Pilotage : non confirmé";
    }

    private void TickSimulation()
    {
        var now = DateTime.UtcNow;
        double dt = Math.Clamp((now - _lastFrame).TotalSeconds, 0, 0.15);
        _lastFrame = now;
        if (_simMoving)
        {
            double d = (double)_active.ReturnRate * dt;
            _angle = Math.Abs(_angle) <= d ? 0 : _angle - Math.Sign(_angle) * d;
        }
        else
        {
            int dir = (_simRight ? 1 : 0) - (_simLeft ? 1 : 0);
            if (_active.Inverted) dir *= -1;
            _angle = Math.Clamp(_angle + dir * (double)_active.SteeringRate * dt,
                -(double)_active.MaxAngle, (double)_active.MaxAngle);
        }
        if (_dashboardAngle != null)
            _dashboardAngle.Text = _angle.ToString("0.0") + "°";
    }
}
