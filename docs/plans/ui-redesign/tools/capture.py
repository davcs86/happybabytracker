#!/usr/bin/env python3
"""State-matrix capture for the Happy Baby Tracker prototype (reproducible).
usage: capture.py <proj_dir> <out_dir> <fonts_dir> <axe.min.js> <Page Name> [<Page Name> ...]
 proj_dir  : folder holding the *.dc.html pages (react scripts must be injected, see README)
 out_dir   : screenshots/<page-slug>/<variant>/NN-<state>.png + manifest.json + layout JSON
Variants: phone-light, phone-dark, desktop-light, desktop-dark, wide-light, wide-dark  (390x800, 980x780 and 1280x780 app frames; frame radius and
shadow removed so the image is the product surface only).
Each state is captured in a fresh browser context. Duplicate images (state with no visible change) are not saved;
the manifest lists them as behaviour-only. axe (wcag2a/aa/21aa + best-practice) runs on every saved state."""
import sys,os,re,json,hashlib,glob,time
from playwright.sync_api import sync_playwright
proj,out,fonts,axe_path=sys.argv[1:5]; pages=sys.argv[5:]
AX=open(axe_path).read()
def fcss():
    def ff(fam,fn,rng='U+0000-00FF,U+0131,U+0152-0153,U+02BB-02BC,U+02C6,U+02DA,U+02DC,U+0304,U+0308,U+0329,U+2000-206F,U+20AC,U+2122,U+2191,U+2193,U+2212,U+2215,U+FEFF,U+FFFD'):
        return "@font-face{font-family:'%s';src:url('file://%s/%s') format('woff2');font-weight:100 900;font-style:normal;unicode-range:%s}"%(fam,fonts,fn,rng)
    return ff('Geist','geist-latin-wght-normal.woff2')+ff('Geist','geist-latin-ext-wght-normal.woff2','U+0100-02BA,U+02BD-02C5,U+02C7-02CC,U+02CE-02D7,U+02DD-02FF,U+0304,U+0308,U+0329,U+1D00-1DBF,U+1E00-1E9F,U+1EF2-1EFF,U+2020,U+20A0-20AB,U+20AD-20C0,U+2113,U+2C60-2C7F,U+A720-A7FF')+ff('Geist Mono','geist-mono-latin-wght-normal.woff2')+ff('Geist Mono','geist-mono-latin-ext-wght-normal.woff2','U+0100-02BA,U+02BD-02C5,U+02C7-02CC,U+02CE-02D7,U+02DD-02FF,U+0304,U+0308,U+0329,U+1D00-1DBF,U+1E00-1E9F,U+1EF2-1EFF,U+2020,U+20A0-20AB,U+20AD-20C0,U+2113,U+2C60-2C7F,U+A720-A7FF')
FONT_CSS=fcss()
FRAME_CSS="[data-hbt=app]{border-radius:0!important;box-shadow:none!important} *{animation:none!important;transition:none!important;caret-color:transparent!important}"
VARIANTS=[('phone',False,False),('phone',True,False),('desktop',False,True),('desktop',True,True),('wide',False,'wide'),('wide',True,'wide')]
def slug(s): return re.sub(r'[^a-z0-9]+','-',s.lower()).strip('-')[:48] or 'x'
def click_panel(pg,t):
    l=pg.locator('[data-hbt=panel] button')
    for i in range(l.count()):
        if l.nth(i).inner_text().strip()==t: l.nth(i).click(); return True
    return False
def open_page(b,page,dark,desk):
    pg=b.new_page(viewport={'width':1700 if desk=='wide' else (1300 if desk else 1100),'height':1000})
    pg.route('**/fonts.googleapis.com/**',lambda r:r.fulfill(body=FONT_CSS,content_type='text/css'))
    pg.route('**/fonts.gstatic.com/**',lambda r:r.abort())
    pg.goto('file://%s/%s'%(proj,page.replace(' ','%20')+'.dc.html')); pg.wait_for_timeout(650)
    pg.add_style_tag(content=FRAME_CSS)
    if dark: click_panel(pg,'Dark')
    if desk=='wide': click_panel(pg,'≥1280 two-column')
    elif desk: click_panel(pg,'≥768 side-nav')
    pg.wait_for_timeout(250); pg.evaluate('document.fonts.ready'); return pg
GROUPS_JS="""()=>{const p=document.querySelector('[data-hbt=panel]');if(!p)return [];const out={};
 p.querySelectorAll('button').forEach(b=>{const g=b.parentElement&&b.parentElement.parentElement;const t=g&&g.firstElementChild?g.firstElementChild.innerText.trim():'';
  (out[t]=out[t]||[]).push({text:b.innerText.trim(),pressed:b.getAttribute('aria-pressed')==='true'||b.getAttribute('aria-current')==='true'});});
 return Object.entries(out).map(([t,o])=>({title:t,opts:o}))}"""
FULL_JS="""()=>{const a=document.querySelector('[data-hbt=app]');if(!a)return null;if(a.querySelector('[role=dialog]'))return null;
 let best=null,ex=0;a.querySelectorAll('*').forEach(e=>{const cs=getComputedStyle(e);if(!/(auto|scroll)/.test(cs.overflowY))return;const d=e.scrollHeight-e.clientHeight;if(d>ex){ex=d;best=e}});
 if(!best||ex<3)return {extra:0};a.style.height=(a.clientHeight+ex)+'px';return {extra:ex,orig:a.clientHeight}}"""
def shot(pg,path,full):
    app=pg.locator('[data-hbt=app]').first
    info=pg.evaluate(FULL_JS) if full else None
    pg.wait_for_timeout(120)
    png=app.screenshot(path=None)
    return png,(info or {}).get('extra',0)
def axe(pg):
    pg.add_script_tag(content=AX)
    return pg.evaluate("axe.run('[data-hbt=app]',{runOnly:['wcag2a','wcag2aa','wcag21aa','best-practice']}).then(r=>r.violations.map(v=>v.id+':'+v.nodes.length+':'+v.nodes[0].target.join(' ').slice(0,60)))")
LAYOUT_JS="""()=>{const a=document.querySelector('[data-hbt=app]');if(!a)return null;const ar=a.getBoundingClientRect();
 const R=n=>Math.round(n*2)/2;const out=[];const props=['fontSize','fontWeight','lineHeight','letterSpacing','color','backgroundColor','borderTopWidth','borderTopColor','borderTopLeftRadius','paddingTop','paddingRight','paddingBottom','paddingLeft','gap','display','flexDirection','justifyContent','alignItems','textAlign','opacity','boxShadow'];
 const vars=['--bg','--card','--fg','--muted','--border','--primary','--primary-fg','--soft','--ink','--destructive','--toast-link'];
 const csA=getComputedStyle(a);const cv={};vars.forEach(v=>cv[v]=csA.getPropertyValue(v).trim());
 a.querySelectorAll('*').forEach(e=>{const r=e.getBoundingClientRect();if(r.width<1||r.height<1)return;const cs=getComputedStyle(e);if(cs.visibility==='hidden'||cs.display==='none')return;
  const tag=e.tagName.toLowerCase();const interesting=e.hasAttribute('data-shadcn')||e.hasAttribute('role')||e.hasAttribute('data-hbt')||/^(h1|h2|h3|button|a|input|select|textarea|label|svg|th|td|li)$/.test(tag)||(e.children.length===0&&e.textContent.trim());
  if(!interesting)return;const o={t:tag,x:R(r.left-ar.left),y:R(r.top-ar.top),w:R(r.width),h:R(r.height)};
  if(e.getAttribute('data-shadcn'))o.shadcn=e.getAttribute('data-shadcn');if(e.getAttribute('data-hbt'))o.hbt=e.getAttribute('data-hbt');if(e.getAttribute('role'))o.role=e.getAttribute('role');
  const nm=e.getAttribute('aria-label')||(e.children.length===0?e.textContent.trim().slice(0,60):'')||(tag==='input'?(e.getAttribute('name')||e.id):'');if(nm)o.name=nm;
  if(e.getAttribute('aria-current'))o.current=e.getAttribute('aria-current');if(e.getAttribute('aria-pressed'))o.pressed=e.getAttribute('aria-pressed');if(e.getAttribute('href'))o.href=e.getAttribute('href');
  const s={};props.forEach(p=>{const v=cs[p];if(v&&!['normal','none','0px','rgba(0, 0, 0, 0)','0','auto','start'].includes(v))s[p]=v});o.s=s;out.push(o)});
 return {frame:{w:R(ar.width),h:R(ar.height)},vars:cv,n:out.length,els:out}}"""
SCRIPTS={
 'Sign In':[('blank submit',[('click','button[type=submit]')]),('signed in',[('fill','#si-user','ana'),('fill','#si-pass','a-long-password'),('click','button[type=submit]')])],
 'Set Password':[('short password',[('fill','#sp-new','short'),('fill','#sp-conf','short'),('click','button[type=submit]')]),('mismatch',[('fill','#sp-new','abcdefghijkl'),('fill','#sp-conf','abcdefghijkx'),('click','button[type=submit]')]),('password set',[('fill','#sp-new','abcdefghijkl'),('fill','#sp-conf','abcdefghijkl'),('click','button[type=submit]')])],
}
PREFILL={'Sign In':[('fill','#si-user','ana'),('fill','#si-pass','a-long-password')],'Set Password':[('fill','#sp-new','abcdefghijkl'),('fill','#sp-conf','abcdefghijkl')]}
def do_steps(pg,steps):
    for st in steps:
        if st[0]=='fill': pg.fill(st[1],st[2])
        elif st[0]=='click': pg.locator('[data-hbt=app] '+st[1]).first.click()
    pg.wait_for_timeout(350)
def run_page(b,page):
    ps=slug(page); man=[]; used=set()
    pdf=(page=='PDF Export')
    for vname,dark,desk in VARIANTS:
        if os.environ.get('HBT_ONLY') and vname!=os.environ['HBT_ONLY']: continue
        var=f"{vname}-{'dark' if dark else 'light'}"
        d=os.path.join(out,'screenshots',ps,var); os.makedirs(d,exist_ok=True)
        idx=[0]; seen={}
        def save(png,label,kind,extra=None,pg=None,note=''):
            h=hashlib.sha1(png).hexdigest()
            rec={'variant':var,'label':label,'kind':kind,'note':note}
            if extra is not None: rec['scrollExtraPx']=extra
            if h in seen: rec['duplicateOf']=seen[h]; man.append(rec); return
            idx[0]+=1; fn=f"{idx[0]:02d}-{slug(label)}.png"; open(os.path.join(d,fn),'wb').write(png)
            seen[h]=fn; rec['file']=f"screenshots/{ps}/{var}/{fn}"
            try: rec['axe']=axe(pg) if pg is not None else None
            except Exception as e: rec['axe']=['axe-error:'+str(e)[:60]]
            man.append(rec)
        try:
            pg=open_page(b,page,dark,desk)
            if pdf:
                els=pg.locator('[role=img]'); 
                for i in range(els.count()):
                    save(els.nth(i).screenshot(),f'sheet-{i+1}','paper',pg=None)
                pg.close(); continue
            # default: frame + full
            png,_=shot(pg,None,False); save(png,'default-frame','frame',pg=pg)
            pg.close(); pg=open_page(b,page,dark,desk)
            png,ex=shot(pg,None,True); 
            if ex: save(png,'default-full-height','full',extra=ex,pg=pg)
            if var in ('phone-light','desktop-light','wide-light'):
                try:
                    lay=pg.evaluate(LAYOUT_JS)
                    pg.close(); pg=open_page(b,page,dark,desk)  # restore
                    os.makedirs(os.path.join(out,'specs'),exist_ok=True)
                    json.dump(lay,open(os.path.join(out,'specs',f'{ps}.{var}.layout.json'),'w'),separators=(',',':'))
                except Exception as e: man.append({'variant':var,'label':'layout-json','error':str(e)[:100]})
            groups=pg.evaluate(GROUPS_JS); pg.close()
            for g in groups:
                if g['title'].strip().lower() in ('theme','layout',''): continue
                gt=g['title'] if (len(g['title'])<=28 and '\n' not in g['title']) else ''  # standalone panel buttons have no group title
                for o in g['opts']:
                    if o['pressed'] or o['text'] in ('Reset sample data',): continue
                    pg=open_page(b,page,dark,desk)
                    try:
                        click_panel(pg,o['text']); pg.wait_for_timeout(350)
                        png,ex=shot(pg,None,True); save(png,f"{gt} {o['text']}".strip(),'state',extra=ex or None,pg=pg,note=gt)
                        # failure injectors: try one action, expect a toast/message
                        if re.search(r'save|answer|simulated|one-tap|next|after|link',g['title'],re.I) or o['text'] in ('Offline','Error','Course gone','Dose cap','Cap reached','Already marked'):
                            if not attempt_action(pg,page):
                                pass
                            else:
                                png,ex=shot(pg,None,False); save(png,f"{gt} {o['text']} after action".strip(),'state-after-action',pg=pg,note=gt)
                    except Exception as e: man.append({'variant':var,'label':f"{gt} {o['text']}".strip(),'error':str(e)[:120]})
                    pg.close()
            for lab,steps in SCRIPTS.get(page,[]):
                pg=open_page(b,page,dark,desk)
                try:
                    do_steps(pg,steps); png,ex=shot(pg,None,True); save(png,lab,'flow',extra=ex or None,pg=pg)
                except Exception as e: man.append({'variant':var,'label':lab,'error':str(e)[:120]})
                pg.close()
            # drawers and sheets
            pg=open_page(b,page,dark,desk)
            keys=pg.evaluate("()=>[...new Set([...document.querySelectorAll('[data-hbt=app] [data-drawer]')].map(e=>e.getAttribute('data-drawer')))]")
            has_more=pg.locator('[data-hbt=tabs] >> text=More').count()>0 and pg.locator('[data-hbt=tabs]').first.is_visible()
            pg.close()
            targets=[('drawer',k) for k in keys]+([('more',None)] if has_more else [])
            for kind,k in targets:
                pg=open_page(b,page,dark,desk)
                try:
                    if kind=='drawer': pg.locator(f'[data-hbt=app] [data-drawer="{k}"]').first.click()
                    else: pg.locator('[data-hbt=tabs]').get_by_text('More',exact=True).first.click()
                    pg.wait_for_timeout(450)
                    if pg.locator('[data-hbt=app] [role=dialog]').count()==0: pg.close(); continue
                    label=k or 'More'
                    png,_=shot(pg,None,False); save(png,f"sheet {label}",'sheet',pg=pg)
                    # submit empty
                    if attempt_submit(pg,True):
                        pg.wait_for_timeout(350); png,_=shot(pg,None,False); save(png,f"sheet {label} submit empty",'sheet-validation',pg=pg)
                except Exception as e: man.append({'variant':var,'label':f"sheet {k}",'error':str(e)[:120]})
                pg.close()
        except Exception as e:
            man.append({'variant':var,'label':'page','error':str(e)[:160]})
    json.dump(man,open(os.path.join(out,'manifest-'+ps+'.json'),'w'),indent=1)
    return man
def attempt_action(pg,page=''):
    """open the first drawer (if any) else press the first primary-looking button; used after a failure injector"""
    try:
        if page in PREFILL:
            do_steps(pg,PREFILL[page]+[('click','button[type=submit]')]); return True
        dr=pg.locator('[data-hbt=app] [data-drawer]:not([data-drawer="Switch child"]):not([data-drawer="More"])')
        if dr.count():
            dr.first.click(); pg.wait_for_timeout(400)
            return attempt_submit(pg,False)
        for sel in ['button:has-text("Save")','button:has-text("Log")','button:has-text("Mark")','button:has-text("Add")','button:has-text("Download")','button:has-text("Try again")','button:has-text("Undo")']:
            l=pg.locator('[data-hbt=app] '+sel)
            for i in range(l.count()):
                if l.nth(i).is_visible() and l.nth(i).get_attribute('aria-disabled')!='true': l.nth(i).click(); pg.wait_for_timeout(350); return True
    except Exception: return False
    return False
def attempt_submit(pg,empty):
    d=pg.locator('[data-hbt=app] [role=dialog]')
    if d.count()==0: return False
    btns=d.first.locator('button')
    best=None
    for i in range(btns.count()):
        b=btns.nth(i); 
        try: t=(b.inner_text() or '').strip(); lab=b.get_attribute('aria-label') or ''
        except Exception: continue
        if lab=='Close' or t in ('','Close','Cancel','Keep access','Not now'): continue
        if b.is_visible(): best=b
    if best is None: return False
    best.click(); pg.wait_for_timeout(300); return True
if __name__=='__main__':
    os.makedirs(out,exist_ok=True)
    with sync_playwright() as p:
        b=p.chromium.launch(executable_path=os.environ.get('CHROME','/usr/bin/google-chrome'),args=['--no-sandbox','--allow-file-access-from-files'])
        for pgname in pages:
            t=time.time(); m=run_page(b,pgname); print(pgname,len(m),'records',round(time.time()-t),'s',flush=True)
        b.close()
