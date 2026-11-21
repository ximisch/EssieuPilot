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
  document.addEventListener('click', (event) => {
    const control = event.target.closest?.('[data-action="toggle-manual"]');
    if (!control) return;
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
    const row = document.createElement('div');
    row.className = 'modern-diagnostic-row';
    row.innerHTML = '<div><b>Touches configurables → commandes ATS (Original.7)</b>' +
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
        if (c.physical_control_enabled || c.dll_confirmed)
          throw new Error('État physique non autorisé pendant le premier essai');
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
          '\nAppuis bruts détectés (y compris INACTIF, diagnostic seul) : ' + counts +
          '\n' + last + '\nHistorique : ' + recent +
          '\nATTENTION : aucun ordre de direction physique envoyé à ATS dans Original.7.' +
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
