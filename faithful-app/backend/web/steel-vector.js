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
    const heroRow=make('div','sv-hero-row');
    const frame=make('div','sv-photo-frame');
    const image=make('img');
    image.src='/hero_reference.jpg';image.alt='Remorque surbaissée EssieuPilot — panorama';
    image.decoding='async';image.loading='eager';
    // Use the available original photo without stretching pixels; on wide
    // monitors the surrounding frame remains vector/CSS crisp.
    frame.appendChild(image);
    const watermark=make('span','sv-photo-caution','Illustration • mesures ATS non simulées');
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
    board.append(heroRow,metrics,action);
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