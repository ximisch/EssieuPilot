'use strict';
// Original 20: read-only, signed two-direction diagnostic. Steel Vector-only UI.
// Zero changes to ATS offsets or steering physics and no automatic calibration.
(function(){
  function init(){
    const page=document.querySelector('.modern-page[data-page="diagnostics"]');
    if(!page||document.querySelector('#ep20DirectionReport'))return;
    const block=document.createElement('section');
    block.id='ep20DirectionReport';
    block.className='sv20-panel';
    block.innerHTML=`
      <div class="sv20-heading">
        <div><span class="sv20-kicker">ORIGINAL 20 · ESSAI PROGRESSIF</span>
          <h3>Diagnostic gauche / droite</h3>
          <p>Teste les deux directions dans ATS, puis reviens ici. Les résultats de chaque côté restent enregistrés.</p>
        </div>
        <a href="/api/physical-diagnostic" download="EssieuPilot_Diagnostic_Gauche_Droite.json" class="sv20-export">Télécharger le diagnostic</a>
      </div>
      <div class="sv20-states">
        <div class="sv20-side"><strong>◀ Gauche</strong><span id="sv20Left" class="sv20-value">Pas encore testée</span><small id="sv20LeftDetail">Aucune commande enregistrée pendant ATS.</small></div>
        <div class="sv20-side"><strong>Droite ▶</strong><span id="sv20Right" class="sv20-value">Pas encore testée</span><small id="sv20RightDetail">Aucune commande enregistrée pendant ATS.</small></div>
      </div>
      <div id="sv20Input" class="sv20-observe">Lecture des commandes configurées…</div>
      <p id="sv20Autoload" class="sv20-note">Réglages automatiques : les profils déjà associés à un identifiant ATS sont rechargés sans inventer les limites physiques.</p>
      <p class="sv20-caution">Un paquet signé et un callback DLL ne prouvent pas le mouvement réel des roues. Le nombre d'essieux directeurs et leur géométrie ne sont pas déduits automatiquement du SDK ATS.</p>`;
    page.insertBefore(block,page.firstChild);
    const $=id=>document.getElementById(id);
    let busy=false;
    function renderSide(side,id,detailId,expectedSign){
      const a=$(id),b=$(detailId);
      if(!a||!b)return;
      if(!side){a.textContent='Pas encore testée';b.textContent='Effectue un essai dans ATS avec la touche configurée.';a.dataset.state='missing';return;}
      const sign=Number(side.normalized_ipc_angle||0);
      const target=Number(side.target_angle_degrees||0);
      const sent=side.ipc_packet_sent===true;
      const good=sent&&Math.abs(sign)>0.0005&&Math.sign(sign)===expectedSign;
      const opposite=sent&&Math.abs(sign)>0.0005&&Math.sign(sign)!==expectedSign;
      a.dataset.state=good?'ok':opposite?'reversed':'missing';
      a.textContent=good?'Consigne envoyée':opposite?'Signe opposé':'Commande à vérifier';
      b.textContent='Cible '+target.toFixed(1)+'° · IPC '+(sign>0?'+':'')+sign.toFixed(3)+
        ' · '+(sent?'paquet transmis':'paquet non transmis')+
        ' · '+(side.dll_callback_ack?'callback DLL reçu (pas une preuve des roues)':'callback DLL non confirmé')+
        (side.blockers?.length?' · blocages : '+side.blockers.join(', '):'');
    }
    async function update(){
      if(busy)return;busy=true;
      try{
        const [configResult,diagResult]=await Promise.all([
          fetch('/api/control-link',{cache:'no-store'}),
          fetch('/api/physical-diagnostic',{cache:'no-store'})
        ]);
        if(configResult.ok){
          const c=await configResult.json();
          const label=v=>String(v||'non définie').replace(/^KEY:/,'');
          $('sv20Input').textContent=
            'Activation : '+label(c.toggle_binding)+' · Gauche : '+label(c.left_binding)+
            ' · Droite : '+label(c.right_binding)+
            ' · Mode : '+(c.manual_armed?'ACTIVÉ':'INACTIF')+
            ' · Touches détectées (G/D) : '+(c.left_count||0)+' / '+(c.right_count||0)+
            ' · Commandes autorisées (G/D) : '+(c.allowed_left_count||0)+' / '+(c.allowed_right_count||0);
          const selected=String(c.physical_selection_reason||'');
          const recognized=selected==='ASSOCIATION_AUTOMATIQUE_EXACTE';
          const trailer=c.physical_detected_trailer_id||'aucune';
          const configured=c.physical_selected_trailer_id||'aucun';
          $('sv20Autoload').textContent='Remorque ATS : '+trailer+' · Réglage : '+configured+
            (recognized?' · profil reconnu automatiquement':
              ' · association non confirmée : '+(selected||'à vérifier'))+
            '. Les angles sont des réglages enregistrés, pas une lecture fiable de la géométrie.';
        }
        if(diagResult.ok){
          const d=await diagResult.json();
          renderSide(d.last_with_left_input,'sv20Left','sv20LeftDetail',-1);
          renderSide(d.last_with_right_input,'sv20Right','sv20RightDetail',1);
        }else{
          renderSide(null,'sv20Left','sv20LeftDetail',-1);
          renderSide(null,'sv20Right','sv20RightDetail',1);
        }
      }catch(_e){
        $('sv20Input').textContent='Diagnostic temporairement indisponible.';
      }finally{busy=false;}
    }
    update();setInterval(update,1800);
    document.addEventListener('visibilitychange',()=>{if(!document.hidden)update();});
  }
  if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',init,{once:true});
  else init();
})();
