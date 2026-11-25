import { readFileSync } from 'node:fs';
import { expect, test, type Page } from '@playwright/test';
import { ADMIN, ANA, MIA, TOKENS, resetStub } from '../../e2e/stub-api.js';
import { login } from '../../e2e/helpers.js';
import index from './index.json' with { type: 'json' };
import { MATCHED_PAGES } from './matched.js';

// Compares a page with the design's baseline. Only pages whose stub data equals the prototype's sample data are listed in matched.ts.
const ROUTE_ID = { '{kid}': MIA.id, '{dose}': 'd-1', '{token}': TOKENS.invite };
const route = (r: string) => Object.entries(ROUTE_ID).reduce((acc, [k, v]) => acc.replace(k, v), r);
// A baseline's size is in its PNG header: the frame is the viewport.
const sizeOf = (variant: string, file: string) => {
	const png = readFileSync(`tests/visual/__baselines__/${variant}/${file}`);
	return { width: png.readUInt32BE(16), height: png.readUInt32BE(20) };
};
const PUBLIC = /^\/(login|invite|reset)/;

async function signIn(page: Page, path: string) {
	if (PUBLIC.test(path)) return;
	const who = path.startsWith('/admin') ? ADMIN : ANA;
	await page.goto('/login');
	await login(page, who.username, who.password);
	await page.waitForURL((u) => u.pathname !== '/login');
}

for (const s of index) {
	test(`${s.page} · ${s.state} · ${s.variant}`, async ({ page }, info) => {
		test.skip(info.project.name !== s.variant, 'other variant');
		const tolerance = MATCHED_PAGES[s.page];
		test.fixme(tolerance === undefined, 'stub data differs from the prototype sample data');
		await page.setViewportSize(sizeOf(s.variant, `${s.page}/${s.file.split('/').pop()}`));
		await resetStub();
		await signIn(page, s.route);
		await page.goto(route(s.route));
		await page.evaluate(() => document.fonts.ready);
		if (s.state === 'default-full-height') {
			const h = await page.evaluate(() => document.documentElement.scrollHeight);
			await page.setViewportSize({ width: page.viewportSize()!.width, height: h });
		}
		await expect(page).toHaveScreenshot([s.page, s.file.split('/').pop()!], {
			maxDiffPixelRatio: tolerance,
			animations: 'disabled',
			caret: 'hide'
		});
	});
}
