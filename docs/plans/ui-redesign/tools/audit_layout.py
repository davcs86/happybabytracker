#!/usr/bin/env python3
"""Audit the measured layout JSON: interactive targets under 44 px and horizontal overflow.
usage: audit_layout.py [specs_dir]   (default: ../specs)
Native checkbox and radio inputs are 20 px by design: they sit inside label rows of 44 px or more, which is the target.
Reports every other interactive element under 44 px in either dimension. Exit 1 when something is reported."""
import sys,os,json,glob
d=sys.argv[1] if len(sys.argv)>1 else os.path.join(os.path.dirname(os.path.abspath(__file__)),'..','specs')
bad=[];over=[]
for f in sorted(glob.glob(os.path.join(d,'*.layout.json'))):
    j=json.load(open(f));W=j['frame']['w'];els=j['els']
    for i,e in enumerate(els):
        t=e['t']
        inter=t in('button','a','input','select','textarea') or e.get('role') in('button','link','tab','checkbox','radio','switch')
        if inter and e['w']>0 and (e['h']<44 or e['w']<44):
            if t=='input' and e.get('s',{}).get('display')=='none': continue
            parent=[x for x in els[max(0,i-3):i] if x['t']=='label' and x['h']>=44 and x['x']<=e['x'] and x['x']+x['w']>=e['x']+e['w']-1]
            if parent: continue
            bad.append((os.path.basename(f),t,e.get('name','')[:40],round(e['w']),round(e['h'])))
        if e['x']+e['w']>W+1 or e['x']<-1: over.append((os.path.basename(f),t,e.get('name','')[:40]))
print('targets under 44 px:',len(bad));[print(' ',b) for b in bad[:50]]
print('horizontal overflow:',len(over));[print(' ',o) for o in over[:50]]
sys.exit(1 if bad or over else 0)
