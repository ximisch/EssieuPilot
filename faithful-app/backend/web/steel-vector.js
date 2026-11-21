'use strict';
/* EssieuPilot Steel Vector. Display layer ONLY on top of Original.15.
 * Every existing profile, setting, binding, telemetry and ATS handler is retained.
 * Statuses are based on reported API data, never fabricated from the mockup. */
(function(){
  const one=(s,r=document)=>r.querySelector(s);
  function start(){
    const page=id=>one('.modern-page[data-page="'+id+'"]');
    const dashboard=page('dashboard'),trailers=page('trailers');
    const inputs=page('inputs'),viewer=page('viewer');
    if(!dashboard||!trailers||!inputs||!viewer)return;
    if(document.body.classList.contains('sv3'))return;
    document.body.classList.add('sv3');
    // The application has seven visible destinations. Essieu settings stay
    // fully editable INSIDE Mes remorques rather than a hidden eighth menu.
    const nav=one('.studio-nav');
    nav?.querySelector('[data-modern-tab="axles"]')?.remove();
    const labels={
      trailers:['Mes remorques','▱'],inputs:['Commande','⌘'],
      viewer:['Simulation','⬡'],diagnostics:['Diagnostique','⌁'],
      updates:['Mises à jour','↻'],about:['À propos','ⓘ']};
    for(const [id,[label,ico]] of Object.entries(labels)){
      const b=nav?.querySelector('[data-modern-tab="'+id+'"]');
      if(b){const x=one('strong',b),y=one('.modern-nav-icon',b);if(x)x.textContent=label;if(y)y.textContent=ico;}
    }
    const profiles=Array.from(document.querySelectorAll('.sidebar .side-section')).find(x=>one('#profiles',x));
    if(profiles){profiles.classList.add('sv-profiles');profiles.classList.remove('ep16-legacy-hidden');profiles.querySelector('.side-title')?.setAttribute('aria-label','Profils de remorques');trailers.insertBefore(profiles,trailers.querySelector('.axle-panel'));}
    // Move only DOM nodes: this preserves listeners and state attached by
    // app.js + modern.js. Do not recreate any form controls.
    const move=(selector,target)=>{const n=one(selector);if(n)target.appendChild(n);};
    move('.settings',trailers);
    move('.controls',inputs);
    move('.visualize',viewer);
    // Put the bindings and manual steering tools in a single responsive grid.
    // This prevents inherited legacy grid-row rules from overlapping the panels.
    const bindings=inputs.querySelector('.binding-panel');
    const directControls=inputs.querySelector('.controls');
    if(bindings&&directControls){
      const group=document.createElement('div');
      group.className='sv-input-grid';
      bindings.before(group);
      group.append(bindings,directControls);
    }
    // Keep the original save/import/export actions available even though
    // the old header is visually replaced by the Steel Vector cockpit.
    const tools=document.createElement('div');
    tools.className='sv-action-tools';
    for(const [label,action] of [['＋ Nouvelle remorque','add-trailer'],
                                  ['⇩ Importer','import'],['⇧ Exporter','export'],
                                  ['▣ Enregistrer','save']]){
      const button=document.createElement('button');button.type='button';
      button.className=action==='save'?'btn btn-primary':'btn btn-ghost';
      button.dataset.action=action;button.textContent=label;tools.append(button);
    }
    trailers.append(tools);
    // Sharp vector schematic of the CONFIGURED axles, never a claim about the
    // wheels or trailers actually identified by ATS.
    const axlePanel=trailers.querySelector('.axle-panel');
    const selector=axlePanel?.querySelector('.trailer-select');
    let profileSketch=null;
    if(axlePanel&&selector){
      profileSketch=document.createElement('div');
      profileSketch.className='sv-trailer-art';
      profileSketch.setAttribute('role','img');
      profileSketch.setAttribute('aria-label','Schéma de la remorque et de ses essieux configurés');
      selector.insertAdjacentElement('afterend',profileSketch);
    }
    let priorCount=-1;
    function refreshSketch(){
      if(!profileSketch)return;
      const raw=parseInt(document.querySelector('#countAxles')?.textContent||'0',10);
      const count=Number.isFinite(raw)?Math.max(0,Math.min(raw,32)):0;
      if(count===priorCount)return;priorCount=count;
      const wheelCount=Math.min(count,16);
      const first=530,last=850;
      let wheels='';
      for(let i=0;i<wheelCount;i++){
        const x=wheelCount===1?last:first+(last-first)*i/(wheelCount-1);
        wheels+='<circle cx="'+x.toFixed(0)+'" cy="117" r="14" fill="#041924" stroke="#06dfed" stroke-width="3"/>'+
          '<circle cx="'+x.toFixed(0)+'" cy="117" r="6" fill="#057d99" stroke="#a7ebf1" stroke-width="1.5"/>';
      }
      profileSketch.innerHTML='<div class="sv-trailer-art-title">Remorque configurée <span>Schéma vectoriel · '+count+' essieu(x)</span></div>'+
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 930 150" role="presentation" aria-hidden="true">'+
        '<g fill="none" stroke="#06ddee" stroke-width="3" stroke-linejoin="round">'+
        '<path d="M28 107 L28 52 L119 52 L119 33 L217 33 L217 89 L320 89 L360 75 L890 75 L890 100 L905 100 L905 110 L359 110 L322 120 L28 120 Z"/>'+
        '<path d="M119 54 L217 54 M340 91 L875 91 M28 77 L214 77" stroke-width="1.4" opacity=".55"/>'+
        '<path d="M62 122 L62 132 M86 122 L86 132 M110 122 L110 132" stroke="#88dce8" stroke-width="2"/>'+
        '</g><path d="M320 120 L360 110 L892 110" fill="none" stroke="#fa8a13" stroke-width="2" opacity=".75"/>'+
        wheels+'</svg>';
    }
    refreshSketch();
    const axlePage=page('axles');
    if(axlePage)axlePage.setAttribute('aria-hidden','true');
    const titles={dashboard:'Tableau de bord',trailers:'Mes remorques',
      inputs:'Commande',viewer:'Simulation',diagnostics:'Diagnostique',
      updates:'Mises à jour',about:'À propos'};
    for(const [id,label] of Object.entries(titles)){
      const p=page(id);const heading=p&&one('.modern-section-intro h2',p);
      if(heading)heading.textContent=label;
    }
    const make=(tag,cls,content)=>{
      const e=document.createElement(tag);
      if(cls)e.className=cls;if(content!==undefined)e.textContent=content;return e;
    };
    const board=make('div','sv-dashboard');
    const headline=make('header','sv-headline');
    const headingText=make('div');
    headingText.append(make('h1','','// Tableau de bord'),
      make('p','','Vue d’ensemble du système, de la remorque active et des essieux.'));
    headline.append(headingText);

    const heroRow=make('div','sv-hero-row');
    const frame=make('div','sv-photo-frame');
    const image=make('img');
    image.src='/steel-vector-hero.webp';image.alt='Remorque surbaissée EssieuPilot — panorama';
    image.decoding='async';image.loading='eager';
    // Use the available original photo without stretching pixels; on wide
    // monitors the surrounding frame remains vector/CSS crisp.
    frame.appendChild(image);
    const watermark=make('span','sv-photo-caution','Illustration panoramique • aucune mesure physique inventée');
    frame.appendChild(watermark);
    const states=make('div','sv-status-column');
    function statusCard(title,sub,id){
      const box=make('section','sv-status');
      const h=make('h3');const symbol=make('span','',id==='sv-game'?'⌁':'◉');
      h.append(symbol,document.createTextNode(title));
      const a=make('p','sv-state-main');
      a.append(make('i','sv-lamp'),make('span','sv-value',sub));
      const b=make('p','sv-hint','Vérification en cours');
      box.id=id;box.dataset.state='unknown';box.append(h,a,b);
      return box;
    }
    const game=statusCard('État du jeu','État inconnu','sv-game');
    const axles=statusCard('Essieux en jeu','État inconnu','sv-axles');
    states.append(game,axles);heroRow.append(frame,states);
    const metrics=make('div','sv-metrics');
    const cells=[
      ['Remorque active','sv-remorque'],['Essieux configurés','sv-axle-count'],
      ['Profils enregistrés','sv-profile-count'],['Angle simulé','sv-angle'],
      ['Mode','sv-mode']];
    for(const [label,id] of cells){
      const el=make('div','sv-metric');
      const strong=make('strong','sv-unknown','—');strong.id=id;
      el.append(make('small','',label),strong);metrics.append(el);
    }
    const action=make('div','sv-cta-row');
    action.append(make('span','sv-note','Les valeurs indiquées sont observées ou simulées : elles ne prouvent pas le mouvement physique des roues.'));
    const launch=make('button','sv-launch','▶ Ouvrir la simulation');
    launch.type='button';launch.dataset.modernGo='viewer';action.append(launch);
    board.append(headline,heroRow,metrics,action);
    dashboard.append(board);
    const text=(id,value)=>{
      const n=document.getElementById(id);
      if(n){n.textContent=String(value??'—');n.classList.toggle('sv-unknown',!value||value==='—');}
    };
    function drawValues(){
      const current=one('#trailerSelect');
      text('sv-remorque',current?.selectedOptions?.[0]?.textContent?.trim()||'Aucune');
      text('sv-axle-count',one('#countAxles')?.textContent?.trim()||'—');
      text('sv-profile-count',one('#modernProfilesCount')?.textContent?.trim()||'—');
      text('sv-angle',one('#angleBadge')?.textContent?.trim()||'—');
      const manual=one('#manualStatus')?.textContent?.trim()||'Non disponible';
      text('sv-mode',manual);
    }
    async function refreshStatuses(){
      const value=one('#modernGameState')?.textContent?.trim()||'';
      const on=/en ligne|jeu détecté|connecté/i.test(value)&&!/non connecté|hors ligne|non détecté/i.test(value);
      const off=/non connecté|hors ligne|non détecté/i.test(value);
      game.dataset.state=on?'on':off?'off':'unknown';
      one('.sv-value',game).textContent=on?'En ligne':off?'Hors ligne':'État inconnu';
      one('.sv-hint',game).textContent=on?'ATS détecté par le programme':off?'ATS non détecté':'Vérification du processus ATS';
      let active=false,known=false,detail='Aucune preuve de commande DLL';
      try{
        const [sr,cr]=await Promise.all([
          fetch('/api/status',{cache:'no-store'}),
          fetch('/api/control-link',{cache:'no-store'})
        ]);
        if(sr.ok&&cr.ok){
          const [s,c]=await Promise.all([sr.json(),cr.json()]);
          known=true;
          active=s.game_running===true&&c.manual_armed===true&&
            c.physical_control_enabled===true&&c.physical_callback_ack===true;
          detail=active?'Commande signalée par la DLL ; mouvement réel non prouvé':
            'Commande physique non confirmée';
        }
      }catch(_e){detail='Connexion aux diagnostics indisponible';}
      axles.dataset.state=known?(active?'on':'off'):'unknown';
      one('.sv-value',axles).textContent=known?(active?'Commande active':'Inactifs'):'État inconnu';
      one('.sv-hint',axles).textContent=detail;
      drawValues();
      refreshSketch();
    }
    const preview=new URLSearchParams(window.location.search).get('sv-preview');
    if(['trailers','inputs','viewer','diagnostics','updates','about'].includes(preview)){
      requestAnimationFrame(()=>document.querySelector('[data-modern-tab="'+preview+'"]')?.click());
    }
    drawValues();refreshStatuses();
    const schedule=setInterval(refreshStatuses,2500);
    document.addEventListener('visibilitychange',()=>{
      if(!document.hidden)refreshStatuses();
    });
    // The current seven-page nav retains existing modern.js handlers.
    // The simulation drawings are rendered at the monitor devicePixelRatio.
  }
  if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',start,{once:true});
  else start();
})();