'use strict';
// Original21 / Convoi: observational panel only; no remote player modification.
// A nonzero SDK time offset is multiplayer evidence. Zero is NOT Solo proof.
(function(){
  function start(){
    const page=document.querySelector('.modern-page[data-page="diagnostics"]');
    if(!page||document.getElementById('ep21Convoy'))return;
    const box=document.createElement('section');
    box.className='ep21-convoy';
    box.id='ep21Convoy';
    box.innerHTML=`
      <header class="ep21-heading">
        <div><small>ORIGINAL 21 · CONVOI</small><h3>Compatibilité multijoueur</h3>
          <p>Diagnostic du mode Convoi d'ATS, sans intervention sur les remorques d'autres joueurs.</p></div>
        <span id="ep21State" role="status" aria-live="polite">Vérification…</span>
      </header>
      <div class="ep21-values">
        <div><span>Jeu ATS</span><strong id="ep21Game">Inconnu</strong></div>
        <div><span>Télémétrie Convoi</span><strong id="ep21Evidence">Indéterminée</strong></div>
        <div><span>Sécurité physique</span><strong id="ep21Safety">À vérifier</strong></div>
      </div>
      <p id="ep21Detail" class="ep21-detail">Lecture de la DLL ATS…</p>
      <div class="ep21-important">
        <strong>Ce que cette version permet</strong>
        <p>Utiliser EssieuPilot pour lire les informations ATS, conserver les profils et contrôler la sécurité Convoi. Dès qu'un indice de multijoueur est détecté, la DLL suspend le braquage expérimental jusqu'au prochain démarrage d'ATS.</p>
        <p><strong>Non validé :</strong> braquage physique en Convoi, synchronisation des roues chez les autres joueurs et compatibilité TruckersMP. Un offset égal à zéro ne suffit pas à prouver le mode Solo.</p>
      </div>
      <div class="ep21-actions">
        <button type="button" id="ep21Download">Exporter le diagnostic Convoi</button>
        <span>Test recommandé : session privée, même version ATS et mêmes mods chez les participants ; vérification de la télémétrie avant les essais physiques.</span>
      </div>`;
    const title=page.querySelector('.modern-section-intro');
    if(title)title.insertAdjacentElement('afterend',box);
    else page.prepend(box);
    const $=id=>document.getElementById(id);
    let last=null,updating=false;
    async function refresh(){
      if(updating)return;updating=true;
      try{
        const r=await fetch('/api/convoy-probe',{cache:'no-store'});
        if(!r.ok)throw Error('État multijoueur indisponible');
        const data=await r.json();
        if(data.physical_convoy_supported!==false)throw Error('Compatibilité physique Convoi non vérifiée');
        last=data;
        const p=data.report||{};
        $('ep21Game').textContent=data.game_running?'ATS lancé':'ATS non détecté';
        $('ep21Evidence').textContent=p.multiplayer_evidence?'Indice multijoueur observé':
           p.sdk_channel_seen?'Offset nul — non concluant':'État non déterminé';
        $('ep21Safety').textContent=p.native_safety_active?'Intervention DLL suspendue':
          'Aucune suspension multijoueur confirmée';
        $('ep21Detail').textContent=String(p.status||'Sonde indisponible')+
          '. Le pilotage physique en Convoi reste non validé.';
        const state=$('ep21State');
        state.textContent=p.multiplayer_evidence?'Indice multijoueur détecté':
          p.available?'Télémétrie partielle':'Signal indisponible';
        state.dataset.state=p.multiplayer_evidence?'multi':p.available?'unknown':'offline';
      }catch(e){
        $('ep21Detail').textContent='Lecture impossible : '+e.message;
        $('ep21State').textContent='Diagnostic indisponible';
        $('ep21State').dataset.state='offline';
      }finally{updating=false}
    }
    $('ep21Download').addEventListener('click',()=>{
      if(!last)return;
      const data=JSON.stringify({capture:new Date().toISOString(),...last},null,2);
      const url=URL.createObjectURL(new Blob([data],{type:'application/json'}));
      const a=document.createElement('a');a.href=url;a.download='EssieuPilot_Convoi_Original21.json';
      document.body.append(a);a.click();a.remove();setTimeout(()=>URL.revokeObjectURL(url),1000);
    });
    refresh();setInterval(refresh,1500);
    document.addEventListener('visibilitychange',()=>{if(!document.hidden)refresh()});
  }
  if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',start,{once:true});
  else start();
})();
