import { request as httpRequest } from 'node:http';
import { expect, test, type Page } from '@playwright/test';
import { expectAccessible, expectLanding, login } from './helpers.js';
import { ANA, LEO, MIA, UI_PORT, resetStub } from './stub-api.js';

test.beforeEach(async ({ page }) => {
	await resetStub();
	await page.goto('/login');
	await login(page, ANA.username, ANA.password);
	await expectLanding(page, { kid: MIA });
});

const primary = (page: Page) => page.getByRole('navigation', { name: 'Primary' });

test('a tab bar at phone width: Home, Timeline, Growth and More', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	const nav = primary(page);
	await expect(nav).toHaveCount(1);
	const box = await nav.boundingBox();
	expect(box!.y + box!.height).toBeCloseTo(844, 0);
	expect(box!.width).toBeCloseTo(390, 0);
	await expect(nav.getByRole('link')).toHaveText(['Home', 'Timeline', 'Growth']);
	await expect(nav.getByRole('link', { name: 'Home' })).toHaveAttribute('aria-current', 'page');
	await expect(nav.getByRole('button', { name: 'More' })).toBeVisible();
	await expectAccessible(page);
});

test('the More sheet lists the other pages, Sign out last', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	const sheet = page.getByRole('dialog', { name: 'More' });
	await expect(async () => {
		await page.getByRole('button', { name: 'More', exact: true }).click();
		await expect(sheet).toBeVisible({ timeout: 1000 });
	}).toPass();
	await expect(sheet.getByRole('link')).toHaveText([
		/^\s*Vaccines/,
		/^\s*Medication/,
		/^\s*Milestones/,
		/^\s*Export PDF/,
		/^\s*Modules/,
		/^\s*Child profile/,
		/^\s*My profile/
	]);
	await expect(sheet.getByRole('button')).toHaveText([
		/^\s*Change theme/,
		/^\s*Sign out/,
		/^\s*Close$/
	]);
	await expectAccessible(page);
	await page.keyboard.press('Escape');
	await expect(sheet).toBeHidden();
});

test('a side navigation from 768 px carries the same entries inline and no More', async ({
	page
}) => {
	await page.setViewportSize({ width: 1280, height: 800 });
	const nav = primary(page);
	await expect(nav).toHaveCount(1);
	const box = await nav.boundingBox();
	expect(box).toMatchObject({ x: 0, width: 225 });
	expect(box!.height).toBeCloseTo(800, 0);
	await expect(nav.getByRole('link')).toHaveText([
		'Home',
		'Timeline',
		'Growth',
		'Vaccines',
		'Medication',
		'Milestones',
		'Export PDF',
		'Modules',
		'Child profile',
		'My profile'
	]);
	await expect(nav.getByRole('button')).toHaveText(['Change theme', 'Sign out']);
	await expect(page.getByRole('button', { name: 'More' })).toHaveCount(0);
	await expectAccessible(page);
});

test('the header names the child and the switcher offers every child', async ({ page }) => {
	const switcher = page.getByRole('link', { name: `Child: ${MIA.name}. Switch child` });
	await expect(switcher).toBeVisible();
	await expect(page.getByRole('link', { name: 'Notification settings' })).toBeVisible();
	const sheet = page.getByRole('dialog', { name: 'Switch child' });
	await expect(async () => {
		await switcher.click();
		await expect(sheet).toBeVisible({ timeout: 1000 });
	}).toPass();
	await expect(sheet.getByRole('link', { name: new RegExp(`^${MIA.name}`) })).toHaveAttribute(
		'aria-current',
		'true'
	);
	await expectAccessible(page);
});

test('the switcher link is the Choose a child page without JavaScript', async ({ browser }) => {
	const context = await browser.newContext({ javaScriptEnabled: false });
	const page = await context.newPage();
	await page.goto('/login');
	await login(page, ANA.username, ANA.password);
	await page.getByRole('link', { name: `Child: ${MIA.name}. Switch child` }).click();
	await expect(page).toHaveURL(/\/kids\?to=$/);
	await expect(page.getByRole('heading', { name: 'Choose a child', level: 1 })).toBeVisible();
	await page.getByRole('link', { name: new RegExp(`^${LEO.name}`) }).click();
	await expect(page).toHaveURL(new RegExp(`/kids/${LEO.id}$`));
	await context.close();
});

test('Change theme applies at once and persists', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	const more = page.getByRole('dialog', { name: 'More' });
	await expect(async () => {
		await page.getByRole('button', { name: 'More', exact: true }).click();
		await expect(more).toBeVisible({ timeout: 1000 });
	}).toPass();
	await more.getByRole('button', { name: /^Change theme/ }).click();
	const sheet = page.getByRole('dialog', { name: 'Change theme' });
	await sheet.getByRole('radio', { name: 'Dark' }).click();
	await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
	await expect(page.locator('html')).toHaveAttribute('data-resolved', 'dark');
	await expectAccessible(page);
	await sheet.getByRole('button', { name: 'Done' }).click();
	await page.reload();
	await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
});

test('the page fits a phone without horizontal scroll', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	const overflow = await page.evaluate(
		() => document.documentElement.scrollWidth - document.documentElement.clientWidth
	);
	expect(overflow).toBe(0);
});

test('/ is sent with Cache-Control: no-store', async ({ page }) => {
	const response = await page.request.get('/');
	expect(response.status()).toBe(200);
	expect(response.headers()['cache-control']).toBe('no-store');
});

test('a foreign host gets 400 Bad origin, except on /healthz', async ({ request }) => {
	const forwarded = await request.get('/', { headers: { 'x-forwarded-host': 'evil.example' } });
	expect(forwarded.status()).toBe(400);
	expect(await forwarded.text()).toBe('Bad origin');

	const raw = await new Promise<{ status: number; body: string }>((resolve, reject) => {
		const req = httpRequest(
			{ host: '127.0.0.1', port: UI_PORT, path: '/login', headers: { host: 'evil.example' } },
			(res) => {
				let body = '';
				res.on('data', (c) => (body += c));
				res.on('end', () => resolve({ status: res.statusCode ?? 0, body }));
			}
		);
		req.on('error', reject);
		req.setHeader('x-forwarded-proto', 'http');
		req.setHeader('x-forwarded-for', '203.0.113.10');
		req.end();
	});
	expect(raw).toEqual({ status: 400, body: 'Bad origin' });

	const health = await request.get('/healthz', { headers: { 'x-forwarded-host': 'evil.example' } });
	expect(health.status()).toBe(200);
});

test('a parent gets the 404 page on /admin/*', async ({ page }) => {
	for (const path of ['/admin', '/admin/parents']) {
		const response = await page.goto(path);
		expect(response?.status()).toBe(404);
		await expect(page.getByRole('heading', { name: 'Not found' })).toBeVisible();
		await expectAccessible(page);
	}
});
