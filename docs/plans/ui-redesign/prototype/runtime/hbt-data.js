/* hbt-data.js — the one sample-data store shared by Kid Home, Timeline (and later pages).
   Prototype stand-in for the API: entries live in localStorage (in memory if storage throws), so a bottle logged
   on Kid Home shows on Timeline and in Today. Wording/units/time-zone formatting lives here ONCE. */
(function(){
  var KEY='hbt.v2',TZ='America/Los_Angeles',MIN=6e4,DAY=864e5,FUTURE=5*MIN,MAXRUN=24*60*MIN;
  var KIDS=[{id:'ivy',name:'Ivy'},{id:'mia',name:'Mia'},{id:'leo',name:'Leo'}];
  var TYPES=['bottle','nursing','diaper','sleep','solids','pumping'];
  var NAME={bottle:'Bottle',nursing:'Nursing',diaper:'Diaper',sleep:'Sleep',solids:'Solids',pumping:'Pumping',symptom:'Symptom'};
  var CONTENTS=['Breast milk','Formula',"Cow's milk",'Other'];
  var KINDS=['Wet','Dirty','Both','Dry'];
  var mem=null,rawSeen=null;
  var fHM=new Intl.DateTimeFormat('en-GB',{timeZone:TZ,hour:'2-digit',minute:'2-digit',hourCycle:'h23'});
  var fKey=new Intl.DateTimeFormat('en-CA',{timeZone:TZ,year:'numeric',month:'2-digit',day:'2-digit'});
  var fParts=new Intl.DateTimeFormat('en-GB',{timeZone:TZ,hour:'numeric',minute:'numeric',second:'numeric',hourCycle:'h23'});
  var fHead=new Intl.DateTimeFormat('en-US',{timeZone:TZ,weekday:'short',month:'short',day:'numeric'});
  var z2=function(n){return String(n).padStart(2,'0');};
  function dayKey(t){return fKey.format(t);}
  function hm(t){return fHM.format(t);}
  function dayStart(t){var p=fParts.formatToParts(t),g={};p.forEach(function(x){g[x.type]=Number(x.value);});return t-((g.hour*60+g.minute)*60+g.second)*1000-(t%1000);}
  function heading(t,now){var k=dayKey(t);if(k===dayKey(now))return 'Today';if(k===dayKey(now-DAY))return 'Yesterday';return fHead.format(t);}
  function fmtDur(m){m=Math.max(0,Math.round(m));if(m<60)return m+' min';var h=Math.floor(m/60),r=m%60;return h+' h'+(r?' '+r+' min':'');}
  function clock(ms){var x=Math.max(0,Math.floor(ms/1000)),h=Math.floor(x/3600);return (h?h+':':'')+z2(Math.floor(x%3600/60))+':'+z2(x%60);}
  function oz2(ml){return String(parseFloat((ml/29.5735).toFixed(2)));}
  function fmtVol(ml,us){return us?oz2(ml)+' fl oz':Math.round(ml)+' ml';}
  function ago(t,now){if(!t)return 'Nothing logged yet';var m=Math.floor((now-t)/MIN);if(m<1)return 'Just now';if(m<60)return m+' min ago';return Math.floor(m/60)+' h '+(m%60)+' min ago';}
  function uid(){return 'e'+Math.random().toString(36).slice(2,10)+Date.now().toString(36).slice(-3);}

  /* ---- seed: Leo has a week of entries; Ivy and Mia start empty (empty states) ---- */
  var TPL=[['diaper',405,{kind:'Wet'}],['bottle',430,{ml:120,contents:['Breast milk']}],['sleep',470,{min:90}],['nursing',600,{left:8,right:5,wb:3400,wa:3460}],['diaper',640,{kind:'Both'}],['solids',690,{what:'Banana',reaction:'Liked it'}],['sleep',740,{min:105}],['bottle',800,{ml:150,contents:['Formula']}],['diaper',860,{kind:'Wet'}],['pumping',900,{side:'Both',ml:90,min:15,sides:{left:50,right:40}}],['nursing',990,{left:6,right:7}],['diaper',1030,{kind:'Dirty'}],['bottle',1080,{ml:130,contents:['Breast milk','Formula'],sources:{'Breast milk':70,'Formula':60}}]];
  function seed(now){
    var out=[],base=dayStart(now),nowMin=Math.floor((now-base)/MIN);
    for(var d=0;d<7;d++){TPL.forEach(function(r,i){
      if(d>0&&(i+d)%7===6)return;
      var m=r[1]+(d*7%25)-10;if(m<0)m=r[1];
      if(d===0&&m>nowMin-10)return;
      var at=dayStart(now-d*DAY)+m*MIN,x=r[2],e={id:uid(),kid:'leo',type:r[0],at:at,end:null,by:(i%3===0?'Cleo':'You'),note:''};
      if(r[0]==='bottle'){e.ml=x.ml+(d%3)*10;e.contents=x.contents;if(x.sources){e.sources={};for(var sk in x.sources)e.sources[sk]=x.sources[sk];e.sources['Breast milk']+=(d%3)*10;}}
      if(r[0]==='diaper')e.kind=x.kind;
      if(r[0]==='sleep'){e.end=at+(x.min+(d%2)*15)*MIN;}
      if(r[0]==='nursing'){e.leftMs=x.left*MIN;e.rightMs=x.right*MIN;e.end=at+(x.left+x.right)*MIN;e.side='Right';if(x.wb){e.wb=x.wb;e.wa=x.wa+(d%3===1?-20:d%3*2);e.wu='kg';}}
      if(r[0]==='solids'){e.what=x.what;e.reaction=x.reaction;}
      if(r[0]==='pumping'){e.side=x.side;e.ml=x.ml;e.min=x.min;e.end=at+x.min*MIN;if(x.sides)e.sides={left:x.sides.left,right:x.sides.right};}
      out.push(e);});}
    return {rev:1,seedDay:dayKey(now),kid:'leo',entries:out,prefs:{us:false,lb:false,band:0,hidden:{}}};
  }
  function read(){
    var raw=null;try{raw=localStorage.getItem(KEY);}catch(e){}
    if(raw&&raw===rawSeen&&mem)return mem;
    var d=null;if(raw){try{d=JSON.parse(raw);}catch(e){d=null;}}
    if(!d)d=mem;
    if(!d||d.seedDay!==dayKey(Date.now())){var keep=d&&d.prefs,ex=d&&d.extra;var carry=d||{};d=seed(Date.now());if(keep)d.prefs=keep;if(ex)d.extra=ex;['meta','names','gone'].forEach(function(k){if(carry[k])d[k]=carry[k];});write(d);return d;}
    mem=d;rawSeen=raw;return d;
  }
  function write(d){d.rev=(d.rev||0)+1;mem=d;try{var s=JSON.stringify(d);localStorage.setItem(KEY,s);rawSeen=s;}catch(e){rawSeen=null;}}


  /* ---------- plan 5.1: advanced bottle / nursing / pumping ---------- */
  var SRC=['Breast milk','Formula',"Cow's milk",'Other'];
  var OWNER_DENSITY='The volume assumes about 1.03 g per ml, an approximation in the range commonly used (1.0–1.03 g/ml); not measured.';
  var OWNER_ACCURACY='A single-feed reading from a home scale can be off by several grams.';
  var FMT_VOL='whole millilitres or ounces with up to two decimals';
  var FMT_W='Enter a weight: kilograms with up to three decimals, or pounds and 0 to 15.99 ounces';
  function pVol(t,u){t=String(t==null?'':t).trim().replace(',','.');if(u==='ml'){return /^\d+$/.test(t)&&parseInt(t,10)>=1?parseInt(t,10):null;}
    if(!/^\d+(\.\d{1,2})?$/.test(t)||parseFloat(t)<0.02)return null;return Math.round(parseFloat(t)*29.5735);}
  function vText(ml,u){return u==='floz'?oz2(ml):String(Math.round(ml));}
  function uLabel(u){return u==='floz'?'fl oz':'ml';}
  function pW(k,l,o,u){var g;if(u==='kg'){var t=String(k||'').trim().replace(',','.');if(!t)return null;if(!/^\d+(\.\d{1,3})?$/.test(t))return {err:FMT_W};g=Math.round(parseFloat(t)*1000);}
    else{var lt=String(l||'').trim(),ot=String(o||'').trim().replace(',','.');if(!lt&&!ot)return null;if((lt&&!/^\d+$/.test(lt))||(ot&&!/^\d+(\.\d{1,2})?$/.test(ot))||parseFloat(ot||'0')>=16)return {err:FMT_W};g=Math.round((parseInt(lt||'0',10)*16+parseFloat(ot||'0'))*28.3495);}
    if(g<200||g>250000)return {err:u==='kg'?'Enter a weight from 0.2 to 250 kg':'Enter a weight from 7 oz to 551 lb'};return {g:g};}
  function wFields(g,u){if(g==null)return {k:'',l:'',o:''};if(u==='kg')return {k:String(parseFloat((g/1000).toFixed(3))),l:'',o:''};var oz=g/28.3495,lb=Math.floor(oz/16);return {k:'',l:String(lb),o:String(parseFloat((oz-lb*16).toFixed(2)))};}
  /* estimate: arithmetic on two typed weights (grams); the API does this for real. null unless both exist and after >= before */
  function milk(wb,wa,lb,us){if(wb==null||wa==null||wa<wb)return null;var g=wa-wb,ml=Math.round(g*100/103),fh=Math.round(g*1e8/(103*295735))/100;
    return {gain:lb?String(parseFloat((g/28.3495).toFixed(2)))+' oz':g+' g',vol:us?String(parseFloat(fh.toFixed(2)))+' fl oz':ml+' ml'};}
  function sumText(ml,u){return 'Total: '+vText(ml,u)+' '+uLabel(u);}
  var H0={
    SRC:SRC,OWNER_DENSITY:OWNER_DENSITY,OWNER_ACCURACY:OWNER_ACCURACY,milk:milk,pVol:pVol,vText:vText,
    toggleLabel:function(type,adv){var m={bottle:['Advanced: volume per source','Use one total'],nursing:['Advanced: weigh before and after','Hide weigh-ins'],pumping:['Advanced: volume per side','Use one total']}[type];return adv?m[1]:m[0];},
    /* does saving the simple form destroy parts? (bottle/pumping only; nursing never deletes a weight) */
    destroys:function(type){return type==='bottle'||type==='pumping';},
    /* descriptors for the advanced sections; pages render them with H.ui */
    items:function(type,f,o){var u=f.u||(o.us?'floz':'ml'),it=[],t;
      if(type==='bottle'){it.push({kind:'select',id:'f-unit',key:'u',label:'Volume unit',options:[['ml','ml'],['floz','fl oz']],value:u});
        it.push({kind:'legend',label:'Contents',hint:'Tick each source and enter its volume'});
        var tot=0;SRC.forEach(function(n,i){var on=(f.c||[]).indexOf(n)>=0,v=(f.sv||{})[n]||'';if(on){var p=pVol(v,u);if(p)tot+=p;}
          it.push({kind:'check',id:'f-src'+i,key:'c',name:n,label:n,checked:on,inputLabel:n+' volume',unit:uLabel(u),value:v,errKey:'sv:'+n});});
        it.push({kind:'alert',errKey:'c'});it.push({kind:'alert',errKey:'sv'});it.push({kind:'total',text:sumText(tot,u),errKey:'vol'});}
      if(type==='pumping'){it.push({kind:'select',id:'f-unit',key:'u',label:'Volume unit',options:[['ml','ml'],['floz','fl oz']],value:u});
        it.push({kind:'legend',label:'Volume per side',hint:'Enter at least one side; leave the other blank'});
        it.push({kind:'input',id:'f-vl',key:'vl',label:'Left volume ('+uLabel(u)+')',mode:'decimal',value:f.vl||'',errKey:'sides',quiet:true});
        it.push({kind:'input',id:'f-vr',key:'vr',label:'Right volume ('+uLabel(u)+')',mode:'decimal',value:f.vr||'',errKey:'sides',quiet:true});it.push({kind:'alert',errKey:'sides'});
        var a=pVol(f.vl,u)||0,b=pVol(f.vr,u)||0;it.push({kind:'total',text:sumText(a+b,u),errKey:'vol'});}
      if(type==='nursing'){var wu=f.wu||(o.lb?'lb':'kg');
        it.push({kind:'select',id:'f-wu',key:'wu',label:'Weight unit (before and after)',options:[['kg','kg'],['lb','lb']],value:wu});
        [['b','before','Weight before'],['a','after','Weight after']].forEach(function(w){it.push({kind:'legend',label:w[2]});
          if(wu==='kg')it.push({kind:'input',id:'f-w'+w[0]+'k',key:'w'+w[0]+'k',label:w[2]+': kilograms',mode:'decimal',value:f['w'+w[0]+'k']||'',errKey:'w'+w[0],quiet:true});
          else{it.push({kind:'input',id:'f-w'+w[0]+'l',key:'w'+w[0]+'l',label:w[2]+': pounds',mode:'numeric',value:f['w'+w[0]+'l']||'',errKey:'w'+w[0],quiet:true});it.push({kind:'input',id:'f-w'+w[0]+'o',key:'w'+w[0]+'o',label:w[2]+': ounces',mode:'decimal',value:f['w'+w[0]+'o']||'',errKey:'w'+w[0],quiet:true});}it.push({kind:'alert',errKey:'w'+w[0]});});
        var B=pW(f.wbk,f.wbl,f.wbo,wu),A=pW(f.wak,f.wal,f.wao,wu),est=null;
        if(B&&A&&B.g!=null&&A.g!=null){est=milk(B.g,A.g,o.lb,o.us);if(!est)est={none:true};}
        var shown=!!(est&&!est.none);
        it.push({kind:'hint',text:'In your weight unit. These weights stay with this session; they are not added to growth. '+OWNER_ACCURACY});
        if(est&&!est.none)it.push({kind:'estimate',line:'Weight gain '+est.gain+', about '+est.vol,small:OWNER_DENSITY});
        else if(est&&est.none)it.push({kind:'estimate',line:'No estimate',small:''});}
      return it;},
    /* validate + build the entry fields from the form; returns {err,patch} */
    collect:function(type,f,o){var u=f.u||(o.us?'floz':'ml'),err={},patch={};
      if(type==='bottle'){var tk=SRC.filter(function(n){return (f.c||[]).indexOf(n)>=0;}),src={},sum=0,bad=false;
        if(!tk.length)err.c='Choose at least one';
        tk.forEach(function(n){var p=pVol((f.sv||{})[n],u);if(p==null){bad=true;err['sv:'+n]=' ';}else{src[n]=p;sum+=p;}});
        if(bad)err.sv='Enter a volume for each source you ticked, in '+FMT_VOL;
        else if(tk.length&&(sum<1||sum>1000))err.vol='The total must be from 1 to 1000 ml';
        patch={sources:src,contents:tk,ml:sum};}
      if(type==='pumping'){var l=pVol(f.vl,u),r=pVol(f.vr,u),ft=String(f.vl||'').trim(),rt=String(f.vr||'').trim(),sd={};
        if((ft&&l==null)||(rt&&r==null)||(l==null&&r==null))err.sides='Enter a volume for at least one side, in '+FMT_VOL;
        else{if(l!=null)sd.left=l;if(r!=null)sd.right=r;var sm=(l||0)+(r||0);if(sm<1||sm>2000)err.vol='The total must be from 1 to 2000 ml';
          patch={sides:sd,ml:sm,side:l!=null&&r!=null?'Both':l!=null?'Left':'Right'};}}
      if(type==='nursing'){var wu=f.wu||(o.lb?'lb':'kg'),B=pW(f.wbk,f.wbl,f.wbo,wu),A=pW(f.wak,f.wal,f.wao,wu);
        if(B&&B.err)err.wb=B.err;if(A&&A.err)err.wa=A.err;
        if(!err.wb&&!err.wa){patch.wb=B?B.g:null;patch.wa=A?A.g:null;patch.wu=wu;}}
      return {err:err,patch:patch};},
    /* initial form values from a stored entry (viewer's units) */
    fill:function(e,o){var f={},u=o.us?'floz':'ml';
      if(e.type==='bottle'&&e.sources){f.adv=true;f.u=u;f.c=e.contents.slice();f.sv={};for(var k in e.sources)f.sv[k]=vText(e.sources[k],u);}
      if(e.type==='pumping'&&e.sides){f.adv=true;f.u=u;f.vl=e.sides.left!=null?vText(e.sides.left,u):'';f.vr=e.sides.right!=null?vText(e.sides.right,u):'';}
      if(e.type==='nursing'){var wu=o.lb?'lb':'kg';f.wu=wu;var b=wFields(e.wb,wu),a=wFields(e.wa,wu);f.wbk=b.k;f.wbl=b.l;f.wbo=b.o;f.wak=a.k;f.wal=a.l;f.wao=a.o;if(e.wb!=null||e.wa!=null)f.adv=true;}
      return f;},
    /* render objects for the pages' templates */
    ui:function(items,f,er,set){return items.map(function(d){var o={id:d.id||'',label:d.label||'',hint:d.hint||'',text:d.text||'',line:d.line||'',small:d.small||'',kind:d.kind,
        isLegend:d.kind==='legend',isHint:d.kind==='hint',isTotal:d.kind==='total',isCheck:d.kind==='check',isSelect:d.kind==='select',isInput:d.kind==='input',isEst:d.kind==='estimate',isAlert:d.kind==='alert',
        err:(d.errKey&&!d.quiet&&er[d.errKey]&&er[d.errKey]!==' ')?er[d.errKey]:(d.kind==='alert'&&er[d.errKey])||'',invalid:!!(d.errKey&&er[d.errKey]),type:'text',mode:d.mode||'text',value:d.value==null?'':d.value,checked:!!d.checked,inputLabel:d.inputLabel||'',unit:d.unit||'',step:false,options:[],isErrFirst:false};
      if(d.kind==='select'){o.options=d.options.map(function(x){return {label:x[1],value:x[0],selected:x[0]===d.value};});o.on=function(ev){set(d.key,ev.target.value);};}
      else if(d.kind==='check'){o.on=function(){var c=(f.c||[]).slice(),i=c.indexOf(d.name);if(i>=0)c.splice(i,1);else c.push(d.name);set('c',c);};o.onVal=function(ev){var m=Object.assign({},f.sv||{});m[d.name]=ev.target.value;set('sv',m);};o.err='';}
      else if(d.kind==='input'){o.on=function(ev){set(d.key,ev.target.value);};}
      return o;});},
    H:null
  };
  var H={ADV:H0,TZ:TZ,kids:function(){var d=read(),g=d.gone||[],n=d.names||{};return KIDS.concat(d.extra||[]).filter(function(k){return g.indexOf(k.id)<0;}).map(function(k){return {id:k.id,name:n[k.id]||k.name};});},addKid:function(name){var d=read();d.extra=d.extra||[];var id='k'+Math.random().toString(36).slice(2,7);d.extra.push({id:id,name:name});d.kid=id;d.prefs.none=false;write(d);return id;},KIDS:KIDS,TYPES:TYPES,NAME:NAME,CONTENTS:CONTENTS,KINDS:KINDS,MIN:MIN,DAY:DAY,FUTURE:FUTURE,MAXRUN:MAXRUN,
    z2:z2,dayKey:dayKey,hm:hm,dayStart:dayStart,heading:heading,fmtDur:fmtDur,clock:clock,fmtVol:fmtVol,ago:ago,
    get:function(){return read();},
    kid:function(){return read().kid;},
    kidName:function(id){var k=H.kids().filter(function(x){return x.id===(id||read().kid);})[0];return k?k.name:'';},
    setKid:function(id){var d=read();d.kid=id;write(d);},
    pref:function(k){return read().prefs[k];},
    setPref:function(k,v){var d=read();d.prefs[k]=v;write(d);},
    hidden:function(kid){return (read().prefs.hidden[kid||read().kid])||[];},
    setHidden:function(kid,list){var d=read();d.prefs.hidden[kid]=list;write(d);},
    entries:function(kid){kid=kid||read().kid;return read().entries.filter(function(e){return e.kid===kid;}).sort(function(a,b){return b.at-a.at;});},
    find:function(id){return read().entries.filter(function(e){return e.id===id;})[0]||null;},
    running:function(kid,type){kid=kid||read().kid;return read().entries.filter(function(e){return e.kid===kid&&e.type===type&&e.end===null;})[0]||null;},
    add:function(e){var d=read();e.id=e.id||uid();e.kid=e.kid||d.kid;e.by=e.by||'You';e.note=e.note||'';d.entries.push(e);write(d);return e.id;},
    patch:function(id,f){var d=read();d.entries.forEach(function(e){if(e.id===id)for(var k in f)e[k]=f[k];});write(d);},
    remove:function(id){var d=read();d.entries=d.entries.filter(function(e){return e.id!==id;});write(d);},
    snapshot:function(){return JSON.stringify(read().entries);},
    restore:function(s){var d=read();d.entries=JSON.parse(s);write(d);},
    reset:function(){var d=seed(Date.now());d.prefs=read().prefs;write(d);},
    /* child profile: metadata the API keeps per child. Defaults are sample data. */
    kidMeta:function(id){var d=read(),m=(d.meta||{})[id],D={ivy:{born:'2026-07-11',gaw:40,gad:0,sex:''},mia:{born:'2026-09-22',gaw:40,gad:0,sex:''},leo:{born:'2026-07-11',gaw:35,gad:2,sex:''}}[id]||{born:'',gaw:40,gad:0,sex:''};return Object.assign({zone:TZ},D,m||{},{name:H.kidName(id)});},
    /* returns {err:{field:msg}} or {ok:true}; name is stored separately so Timeline/Home pick it up */
    kidErrors:function(f){var e={},nm=String(f.name||'').trim(),now=Date.now(),b=String(f.born||''),gw=Number(f.gaw),gd=Number(f.gad);
      if(!nm||nm.length>40)e.name='Enter a name of up to 40 characters.';
      var bt=/^\d{4}-\d{2}-\d{2}$/.test(b)?Date.parse(b+'T12:00:00Z'):NaN;
      if(isNaN(bt))e.born='Enter a birth date, like 2026-07-11.';else if(bt>now+DAY)e.born='The birth date can’t be in the future.';else if(bt<now-18*365*DAY)e.born='Enter a birth date in the last 18 years.';
      if(!(gw>=22&&gw<=44&&gw%1===0))e.gaw='Enter whole weeks from 22 to 44.';
      if(!(gd>=0&&gd<=6&&gd%1===0))e.gad='Enter whole days from 0 to 6.';
      if(['','Female','Male','Another'].indexOf(f.sex||'')<0)e.sex='Choose one of the options.';
      var zn=String(f.zone||'').trim();try{new Intl.DateTimeFormat('en',{timeZone:zn});}catch(x){e.zone='Enter a time zone name, like America/New_York.';}if(!zn)e.zone='Enter a time zone name, like America/New_York.';
      return e;},
    /* one field set for add and edit: name, birth date, sex (optional), weeks and days at birth, time zone */
    setKidMeta:function(id,f){var e=H.kidErrors(f);if(Object.keys(e).length)return {err:e};
      var nm=String(f.name||'').trim(),d=read();d.meta=d.meta||{};d.meta[id]={born:String(f.born||''),gaw:Number(f.gaw),gad:Number(f.gad),sex:f.sex||'',zone:String(f.zone||'').trim()};d.names=d.names||{};d.names[id]=nm;write(d);return {ok:true};},
    deviceTZ:function(){try{return Intl.DateTimeFormat().resolvedOptions().timeZone||TZ;}catch(x){return TZ;}},
    createKid:function(f){var e=H.kidErrors(f);if(Object.keys(e).length)return {err:e};var nm=String(f.name||'').trim(),id=H.addKid(nm);H.setKidMeta(id,f);return {ok:true,id:id,name:nm};},
    /* deletes every entry and the child itself; callers must have asked twice. Returns the number of entries removed. */
    deleteKid:function(id){var d=read(),n=d.entries.filter(function(e){return e.kid===id;}).length;d.entries=d.entries.filter(function(e){return e.kid!==id;});d.gone=(d.gone||[]).concat([id]);if(d.meta)delete d.meta[id];if(window.HBTM)window.HBTM.dropKid(id);if(window.HBTMS)window.HBTMS.dropKid(id);
      var left=H.kids();d.kid=left.length?(d.kid===id?left[0].id:d.kid):d.kid;if(!left.length)d.prefs.none=true;write(d);return n;},
    entryCount:function(id){return read().entries.filter(function(e){return e.kid===id;}).length;},
    clear:function(kid){var d=read();d.entries=d.entries.filter(function(e){return e.kid!==kid;});write(d);},
    /* time entry: "HH:MM" on the kid's today -> ms, or an error string. Rule: at most 5 min in the future. */
    parseTime:function(v,now){if(!/^\d{2}:\d{2}$/.test(v||''))return {err:'Enter a time, like 13:05.'};var a=v.split(':').map(Number);if(a[0]>23||a[1]>59)return {err:'Enter a time, like 13:05.'};var t=dayStart(now)+(a[0]*60+a[1])*MIN;if(t>now+FUTURE)return {err:'The time can’t be more than 5 minutes ahead.'};return {t:t};},
    /* Prototype failure simulator: prefs.fail = off|network|server|conflict. One failure, then it resets so "Try again" works.
       conflict mimics the API's 409s (already_running / not_running) or a row deleted by someone else. Returns null or {code,msg}. */
    ERR:{network:'No connection, so this wasn’t saved. Try again when you’re back online.',server:'Something went wrong on our side and this wasn’t saved. Try again in a moment.',already_running:'Someone else already started this on another phone. It’s shown now.',not_running:'Someone else already ended this on another phone. It’s updated now.',gone:'Someone deleted this entry, so it can’t be saved. It’s been removed from your list.'},
    tryWrite:function(act,id,kid){
      var d=read(),m=d.prefs.fail;if(!m||m==='off')return null;
      function consume(){var x=read();x.prefs.fail='off';write(x);}
      if(m==='network'||m==='server'){consume();return {code:m,msg:H.ERR[m]};}
      kid=kid||d.kid;var e,now=Date.now();
      if(act==='start-sleep'||act==='start-nursing'){var ty=act==='start-sleep'?'sleep':'nursing';if(H.running(kid,ty))return null;consume();
        H.add(ty==='sleep'?{type:'sleep',at:now-2*MIN,end:null,by:'Cleo',kid:kid}:{type:'nursing',at:now-2*MIN,end:null,side:'Left',sideSince:now-2*MIN,leftMs:0,rightMs:0,by:'Cleo',kid:kid});return {code:'already_running',msg:H.ERR.already_running};}
      if(act==='end'){e=H.find(id);if(!e||e.end!==null)return null;consume();var f={end:now-MIN};if(e.type==='nursing'){var x=Math.max(0,now-MIN-e.sideSince);f.leftMs=(e.leftMs||0)+(e.side==='Left'?x:0);f.rightMs=(e.rightMs||0)+(e.side==='Right'?x:0);}H.patch(id,f);return {code:'not_running',msg:H.ERR.not_running};}
      if(act==='edit'){e=H.find(id);if(!e)return null;consume();H.remove(id);return {code:'gone',msg:H.ERR.gone};}
      return null;},
    /* nursing per-side ms at "now" (a running session adds the current side) */
    nursingMs:function(e,now){var l=e.leftMs||0,r=e.rightMs||0;if(e.end===null){var x=now-e.sideSince;if(e.side==='Left')l+=x;else r+=x;}return {l:l,r:r};},
    /* one wording for a row, used by Timeline and by Home's “Latest” list */
    line:function(e,us,now){
      var t=e.type,sub='',title=NAME[t];
      if(t==='bottle')sub=fmtVol(e.ml,us)+', '+e.contents.map(function(c){return c==="Cow's milk"?c.toLowerCase():c.toLowerCase();}).join(' + ');
      else if(t==='diaper')sub=e.kind.toLowerCase();
      else if(t==='solids')sub=(e.what||'')+(e.reaction?' · '+e.reaction.toLowerCase():'');
      else if(t==='pumping')sub=e.min+' min, '+fmtVol(e.ml,us)+' ('+(e.sides?[e.sides.left!=null?'L '+fmtVol(e.sides.left,us):'',e.sides.right!=null?'R '+fmtVol(e.sides.right,us):''].filter(Boolean).join(' · '):e.side.toLowerCase())+')';
      else if(t==='sleep')sub=e.end===null?'Asleep since '+hm(e.at):hm(e.at)+'–'+hm(e.end)+' ('+fmtDur((e.end-e.at)/MIN)+')';
      else if(t==='nursing'){if(e.end===null)sub=e.side+' side since '+hm(e.sideSince);else{var p=[];if(e.leftMs>=MIN/2)p.push('L '+Math.round(e.leftMs/MIN)+' min');if(e.rightMs>=MIN/2)p.push('R '+Math.round(e.rightMs/MIN)+' min');sub=p.join(' · ')||'Under a minute';var mk=H.ADV.milk(e.wb,e.wa,H.pref('lb'),us);if(mk)sub+=', weight gain '+mk.gain+', about '+mk.vol;}}
      else if(t==='symptom'){title=window.HBTM?window.HBTM.symptomText(e):e.sym;sub='';}
      if(t==='diaper'){title='Diaper '+sub;sub='';}
      if(e.note)sub=sub?sub+' · '+e.note:e.note;
      var aria=title+(sub?', '+sub:'')+' at '+hm(e.at)+'.';
      return {title:title,sub:sub,time:hm(e.at),aria:aria,note:e.note||''};
    },
    /* kid-local day totals (sleep clipped to the day, running counts up to now; nursing counts sessions that start that day) */
    totals:function(kid,key,now){
      var T={bottles:{n:0,ml:0},nursing:{n:0,min:0},diapers:{n:0,wet:0,dirty:0,both:0,dry:0},sleepMin:0,sleepN:0,meals:0,pump:{n:0,ml:0},count:0};
      var ds=null;
      H.entries(kid).forEach(function(e){
        if(e.type==='sleep'){var s=e.at,en=e.end===null?now:e.end;if(ds===null){}
          var a=dayStartOfKey(key),b=a+DAY;var lo=Math.max(s,a),hi=Math.min(en,b);if(hi>lo){T.sleepMin+=(hi-lo)/MIN;}if(dayKey(e.at)===key){T.sleepN++;T.count++;}return;}
        if(dayKey(e.at)!==key||e.type==='symptom')return;T.count++;
        if(e.type==='bottle'){T.bottles.n++;T.bottles.ml+=e.ml;}
        else if(e.type==='nursing'){var m=H.nursingMs(e,now);T.nursing.n++;T.nursing.min+=(m.l+m.r)/MIN;}
        else if(e.type==='diaper'){T.diapers.n++;T.diapers[e.kind.toLowerCase()]++;}
        else if(e.type==='solids')T.meals++;
        else if(e.type==='pumping'){T.pump.n++;T.pump.ml+=e.ml;}
      });return T;},
    totalsText:function(T,us){var p=[];if(T.bottles.n)p.push(T.bottles.n+(T.bottles.n===1?' bottle':' bottles')+' ('+fmtVol(T.bottles.ml,us)+')');if(T.nursing.n)p.push(T.nursing.n+' nursing ('+fmtDur(T.nursing.min)+')');if(T.diapers.n)p.push(T.diapers.n+(T.diapers.n===1?' diaper':' diapers'));if(T.sleepN||T.sleepMin)p.push(fmtDur(T.sleepMin)+' sleep');if(T.meals)p.push(T.meals+' solids');if(T.pump.n)p.push(T.pump.n+' pumping');return p.join(' · ')||'No entries';},
    lastAt:function(kid,type){var r=H.entries(kid).filter(function(e){return e.type===type;})[0];return r?(r.end===null?r.at:r.at):null;},
    /* the real "since last" for a tracker: sleep & nursing measure from their end */
    sinceAt:function(kid,type){var r=H.entries(kid).filter(function(e){return e.type===type&&e.end!==null||e.type===type&&type!=='sleep'&&type!=='nursing'&&type!=='pumping';}).sort(function(a,b){return (b.end||b.at)-(a.end||a.at);})[0];return r?(r.end||r.at):null;}
  };
  function dayStartOfKey(key){var t=Date.now();for(var i=0;i<400;i++){var x=t-i*DAY;if(dayKey(x)===key)return dayStart(x);}return 0;}
  H.dayStartOfKey=dayStartOfKey;
  window.HBT=H;
})();
