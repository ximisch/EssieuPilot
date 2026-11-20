'use strict';
// A minimal enhancement to the genuine v1.6.2 UI.
// No replaced navigation, sidebar, dashboard, colors or profile controls.
(function () {
  function display(dot, active) {
    if (!dot) return;
    dot.classList.remove('amber');
    dot.style.background = active ? '#33db83' : '#f18a5b';
    dot.style.boxShadow = active ? '0 0 0 4px #33db8324' : '0 0 0 4px #f18a5b24';
  }
  const aside = document.querySelector('.side-bottom');
  const original = aside && aside.querySelector('.connection');
  if (!aside || !original) return;
  // Exactly one additional row, reusing all original EssieuPilot CSS classes.
  const pilot = original.cloneNode(true);
  pilot.id = 'originalPhysicalPilotStatus';
  pilot.style.marginTop = '9px';
  const pilotLight = pilot.querySelector('.status-light');
  pilot.querySelector('strong').textContent = 'PILOTAGE NON CONFIRMÉ';
  pilot.querySelector('small').textContent = 'Aucun retour de la DLL de pilotage';
  aside.insertBefore(pilot, aside.querySelector('.side-help'));
  const atsLight = original.querySelector('.status-light');
  const atsTitle = original.querySelector('#sideConnection');
  const atsDetail = original.querySelector('#sideConnectionDetail');
  const pilotTitle = pilot.querySelector('strong');
  const pilotDetail = pilot.querySelector('small');
  const quick = document.querySelector('.modern-side-quick');
  const quickLight = quick?.querySelector('.modern-small-dot');
  const quickText = quick?.querySelector('strong');
  const quickDetail = quick?.querySelector('small');
  let busy = false;
  async function refresh() {
    if (busy) return;
    busy = true;
    try {
      const response = await fetch('/api/status', { cache: 'no-store' });
      if (!response.ok) throw new Error('Statut ATS indisponible');
      const data = await response.json();
      if (typeof data.game_running !== 'boolean' || typeof data.steering_ack !== 'boolean')
        throw new Error('Réponse de diagnostic invalide');
      const running = data.game_running;
      const applied = running && data.steering_ack;
      display(atsLight, running);
      display(pilotLight, applied);
      display(quickLight, running);
      atsTitle.textContent = running ? 'ATS EN LIGNE' : 'ATS HORS LIGNE';
      atsDetail.textContent = running ? 'American Truck Simulator détecté' : 'Jeu non lancé';
      pilotTitle.textContent = applied ? 'PILOTAGE · RETOUR DLL' : 'PILOTAGE NON CONFIRMÉ';
      pilotDetail.textContent = applied ? 'Commande acquittée par la DLL (roues non vérifiées)' : 'Aucune commande confirmée';
      if (quickText) quickText.textContent = running ? 'ATS · EN LIGNE' : 'ATS · HORS LIGNE';
      if (quickDetail) quickDetail.textContent = applied ? 'Retour de la DLL de pilotage actif' : 'Pilotage physique non confirmé';
      const gameStat = document.getElementById('modernGameState');
      if (gameStat) gameStat.textContent = running ? '● ATS EN LIGNE' : '● ATS HORS LIGNE';
    } catch (_error) {
      display(atsLight, false);
      display(pilotLight, false);
      display(quickLight, false);
      atsTitle.textContent = 'ATS · ÉTAT INDISPONIBLE';
      atsDetail.textContent = 'Diagnostic non accessible';
      pilotTitle.textContent = 'PILOTAGE NON CONFIRMÉ';
      pilotDetail.textContent = 'Pas de signal confirmé';
    } finally { busy = false; }
  }
  refresh();
  setInterval(refresh, 1000);
})();
