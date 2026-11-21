'use strict';
// Exact EssieuPilot 1.6.2 design: ONE ATS lamp in the existing top quick-status card.
// The original telemetry-only bottom tile and the experimental pilot tile are removed.
// Steering will get its own tab only after future real in-game testing.
(function () {
  const aside = document.querySelector('.side-bottom');
  if (aside) {
    aside.querySelector('.connection')?.remove();
    aside.querySelector('#originalPhysicalPilotStatus')?.remove();
  }
  const quick = document.querySelector('.modern-side-quick');
  const lamp = quick?.querySelector('.modern-small-dot');
  const title = quick?.querySelector('strong');
  const detail = quick?.querySelector('small');
  const gameStat = document.getElementById('modernGameState');
  if (!quick || !lamp || !title || !detail) return;

  function setLamp(color) {
    lamp.style.background = color;
    lamp.style.boxShadow = '0 0 0 4px ' + color + '24';
  }
  function showStatus(running) {
    if (running === true) {
      setLamp('#33db83');
      title.textContent = 'ATS · EN LIGNE';
      detail.textContent = 'American Truck Simulator détecté';
    } else if (running === false) {
      setLamp('#f18a5b');
      title.textContent = 'ATS · HORS LIGNE';
      detail.textContent = 'Jeu fermé ou non détecté';
    } else {
      setLamp('#e1ab6b');
      title.textContent = 'ATS · ÉTAT INDISPONIBLE';
      detail.textContent = 'Vérification du jeu impossible';
    }
    if (gameStat) {
      gameStat.textContent = running === true ? '● ATS EN LIGNE' :
        running === false ? '● ATS HORS LIGNE' : '● ÉTAT INDISPONIBLE';
    }
  }
  let busy = false;
  async function refresh() {
    if (busy) return;
    busy = true;
    try {
      const response = await fetch('/api/status', {cache: 'no-store'});
      if (!response.ok) throw new Error('Status service unavailable');
      const data = await response.json();
      if (typeof data.game_running !== 'boolean')
        throw new Error('Invalid game status payload');
      showStatus(data.game_running);
    } catch (_) {
      // Read failures are unknown, not evidence that ATS has stopped.
      showStatus(null);
    } finally {
      busy = false;
    }
  }
  // Synchronize the existing original UI "Activer le braquage manuel" button
  // with the controller's SAFE test-mode state. This is not a physical command.
  // The original 1.6.2 app still handles local simulation and its appearance.
  let uiToggleNotice = '';
  let syncingManualFromBackend = false;
  let localManualPendingUntil = 0;
  document.addEventListener('click', (event) => {
    const control = event.target.closest?.('[data-action="toggle-manual"]');
    if (!control || syncingManualFromBackend) return;
    // Give the existing button's API request time to commit before syncing
    // from the authoritative game-input controller.
    localManualPendingUntil = Date.now() + 1500;
    // The original app click handler updates the button's .enabled class
    // synchronously; wait until it completes before reading the new state.
    setTimeout(async () => {
      const toggle = document.getElementById('toggleManual');
      const enabled = !!toggle?.classList.contains('enabled');
      try {
        const response = await fetch('/api/control-link/mode', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-ATS-Token': document.body.dataset.token
          },
          body: JSON.stringify({enabled})
        });
        const result = await response.json();
        uiToggleNotice = response.ok ? 'Bouton de l’interface synchronisé avec le contrôleur de test.' :
          'Bouton de simulation uniquement : ' + (result.error || 'contrôleur non disponible');
      } catch (error) {
        uiToggleNotice = 'Le bouton de simulation n’a pas pu joindre le contrôleur : ' + error.message;
      }
    }, 0);
  }, true);

  // Stage 1 is displayed in the existing Diagnostic tab, not a new Pilotage tab.
  const diag = document.querySelector('.modern-diagnostic');
  if (diag) {
    // Original.12: download the DURABLE last-in-game report. It survives
    // Alt+Tab and stopping ATS and cannot enable any physical steering.
    const physicalReportRow = document.createElement('div');
    physicalReportRow.className = 'modern-diagnostic-row';
    const reportText = document.createElement('div');
    const reportHeading = document.createElement('b');
    reportHeading.textContent = 'Original.12 · Diagnostic automatique du braquage physique';
    const reportHelp = document.createElement('p');
    reportHelp.textContent = 'Effectue tes essais avec ATS au premier plan, puis télécharge ici le dernier rapport. Il reste disponible après avoir quitté le jeu. Chemin local : %LOCALAPPDATA%\\EssieuPilot\\ATS-Physical-Test\\diagnostic-original12.json. Aucun braquage réel n’est encore validé.';
    const reportLink = document.createElement('a');
    reportLink.href = '/api/physical-diagnostic';
    reportLink.download = 'EssieuPilot_Diagnostic_Physique_Original12.json';
    reportLink.textContent = 'Télécharger le diagnostic physique (JSON)';
    reportLink.style.textDecoration = 'underline';
    reportLink.style.fontWeight = 'bold';
    reportText.append(reportHeading, reportHelp, reportLink);
    physicalReportRow.appendChild(reportText);
    diag.appendChild(physicalReportRow);
    // Original.9 : identifie la remorque depuis la télémétrie SCS,
    // puis retrouve le profil associé sans piloter ni modifier le jeu.
    const autoRow = document.createElement('div');
    autoRow.className = 'modern-diagnostic-row';
    autoRow.innerHTML = '<div><b>Remorque et profil reconnus automatiquement (Original.9)</b>' +
      '<p id="epTrailerDetectionText">Vérification de la télémétrie ATS…</p></div>' +
      '<span id="epTrailerDetectionBadge" class="modern-pending-status">LECTURE</span>';
    diag.appendChild(autoRow);
    let detectingTrailer = false;
    async function refreshTrailerDetection() {
      if (detectingTrailer) return;
      detectingTrailer = true;
      const output = document.getElementById('epTrailerDetectionText');
      const badge = document.getElementById('epTrailerDetectionBadge');
      try {
        const [telemetryRes, profilesRes] = await Promise.all([
          fetch('/api/telemetry', {cache:'no-store'}),
          fetch('/api/config', {cache:'no-store'})
        ]);
        if (!telemetryRes.ok || !profilesRes.ok) throw new Error('Service de lecture indisponible');
        const telemetry = await telemetryRes.json();
        const config = await profilesRes.json();
        const trailer = telemetry.connected && Array.isArray(telemetry.trailers) ?
          telemetry.trailers[0] : null;
        if (!trailer || !trailer.id) {
          output.textContent = 'Aucune remorque reconnue : ' +
            (telemetry.note || 'ATS et le SCS Telemetry Hub doivent être actifs.') +
            ' Aucun braquage physique envoyé.';
          badge.textContent = 'NON DÉTECTÉE';
        } else {
          const id = String(trailer.id).toLowerCase();
          const profiles = Array.isArray(config.profiles) ? config.profiles : [];
          const selected = profiles.find(p =>
            String(p.match_key || '').toLowerCase() === id && String(p.match_key || '') !== '');
          const fallback = profiles.find(p => p.id === config.active_id &&
            !String(p.match_key || '').trim());
          const axleCount = Number(trailer.estimated_axles || 0);
          const axleDescription = axleCount > 0 ?
            axleCount + ' essieux estimés (direction non connue)' : 'nombre d’essieux inconnu';
          const name = String(trailer.name || trailer.id);
          output.textContent = 'Remorque : ' + name + ' (' + trailer.id + ') · ' + axleDescription +
            '\n' + (selected ? 'Profil associé : ' + String(selected.name || selected.id) :
              fallback ? 'Profil par défaut : ' + String(fallback.name || fallback.id) +
                ' (non associé spécialement)' :
              'Aucun profil associé. Utilise Remorques → Associer la remorque détectée à ce profil.') +
            '\nTélémétrie : lecture uniquement ; aucun essieu piloté physiquement.';
          output.style.whiteSpace = 'pre-line';
          badge.textContent = selected ? 'PROFIL RECONNU' : 'REMORQUE DÉTECTÉE';
        }
      } catch (error) {
        output.textContent = 'Détection non disponible : ' + error.message;
        badge.textContent = 'INDISPONIBLE';
      } finally {
        detectingTrailer = false;
      }
    }
    refreshTrailerDetection();
    setInterval(refreshTrailerDetection, 2000);

    const row = document.createElement('div');
    row.className = 'modern-diagnostic-row';
    row.innerHTML = '<div><b>Touches configurables → commandes et DLL ATS (Original.12 - diagnostic)</b>' +
      '<p id="epInputBridgeText">Chargement du profil enregistré…</p></div>' +
      '<span id="epInputBridgeBadge" class="modern-pending-status">VÉRIFICATION</span>';
    diag.appendChild(row);
    let checking = false;
    async function checkBindings() {
      if (checking) return;
      checking = true;
      const text = document.getElementById('epInputBridgeText');
      const badge = document.getElementById('epInputBridgeBadge');
      try {
        const res = await fetch('/api/control-link', {cache:'no-store'});
        if (!res.ok) throw new Error('Aucune réponse du contrôleur');
        const c = await res.json();
        // Original 2D/3D app stores `manual` internally. Trigger its own
        // existing button (without a second API request) to keep it aligned
        // with the ATS-side state when F8/joystick toggled while ATS focused.
        const manualButton = document.getElementById('toggleManual');
        if (manualButton && Date.now() >= localManualPendingUntil &&
            manualButton.classList.contains('enabled') !== !!c.manual_armed) {
          syncingManualFromBackend = true;
          try { manualButton.click(); } finally { syncingManualFromBackend = false; }
        }
        if (c.dll_confirmed)
          throw new Error('Aucun test de braquage physique n’a encore été validé dans ATS');
        const name = c.profile_name || 'Aucun profil enregistré';
        const keys = [
          'Activation : ' + (c.toggle_binding || 'Non définie'),
          'Gauche : ' + (c.left_binding || 'Non définie'),
          'Droite : ' + (c.right_binding || 'Non définie')
        ].join(' · ');
        // Persistent mode state and raw press counters are retained after
        // switching back to the EssieuPilot Diagnostic window.
        const armed = c.manual_armed ? 'ACTIVÉ (test)' : 'DÉSACTIVÉ';
        const focus = c.ats_foreground ? 'ATS au premier plan : entrée du jeu autorisée' :
          c.ats_running ? 'ATS lancé, mais fenêtre ATS non active : commande suspendue' : 'ATS non lancé';
        const windowsInfo = 'Détection Windows : ' + (c.input_reason || 'Non disponible') +
          ' · PID actif ' + (c.foreground_pid || '—') +
          ' / PID ATS ' + (c.ats_process_pid || '—') +
          ' · codes clavier gauche=' + (c.left_virtual_key || '—') +
          ', droite=' + (c.right_virtual_key || '—');
        const counts = 'Activation : ' + (c.activation_count || 0) +
          ' · Gauche : ' + (c.left_count || 0) +
          ' · Droite : ' + (c.right_count || 0);
        const allowed = c.steering_input_allowed ?
          'AUTORISÉ (diagnostic uniquement)' : 'BLOQUÉ (INACTIF ou hors ATS)';
        const held = c.held_direction === 'GAUCHE' || c.held_direction === 'DROITE' ?
          c.held_direction + ' · ' + (c.requested_source || 'Commande') +
            ' · ' + Math.abs(Number(c.requested_percent || 0)).toFixed(1) + '% (maintenu)' :
          'AUCUNE (relâchée, inactive ou hors ATS)';
        const admitted = 'Gauche : ' + (c.allowed_left_count || 0) +
          ' · Droite : ' + (c.allowed_right_count || 0);
        const analog = 'Axe analogique : détecté ' +
          Number(c.analog_detected_percent || 0).toFixed(1) +
          ' % · autorisé ' + Number(c.analog_authorized_percent || 0).toFixed(1) +
          ' % · changements autorisés gauche=' + (c.analog_left_count || 0) +
          ', droite=' + (c.analog_right_count || 0);
        const dllProbe = 'CHARGEMENT DLL ATS : ' +
          (c.physical_dll_loaded ?
            ('CONFIRMÉ · version signalée ' + (c.physical_dll_game_version || '?') +
             (c.physical_dll_sdk_accepted ? ' · SDK ATS reconnu' : ' · VERSION NON RECONNUE')) :
            'NON CONFIRMÉ') +
          ' · ' + (c.physical_dll_probe_message || 'Aucun diagnostic disponible');
        const experimental = 'ENVOI PHYSIQUE EXPÉRIMENTAL : ' +
          (!c.physical_experiment_opt_in ? 'DÉSACTIVÉ par défaut' :
           c.physical_packet_sent ?
             ('PAQUET ENVOYÉ (' + Number(c.physical_packet_normalized || 0).toFixed(3) +
             ') · retour de la DLL : ' + (c.physical_callback_ack ?
               ('callback exécuté, age ' + (c.physical_callback_ack_age_ms || 0) + ' ms') :
               'NON CONFIRMÉ')) :
             'AUTORISATION ABSENTE : vérifier ACTIF, ATS au premier plan, télémétrie, remorque et DLL');
        const physical = 'BRAQUAGE RÉEL : NON VALIDÉ · ' +
          (c.physical_validation_status || 'Aucune observation physique ATS');
        const plan = c.planned_angle_valid ?
          'Angle prévu HORS JEU : ' + Number(c.planned_angle_degrees || 0).toFixed(2) +
            '° · ' + (c.planned_angle_reason || 'Non appliqué') :
          'Angle prévu : indisponible — ' + (c.planned_angle_reason || 'ATS hors jeu');
        const speed = c.observed_speed_valid ?
          'Vitesse ATS mesurée : ' + Number(c.observed_speed_kmh || 0).toFixed(1) + ' km/h' :
          'Vitesse ATS indisponible (pas de recentrage calculé)';
        const actualWheels = c.observed_wheel_steering_valid &&
          Array.isArray(c.observed_wheel_steering_degrees) ?
          'ANGLES MESURÉS PAR ATS (roues de remorque) : ' +
          c.observed_wheel_steering_degrees.slice(0,20).map(n=>Number(n).toFixed(1)+'°').join(' / ') +
          ' · lecture SDK, pas preuve visuelle' :
          'ANGLES MESURÉS PAR ATS : données de direction des roues indisponibles';
        const lastAccepted = c.last_authorized_command ?
          c.last_authorized_command + (c.last_authorized_time ?
            ' à ' + c.last_authorized_time.slice(11,19) : '') :
          'Aucune (mode ACTIF requis)';
        const recent = Array.isArray(c.input_history) && c.input_history.length ?
          c.input_history.slice(-5).join(' / ') : 'Aucun appui reconnu';
        const last = c.last_input ? 'Dernière commande : ' + c.last_input +
          (c.last_event_time ? ' à ' + c.last_event_time.slice(11,19) : '') :
          'Aucune commande enregistrée';
        text.style.whiteSpace = 'pre-line';
        text.textContent = 'Profil : ' + name + '\nTouches : ' + keys +
          '\nMode manuel : ' + armed + '  |  ' + focus +
          '\n' + windowsInfo +
          '\nCOMMANDES AUTORISÉES (ACTIF uniquement) : ' + admitted +
          '\nAutorisation : ' + allowed +
          '\nDirection actuelle : ' + held +
          '\n' + analog +
          '\nDernière commande autorisée : ' + lastAccepted +
          '\n' + plan + '\n' + speed + '\n' + actualWheels + '\n' + dllProbe + '\n' + experimental + '\n' + physical +
          '\nAppuis bruts détectés (y compris INACTIF, diagnostic seul) : ' + counts +
          '\n' + last + '\nHistorique : ' + recent +
          '\nATTENTION : mode R&D, usage expérimental non validé : un ACK indique seulement un appel DLL, jamais la preuve du mouvement des roues.' +
          (uiToggleNotice ? '\n' + uiToggleNotice : '');
        badge.textContent = !c.service_running || !c.ipc_ready || !c.profile_saved ?
          'À VÉRIFIER' : c.manual_armed ? 'ACTIVÉ · TEST' : 'INACTIF · TEST';
        badge.className = 'modern-pending-status';
      } catch (error) {
        text.textContent = 'Test des touches indisponible : ' + error.message;
        badge.textContent = 'INDISPONIBLE';
      } finally {
        checking = false;
      }
    }
    checkBindings();
    setInterval(checkBindings, 500);
  }

  showStatus(null);
  refresh();
  setInterval(refresh, 1000);
})();
