import { expect, test, type BrowserContext, type Page, type Response } from '@playwright/test';
import { expectAccessible, expectLanding, sessionCookie } from './helpers.js';
import {
	ADMIN,
	ANA,
	INVITEE,
	ISSUER_URL,
	TOKENS,
	UI_ORIGIN,
	linkStubIdentity,
	moveStubClock,
	resetStub,
	setAuthMode,
	stubAs,
	type AuthMode
} from './stub-api.js';
import { configureIssuer, issuerAuthorizeLog, resetIssuer } from './stub-issuer.js';

const BO_SUB = 'stub-subject-bo-0001';
const BO_ID = 'p-bo';
const OIDC_COOKIE = '__Secure-hbt_oidc';
const SSO = 'Continue with single sign-on';
const DID_NOT_COMPLETE = 'Single sign-on did not complete';
const NO_LONGER_VALID =
	'This link is no longer valid. Ask the person who sent it for a new one. This kind of link lasts 72 hours.';
const FALLBACK_LABEL = 'the single sign-on account you just signed in with';
const RESET_OFF = 'This link works only while password sign-in is on. Ask the person who sent it.';
const GOOD = 'a long enough password';
const INVITE = `/invite/${TOKENS.invite}`;

test.beforeEach(async () => {
	await resetStub();
	await resetIssuer();
});

async function setup(mode: AuthMode, preferredUsername: string | null = 'bo.sso') {
	await setAuthMode(mode);
	await configureIssuer({
		user: preferredUsername === null ? { sub: BO_SUB } : { sub: BO_SUB, preferredUsername }
	});
}

const isCallback = (r: Response) => new URL(r.url()).pathname === '/auth/callback';

async function continueWithSso(page: Page): Promise<Response> {
	const callback = page.waitForResponse(isCallback);
	await page.getByRole('button', { name: SSO }).click();
	return callback;
}

const oidcCookie = async (context: BrowserContext) =>
	(await context.cookies()).find((c) => c.name === OIDC_COOKIE);

async function toConfirmation(page: Page, path = INVITE) {
	await page.goto(path);
	const callback = await continueWithSso(page);
	expect(callback.status()).toBe(200);
	await expect(page.getByRole('button', { name: 'Confirm' })).toBeVisible();
}

async function expectDidNotComplete(page: Page, message: string | null, back: [string, string]) {
	await expect(page.getByRole('heading', { level: 1 })).toHaveText(DID_NOT_COMPLETE);
	if (message === null) await expect(page.getByRole('alert')).toHaveCount(0);
	else await expect(page.getByRole('alert')).toHaveText(message);
	await expect(page.getByRole('link', { name: back[0] })).toHaveAttribute('href', back[1]);
}

const TO_SIGN_IN: [string, string] = ['Go to sign in', '/login'];
const TO_INVITE: [string, string] = ['Go to your invite link', INVITE];

test('oidc offers only SSO, re-authenticates, confirms, and signs the invitee in', async ({
	page,
	context
}) => {
	await setup('oidc');
	await page.goto(INVITE);
	await expect(page.getByRole('heading', { level: 1 })).toHaveText(
		'Finish setting up your account'
	);
	await expect(page.getByText('Continue with single sign-on to finish.')).toBeVisible();
	await expect(page.getByLabel('Username')).toHaveValue(INVITEE);
	await expect(page.getByLabel('Username')).toHaveAttribute('readonly', '');
	await expect(page.getByLabel('New password')).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Set password' })).toHaveCount(0);

	const callback = await continueWithSso(page);
	expect(callback.status()).toBe(200);
	expect(callback.headers().location).toBeUndefined();
	expect(await issuerAuthorizeLog()).toEqual([{ prompt: 'login', maxAge: '1' }]);
	await expect(page.getByRole('heading', { level: 1 })).toHaveText(
		`You are linking bo.sso to ${INVITEE}`
	);
	expect(await sessionCookie(context)).toBeUndefined();
	const pending = await oidcCookie(context);
	expect(pending).toMatchObject({ httpOnly: true, secure: true, sameSite: 'Lax', path: '/auth' });
	expect(pending!.value.length).toBeLessThan(4096);

	await page.getByRole('button', { name: 'Confirm' }).click();
	await expectLanding(page, { welcome: 'Bo' });
	const cookie = await sessionCookie(context);
	expect(cookie).toMatchObject({ httpOnly: true, secure: true, sameSite: 'Strict', path: '/' });
	const days = ((cookie?.expires ?? 0) - Date.now() / 1000) / 86400;
	expect(days).toBeGreaterThan(29.9);
	expect(days).toBeLessThanOrEqual(30);
	expect(await oidcCookie(context)).toBeUndefined();

	await page.goto('/account');
	const current = page
		.getByRole('list', { name: 'Sessions' })
		.getByRole('listitem')
		.filter({ hasText: 'Active now' });
	expect(await current.innerText()).toMatch(/^Chrome on Linux\b/);

	// The identity is held: a later SSO sign-in finds Bo.
	await context.clearCookies();
	await page.goto('/login');
	await continueWithSso(page);
	await expectLanding(page, { welcome: 'Bo' });
});

test('local offers only "Set password", and ?/sso posted by hand fails', async ({ page }) => {
	await setup('local');
	await page.goto(INVITE);
	await expect(page.getByRole('heading', { level: 1 })).toHaveText('Set your password');
	await expect(page.getByRole('button', { name: 'Set password' })).toBeVisible();
	await expect(page.getByRole('button', { name: SSO })).toHaveCount(0);
	const res = await page.request.post(`${INVITE}?/sso`, {
		form: {},
		headers: { origin: new URL(page.url()).origin },
		maxRedirects: 0
	});
	expect(res.status()).toBe(503);
	expect(await res.text()).toContain('Single sign-on is unavailable');
	expect(await issuerAuthorizeLog()).toEqual([]);
});

test('both offers both; the password path still works', async ({ page }) => {
	await setup('both');
	await page.goto(INVITE);
	await expect(page.getByRole('heading', { level: 1 })).toHaveText('Set your password');
	await expect(page.getByRole('button', { name: SSO })).toBeVisible();
	await page.getByLabel('New password').fill(GOOD);
	await page.getByLabel('Confirm password').fill(GOOD);
	await page.getByRole('button', { name: 'Set password' }).click();
	await expectLanding(page, { welcome: 'Bo' });
});

test('both: single sign-on completes the invite too', async ({ page }) => {
	await setup('both');
	await toConfirmation(page);
	await page.getByRole('button', { name: 'Confirm' }).click();
	await expectLanding(page, { welcome: 'Bo' });
});

test('Cancel returns to the invite page, which is still valid', async ({ page, context }) => {
	await setup('oidc');
	await toConfirmation(page);
	await page.getByRole('button', { name: 'Cancel' }).click();
	await expect(page).toHaveURL(new RegExp(`${INVITE}$`));
	await expect(page.getByRole('button', { name: SSO })).toBeVisible();
	expect(await sessionCookie(context)).toBeUndefined();
	expect(await oidcCookie(context)).toBeUndefined();
});

test('a second Confirm with the same cookie does not complete and adds no session', async ({
	page,
	context
}) => {
	await setup('oidc');
	await toConfirmation(page);
	const pending = (await oidcCookie(context))!;
	await page.getByRole('button', { name: 'Confirm' }).click();
	await expectLanding(page, { welcome: 'Bo' });
	const first = await sessionCookie(context);

	await context.addCookies([pending]);
	const again = await page.request.post('/auth/confirm', {
		form: { choice: 'confirm' },
		headers: { origin: new URL(page.url()).origin, accept: 'text/html' },
		maxRedirects: 0
	});
	expect(again.status()).toBe(200);
	const body = await again.text();
	expect(body).toContain(DID_NOT_COMPLETE);
	expect((await sessionCookie(context))?.value).toBe(first?.value);
	expect(await oidcCookie(context)).toBeUndefined();
});

test('Confirm without the pending cookie does not complete', async ({ page, context }) => {
	await setup('oidc');
	await toConfirmation(page);
	await context.clearCookies({ name: OIDC_COOKIE });
	await page.getByRole('button', { name: 'Confirm' }).click();
	await expectDidNotComplete(page, null, TO_SIGN_IN);
	expect(await sessionCookie(context)).toBeUndefined();
});

test.describe('the invite changes between the start and Confirm', () => {
	test('the admin reissues it: no session, and the new link stays usable', async ({
		page,
		context
	}) => {
		await setup('both');
		await toConfirmation(page);
		const admin = await stubAs(ADMIN);
		const reissued = await admin('POST', `/v1/admin/parents/${BO_ID}/reset-link`);
		expect(reissued.status).toBe(200);
		await page.getByRole('button', { name: 'Confirm' }).click();
		await expectDidNotComplete(page, NO_LONGER_VALID, TO_SIGN_IN);
		expect(await sessionCookie(context)).toBeUndefined();

		await page.goto(new URL(reissued.body.invite_url).pathname);
		await expect(page.getByLabel('Username')).toHaveValue(INVITEE);
		await expect(page.getByRole('button', { name: 'Set password' })).toBeVisible();
	});

	test('it is used by password in another browser', async ({ page, context, browser }) => {
		await setup('both');
		await toConfirmation(page);
		const other = await browser.newContext({
			baseURL: UI_ORIGIN,
			extraHTTPHeaders: { 'x-forwarded-proto': 'http', 'x-forwarded-for': '203.0.113.11' }
		});
		const otherPage = await other.newPage();
		await otherPage.goto(INVITE);
		await otherPage.getByLabel('New password').fill(GOOD);
		await otherPage.getByLabel('Confirm password').fill(GOOD);
		await otherPage.getByRole('button', { name: 'Set password' }).click();
		await expectLanding(otherPage, { welcome: 'Bo' });
		await other.close();

		await page.getByRole('button', { name: 'Confirm' }).click();
		await expectDidNotComplete(page, NO_LONGER_VALID, TO_SIGN_IN);
		expect(await sessionCookie(context)).toBeUndefined();
	});

	test('it expires', async ({ page, context }) => {
		await setup('oidc');
		await toConfirmation(page);
		await moveStubClock(72 * 3600 + 1);
		await page.getByRole('button', { name: 'Confirm' }).click();
		await expectDidNotComplete(page, NO_LONGER_VALID, TO_SIGN_IN);
		expect(await sessionCookie(context)).toBeUndefined();
	});

	test('the parent is disabled', async ({ page, context }) => {
		await setup('both');
		await toConfirmation(page);
		const admin = await stubAs(ADMIN);
		expect((await admin('POST', `/v1/admin/parents/${BO_ID}/disable`)).status).toBe(204);
		await page.getByRole('button', { name: 'Confirm' }).click();
		await expectDidNotComplete(page, NO_LONGER_VALID, TO_SIGN_IN);
		expect(await sessionCookie(context)).toBeUndefined();
	});
});

test('a reset token started as an invite stops at the callback', async ({ page, context }) => {
	await setup('oidc');
	await page.goto(`/invite/${TOKENS.reset}`);
	await expect(page.getByRole('alert')).toContainText('This link is no longer valid');
	// The page offers no button for it: post the form by hand, as a crafted page would.
	const callback = page.waitForResponse(isCallback);
	await page.evaluate(() => {
		const form = document.createElement('form');
		form.method = 'POST';
		form.action = '?/sso';
		document.body.append(form);
		form.submit();
	});
	expect((await callback).status()).toBe(200);
	await expectDidNotComplete(page, NO_LONGER_VALID, TO_SIGN_IN);
	await expect(page.getByRole('button', { name: 'Confirm' })).toHaveCount(0);
	expect(await sessionCookie(context)).toBeUndefined();
	expect(await oidcCookie(context)).toBeUndefined();
});

for (const [what, authTime] of [
	['120 s before the start', -120],
	['missing', null]
] as const) {
	test(`an auth_time ${what} does not complete at the callback`, async ({ page, context }) => {
		await setup('oidc');
		await configureIssuer({
			idToken: { authTime: authTime === null ? null : Math.floor(Date.now() / 1000) + authTime }
		});
		await page.goto(INVITE);
		const callback = await continueWithSso(page);
		expect(callback.status()).toBe(200);
		await expectDidNotComplete(page, null, TO_INVITE);
		await expect(page.getByRole('button', { name: 'Confirm' })).toHaveCount(0);
		expect(await sessionCookie(context)).toBeUndefined();
		expect(await oidcCookie(context)).toBeUndefined();

		await page.getByRole('link', { name: 'Go to your invite link' }).click();
		await expect(page.getByLabel('Username')).toHaveValue(INVITEE);
		await expect(page.getByRole('button', { name: SSO })).toBeVisible();
	});
}

test('a 300-character label with markup renders escaped and cut, and Confirm still works', async ({
	page,
	context
}) => {
	const long = `<b>${'a'.repeat(297)}`;
	await setup('oidc', long);
	await toConfirmation(page);
	const cut = [...long].slice(0, 64).join('');
	await expect(page.getByRole('heading', { level: 1 })).toHaveText(
		`You are linking ${cut} to ${INVITEE}`
	);
	await expect(page.locator('h1 b')).toHaveCount(0);
	expect((await oidcCookie(context))!.value.length).toBeLessThan(4096);
	await page.getByRole('button', { name: 'Confirm' }).click();
	await expectLanding(page, { welcome: 'Bo' });
});

test('no preferred_username gives the fallback label', async ({ page }) => {
	await setup('oidc', null);
	await toConfirmation(page);
	await expect(page.getByRole('heading', { level: 1 })).toHaveText(
		`You are linking ${FALLBACK_LABEL} to ${INVITEE}`
	);
});

test('an SSO account linked to someone else is refused at Confirm', async ({ page, context }) => {
	await setup('oidc');
	await linkStubIdentity(ISSUER_URL, BO_SUB, ANA.username);
	await toConfirmation(page);
	await page.getByRole('button', { name: 'Confirm' }).click();
	await expectDidNotComplete(
		page,
		'That single sign-on account is already linked to an account',
		TO_INVITE
	);
	expect(await sessionCookie(context)).toBeUndefined();
});

test('Confirm after the app turned single sign-on off is refused', async ({ page, context }) => {
	await setup('oidc');
	await toConfirmation(page);
	await setAuthMode('local');
	await page.getByRole('button', { name: 'Confirm' }).click();
	await expectDidNotComplete(page, 'Single sign-on is turned off', TO_INVITE);
	expect(await sessionCookie(context)).toBeUndefined();
});

test('the reset page in oidc says the link needs password sign-in, with no form', async ({
	page
}) => {
	await setup('oidc');
	await page.goto(`/reset/${TOKENS.reset}`);
	await expect(page.getByRole('alert')).toHaveText(RESET_OFF);
	await expect(page.getByRole('link', { name: 'Go to sign in' })).toHaveAttribute('href', '/login');
	await expect(page.getByLabel('New password')).toHaveCount(0);
});

for (const path of [`${INVITE}?/password`, `/reset/${TOKENS.reset}`]) {
	test(`a password post to ${path.split('/')[1]} in oidc is turned off`, async ({ page }) => {
		await setup('oidc');
		await page.goto(path.replace('?/password', ''));
		const res = await page.request.post(path, {
			form: { password: GOOD, confirm: GOOD },
			headers: { origin: new URL(page.url()).origin },
			maxRedirects: 0
		});
		expect(res.status()).toBe(403);
		expect(await res.text()).toContain('Password sign-in is turned off');
	});
}

test('GET /auth/confirm has nothing to confirm', async ({ page }) => {
	const res = await page.goto('/auth/confirm');
	expect(res?.status()).toBe(200);
	expect(res?.headers()['cache-control']).toBe('no-store');
	await expect(page.getByRole('heading', { level: 1 })).toHaveText('Nothing to confirm');
	await expect(page.getByRole('link', { name: 'Go to sign in' })).toHaveAttribute('href', '/login');
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('the oidc invite completes', async ({ page }) => {
		await setup('oidc');
		await toConfirmation(page);
		await page.getByRole('button', { name: 'Confirm' }).click();
		await expectLanding(page, { welcome: 'Bo' });
	});
});

for (const colorScheme of ['light', 'dark'] as const) {
	test.describe(`accessibility, ${colorScheme}`, () => {
		test.use({ colorScheme });

		for (const mode of ['oidc', 'both'] as const) {
			test(`the invite page in ${mode}`, async ({ page }) => {
				await setup(mode);
				await page.goto(INVITE);
				await expect(page.getByRole('button', { name: SSO })).toBeVisible();
				await expectAccessible(page);
			});
		}

		test('the reset page in oidc', async ({ page }) => {
			await setup('oidc');
			await page.goto(`/reset/${TOKENS.reset}`);
			await expect(page.getByRole('alert')).toHaveText(RESET_OFF);
			await expectAccessible(page);
		});

		test('the confirmation', async ({ page }) => {
			await setup('oidc');
			await toConfirmation(page);
			await expectAccessible(page);
		});

		test('GET /auth/confirm', async ({ page }) => {
			await page.goto('/auth/confirm');
			await expect(page.getByRole('heading', { level: 1 })).toHaveText('Nothing to confirm');
			await expectAccessible(page);
		});

		test('a confirm refusal', async ({ page }) => {
			await setup('oidc');
			await linkStubIdentity(ISSUER_URL, BO_SUB, ANA.username);
			await toConfirmation(page);
			await page.getByRole('button', { name: 'Confirm' }).click();
			await expect(page.getByRole('heading', { level: 1 })).toHaveText(DID_NOT_COMPLETE);
			await expectAccessible(page);
		});
	});
}
