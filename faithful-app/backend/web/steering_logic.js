'use strict';
// Moteur independant du jeu : simulation et tests. Ne pilote pas la physique d'ATS.
(function (root, create) {
  const engine = create();
  if (typeof module !== 'undefined' && module.exports) module.exports = engine;
  if (root) root.SteeringLogic = engine;
})(typeof globalThis !== 'undefined' ? globalThis : null, () => {
  const clamp = (n, low, high) => Math.max(low, Math.min(high, n));
  const approach = (a, b, step) => Math.abs(b - a) <= step ? b : a + Math.sign(b - a) * step;
  /**
   * Retourne un signal gradue [-1,+1]. Le zero est la position de repos
   * enregistree a la capture, avec zone morte et courbe de sensibilite.
   */
  function analogPower(raw, neutral = 0, deadzone = 0.08, curve = 1.35, inverted = false, fullScalePct = 100, outputPct = 100) {
    if (!Number.isFinite(raw)) return 0;
    neutral = clamp(Number(neutral) || 0, -0.95, 0.95);
    deadzone = clamp(Number(deadzone) || 0, 0, 0.4);
    curve = clamp(Number(curve) || 1, 0.5, 3);
    const centered = raw >= neutral ? (raw - neutral) / (1 - neutral) : (raw - neutral) / (1 + neutral);
    const abs = Math.abs(centered);
    if (abs <= deadzone) return 0;
    // Une course physique partielle peut correspondre a 100 % de braquage.
    // Exemple: fullScalePct=75 -> 75 % de course = 100 % de commande.
    // outputPct borne l'angle de sortie, par exemple 80 % de 60 degres = 48 degres.
    fullScalePct = clamp(Number(fullScalePct) || 100, 20, 100);
    outputPct = clamp(Number(outputPct) >= 0 ? Number(outputPct) : 100, 0, 100);
    const pct = clamp(abs * 100 / fullScalePct, 0, 1);
    return (inverted ? -1 : 1) * Math.sign(centered) * Math.pow(pct, curve) * outputPct / 100;
  }
  /** 
   * Regles de priorite :
   * 1) Stick analogue hors neutre : angle proportionnel ; touches : degrés/seconde.
   * 2) Relâchement + vitesse non nulle, marche AV ou AR : retour doux au centre.
   * 3) Arrêt réel : l'angle est maintenu pour toutes les commandes.
   * En aucune circonstance un mouvement du joystick ne desactive le mode manuel.
   */
  function updateAxle(axle, control) {
    const dt = clamp(Number(control.dt) || 0, 0, 0.1);
    const limit = clamp(Number(axle.max_angle) || 0, 0, 60);
    const ratio = clamp(Number(axle.ratio) || 1, 0.05, 2);
    const before = clamp(Number(axle.current_angle) || 0, -limit, limit);
    const steer = control.manual && axle.enabled ? clamp(Number(control.steer) || 0, -1, 1) : 0;
    const speed = Math.abs(Number(control.speedKmh) || 0);
    const moving = Number.isFinite(Number(control.speedKmh)) ? speed >= 0.2 : (control.mode === 'forward' || control.mode === 'reverse');
    const gain = clamp(Number(control.speedGain) || 0, 0, 4);
    const centerSpeedFactor = 1 + speed / 30 * gain;
    const result = {angle: before, action: 'maintien'};
    if (control.manual && axle.enabled && control.analogActive && Math.abs(Number(control.analogSteer) || 0) > 1e-6) {
      // Hors zone morte : la position du stick commande l'angle immédiatement.
      // Au neutre : l'essieu reste bloqué à l'arrêt, recentrage progressif uniquement en roulant.
      const stick = clamp(Number(control.analogSteer) || 0, -1, 1);
      const sign = axle.reversed ? -1 : 1;
      result.angle = clamp(stick * limit * ratio * sign, -limit, limit);
      result.action = 'position analogique';
    } else if (Math.abs(steer) > 1e-6) {
      const target = Math.sign(steer) * (axle.reversed ? -1 : 1) * limit;
      const instantTarget = target * Math.abs(steer);
      const rate = Math.max(0, Number(axle.turn_rate) || 0) * dt * ratio * Math.abs(steer);
      result.angle = axle.turn_instant ? instantTarget : approach(before, target, rate);
      result.action = 'correction';
    } else if (moving && control.autoCenter !== false) {
      const rate = Math.max(0, Number(axle.center_rate) || 0) * dt * centerSpeedFactor;
      result.angle = axle.center_instant ? 0 : approach(before, 0, rate);
      result.action = 'recentrage';
    }
    result.angle = clamp(result.angle, -limit, limit);
    return result;
  }
  return {analogPower, updateAxle};
});
