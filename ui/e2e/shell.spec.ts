import { request as httpRequest } from 'node:http';
import { expect, test, type Page } from '@playwright/test';
import { expectAccessible, login } from './helpers.js';
import { ANA, UI_PORT, resetStub } from './stub-api.js';

test.beforeEach(async ({ page }) => {
	await resetStub();
	await page.goto('/login');
	await login(page, ANA.username, ANA.password);
	await expect(page.getByRole('heading', { name: `Welcome, ${ANA.displayName}` })).toBeVisible();
});

async function navLinks(page: Page) {
	const nav = page.getByRole('navigation', { name: 'Main' });
	const links = await nav
		.getByRole('link')
		.evaluateAll((els) =>
			els.map((a) => ({ text: a.textContent?.trim(), href: a.getAttribute('href') }))
		);
	return { box: await nav.boundingBox(), links };
}

test('a bottom bar at phone width and side navigation at desktop width, same links', async ({
	page
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	const phone = await navLinks(page);
	await expectAccessible(page);
	expect(phone.box).not.toBeNull();
	expect(phone.box!.y + phone.box!.height).toBeCloseTo(844, 0);
	expect(phone.box!.width).toBeCloseTo(390, 0);

	await page.setViewportSize({ width: 1280, height: 800 });
	const desktop = await navLinks(page);
	await expectAccessible(page);
	expect(desktop.box).not.toBeNull();
	expect(desktop.box!.x).toBe(0);
	expect(desktop.box!.height).toBeCloseTo(800, 0);
	expect(desktop.box!.width).toBeLessThan(300);

	expect(phone.links).toEqual([
		{ text: 'Home', href: '/' },
		{ text: 'Account', href: '/account' }
	]);
	expect(desktop.links).toEqual(phone.links);
	await expect(page.getByRole('link', { name: 'Home' })).toHaveAttribute('aria-current', 'page');
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
