#!/usr/bin/env python3
"""usage: gen_visual.py <out_dir(capture)> <dest_dir>  -> dest/screenshots/INDEX.md, dest/tests/visual/{index.json,states.ts,pages.visual.spec.ts,playwright.visual.config.ts}"""
import sys,json,glob,os,re
out,dest=sys.argv[1:3]
ROUTE={'kid-home':'/kids/{kid}','timeline':'/kids/{kid}/timeline','growth':'/kids/{kid}/growth','vaccines':'/kids/{kid}/vaccines','milestones':'/kids/{kid}/milestones','milestone-form':'/kids/{kid}/milestones/new','medication':'/kids/{kid}/medication','course':'/kids/{kid}/medication/new','dose':'/kids/{kid}/medication/doses/{dose}','symptom':'/kids/{kid}/log/symptom','export':'/kids/{kid}/export','pdf-export':'/kids/{kid}/export/pdf','child-profile':'/kids/{kid}/settings','my-profile':'/account','notifications':'/account/notifications','choose-child':'/kids','add-a-child':'/kids/new','child-settings':'/kids/{kid}/settings','access':'/kids/{kid}/access','admin':'/admin/parents','sign-in':'/login','set-password':'/invite/{token}','error':'/does-not-exist'}
VARS=['phone-light','phone-dark','desktop-light','desktop-dark','wide-light','wide-dark']
idx=[]; md=['# Screenshot index','','Baselines are PNGs under `screenshots/<page>/<variant>/NN-<state>.png`. Variants: `phone-light`, `phone-dark` (390 px wide frame), `desktop-light`, `desktop-dark` (980 px wide frame, side nav, one column), `wide-light`, `wide-dark` (1280 px wide frame, side nav, two-column block). `NN` is the capture order within one variant (numbering can differ between variants when a state is a duplicate in one of them, so always follow the links below). A state marked **dup** produced a picture identical to an earlier state in that variant (the control changed nothing visible), so no extra file is stored for it; the earlier file is the baseline.','','Images were captured with Geist loaded, animations off, the prototype frame chrome (corner radius, outer shadow, simulation panel) removed.','']
tot=0;dups=0
for mf in sorted(glob.glob(out+'/manifest-*.json')):
    slug=os.path.basename(mf)[9:-5]; man=json.load(open(mf))
    md+=[f'## {slug}','',f'Route in the app: `{ROUTE.get(slug,"?")}`','','| State | Kind | '+' | '.join(VARS)+' |','|---|---|'+'---|'*len(VARS)]
    states={}
    for r in man:
        if r.get('error') or 'label' not in r: continue
        states.setdefault((r['label'],r['kind']),{})[r['variant']]=r
    for (lab,kind),vs in states.items():
        cells=[]
        for v in VARS:
            r=vs.get(v)
            if not r: cells.append('·'); continue
            if r.get('duplicateOf'): cells.append('dup'); dups+=1; continue
            fn=r['file']; cells.append(f'[{fn.split("/")[-1].split("-")[0]}]({fn})'); tot+=1
            idx.append({'page':slug,'route':ROUTE.get(slug),'variant':v,'state':lab,'kind':kind,'file':fn,'note':r.get('note','')})
        md.append(f'| {lab[:70].replace("|","/")} | {kind} | '+' | '.join(cells)+' |')
    md.append('')
md.insert(3,f'{tot} images, {dups} duplicate state records.\n')
os.makedirs(dest+'/tools',exist_ok=True); os.makedirs(dest+'/screenshots',exist_ok=True); os.makedirs(dest+'/tests/visual',exist_ok=True)
open(dest+'/screenshots/INDEX.md','w').write('\n'.join(md))
json.dump(idx,open(dest+'/tests/visual/index.json','w'),indent=1)
labels=sorted({(i['page'],i['state']) for i in idx if i['kind'] not in ('frame','full')})
open(dest+'/tests/visual/states.ts','w').write('''// Registry: how to reach each captured state in the real app.
// `default-frame` and `default-full-height` work out of the box. Every other state needs a stub-API fixture (role, units, next-save outcome,
// children present/absent, ...). Register a handler below; states without one are skipped (test.fixme) and listed in the run output.
import type { Page } from '@playwright/test';
export type Handler = (page: Page, ctx: { kid: string; base: string }) => Promise<void>;
export const handlers: Record<string, Handler> = {
  // 'kid-home::ROLE Caregiver': async (page) => { await useStub(page, { role: 'caregiver' }); await page.reload(); },
};
export const pendingStates = %s;
''' % json.dumps([f'{p}::{s}' for p,s in labels],indent=1))
open(dest+'/tests/visual/pages.visual.spec.ts','w').write('''import { test, expect } from '@playwright/test';
import index from './index.json' with { type: 'json' };
import { handlers } from './states.ts';

// Compares the implementation with the prototype baselines. Run: pnpm exec playwright test -c playwright.visual.config.ts
// Project name (phone-light, ...) selects the variant. ROUTE placeholders: set KID and TOKEN fixtures in e2e/stub-api.
const KID = process.env.VISUAL_KID ?? 'kid-1';
const route = (r: string) => r.replace('{kid}', KID).replace('{dose}', process.env.VISUAL_DOSE ?? 'dose-1').replace('{token}', process.env.VISUAL_TOKEN ?? 'token-1');

for (const s of index as any[]) {
  test(`${s.page} · ${s.state}`, async ({ page }, info) => {
    test.skip(info.project.name !== s.variant, 'other variant');
    const key = `${s.page}::${s.state}`;
    const isDefault = s.state === 'default-frame' || s.state === 'default-full-height';
    test.fixme(!isDefault && !handlers[key], 'no handler registered for this state in tests/visual/states.ts');
    await page.goto(route(s.route));
    await page.evaluate(() => document.fonts.ready);
    if (handlers[key]) await handlers[key](page, { kid: KID, base: '' });
    if (s.state === 'default-full-height') {
      // The prototype frame grows to its content; do the same so the fixed tab bar sits at the bottom of the picture.
      const h = await page.evaluate(() => document.documentElement.scrollHeight);
      await page.setViewportSize({ width: page.viewportSize()!.width, height: h });
    }
    await expect(page).toHaveScreenshot([s.page, s.file.split('/').pop()!], { maxDiffPixelRatio: 0.002, animations: 'disabled', caret: 'hide' });
  });
}
''')
open(dest+'/tests/visual/playwright.visual.config.ts','w').write('''import { defineConfig } from '@playwright/test';
// Copy next to the repo's playwright.config.ts (reuse its webServer / stub API settings), and point snapshotPathTemplate at the baselines.
const base = { deviceScaleFactor: 1, locale: 'en-US', timezoneId: 'America/Los_Angeles' };
export default defineConfig({
  testDir: 'tests/visual',
  testMatch: '**/*.visual.spec.ts',
  // Baselines live in tests/visual/__baselines__/<variant>/<page>/NN-<state>.png : run tools/install-baselines.sh once to copy them from screenshots/.
  snapshotPathTemplate: '{testDir}/__baselines__/{projectName}/{arg}{ext}',
  workers: 1,
  projects: [
    { name: 'phone-light',   use: { ...base, viewport: { width: 390, height: 800 }, colorScheme: 'light' } },
    { name: 'phone-dark',    use: { ...base, viewport: { width: 390, height: 800 }, colorScheme: 'dark' } },
    { name: 'desktop-light', use: { ...base, viewport: { width: 980, height: 780 }, colorScheme: 'light' } },
    { name: 'desktop-dark',  use: { ...base, viewport: { width: 980, height: 780 }, colorScheme: 'dark' } },
    { name: 'wide-light',    use: { ...base, viewport: { width: 1280, height: 780 }, colorScheme: 'light' } },
    { name: 'wide-dark',     use: { ...base, viewport: { width: 1280, height: 780 }, colorScheme: 'dark' } },
  ],
});
''')
open(dest+'/tools/install-baselines.sh','w').write('''#!/usr/bin/env bash
# usage: tools/install-baselines.sh <export_dir> <repo_ui_dir>   (copies screenshots/<page>/<variant>/*.png to <repo_ui_dir>/tests/visual/__baselines__/<variant>/<page>/)
set -euo pipefail
src="$1/screenshots"; dst="$2/tests/visual/__baselines__"
for page in "$src"/*/; do p=$(basename "$page"); for v in "$page"*/; do vv=$(basename "$v"); mkdir -p "$dst/$vv/$p"; cp "$v"*.png "$dst/$vv/$p/"; done; done
echo "installed into $dst"
''')
os.chmod(dest+'/tools/install-baselines.sh',0o755)
print(tot,'images',dups,'dups',len(labels),'pending states')
