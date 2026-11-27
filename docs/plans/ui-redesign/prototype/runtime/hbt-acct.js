/* hbt-acct.js — sample account/access store for Admin, My profile and Child profile.
   Separate from hbt-data.js (child data) on purpose: an admin works on accounts and never reads child data.
   Prototype stand-in for the API; localStorage key hbt.acct.v1, in memory if storage throws. */
(function(){
  var KEY='hbt.acct.v1',MIN=6e4,HR=60*MIN,DAY=864e5,mem=null,rawSeen=null;
  var EMAIL=/^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/;
  function uid(p){return p+Math.random().toString(36).slice(2,8);}
  function seed(){var n=Date.now();return {rev:1,you:'dan',
    people:[
      {id:'dan',username:'dan',name:'Dan',status:'active',access:{ivy:'Guardian',mia:'Guardian',leo:'Guardian'},last:n-2*MIN,pwAt:n-90*DAY,link:null},
      {id:'ana',username:'ana.k',name:'Ana',status:'active',access:{leo:'Guardian',mia:'Guardian'},last:n-3*HR,pwAt:n-200*DAY,link:null},
      {id:'rae',username:'rae_b',name:'Rae',status:'active',access:{ivy:'Guardian'},last:n-5*DAY,pwAt:n-30*DAY,link:null},
      {id:'cleo',username:'cleo',name:'Cleo',status:'active',access:{leo:'Caregiver'},last:n-26*HR,pwAt:n-12*DAY,link:null},
      {id:'sam',username:'sam-w',name:'Sam',status:'disabled',access:{leo:'Caregiver'},last:n-40*DAY,pwAt:n-300*DAY,link:null},
      {id:'nina',username:'nina',name:'Nina',status:'invited',access:{},last:null,pwAt:null,link:{kind:'invite',at:n-DAY,exp:n+2*DAY}}],
    sessions:[
      {id:'s1',device:'iPhone, Safari',place:'Oakland, CA',at:n-2*MIN,since:n-3*DAY,current:true},
      {id:'s2',device:'MacBook, Chrome',place:'Oakland, CA',at:n-5*HR,since:n-20*DAY,current:false},
      {id:'s3',device:'iPad, Claude app',place:'Hayward, CA',at:n-9*DAY,since:n-60*DAY,current:false}]};}
  function read(){var raw=null;try{raw=localStorage.getItem(KEY);}catch(e){}
    if(raw&&raw===rawSeen&&mem)return mem;var d=null;if(raw){try{d=JSON.parse(raw);}catch(e){d=null;}}
    if(!d)d=mem;if(!d){d=seed();write(d);return d;}mem=d;rawSeen=raw;return d;}
  function write(d){d.rev=(d.rev||0)+1;mem=d;try{var s=JSON.stringify(d);localStorage.setItem(KEY,s);rawSeen=s;}catch(e){rawSeen=null;}}
  function P(id){return read().people.filter(function(p){return p.id===id;})[0]||null;}
  function activeGuardians(kid,except){return read().people.filter(function(p){return p.status==='active'&&p.access[kid]==='Guardian'&&p.id!==except;});}
  function when(t,now){var H=window.HBT;return H.heading(t,now)+' '+H.hm(t);}
  var RESET_H=1,INVITE_H=72;
  var ERR={
    sole:'This parent is the sole guardian of at least one child.',
    last:'A child needs at least one active guardian.',
    gone:'This was already changed by someone else. The list is updated.',
    state:'This parent can’t get that kind of link right now.',
    self:'You can’t do this to your own access here.'};
  function token(){var c='abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_',o='';for(var i=0;i<43;i++)o+=c.charAt(Math.floor(Math.random()*c.length));return o;}
  var A={
    ERR:ERR,when:when,RESET_H:RESET_H,INVITE_H:INVITE_H,
    get:function(){return read();},
    reset:function(){write(seed());},
    snapshot:function(){var d=read();return JSON.stringify({p:d.people,s:d.sessions});},
    restore:function(x){var o=JSON.parse(x),d=read();d.people=o.p;d.sessions=o.s;write(d);},
    you:function(){return P(read().you);},
    person:P,
    people:function(){return read().people.slice();},
    /* ----- admin: a separate principal. Parents list only: no child names, no counts. ----- */
    setStatus:function(id,st){var p=P(id);if(!p)return {err:ERR.gone};
      if(st==='disabled'&&Object.keys(p.access).some(function(k){return p.access[k]==='Guardian'&&activeGuardians(k,id).length===0;}))return {err:ERR.sole};
      var d=read();P(id).status=st==='active'&&!P(id).pwAt?'invited':st;write(d);if(st==='disabled'&&window.HBTM)window.HBTM.cleanupPerson(id);return {ok:true};},
    issueLink:function(id,kind){var p=P(id),now=Date.now();if(!p)return {err:ERR.gone};
      if(kind==='reset'&&p.status!=='active')return {err:ERR.state};if(kind==='invite'&&p.status!=='invited')return {err:ERR.state};
      var h=kind==='reset'?RESET_H:INVITE_H,d=read();P(id).link={kind:kind,at:now,exp:now+h*HR};write(d);
      return {ok:true,kind:kind,link:'https://happybaby.example/'+(kind==='reset'?'reset/':'invite/')+token(),exp:now+h*HR,hours:h};},
    createParent:function(username,name){var e={},u=String(username||'').trim(),n=String(name||'').trim(),d=read();
      if(!/^[a-z0-9._-]{3,32}$/.test(u))e.username='Use 3 to 32 characters: a-z, 0-9, dot, underscore, hyphen.';else if(d.people.some(function(p){return p.username===u;}))e.username='This username is taken.';
      if(!n||n.length>60)e.name='Enter a name of up to 60 characters.';
      if(Object.keys(e).length)return {err:e};
      var id=uid('p');d.people.push({id:id,username:u,name:n,status:'invited',access:{},last:null,pwAt:null,link:null});write(d);var r=A.issueLink(id,'invite');r.id=id;return r;},
    /* ----- my profile: display name only; the username is permanent ----- */
    updateProfile:function(f){var e={},nm=String(f.name||'').trim();
      if(!nm||nm.length>60)e.name='Enter a name of up to 60 characters.';
      if(Object.keys(e).length)return {err:e};
      var d=read();A.you().name=nm;write(d);return {ok:true};},
    changePassword:function(f){var e={},cur=String(f.cur||''),nw=String(f.nw||''),cf=String(f.cf||'');
      if(!cur)e.cur='Enter your current password.';else if(cur==='wrong')e.cur='That isn’t your current password.';
      if(nw.length<12)e.nw='Use at least 12 characters.';else if(nw.length>128)e.nw='Use up to 128 characters.';else if(nw===cur)e.nw='Choose a password you haven’t just used.';
      if(!e.nw&&cf!==nw)e.cf='The two passwords don’t match.';
      if(Object.keys(e).length)return {err:e};
      var d=read(),n=0;if(f.out){d.sessions=d.sessions.filter(function(s){if(s.current)return true;n++;return false;});}
      A.you().pwAt=Date.now();write(d);return {ok:true,closed:n};},
    sessions:function(){return read().sessions.slice().sort(function(a,b){return (b.current?1:0)-(a.current?1:0)||b.at-a.at;});},
    closeSession:function(id){var d=read(),s=d.sessions.filter(function(x){return x.id===id;})[0];if(!s)return {err:'That session was already closed. The list is updated.'};
      if(s.current)return {err:'This is the device you’re using now.'};d.sessions=d.sessions.filter(function(x){return x.id!==id;});write(d);return {ok:true};},
    closeOthers:function(){var d=read(),n=d.sessions.filter(function(x){return !x.current;}).length;d.sessions=d.sessions.filter(function(x){return x.current;});write(d);return {ok:true,n:n};},
    /* ----- child profile: who can see this child ----- */
    accessFor:function(kid){return read().people.filter(function(p){return p.access[kid];}).map(function(p){return {id:p.id,name:p.name,username:p.username,role:p.access[kid],status:p.status,you:p.id===read().you,sole:p.access[kid]==='Guardian'&&p.status==='active'&&activeGuardians(kid,p.id).length===0};})
      .sort(function(a,b){return (a.role==='Guardian'?0:1)-(b.role==='Guardian'?0:1)||a.name.localeCompare(b.name);});},
    myRole:function(kid){var y=A.you();return y&&y.access[kid]||null;},
    setRole:function(pid,kid,role){var p=P(pid);if(!p||!p.access[kid])return {err:ERR.gone};
      if(role==='Caregiver'&&p.access[kid]==='Guardian'&&p.status==='active'&&activeGuardians(kid,pid).length===0)return {err:ERR.last};
      var d=read();P(pid).access[kid]=role;write(d);return {ok:true};},
    revoke:function(pid,kid){var p=P(pid);if(!p||!p.access[kid])return {err:ERR.gone};
      if(p.access[kid]==='Guardian'&&p.status==='active'&&activeGuardians(kid,pid).length===0)return {err:ERR.last};
      var d=read();delete P(pid).access[kid];write(d);if(window.HBTM)window.HBTM.cleanup(kid,pid);return {ok:true};},
    /* prototype assumption: a guardian adds an existing parent by username (the app has no email) */
    grant:function(kid,username,role){var u=String(username||'').trim().toLowerCase(),e={};
      if(!u)return {err:{username:'Enter the parent’s username.'}};
      var p=read().people.filter(function(x){return x.username===u;})[0];
      if(!p)return {err:{username:'No parent has this username. Ask the admin to create the account.'}};
      if(p.access[kid])return {err:{username:'This person already has access.'}};
      var d=read();P(p.id).access[kid]=role==='Guardian'?'Guardian':'Caregiver';write(d);return {ok:true,name:p.name};},
    dropKid:function(kid){var d=read();d.people.forEach(function(p){delete p.access[kid];});write(d);if(window.HBTM)window.HBTM.dropKid(kid);}
  };
  window.HBTA=A;
})();
