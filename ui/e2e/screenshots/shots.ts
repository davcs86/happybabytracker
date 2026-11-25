import { appendFileSync, mkdirSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { test as base, expect, type Page } from '@playwright/test';
import { AREAS, type Area } from './areas.js';
import { SHOTS_NOW } from './clock.js';
import { login } from '../helpers.js';
import { resetStub } from '../stub-api.js';

export const REPO_ROOT = resolve(import.meta.dirname, '../../..');
export const SHOTS_DIR = join(REPO_ROOT, 'docs/screenshots');
// Playwright empties test-results at the start of a run, so the manifest holds one run only.
export const MANIFEST = join(REPO_ROOT, 'ui/test-results/screenshots-manifest.jsonl');

export type ManifestEntry = {
	area: Area;
	name: string;
	about: string;
	project: string;
	path: string;
};

const NAME = /^\d{2}-[a-z0-9]+(-[a-z0-9]+)*$/;
export const NOW_MS = Date.parse(SHOTS_NOW);
export const minutesAgo = (m: number) =>
	new Date(NOW_MS - m * 60_000).toISOString().replace(/\.\d{3}Z$/, 'Z');

// Every test starts from a fresh stub with the browser clock frozen at SHOTS_NOW.
export const test = base.extend<{ frozen: void }>({
	frozen: [
		async ({ page }, use) => {
			await resetStub();
			await page.clock.setFixedTime(new Date(NOW_MS));
			await page.emulateMedia({ reducedMotion: 'reduce' });
			await use();
		},
		{ auto: true }
	]
});
export { expect };

export async function signInAs(page: Page, who: { username: string; password: string }) {
	await page.context().clearCookies();
	await page.goto('/login');
	await login(page, who.username, who.password);
	await page.waitForURL((u) => u.pathname !== '/login');
	await page.waitForLoadState('networkidle');
}

export async function shot(page: Page, area: Area, name: string, about: string) {
	if (!AREAS.includes(area)) throw new Error(`unknown area ${area}`);
	if (!NAME.test(name)) throw new Error(`shot name ${name} is not NN-slug`);
	const project = test.info().project.name;
	await page.waitForLoadState('networkidle');
	await page.evaluate(() => document.fonts.ready);
	await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur?.());
	// A caret left from typing can still be painted for a frame after the blur.
	await page.evaluate(
		() => new Promise((done) => requestAnimationFrame(() => requestAnimationFrame(done)))
	);
	await page.mouse.move(0, 0);
	const path = join(SHOTS_DIR, area, `${name}-${project}.png`);
	mkdirSync(dirname(path), { recursive: true });
	// A fixed bottom bar lands mid-page in a plain full-page capture: grow the viewport to the page.
	// Growing the viewport can reflow the page, so repeat until the page fits.
	const size = page.viewportSize()!;
	let height = size.height;
	const settle = () =>
		page.evaluate(
			() => new Promise((done) => requestAnimationFrame(() => requestAnimationFrame(done)))
		);
	for (let pass = 0; pass < 8; pass++) {
		await settle();
		await page.waitForLoadState('networkidle');
		const needed = await page.evaluate(() => document.documentElement.scrollHeight);
		if (needed <= height) break;
		height = needed;
		await page.setViewportSize({ width: size.width, height });
	}
	await page.screenshot({ path, fullPage: true, animations: 'disabled', caret: 'hide' });
	await page.setViewportSize(size);
	const entry: ManifestEntry = { area, name, about, project, path: new URL(page.url()).pathname };
	mkdirSync(dirname(MANIFEST), { recursive: true });
	appendFileSync(MANIFEST, `${JSON.stringify(entry)}\n`);
}

// Hydration can close a details element opened just before it: click until it stays open.
export async function openDetails(page: Page, summary: string | RegExp) {
	const toggle = page.locator('summary').filter({ hasText: summary }).first();
	const details = toggle.locator('xpath=..');
	await expect(async () => {
		if ((await details.getAttribute('open')) === null) await toggle.click();
		await expect(details).toHaveAttribute('open', '', { timeout: 1000 });
	}).toPass({ timeout: 10_000 });
}
