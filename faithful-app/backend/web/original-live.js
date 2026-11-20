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
  // Stage 1 is displayed in the existing Diagnostic tab, not a new Pilotage tab.
  const diag = document.querySelector('.modern-diagnostic');
  if (diag) {
    const row = document.createElement('div');
    row.className = 'modern-diagnostic-row';
    row.innerHTML = '<div><b>Touches configurables → canal DLL (étape 1)</b>' +
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
        const focus = c.ats_foreground ? 'ATS au premier plan' :
          c.ats_running ? 'ATS lancé, mais EssieuPilot est au premier plan' : 'ATS non lancé';
        const counts = 'Activation : ' + (c.activation_count || 0) +
          ' · Gauche : ' + (c.left_count || 0) +
          ' · Droite : ' + (c.right_count || 0);
        const recent = Array.isArray(c.input_history) && c.input_history.length ?
          c.input_history.slice(-5).join(' / ') : 'Aucun appui reconnu';
        const last = c.last_input ? 'Dernière commande : ' + c.last_input +
          (c.last_event_time ? ' à ' + c.last_event_time.slice(11,19) : '') :
          'Aucune commande enregistrée';
        text.style.whiteSpace = 'pre-line';
        text.textContent = 'Profil : ' + name + '\nTouches : ' + keys +
          '\nMode manuel : ' + armed + '  |  ' + focus +
          '\nCompteurs des appuis : ' + counts +
          '\n' + last + '\nHistorique : ' + recent +
          '\nCanal IPC de test uniquement : aucune direction envoyée aux roues d’ATS.';
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
