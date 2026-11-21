'use strict';
// Interface V1.4.1 : navigation sans framework, ancienne structure de profils conservee.
(function(){
  const navItems = [
    ['dashboard','⌂','Tableau de bord','Accueil'],
    ['trailers','▣','Remorques','Profils'],
    ['axles','◉','Essieux','Réglages par essieu'],
    ['inputs','⌘','Commandes','Périphériques'],
    ['viewer','⬡','Simulation 3D','Visualisation'],
    ['diagnostics','〽','Diagnostic','Connexion ATS'],
    ['updates','↻','Mises à jour','Compatibilité et logiciel'],
    ['about','ⓘ','À propos','Informations']
  ];
  const main = document.querySelector('.main');
  const side = document.querySelector('.sidebar');
  const oldGrid = document.querySelector('.main-grid');
  const top = document.querySelector('.topbar');
  main.classList.add('modern-main');
  side.classList.add('modern-sidebar');
  const nav = document.createElement('nav');
  nav.className='studio-nav';
  nav.setAttribute('aria-label','Navigation principale');
  nav.innerHTML = '<div class="modern-nav-heading">ESSIEUPILOT</div>' + navItems.map(([id,ico,label,sub],i) => `<button type="button" class="modern-nav-item ${i===0?'active':''}" data-modern-tab="${id}"><span class="modern-nav-icon">${ico}</span><span><strong>${label}</strong><small>${sub}</small></span><span class="modern-nav-chevron">›</span></button>`).join('');
  side.insertBefore(nav,side.querySelector('.side-section'));
  const quick = document.createElement('div');
  quick.className='modern-side-quick';
  quick.innerHTML='<span class="modern-small-dot"></span><span><strong>ATS · CONNEXION</strong><small>Lecture seule · physique non validée</small></span>';
  side.insertBefore(quick,side.querySelector('.side-bottom'));
  // Les profils restent accessibles en permanence, les périphériques passent dans Diagnostic.
  const deviceSection = [...side.querySelectorAll('.side-section')].find(n=>n.querySelector('#devices'));
  if(deviceSection)deviceSection.classList.add('device-section-moved');
  const pages = {};
  for(const [id,,label] of navItems){
    const page=document.createElement('section');page.className='modern-page';page.dataset.page=id;
    if(id!=='dashboard')page.hidden=true;
    page.innerHTML=`<div class="modern-section-intro"><div><div class="modern-kicker">ESSIEUPILOT / ${label.toUpperCase()}</div><h2>${label}</h2><p class="modern-section-description"></p></div></div>`;
    pages[id]=page;oldGrid.appendChild(page);
  }
  const descriptions={dashboard:'Une vue claire pour piloter, visualiser et tester en temps réel.',trailers:'Classe les profils et prépare la future association automatique aux remorques.',axles:'Un réglage indépendant pour chaque essieu directeur et chaque remorque.',inputs:'Affecte tes touches et tes axes analogiques en déplaçant le périphérique.',viewer:'Observe en trois dimensions la direction et les essieux de ta configuration.',diagnostics:'Contrôle la télémétrie ATS, les périphériques et les limitations.',updates:'État des mises à jour et compatibilité ATS.',about:'Informations sur EssieuPilot et les fonctions réellement disponibles.'};
  for(const id of Object.keys(pages))pages[id].querySelector('.modern-section-description').textContent=descriptions[id];
  function move(selector,id){const item=document.querySelector(selector);if(item) pages[id].appendChild(item);}
  move('.visualize','dashboard');move('.controls','dashboard');move('.axle-panel','trailers');move('.settings','axles');move('.binding-panel','inputs');move('.about-panel','about');
  if(deviceSection)pages.diagnostics.appendChild(deviceSection);
  // Page d'accueil fidèle à la maquette choisie par l'utilisateur ; ce sont de vrais boutons.
  const hero=document.createElement('div'); hero.className='essieupilot-hero';
  hero.innerHTML=`<img src="/hero_reference.jpg" alt="EssieuPilot : camion et remorque surbaissée, identité visuelle validée" />
    <div class="essieupilot-hero-foot"><span>◈ Pilotage intelligent des essieux</span><span class="essieupilot-pending">Convoi : non validé</span></div>`;
  const featureGrid=document.createElement('div');featureGrid.className='essieupilot-feature-grid';
  const featureNames=['Tableau de bord','Remorques','Essieux','Commandes','Simulation 3D','Diagnostic'];
  const featurePages=['dashboard','trailers','axles','inputs','viewer','diagnostics'];
  featureGrid.innerHTML=featurePages.map((id,i)=>`<button type="button" data-modern-go="${id}" class="essieupilot-feature" aria-label="Ouvrir ${featureNames[i]}"><img alt="${featureNames[i]}" src="/feature_${i}.jpg"></button>`).join('');
  const homeHeading=document.createElement('div');homeHeading.className='essieupilot-home-header';
  homeHeading.innerHTML='<span>◉ CENTRE DE PILOTAGE</span><b>EssieuPilot</b><span>Une seule application · Configuration, simulation et profils</span>';
  pages.dashboard.insertBefore(homeHeading,pages.dashboard.querySelector('.visualize'));
  pages.dashboard.insertBefore(hero,pages.dashboard.querySelector('.visualize'));
  pages.dashboard.insertBefore(featureGrid,pages.dashboard.querySelector('.visualize'));
  const stats=document.createElement('div');stats.className='modern-statbar';stats.innerHTML='<div><small>STATUT DU JEU</small><strong id="modernGameState"><i class="modern-offline-dot"></i>Non connecté</strong></div><div><small>VITESSE ATS</small><strong id="modernGameSpeed">— km/h</strong></div><div><small>ANGLE MAXIMAL</small><strong>60° par essieu</strong></div><div><small>PROFILS</small><strong id="modernProfilesCount">1 enregistré</strong></div>';
  pages.dashboard.insertBefore(stats,pages.dashboard.querySelector('.visualize'));
  const trailerInfo=document.createElement('div');trailerInfo.className='modern-info-panel';trailerInfo.innerHTML=`<div><strong>Identification et association des profils</strong><p>Si une télémétrie SCS compatible est installée, EssieuPilot reconnaît l'identifiant de la remorque attelée et recharge le profil déjà associé.</p></div><label class="modern-match-label">Identifiant de remorque (préparation)<input id="matchKey" type="text" maxlength="120" placeholder="Ex. lowboy_3_axles" autocomplete="off"></label><div id="detectedTrailer" class="modern-detected">Remorque détectée : aucune</div><button id="associateTrailer" class="btn btn-primary" type="button">Associer la remorque détectée à ce profil</button><div class="modern-match-label">Modèle visuel<select id="modelType"><option value="surbaissée">Surbaissée · lowboy</option><option value="plateau">Plateau</option><option value="extensible">Plateau extensible</option><option value="fermée">Fourgon / caisse</option><option value="citerne">Citerne</option><option value="forestière">Forestière</option><option value="générique">Générique</option></select></div><span class="modern-muted-note">Les formes 2D et 3D sont des modèles schématiques adaptés, non des maillages extraits du jeu.</span>`;
  pages.trailers.appendChild(trailerInfo);
  const axleIntro=document.createElement('div');axleIntro.className='modern-shortcut';axleIntro.innerHTML='<div><b>Essieu sélectionné</b><p>Pour choisir un autre essieu, utilise l’onglet Remorques & profils.</p></div><button class="btn btn-ghost" data-modern-go="trailers">Choisir un essieu →</button>';
  pages.axles.insertBefore(axleIntro,pages.axles.querySelector('.settings'));
  const analogInfo=document.createElement('div');analogInfo.className='modern-info-panel';analogInfo.innerHTML='<div><strong>Commande proportionnelle jusqu’à 100 %</strong><p>En réglage standard : 10 % du stick = 10 % de l’angle maximal, 100 % = 100 %. Ajuste « Course du stick pour 100 % » et « Braquage maximal utilisé » selon ton périphérique. Les boutons numériques gardent les °/s.</p></div><div class="modern-formula">100 % × 60° = <b>60°</b></div>';
  pages.inputs.insertBefore(analogInfo,pages.inputs.querySelector('.binding-panel'));
  const viewer=document.createElement('div');viewer.className='modern-viewer-layout';viewer.innerHTML=`
    <div class="modern-3d-panel"><div class="modern-3d-head"><div><span class="modern-kicker">MODÈLE INTERACTIF</span><h3>Vue 3D des essieux</h3><small>Modèle schématique · rotations à la souris</small></div><span class="modern-3d-badge">3D SIMULÉE</span></div>
      <div class="modern-3d-stage"><canvas id="scene3d" aria-label="Simulation tridimensionnelle des essieux et remorques"></canvas><div class="modern-3d-overlays"><span>GLISSER POUR ORIENTER</span><span>MOLETTE POUR ZOOMER</span></div></div>
      <div class="modern-3d-toolbar"><button data-camera="iso">Isométrique</button><button data-camera="top">Dessus</button><button data-camera="side">Profil</button><button data-camera="reset">Réinitialiser</button></div></div>
    <div class="modern-viewer-side"><div class="modern-panel-inner"><div class="modern-kicker">STICK ANALOGIQUE · −100 % À +100 %</div><h3>Test en direct</h3><p>Teste les angles sans brancher de périphérique. Relâcher le stick à l'arrêt conserve l'angle actuel ; en roulant il revient progressivement selon la vitesse réglée.</p><label class="modern-check"><input id="virtualAnalogEnabled" type="checkbox"> Activer l’axe virtuel</label><div class="modern-big-angle" id="virtualAngle">0°</div><input id="virtualAnalogRange" type="range" min="-100" max="100" step="1" value="0" disabled><div class="range-ends"><span>−100 %</span><span id="virtualAnalogPct">0 %</span><span>+100 %</span></div><button class="btn btn-ghost" id="virtualReset">Relâcher le stick</button><button class="btn btn-primary" id="virtualManual">Activer le pilotage</button></div><div class="modern-panel-inner"><div class="modern-kicker">ANGLES INSTANTANÉS</div><h3>Mesures par essieu</h3><div id="modern3DAxles" class="modern-3d-axles"></div></div></div>`;
  pages.viewer.appendChild(viewer);
  const diag=document.createElement('div');diag.className='modern-diagnostic';diag.innerHTML=`<div class="modern-diagnostic-row"><div><b>Connexion à American Truck Simulator</b><p id="telemetryDetail">Lecture possible depuis une DLL SCS compatible qui publie Local\SCSTelemetry (révision 12).</p></div><span id="telemetryBadge" class="modern-bad-status">NON CONNECTÉ</span></div><div class="modern-diagnostic-row"><div><b>Remorques et profils</b><p>Détection en lecture seule, association manuelle d'un identifiant et recharge automatique des profils reconnus.</p></div><span class="modern-pending-status">CONDITIONNEL</span></div><div class="modern-diagnostic-row"><div><b>Braquage physique des essieux</b><p>Non disponible : le SDK officiel fournit de la télémétrie, pas une API d'action sur la physique.</p></div><span class="modern-bad-status">NON VALIDÉ</span></div><div class="modern-diagnostic-row"><div><b>Mode Convoi</b><p>Les profils et les simulations sont locaux ; aucun contrôle physique synchronisé entre les joueurs n'a été validé.</p></div><span class="modern-bad-status">NON VALIDÉ</span></div><div class="modern-diagnostic-row"><div><b>Mises à jour de l'application</b><p>Le programme s'installe dans le même dossier EssieuPilot et conserve ses profils. La recherche automatique nécessite une source de publication officielle.</p></div><span class="modern-pending-status">À CONFIGURER</span></div><div class="modern-diagnostic-row"><div><b>Vue 3D</b><p>Calculée localement selon la configuration des essieux et le mouvement simulé.</p></div><span class="modern-ok-status">DISPONIBLE</span></div>`;
  pages.diagnostics.appendChild(diag);
  // Centre de mise à jour : lit les publications officielles GitHub, sans installation silencieuse forcée.
  const updates=document.createElement('div');updates.className='essieupilot-utility-panel';
  updates.innerHTML=`<div class="essieupilot-panel-title"><span class="eyebrow">ESSIEUPILOT UPDATE</span><h3>Mises à jour</h3><p>Une seule installation sur le Bureau. Les nouvelles versions peuvent remplacer l'application tout en gardant les profils enregistrés dans AppData.</p></div>
  <div class="essieupilot-utility-grid"><div class="essieupilot-metric"><small>VERSION INSTALLÉE</small><b id="updateLocalVersion">—</b></div><div class="essieupilot-metric"><small>SOURCE OFFICIELLE</small><b id="updateFeedStatus">Non configurée</b></div><div class="essieupilot-metric"><small>COMPATIBILITÉ ATS</small><b id="updateATSStatus">À vérifier</b></div></div>
  <button id="checkUpdates" class="btn btn-primary" type="button">↻ Vérifier les mises à jour</button> <button id="installUpdate" class="btn btn-secondary" type="button" disabled>↓ Télécharger et installer</button><p id="updateMessage" class="essieupilot-utility-note">Recherche depuis les publications officielles GitHub.</p><p id="latestUpdateNotes" class="essieupilot-utility-note"></p>
  <div class="essieupilot-list"><b>Ce qui pourra être mis à jour</b><div>✓ Interface, commandes et corrections du logiciel</div><div>✓ Profils et nouveaux modèles de remorques</div><div>✓ Correctifs de connexion ATS lorsqu'ils seront développés</div><div>✓ Migration automatique des réglages entre versions compatibles</div></div>`;
  pages.updates.appendChild(updates);
  const aboutExtra=document.createElement('div');aboutExtra.className='essieupilot-utility-panel';
  aboutExtra.innerHTML=`<div class="essieupilot-panel-title"><span class="eyebrow">À PROPOS</span><h3>EssieuPilot</h3><p>Pilotage intelligent des essieux — une application Windows pour préparer, visualiser et sauvegarder des réglages de remorques d'American Truck Simulator.</p></div><div class="essieupilot-list"><b>Fonctions disponibles</b><div>✓ Réglages de braquage indépendants jusqu'à 60°</div><div>✓ Clavier, boutons de manette/volant et sticks analogiques</div><div>✓ Retour au centre en mouvement, angle conservé à l'arrêt</div><div>✓ Modèles schématiques 2D et 3D par type de remorque</div><div>✓ Sauvegarde manuelle des profils et détection de télémétrie compatible</div><div>⚠ Braquage physique des roues et synchronisation Convoi non disponibles</div></div><p class="essieupilot-utility-note">Le numéro de version est réservé aux diagnostics et aux mises à jour. Le nom de l'application reste « EssieuPilot ».</p>`;
  pages.about.insertBefore(aboutExtra,pages.about.querySelector('.about-panel'));
  const atsHelp=document.createElement('div');atsHelp.className='essieupilot-utility-panel';
  atsHelp.innerHTML=`<div class="essieupilot-panel-title"><span class="eyebrow">ATS · INTÉGRATION</span><h3>Connexion au jeu</h3><p>EssieuPilot peut lire une télémétrie locale au format SCSTelemetry révision 12, publiée par une DLL SCS compatible installée dans ATS. Ce logiciel ne modifie pas la physique du jeu.</p></div>
  <div class="essieupilot-utility-grid"><div class="essieupilot-metric"><small>JEU ATS</small><b id="atsInstallation">Recherche...</b></div><div class="essieupilot-metric"><small>PLUGIN TÉLÉMÉTRIE</small><b id="atsTelemetryPlugin">Recherche...</b></div><div class="essieupilot-metric"><small>CONVOI</small><b>Non vérifié</b></div></div>
  <button id="scanATS" type="button" class="btn btn-primary">Rechercher ATS</button>
  <div class="essieupilot-list"><b>Comment obtenir la connexion en lecture seule</b><div>1. Installe une DLL de télémétrie SCS compatible dans le dossier <code>bin\win_x64\plugins</code> d'ATS.</div><div>2. Relance ATS et accepte l'avertissement SDK à son démarrage.</div><div>3. Reviens dans EssieuPilot : la vitesse et les remorques pourront être détectées si le format publié est compatible.</div><div>4. Si la télémétrie n'apparaît pas, utilise Diagnostic pour comprendre l'erreur.</div></div><p class="essieupilot-utility-note" id="atsScanNote">Le module de braquage physique indépendant n'est pas encore développé. L'utilisation en Convoi doit être testée avant toute affirmation de compatibilité.</p>`;
  pages.diagnostics.insertBefore(atsHelp,pages.diagnostics.firstChild.nextSibling);
  async function loadATSInstallations(){try{const r=await fetch('/api/ats/installations');const j=await r.json();document.getElementById('atsInstallation').textContent=j.found?'Trouvé':'Non localisé';document.getElementById('atsTelemetryPlugin').textContent=j.plugin_found?'DLL présente':'DLL non détectée';document.getElementById('atsScanNote').textContent=j.message||'';}catch(e){document.getElementById('atsScanNote').textContent='Erreur de recherche ATS : '+e.message;}}
  document.getElementById('scanATS').addEventListener('click',loadATSInstallations);loadATSInstallations();
  let latestUpdateState=null;
  async function loadUpdateState(){
    const check=document.getElementById('checkUpdates');
    const install=document.getElementById('installUpdate');
    const message=document.getElementById('updateMessage');
    check.disabled=true;install.disabled=true;message.textContent='Vérification de GitHub Releases en cours…';
    try{
      const r=await fetch('/api/updates',{cache:'no-store'});
      if(!r.ok)throw Error('Le service de mise à jour ne répond pas');
      const s=await r.json();latestUpdateState=s;
      document.getElementById('updateLocalVersion').textContent=s.version;
      document.getElementById('updateFeedStatus').textContent=s.feed_configured?'GitHub officiel':'Indisponible';
      message.textContent=s.message||'Aucun résultat disponible.';
      document.getElementById('latestUpdateNotes').textContent=s.notes ? 'Notes de publication : '+s.notes : '';
      install.disabled=!(s.available&&s.download_ready);
      if(s.available&&s.latest_version)install.textContent='↓ Installer '+s.latest_version;
    }catch(e){latestUpdateState=null;message.textContent='Erreur de recherche : '+e.message;}
    finally{check.disabled=false;}
  }
  document.getElementById('checkUpdates').addEventListener('click',loadUpdateState);
  document.getElementById('installUpdate').addEventListener('click',async()=>{
    if(!latestUpdateState?.available||!latestUpdateState.download_ready)return;
    if(!window.confirm('Installer la mise à jour officielle '+latestUpdateState.latest_version+' ? EssieuPilot devra redémarrer. Les profils seront conservés.'))return;
    const button=document.getElementById('installUpdate');button.disabled=true;
    const message=document.getElementById('updateMessage');message.textContent='Téléchargement et vérification SHA-256 en cours…';
    try{
      const r=await fetch('/api/updates/install',{method:'POST',headers:{'X-ATS-Token':document.body.dataset.token}});
      const s=await r.json();if(!r.ok)throw Error(s.error||'Échec de la mise à jour');
      message.textContent=s.message||'Installation lancée.';
    }catch(e){message.textContent='Installation impossible : '+e.message;button.disabled=false;}
  });
  loadUpdateState();

  document.getElementById('associateTrailer').addEventListener('click',()=>{
    const id=window.EssieuPilotTelemetry?.trailers?.[0]?.id;
    if(!id||!data||!getProfile()){toast('Aucune remorque détectée actuellement.',true);return;}
    getProfile().match_key=id;
    getTrailer().model_type=window.EssieuPilotTelemetry?.trailers?.[0]?.type||'générique';
    markDirty();refreshModern();toast('Remorque associée : '+id+'. Clique sur Enregistrer.');
  });
  document.getElementById('modelType').addEventListener('change',e=>{
    if(!data||!getTrailer())return;getTrailer().model_type=e.target.value;markDirty();
  });
  window.refreshEssieuTelemetry=function(t,matchText){
    const connected=!!t?.connected;
    const tr=t?.trailers?.[0];
    const status=document.getElementById('modernGameState');
    // Ne jamais écraser le voyant ATS du jeu : sa source est /api/status, pas la télémétrie SCS.
    const speed=document.getElementById('modernGameSpeed');
    if(speed)speed.textContent=connected?Math.abs(t.speed_kmh).toFixed(1)+' km/h':'— km/h';
    const detail=document.getElementById('telemetryDetail');
    if(detail)detail.textContent=t?.note||'Source de télémétrie introuvable';
    const badge=document.getElementById('telemetryBadge');
    if(badge){badge.textContent=connected?'TÉLÉMÉTRIE ACTIVE':'NON CONNECTÉ';badge.className=connected?'modern-ok-status':'modern-bad-status';}
    const detected=document.getElementById('detectedTrailer');
    if(detected)detected.textContent=tr?('Remorque : '+tr.name+' · '+tr.type+' · '+tr.estimated_axles+' essieu(x) estimés · '+matchText):'Remorque détectée : aucune';
    const side=document.getElementById('sideConnection');if(side)side.textContent=connected?'ATS CONNECTÉ (LECTURE)':'ATS NON CONNECTÉ';
    const sideDet=document.getElementById('sideConnectionDetail');if(sideDet)sideDet.textContent=connected?'Physique non pilotée':'Télémétrie SCS compatible requise';
  };
  function selectTab(id){
    if(!pages[id])return;
    Object.entries(pages).forEach(([k,p])=>p.hidden=k!==id);
    nav.querySelectorAll('button').forEach(b=>b.classList.toggle('active',b.dataset.modernTab===id));
    document.body.dataset.activeTab=id;
    main.scrollTop=0;window.scrollTo({top:0,behavior:'instant'});
    if(id==='viewer')resizeScene3D();
  }
  nav.addEventListener('click',e=>{const btn=e.target.closest('[data-modern-tab]');if(btn)selectTab(btn.dataset.modernTab);});
  document.addEventListener('click',e=>{const btn=e.target.closest('[data-modern-go]');if(btn)selectTab(btn.dataset.modernGo);});
  // Original.16: clicking an axle edits it in place inside Mes remorques.
  // app.js document delegation still selects the individual axle.
  document.getElementById('matchKey').addEventListener('change',e=>{if(typeof getProfile!=='function'||!getProfile())return;getProfile().match_key=e.target.value.trim();markDirty();});
  // Un rendu est déclenché après chaque changement de profil ou de sélection.
  const origRenderAll=renderAll;
  renderAll=function(){origRenderAll();refreshModern();};
  function refreshModern(){
    if(!data)return;
    document.getElementById('modernProfilesCount').textContent=data.profiles.length+' profil(s)';
    const match=document.getElementById('matchKey');
    if(match && document.activeElement!==match)match.value=getProfile().match_key||'';
    const model=document.getElementById('modelType');if(model && getTrailer())model.value=getTrailer().model_type||'surbaissée';
  }
  window.ATSStudioVirtualAnalog={enabled:false,value:0};
  const enable=document.getElementById('virtualAnalogEnabled');
  const range=document.getElementById('virtualAnalogRange');
  const pct=document.getElementById('virtualAnalogPct');
  const number=document.getElementById('virtualAngle');
  function refreshVirtual(){
    window.ATSStudioVirtualAnalog.enabled=enable.checked;
    range.disabled=!enable.checked;
    window.ATSStudioVirtualAnalog.value=enable.checked?Number(range.value)/100:0;
    pct.textContent=(Number(range.value)>0?'+':'')+range.value+' %';
    const profile=typeof getProfile==='function'?getProfile():null;
    const raw=Number(range.value)/100;
    const calibrated=profile?SteeringLogic.analogPower(raw,0,profile.analog_deadzone,profile.analog_curve,profile.analog_invert,profile.analog_full_scale,profile.analog_output_pct):raw;
    const axle=typeof getAxle==='function'?getAxle():null;
    const sign=axle?.reversed?-1:1;
    const angle=Math.max(-(axle?.max_angle||60),Math.min(axle?.max_angle||60,calibrated*(axle?.max_angle||60)*(axle?.ratio||1)*sign));
    number.textContent=fmt(angle)+'°';
  }
  enable.addEventListener('change',refreshVirtual);
  range.addEventListener('input',refreshVirtual);
  document.getElementById('virtualReset').addEventListener('click',()=>{range.value=0;refreshVirtual();});
  document.getElementById('virtualManual').addEventListener('click',()=>{document.getElementById('toggleManual').click();});
  refreshVirtual();
  // Rendu 3D autonome sans chargement de bibliothèques ni dépendances réseau.
  const canvas=document.getElementById('scene3d');const ctx=canvas.getContext('2d');
  const camera={yaw:.72,pitch:.72,zoom:1};let dragging=false,lastX=0,lastY=0;
  const p3=(x,y,z)=>{
    const c=Math.cos(camera.yaw),s=Math.sin(camera.yaw);
    const h=x*c+z*s,depth=-x*s+z*c;
    const vert=y*Math.cos(camera.pitch)-depth*Math.sin(camera.pitch);
    return {x:canvas.clientWidth/2+h*34*camera.zoom,y:canvas.clientHeight*.55-vert*34*camera.zoom,depth};
  };
  const color={body:'#24415a',top:'#3d6981',side:'#2b5069',roof:'#6c9faf',wheel:'#111d2b',wheelTop:'#253747',axle:'#5ce3ce',selected:'#b6a4ff',inactive:'#667589'};
  function face(points,fill,stroke){const pts=points.map(v=>p3(...v));ctx.beginPath();ctx.moveTo(pts[0].x,pts[0].y);for(let i=1;i<pts.length;i++)ctx.lineTo(pts[i].x,pts[i].y);ctx.closePath();ctx.fillStyle=fill;ctx.fill();if(stroke){ctx.strokeStyle=stroke;ctx.lineWidth=1;ctx.stroke();}}
  const polys=[];
  function poly(pts,col,stroke){const depth=pts.reduce((s,p)=>s+p3(...p).depth,0)/pts.length;polys.push({pts,col,stroke,depth});}
  function cuboid(cx,cy,cz,wx,hy,lz,front,top,side){
    const x0=cx-wx/2,x1=cx+wx/2,y0=cy,y1=cy+hy,z0=cz-lz/2,z1=cz+lz/2;
    poly([[x0,y0,z0],[x1,y0,z0],[x1,y0,z1],[x0,y0,z1]],front);
    poly([[x0,y1,z0],[x1,y1,z0],[x1,y1,z1],[x0,y1,z1]],top,'#81bdd022');
    poly([[x0,y0,z0],[x0,y1,z0],[x0,y1,z1],[x0,y0,z1]],side);
    poly([[x1,y0,z0],[x1,y1,z0],[x1,y1,z1],[x1,y0,z1]],side);
    poly([[x0,y0,z0],[x1,y0,z0],[x1,y1,z0],[x0,y1,z0]],side);
    poly([[x0,y0,z1],[x1,y0,z1],[x1,y1,z1],[x0,y1,z1]],side);
  }
  function wheel(cx,cz,angle,hi){
    const th=angle*Math.PI/180, ux=Math.sin(th)*.47,uz=Math.cos(th)*.47, vx=Math.cos(th)*.24,vz=-Math.sin(th)*.24;
    const corners=[[-ux-vx,-uz-vz],[ux-vx,uz-vz],[ux+vx,uz+vz],[-ux+vx,-uz+vz]];
    poly(corners.map(([x,z])=>[cx+x,.05,cz+z]),color.wheel);
    poly(corners.map(([x,z])=>[cx+x,.33,cz+z]),hi,'#dafcf72a');
  }
  function draw3d(){
    const w=canvas.clientWidth,h=canvas.clientHeight;if(w<=0||h<=0)return;
    const dpr=Math.min(devicePixelRatio||1,2),rw=Math.floor(w*dpr),rh=Math.floor(h*dpr);
    if(canvas.width!==rw||canvas.height!==rh){canvas.width=rw;canvas.height=rh;}
    ctx.setTransform(dpr,0,0,dpr,0,0);
    const bg=ctx.createLinearGradient(0,0,w,h);bg.addColorStop(0,'#0a1b2b');bg.addColorStop(1,'#0e2533');ctx.fillStyle=bg;ctx.fillRect(0,0,w,h);
    ctx.lineWidth=1;ctx.strokeStyle='#40647a35';
    for(let a=-18;a<=18;a+=2){let aa=p3(a,0,-18),bb=p3(a,0,18);ctx.beginPath();ctx.moveTo(aa.x,aa.y);ctx.lineTo(bb.x,bb.y);ctx.stroke();aa=p3(-18,0,a);bb=p3(18,0,a);ctx.beginPath();ctx.moveTo(aa.x,aa.y);ctx.lineTo(bb.x,bb.y);ctx.stroke();}
    polys.length=0;
    if(data && getProfile()){
      const ts=getProfile().trailers;
      const len=5.6,gap=.65;
      const start=-(ts.length-1)*(len+gap)/2;
      cuboid(0,.4,start-len*.67,1.9,1.5,2.25,color.body,color.roof,color.side);
      cuboid(0,.31,start-len*.22,1.5,.24,1.3,'#3e6877','#5c8594','#39606e');
      ts.forEach((t,i)=>{
        const z=start+i*(len+gap),axles=t.axles||[];
        const type=t.model_type||((ts.length===1&&window.EssieuPilotTelemetry?.trailers?.[0]?.type)||'surbaissée');
        if(type==='surbaissée'){
          cuboid(0,.25,z,2.65,.17,len-.24,color.body,color.top,color.side);
          cuboid(0,.42,z-len*.42,2.4,.44,.7,'#274a5e','#508ea1','#305c69');
          cuboid(0,.42,z+len*.42,2.5,.36,.7,'#284e5f','#4b8498','#284a5c');
        }else if(type==='fermée'){
          cuboid(0,.35,z,2.65,1.85,len-.24,'#304858','#5d8295','#2a4b5e');
        }else if(type==='citerne'){
          cuboid(0,.40,z,2.25,.95,len-.55,'#244755','#7ca1b3','#3a6576');
        }else if(type==='forestière'){
          cuboid(0,.47,z,2.35,.2,len-.24,color.body,color.top,color.side);
          for(const zc of [-1.6,-.4,.8,1.9])cuboid(0,.48,z+zc,2.4,.75,.11,'#315968','#6d9ead','#315968');
        }else{
          cuboid(0,.47,z,2.65,.3,len-.24,color.body,color.top,color.side);
          cuboid(0,.78,z-.5,2.15,.10,len-1.4,'#39536a','#557d92','#31536a');
        }
        const rowStep=axles.length>1?Math.min(.9,3.4/(axles.length-1)):0;
        axles.forEach((a,j)=>{
          const az=z+len*.5-.6-j*rowStep;
          const select=a.id===selectedAxleId;
          const tint=!a.enabled?color.inactive:select?color.selected:color.axle;
          cuboid(0,.25,az,2.6,.13,.13,'#5d7f8e','#7ca0ad','#446478');
          wheel(-1.42,az,a.current_angle||0,tint);
          wheel(1.42,az,a.current_angle||0,tint);
        });
      });
    }
    polys.sort((a,b)=>b.depth-a.depth);
    polys.forEach(v=>face(v.pts,v.col,v.stroke));
    ctx.fillStyle='#a7bdcc';ctx.font='11px Segoe UI,sans-serif';ctx.fillText('MAQUETTE 3D · VISUALISATION APPROXIMATIVE',20,h-20);
  }
  function resizeScene3D(){draw3d();}
  canvas.addEventListener('pointerdown',e=>{dragging=true;lastX=e.clientX;lastY=e.clientY;canvas.setPointerCapture(e.pointerId);});
  canvas.addEventListener('pointermove',e=>{if(!dragging)return;camera.yaw+=(e.clientX-lastX)*.007;camera.pitch=Math.max(.18,Math.min(1.46,camera.pitch-(e.clientY-lastY)*.006));lastX=e.clientX;lastY=e.clientY;});
  ['pointerup','pointercancel'].forEach(n=>canvas.addEventListener(n,()=>{dragging=false;}));
  canvas.addEventListener('wheel',e=>{e.preventDefault();camera.zoom=Math.max(.35,Math.min(1.9,camera.zoom*(e.deltaY>0?.9:1.1)));},{passive:false});
  pages.viewer.querySelectorAll('[data-camera]').forEach(b=>b.addEventListener('click',()=>{switch(b.dataset.camera){case'top':camera.yaw=0;camera.pitch=1.48;break;case'side':camera.yaw=1.57;camera.pitch=.25;break;case'reset':camera.yaw=.72;camera.pitch=.72;camera.zoom=1;break;default:camera.yaw=.72;camera.pitch=.72;}}));
  let lastNumbers=0;
  function frame(now){
    if(!pages.viewer.hidden){draw3d();if(now-lastNumbers>130){
      lastNumbers=now;
      const t=getTrailer();
      if(t){document.getElementById('modern3DAxles').innerHTML=t.axles.map((a,i)=>`<div><span>${escapeHTML(a.name)}</span><b>${fmt(a.current_angle)}°</b></div>`).join('');}
      document.getElementById('virtualManual').textContent=manual?'Désactiver le pilotage':'Activer le pilotage';
      if(!enable.checked)number.textContent=fmt(getAxle()?.current_angle||0)+'°';
    }}
    requestAnimationFrame(frame);
  }
  refreshModern();
  selectTab('dashboard');
  requestAnimationFrame(frame);
})();
