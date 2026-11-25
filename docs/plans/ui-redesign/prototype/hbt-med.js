/* hbt-med.js — plan 6 sample store: medication courses and doses, the Due list, the overdue window,
   Web Push devices and ntfy opt-ins. Prototype stand-in for the API; localStorage key hbt.med.v1 (memory fallback).
   Symptoms are ordinary timeline entries (type 'symptom') in hbt-data.js; this file only holds what they need
   (the viewer's temperature unit, the validation and the "API columns" for °C and °F).
   The UI never interprets a value: it shows what this store returns. Occurrence/“done” logic lives here ONCE,
   standing in for the server. */
(function(){
  var KEY='hbt.med.v1',MIN=6e4,HR=60*MIN,DAY=864e5,mem=null,rawSeen=null;
  var MAX_COURSES=50,MAX_DOSES=5000,MAX_DEVICES=5,EARLY=30*MIN,SHOWN=50;
  var SYMPTOMS=['Congestion','Cough','Fever','Vomiting','Diarrhea','Constipation','Rash','Ear pain','Sore throat','Poor appetite','Unusually sleepy','Other'];
  var SEVS=['Mild','Moderate','Severe'];
  var METHODS=['Oral','Rectal','Armpit','Ear','Forehead','Other'];
  var ERR={cap:'This child has the maximum number of medication courses',doseCap:'This child has the maximum number of logged doses',goneCourse:'This course is no longer in the list',
    network:'The app cannot reach its server, try again soon',server:'Something went wrong, try again',
    amount:'Enter an amount of up to 80 characters',time:'Enter a time that is not in the future and not before the birth date',note:'The note is too long',
    devices:'You have the most devices allowed. Remove one, then try again.',refused:'The server refused this browser’s notification address. Try again later.'};
  function z2(n){return String(n).padStart(2,'0');}
  function uid(p){return p+Math.random().toString(36).slice(2,8);}
  function H(){return window.HBT;}
  function dk(t){return H().dayKey(t);}
  function hm(t){return H().hm(t);}
  function toLocal(ms){return dk(ms)+'T'+hm(ms);}
  function parseLocal(s){if(!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(s||''))return NaN;var g=Date.parse(s+':00Z');if(isNaN(g))return NaN;var loc=Date.parse(toLocal(g)+':00Z');return g-(loc-g);}
  function dayStartKey(key){return H().dayStartOfKey(key)||Date.parse(key+'T00:00:00Z');}
  function fmtAt(t,now){return dk(t)===dk(now)?hm(t):dk(t)+' '+hm(t);}
  function yyyy(t){return dk(t);}

  function seed(prev){
    var h=H(),now=Date.now(),ds=h.dayStart(now),n=now;
    var d={day:dk(now),unit:prev?prev.unit:'C',channel:prev?prev.channel:'both',ovd:prev?prev.ovd:{},devices:prev?prev.devices:[],here:prev?prev.here:null,ntfy:prev?prev.ntfy:{},courses:[],doses:[]};
    if(!prev){
      d.devices=[{id:'dv1',pid:'dan',label:'Work laptop',added:Date.UTC(2026,10,2,10,15),kids:['leo']},{id:'dv2',pid:'ana',label:'Ana’s phone',added:Date.UTC(2026,10,5,8,0),kids:['leo','mia']}];
    }
    var a=ds-3*DAY;
    d.courses=[
      {id:'c1',kid:'leo',name:'Calpol',dose:'2.5 ml',kind:'fixed',times:['08:00','14:00','20:00'],every:0,start:a,last:'',active:true,note:'With food'},
      {id:'c2',kid:'leo',name:'Amoxil',dose:'5 ml',kind:'every',times:[],every:6,start:n-Math.round(6.5*HR/MIN)*MIN,last:'',active:true,note:''},
      {id:'c3',kid:'leo',name:'Vitamin D drops',dose:'1 drop',kind:'asneeded',times:[],every:0,start:a,last:'',active:true,note:''},
      {id:'c4',kid:'leo',name:'Nurofen',dose:'2.5 ml',kind:'fixed',times:['09:00'],every:0,start:ds-10*DAY,last:dk(ds-4*DAY),active:true,note:''},
      {id:'c5',kid:'leo',name:'Saline spray',dose:'1 spray',kind:'asneeded',times:[],every:0,start:ds-20*DAY,last:'',active:false,note:''}];
    var D=function(c,at,amt,note,by){if(at<=n)d.doses.push({id:uid('d'),kid:'leo',course:c,at:at,amount:amt||'',note:note||'',by:by||'You'});};
    D('c1',ds-DAY+20*HR+5*MIN,'2.5 ml','','Cleo');D('c1',ds+8*HR+5*MIN,'2.5 ml','');
    D('c2',n-Math.round(6.4*HR/MIN)*MIN,'5 ml','');
    D('c3',ds-DAY+9*HR,'1 drop','');D('c4',ds-5*DAY+9*HR,'2.5 ml','');
    return d;}
  function read(){var raw=null;try{raw=localStorage.getItem(KEY);}catch(e){}
    if(raw&&raw===rawSeen&&mem&&mem.day===dk(Date.now()))return mem;
    var d=null;if(raw){try{d=JSON.parse(raw);}catch(e){d=null;}}if(!d)d=mem;
    if(!d){d=seed();write(d);return d;}
    if(d.day!==dk(Date.now())){d=seed(d);write(d);return d;}
    mem=d;rawSeen=raw;return d;}
  function write(d){mem=d;try{var s=JSON.stringify(d);localStorage.setItem(KEY,s);rawSeen=s;}catch(e){rawSeen=null;}}

  /* ---- occurrences (server rules; the UI only displays what comes back) ---- */
  function occ(c,a,b){var out=[],h=H();
    if(c.kind==='asneeded')return out;
    if(c.kind==='every'){var step=c.every*HR,k=Math.max(0,Math.ceil((a-c.start)/step));for(var t=c.start+k*step;t<=b&&out.length<400;t+=step)out.push(t);}
    else{for(var day=h.dayStart(a);day<=b;day=h.dayStart(day+DAY+2*HR)){c.times.forEach(function(x){var p=x.split(':'),t=day+(+p[0]*60+ +p[1])*MIN;if(t>=a&&t<=b&&t>=c.start)out.push(t);});}}
    out.sort(function(x,y){return x-y;});
    if(c.last)out=out.filter(function(t){return dk(t)<=c.last;});
    return out;}
  function doneFor(L,i,doses){var o=L[i],nx=i+1<L.length?L[i+1]-EARLY:Infinity;
    return doses.some(function(x){return x.at>=o-EARLY&&x.at<nx;});}
  function ended(c,now){return !!c.last&&c.last<dk(now);}
  function dosesOf(d,cid){return d.doses.filter(function(x){return x.course===cid;}).sort(function(a,b){return b.at-a.at;});}

  var pickMem={};
  var M={
    pick:function(k,id){pickMem[k]=id;try{localStorage.setItem('hbt.pick.'+k,id||'');}catch(e){}},picked:function(k){var v=pickMem[k];if(v===undefined){try{v=localStorage.getItem('hbt.pick.'+k)||'';}catch(e){v='';}}return v||'';},
    ERR:ERR,SYMPTOMS:SYMPTOMS,SEVS:SEVS,METHODS:METHODS,MAX_DEVICES:MAX_DEVICES,SHOWN:SHOWN,fmtAt:fmtAt,toLocal:toLocal,parseLocal:parseLocal,
    reset:function(){var d=read();write(seed({unit:d.unit,channel:'both',ovd:{},devices:null,here:null,ntfy:{}}));var x=read();x.devices=seed().devices;x.here=null;x.ntfy={};write(x);},
    snapshot:function(){return JSON.stringify(read());},restore:function(s){write(JSON.parse(s));},
    unit:function(){return read().unit;},setUnit:function(u){var d=read();d.unit=u==='F'?'F':'C';write(d);},
    channel:function(){return read().channel;},setChannel:function(c){var d=read();d.channel=c;write(d);},
    pushOn:function(){return read().channel!=='none';},ntfyConfigured:function(){return read().channel==='both';},
    /* ---- symptoms: the "API" stores both temperature columns, the UI shows the viewer's ---- */
    saveSymptom:function(kid,f,id){var e={},h=H(),now=Date.now(),t=parseLocal(f.time),born=Date.parse((h.kidMeta(kid).born||'1970-01-01')+'T00:00:00Z');
      if(isNaN(t)||t>now+5*MIN||t<born)e.time=ERR.time;
      if(SYMPTOMS.indexOf(f.sym)<0)e.sym='Choose a symptom';
      if(f.sev&&SEVS.indexOf(f.sev)<0)e.sev='Choose mild, moderate or severe, or leave it unset';
      var o=String(f.other||'').trim(),tc=null,tf=null,mt='';
      if(f.sym==='Other'&&(!o||o.length>80))e.other='Enter what the symptom is, up to 80 characters';
      if(f.sym==='Fever'){var raw=String(f.temp||'').trim().replace(',','.'),x=/^\d{1,3}(\.\d{1,2})?$/.test(raw)?parseFloat(raw):NaN,u=f.unit==='F'?'F':'C';
        var lo=u==='C'?30:86,hi=u==='C'?45:113;
        if(isNaN(x)||x<lo||x>hi)e.temp='Enter a temperature from 30 to 45 °C (86 to 113 °F), with up to two decimals';
        else{tc=u==='C'?x:Math.round((x-32)*5/9*100)/100;tf=u==='F'?x:Math.round((x*9/5+32)*100)/100;}
        if(METHODS.indexOf(f.method)<0)e.method='Choose how the temperature was taken';else mt=f.method;}
      if(String(f.note||'').length>500)e.note=ERR.note;
      if(Object.keys(e).length)return {err:e};
      var rec={at:t,sym:f.sym,sev:f.sev||'',other:f.sym==='Other'?o:'',tc:tc,tf:tf,method:mt,note:String(f.note||'').trim()};
      if(id){h.patch(id,rec);return {ok:true,id:id};}
      rec.type='symptom';rec.end=null;rec.kid=kid;return {ok:true,id:h.add(rec)};},
    /* “09:12 Fever 38.4 °C (armpit), mild” parts without the time */
    symptomText:function(e){var s=e.sym==='Other'?'Other: '+e.other:e.sym;
      if(e.sym==='Fever'&&e.tc!=null){var u=read().unit,v=u==='F'?e.tf:e.tc;s+=' '+v+' °'+u+' ('+String(e.method||'').toLowerCase()+')';}
      if(e.sev)s+=', '+e.sev.toLowerCase();return s;},
    /* ---- courses ---- */
    courses:function(kid){return read().courses.filter(function(c){return c.kid===kid;}).sort(function(a,b){return b.start-a.start;});},
    course:function(id){return read().courses.filter(function(c){return c.id===id;})[0]||null;},
    doseCount:function(cid){return read().doses.filter(function(x){return x.course===cid;}).length;},
    splitCourses:function(kid,now){var all=M.courses(kid);return {all:all.length,active:all.filter(function(c){return c.active&&!ended(c,now);}),other:all.filter(function(c){return !c.active||ended(c,now);})};},
    schedule:function(c){if(c.kind==='asneeded')return 'As needed';if(c.kind==='fixed')return c.times.slice().sort().join(', ');var s=hm(c.start);return c.every===1?'Every hour from '+s:'Every '+c.every+' hours from '+s;},
    lastText:function(c,now){var x=dosesOf(read(),c.id)[0];return x?'Last dose logged at '+fmtAt(x.at,now):'No dose logged yet';},
    nextText:function(c,now){if(c.kind==='asneeded'||!c.active||ended(c,now))return '';var d=read(),ds=dosesOf(d,c.id),L=occ(c,now-DAY,now+9*DAY);
      for(var i=0;i<L.length;i++){if(L[i]>now&&!doneFor(L,i,ds))return 'Next due '+fmtAt(L[i],now);}return '';},
    /* fields = {name,dose,kind,times[12],every,start,last,active,note} */
    saveCourse:function(kid,f,id){var e={},h=H(),now=Date.now(),born=Date.parse((h.kidMeta(kid).born||'1970-01-01')+'T00:00:00Z');
      var nm=String(f.name||'').trim(),ds=String(f.dose||'').trim();
      if(!nm||nm.length>80)e.name='Enter a name of up to 80 characters';
      if(!ds||ds.length>80)e.dose='Enter a dose of up to 80 characters';
      var times=[],ev=0,bad=false;
      if(f.kind==='fixed'){(f.times||[]).forEach(function(x){x=String(x||'').trim();if(x){if(!/^\d{2}:\d{2}$/.test(x)||+x.slice(0,2)>23||+x.slice(3)>59)bad=true;else times.push(x);}});
        if(bad||!times.length||times.length>12||times.some(function(x,i){return times.indexOf(x)!==i;}))e.sched='Choose as needed, up to 12 different times of day, or a whole number of hours from 1 to 168';}
      else if(f.kind==='every'){var r=String(f.every||'').trim();ev=/^\d{1,3}$/.test(r)?+r:0;if(ev<1||ev>168)e.sched='Choose as needed, up to 12 different times of day, or a whole number of hours from 1 to 168';}
      else if(f.kind!=='asneeded')e.sched='Choose as needed, up to 12 different times of day, or a whole number of hours from 1 to 168';
      var old=id?M.course(id):null,st=old?old.start:now;
      if(String(f.start||'').trim()){var p=parseLocal(f.start);if(isNaN(p)||p<born)e.start='Enter a start that is not before the birth date';else st=p;}
      var last=String(f.last||'').trim();
      if(last){var lt=/^\d{4}-\d{2}-\d{2}$/.test(last)?Date.parse(last+'T00:00:00Z'):NaN;if(isNaN(lt)||new Date(lt).toISOString().slice(0,10)!==last||last<dk(st))e.last='Enter a last day that is a real date and not before the start';}
      if(String(f.note||'').length>500)e.note=ERR.note;
      if(Object.keys(e).length)return {err:e};
      var d=read();
      if(!id&&d.courses.filter(function(c){return c.kid===kid;}).length>=MAX_COURSES)return {page:ERR.cap};
      var rec={name:nm,dose:ds,kind:f.kind,times:times,every:ev,start:st,last:last,active:!!f.active,note:String(f.note||'').trim()};
      if(id){var c=d.courses.filter(function(x){return x.id===id;})[0];if(!c)return {page:ERR.goneCourse};Object.assign(c,rec);write(d);return {ok:true};}
      rec.id=uid('c');rec.kid=kid;d.courses.push(rec);write(d);return {ok:true};},
    deleteCourse:function(id){var d=read();if(!d.courses.some(function(c){return c.id===id;}))return {page:ERR.goneCourse};d.courses=d.courses.filter(function(c){return c.id!==id;});d.doses=d.doses.filter(function(x){return x.course!==id;});write(d);return {ok:true};},
    /* ---- doses ---- */
    doses:function(kid,cid){var d=read();return d.doses.filter(function(x){return x.kid===kid&&(!cid||x.course===cid);}).sort(function(a,b){return b.at-a.at;}).slice(0,SHOWN).map(function(x){var c=d.courses.filter(function(k){return k.id===x.course;})[0];return Object.assign({course_name:c?c.name:''},x);});},
    dose:function(id){var d=read(),x=d.doses.filter(function(y){return y.id===id;})[0];if(!x)return null;var c=d.courses.filter(function(k){return k.id===x.course;})[0];return Object.assign({course_name:c?c.name:''},x);},
    logDose:function(kid,cid,amount){var d=read(),c=d.courses.filter(function(x){return x.id===cid;})[0],a=String(amount||'').trim();
      if(!c)return {page:ERR.goneCourse};if(a.length>80)return {err:{amount:ERR.amount}};
      if(d.doses.filter(function(x){return x.kid===kid;}).length>=MAX_DOSES)return {page:ERR.doseCap};
      d.doses.push({id:uid('d'),kid:kid,course:cid,at:Date.now(),amount:a,note:'',by:'You'});write(d);return {ok:true};},
    saveDose:function(id,f){var d=read(),x=d.doses.filter(function(y){return y.id===id;})[0],e={};if(!x)return {page:'This dose is no longer in the list'};
      var t=parseLocal(f.at),born=Date.parse((H().kidMeta(x.kid).born||'1970-01-01')+'T00:00:00Z');
      if(isNaN(t)||t>Date.now()+5*MIN||t<born)e.at=ERR.time;if(String(f.amount||'').trim().length>80)e.amount=ERR.amount;if(String(f.note||'').length>500)e.note=ERR.note;
      if(Object.keys(e).length)return {err:e};x.at=t;x.amount=String(f.amount||'').trim();x.note=String(f.note||'').trim();write(d);return {ok:true};},
    deleteDose:function(id){var d=read();d.doses=d.doses.filter(function(x){return x.id!==id;});write(d);return {ok:true};},
    /* ---- overdue window (kid settings, guardian only) ---- */
    overdueMin:function(kid){var v=read().ovd[kid];return v==null?60:v;},
    setOverdue:function(kid,v){var r=String(v==null?'':v).trim(),n=/^\d{1,4}$/.test(r)?+r:NaN;if(isNaN(n)||n<5||n>1440)return {err:'Enter minutes from 5 to 1440'};var d=read();d.ovd[kid]=n;write(d);return {ok:true};},
    /* ---- Due list: medication lines only; Kid Home merges vaccine lines and applies the shared cap ---- */
    dueLines:function(kid,now){var d=read(),ov=M.overdueMin(kid)*MIN,out=[];
      d.courses.filter(function(c){return c.kid===kid&&c.active&&c.kind!=='asneeded'&&!ended(c,now);}).forEach(function(c){
        var ds=dosesOf(d,c.id),L=occ(c,now-DAY-ov,now+9*DAY);
        L.forEach(function(o,i){if(doneFor(L,i,ds))return;
          if(o<=now){var nx=i+1<L.length?L[i+1]:Infinity;if(now<Math.min(o+ov,nx))out.push({course:c.id,name:c.name,at:o,overdue:true});}
          else if(dk(o)===dk(now))out.push({course:c.id,name:c.name,at:o,overdue:false});});});
      out.sort(function(a,b){return a.at-b.at;});
      return out.map(function(x){return {course:x.course,name:x.name,overdue:x.overdue,text:x.name+' · '+fmtAt(x.at,now)+(x.overdue?' · overdue':'')};});},
    /* ---- Web Push devices and ntfy (per parent) ---- */
    devices:function(pid){return read().devices.filter(function(x){return x.pid===pid;});},
    here:function(pid){var d=read(),x=d.devices.filter(function(y){return y.id===d.here&&y.pid===pid;})[0];return x||null;},
    hereExists:function(){var d=read();return d.devices.some(function(y){return y.id===d.here;});},
    enableHere:function(pid){var d=read();
      var cur=d.devices.filter(function(y){return y.id===d.here;})[0];
      if(cur&&cur.pid===pid)return {ok:true,device:cur};
      if(!cur&&d.devices.filter(function(y){return y.pid===pid;}).length>=MAX_DEVICES)return {err:ERR.devices};
      if(cur){d.devices=d.devices.filter(function(y){return y.id!==cur.id;});}
      var n={id:uid('dv'),pid:pid,label:'This device',added:Date.now(),kids:[]};d.devices.push(n);d.here=n.id;write(d);return {ok:true,device:n,moved:!!cur};},
    saveDevice:function(id,f,kidsAllowed){var d=read(),x=d.devices.filter(function(y){return y.id===id;})[0],e={},l=String(f.label||'').trim();if(!x)return {page:'Not found'};
      if(!l||l.length>40)e.label='Enter a name of 1 to 40 characters';if(Object.keys(e).length)return {err:e};
      x.label=l;x.kids=(f.kids||[]).filter(function(k){return kidsAllowed.indexOf(k)>=0;});write(d);return {ok:true};},
    removeDevice:function(id){var d=read();d.devices=d.devices.filter(function(y){return y.id!==id;});if(d.here===id)d.here=null;write(d);return {ok:true};},
    ntfy:function(pid){return read().ntfy[pid]||null;},
    ntfyOn:function(pid){var d=read();if(!d.ntfy[pid])d.ntfy[pid]={topic:'hb-'+Math.random().toString(36).slice(2,12),kids:[]};write(d);return {ok:true};},
    ntfyNew:function(pid){var d=read();if(d.ntfy[pid]){d.ntfy[pid].topic='hb-'+Math.random().toString(36).slice(2,12);write(d);}return {ok:true};},
    ntfyOff:function(pid){var d=read();delete d.ntfy[pid];write(d);return {ok:true};},
    ntfySave:function(pid,kids,allowed){var d=read();if(!d.ntfy[pid])return {page:'ntfy is not configured on this server.'};d.ntfy[pid].kids=(kids||[]).filter(function(k){return allowed.indexOf(k)>=0;});write(d);return {ok:true};},
    ntfyServer:function(){return 'ntfy.happybaby.example';},
    /* ---- cleanup rules (revoke / leave / disable / delete a child) ---- */
    cleanup:function(kid,pid){var d=read();d.devices.forEach(function(x){if(!pid||x.pid===pid)x.kids=x.kids.filter(function(k){return k!==kid;});});
      Object.keys(d.ntfy).forEach(function(p){if(!pid||p===pid)d.ntfy[p].kids=d.ntfy[p].kids.filter(function(k){return k!==kid;});});write(d);},
    cleanupPerson:function(pid){var d=read();d.devices=d.devices.filter(function(x){return x.pid!==pid;});if(!d.devices.some(function(x){return x.id===d.here;}))d.here=null;delete d.ntfy[pid];write(d);},
    dropKid:function(kid){var d=read();d.courses=d.courses.filter(function(c){return c.kid!==kid;});d.doses=d.doses.filter(function(x){return x.kid!==kid;});delete d.ovd[kid];write(d);M.cleanup(kid);}
  };
  window.HBTM=M;
})();
