import { expect, test, type BrowserContext, type Page, type Response } from '@playwright/test';
import {
	expectAccessible,
	expectLanding,
	login,
	sessionCookie,
	signIn,
	signInWithForm
} from './helpers.js';
import {
	ADMIN,
	ANA,
	ISSUER_URL,
	MIA,
	TOKENS,
	linkStubIdentity,
	resetStub,
	setAuthMode,
	setScenario,
	type AuthMode
} from './stub-api.js';
import { configureIssuer, issuerAuthorizeLog, issuerDown, resetIssuer } from './stub-issuer.js';

const ANA_SUB = 'stub-subject-ana-0001';
const ADMIN_SUB = 'stub-subject-admin-0001';
const OIDC_COOKIE = 'hbt_oidc';
const SSO = 'Continue with single sign-on';
const DID_NOT_COMPLETE = 'Single sign-on did not complete';
const UNAVAILABLE = 'Single sign-on is unavailable';

test.beforeEach(async () => {
	await resetStub();
	await resetIssuer();
});

// The mode, and Ana linked to the issuer's user, as a link would leave her.
async function setup(mode: AuthMode, who = { sub: ANA_SUB, username: ANA.username }) {
	await setAuthMode(mode);
	await linkStubIdentity(ISSUER_URL, who.sub, who.username);
	await configureIssuer({ user: { sub: who.sub, preferredUsername: 'ana.sso' } });
}

const isCallback = (r: Response) => new URL(r.url()).pathname === '/auth/callback';

async function continueWithSso(page: Page): Promise<Response> {
	const callback = page.waitForResponse(isCallback);
	await page.getByRole('button', { name: SSO }).click();
	return callback;
}

const oidcCookie = async (context: BrowserContext) =>
	(await context.cookies()).find((c) => c.name === OIDC_COOKIE);

async function expectDidNotComplete(page: Page) {
	await expect(page.getByRole('heading', { level: 1 })).toHaveText(DID_NOT_COMPLETE);
	await expect(page.getByRole('link', { name: 'Go to sign in' })).toHaveAttribute('href', '/login');
}

test('oidc shows only the single sign-on button, and the callback answers 200', async ({
	page,
	context
}) => {
	await setup('oidc');
	await page.goto('/login');
	await expect(page.getByRole('button', { name: SSO })).toBeVisible();
	await expect(page.getByLabel('Username')).toHaveCount(0);
	await expect(page.getByLabel('Password')).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Sign in' })).toHaveCount(0);

	const callback = await continueWithSso(page);
	expect(callback.status()).toBe(200);
	expect(callback.headers().location).toBeUndefined();
	await expectLanding(page, { kid: MIA });
	const cookie = await sessionCookie(context);
	expect(cookie).toMatchObject({ httpOnly: true, secure: true, sameSite: 'Strict', path: '/' });
	const days = ((cookie?.expires ?? 0) - Date.now() / 1000) / 86400;
	expect(days).toBeGreaterThan(29.9);
	expect(days).toBeLessThanOrEqual(30);
	expect(await oidcCookie(context)).toBeUndefined();
	expect(await issuerAuthorizeLog()).toEqual([{ prompt: null, maxAge: null }]);
});

test('the Signing you in page has one h1 and one Continue link to the landing', async ({
	page
}) => {
	await setup('oidc');
	// Holds the page on the callback: the meta refresh's navigation is cancelled.
	await page.route(
		(url) => url.pathname === '/',
		(route) => route.abort('aborted')
	);
	await page.goto('/login');
	const callback = await continueWithSso(page);
	expect(callback.status()).toBe(200);
	await expect(page.getByRole('heading', { level: 1 })).toHaveText('Signing you in');
	await expect(page.getByRole('heading')).toHaveCount(1);
	const links = page.getByRole('link');
	await expect(links).toHaveCount(1);
	await expect(links).toHaveText('Continue');
	await expect(links).toHaveAttribute('href', '/');
	await expect(page.locator('meta[http-equiv="refresh"]')).toHaveAttribute('content', '0;url=/');
});

test('the session lists the device label the password login sends', async ({ page, context }) => {
	await setup('both');
	const current = page
		.getByRole('list', { name: 'Sessions' })
		.getByRole('listitem')
		.filter({ hasText: 'Active now' });
	await signInWithForm(page, ANA);
	await page.goto('/account');
	const byPassword = await current.innerText();
	expect(byPassword).toMatch(/^Chrome on Linux\b/);

	await context.clearCookies();
	await page.goto('/login');
	await continueWithSso(page);
	await expectLanding(page, { kid: MIA });
	await page.goto('/account');
	expect(await current.innerText()).toBe(byPassword);
});

test('next survives the round trip; an off-site next lands on /', async ({ page, context }) => {
	await setup('oidc');
	await page.goto(`/login?next=${encodeURIComponent(`/kids/${MIA.id}/timeline`)}`);
	await continueWithSso(page);
	await expect(page).toHaveURL(new RegExp(`/kids/${MIA.id}/timeline$`));

	await context.clearCookies();
	await page.goto(`/login?next=${encodeURIComponent('//evil.example')}`);
	await continueWithSso(page);
	await expectLanding(page, { kid: MIA });
});

test('an admin linked by SSO lands on /admin/parents and is kept out of kid pages', async ({
	page
}) => {
	await setup('oidc', { sub: ADMIN_SUB, username: ADMIN.username });
	await page.goto('/login');
	await continueWithSso(page);
	await expect(page).toHaveURL(/\/admin\/parents$/);
	await expect(page.getByRole('heading', { name: 'Parents' })).toBeVisible();
	const kid = await page.request.get(`/kids/${MIA.id}`, { maxRedirects: 0 });
	expect(kid.status()).toBe(303);
	expect(kid.headers().location).toBe('/admin/parents');
});

for (const [what, behaviour] of [
	['an unlinked sub', { user: { sub: 'stub-subject-nobody-0001' } }],
	['a denied sign-in', { deny: true }]
] as const) {
	test(`${what} does not complete and sets no session`, async ({ page, context }) => {
		await setup('oidc');
		await configureIssuer(behaviour);
		await page.goto('/login');
		const callback = await continueWithSso(page);
		expect(callback.status()).toBe(200);
		await expectDidNotComplete(page);
		expect(await sessionCookie(context)).toBeUndefined();
		expect(await oidcCookie(context)).toBeUndefined();
	});
}

test('a replayed callback does not complete and keeps the session', async ({ page, context }) => {
	await setup('oidc');
	await page.goto('/login');
	const callback = await continueWithSso(page);
	await expectLanding(page, { kid: MIA });
	const before = await sessionCookie(context);

	const replay = await page.goto(callback.url());
	expect(replay?.status()).toBe(200);
	await expectDidNotComplete(page);
	expect((await sessionCookie(context))?.value).toBe(before?.value);
});

test('the callback answers 200 and clears hbt_oidc while the API is down, even signed in', async ({
	page,
	context
}) => {
	await setup('both');
	await signIn(page, ANA);
	const start = await page.request.post('/auth/sso', {
		form: { next: '' },
		headers: { origin: new URL(page.url()).origin, accept: 'text/html' },
		maxRedirects: 0
	});
	expect(start.status()).toBe(303);
	expect(await oidcCookie(context)).toBeDefined();
	expect(await sessionCookie(context)).toBeDefined();

	await setScenario('down');
	const res = await page.goto('/auth/callback?code=stub-code&state=stub-state');
	expect(res?.status()).toBe(200);
	await expect(page.getByRole('heading', { level: 1 })).toHaveText(DID_NOT_COMPLETE);
	expect(await oidcCookie(context)).toBeUndefined();
});

test('the start sets a hardened hbt_oidc cookie; without it the callback does not complete', async ({
	page,
	context
}) => {
	await setup('oidc');
	// A redirect is not routed: read the start's own answer, and pass it on without its cookie
	// (route.fetch stores it in the context, so it is cleared there too).
	let setCookie = '';
	await page.route(
		(url) => url.pathname === '/auth/sso',
		async (route) => {
			const res = await route.fetch({ maxRedirects: 0 });
			const headers = res.headersArray();
			setCookie = headers.find((h) => /^set-cookie$/i.test(h.name))?.value ?? '';
			await context.clearCookies({ name: OIDC_COOKIE });
			await route.fulfill({
				status: res.status(),
				headers: Object.fromEntries(
					headers.filter((h) => !/^set-cookie$/i.test(h.name)).map((h) => [h.name, h.value])
				)
			});
		}
	);
	await page.goto('/login');
	const callback = await continueWithSso(page);
	const [pair, ...attributes] = setCookie.split(/;\s*/);
	expect(pair).toMatch(new RegExp(`^${OIDC_COOKIE}=[A-Za-z0-9_-]+$`));
	expect(attributes.map((a) => a.toLowerCase()).sort()).toEqual(
		['httponly', 'max-age=600', 'path=/auth', 'samesite=lax', 'secure'].sort()
	);
	expect(callback.status()).toBe(200);
	await expectDidNotComplete(page);
	expect(await sessionCookie(context)).toBeUndefined();
	expect(await oidcCookie(context)).toBeUndefined();
});

test('both shows the password card and the SSO button; the password works with the issuer down', async ({
	page,
	context
}) => {
	await setup('both');
	await page.goto('/login');
	await expect(page.getByLabel('Username')).toBeVisible();
	await expect(page.getByLabel('Password')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible();
	await expect(page.getByRole('button', { name: SSO })).toBeVisible();

	await issuerDown();
	await page.getByRole('button', { name: SSO }).click();
	await page.waitForLoadState();
	expect(await sessionCookie(context)).toBeUndefined();

	await page.goto('/login');
	await login(page, ANA.username, ANA.password);
	await expectLanding(page, { kid: MIA });
});

test('local shows no SSO button, and POST /auth/sso is unavailable', async ({ page }) => {
	await setup('local');
	await page.goto('/login');
	await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible();
	await expect(page.getByRole('button', { name: SSO })).toHaveCount(0);
	const res = await page.request.post('/auth/sso', {
		form: { next: '' },
		headers: { origin: new URL(page.url()).origin },
		maxRedirects: 0
	});
	expect(res.status()).toBe(503);
	expect(await res.text()).toContain(UNAVAILABLE);
});

test('referrer-policy is same-origin on the callback, invite and reset pages', async ({
	request
}) => {
	for (const path of ['/auth/callback', `/invite/${TOKENS.invite}`, `/reset/${TOKENS.reset}`]) {
		const res = await request.get(path);
		expect(res.status()).toBe(200);
		expect(res.headers()['referrer-policy']).toBe('same-origin');
		expect(res.headers()['cache-control']).toBe('no-store');
	}
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('the oidc sign-in works', async ({ page }) => {
		await setup('oidc');
		await page.goto('/login');
		await continueWithSso(page);
		await expectLanding(page, { kid: MIA });
	});
});

for (const colorScheme of ['light', 'dark'] as const) {
	test.describe(`accessibility, ${colorScheme}`, () => {
		test.use({ colorScheme });

		for (const mode of ['oidc', 'both'] as const) {
			test(`/login in ${mode}`, async ({ page }) => {
				await setup(mode);
				await page.goto('/login');
				await expect(page.getByRole('button', { name: SSO })).toBeVisible();
				await expectAccessible(page);
			});
		}

		test('/auth/sso', async ({ page }) => {
			const res = await page.goto('/auth/sso');
			expect(res?.status()).toBe(200);
			await expect(page.getByRole('heading', { level: 1 })).toHaveText(UNAVAILABLE);
			await expect(page.getByRole('link', { name: 'Go to sign in' })).toBeVisible();
			await expectAccessible(page);
		});

		test('the callback that did not complete', async ({ page }) => {
			const res = await page.goto('/auth/callback');
			expect(res?.status()).toBe(200);
			await expectDidNotComplete(page);
			await expectAccessible(page);
		});
	});
}
