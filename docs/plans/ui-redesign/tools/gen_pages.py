#!/usr/bin/env python3
"""Generate pages/<slug>.md and route-map.md from: prototype pages (notes), capture manifests and layout JSON.
usage: gen_pages.py <proj_dir> <out_dir(capture)> <dest_dir>"""
import sys,re,json,os,glob,html
proj,out,dest=sys.argv[1:4]
os.makedirs(dest+'/pages',exist_ok=True)
def slug(s): return re.sub(r'[^a-z0-9]+','-',s.lower()).strip('-')[:48] or 'x'
R='ui/src/routes/(app)/kids/[kid]'
PAGES=[ # file, title, legacy routes, legacy files, status/notes
 ('Sign In','Sign in',['/login'],['routes/login/+page.svelte','routes/login/+page.server.ts'],'exists'),
 ('Set Password','Set password (invite and reset)',['/invite/[token]','/reset/[token]'],['routes/invite/[token]/+page.svelte','routes/reset/[token]/+page.svelte','lib/components/SetPasswordForm.svelte'],'exists'),
 ('Error','Error pages (404, 503)',['(any failing route)'],['routes/+error.svelte'],'exists'),
 ('Kid Home','Home',['/kids/[kid]','/ (no children yet: Welcome state)','/kids/[kid]/log/[tracker] (tile forms)'],[R+'/+page.svelte','routes/(app)/+page.svelte','lib/components/QuickLog.svelte','lib/components/RunningTimer.svelte','lib/components/DueList.svelte','lib/components/KidLinks.svelte'],'exists'),
 ('Timeline','Timeline',['/kids/[kid]/timeline','/kids/[kid]/log/[tracker]/[entry] (row edit)'],[R+'/timeline/+page.svelte',R+'/log/[tracker=tracker]/[entry]/+page.svelte'],'exists'),
 ('Growth','Growth',['/kids/[kid]/growth','/kids/[kid]/growth/new','/kids/[kid]/growth/[entry]'],[R+'/growth/+page.svelte',R+'/growth/new/+page.svelte',R+'/growth/[entry]/+page.svelte','lib/components/GrowthChart.svelte','lib/components/GrowthTable.svelte'],'exists'),
 ('Vaccines','Vaccines',['/kids/[kid]/vaccines','/kids/[kid]/vaccines/new','/kids/[kid]/vaccines/[administration]'],[R+'/vaccines/+page.svelte',R+'/vaccines/new/+page.svelte',R+'/vaccines/[administration]/+page.svelte'],'exists'),
 ('Milestones','Milestones',['/kids/[kid]/milestones'],[R+'/milestones/+page.svelte'],'exists'),
 ('Milestone Form','Mark as reached / add a milestone',['/kids/[kid]/milestones/new[?item=]','/kids/[kid]/milestones/[entry]'],[R+'/milestones/new/+page.svelte',R+'/milestones/[entry]/+page.svelte'],'exists'),
 ('Medication','Medication',['/kids/[kid]/medication'],[R+'/medication/+page.svelte'],'exists'),
 ('Course','Medication: course (add and edit)',['/kids/[kid]/medication/new','/kids/[kid]/medication/[medication]'],[R+'/medication/new/+page.svelte',R+'/medication/[medication]/+page.svelte'],'exists'),
 ('Dose','Medication: edit dose',['/kids/[kid]/medication/doses/[dose]'],[R+'/medication/doses/[dose]/+page.svelte'],'exists'),
 ('Symptom','Sickness: log a symptom',['/kids/[kid]/log/symptom','/kids/[kid]/log/symptom/[entry]'],[R+'/log/[tracker=tracker]/+page.svelte',R+'/log/[tracker=tracker]/[entry]/+page.svelte'],'exists (tracker route; confirm the symptom tracker id)'),
 ('Export','Export PDF',['/kids/[kid]/export'],[R+'/export/+page.svelte'],'exists'),
 ('PDF Export','PDF layout',['/kids/[kid]/export/pdf'],[R+'/export/pdf/+server.ts'],'exists (server-generated; design is the print layout)'),
 ('Child Profile','Child profile (details, settings, people with access)',['/kids/[kid]/settings','/kids/[kid]/access','/kids/[kid]/modules'],[R+'/settings/+page.svelte',R+'/access/+page.svelte',R+'/modules/+page.svelte'],'exists. The prototype merges details, settings and access into one profile page with sheets. The sheets\' no-JS pages are designed separately (Child Settings, Access). **Modules has no designed page** (sheet only): confirm its no-JS route with the owner before building'),
 ('Add a Child','Add a child (no-JS page of the Add child sheet)',['/kids/new'],[R+'/../new/+page.svelte'],'exists (restyle). Designed 2026-11-24; same field set and validation as the sheet'),
 ('Child Settings','Child settings (no-JS page of the Edit details sheet)',['/kids/[kid]/settings'],[R+'/settings/+page.svelte'],'exists (restyle). Designed 2026-11-24; caregivers see the not-found card'),
 ('Access','Access (people with access, add a person)',['/kids/[kid]/access'],[R+'/access/+page.svelte'],'exists (restyle). Designed 2026-11-24; the sole guardian cannot be removed or demoted'),
 ('My Profile','My profile',['/account'],['routes/(app)/account/+page.svelte'],'exists'),
 ('Notifications','Notifications',['/account/notifications','/account/notifications/subscribe (POST)'],['routes/(app)/account/notifications/+page.svelte','lib/components/NotificationDevice.svelte'],'exists'),
 ('Choose Child','Choose a child (no-JS fallback of the child switcher)',['/kids (the child switcher\'s no-JS page)'],['lib/components/KidSwitcher.svelte (replaced)'],'exists as the switcher\'s no-JS page (`/kids`); if the legacy repo has no `/kids` list route, add it'),
 ('Admin','Admin',['/admin/parents'],['routes/admin/parents/+page.svelte','routes/admin/+layout.svelte'],'exists'),
]
def notes(f):
    s=open(f'{proj}/{f}.dc.html').read()
    i=s.find('<b>shadcn-svelte parts</b>')
    if i<0: return ''
    j=s.find('</div>',i); blk=s[i:j]
    blk=re.sub(r'<br\s*/?>','\n',blk); blk=re.sub(r'<[^>]+>','',blk)
    return html.unescape(re.sub(r'\n{3,}','\n\n',blk)).strip()
def load(p):
    try: return json.load(open(p))
    except Exception: return None
IMAP={'a':'link','button':'button','input':'input','select':'select','textarea':'textarea','summary':'disclosure','label':'label'}
rows=[]
for f,title,routes,files,status in PAGES:
    sl=slug(f); md=[f'# {title}','',f'Prototype page: `{f}.dc.html`  ·  baselines: `screenshots/{sl}/<variant>/NN-<state>.png`  ·  layout data: `specs/{sl}.<variant>.layout.json`','']
    md+=['## Routes and files (legacy repo)','','| Route | Status |','|---|---|']+[f'| `{r}` | {status} |' for r in routes]+['','Legacy files to restyle: '+', '.join(f'`ui/src/{x}`' if not x.startswith('ui/') else f'`{x}`' for x in files),'']
    n=notes(f)
    if n: md+=['## Implementation notes carried over from the prototype','','These are the design notes that sit in the prototype\'s own side panel (they are part of the design, the panel itself is not).','','```text',n,'```','']
    L={v:load(f'{out}/specs/{sl}.{v}.layout.json') for v in ('phone-light','desktop-light','wide-light')}
    if L['phone-light'] and L['desktop-light']:
        P,D,W=L['phone-light'],L['desktop-light'],L.get('wide-light')
        md+=[f"## Layout (measured from the rendered prototype, Geist loaded)",'',f"Phone frame 390 px wide, content height {P['frame']['h']} px; side-nav frame 980 px wide, content height {D['frame']['h']} px{(', two-column frame 1280 px wide, content height '+str(W['frame']['h'])+' px') if W else ''}. All numbers are CSS px at 1×.",'']
        hs=[e for e in P['els'] if e['t'] in('h1','h2','h3')]
        md+=['**Headings (phone)**','','| Level | Text | Box x,y w×h | font |','|---|---|---|---|']
        for e in hs[:30]:
            md.append(f"| {e['t']} | {e.get('name','')} | {round(e['x'])},{round(e['y'])} {round(e['w'])}×{round(e['h'])} | {e['s'].get('fontSize')}/{e['s'].get('fontWeight')} |")
        def key(e): return (e['t'],e.get('name',''))
        dmap={}
        for e in D['els']:
            if e['t'] in IMAP and e.get('name'): dmap.setdefault(key(e),e)
        wmap={}
        for e in (W['els'] if W else []):
            if e['t'] in IMAP and e.get('name'): wmap.setdefault(key(e),e)
        md+=['','**Interactive elements (default state)**','','| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |','|---|---|---|---|---|---|---|---|']
        seen=set(); n_rows=0
        for e in P['els']:
            if e['t'] in IMAP and e.get('name') and key(e) not in seen:
                seen.add(key(e)); d=dmap.get(key(e)); st=e['s']
                md.append(f"| {e['name'][:48].replace('|','/')} | {IMAP[e['t']]}{' ('+e['role']+')' if e.get('role') else ''} | {e.get('shadcn','')} | {round(e['x'])},{round(e['y'])} {round(e['w'])}×{round(e['h'])} | {(str(round(d['x']))+','+str(round(d['y']))+' '+str(round(d['w']))+'×'+str(round(d['h']))) if d else 'n/a'} | {(lambda w:(str(round(w['x']))+','+str(round(w['y']))+' '+str(round(w['w']))+'×'+str(round(w['h']))) if w else 'n/a')(wmap.get(key(e)))} | {st.get('borderTopLeftRadius') or st.get('borderRadius') or ''} | {st.get('fontSize')}/{st.get('fontWeight')} |")
                n_rows+=1
                if n_rows>=90: md.append('| … | | | | | | | (full list in the layout JSON) |'); break
        md.append('')
    man=load(f'{out}/manifest-{sl}.json') or []
    if man:
        states={}
        for r in man:
            if r.get('error'): continue
            states.setdefault((r['label'],r['kind']),{})[r['variant']]=r
        md+=['## States captured','','`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).','','| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |','|---|---|---|---|---|---|']
        for (lab,kind),vs in states.items():
            pl=vs.get('phone-light'); 
            file=(pl.get('duplicateOf') and 'dup of '+pl['duplicateOf'].split('-',1)[0]) or (pl.get('file','').split('/')[-1] if pl else '')
            ax='/'.join(str(len(vs[v]['axe'])) if v in vs and vs[v].get('axe') is not None else '·' for v in ('phone-light','phone-dark','desktop-light','desktop-dark','wide-light','wide-dark'))
            num=pl['file'].split('/')[-1].split('-')[0] if pl and 'file' in pl else ''
            md.append(f"| {num} | {lab[:60].replace('|','/')} | {kind} | {file} | {ax} | {(pl.get('note') or '')[:40] if pl else ''} |")
        errs=[r for r in man if r.get('error')]
        if errs: md+=['','Capture errors (not product errors): '+'; '.join(f"{r['variant']} {r['label'][:30]}: {r['error'][:60]}" for r in errs[:8])]
        md.append('')
    md+=['## Acceptance','','- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).','- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).','- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).','- The no-JS path in the notes above works with JavaScript disabled.','']
    open(f'{dest}/pages/{sl}.md','w').write('\n'.join(md))
    rows.append((f,title,routes,status,sl))
rm=['# Route map: prototype page → SvelteKit route','','Prototype route names inside the side-panel notes (e.g. `/children/:id/edit`, `/profile/name`) are illustrative; **use the legacy routes below** unless "NEW".','','| Prototype page | Spec | Legacy route(s) | Status |','|---|---|---|---|']
for f,title,routes,status,sl in rows: rm.append(f"| {f} | [pages/{sl}.md](pages/{sl}.md) | {'<br>'.join('`'+r+'`' for r in routes)} | {status} |")
rm+=['','Legacy routes with **no prototype page** (restyle with the same components; not pixel-specified, see assumption A7 in `IMPLEMENTATION.md`): the Modules page (`/kids/[kid]/modules`, sheet only in the prototype), `/healthz*`, `/logout` (POST only).']
open(f'{dest}/route-map.md','w').write('\n'.join(rm)+'\n')
print('pages',len(rows))
