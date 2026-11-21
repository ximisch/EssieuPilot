'use strict';
// EssieuPilot Original.16 — interactive cockpit skin. Physical hook unchanged.
// All trailer config changes use the existing schema-1 profile store.
(function(){
  const q=(s,root=document)=>root.querySelector(s);
  const el=(tag,cls,html)=>{const x=document.createElement(tag);if(cls)x.className=cls;if(html!==undefined)x.innerHTML=html;return x;};
  const esc=s=>String(s??'').replace(/[&<>"']/g,x=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[x]));
  const page=id=>q('.modern-page[data-page="'+id+'"]');
  const dashboard=page('dashboard'),trailers=page('trailers'),inputs=page('inputs'),viewer=page('viewer'),diagnostics=page('diagnostics'),updates=page('updates'),about=page('about');
  if(!dashboard||!trailers||!inputs||!viewer||!diagnostics||!updates||!about)return;
  document.body.classList.add('original16');
  const side=q('.sidebar'),main=q('.main'),nav=q('.studio-nav');
  // Preserve all controllers and their handlers while changing presentation.
  q('.modern-side-quick')?.classList.add('ep16-legacy-hidden');
  q('.side-bottom')?.classList.add('ep16-legacy-hidden');
  q('.topbar')?.classList.add('ep16-original-topbar');
  q('.notice')?.classList.add('ep16-legacy-hidden');
  q('.version')?.classList.add('ep16-nav-heading');
  const axlesNav=q('[data-modern-tab="axles"]');
  if(axlesNav)axlesNav.remove();
  const trailerNav=q('[data-modern-tab="trailers"]');
  if(trailerNav){trailerNav.querySelector('strong').textContent='Mes remorques';trailerNav.querySelector('small').textContent='Profils · Remorques · Essieux';}
  const viewerNav=q('[data-modern-tab="viewer"]');
  if(viewerNav){viewerNav.querySelector('strong').textContent='Simulation';viewerNav.querySelector('small').textContent='Vue 2D · Vue 3D';}
  const diagnosticNav=q('[data-modern-tab="diagnostics"]');if(diagnosticNav)diagnosticNav.querySelector('small').textContent='État · Tests · Rapports';
  // All menu transitions continue to use modern.js's established data-modern-tab handler.
  const go=id=>q('[data-modern-tab="'+id+'"]')?.click();
  const menuButtons=[['trailers','Mes remorques','Gérer tes profils, remorques et essieux','▱'],['inputs','Commandes','Touches, manette et joystick','⌘'],['viewer','Simulation','Tester les vues 2D et 3D','⬡'],['diagnostics','Diagnostic','Contrôler les connexions et rapports','⌁'],['updates','Mises à jour','Installer une nouvelle version','↻'],['about','À propos','Informations et compatibilité','ⓘ']];
  // Dashboard: wide hero, two TRUE game/steering-state indicators, all shortcuts visible.
  for(const selector of ['.essieupilot-home-header','.essieupilot-hero','.essieupilot-feature-grid','.modern-statbar'])dashboard.querySelector(selector)?.classList.add('ep16-legacy-hidden');
  const original2D=q('.visualize'),originalController=q('.controls');
  if(original2D)viewer.insertBefore(original2D,viewer.querySelector('.modern-viewer-layout'));
  if(originalController)inputs.appendChild(originalController);
  const home=el('div','ep16-home');
  home.innerHTML='<section class="ep16-home-hero" aria-label="Accueil EssieuPilot">'+
   '<img class="ep16-home-hero-image" src="/hero_reference.jpg" alt="Remorque surbaissée dans un paysage nocturne">'+
   '<div class="ep16-home-shade"></div><div class="ep16-hero-brand"><span class="ep16-hero-logo">Essieu<span>Pilot</span></span><small>PILOTAGE DES ESSIEUX · AMERICAN TRUCK SIMULATOR</small></div>'+
   '<div class="ep16-hero-statuses" role="status" aria-live="polite">'+
   '<div class="ep16-led unknown" id="ep16ATSStatus"><i></i><span><b>ATS · VÉRIFICATION</b><small>Recherche du jeu…</small></span></div>'+
   '<div class="ep16-led unknown" id="ep16AxleStatus"><i></i><span><b>ESSIEUX · ÉTAT INCONNU</b><small>Contrôle du pilotage…</small></span></div></div></section>'+
   '<section class="ep16-home-menu" id="ep16HomeMenu" aria-label="Accès aux menus"></section>'+
   '<section class="ep16-home-bottom"><div class="ep16-home-stats"><h3>▥ STATISTIQUES</h3><p><span>Remorques configurées</span><strong id="ep16NumTrailers">—</strong></p><p><span>Essieux enregistrés</span><strong id="ep16NumAxles">—</strong></p><p><span>Profils enregistrés</span><strong id="ep16NumProfiles">—</strong></p></div>'+
   '<div class="ep16-home-actions"><h3>ϟ ACTIONS RAPIDES</h3><div class="ep16-home-buttons">'+
   '<button type="button" class="ep16-primary" id="ep16StartControl">▶ Piloter les essieux</button>'+
   '<button type="button" data-action="import">⇩ Importer</button><button type="button" data-action="export">⇧ Exporter</button><button type="button" data-action="save">▣ Enregistrer</button></div><small>L’activation dépend de la connexion ATS, des autorisations et de la configuration enregistrée.</small></div></section>'+
   '<footer class="ep16-home-footer"><span>▱ EssieuPilot</span><span id="ep16HomeTrailer">Aucune remorque détectée</span><span id="ep16HomeFooterState">Solo expérimental · Convoi non validé</span></footer>';
  dashboard.prepend(home);
  const cards=q('#ep16HomeMenu');
  menuButtons.forEach(([id,title,subtitle,icon])=>{
     const b=el('button','ep16-menu-card');
     b.type='button';b.dataset.modernGo=id;
     b.innerHTML='<span class="ep16-card-icon">'+icon+'</span><span><strong>'+title+'</strong><small>'+subtitle+'</small></span><span class="ep16-card-chevron">›</span>';
     cards.appendChild(b);
  });
  q('#ep16StartControl').addEventListener('click',()=>go('inputs'));
  // Single "Mes remorques" page: profile manager, saved trailers, axle config,
  // physical limits and ATS associations live together, not in Diagnostic.
  const profileSidebar=[...q('.sidebar').querySelectorAll('.side-section')].find(x=>x.querySelector('#profiles'));
  const trailerShell=el('div','ep16-trailer-shell');
  const left=el('div','ep16-trailer-left');
  const right=el('div','ep16-trailer-right');
  if(profileSidebar){profileSidebar.classList.add('ep16-profiles-manager');left.appendChild(profileSidebar);}
  const presets=el('section','ep16-panel ep16-presets');
  presets.innerHTML='<h3>＋ Modèles de remorques</h3><p>Modèles de départ à personnaliser. Le nombre réel d’essieux directeurs et l’identifiant ATS doivent être vérifiés avant tout pilotage physique.</p>'+
   '<label>Type de modèle<select id="ep16Preset">'+
   '<option value="surbaissée">Surbaissée · Lowboy</option><option value="plateau">Plateau · Flatbed</option>'+
   '<option value="extensible">Porte-engins · RGN</option><option value="fermée">Fourgon · Caisse</option>'+
   '<option value="citerne">Citerne</option><option value="forestière">Forestière</option>'+
   '<option value="générique">Autre / Mod personnalisé</option></select></label>'+
   '<label>Essieux configurés (indicatif, à vérifier)<input id="ep16PresetAxles" type="number" min="1" max="32" step="1" value="2"></label>'+
   '<button type="button" id="ep16AddPreset">＋ Ajouter ce modèle</button>'+
   '<small>Ces présélections ne prétendent pas identifier toutes les remorques SCS ou les essieux directeurs des mods.</small>';
  left.appendChild(presets);
  const summary=el('section','ep16-panel ep16-current-trailer');
  summary.innerHTML='<div class="ep16-panel-title"><h3>Remorque sélectionnée</h3><span id="ep16CurrentType">Configuration locale</span></div>'+
   '<div class="ep16-trailer-visual"><img src="/hero_reference.jpg" alt="Illustration de remorque"><div><strong id="ep16TrailerName">—</strong><small id="ep16TrailerInfo">Sélectionne une remorque</small></div></div>';
  right.appendChild(summary);
  const studio=q('.axle-panel');if(studio)right.appendChild(studio);
  const axleSettings=q('.modern-page[data-page="axles"] .settings');if(axleSettings)right.appendChild(axleSettings);
  const presetPhysical=el('section','ep16-panel ep16-physics');
  presetPhysical.innerHTML='<div class="ep16-panel-title"><h3>⚙ Réglages physiques de cette remorque</h3><span>SOLO · EXPÉRIMENTAL</span></div>'+
    '<div class="ep16-physical-fields"><label>Limite physique (0 à 60°)<input id="ep16PhysicalLimit" type="number" min="0" max="60" step="0.5"></label>'+
    '<label>Référence d’étalonnage (35 à 120°)<input id="ep16PhysicalCalibration" type="number" min="35" max="120" step="1"></label></div>'+
    '<p>0° demande des roues droites. L’étalonnage est une référence de commande, pas une mesure garantie de l’angle physique dans ATS.</p>'+
    '<button id="ep16SavePhysical" type="button">Enregistrer les réglages de cette remorque</button><span id="ep16PhysicalMessage" role="status"></span>';
  right.appendChild(presetPhysical);
  const physicalAssociation=q('#epPhysicalTrailerChoice')?.closest('.modern-diagnostic-row');
  if(physicalAssociation){
    physicalAssociation.classList.add('ep16-association');
    const associationWrap=el('section','ep16-panel ep16-association-wrap');
    associationWrap.innerHTML='<h3>▱ Association automatique avec ATS</h3><p>Lorsqu’un identifiant unique est reconnu, associe cette remorque une seule fois pour retrouver ensuite ses réglages.</p>';
    associationWrap.appendChild(physicalAssociation);
    right.appendChild(associationWrap);
  }
  const oldInfo=q('.modern-page[data-page="trailers"] .modern-info-panel');
  if(oldInfo){oldInfo.classList.add('ep16-legacy-hidden');trailers.appendChild(oldInfo);}
  trailerShell.append(left,right);trailers.appendChild(trailerShell);
  q('.modern-page[data-page="axles"]')?.classList.add('ep16-legacy-hidden');
  // Restore useful 2D + 3D simulation in the same menu. Existing 2D canvas
  // uses app.js; the existing 3D canvas uses modern.js. Both stay functional.
  const switcher=el('div','ep16-view-switch');
  switcher.innerHTML='<button type="button" data-ep16-view="2d">▱ Vue 2D</button><button type="button" data-ep16-view="3d" class="active">⬡ Vue 3D</button>';
  viewer.insertBefore(switcher,viewer.firstChild);
  const viewer3d=q('.modern-viewer-layout');
  const setViewer=mode=>{
    for(const b of switcher.querySelectorAll('button'))b.classList.toggle('active',b.dataset.ep16View===mode);
    if(original2D)original2D.hidden=mode!=='2d';
    if(viewer3d)viewer3d.hidden=mode!=='3d';
    if(mode==='3d')q('[data-camera="reset"]')?.click();
  };
  switcher.addEventListener('click',e=>{const b=e.target.closest('[data-ep16-view]');if(b)setViewer(b.dataset.ep16View);});
  setViewer('3d');
  // Diagnostic: preserve actual native reports and telemetry, remove outdated
  // static declarations that incorrectly say steering is unavailable.
  const diagCore=q('.modern-diagnostic');
  if(diagCore){
    const staticRows=[...diagCore.children].filter(x=>x.classList.contains('modern-diagnostic-row') &&
      /Connexion à American Truck Simulator|Remorques et profils|Braquage physique des essieux|Mode Convoi|Mises à jour de l'application|Vue 3D/.test(x.textContent));
    for(const row of staticRows)row.remove();
    const diagHeader=el('section','ep16-diagnostic-intro');
    diagHeader.innerHTML='<h3>État du système</h3><div class="ep16-diagnostic-states">'+
      '<article><small>ATS</small><strong id="ep16DiagATS">Recherche…</strong></article>'+
      '<article><small>TÉLÉMÉTRIE</small><strong id="ep16DiagTelemetry">Recherche…</strong></article>'+
      '<article><small>DLL PHYSIQUE</small><strong id="ep16DiagDLL">Recherche…</strong></article>'+
      '<article><small>PILOTAGE</small><strong id="ep16DiagSteering">Inactif</strong></article></div>'+
      '<h3>Remorque et configuration reconnues</h3><p id="ep16DiagTrailer">En attente du jeu.</p>'+
      '<h3>Commandes et retour physique</h3><div class="ep16-diagnostic-data">'+
      '<span>Mode manuel</span><b id="ep16DiagArmed">—</b><span>Consigne envoyée</span><b id="ep16DiagCommand">—</b>'+
      '<span>Callback DLL</span><b id="ep16DiagAck">—</b><span>Limitations</span><b>Une commande globale par remorque · Convoi non validé</b></div>'+
      '<h3>Rapports et contrôles</h3>';
    diagnostics.insertBefore(diagHeader,diagCore);
    diagCore.classList.add('ep16-diagnostic-logs');
  }
  // Installation instructions and historical context belong to À propos.
  const atsHelp=q('.modern-page[data-page="diagnostics"] .essieupilot-utility-panel');
  if(atsHelp){atsHelp.classList.add('ep16-ats-help');about.appendChild(atsHelp);}
  const aboutExtra=q('.modern-page[data-page="about"] .essieupilot-utility-panel:not(.ep16-ats-help)');
  if(aboutExtra){
    const old=aboutExtra.querySelector('.essieupilot-list');
    if(old)old.innerHTML='<b>Fonctions disponibles et état réel</b>'+
      '<div>✓ Profils, remorques et essieux configurables dans « Mes remorques »</div>'+
      '<div>✓ Touches, manettes, joysticks et commande analogique</div>'+
      '<div>✓ Simulation 2D et 3D, recentrage en mouvement</div>'+
      '<div>✓ Reconnaissance des remorques par identifiant ATS unique, après association</div>'+
      '<div>◉ Braquage physique expérimental observé sur certaines remorques SOLO ; réglages exacts non calibrés</div>'+
      '<div>⚠ Convoi et pilotage indépendant de chaque essieu physique non validés</div>';
    aboutExtra.querySelector('.essieupilot-panel-title p').textContent='EssieuPilot est un logiciel Windows de pilotage expérimental des essieux de remorque pour American Truck Simulator.';
  }
  // Keep update API and confirmation flow completely unchanged; only layout.
  const updatesPanel=q('.modern-page[data-page="updates"] .essieupilot-utility-panel');
  if(updatesPanel){updatesPanel.classList.add('ep16-update-panel');}
  // Status indicator: only assert ACTIVE when ATS runs, manual is armed,
  // the native DLL is acknowledged and a physical command was emitted.
  let lastGame=null,lastInput=null,lastTelemetry=null,fetchBusy=false;
  const paintLED=(id,state,title,detail)=>{
    const x=q('#'+id);if(!x)return;
    x.classList.remove('on','off','unknown');x.classList.add(state);
    x.querySelector('b').textContent=title;x.querySelector('small').textContent=detail;
  };
  const setText=(id,v)=>{const x=q('#'+id);if(x)x.textContent=String(v??'—');};
  function refreshDashboard(){
    if(data&&data.profiles){
      setText('ep16NumProfiles',data.profiles.length);
      setText('ep16NumTrailers',data.profiles.reduce((v,p)=>v+(p.trailers||[]).length,0));
      setText('ep16NumAxles',data.profiles.reduce((v,p)=>v+(p.trailers||[]).reduce((a,t)=>a+(t.axles||[]).length,0),0));
    }
    const tr=lastTelemetry?.connected&&lastTelemetry?.trailers?.length===1?lastTelemetry.trailers[0]:null;
    setText('ep16HomeTrailer',tr?'Remorque ATS : '+(tr.name||tr.id):'Aucune remorque ATS reconnue');
    const inp=lastInput;
    if(!lastGame||typeof lastGame.game_running!=='boolean')paintLED('ep16ATSStatus','unknown','ATS · ÉTAT INCONNU','Connexion à vérifier');
    else if(lastGame.game_running)paintLED('ep16ATSStatus','on','ATS · EN LIGNE','Jeu détecté et lancé');
    else paintLED('ep16ATSStatus','off','ATS · HORS LIGNE','Jeu fermé ou non détecté');
    const running=lastGame?.game_running===true;
    const active=running && inp?.manual_armed===true&&inp?.physical_control_enabled===true&&inp?.physical_callback_ack===true;
    if(active)paintLED('ep16AxleStatus','on','ESSIEUX · ACTIFS','Commande physique transmise à la DLL (solo)');
    else if(inp?.service_running===false||!lastGame)paintLED('ep16AxleStatus','unknown','ESSIEUX · ÉTAT INCONNU','Contrôleur non disponible');
    else paintLED('ep16AxleStatus','off','ESSIEUX · INACTIFS',!running?'ATS non lancé':!inp?.manual_armed?'Pilotage manuel désactivé':'Aucun retour physique confirmé');
    setText('ep16DiagATS',running?'Jeu détecté':lastGame?'Hors ligne':'Inconnu');
    setText('ep16DiagTelemetry',lastTelemetry?.connected?'Connectée':'Non disponible');
    setText('ep16DiagDLL',inp?.physical_dll_sdk_accepted?'Acceptée par ATS':'Non confirmée');
    setText('ep16DiagSteering',active?'Commande physique confirmée côté DLL':'Inactif / non confirmé');
    setText('ep16DiagTrailer',tr?(tr.name||tr.id)+' · Identifiant ATS : '+tr.id+' · Association : '+(inp?.physical_selection_reason||'inconnue'):'Aucune remorque reconnue pour le moment.');
    setText('ep16DiagArmed',inp?.manual_armed?'Activé':'Désactivé');
    setText('ep16DiagCommand',inp?.physical_packet_sent?Number(inp.physical_packet_normalized*100).toFixed(1)+' % ATS':'Aucune commande envoyée');
    setText('ep16DiagAck',inp?.physical_callback_ack?'Réponse DLL reçue':'Aucune réponse récente');
  }
  async function poll(){
    if(fetchBusy)return;fetchBusy=true;
    const fetchJSON=async url=>{try{const r=await fetch(url,{cache:'no-store'});if(!r.ok)throw Error('indisponible');return await r.json();}catch(_){return null;}};
    try{
      const [game,input,t]=await Promise.all([fetchJSON('/api/status'),fetchJSON('/api/control-link'),fetchJSON('/api/telemetry')]);
      lastGame=game;lastInput=input;lastTelemetry=t;
      refreshDashboard();
    }finally{fetchBusy=false;}
  }
  const types=['Surbaissée · Lowboy','Plateau · Flatbed','Porte-engins · RGN','Fourgon · Caisse','Citerne','Forestière','Autre / Mod personnalisé'];
  function refreshTrailer(){
    if(!data||!getProfile())return;
    const tr=getTrailer();if(!tr)return;
    const n=q('#ep16TrailerName'),typ=q('#ep16CurrentType');
    if(n)n.textContent=tr.name;
    if(typ)typ.textContent=(tr.model_type||'Personnalisée');
    setText('ep16TrailerInfo',(tr.axles||[]).length+' essieu(x) configuré(s) · Association ATS : '+(tr.ats_match_key||'non enregistrée'));
    if(document.activeElement?.id!=='ep16PhysicalLimit')
      q('#ep16PhysicalLimit').value=tr.physical_angle_limit_degrees??Math.max(0,...tr.axles.filter(a=>a.enabled).map(a=>Number(a.max_angle)||0));
    if(document.activeElement?.id!=='ep16PhysicalCalibration')
      q('#ep16PhysicalCalibration').value=tr.physical_full_scale_degrees||60;
    const selected=q('#epPhysicalTrailerChoice');
    if(selected&&selected.value!==tr.id && [...selected.options].some(o=>o.value===tr.id)){
      selected.value=tr.id;selected.dispatchEvent(new Event('change',{bubbles:true}));
    }
  }
  // Never mutate presets or discard existing profiles just by upgrading.
  q('#ep16AddPreset').addEventListener('click',()=>{
    if(!data||!getProfile())return;
    const p=getProfile();
    if(p.trailers.length>=20){toast('Maximum : 20 configurations par profil.',true);return;}
    const selection=q('#ep16Preset'),count=Number(q('#ep16PresetAxles').value);
    if(!Number.isInteger(count)||count<1||count>32){toast('Choisis de 1 à 32 essieux configurables.',true);return;}
    const idx=selection.selectedIndex,type=selection.value;
    const names=['Lowboy','Plateau','Porte-engins RGN','Fourgon','Citerne','Forestière','Remorque personnalisée'];
    const item=makeTrailer(p.trailers.length+1);
    item.name=names[idx]+' · '+(p.trailers.length+1);
    item.model_type=type;
    item.axles=Array.from({length:count},(_,i)=>{const a=makeAxle(i+1);a.max_angle=35;return a;});
    item.ats_match_key='';item.physical_full_scale_degrees=60;
    item.physical_angle_limit_degrees=35;
    p.trailers.push(item);
    selectedTrailerId=item.id;selectedAxleId=null;
    markDirty();renderAll();refreshTrailer();
    toast('Modèle ajouté. Vérifie les essieux et associe la remorque dans ATS avant tout essai physique.');
  });
  q('#ep16SavePhysical').addEventListener('click',async()=>{
    const tr=getTrailer();if(!tr)return;
    const n=Number(q('#ep16PhysicalLimit').value),ref=Number(q('#ep16PhysicalCalibration').value);
    const max=Math.max(0,...tr.axles.filter(a=>a.enabled).map(a=>Number(a.max_angle)||0));
    const msg=q('#ep16PhysicalMessage');
    if(!Number.isFinite(n)||n<0||n>60||n>max||!Number.isFinite(ref)||ref<35||ref>120||ref<n){
      msg.textContent='Valeurs invalides : limite 0–60°, au plus le maximum d’essieu, et référence 35–120° supérieure à la limite.';
      return;
    }
    tr.physical_angle_limit_degrees=n;
    tr.physical_full_scale_degrees=ref;
    markDirty();await save();
    msg.textContent=dirty?'Erreur de sauvegarde : vérifie les profils.':'Réglages enregistrés pour « '+tr.name+' ».';
    refreshTrailer();
  });
  q('#trailerSelect')?.addEventListener('change',()=>queueMicrotask(refreshTrailer));
  // Called after app.js changes profile, trailer, axle, or imports.
  const oldRenderAll=renderAll;
  renderAll=function(){oldRenderAll();refreshTrailer();refreshDashboard();};
  refreshTrailer();refreshDashboard();
  poll();setInterval(poll,1400);
  // No physical claims from simulated per-axle animations; measured wheel angles
  // are not exposed reliably by the current experimental ATS telemetry.
})();