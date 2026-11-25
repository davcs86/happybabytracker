import { expect, test, type Page } from '@playwright/test';
import { expectAccessible } from './helpers.js';
import { UI_ORIGIN } from './stub-api.js';

const attr = (page: Page, name: string) => page.locator('html').getAttribute(name);
const colorScheme = (page: Page) =>
	page.evaluate(() => getComputedStyle(document.documentElement).colorScheme);

async function choose(page: Page, theme: string, back: string) {
	return page.request.post('/theme', {
		form: { theme, return: back },
		headers: { origin: UI_ORIGIN },
		maxRedirects: 0
	});
}

test.describe('no choice made', () => {
	test.use({ colorScheme: 'dark' });

	test('auto follows the device before first paint and live', async ({ page }) => {
		await page.goto('/login');
		expect(await attr(page, 'data-theme')).toBe('auto');
		expect(await attr(page, 'data-resolved')).toBe('dark');
		expect(await colorScheme(page)).toBe('dark');
		await expectAccessible(page);

		await page.emulateMedia({ colorScheme: 'light' });
		await expect(page.locator('html')).toHaveAttribute('data-resolved', 'light');
		expect(await colorScheme(page)).toBe('light');
		await expectAccessible(page);
	});
});

test('a choice is stored in a cookie and rendered by the server', async ({ page }) => {
	const response = await choose(page, 'dark', '/login');
	expect(response.status()).toBe(303);
	expect(response.headers()['location']).toBe('/login');
	expect(response.headers()['set-cookie']).toMatch(/^hbt_theme=dark;/);

	await page.goto('/login');
	expect(await attr(page, 'data-theme')).toBe('dark');
	expect(await attr(page, 'data-resolved')).toBe('dark');
	expect(await colorScheme(page)).toBe('dark');
	await expectAccessible(page);

	await choose(page, 'light', '/login');
	await page.goto('/login');
	expect(await attr(page, 'data-theme')).toBe('light');
	expect(await colorScheme(page)).toBe('light');
	await expectAccessible(page);
});

test('the choice applies with JavaScript off', async ({ browser }) => {
	const context = await browser.newContext({ javaScriptEnabled: false, baseURL: UI_ORIGIN });
	const page = await context.newPage();
	await choose(page, 'dark', '/login');
	await page.goto('/login');
	expect(await attr(page, 'data-theme')).toBe('dark');
	expect(await attr(page, 'data-resolved')).toBeNull();
	expect(await colorScheme(page)).toBe('dark');
	await context.close();
});

test('JavaScript off with no choice follows the device', async ({ browser }) => {
	const context = await browser.newContext({
		javaScriptEnabled: false,
		baseURL: UI_ORIGIN,
		colorScheme: 'dark'
	});
	const page = await context.newPage();
	await page.goto('/login');
	expect(await colorScheme(page)).toBe('dark');
	await context.close();
});

test('a value that is not a theme is ignored and a return outside the app goes home', async ({
	page
}) => {
	for (const back of ['//evil.example', 'https://evil.example', '/\\evil.example']) {
		const response = await choose(page, 'sepia', back);
		expect(response.status()).toBe(303);
		expect(response.headers()['location']).toBe('/');
		expect(response.headers()['set-cookie']).toBeUndefined();
	}
});
