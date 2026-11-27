import { test, expect } from '@playwright/test';
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
