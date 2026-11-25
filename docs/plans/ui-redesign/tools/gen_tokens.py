import math,json,re,os
OUT='/home/claude/export/design-system/'
def oklch_to_rgb(L,C,h):
    a=C*math.cos(math.radians(h)); b=C*math.sin(math.radians(h))
    l_=L+0.3963377774*a+0.2158037573*b; m_=L-0.1055613458*a-0.0638541728*b; s_=L-0.0894841775*a-1.2914855480*b
    l,m,s=l_**3,m_**3,s_**3
    r=4.0767416621*l-3.3077115913*m+0.2309699292*s
    g=-1.2684380046*l+2.6097574011*m-0.3413193965*s
    bl=-0.0041960863*l-0.7034186147*m+1.7076147010*s
    def f(x):
        x=max(0,min(1,x)); return 12.92*x if x<=0.0031308 else 1.055*x**(1/2.4)-0.055
    return tuple(round(255*f(v)) for v in (r,g,bl))
def parse(s):
    m=re.match(r'oklch\(([\d.]+) ([\d.]+) ([\d.]+)\)',s); return tuple(map(float,m.groups()))
def hexof(s): r=oklch_to_rgb(*parse(s)); return '#%02x%02x%02x'%r
def lum(s):
    r,g,b=oklch_to_rgb(*parse(s))
    f=lambda c:(c/255/12.92) if c/255<=0.03928 else ((c/255+0.055)/1.055)**2.4
    return 0.2126*f(r)+0.7152*f(g)+0.0722*f(b)
def cr(a,b):
    x,y=lum(a),lum(b); x,y=max(x,y),min(x,y); return round((x+0.05)/(y+0.05),2)
LIGHT=dict(bg='oklch(0.982 0.008 70)',card='oklch(0.998 0.003 70)',fg='oklch(0.24 0.02 50)',muted='oklch(0.48 0.02 50)',border='oklch(0.9 0.012 60)',primary='oklch(0.53 0.17 38)',primaryFg='oklch(0.99 0 0)',toastLink='oklch(0.8 0.12 55)',soft='oklch(0.95 0.04 55)',soft2='oklch(0.95 0.01 60)',ink='oklch(0.42 0.12 38)',destructive='oklch(0.46 0.17 27)')
DARK=dict(bg='oklch(0.19 0.015 50)',card='oklch(0.245 0.015 50)',fg='oklch(0.95 0.01 70)',muted='oklch(0.74 0.02 60)',border='oklch(0.36 0.015 50)',primary='oklch(0.78 0.13 55)',primaryFg='oklch(0.2 0.02 50)',toastLink='oklch(0.45 0.15 38)',soft='oklch(0.3 0.05 55)',soft2='oklch(0.27 0.015 50)',ink='oklch(0.84 0.11 55)',destructive='oklch(0.76 0.14 25)')
HUE=dict(bottle=65,nursing=20,diaper=95,sleep=275,pumping=330,solids=140,symptom=60)
# contrast table
pairs=[('fg','soft2'),('muted','soft2'),('fg','bg'),('fg','card'),('muted','bg'),('muted','card'),('primaryFg','primary'),('ink','bg'),('ink','soft'),('primary','bg'),('primary','card'),('destructive','bg'),('destructive','card'),('toastLink','fg')]
def table(T,name):
    rows=[]
    for a,b in pairs:
        # toast sits on inverse surface (fg) in both themes
        rows.append({'fg':a,'on':b,'ratio':cr(T[a],T[b]),'passAA':cr(T[a],T[b])>=4.5})
    return rows
ct={'light':table(LIGHT,'light'),'dark':table(DARK,'dark')}
# tracker chip colours
def chip(h,dark):
    return dict(bg=f'oklch({0.3 if dark else 0.95} {0.05 if dark else 0.04} {h})',fg=f'oklch({0.84 if dark else 0.42} {0.11 if dark else 0.12} {h})',bar=f'oklch({0.7 if dark else 0.6} {0.12 if dark else 0.14} {h})')
trk={k:{'hue':h,'light':chip(h,False),'dark':chip(h,True)} for k,h in HUE.items()}
for k,v in trk.items():
    for th in('light','dark'): v[th]['chipContrast']=cr(v[th]['fg'],v[th]['bg'])
tok={
 '$description':'Happy Baby Tracker brand preset tokens. Source of truth: the Claude Design prototype (values copied from the pages, not re-derived).',
 'color':{'light':{k:{'oklch':v,'hex':hexof(v)} for k,v in LIGHT.items()},'dark':{k:{'oklch':v,'hex':hexof(v)} for k,v in DARK.items()}},
 'contrast':ct,'tracker':trk,
 'font':{'sans':"'Geist', system-ui, sans-serif",'mono':"'Geist Mono', monospace",'weights':[400,500,600,700],'package':'@fontsource-variable/geist, @fontsource-variable/geist-mono'},
 'type':{'scalePx':{'micro':11,'caption-sm':11.5,'caption':12,'caption-md':12.5,'small':13,'small-md':13.5,'body-sm':14,'body-md':14.5,'body':15,'input':16,'lead':17,'title':18,'display-sm':22,'display':30},
   'lineHeight':{'tight':1.1,'snug':1.4,'normal':1.5,'relaxed':1.6},'letterSpacing':{'display':'-0.02em','label':'0.06em'},
   'roles':{'h1':{'size':30,'weight':600,'lh':1.1,'ls':'-0.02em'},'sheetTitle':{'size':18,'weight':600},'sectionTitle':{'size':18,'weight':600},'body':{'size':15,'lh':1.5},'control':{'size':15,'weight':600},'input':{'size':16,'weight':400},'meta':{'size':12,'weight':400,'color':'muted'},'badge':{'size':12,'weight':600}}},
 'radius':{'xs':2,'sm':8,'md':10,'lg':12,'xl':14,'2xl':16,'3xl':20,'full':99,'usage':{'button, alert, tracker chip, radio row, tab track':12,'input, select, textarea, ToggleGroup chip':8,'unit segment, side-nav item, compact control':10,'measure tab (inside a 12 px track)':9,'checkbox':4,'card, collapsible':14,'section card, empty-state card':16,'large button (sheet Done/Cancel)':14,'sheet top corners':20,'badge, pill':99,'drag handle':2}},
 'spacingPx':[2,4,6,8,10,12,14,16,18,22,32],
 'target':{'min':44,'control':48,'primaryAction':52,'sheetDone':56,'segmentMin':34,'note':'34 px segments exist only in the prototype sim panel, never in the product'},
 'shadow':{'sheet':'0 -8px 32px rgba(0,0,0,.2)','menu':'0 8px 24px rgba(0,0,0,.25)','overlay':'rgba(20,12,8,.5)'},
 'layout':{'phoneWidth':390,'phoneHeight':800,'desktopWidth':980,'desktopHeight':780,'sideNavBreakpoint':768,'wideBreakpoint':1280,'wideWidth':1280,'wideHeight':780,'contentBlock':920,'column':448,'columnGap':24,'rowGap':18,'sheetMaxWidth':480,'authCardMaxWidth':400,'sheetHandle':{'w':36,'h':4}},
 'focus':{'outline':'2px solid var(--primary)','offset':2},
}
json.dump(tok,open(OUT+'tokens/tokens.json','w'),indent=1)
# ---- tokens.css (Tailwind 4 + shadcn variable names)
def vars_(T):
    return f'''  --background: {T['bg']};
  --foreground: {T['fg']};
  --card: {T['card']};
  --card-foreground: {T['fg']};
  --popover: {T['card']};
  --popover-foreground: {T['fg']};
  --primary: {T['primary']};
  --primary-foreground: {T['primaryFg']};
  --secondary: {T['card']};
  --secondary-foreground: {T['fg']};
  --muted: {T['soft2']};
  --muted-foreground: {T['muted']};
  --accent: {T['soft']};
  --accent-foreground: {T['ink']};
  --destructive: {T['destructive']};
  --border: {T['border']};
  --input: {T['border']};
  --ring: {T['primary']};
  --primary-ink: {T['ink']};
  --toast-link: {T['toastLink']};'''
css=f'''/* Happy Baby Tracker brand preset: Tailwind 4 + shadcn (svelte and react) variable names.
   Generated from the Claude Design prototype by gen_tokens.py. Do not edit values by hand:
   change the prototype/preset, regenerate.
   Mapping from the prototype's own variable names (they differ from shadcn's):
     --bg -> --background     --fg -> --foreground     --card -> --card
     --muted (a TEXT colour in the prototype) -> --muted-foreground
     --soft2 (tab track) -> --muted (shadcn: a quiet surface)
     --soft -> --accent       --ink -> --primary-ink (and --accent-foreground)
     --primary-fg -> --primary-foreground     --toast-link, --destructive, --border, --primary unchanged */
:root {{
  color-scheme: light;
{vars_(LIGHT)}
  --overlay: rgba(20, 12, 8, 0.5);
  --radius: 12px;
  /* tracker hues: one hue per tracker, lightness and chroma fixed per theme */
{chr(10).join(f"  --hue-{k}: {h};" for k,h in HUE.items())}
  --chip-bg-l: 0.95; --chip-bg-c: 0.04; --chip-fg-l: 0.42; --chip-fg-c: 0.12; --chip-bar-l: 0.6; --chip-bar-c: 0.14;
}}
/* Dark palette. Three ways to arrive here, one set of values:
   1. data-theme="dark"      the user chose Dark (server renders it from the theme cookie: works without JS)
   2. data-resolved="dark"   a tiny head script resolved Auto (or the choice) to dark before first paint
   3. no data-theme/resolved the user has not chosen and JS is off: follow the device (prefers-color-scheme) */
:root[data-theme='dark'],
:root[data-resolved='dark'] {{
{vars_(DARK)}
  --chip-bg-l: 0.3; --chip-bg-c: 0.05; --chip-fg-l: 0.84; --chip-fg-c: 0.11; --chip-bar-l: 0.7; --chip-bar-c: 0.12;
  color-scheme: dark;
}}
@media (prefers-color-scheme: dark) {{
  :root:not([data-theme='light']):not([data-theme='dark']):not([data-resolved]) {{
{vars_(DARK).replace(chr(10),chr(10)+"  ")}
    --chip-bg-l: 0.3; --chip-bg-c: 0.05; --chip-fg-l: 0.84; --chip-fg-c: 0.11; --chip-bar-l: 0.7; --chip-bar-c: 0.12;
    color-scheme: dark;
  }}
}}
/* `dark:` utilities (shadcn parts ship some): follow the resolved theme. Prefer tokens; avoid new `dark:` classes. */
@custom-variant dark (&:where([data-theme='dark'], [data-theme='dark'] *, [data-resolved='dark'], [data-resolved='dark'] *));

@theme {{
  --font-sans: 'Geist Variable', 'Geist', system-ui, sans-serif;
  --font-mono: 'Geist Mono Variable', 'Geist Mono', ui-monospace, monospace;
  --shadow-sheet: 0 -8px 32px rgba(0, 0, 0, 0.2);
  --shadow-menu: 0 8px 24px rgba(0, 0, 0, 0.25);
}}

@theme inline {{
  --color-background: var(--background);
  --color-foreground: var(--foreground);
  --color-card: var(--card);
  --color-card-foreground: var(--card-foreground);
  --color-popover: var(--popover);
  --color-popover-foreground: var(--popover-foreground);
  --color-primary: var(--primary);
  --color-primary-foreground: var(--primary-foreground);
  --color-secondary: var(--secondary);
  --color-secondary-foreground: var(--secondary-foreground);
  --color-muted: var(--muted);
  --color-muted-foreground: var(--muted-foreground);
  --color-accent: var(--accent);
  --color-accent-foreground: var(--accent-foreground);
  --color-destructive: var(--destructive);
  --color-border: var(--border);
  --color-input: var(--input);
  --color-ring: var(--ring);
  --color-primary-ink: var(--primary-ink);
  --color-toast-link: var(--toast-link);
  --radius-xs: 2px; --radius-sm: 8px; --radius-md: 10px; --radius-lg: 12px; --radius-xl: 14px; --radius-2xl: 16px; --radius-3xl: 20px;
  --text-micro: 11px; --text-caption-sm: 11.5px; --text-caption: 12px; --text-caption-md: 12.5px; --text-small: 13px;
  --text-small-md: 13.5px; --text-body-sm: 14px; --text-body-md: 14.5px; --text-body: 15px; --text-input: 16px;
  --text-lead: 17px; --text-title: 18px; --text-display-sm: 22px; --text-display: 30px;
}}

@layer base {{
  * {{ border-color: var(--border); }}
  body {{ background: var(--background); color: var(--foreground); font-family: var(--font-sans); font-size: 15px; line-height: 1.5; }}
  :focus-visible {{ outline: 2px solid var(--primary); outline-offset: 2px; }}
  h1 {{ font-size: 30px; font-weight: 600; letter-spacing: -0.02em; line-height: 1.1; overflow-wrap: anywhere; }}
  input, select, textarea {{ font-size: 16px; }} /* 16 px stops iOS zooming on focus */
  /* Form controls: the border turns primary and a 3 px 32 % ring is added (buttons and links use the 2 px outline above). */
  input:focus-visible, select:focus-visible, textarea:focus-visible {{ outline: none; border-color: var(--primary);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--primary) 32%, transparent); }}
  [aria-invalid='true'] {{ border-color: var(--destructive); }} /* with an adjacent text message; never colour alone */
}}
@layer components {{
  /* Tracker chip: 38 px square (40 on running cards), radius 12 */
  .tracker-chip {{ width: 38px; height: 38px; border-radius: 12px; display: flex; align-items: center; justify-content: center; flex: none;
    background: oklch(var(--chip-bg-l) var(--chip-bg-c) var(--h)); color: oklch(var(--chip-fg-l) var(--chip-fg-c) var(--h)); }}
  .tracker-bar {{ background: oklch(var(--chip-bar-l) var(--chip-bar-c) var(--h)); }}
  {" ".join(f".tracker-{k} {{ --h: var(--hue-{k}); }}" for k in HUE)}
}}
'''
open(OUT+'tokens/tokens.css','w').write(css)
print('contrast failures (<4.5):')
for th in ('light','dark'):
    for r in ct[th]:
        if not r['passAA']: print(' ',th,r)
print('chip contrast min', min(v[t]['chipContrast'] for v in trk.values() for t in ('light','dark')))
