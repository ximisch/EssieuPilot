'use strict';
/* EssieuPilot Original.18 – presentation only. Do not replace backend state,
   navigation actions, saved profiles, physical safety gates or updater handlers. */
(function () {
  function initialize() {
    const home=document.querySelector('.ep16-home');
    if(!home || document.querySelector('.ep18-page-head')) return;
    document.body.classList.add('ep18');
    const oldHero=home.querySelector('.ep16-home-hero');
    const leds=oldHero?.querySelector('.ep16-hero-statuses');
    if(!oldHero || !leds) return;
    const element=(tag,cl,text)=>{
      const node=document.createElement(tag);
      if(cl)node.className=cl;
      if(text!==undefined)node.textContent=text;
      return node;
    };
    const title=element('header','ep18-page-head');
    const bars=element('span','ep18-bars','▮▮▮');
    const titleText=element('div');
    titleText.append(element('h1','', 'Tableau de bord'),
      element('p','', 'Contrôle et suivi de la direction de vos remorques'));
    title.append(bars,titleText);
    const layout=element('div','ep18-main-grid');
    const panorama=element('section','ep18-panorama');
    panorama.append(oldHero);
    const statusColumn=element('aside','ep18-side-status');
    const ledElements=[...leds.querySelectorAll('.ep16-led')];
    for(const led of ledElements) {
      const wrap=element('section','ep18-status-card');
      wrap.append(led);
      statusColumn.append(wrap);
    }
    leds.remove();
    layout.append(panorama,statusColumn);
    const info=element('div','ep18-info');
    const axleCard=element('section','ep18-card');
    axleCard.append(element('h3','','◉ État des essieux'));
    const axleText=element('strong','', 'En attente de confirmation');
    axleText.style.fontSize='17px';
    axleCard.append(axleText);
    const angleCard=element('section','ep18-card');
    angleCard.append(element('h3','','◉ Angle de braquage'));
    const angleText=element('strong','ep18-angle','—');
    angleCard.append(angleText,element('span','ep18-muted','Angle réel non confirmé par la télémétrie ATS'));
    info.append(axleCard,angleCard);
    const metrics=element('section','ep18-metrics ep18-card');
    metrics.append(element('h3','','▮ Indicateurs clés'));
    const metricItems=element('div','ep18-metrics-items');
    function metric(label,sourceId,fallback){
      const pane=element('div','ep18-metric');
      pane.append(element('small','',label));
      const val=element('strong','',fallback);
      pane.append(val);metricItems.append(pane);
      const source=document.getElementById(sourceId);
      if(source){
        const update=()=>{val.textContent=source.textContent||fallback;};
        update();new MutationObserver(update).observe(source,{characterData:true,childList:true,subtree:true});
      }
    }
    metric('Remorque active','ep16HomeTrailer','Aucune remorque');
    metric('Essieux configurés','ep16NumAxles','—');
    metric('Profils enregistrés','ep16NumProfiles','—');
    metric('Mode de direction','ep16HomeFooterState','État à vérifier');
    metrics.append(metricItems);
    home.prepend(title,layout,info,metrics);
    const axleLed=document.getElementById('ep16AxleStatus');
    if(axleLed){
      const render=()=>{
        axleText.textContent=axleLed.classList.contains('on')?'Actifs (retour DLL requis)':
          axleLed.classList.contains('off')?'Inactifs':'État inconnu';
        axleText.style.color=axleLed.classList.contains('on')?'#58ff49':'#ffab69';
      };
      render();new MutationObserver(render).observe(axleLed,{attributes:true,attributeFilter:['class']});
    }
    // Measurements intentionally remain unknown: never turn a simulated value
    // into a claim about physical wheels in American Truck Simulator.
    angleText.title='Aucun angle physique fiable disponible';
  }
  if(document.readyState==='loading') document.addEventListener('DOMContentLoaded',initialize,{once:true});
  else initialize();
})();
