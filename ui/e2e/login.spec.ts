import { expect, test } from '@playwright/test';
import {
	expectAccessible,
	expectFieldError,
	expectLanding,
	login,
	sessionCookie
} from './helpers.js';
import {
	ADMIN,
	ANA,
	MIA,
	receivedRequests,
	resetStub,
	setAuthMode,
	setScenario
} from './stub-api.js';

const ADDRESS = '198.51.100.23';

test.beforeEach(async ({ context }) => {
	await resetStub();
	await context.setExtraHTTPHeaders({ 'x-forwarded-proto': 'http', 'x-forwarded-for': ADDRESS });
});

test('an unauthenticated visit to / goes to /login with next', async ({ page }) => {
	await page.goto('/');
	await expect(page).toHaveURL(/\/login\?next=%2F$/);
	await expectAccessible(page);
});

test('a parent logs in and lands on the kid home with a hardened 30-day cookie', async ({
	page,
	context
}) => {
	await page.goto('/login');
	await expectAccessible(page);
	await login(page, ANA.username, ANA.password);
	await expectLanding(page, { kid: MIA });
	await expectAccessible(page);
	const cookie = await sessionCookie(context);
	expect(cookie).toMatchObject({ httpOnly: true, secure: true, sameSite: 'Strict', path: '/' });
	const days = ((cookie?.expires ?? 0) - Date.now() / 1000) / 86400;
	expect(days).toBeGreaterThan(29.9);
	expect(days).toBeLessThanOrEqual(30);

	const posted = (await receivedRequests()).find((r) => r.method === 'POST');
	expect(posted).toMatchObject({ path: '/v1/session', authorized: true, clientAddress: ADDRESS });
});

test('the admin lands on /admin/parents with a 12-hour cookie', async ({ page, context }) => {
	await page.goto('/login');
	await login(page, ADMIN.username, ADMIN.password);
	await expect(page).toHaveURL(/\/admin\/parents$/);
	await expect(page.getByRole('heading', { name: 'Parents' })).toBeVisible();
	await expectAccessible(page);
	const hours = (((await sessionCookie(context))?.expires ?? 0) - Date.now() / 1000) / 3600;
	expect(hours).toBeGreaterThan(11.9);
	expect(hours).toBeLessThanOrEqual(12);
});

test('a wrong password and an unknown user get one generic message', async ({ page, context }) => {
	for (const [username, password] of [
		[ANA.username, 'not-the-password'],
		['nobody', ANA.password]
	]) {
		await page.goto('/login');
		const response = await login(page, username, password);
		expect(response.status()).toBe(400);
		await expect(page).toHaveURL(/\/login$/);
		await expect(page.getByRole('alert')).toHaveText('Incorrect username or password');
		await expectAccessible(page);
		expect(await sessionCookie(context)).toBeUndefined();
	}
});

test('blank fields are refused on the field and no sign-in is sent', async ({ page }) => {
	await page.goto('/login');
	await page.getByRole('button', { name: 'Sign in' }).click();
	await expectFieldError(page.getByLabel('Username'), 'Enter your username');
	await expectFieldError(page.getByLabel('Password'), 'Enter your password');
	await expectAccessible(page);
	expect((await receivedRequests()).filter((r) => r.method === 'POST')).toEqual([]);
});

test('a rate-limited login says to wait a minute', async ({ page, context }) => {
	await setScenario('rate_limited');
	await page.goto('/login');
	const response = await login(page, ANA.username, ANA.password);
	expect(response.status()).toBe(429);
	await expect(page.getByRole('alert')).toHaveText('Too many attempts, try again in a minute');
	await expectAccessible(page);
	expect(await sessionCookie(context)).toBeUndefined();
});

test('next is honoured for a same-origin path', async ({ page }) => {
	await page.goto('/account?tab=1');
	await expect(page).toHaveURL(/\/login\?next=%2Faccount%3Ftab%3D1$/);
	await login(page, ANA.username, ANA.password);
	await expect(page).toHaveURL(/127\.0\.0\.1:\d+\/account\?tab=1$/);
});

for (const next of [
	'//evil.example',
	'/\\evil.example',
	'https://evil.example',
	'javascript:alert(1)'
]) {
	test(`next=${next} is ignored`, async ({ page }) => {
		await page.goto(`/login?next=${encodeURIComponent(next)}`);
		await login(page, ANA.username, ANA.password);
		await expectLanding(page, { kid: MIA });
	});
}

test('a signed-in visit to /login goes home', async ({ page }) => {
	await page.goto('/login');
	await login(page, ANA.username, ANA.password);
	await page.goto('/login');
	await expectLanding(page, { kid: MIA });
});

test('GET /logout is 405', async ({ request }) => {
	expect((await request.get('/logout', { maxRedirects: 0 })).status()).toBe(405);
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('the login form still works', async ({ page }) => {
		await page.goto('/login');
		await login(page, ANA.username, ANA.password);
		await expectLanding(page, { kid: MIA });
	});

	test('a password login posted by hand in oidc says password sign-in is turned off', async ({
		page,
		context
	}) => {
		await page.goto('/login');
		await setAuthMode('oidc');
		const response = await login(page, ANA.username, ANA.password);
		expect(response.status()).toBe(403);
		await expect(page.getByRole('alert')).toHaveText('Password sign-in is turned off');
		expect(await sessionCookie(context)).toBeUndefined();
	});
});
