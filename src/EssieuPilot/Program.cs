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
#if TEST_VERSION_6
    private const string TechnicalVersion = "1.7.0-test.6";
#elif TEST_VERSION_5
    private const string TechnicalVersion = "1.7.0-test.5";
#elif TEST_VERSION_4
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
    private static readonly Color Aqua = Color.FromArgb(0, 226, 232);
    private static readonly Color Orange = Color.FromArgb(255, 112, 15);
    private static readonly Color Green = Color.FromArgb(59, 244, 88);
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
        ClientSize = new Size(1238, 665);
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

        var top = Box(0, 0, 1238, 34, Surface, this);
        top.Controls.Add(LabelAt("▣  EssieuPilot", 17, 6, 470, 24, 10, Fore, true));

        var side = Box(0, 34, 240, 599, Color.FromArgb(10, 23, 40), this);
        side.Controls.Add(LabelAt("▰▰▰▰", 31, 22, 182, 43, 19, Fore, true));
        side.Controls.Add(LabelAt("Essieu", 28, 63, 120, 48, 23, Fore, true));
        side.Controls.Add(LabelAt("Pilot", 138, 63, 94, 48, 23, Orange, true));
        side.Controls.Add(LabelAt("━━━━━━", 37, 102, 170, 28, 13, Orange, true));
        side.Controls.Add(ActionButton("▥   Tableau de bord", 10, 154, 220, 46, ShowDashboard, true));
        side.Controls.Add(ActionButton("▱   Mes remorques", 10, 208, 220, 45, ShowProfiles));
        side.Controls.Add(ActionButton("☷   Commande", 10, 261, 220, 45, ShowControls));
        side.Controls.Add(ActionButton("▣   Simulation", 10, 314, 220, 45, ShowSimulation));
        side.Controls.Add(ActionButton("♢   Diagnostique", 10, 367, 220, 45, ShowDiagnostic));
        side.Controls.Add(ActionButton("↻   Mises à jour", 10, 420, 220, 45, ShowUpdates));
        side.Controls.Add(ActionButton("ⓘ   À propos", 10, 473, 220, 45, ShowAbout));
        side.Controls.Add(LabelAt("▰ EssieuPilot", 18, 543, 210, 31, 12, Orange, true));

        _body.SetBounds(240, 34, 998, 599);
        _body.BackColor = Night;
        Controls.Add(_body);
        _status.SetBounds(14, 636, 1215, 24);
        _status.Text = "Édition de test · Aucune connexion ni action physique dans ATS validée.";
        _status.ForeColor = Muted;
        _status.Font = new Font("Segoe UI", 9);
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
            BackColor = primary ? Color.FromArgb(12, 99, 119) : Color.FromArgb(12, 31, 49),
            ForeColor = primary ? Aqua : Fore,
            Font = new Font("Segoe UI", 10, FontStyle.Bold),
            Cursor = Cursors.Hand
        };
        b.FlatAppearance.BorderColor = primary ? Aqua : Color.FromArgb(53, 83, 103);
        b.FlatAppearance.BorderSize = 1;
        b.Click += (_, _) => click();
        return b;
    }

    private static void BorderPanel(Panel p, Color color)
    {
        p.Paint += (_, e) =>
        {
            using var pen = new Pen(color, 1.4f);
            e.Graphics.DrawRectangle(pen, 1, 1, p.Width - 3, p.Height - 3);
        };
    }

    private void ResetPage(string title, string description, string icon = "▥")
    {
        _body.Controls.Clear();
        _angleText = null;
        var banner = Box(0, 0, 998, 99, Surface, _body);
        banner.Controls.Add(LabelAt(icon, 24, 18, 57, 53, 27, Aqua, true));
        banner.Controls.Add(LabelAt(title, 83, 13, 855, 51, 25, Fore, true));
        banner.Controls.Add(LabelAt(description, 85, 60, 868, 27, 10, Muted));
        banner.Paint += (_, e) =>
        {
            using var p = new Pen(Color.FromArgb(25, 149, 170));
            e.Graphics.DrawLine(p, 0, 97, 998, 97);
        };
    }

    private Panel StyledCard(int x, int y, int w, int h, Control parent)
    {
        var panel = Box(x, y, w, h, Color.FromArgb(12, 33, 53), parent);
        BorderPanel(panel, Color.FromArgb(0, 167, 192));
        return panel;
    }

    private void PaintTrailer(Panel panel)
    {
        panel.Paint += (_, e) =>
        {
            var g = e.Graphics;
            g.SmoothingMode = System.Drawing.Drawing2D.SmoothingMode.AntiAlias;
            using var backdrop = new System.Drawing.Drawing2D.LinearGradientBrush(
                panel.ClientRectangle, Color.FromArgb(31, 50, 76),
                Color.FromArgb(124, 58, 47), 83f);
            g.FillRectangle(backdrop, panel.ClientRectangle);
            using var sky = new SolidBrush(Color.FromArgb(64, 15, 23, 38));
            g.FillRectangle(sky, 0, panel.Height / 2, panel.Width, panel.Height / 2);
            using var steel = new SolidBrush(Color.FromArgb(17, 29, 39));
            using var edge = new Pen(Orange, 3);
            using var accent = new Pen(Color.FromArgb(212, 112, 69), 2);
            int w = panel.Width, h = panel.Height;
            var body = new [] {
                new Point(23, h - 115), new Point(w - 100, h - 115),
                new Point(w - 63, h - 97), new Point(w - 20, h - 97),
                new Point(w - 20, h - 72), new Point(23, h - 72)
            };
            g.FillPolygon(steel, body);
            g.DrawLine(edge, 23, h - 115, w - 100, h - 115);
            g.DrawLine(accent, 23, h - 75, w - 22, h - 75);
            g.FillRectangle(steel, 28, h - 146, 75, 33);
            g.FillRectangle(steel, 35, h - 171, 48, 27);
            for(int i = 0; i < 6; i++)
            {
                int xx = w - 320 + i * 47;
                g.FillEllipse(Brushes.Black, xx, h - 93, 42, 42);
                using var rim = new Pen(Color.FromArgb(159, 178, 190), 5);
                g.DrawEllipse(rim, xx + 6, h - 87, 30, 30);
            }
            using var glow = new Pen(Aqua, 1);
            g.DrawLine(glow, 0, h - 27, w, h - 27);
            using var note = new Font("Segoe UI", 9, FontStyle.Bold);
            g.DrawString("VISUALISATION SCHÉMATIQUE · MODE TEST", note, Brushes.White, 20, 15);
        };
    }

    private void ShowDashboard()
    {
        ResetPage("Tableau de bord", "Contrôle et suivi de la direction de vos remorques", "▥");
        var trailer = StyledCard(15, 110, 711, 228, _body);
        PaintTrailer(trailer);
        var connection = StyledCard(739, 110, 244, 106, _body);
        connection.Controls.Add(LabelAt("État du jeu", 20, 13, 215, 32, 14, Fore, true));
        connection.Controls.Add(LabelAt("● Hors ligne", 34, 50, 215, 43, 17, Color.Orange, true));
        var enabled = StyledCard(739, 228, 244, 110, _body);
        enabled.Controls.Add(LabelAt("Essieux en jeu", 20, 12, 220, 31, 14, Fore, true));
        enabled.Controls.Add(LabelAt("● Inactifs", 34, 49, 190, 45, 18, Color.Orange, true));
        var axles = StyledCard(15, 349, 349, 110, _body);
        axles.Controls.Add(LabelAt("État des essieux", 15, 10, 320, 31, 14, Fore, true));
        axles.Controls.Add(LabelAt("● Simulation uniquement", 23, 50, 321, 49, 15, Aqua, true));
        var steer = StyledCard(375, 349, 608, 110, _body);
        steer.Controls.Add(LabelAt("Angle de braquage", 18, 10, 370, 31, 14, Fore, true));
        _angleText = LabelAt("0,0°", 32, 47, 250, 53, 25, Aqua, true);
        steer.Controls.Add(_angleText);
        steer.Controls.Add(LabelAt("A ←   D →    (simulation)", 291, 62, 300, 29, 11, Muted));
        var kpis = StyledCard(15, 472, 968, 112, _body);
        kpis.Controls.Add(LabelAt("Indicateurs clés", 18, 8, 480, 29, 13, Fore, true));
        for (int i = 1; i <= 3; i++)
        {
            int xx = 18 + (i - 1) * 243;
            using var dummy = new Pen(Color.Aqua);
            var item = Box(xx, 40, 230, 60, PanelColor, kpis);
            BorderPanel(item, Color.FromArgb(37, 84, 112));
            if (i == 1)
            {
                item.Controls.Add(LabelAt("Remorque active", 9, 5, 205, 22, 10, Muted));
                item.Controls.Add(LabelAt(_profile.Name, 9, 28, 208, 26, 12, Aqua, true));
            }
            if (i == 2)
            {
                item.Controls.Add(LabelAt("Essieux configurés", 9, 5, 205, 22, 10, Muted));
                item.Controls.Add(LabelAt(_profile.Axles.ToString(), 9, 28, 208, 26, 12, Aqua, true));
            }
            if (i == 3)
            {
                item.Controls.Add(LabelAt("Angle maximal", 9, 5, 205, 22, 10, Muted));
                item.Controls.Add(LabelAt("±" + _profile.MaxAngle + "°", 9, 28, 208, 26, 12, Aqua, true));
            }
        }
        var mode = Box(747, 40, 206, 60, PanelColor, kpis);
        BorderPanel(mode, Color.FromArgb(37, 84, 112));
        mode.Controls.Add(LabelAt("Mode de direction", 9, 5, 196, 21, 10, Muted));
        mode.Controls.Add(LabelAt("Simulation", 9, 27, 196, 28, 12, Aqua, true));
        _status.Text = "Aucune commande physique ATS validée. Les valeurs du tableau de bord sont locales.";
    }

    private void ShowControls()
    {
        ResetPage("Commande", "Configuration des touches et des paramètres de direction", "☷");
        var keys = StyledCard(15, 110, 458, 279, _body);
        keys.Controls.Add(LabelAt("Affectation des commandes", 17, 12, 422, 34, 15, Fore, true));
        string[] labels = { "Braquer à gauche", "Braquer à droite", "Centrer les essieux", "Mode de direction" };
        string[] values = { "A", "D", "S (aperçu)", "Manuel" };
        for(int i=0;i<4;i++)
        {
            keys.Controls.Add(LabelAt(labels[i], 18, 56 + i*48, 283, 30, 11, Fore));
            keys.Controls.Add(LabelAt(values[i], 332, 56 + i*48, 112, 30, 11, Aqua, true));
        }
        var cfg = StyledCard(486, 110, 497, 279, _body);
        cfg.Controls.Add(LabelAt("Paramètres de direction", 17, 12, 462, 34, 15, Fore, true));
        var lines = new [] {
            "Braquage : " + _profile.SteeringRate + "°/s",
            "Recentrage : " + _profile.ReturnRate + "°/s",
            "Angle maximal : ±" + _profile.MaxAngle + "°",
            "Sens de direction : " + (_profile.Inverted ? "Inversé" : "Normal")
        };
        for(int i=0;i<lines.Length;i++)
            cfg.Controls.Add(LabelAt(lines[i], 22, 58+i*49, 450, 30, 12, i==2?Aqua:Fore));
        var actions=StyledCard(15, 402, 968, 182, _body);
        actions.Controls.Add(LabelAt("Commandes en simulation", 20, 14, 916, 32, 15, Fore, true));
        actions.Controls.Add(ActionButton("◀ Tourner à gauche", 24, 66, 256, 55, () => { _left = true; _right = false; }));
        actions.Controls.Add(ActionButton("Centrer (avance)", 315, 66, 273, 55, () => { _forward = !_forward; _left = _right = false; }));
        actions.Controls.Add(ActionButton("Tourner à droite ▶", 625, 66, 300, 55, () => { _right = true; _left = false; }));
        actions.Controls.Add(LabelAt("Les commandes ne pilotent pas les roues du jeu. Régler les vitesses dans Mes remorques.", 24, 139, 921, 31, 10, Muted));
    }

    private void ShowSimulation()
    {
        ResetPage("Simulation", "Visualisation 2D / 3D du comportement des essieux", "▣");
        var illustration=StyledCard(15, 109, 676, 357, _body);
        illustration.Controls.Add(LabelAt("Vue schématique 2D de la remorque", 19, 15, 610, 32, 14, Fore, true));
        illustration.Paint += (_, e) =>
        {
            var g=e.Graphics;
            g.SmoothingMode=System.Drawing.Drawing2D.SmoothingMode.AntiAlias;
            using var pen=new Pen(Aqua,3);
            using var accent=new Pen(Orange,2);
            g.DrawRectangle(pen,90,125,475,105);
            g.DrawRectangle(pen,27,154,65,47);
            g.DrawLine(pen,92,170,130,170);
            for(int i=0;i<Math.Min(_profile.Axles,6);i++)
            {
                int x=365+i*33;
                g.DrawLine(accent,x,113,x,245);
            }
        };
        var info=StyledCard(705,109,278,357,_body);
        info.Controls.Add(LabelAt("Paramètres",18,14,245,31,14,Fore,true));
        info.Controls.Add(LabelAt("Remorque : "+_profile.Name,15,69,250,45,11,Aqua,true));
        info.Controls.Add(LabelAt("Essieux : "+_profile.Axles,15,124,250,31,12,Fore));
        info.Controls.Add(LabelAt("Angle max : ±"+_profile.MaxAngle+"°",15,171,250,31,12,Fore));
        info.Controls.Add(LabelAt("Vue 3D non validée",15,226,250,55,12,Color.Orange,true));
        var controls=StyledCard(15,479,968,105,_body);
        controls.Controls.Add(LabelAt("Simulation locale uniquement — A / D pour faire varier l’angle",20,13,915,32,14,Fore,true));
        controls.Controls.Add(LabelAt("Aucune déformation ni réaction physique des essieux d'ATS n'est simulée dans le moteur du jeu.",20,57,917,36,11,Muted));
        _status.Text="Simulation 2D illustrative ; rendu 3D et physique ATS à développer.";
    }

    private void ShowDiagnostic()
    {
        ResetPage("Diagnostique", "État de l'application et diagnostic de l'intégration ATS", "♢");
        var state=StyledCard(15,110,470,335,_body);
        state.Controls.Add(LabelAt("État du système",18,13,438,33,15,Fore,true));
        var data=new [] { "Application : prête", "Connexion ATS : hors ligne", "Remorque ATS : non détectée", "DLL ATS : non vérifiée", "Essieux physiques : inactifs" };
        for (int i=0;i<data.Length;i++)
            state.Controls.Add(LabelAt(data[i],20,63+i*48,430,31,12,i==0?Green:Muted));
        var log=StyledCard(499,110,484,335,_body);
        log.Controls.Add(LabelAt("Informations techniques",20,13,450,33,15,Fore,true));
        log.Controls.Add(LabelAt("Version test : "+TechnicalVersion,20,72,437,32,11,Aqua));
        log.Controls.Add(LabelAt("Profils chargés : "+_profiles.Count,20,120,438,32,11,Fore));
        log.Controls.Add(LabelAt("Profil : "+_profile.Name,20,168,438,32,11,Fore));
        log.Controls.Add(LabelAt("Aucune DLL n'est installée par cette édition.",20,223,438,74,11,Color.Orange));
        var foot=StyledCard(15,457,968,127,_body);
        foot.Controls.Add(LabelAt("Aucune donnée de télémétrie ATS en temps réel",20,21,922,37,15,Color.Orange,true));
        foot.Controls.Add(LabelAt("Les états réels apparaîtront uniquement après intégration et vérification de la DLL.",20,65,918,39,11,Muted));
        _status.Text="Diagnostic : connexion et braquage physique ATS non disponibles.";
    }

    private void ShowAbout()
    {
        ResetPage("À propos", "Informations sur l'application EssieuPilot", "ⓘ");
        var panel=StyledCard(15,111,968,472,_body);
        panel.Controls.Add(LabelAt("Essieu",32,32,214,62,34,Fore,true));
        panel.Controls.Add(LabelAt("Pilot",241,32,220,62,34,Orange,true));
        panel.Controls.Add(LabelAt("EssieuPilot — édition de test",36,110,890,42,18,Aqua,true));
        panel.Controls.Add(LabelAt("Version technique : "+TechnicalVersion,36,181,870,33,15,Fore,true));
        panel.Controls.Add(LabelAt("Configuration des profils de remorques et simulation locale des essieux directeurs.",36,239,886,70,13,Muted));
        panel.Controls.Add(LabelAt("American Truck Simulator — intégration physique non disponible dans cette édition.",36,312,896,54,12,Color.Orange));
        panel.Controls.Add(LabelAt("Nom de l'application : EssieuPilot",36,404,890,31,11,Aqua));
        _status.Text="EssieuPilot · habillage EssieuPilot (test).";
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
        ResetPage("Mes remorques", "Gérer vos profils et les paramètres des essieux", "▱");
        var left=StyledCard(15,110,320,474,_body);
        left.Controls.Add(LabelAt("Profils de remorque",13,10,291,38,15,Fore,true));
        var listing=new ListBox {
            Bounds=new Rectangle(14,60,290,285),
            BackColor=PanelColor,ForeColor=Fore,BorderStyle=BorderStyle.None,
            Font=new Font("Segoe UI",11)
        };
        listing.Items.AddRange(_profiles.Select(p=>p.Name).Cast<object>().ToArray());
        left.Controls.Add(listing);
        left.Controls.Add(ActionButton("+ Nouveau profil",13,362,292,42,()=>{
            var p=new TrailerProfile { Name="Remorque "+(_profiles.Count+1) };
            _profiles.Add(p); _profile=p; SaveProfiles(); ShowProfiles();
        },true));
        left.Controls.Add(ActionButton("Supprimer le profil",13,414,292,40,()=>{
            if(_profiles.Count<=1){
                MessageBox.Show("Impossible de supprimer le dernier profil.", "EssieuPilot"); return;
            }
            var selected=_profile;
            if(MessageBox.Show("Supprimer le profil « "+selected.Name+" » ?",
                "Confirmer la suppression",MessageBoxButtons.YesNo,MessageBoxIcon.Warning)!=DialogResult.Yes)return;
            _profiles.Remove(selected); _profile=_profiles[0]; SaveProfiles(); ShowProfiles();
        }));
        var right=StyledCard(348,110,635,474,_body);
        right.Controls.Add(LabelAt("Configuration de la remorque",19,11,589,36,15,Fore,true));
        right.Controls.Add(LabelAt("Nom du profil",20,59,235,29,11,Muted));
        var name=Input(20,90,581,_profile.Name,right);
        right.Controls.Add(LabelAt("Essieux",20,153,124,29,10,Muted));
        right.Controls.Add(LabelAt("Angle max.",163,153,126,29,10,Muted));
        right.Controls.Add(LabelAt("Braquage °/s",309,153,132,29,10,Muted));
        right.Controls.Add(LabelAt("Retour °/s",458,153,134,29,10,Muted));
        var axles=Number(20,186,120,_profile.Axles,1,30,right);
        var max=Number(163,186,120,_profile.MaxAngle,1,90,right);
        var turn=Number(309,186,120,_profile.SteeringRate,1,90,right);
        var center=Number(458,186,120,_profile.ReturnRate,1,90,right);
        var invert=new CheckBox {
            Text="Inverser le sens de braquage", Bounds=new Rectangle(20,255,467,36),
            Checked=_profile.Inverted,ForeColor=Fore,BackColor=right.BackColor
        };
        right.Controls.Add(invert);
        right.Controls.Add(ActionButton("Enregistrer les modifications",20,334,578,48,()=>{
            _profile.Name=string.IsNullOrWhiteSpace(name.Text)?"Remorque":name.Text.Trim();
            _profile.Axles=(int)axles.Value; _profile.MaxAngle=max.Value;
            _profile.SteeringRate=turn.Value; _profile.ReturnRate=center.Value;
            _profile.Inverted=invert.Checked; SaveProfiles();ShowProfiles();
            _status.Text="Profil sauvegardé dans le dossier de test.";
        },true));
        right.Controls.Add(LabelAt("Enregistrement local, sans action physique sur les essieux dans ATS.",20,413,582,37,10,Muted));
        listing.SelectedIndexChanged+=(_,_)=>{
            if(listing.SelectedIndex<0||listing.SelectedIndex>=_profiles.Count)return;
            _profile=_profiles[listing.SelectedIndex];
            name.Text=_profile.Name;axles.Value=_profile.Axles;max.Value=_profile.MaxAngle;
            turn.Value=_profile.SteeringRate;center.Value=_profile.ReturnRate;
            invert.Checked=_profile.Inverted;
        };
        listing.SelectedIndex=Math.Max(0,_profiles.IndexOf(_profile));
        _status.Text="Profils séparés de la version officielle. Supprimer exige confirmation.";
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
        ResetPage("Mises à jour", "Vérifier les publications de test et les modifications", "↻");
        var current=StyledCard(15,110,468,186,_body);
        current.Controls.Add(LabelAt("Version technique installée",20,12,422,35,14,Fore,true));
        current.Controls.Add(LabelAt(TechnicalVersion,20,62,422,55,24,Aqua,true));
        current.Controls.Add(LabelAt("Canal : édition de test indépendante",20,137,423,33,11,Muted));
        var next=StyledCard(499,110,484,186,_body);
        next.Controls.Add(LabelAt("Vérification des mises à jour",20,13,444,36,14,Fore,true));
        _updateMessage=LabelAt("Clique sur « Rechercher une mise à jour ».",20,63,447,105,12,Muted);
        next.Controls.Add(_updateMessage);
        var change=StyledCard(15,309,968,275,_body);
        change.Controls.Add(LabelAt("Centre des mises à jour — EssieuPilot",21,12,917,32,15,Fore,true));
        change.Controls.Add(LabelAt("• Nouvelle interface : 7 pages assorties aux couleurs bleu nuit, orange et turquoise",21,56,908,34,11,Fore));
        change.Controls.Add(LabelAt("• Profils sauvegardés et suppression avec confirmation",21,98,908,34,11,Fore));
        change.Controls.Add(LabelAt("• États ATS explicitement hors ligne lorsque la connexion n'est pas vérifiée",21,140,909,34,11,Fore));
        change.Controls.Add(ActionButton("Rechercher une mise à jour",21,203,435,51,async ()=>await CheckUpdates(),true));
        _install=ActionButton("Télécharger et installer",482,203,458,51,async ()=>await InstallUpdate());
        _install.Enabled=_update is not null; change.Controls.Add(_install);
        _status.Text="Mise à jour installée seulement après confirmation, SHA-256 vérifié.";
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
