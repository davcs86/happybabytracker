/* hbt-ms.js — plan 7 sample store: CDC milestone catalog, marks, custom milestones, age basis, PDF export validation.
   Prototype stand-in for the API; localStorage key hbt.ms.v1 (memory fallback). The UI never interprets a value:
   current/next list, age basis, the one-tap rule and the D3 "first mark turns the module on" rule live here ONCE.
   CATALOG: transcribed from the CDC checklist pages via a summarizing page fetch; the API's catalog CSV is the source of truth. */
(function(){
  var KEY='hbt.ms.v1',mem=null,CAP=1000,DAY=864e5;
  var DOMAINS=[['se','Social/Emotional'],['lc','Language/Communication'],['cg','Cognitive (learning, thinking, problem-solving)'],['mv','Movement/Physical Development']];
  var CAT=[{"age": 2, "title": "By 2 months", "d": {"se": ["Calms down when spoken to or picked up", "Looks at your face", "Seems happy to see you when you walk up to her", "Smiles when you talk to or smile at her"], "lc": ["Makes sounds other than crying", "Reacts to loud sounds"], "cg": ["Watches you as you move", "Looks at a toy for several seconds"], "mv": ["Holds head up when on tummy", "Moves both arms and both legs", "Opens hands briefly"]}}, {"age": 4, "title": "By 4 months", "d": {"se": ["Smiles on his own to get your attention", "Chuckles (not yet a full laugh) when you try to make him laugh", "Looks at you, moves, or makes sounds to get or keep your attention"], "lc": ["Makes sounds like “oooo”, “aahh” (cooing)", "Makes sounds back when you talk to him", "Turns head towards the sound of your voice"], "cg": ["If hungry, opens mouth when he sees breast or bottle", "Looks at her hands with interest"], "mv": ["Holds head steady without support when you are holding him", "Holds a toy when you put it in his hand", "Uses his arm to swing at toys", "Brings hands to mouth", "Pushes up onto elbows/forearms when on tummy"]}}, {"age": 6, "title": "By 6 months", "d": {"se": ["Knows familiar people", "Likes to look at self in a mirror", "Laughs"], "lc": ["Takes turns making sounds with you", "Blows “raspberries” (sticks tongue out and blows)", "Makes squealing noises"], "cg": ["Puts things in her mouth to explore them", "Reaches to grab a toy she wants", "Closes lips to show she doesn’t want more food"], "mv": ["Rolls from tummy to back", "Pushes up with straight arms when on tummy", "Leans on hands to support herself when sitting"]}}, {"age": 9, "title": "By 9 months", "d": {"se": ["Is shy, clingy, or fearful around strangers", "Shows several facial expressions, like happy, sad, angry, and surprised", "Looks when you call her name", "Reacts when you leave (looks, reaches for you, or cries)", "Smiles or laughs when you play peek-a-boo"], "lc": ["Makes a lot of different sounds like “mamamama” and “bababababa”", "Lifts arms up to be picked up"], "cg": ["Looks for objects when dropped out of sight (like his spoon or toy)", "Bangs two things together"], "mv": ["Gets to a sitting position by herself", "Moves things from one hand to her other hand", "Uses fingers to “rake” food towards himself", "Sits without support"]}}, {"age": 12, "title": "By 1 year", "d": {"se": ["Plays games with you, like pat-a-cake"], "lc": ["Waves “bye-bye”", "Calls a parent “mama” or “dada” or another special name", "Understands “no” (pauses briefly or stops when you say it)"], "cg": ["Puts something in a container, like a block in a cup", "Looks for things he sees you hide, like a toy under a blanket"], "mv": ["Pulls up to stand", "Walks, holding on to furniture", "Drinks from a cup without a lid, as you hold it", "Picks things up between thumb and pointer finger, like small bits of food"]}}, {"age": 15, "title": "By 15 months", "d": {"se": ["Copies other children while playing, like taking toys out of a container when another child does", "Shows you an object she likes", "Claps when excited", "Hugs stuffed doll or other toy", "Shows you affection (hugs, cuddles, or kisses you)"], "lc": ["Tries to say one or two words besides “mama” or “dada,” like “ba” for ball or “da” for dog", "Looks at a familiar object when you name it", "Follows directions given with both a gesture and words. For example, he gives you a toy when you hold out your hand and say, “Give me the toy.”", "Points to ask for something or to get help"], "cg": ["Tries to use things the right way, like a phone, cup, or book", "Stacks at least two small objects, like blocks"], "mv": ["Takes a few steps on his own", "Uses fingers to feed herself some food"]}}, {"age": 18, "title": "By 18 months", "d": {"se": ["Moves away from you, but looks to make sure you are close by", "Points to show you something interesting", "Puts hands out for you to wash them", "Looks at a few pages in a book with you", "Helps you dress him by pushing arm through sleeve or lifting up foot"], "lc": ["Tries to say three or more words besides “mama” or “dada”", "Follows one-step directions without any gestures, like giving you the toy when you say, “Give it to me.”"], "cg": ["Copies you doing chores, like sweeping with a broom", "Plays with toys in a simple way, like pushing a toy car"], "mv": ["Walks without holding on to anyone or anything", "Scribbles", "Drinks from a cup without a lid and may spill sometimes", "Feeds himself with his fingers", "Tries to use a spoon", "Climbs on and off a couch or chair without help"]}}, {"age": 24, "title": "By 2 years", "d": {"se": ["Notices when others are hurt or upset, like pausing or looking sad when someone is crying", "Looks at your face to see how to react in a new situation"], "lc": ["Points to things in a book when you ask, like “Where is the bear?”", "Says at least two words together, like “More milk.”", "Points to at least two body parts when you ask him to show you", "Uses more gestures than just waving and pointing, like blowing a kiss or nodding yes"], "cg": ["Holds something in one hand while using the other hand; for example, holding a container and taking the lid off", "Tries to use switches, knobs, or buttons on a toy", "Plays with more than one toy at the same time, like putting toy food on a toy plate"], "mv": ["Kicks a ball", "Runs", "Walks (not climbs) up a few stairs with or without help", "Eats with a spoon"]}}, {"age": 30, "title": "By 30 months", "d": {"se": ["Plays next to other children and sometimes plays with them", "Shows you what she can do by saying, “Look at me!”", "Follows simple routines when told, like helping to pick up toys when you say, “It’s clean-up time.”"], "lc": ["Says about 50 words", "Says two or more words together, with one action word, like “Doggie run”", "Names things in a book when you point and ask, “What is this?”", "Says words like “I,” “me,” or “we”"], "cg": ["Uses things to pretend, like feeding a block to a doll as if it were food", "Shows simple problem-solving skills, like standing on a small stool to reach something", "Follows two-step instructions like “Put the toy down and close the door.”", "Shows he knows at least one color, like pointing to a red crayon when you ask, “Which one is red?”"], "mv": ["Uses hands to twist things, like turning doorknobs or unscrewing lids", "Takes some clothes off by himself, like loose pants or an open jacket", "Jumps off the ground with both feet", "Turns book pages, one at a time, when you read to her"]}}, {"age": 36, "title": "By 3 years", "d": {"se": ["Calms down within 10 minutes after you leave her, like at a childcare drop off", "Notices other children and joins them to play"], "lc": ["Talks with you in conversation using at least two back-and-forth exchanges", "Asks “who,” “what,” “where,” or “why” questions, like “Where is mommy/daddy?”", "Says what action is happening in a picture or book when asked, like “running,” “eating,” or “playing”", "Says first name, when asked", "Talks well enough for others to understand, most of the time"], "cg": ["Draws a circle, when you show him how", "Avoids touching hot objects, like a stove, when you warn her"], "mv": ["Strings items together, like large beads or macaroni", "Puts on some clothes by himself, like loose pants or a jacket", "Uses a fork"]}}, {"age": 48, "title": "By 4 years", "d": {"se": ["Pretends to be something else during play (teacher, superhero, dog)", "Asks to go play with children if none are around, like “Can I play with Alex?”", "Comforts others who are hurt or sad, like hugging a crying friend", "Avoids danger, like not jumping from tall heights at the playground", "Likes to be a “helper”", "Changes behavior based on where she is (place of worship, library, playground)"], "lc": ["Says sentences with four or more words", "Says some words from a song, story, or nursery rhyme", "Talks about at least one thing that happened during her day, like “I played soccer.”", "Answers simple questions like “What is a coat for?” or “What is a crayon for?”"], "cg": ["Names a few colors of items", "Tells what comes next in a well-known story", "Draws a person with three or more body parts"], "mv": ["Catches a large ball most of the time", "Serves herself food or pours water, with adult supervision", "Unbuttons some buttons", "Holds crayon or pencil between fingers and thumb (not a fist)"]}}, {"age": 60, "title": "By 5 years", "d": {"se": ["Follows rules or takes turns when playing games with other children", "Sings, dances, or acts for you", "Does simple chores at home, like matching socks or clearing the table after eating"], "lc": ["Tells a story she heard or made up with at least two events. For example, a cat was stuck in a tree and a firefighter saved it", "Answers simple questions about a book or story after you read or tell it to him", "Keeps a conversation going with more than three back-and-forth exchanges", "Uses or recognizes simple rhymes (bat-cat, ball-tall)"], "cg": ["Counts to 10", "Names some numbers between 1 and 5 when you point to them", "Uses words about time, like “yesterday,” “tomorrow,” “morning,” or “night”", "Pays attention for 5 to 10 minutes during activities. For example, during story time or making arts and crafts (screen time does not count)", "Writes some letters in her name", "Names some letters when you point to them"], "mv": ["Buttons some buttons", "Hops on one foot"]}}];
  var SCREEN='Learn the Signs. Act Early. materials are not a substitute for standardized, validated developmental screening tools.';
  var ATTR='Source: CDC, Learn the Signs. Act Early. milestone checklists. CDC does not endorse this application or its use of the material. The material is available free of charge at https://www.cdc.gov/act-early/milestones/index.html.';
  var ERR={already:'Already marked as reached',cap:'This child has the maximum number of recorded milestones',text:'Enter a milestone of up to 200 characters',
    date:'Enter a date that is not in the future and not before the birth date',note:'Enter a note of up to 500 characters',
    network:'The app cannot reach its server, try again soon',server:'Something went wrong, try again',gone:'This milestone no longer exists',
    xFrom:'Enter a start date that is not before the birth date',xTo:'Enter an end date from the start date to today, at most 93 days in all',xMods:'Choose at least one module',
    busy:'The PDF cannot be made right now, try again in a moment'};
  var MODS=[['bottles','Bottle'],['nursing','Nursing'],['pumping','Pumping'],['diapers','Diaper'],['sleep','Sleep'],['feeding','Feeding'],['milestones','Milestones'],['growth','Growth'],['vaccination','Vaccination'],['sickness','Sickness'],['medication','Medication']];
  var MON=['January','February','March','April','May','June','July','August','September','October','November','December'];
  function H(){return window.HBT;}
  function uid(p){return p+Math.random().toString(36).slice(2,8);}
  function today(){return H().dayKey(Date.now());}
  function ymd(s){var a=String(s||'').split('-');return [+a[0],+a[1],+a[2]];}
  function validDate(s){if(!/^\d{4}-\d{2}-\d{2}$/.test(s||''))return false;var p=ymd(s),d=new Date(Date.UTC(p[0],p[1]-1,p[2]));return d.getUTCFullYear()===p[0]&&d.getUTCMonth()===p[1]-1&&d.getUTCDate()===p[2];}
  function fmtDate(s){var p=ymd(s);return p[2]+' '+MON[p[1]-1]+' '+p[0];}
  function addDays(s,n){var p=ymd(s),d=new Date(Date.UTC(p[0],p[1]-1,p[2]+n));return d.toISOString().slice(0,10);}
  function diffDays(a,b){var x=ymd(a),y=ymd(b);return Math.round((Date.UTC(y[0],y[1]-1,y[2])-Date.UTC(x[0],x[1]-1,x[2]))/DAY);}
  /* whole months and leftover days between two civil dates (a<=b) */
  function span(a,b){var x=ymd(a),y=ymd(b),m=(y[0]-x[0])*12+(y[1]-x[1]),dd=y[2]-x[2];if(dd<0){m--;var pm=new Date(Date.UTC(y[0],y[1]-1,0)).getUTCDate();dd=(pm-x[2])+y[2];if(dd<0)dd=0;}return {months:m,days:dd};}
  function ageText(sp){var y=Math.floor(sp.months/12),m=sp.months%12,o=[];if(y)o.push(y+(y===1?' year':' years'));if(m)o.push(m+(m===1?' month':' months'));if(!y&&!m||sp.months<12)o.push(sp.days+(sp.days===1?' day':' days'));if(!o.length)o.push('0 days');return o.join(' ');}
  function fifth(born){var p=ymd(born),d=new Date(Date.UTC(p[0]+5,p[1]-1,p[2]));if(p[1]===2&&p[2]===29&&d.getUTCMonth()!==1)return (p[0]+5)+'-03-01';return d.toISOString().slice(0,10);}

  function blank(){return {rev:1,marks:[],mod:{}};}
  function read(){var raw=null;try{raw=localStorage.getItem(KEY);}catch(e){}var d=null;if(raw){try{d=JSON.parse(raw);}catch(e){}}if(!d&&mem)d=mem;if(!d||!d.marks){d=blank();write(d);}return d;}
  function write(d){mem=d;try{localStorage.setItem(KEY,JSON.stringify(d));}catch(e){}}
  function seed(){var d=read();if(d.seeded)return d;d.seeded=1;var now=today();
    /* Leo (born 2026-07-11, 35w2d): marks from the 2- and 4-month lists so the page is not empty; Ivy none. */
    d.marks=[{id:uid('m'),kid:'leo',item:'2m-se-2',text:'',date:addDays(now,-70),note:''},{id:uid('m'),kid:'leo',item:'4m-lc-1',text:'',date:addDays(now,-20),note:'At the park'},{id:uid('m'),kid:'leo',item:'4m-mv-4',text:'',date:addDays(now,-3),note:'Chewing on his fist'},{id:uid('m'),kid:'leo',item:'',text:'Grabbed Grandpa’s finger',date:addDays(now,-10),note:''}];write(d);return d;}
  function itemsOf(list){var out=[];DOMAINS.forEach(function(dm){(list.d[dm[0]]||[]).forEach(function(t,i){out.push({id:list.age+'m-'+dm[0]+'-'+(i+1),text:t,domain:dm[0],domainLabel:dm[1],age:list.age,title:list.title});});});return out;}
  var INDEX={};CAT.forEach(function(l){itemsOf(l).forEach(function(it){INDEX[it.id]=it;});});
  function ago(mo,dd){var p=ymd(today()),d=new Date(Date.UTC(p[0],p[1]-1-mo,p[2]));if(d.getUTCDate()!==p[2])d=new Date(Date.UTC(p[0],p[1]-mo,0));return addDays(d.toISOString().slice(0,10),-(dd||0));}
  var SIMS={'real':{label:'Stored birth date',real:true},'9m':{label:'9 months',mo:9,dd:3,gaw:40,gad:0},'u2':{label:'Under 2 months',mo:0,dd:20,gaw:40,gad:0},'pre':{label:'Preterm (32w3d)',mo:3,dd:4,gaw:32,gad:3},'4y':{label:'4 years 11 months',mo:59,dd:5,gaw:40,gad:0},'o5':{label:'After 5th birthday',mo:62,dd:9,gaw:40,gad:0},'30m':{label:'30 months (module off)',mo:30,dd:2,gaw:40,gad:0}};
  var sim=null;
  function meta(kid){var m=H().kidMeta(kid);return sim?Object.assign({},m,{born:sim.born,gaw:sim.gaw,gad:sim.gad||0}):m;}

  /* The single age rule: corrected age while born before 37 weeks 0 days and younger than 2 years; the catalog follows it. */
  function age(kid,asOf){var m=meta(kid),t=asOf||today(),b=m.born;if(!b||b>t)return {ok:false};
    var chron=span(b,t),pre=(m.gaw||40)<37,out={ok:true,born:b,chron:chron,chronText:ageText(chron),basisMonths:chron.months,corrected:null,before:false,basis:'chronological'};
    if(pre&&chron.months<24){var gap=(40-(m.gaw||40))*7-(m.gad||0);var exp=addDays(b,gap);out.basis='corrected';
      if(exp>t){out.before=true;out.basisMonths=-1;out.text='before expected term';}
      else{var c=span(exp,t);out.corrected=c;out.basisMonths=c.months;out.text=ageText(c);}}
    var fif=fifth(b);out.over5=t>=fif;out.fifth=fif;
    out.line=out.before?out.chronText+' (before expected term)':(out.corrected?out.chronText+' (corrected '+out.text+')':out.chronText);
    return out;}
  function lists(a){if(!a.ok||a.over5)return null;var cur=null,nxt=null;CAT.forEach(function(l){if(l.age<=a.basisMonths)cur=l.age;else if(nxt===null)nxt=l.age;});return {current:cur,next:cur===null?nxt:null,open:cur!==null?cur:nxt};}

  function marksOf(kid){var b=sim?sim.born:'';return read().marks.filter(function(x){return x.kid===kid&&(!b||x.date>=b);});}
  function reachedByItem(kid){var o={};marksOf(kid).forEach(function(x){if(x.item)o[x.item]=x;});return o;}
  function checkDate(date,kid){var b=meta(kid).born;return validDate(date)&&date<=today()&&(!b||date>=b);}
  function checkNote(n){return String(n||'').length<=500;}

  var M={ERR:ERR,DOMAINS:DOMAINS,CAT:CAT,MODS:MODS,SCREEN:SCREEN,ATTR:ATTR,CAP:CAP,fmtDate:fmtDate,validDate:validDate,addDays:addDays,diffDays:diffDays,today:today,age:age,lists:lists,item:function(id){return INDEX[id]||null;},
    seed:seed,
    marks:function(kid){seed();return marksOf(kid);},
    reached:function(kid){seed();return reachedByItem(kid);},
    count:function(kid){seed();return marksOf(kid).length;},
    find:function(id){seed();return read().marks.filter(function(x){return x.id===id;})[0]||null;},
    /* mark(kid,{item|text,date,note,tap}) → {ok,id,turnedOn} | {err:{field:msg}} | {rowErr,msg} */
    mark:function(kid,f){seed();var d=read(),e={},it=f.item?INDEX[f.item]:null;
      if(f.item&&!it)return {gone:true};
      if(it&&reachedByItem(kid)[it.id])return {rowErr:ERR.already};
      if(marksOf(kid).length>=CAP)return {rowErr:ERR.cap,pageErr:ERR.cap};
      var date=f.tap?today():String(f.date||'').trim(),text=String(f.text||'').trim(),note=String(f.note||'').trim();
      if(!it&&(!text||text.length>200))e.text=ERR.text;
      if(!checkDate(date,kid))e.date=ERR.date;
      if(!checkNote(note))e.note=ERR.note;
      if(Object.keys(e).length)return {err:e};
      var r={id:uid('m'),kid:kid,item:it?it.id:'',text:it?'':text,date:date,note:note};d.marks.push(r);
      /* D3: the first mark for a child whose milestones module is off turns it on (stored as an override) */
      var turnedOn=false,h=H().get().prefs.hidden[kid];if(h&&!Array.isArray(h)&&h.milestones===false){var n=Object.assign({},h);n.milestones=true;H().setHidden(kid,n);turnedOn=true;}
      else if(M.offByDefault(kid)&&(!h||Array.isArray(h)||h.milestones===undefined)){var n2=(h&&!Array.isArray(h))?Object.assign({},h):{};n2.milestones=true;H().setHidden(kid,n2);turnedOn=true;}
      write(d);return {ok:true,id:r.id,turnedOn:turnedOn};},
    edit:function(kid,id,f){seed();var d=read(),r=d.marks.filter(function(x){return x.id===id&&x.kid===kid;})[0],e={};if(!r)return {gone:true};
      var date=f.date===undefined?r.date:String(f.date||'').trim(),text=f.text===undefined?r.text:String(f.text||'').trim(),note=f.note===undefined?r.note:String(f.note||'').trim();
      if(!r.item&&(!text||text.length>200))e.text=ERR.text;if(!checkDate(date,kid))e.date=ERR.date;if(!checkNote(note))e.note=ERR.note;
      if(Object.keys(e).length)return {err:e};r.date=date;if(!r.item)r.text=text;r.note=note;write(d);return {ok:true};},
    remove:function(kid,id){seed();var d=read(),n=d.marks.length;d.marks=d.marks.filter(function(x){return !(x.id===id&&x.kid===kid);});if(d.marks.length===n)return {gone:true};write(d);return {ok:true};},
    /* module state as Kid Home computes it: default on below 24 months, off from 24 months (chronological band), override wins */
    offByDefault:function(kid){var a=age(kid);return a.ok&&a.chron.months>=24;},
    setSim:function(s){sim=s||null;},
    SIMS:SIMS,ago:ago,
    /* prototype-only: the sample child scenario the panel is showing; shared by the Milestones, form and export pages */
    /* default scenario 'real': the child's own stored birth date and gestation (same data as Home, Timeline, Growth, Vaccines).
       The other keys are demo age bands for review; they override the birth date for the current child only. */
    useSim:function(){var d=read(),k=d.simKey||'real',s=SIMS[k]||SIMS.real,m;
      if(s.real){m=H().kidMeta(H().kid());sim={born:m.born,gaw:m.gaw,gad:m.gad||0,retired:!!d.simRetired,key:'real'};return 'real';}
      sim={born:ago(s.mo,s.dd),gaw:s.gaw,gad:s.gad,retired:!!d.simRetired,key:k};return k;},
    simKey:function(){return read().simKey||'real';},
    setSimKey:function(k){var d=read();d.simKey=k;write(d);M.useSim();var kid=H().kid(),h=H().get().prefs.hidden[kid];if(h&&!Array.isArray(h)&&h.milestones!==undefined){var n=Object.assign({},h);delete n.milestones;H().setHidden(kid,n);}},
    setSimOn:function(b){var kid=H().kid(),h=H().get().prefs.hidden[kid];var n=(h&&!Array.isArray(h))?Object.assign({},h):{};n.milestones=!!b;H().setHidden(kid,n);},
    setSimRetired:function(b){var d=read();d.simRetired=!!b;write(d);M.useSim();},
    setSimCustom:function(b){var d=read();d.simCustom=b;write(d);},
    moduleOn:function(kid){var h=H().get().prefs.hidden[kid];if(!h||Array.isArray(h))h={};if(h.milestones!==undefined)return !!h.milestones;return !M.offByDefault(kid);},
    /* Milestones page view-model */
    view:function(kid){seed();var a=age(kid),L=lists(a),rc=reachedByItem(kid),all=marksOf(kid);
      var cat=a.over5?null:CAT.map(function(l){var open=L&&L.open===l.age;var label=l.title+(L&&L.current===l.age?' · Current list for this age':(L&&L.next===l.age?' · Next list':''));
        return {age:l.age,title:l.title,label:label,open:open,
          domains:DOMAINS.map(function(dm){return {code:dm[0],label:dm[1],items:(l.d[dm[0]]||[]).map(function(t,i){var id=l.age+'m-'+dm[0]+'-'+(i+1),m=rc[id];return {id:id,text:t,reached:!!m,markId:m?m.id:'',date:m?fmtDate(m.date):'',note:m?m.note:''};})};})};});
      var reachedRows=all.filter(function(x){return x.item&&INDEX[x.item];}).map(function(x){var it=INDEX[x.item];return {id:x.id,text:it.text,sub:it.title+' · '+it.domainLabel,date:fmtDate(x.date),sort:x.date};}).sort(function(a,b){return a.sort<b.sort?1:-1;});
      var showC=read().simCustom!==false;var custom=!showC?[]:all.filter(function(x){return !x.item;}).map(function(x){return {id:x.id,text:x.text,date:fmtDate(x.date),sort:x.date};}).sort(function(a,b){return a.sort<b.sort?1:-1;});
      return {age:a,lists:L,jump:L?(L.current!==null?{href:'#list-'+L.current,label:'Go to the current list'}:{href:'#list-'+L.next,label:'Go to the next list'}):null,cat:cat,reached:reachedRows,custom:custom,retired:(sim&&sim.retired)?[{id:'retired-1',text:'Sits with support for a few seconds',date:fmtDate(addDays(today(),-40))}]:[],moduleOn:M.moduleOn(kid)};},
    /* export */
    born:function(kid){return meta(kid).born;},
    exportDefaults:function(kid){var b=meta(kid).born,t=today(),from=addDays(t,-29);if(b&&from<b)from=b;return {from:from,to:t};},
    exportCheck:function(kid,f){var b=meta(kid).born,t=today(),e={},from=String(f.from||''),to=String(f.to||'');
      if(!validDate(from)||(b&&from<b))e.from=ERR.xFrom;
      if(!validDate(to)||to>t||(!e.from&&to<from)||(!e.from&&diffDays(from,to)+1>93))e.to=ERR.xTo;
      if(!(f.modules||[]).length)e.modules=ERR.xMods;
      return e;},
    snapshot:function(){return JSON.stringify(read());},restore:function(s){try{write(JSON.parse(s));}catch(e){}},
    dropKid:function(kid){var d=read();d.marks=d.marks.filter(function(x){return x.kid!==kid;});write(d);},
    reset:function(){mem=null;sim=null;try{localStorage.removeItem(KEY);}catch(e){}}
  };
  window.HBTMS=M;
})();
