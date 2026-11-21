'use strict';

const $ = (id) => document.getElementById(id);
const token = document.body.dataset.token;
const clamp = (v, min, max) => Math.min(max, Math.max(min, v));
const uid = () => 'id-' + (crypto.randomUUID ? crypto.randomUUID().slice(0, 13) : Math.random().toString(36).slice(2, 15));
const deepCopy = value => JSON.parse(JSON.stringify(value));
const fmt = n => (Number.isFinite(Number(n)) ? Number(n) : 0).toFixed(1).replace('.', ',');
const escapeHTML = text => String(text).replace(/[&<>"']/g, ch => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));

let data = null;
let selectedTrailerId = null;
let selectedAxleId = null;
let dirty = false;
let manual = false;
let mode = 'stop';
let telemetry = null;
let telemetryTrailerKey = '';
let telemetryMatchText = 'En attente de télémétrie';
let leftDown = false;
let rightDown = false;
let analogSteering = 0;
let steeringState = 'maintien';
let keyboard = new Set();
let devices = [];
let prevDevices = [];
let captureField = null;
let captureBaselines = null;
let lastTogglePressed = false;
let angleTrail = [];
let vehicle = {x:0,y:0,yaw:0};
let lastFrame = performance.now();
let lastUiDraw = 0;
let frameId = null;

const getProfile = () => data?.profiles?.find(p => p.id === data.active_id) ?? data?.profiles?.[0];
const getTrailer = () => getProfile()?.trailers?.find(t => t.id === selectedTrailerId) ?? getProfile()?.trailers?.[0];
const getAxle = () => getTrailer()?.axles?.find(a => a.id === selectedAxleId) ?? getTrailer()?.axles?.[0];
const allAxles = () => (getProfile()?.trailers ?? []).flatMap(t => t.axles ?? []);
const makeAxle = (index) => ({id:uid(),name:`Essieu directeur ${index}`,enabled:true,max_angle:60,turn_rate:14,center_rate:7,turn_instant:false,center_instant:false,ratio:1,reversed:false,current_angle:0});
const makeTrailer = (n) => ({id:uid(),name:`Semi-remorque ${n}`,model_type:'surbaissée',ats_match_key:'',physical_full_scale_degrees:60,axles:[makeAxle(1),makeAxle(2),makeAxle(3)]});
const makeProfile = (name) => ({id:uid(),name,toggle_key:'KEY:F8',left_key:'KEY:F9',right_key:'KEY:F10',analog_axis:'',analog_axis_center:0,analog_invert:false,analog_deadzone:0.08,analog_curve:1,analog_full_scale:100,analog_output_pct:100,analog_mode:'position',match_key:'',auto_center_moving:true,center_speed_gain:1,trailers:[makeTrailer(1)]});

function toast(message, error=false) {
  const t = $('toast');
  t.textContent = message;
  t.className = 'toast' + (error ? ' error' : '');
  clearTimeout(toast.timer);
  toast.timer = setTimeout(() => {t.className = 'toast hidden';}, 4500);
}
function markDirty(){ dirty = true; showSaveStatus(); }
function showSaveStatus(text='') {
  const el = $('saveState');
  el.className = 'save-state ' + (dirty ? 'dirty' : 'saved');
  el.textContent = text || (dirty ? '● Modifications non enregistrées' : '✓ Profils enregistrés');
}
function normalizeData(input) {
  if (!input || Number(input.schema) !== 1 || !Array.isArray(input.profiles) || !input.profiles.length || input.profiles.length > 100) throw new Error('Format de profils invalide.');
  for(const p of input.profiles) {
    if(!Array.isArray(p.trailers) || !p.trailers.length || p.trailers.length > 20) throw new Error('Remorques invalides dans le profil.');
    p.id ||= uid(); p.name ||= 'Profil sans nom';
    p.toggle_key ||= 'KEY:F8'; p.left_key ||= 'KEY:F9'; p.right_key ||= 'KEY:F10';
    if(p.auto_center_forward === undefined) p.auto_center_forward = true;
    // Migration silencieuse des profils V1.2 : on conserve l'ancien choix.
    if(p.auto_center_moving === undefined) p.auto_center_moving = p.auto_center_forward !== false;
    p.analog_axis ||= '';
    p.match_key ||= '';
    p.analog_axis_center = clamp(Number(p.analog_axis_center) || 0,-.95,.95);
    p.analog_invert = !!p.analog_invert;
    p.center_speed_gain=clamp(Number(p.center_speed_gain ?? 1),0,4);
    p.analog_deadzone = clamp(Number(p.analog_deadzone ?? .08),0,.4);
    // Les profils des versions 1.3 utilisaient une courbe de vitesse.
    // Le nouveau mode de position est lineaire par defaut.
    if(p.analog_mode !== 'position' && Number(p.analog_curve) === 1.35) p.analog_curve = 1;
    p.analog_mode = 'position';
    p.analog_curve = clamp(Number(p.analog_curve ?? 1),.5,3);
    p.analog_full_scale = clamp(Number(p.analog_full_scale ?? 100),20,100);
    p.analog_output_pct = clamp(Number(p.analog_output_pct ?? 100),0,100);
    for(const t of p.trailers) {
      if(!Array.isArray(t.axles) || !t.axles.length || t.axles.length > 32) throw new Error('Essieux invalides dans la remorque.');
      t.id ||= uid(); t.name ||= 'Semi-remorque'; t.model_type ||= 'surbaissée';
      // Preserve the verified ATS model association across Import / Save.
      // Never infer an ID from the human-readable trailer name.
      t.ats_match_key=String(t.ats_match_key||'').trim().slice(0,255);
      const physicalScale=Number(t.physical_full_scale_degrees);
      t.physical_full_scale_degrees=Number.isFinite(physicalScale)&&physicalScale>=35&&physicalScale<=120?
        physicalScale:60;
      for(const a of t.axles) {
        a.id ||= uid(); a.name ||= 'Essieu directeur';
        a.enabled = a.enabled !== false;
        a.reversed = !!a.reversed;
        a.max_angle = clamp(Number(a.max_angle) || 0,0,60);
        a.turn_rate = Number(a.turn_rate) > 0 && Number.isFinite(Number(a.turn_rate)) ? Number(a.turn_rate) : 14;
        a.center_rate = Number(a.center_rate) > 0 && Number.isFinite(Number(a.center_rate)) ? Number(a.center_rate) : 7;
        a.ratio = clamp(Number(a.ratio) || 1,0.05,2);
        a.turn_instant = !!a.turn_instant; a.center_instant = !!a.center_instant;
        a.current_angle = 0;
      }
    }
  }
  if(!input.profiles.some(p => p.id === input.active_id)) input.active_id = input.profiles[0].id;
  return input;
}
function ensureSelection(){
  const p=getProfile();
  if(!p) return;
  if(!p.trailers.some(t=>t.id===selectedTrailerId)) selectedTrailerId=p.trailers[0].id;
  if(!getTrailer()?.axles?.some(a=>a.id===selectedAxleId)) selectedAxleId=getTrailer().axles[0].id;
}
async function load(){
  try {
    const r=await fetch('/api/config', {cache:'no-store'});
    const payload=await r.json();
    if(!r.ok) throw new Error(payload.error || 'Erreur de lecture');
    data = normalizeData(payload);
    ensureSelection(); dirty=false; renderAll(); showSaveStatus();
    frameId=requestAnimationFrame(animate);
    pollDevices(); setInterval(pollDevices,110);
  } catch(e) {
    console.error(e); showSaveStatus('Erreur de chargement');
    toast('Impossible de lire les profils : ' + e.message,true);
  }
}
async function save(){
  if (!data) return;
  try {
    const payload=deepCopy(data);
    for (const p of payload.profiles) for (const t of p.trailers) for (const a of t.axles) a.current_angle=0;
    const r=await fetch('/api/config',{method:'POST',headers:{'Content-Type':'application/json','X-ATS-Token':token},body:JSON.stringify(payload)});
    const answer=await r.json();
    if(!r.ok)throw new Error(answer.error || 'Enregistrement refusé');
    dirty=false; showSaveStatus(); toast('Profil sauvegardé sur ton PC.');
  } catch(e){toast('Erreur pendant la sauvegarde : ' + e.message,true)}
}
function renderProfiles(){
  $('profiles').innerHTML=data.profiles.map(p => `<button class="profile ${p.id===data.active_id?'active':''}" data-profile="${escapeHTML(p.id)}"><span class="bullet"></span><span><strong>${escapeHTML(p.name)}</strong><small>${p.trailers.length} remorque(s) · ${p.trailers.reduce((n,t)=>n+t.axles.length,0)} essieu(x)</small></span></button>`).join('');
}
function renderTrailerSelect(){
  const p=getProfile();
  $('trailerSelect').innerHTML=p.trailers.map(t=>`<option value="${escapeHTML(t.id)}">${escapeHTML(t.name)}</option>`).join('');
  $('trailerSelect').value=getTrailer().id;
}
function renderAxleCards(){
  $('axleCards').innerHTML=getTrailer().axles.map((a,i)=>`<button class="axle-card ${a.id===getAxle().id?'active':''}" data-axle="${escapeHTML(a.id)}"><span class="axle-index">${String(i+1).padStart(2,'0')}</span><span class="axle-name"><strong>${escapeHTML(a.name)}</strong><small>${a.enabled?'Direction active':'Direction désactivée'} · max ${fmt(a.max_angle)}° ${a.reversed?'· inversé':''}</small></span><span class="axle-value" data-angle-id="${escapeHTML(a.id)}">${fmt(a.current_angle)}°</span></button>`).join('');
}
function renderSettings(){
  const a=getAxle();
  if (!a) return;
  $('axleTitle').textContent=a.name;
  $('axleEnabled').checked=!!a.enabled;
  $('axleReversed').checked=!!a.reversed;
  $('maxAngle').value=a.max_angle;
  $('turnRate').value=a.turn_rate;
  $('centerRate').value=a.center_rate;
  $('ratio').value=a.ratio;
  $('turnInstant').checked=!!a.turn_instant;
  $('centerInstant').checked=!!a.center_instant;
  $('turnRate').disabled=!!a.turn_instant;
  $('centerRate').disabled=!!a.center_instant;
}
function labelBinding(bind){
  if(!bind)return 'Non définie';
  if(!String(bind).includes(':'))return String(bind).replace(/^KEY:/,'');
  if(bind.startsWith('KEY:')){
    const code=bind.slice(4);
    return code.replace(/^Key/,'').replace(/^Digit/,'').replace('Arrow','Flèche ');
  }
  const [kind, id, type, item, sign] = bind.split(':');
  if(kind==='DEVICE'){
    if(type==='AXIS' && sign===undefined)return `${id} · Axe ${item} (analogique)`;
    return `${id} · ${type==='BUTTON'?'Bouton ':type==='POV'?'Croix directionnelle ':'Axe '}${type==='BUTTON'?Number(item)+1:item}${type==='AXIS'?' '+(sign==='+'?'positif':'négatif'):''}`;
  }
  return bind;
}
function renderBindings(){
  const p=getProfile();
  $('bindingToggle').textContent=labelBinding(p.toggle_key);
  $('bindingLeft').textContent=labelBinding(p.left_key);
  $('bindingRight').textContent=labelBinding(p.right_key);
  $('bindingAxis').textContent=labelBinding(p.analog_axis);
  $('axisInvert').checked=!!p.analog_invert;
  $('deadzone').value=p.analog_deadzone;
  $('axisCurve').value=p.analog_curve;
  $('analogFullScale').value=p.analog_full_scale;
  $('analogOutputPct').value=p.analog_output_pct;
}
function renderControl(){
  const p=getProfile();
  $('autoCenter').checked=!!p.auto_center_moving;
  if($('speedGain')) $('speedGain').value=p.center_speed_gain??1;
  $('centerButton').disabled=mode==='stop';
  $('toggleManual').textContent=manual?'◉   Désactiver le braquage manuel':'◉   Activer le braquage manuel';
  $('toggleManual').classList.toggle('enabled',manual);
  $('manualStatus').textContent=manual?'ACTIF':'INACTIF';
  $('manualStatus').classList.toggle('pill-accent',manual);
  $('modeBadge').textContent=mode==='stop'?'À L’ARRÊT':mode==='forward'?'MARCHE AVANT':'MARCHE ARRIÈRE';
  document.querySelectorAll('[data-mode]').forEach(b=>b.classList.toggle('active',b.dataset.mode===mode));
}
function renderAll(){
  ensureSelection();
  renderProfiles();renderTrailerSelect();renderAxleCards();renderSettings();renderBindings();renderControl();
  $('countTrailers').textContent=getProfile().trailers.length;
  $('countAxles').textContent=allAxles().length;
  updateDynamic();
}
function updateDynamic(){
  if(!data)return;
  $('selectedAngle').textContent=fmt(getAxle().current_angle)+'°';
  $('angleBadge').textContent=fmt(getAxle().current_angle)+'°';
  $('speedLabel').textContent=telemetry?.connected ? Math.abs(telemetry.speed_kmh).toFixed(1).replace('.',',')+' km/h · ATS' : mode==='forward'?'35 km/h · simulé':mode==='reverse'?'10 km/h · simulé':'0 km/h';
  $('analogValue').textContent=(analogSteering>=0?'+':'')+Math.round(analogSteering*100)+' %';
  $('analogIndicator').style.left=(50+analogSteering*50)+'%';
  $('steeringState').textContent=steeringState.toUpperCase();
  document.querySelectorAll('[data-angle-id]').forEach(el=>{
    const axle=allAxles().find(a=>a.id===el.dataset.angleId);
    if(axle)el.textContent=fmt(axle.current_angle)+'°';
  });
  for(const side of ['left','right'])document.querySelector(`[data-hold="${side}"]`).classList.toggle('held',side==='left'?leftDown:rightDown);
}
function centerAll(){ for(const a of allAxles())a.current_angle=0; updateDynamic(); }
function resetSim(){vehicle={x:0,y:0,yaw:0};angleTrail=[];updateDynamic();toast('Trajectoire réinitialisée, angles conservés.');}

function performAction(action){
  const p=getProfile();
  if(!p)return;
  if(action==='save') return save();
  if(action==='toggle-manual'){manual=!manual;renderControl();return;}
  if(action==='center'){if(mode==='stop')return toast('Recentrage indisponible à l’arrêt. Passe en marche avant ou arrière.',true);centerAll();return;}
  if(action==='reset-sim'){resetSim();return;}
  if(action==='add-profile'){
    if(data.profiles.length>=100)return toast('Maximum de 100 profils.',true);
    const name=prompt('Nom du nouveau profil :',`Profil ${data.profiles.length+1}`)?.trim();if(!name)return;
    const np=makeProfile(name);data.profiles.push(np);data.active_id=np.id;manual=false;selectedTrailerId=null;selectedAxleId=null;resetSim();markDirty();renderAll();return;
  }
  if(action==='clone-profile'){
    if(data.profiles.length>=100)return toast('Maximum de 100 profils.',true);
    const copy=deepCopy(p);copy.id=uid();copy.name=p.name+' (copie)';
    for(const t of copy.trailers){t.id=uid();for(const a of t.axles){a.id=uid();a.current_angle=0;}}
    data.profiles.push(copy);data.active_id=copy.id;manual=false;selectedTrailerId=null;selectedAxleId=null;markDirty();renderAll();return;
  }
  if(action==='rename-profile'){
    const n=prompt('Nouveau nom du profil :',p.name)?.trim();if(n){p.name=n;markDirty();renderProfiles();}return;
  }
  if(action==='add-trailer'){
    if(p.trailers.length>=20)return toast('Maximum de 20 remorques par profil.',true);
    const tr=makeTrailer(p.trailers.length+1);p.trailers.push(tr);selectedTrailerId=tr.id;selectedAxleId=tr.axles[0].id;markDirty();renderAll();return;
  }
  if(action==='remove-trailer'){
    if(p.trailers.length===1)return toast('Conserve au moins une remorque.',true);
    if(confirm('Supprimer cette remorque du profil ?')){p.trailers=p.trailers.filter(t=>t.id!==getTrailer().id);selectedTrailerId=null;selectedAxleId=null;markDirty();renderAll();}return;
  }
  if(action==='rename-trailer'){
    const t=getTrailer(), n=prompt('Nom de la remorque :',t.name)?.trim();if(n){t.name=n;markDirty();renderAll();}return;
  }
  if(action==='add-axle'){
    const t=getTrailer();if(t.axles.length>=32)return toast('Maximum de 32 essieux par remorque.',true);
    const a=makeAxle(t.axles.length+1);t.axles.push(a);selectedAxleId=a.id;markDirty();renderAll();return;
  }
  if(action==='remove-axle'){
    const t=getTrailer();if(t.axles.length<=1)return toast('Conserve au moins un essieu par remorque.',true);
    if(confirm('Supprimer cet essieu du profil ?')){t.axles=t.axles.filter(a=>a.id!==getAxle().id);selectedAxleId=null;markDirty();renderAll();}return;
  }
  if(action==='rename-axle'){
    const a=getAxle(),n=prompt('Nom de cet essieu :',a.name)?.trim();if(n){a.name=n;markDirty();renderAll();}return;
  }
  if(action==='export'){
    const json=JSON.stringify(deepCopy({...data,profiles:data.profiles.map(p=>({...p,trailers:p.trailers.map(t=>({...t,axles:t.axles.map(a=>({...a,current_angle:0}))}))}))}),null,2);
    const url=URL.createObjectURL(new Blob([json],{type:'application/json'}));
    const link=document.createElement('a');link.href=url;link.download='ATS_Studio_profils.json';document.body.append(link);link.click();link.remove();setTimeout(()=>URL.revokeObjectURL(url),2500);
    toast('Profils exportés au format JSON.');return;
  }
  if(action==='import'){$('importFile').click();return;}
  if(action==='quit'){
    if(dirty&&!confirm('Des changements ne sont pas enregistrés. Quitter ?'))return;
    dirty=false;fetch('/api/quit',{method:'POST',headers:{'X-ATS-Token':token}}).then(()=>{window.close();toast('Fermeture de l’application.');});return;
  }
  if(action==='clear-axis'){getProfile().analog_axis='';getProfile().analog_axis_center=0;markDirty();renderBindings();toast('Axe analogique supprimé.');return;}
  if(action==='cancel-capture')return cancelCapture();
}
function readSettings(ev){
  if(!data)return;
  const p=getProfile(),a=getAxle(),id=ev.target.id;
  if(id==='speedGain'){p.center_speed_gain=clamp(Number(ev.target.value)||0,0,4);markDirty();return;}
  if(id==='autoCenter'){p.auto_center_moving=ev.target.checked;markDirty();return;}
  if(id==='axisInvert'){p.analog_invert=ev.target.checked;markDirty();renderBindings();return;}
  if(['deadzone','axisCurve','analogFullScale','analogOutputPct'].includes(id)){
    const val=Number(ev.target.value);if(!Number.isFinite(val))return;
    const cfg={deadzone:['analog_deadzone',0,.4],axisCurve:['analog_curve',.5,3],analogFullScale:['analog_full_scale',20,100],analogOutputPct:['analog_output_pct',0,100]};
    const [field,min,max]=cfg[id];p[field]=clamp(val,min,max);markDirty();renderBindings();return;
  }
  if(['axleEnabled','axleReversed','turnInstant','centerInstant'].includes(id)){
    a[({axleEnabled:'enabled',axleReversed:'reversed',turnInstant:'turn_instant',centerInstant:'center_instant'})[id]]=ev.target.checked;
    markDirty();renderSettings();renderAxleCards();return;
  }
  const fields={maxAngle:['max_angle',0,60],turnRate:['turn_rate',Number.MIN_VALUE,Number.MAX_VALUE],centerRate:['center_rate',Number.MIN_VALUE,Number.MAX_VALUE],ratio:['ratio',0.05,2]};
  if(fields[id]){
    const [key,min,max]=fields[id];const n=Number(ev.target.value);
    if(!Number.isFinite(n)||(min===Number.MIN_VALUE && n<=0))return toast('Saisis une valeur numérique valide et positive.',true);
    a[key]=clamp(n,min,max);
    a.current_angle=clamp(a.current_angle,-a.max_angle,a.max_angle);
    markDirty();renderSettings();renderAxleCards();return;
  }
}

function bindingPressed(binding) {
  if(!binding)return false;
  if(!binding.includes(':'))return keyboard.has(binding.toUpperCase());
  if(binding.startsWith('KEY:'))return keyboard.has(binding.slice(4));
  if(!binding.startsWith('DEVICE:'))return false;
  const [,id,type,index,sign]=binding.split(':');
  const d=devices.find(item=>item.id===id);
  if(!d)return false;
  const n=Number(index);
  if(type==='BUTTON')return !!d.buttons?.[n];
  if(type==='AXIS')return (Number(d.axes?.[n])||0)*(sign==='+'?1:-1)>.65;
  if(type==='POV')return d.pov===n;
  return false;
}
function captureLabel(field){return ({toggle_key:'Activer le braquage',left_key:'Braquage gauche',right_key:'Braquage droite',analog_axis:'Axe analogique de direction'})[field];}
function startCapture(field){
  captureField=field;
  captureBaselines=deepCopy(devices);
  $('modal').classList.remove('hidden');
  $('captureMessage').textContent='Capture : '+captureLabel(field);
}
function cancelCapture(){captureField=null;captureBaselines=null;$('modal').classList.add('hidden');}
function finishCapture(binding){
  if(!captureField)return;
  const field=captureField;
  getProfile()[field]=binding;
  if(field==='analog_axis'){
    const [,id,,index]=binding.split(':');
    const base=captureBaselines?.find(d=>d.id===id);
    getProfile().analog_axis_center=Number(base?.axes?.[Number(index)])||0;
  }
  cancelCapture();markDirty();renderBindings();
  toast(`${captureLabel(field)} : ${labelBinding(binding)}. Pense à enregistrer.`);
}

async function pollDevices(){
  try{
    const response=await fetch('/api/devices',{cache:'no-store'});
    if(!response.ok)throw new Error('Périphériques non accessibles');
    prevDevices=devices;
    devices=await response.json();
    renderDeviceList();
    if(captureField){
      for(const d of devices){
        const base=captureBaselines?.find(x=>x.id===d.id);
        if(!base){captureBaselines=deepCopy(devices);continue;}
        const pressed=d.buttons?.findIndex((pressed,i)=>pressed&&!base.buttons?.[i]);
        if(captureField!=='analog_axis' && pressed>=0){finishCapture(`DEVICE:${d.id}:BUTTON:${pressed}`);break;}
        if(captureField!=='analog_axis' && d.pov>=0&&d.pov!==base.pov){finishCapture(`DEVICE:${d.id}:POV:${d.pov}`);break;}
        for(let i=0;i<(d.axes?.length??0);i++){
          if(Math.abs(d.axes[i]- (base.axes?.[i]??0))>.55){
            if(captureField==='analog_axis') finishCapture(`DEVICE:${d.id}:AXIS:${i}`);
            else finishCapture(`DEVICE:${d.id}:AXIS:${i}:${d.axes[i]>(base.axes?.[i]??0)?'+':'-'}`);
            break;
          }
        }
        if(!captureField)break;
      }
    }
  }catch(e){$('devices').innerHTML='<span class="muted">Lecture des périphériques indisponible.</span>';}
}
let lastDeviceSignature=null;
function renderDeviceList(){
  const signature=devices.map(d=>d.id+':'+d.name).join('|');
  if(signature===lastDeviceSignature)return;
  lastDeviceSignature=signature;
  $('devices').innerHTML=devices.length ? devices.map(d=>`<div class="device">🎮 ${escapeHTML(d.name)}<span>${escapeHTML(d.id)} · ${d.buttons.length} boutons détectables</span></div>`).join('') : '<span class="muted">Aucun volant, manette ou joystick reconnu. Le clavier reste disponible.</span>';
}

function currentAnalogPower(p) {
  if(!p.analog_axis?.startsWith('DEVICE:'))return 0;
  const [,id,type,index]=p.analog_axis.split(':');
  if(type!=='AXIS')return 0;
  const d=devices.find(d=>d.id===id);
  const raw=Number(d?.axes?.[Number(index)]);
  if(!Number.isFinite(raw))return 0;
  return SteeringLogic.analogPower(raw,p.analog_axis_center,p.analog_deadzone,p.analog_curve,p.analog_invert,p.analog_full_scale,p.analog_output_pct);
}
function tick(dt){
  if(!data)return;
  const p=getProfile();
  const toggle= bindingPressed(p.toggle_key);
  if(toggle&&!lastTogglePressed&&!captureField){manual=!manual;renderControl();}
  lastTogglePressed=toggle;
  leftDown=manual&&!captureField&&bindingPressed(p.left_key);
  rightDown=manual&&!captureField&&bindingPressed(p.right_key);
  const analogConnected = !!p.analog_axis && devices.some(d => d.id === p.analog_axis.split(':')[1]);
  const virtualAnalog = window.ATSStudioVirtualAnalog?.enabled === true;
  analogSteering = manual && !captureField ? (virtualAnalog ? SteeringLogic.analogPower(clamp(window.ATSStudioVirtualAnalog.value,-1,1),0,p.analog_deadzone,p.analog_curve,p.analog_invert,p.analog_full_scale,p.analog_output_pct) : currentAnalogPower(p)) : 0;
  // Les boutons numeriques ont la priorite tant qu'ils sont enfonces.
  const digital=Number(rightDown)-Number(leftDown);
  const analogActive=manual && !captureField && digital===0 && (virtualAnalog || analogConnected);
  const steer=digital;
  const speedKmh=telemetry?.connected&&!telemetry.paused ? Math.abs(telemetry.speed_kmh) : (mode==='forward'?35:mode==='reverse'?10:0);
  const movingMode=telemetry?.connected ? (telemetry.paused || speedKmh<.2 ? 'stop' : telemetry.speed_kmh<0?'reverse':'forward') : mode;
  let correcting=false,centering=false;
  for(const a of allAxles()){
    const result=SteeringLogic.updateAxle(a,{
      manual,mode:movingMode,steer,analogActive,analogSteer:analogSteering,dt,autoCenter:p.auto_center_moving,speedKmh,speedGain:p.center_speed_gain??1
    });
    a.current_angle=result.angle;
    correcting ||= result.action==='correction' || result.action==='position analogique';
    centering ||= result.action==='recentrage';
  }
  steeringState=correcting?(analogActive&&Math.abs(analogSteering)>1e-6?'position analogique':'correction manuelle'):centering?'recentrage automatique':'angle maintenu';
  const steerable=allAxles().filter(a=>a.enabled);
  const mean=steerable.length?steerable.reduce((s,a)=>s+a.current_angle,0)/steerable.length:0;
  if(movingMode!=='stop'){
    const v=(movingMode==='forward'?1:-.38);
    const rad=vehicle.yaw*Math.PI/180;
    vehicle.x+=Math.sin(rad)*v*dt*56;
    vehicle.y-=Math.cos(rad)*v*dt*56;
    vehicle.yaw+=(mean*Math.PI/180)*v*dt*8;
    if(!angleTrail.length||Math.hypot(vehicle.x-angleTrail[angleTrail.length-1].x,vehicle.y-angleTrail[angleTrail.length-1].y)>3){
      angleTrail.push({x:vehicle.x,y:vehicle.y});if(angleTrail.length>250)angleTrail.shift();
    }
  }
}

const cv=$('scene');
const ctx=cv.getContext('2d');
function drawScene(){
  if(!data)return;
  const w=cv.clientWidth,h=cv.clientHeight,dpr=Math.min(devicePixelRatio||1,2);
  if(cv.width!==Math.round(w*dpr)||cv.height!==Math.round(h*dpr)){cv.width=Math.round(w*dpr);cv.height=Math.round(h*dpr);}
  ctx.setTransform(dpr,0,0,dpr,0,0);
  ctx.clearRect(0,0,w,h);
  ctx.save();ctx.globalAlpha=.33;ctx.lineWidth=1;ctx.strokeStyle='#29435b';
  for(let x=w/2%32;x<w;x+=32){ctx.beginPath();ctx.moveTo(x,0);ctx.lineTo(x,h);ctx.stroke();}
  for(let y=h/2%32;y<h;y+=32){ctx.beginPath();ctx.moveTo(0,y);ctx.lineTo(w,y);ctx.stroke();}
  ctx.restore();
  ctx.save();ctx.translate(w/2,h/2+5);
  // Visual camera follows the vehicle. The green trace represents only the simulated trajectory.
  ctx.strokeStyle='#4cd5bd80';ctx.lineWidth=2;ctx.beginPath();
  for(let i=0;i<angleTrail.length;i++){
    const pt=angleTrail[i];
    if(i===0)ctx.moveTo((pt.x-vehicle.x)*.85,(pt.y-vehicle.y)*.85);
    else ctx.lineTo((pt.x-vehicle.x)*.85,(pt.y-vehicle.y)*.85);
  }
  ctx.stroke();
  ctx.rotate(vehicle.yaw*Math.PI/180);
  const trailers=getProfile().trailers;
  const scale=trailers.length>3?.68:trailers.length>2?.82:1;
  ctx.scale(scale,scale);
  const len=124, width=65,gap=18;
  // Truck tractor at head of the chain.
  const firstY=trailers.length*(len+gap)/2 - (len+gap)/2;
  ctx.save();ctx.translate(0,-firstY-88);
  roundRect(ctx,-21,-30,42,65,9,'#43879a','#68e6d9');
  roundRect(ctx,-16,-22,32,24,6,'#213e50','#6699b2');
  ctx.fillStyle='#9ef2ea';ctx.fillRect(-13,-27,26,3);
  ctx.restore();
  trailers.forEach((t,index)=>{
    const cy=index*(len+gap)-firstY;
    ctx.save();ctx.translate(0,cy);
    ctx.fillStyle='#32455b';ctx.fillRect(-3,-len/2-12,6,15);
    const visualType=t.model_type||((trailers.length===1&&window.EssieuPilotTelemetry?.trailers?.[0]?.type)||'surbaissée');
    roundRect(ctx,-width/2,-len/2,width,len,9,'#253e58','#56758e');
    if(visualType==='surbaissée'){
      roundRect(ctx,-width/2+5,-len/2+27,width-10,len-47,5,'#152c3f','#4bdabb');
      roundRect(ctx,-width/2+4,-len/2+3,width-8,22,5,'#365e75','#72b6bd');
    }else if(visualType==='fermée')roundRect(ctx,-width/2+4,-len/2+4,width-8,len-8,6,'#527c90','#a1c1d0');
    else if(visualType==='citerne')roundRect(ctx,-width/2+14,-len/2+7,width-28,len-14,18,'#476d7d','#b2d9e5');
    else if(visualType==='forestière'){for(let k=-2;k<3;k++)roundRect(ctx,k*6-4,-len/2+12,8,len-24,4,'#8e7252','#c8a06c');}
    else roundRect(ctx,-width/2+7,-len/2+8,width-14,len-16,4,'#1b3149','#3b5971');
    ctx.fillStyle='#b4bfcc';ctx.font='10px Segoe UI';ctx.textAlign='center';ctx.fillText(`REMORQUE ${index+1}`,0,0);
    for(let i=0;i<t.axles.length;i++){
      const a=t.axles[i],axY=(len/2-19)-i*Math.min(24,Math.max(9,82/Math.max(1,t.axles.length-1)));
      const turn=a.current_angle*Math.PI/180;
      for(const side of [-1,1]){
        ctx.save();ctx.translate(side*(width/2+6),axY);ctx.rotate(-turn);
        const color=!a.enabled?'#6c798a':a.id===selectedAxleId?'#b6a0ff':'#59dfc8';
        roundRect(ctx,-5,-13,10,26,3,color,color);
        ctx.restore();
      }
      ctx.strokeStyle='#93a3b7';ctx.lineWidth=2;ctx.beginPath();ctx.moveTo(-width/2,axY);ctx.lineTo(width/2,axY);ctx.stroke();
    }
    ctx.restore();
  });
  ctx.restore();
  ctx.fillStyle='#8aa3ba';ctx.textAlign='right';ctx.font='11px Segoe UI';ctx.fillText('VUE DU DESSUS  ·  APERÇU APPROXIMATIF',w-17,21);
}
function roundRect(ctx,x,y,w,h,r,fill,stroke){
  ctx.beginPath();ctx.roundRect(x,y,w,h,r);ctx.fillStyle=fill;ctx.fill();if(stroke){ctx.lineWidth=1.5;ctx.strokeStyle=stroke;ctx.stroke();}
}
function animate(now){
  const dt=clamp((now-lastFrame)/1000,0,.1);lastFrame=now;
  tick(dt);drawScene();
  if(now-lastUiDraw>95){lastUiDraw=now;updateDynamic();}
  frameId=requestAnimationFrame(animate);
}

// Application controls — click, keyboard and pointer input.
document.addEventListener('click',event=>{
  const profileButton=event.target.closest('[data-profile]');
  if(profileButton&&data){
    data.active_id=profileButton.dataset.profile;selectedTrailerId=null;selectedAxleId=null;manual=false;resetSim();markDirty();renderAll();return;
  }
  const axleButton=event.target.closest('[data-axle]');
  if(axleButton){selectedAxleId=axleButton.dataset.axle;renderAxleCards();renderSettings();return;}
  const actionButton=event.target.closest('[data-action]');
  if(actionButton){performAction(actionButton.dataset.action);return;}
  const captureButton=event.target.closest('[data-capture]');
  if(captureButton){startCapture(captureButton.dataset.capture);return;}
  const modeButton=event.target.closest('[data-mode]');
  if(modeButton){mode=modeButton.dataset.mode;renderControl();return;}
});
$('trailerSelect').addEventListener('change',e=>{selectedTrailerId=e.target.value;selectedAxleId=null;ensureSelection();renderAxleCards();renderSettings();});
for(const id of ['speedGain','autoCenter','axisInvert','deadzone','axisCurve','analogFullScale','analogOutputPct','axleEnabled','axleReversed','turnInstant','centerInstant','maxAngle','turnRate','centerRate','ratio']) $(id).addEventListener('change',readSettings);
$('importFile').addEventListener('change',async event=>{
  const file=event.target.files[0];if(!file)return;
  try{if(file.size>2*1024*1024)throw new Error('Fichier trop volumineux.');
    const next=normalizeData(JSON.parse(await file.text()));
    if(!confirm('Importer ces profils à la place de ceux présents ? Pense à enregistrer ensuite.'))return;
    data=next;manual=false;selectedTrailerId=null;selectedAxleId=null;resetSim();markDirty();renderAll();toast('Profils importés. Clique sur Enregistrer pour les conserver.');
  }catch(e){toast('Importation impossible : '+e.message,true)}finally{event.target.value='';}
});
window.addEventListener('keydown',event=>{
  if(captureField){
    event.preventDefault();event.stopPropagation();
    if(event.code==='Escape'){cancelCapture();return;}
    if(['ShiftLeft','ShiftRight','ControlLeft','ControlRight','AltLeft','AltRight','MetaLeft','MetaRight'].includes(event.code))return;
    finishCapture('KEY:'+event.code);return;
  }
  if(event.target.closest('input,select,textarea') || event.ctrlKey || event.altKey || event.metaKey)return;
  keyboard.add(event.code);
  if(['ArrowUp','ArrowDown','ArrowLeft','ArrowRight','Space','F8','F9','F10'].includes(event.code))event.preventDefault();
});
window.addEventListener('keyup',event=>{keyboard.delete(event.code);});
window.addEventListener('blur',()=>{keyboard.clear();leftDown=false;rightDown=false;});
for(const side of ['left','right']){
  const b=document.querySelector(`[data-hold="${side}"]`);
  b.addEventListener('pointerdown',e=>{e.preventDefault();b.setPointerCapture(e.pointerId);keyboard.add('__button-'+side);});
  for(const type of ['pointerup','pointercancel','lostpointercapture'])b.addEventListener(type,()=>keyboard.delete('__button-'+side));
}
// Virtual direction buttons operate alongside any configured keyboard/controller input.
const physicalBindingPressed=bindingPressed;
bindingPressed = function(binding){
  if(data){
    if(binding===getProfile().left_key&&keyboard.has('__button-left'))return true;
    if(binding===getProfile().right_key&&keyboard.has('__button-right'))return true;
  }
  return physicalBindingPressed(binding);
};
window.addEventListener('beforeunload',e=>{if(dirty){e.preventDefault();e.returnValue='';}});
load();

// Connexion réellement lue depuis la télémétrie SCS (DLL compatible, lecture seule).
async function pollTelemetry(){
  try {
    const r=await fetch('/api/telemetry',{cache:'no-store'});
    if(!r.ok)throw new Error('Source indisponible');
    const t=await r.json();telemetry=t;
    const active=t.connected && Array.isArray(t.trailers) && t.trailers.length ? t.trailers[0] : null;
    const nextKey=active?.id||'';
    if(t.connected&&active && nextKey!==telemetryTrailerKey && data){
      const matching=data.profiles.find(p=>p.match_key && p.match_key.toLowerCase()===nextKey.toLowerCase());
      if(matching){data.active_id=matching.id;selectedTrailerId=null;selectedAxleId=null;manual=false;renderAll();}
      telemetryMatchText=matching?'Profil reconnu et chargé : '+matching.name:'Remorque inconnue : associer le profil sélectionné';
    }
    if(data)telemetryTrailerKey=nextKey;
    window.EssieuPilotTelemetry=t;
    if(typeof window.refreshEssieuTelemetry==='function')window.refreshEssieuTelemetry(t,telemetryMatchText);
  } catch(e){telemetry=null;window.EssieuPilotTelemetry=null;}
}
setInterval(pollTelemetry,450);
pollTelemetry();
