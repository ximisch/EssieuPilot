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
  showStatus(null);
  refresh();
  setInterval(refresh, 1000);
})();
