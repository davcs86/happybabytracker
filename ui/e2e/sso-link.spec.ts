import {
	expect,
	test,
	type Browser,
	type BrowserContext,
	type Page,
	type Response
} from '@playwright/test';
import {
	expectAccessible,
	expectFieldError,
	expectLanding,
	sessionCookie,
	signIn,
	signInWithForm,
	signOut
} from './helpers.js';
import {
	ADMIN,
	ANA,
	CLEO,
	ISSUER_URL,
	MIA,
	UI_ORIGIN,
	linkStubIdentity,
	moveStubClock,
	receivedRequests,
	resetStub,
	setAuthMode,
	setScenario,
	stubAs
} from './stub-api.js';
import { configureIssuer, issuerAuthorizeLog, resetIssuer } from './stub-issuer.js';

const OIDC_COOKIE = '__Secure-hbt_oidc';
const SSO = 'Continue with single sign-on';
const LINK = 'Link single sign-on';
const LINKED = 'Single sign-on is linked.';
const DID_NOT_COMPLETE = 'Single sign-on did not complete';
const STUB_USER = 'stub-user';
const ANA_SUB = 'stub-subject-ana-0001';
const UNLINKED = 'Single sign-on unlinked. Their sessions and notification devices were removed.';
const TOO_MANY = 'Too many attempts, try again in a minute';
const TO_PROFILE: [string, string] = ['Go to My profile', '/account'];

test.beforeEach(async () => {
	await resetStub();
	await resetIssuer();
	await setAuthMode('both');
});

const isCallback = (r: Response) => new URL(r.url()).pathname === '/auth/callback';
const ssoSection = (page: Page) => page.getByRole('region', { name: 'Single sign-on' });
const oidcCookie = async (context: BrowserContext) =>
	(await context.cookies()).find((c) => c.name === OIDC_COOKIE);

// The section's password, then the IdP round trip; resolves on the callback's answer.
async function startLink(page: Page, password: string): Promise<Response> {
	const section = ssoSection(page);
	await section.getByLabel('Password', { exact: true }).fill(password);
	const callback = page.waitForResponse(isCallback);
	await section.getByRole('button', { name: LINK }).click();
	return callback;
}

async function toConfirmation(page: Page, who = ANA, path = '/account', label = STUB_USER) {
	await signIn(page, who);
	await page.goto(path);
	const callback = await startLink(page, who.password);
	expect(callback.status()).toBe(200);
	await expect(page.getByRole('heading', { level: 1 })).toHaveText(
		`You are linking ${label} to ${who.username}`
	);
}

async function expectDidNotComplete(page: Page, message: string | null, back = TO_PROFILE) {
	await expect(page.getByRole('heading', { level: 1 })).toHaveText(DID_NOT_COMPLETE);
	if (message === null) await expect(page.getByRole('alert')).toHaveCount(0);
	else await expect(page.getByRole('alert')).toHaveText(message);
	await expect(page.getByRole('link', { name: back[0] })).toHaveAttribute('href', back[1]);
	await expect(page.getByRole('button', { name: 'Confirm' })).toHaveCount(0);
}

const linked = async (who: { username: string; password: string }) => {
	const r = await (await stubAs(who))('GET', '/v1/session');
	return Boolean(r.body.parent?.sso_linked ?? r.body.admin?.sso_linked);
};

const identityPosts = async () =>
	(await receivedRequests()).filter(
		(r) => r.method === 'POST' && /^\/v1(\/admin)?\/me\/identities$/.test(r.path)
	);

// The SSO login of a signed-out browser, against whichever user the issuer has now.
async function ssoLogin(page: Page) {
	await page.context().clearCookies();
	await page.goto('/login');
	const callback = page.waitForResponse(isCallback);
	await page.getByRole('button', { name: SSO }).click();
	return callback;
}

async function otherBrowser(browser: Browser) {
	return browser.newContext({
		baseURL: UI_ORIGIN,
		ignoreHTTPSErrors: true,
		extraHTTPHeaders: { 'x-forwarded-proto': 'http', 'x-forwarded-for': '203.0.113.11' }
	});
}

test('a parent with a password links, and then signs in by single sign-on', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto('/account');
	await expect(ssoSection(page).getByRole('button', { name: LINK })).toBeVisible();
	const callback = await startLink(page, ANA.password);
	expect(callback.status()).toBe(200);
	expect(await issuerAuthorizeLog()).toEqual([{ prompt: 'login', maxAge: '1' }]);
	await expect(page.getByRole('heading', { level: 1 })).toHaveText(
		`You are linking ${STUB_USER} to ${ANA.username}`
	);

	await page.getByRole('button', { name: 'Confirm' }).click();
	await expect(page).toHaveURL(/\/account$/);
	await expect(ssoSection(page)).toContainText(LINKED);
	await expect(ssoSection(page).getByRole('button', { name: LINK })).toHaveCount(0);
	expect(await linked(ANA)).toBe(true);

	await signOut(page);
	await expect(page).toHaveURL(/\/login$/);
	await ssoLogin(page);
	await expectLanding(page, { kid: MIA });
});

test('with another IdP session live, the re-authenticated account is the one linked', async ({
	page
}) => {
	await configureIssuer({
		user: { sub: 'stub-subject-david-0001', preferredUsername: 'david' },
		reauthUser: { sub: 'stub-subject-natalia-0001', preferredUsername: 'natalia' }
	});
	await toConfirmation(page, ANA, '/account', 'natalia');
	await page.getByRole('button', { name: 'Confirm' }).click();
	await expect(ssoSection(page)).toContainText(LINKED);

	await ssoLogin(page);
	await expect(page.getByRole('heading', { level: 1 })).toHaveText(DID_NOT_COMPLETE);
	await configureIssuer({ user: { sub: 'stub-subject-natalia-0001' } });
	await ssoLogin(page);
	await expectLanding(page, { kid: MIA });
});

test.describe('the callback without the session cookie', () => {
	// A redirect is not routed: the start is fetched by hand (route.fetch keeps __Secure-hbt_oidc in the
	// context), and so is the issuer's redirect, so the callback URL is held unvisited.
	async function callbackWithoutSession(page: Page, context: BrowserContext) {
		await signIn(page, ANA);
		await page.goto('/account');
		let held = '';
		await page.route(
			(url) => url.pathname === '/account/link-sso',
			async (route) => {
				const start = await route.fetch({ maxRedirects: 0 });
				const authorize = await page.request.get(start.headers().location, { maxRedirects: 0 });
				held = authorize.headers().location;
				await route.fulfill({ status: 200, contentType: 'text/html', body: '<!doctype html>' });
			}
		);
		await ssoSection(page).getByLabel('Password', { exact: true }).fill(ANA.password);
		await ssoSection(page).getByRole('button', { name: LINK }).click();
		await expect.poll(() => held).toMatch(/\/auth\/callback\?/);
		await page.unrouteAll();
		const saved = (await sessionCookie(context))!;
		await context.clearCookies({ name: '__Secure-hbt_session' });
		const res = await page.goto(held);
		expect(res?.status()).toBe(200);
		await expect(page.getByRole('heading', { level: 1 })).toHaveText(
			`You are linking ${STUB_USER} to ${ANA.username}`
		);
		return saved;
	}

	test('Confirm with the session restored links', async ({ page, context }) => {
		const saved = await callbackWithoutSession(page, context);
		await context.addCookies([saved]);
		await page.getByRole('button', { name: 'Confirm' }).click();
		await expect(ssoSection(page)).toContainText(LINKED);
		expect(await linked(ANA)).toBe(true);
	});

	test('Confirm without a session does not complete', async ({ page, context }) => {
		await callbackWithoutSession(page, context);
		await page.getByRole('button', { name: 'Confirm' }).click();
		await expectDidNotComplete(page, null);
		expect(await linked(ANA)).toBe(false);
		expect(await identityPosts()).toEqual([]);
	});
});

for (const [what, authTime] of [
	['an auth_time 120 s before the start', -120],
	['no auth_time', null]
] as const) {
	test(`re-authentication: ${what} does not complete`, async ({ page }) => {
		await signIn(page, ANA);
		await page.goto('/account');
		await configureIssuer({
			idToken: {
				authTime: authTime === null ? null : Math.floor(Date.now() / 1000) + authTime
			}
		});
		const callback = await startLink(page, ANA.password);
		expect(callback.status()).toBe(200);
		await expectDidNotComplete(page, null);
		await page.goto('/account');
		await expect(ssoSection(page).getByRole('button', { name: LINK })).toBeVisible();
		expect(await linked(ANA)).toBe(false);
	});
}

test('a wrong password is refused under the field, before the IdP', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto('/account');
	await ssoSection(page).getByLabel('Password', { exact: true }).fill('not the password');
	await ssoSection(page).getByRole('button', { name: LINK }).click();
	await expect(page).toHaveURL(/\/account\/link-sso$/);
	await expect(page.getByRole('heading', { level: 1 })).toHaveText(LINK);
	await expectFieldError(
		ssoSection(page).getByLabel('Password', { exact: true }),
		'The password is incorrect'
	);
	expect(await issuerAuthorizeLog()).toEqual([]);
	expect(await oidcCookie(page.context())).toBeUndefined();
});

test('the re-auth limiter answers "Too many attempts"', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto('/account');
	await setScenario('rate_limited');
	await ssoSection(page).getByLabel('Password', { exact: true }).fill(ANA.password);
	await ssoSection(page).getByRole('button', { name: LINK }).click();
	await expect(ssoSection(page).getByRole('alert')).toHaveText(TOO_MANY);
	expect(await issuerAuthorizeLog()).toEqual([]);
});

test.describe('the session changes between the start and Confirm', () => {
	for (const who of [CLEO, ADMIN]) {
		test(`signed in as ${who.username} in another tab: nothing is linked`, async ({
			page,
			context
		}) => {
			await toConfirmation(page);
			const tab = await context.newPage();
			await tab.goto('/account');
			await signOut(tab);
			await expect(tab).toHaveURL(/\/login$/);
			await signInWithForm(tab, who);
			await tab.close();

			await page.getByRole('button', { name: 'Confirm' }).click();
			await expectDidNotComplete(page, null);
			expect(await identityPosts()).toEqual([]);
			expect(await linked(ANA)).toBe(false);
			expect(await linked(who)).toBe(false);
		});
	}
});

test('a second Confirm links once', async ({ page }) => {
	await toConfirmation(page);
	await page.getByRole('button', { name: 'Confirm' }).click();
	await expect(ssoSection(page)).toContainText(LINKED);

	const again = await page.request.post('/auth/confirm', {
		form: { choice: 'confirm' },
		headers: { origin: new URL(page.url()).origin, accept: 'text/html' },
		maxRedirects: 0
	});
	expect(again.status()).toBe(200);
	expect(await again.text()).toContain(DID_NOT_COMPLETE);
	expect(await identityPosts()).toHaveLength(1);
});

test('Cancel returns to My profile and links nothing', async ({ page, context }) => {
	await toConfirmation(page);
	await page.getByRole('button', { name: 'Cancel' }).click();
	await expect(page).toHaveURL(/\/account$/);
	await expect(ssoSection(page).getByRole('button', { name: LINK })).toBeVisible();
	expect(await oidcCookie(context)).toBeUndefined();
	expect(await linked(ANA)).toBe(false);
});

test('a re-auth older than 5 minutes asks for the password again', async ({ page }) => {
	await toConfirmation(page);
	await moveStubClock(5 * 60 + 1);
	await page.getByRole('button', { name: 'Confirm' }).click();
	await expectDidNotComplete(page, 'Confirm your password again, then link');
	expect(await linked(ANA)).toBe(false);
});

for (const mode of ['local', 'oidc'] as const) {
	test(`${mode}: no Single sign-on section, and a hand-made link post is turned off`, async ({
		page
	}) => {
		await signIn(page, ANA);
		await setAuthMode(mode);
		await page.goto('/account');
		await expect(page.getByRole('heading', { name: 'Your profile', level: 1 })).toBeVisible();
		await expect(ssoSection(page)).toHaveCount(0);
		const res = await page.request.post('/account/link-sso', {
			form: { password: ANA.password },
			headers: { origin: new URL(page.url()).origin, accept: 'text/html' },
			maxRedirects: 0
		});
		expect(res.status()).toBe(403);
		expect(await res.text()).toContain('Linking is turned off');
		expect(await issuerAuthorizeLog()).toEqual([]);
	});

	test(`${mode}: the link page offers no form`, async ({ page }) => {
		await signIn(page, ANA);
		await setAuthMode(mode);
		await page.goto('/account/link-sso');
		await expect(ssoSection(page).getByText('Linking is turned off')).toBeVisible();
		await expect(page.getByLabel('Password', { exact: true })).toHaveCount(0);
		await expect(page.getByRole('button', { name: 'Link single sign-on' })).toHaveCount(0);
		await expectAccessible(page);
	});
}

test('the admin opens Account from the nav, links, then signs in by single sign-on', async ({
	page
}) => {
	await signIn(page, ADMIN);
	await expect(page).toHaveURL(/\/admin\/parents$/);
	const wide = page.viewportSize()!.width >= 768;
	const nav = page.getByRole('navigation', { name: 'Primary' });
	const entry = wide
		? nav.getByRole('link', { name: 'Account', exact: true })
		: page.locator('header').getByRole('link', { name: 'Account', exact: true });
	await expect(entry).toBeVisible();
	await entry.click();
	await expect(page).toHaveURL(/\/admin\/account$/);
	await expect(page.getByRole('heading', { name: 'Account', level: 1 })).toBeVisible();
	await expect(entry).toHaveAttribute('aria-current', 'page');
	const details = page.getByRole('region', { name: 'Details' });
	await expect(details.getByLabel('Username')).toHaveValue(ADMIN.username);
	await expect(details.getByLabel('Username')).toHaveAttribute('readonly', '');

	const callback = await startLink(page, ADMIN.password);
	expect(callback.status()).toBe(200);
	await expect(page.getByRole('heading', { level: 1 })).toHaveText(
		`You are linking ${STUB_USER} to ${ADMIN.username}`
	);
	await page.getByRole('button', { name: 'Confirm' }).click();
	await expect(page).toHaveURL(/\/admin\/account$/);
	await expect(ssoSection(page)).toContainText(LINKED);

	await ssoLogin(page);
	await expect(page).toHaveURL(/\/admin\/parents$/);
});

test('an admin link that does not complete goes back to Account', async ({ page }) => {
	await signIn(page, ADMIN);
	await page.goto('/admin/account');
	await configureIssuer({ idToken: { authTime: null } });
	await startLink(page, ADMIN.password);
	await expectDidNotComplete(page, null, ['Go to Account', '/admin/account']);
});

test('the admin unlinks a parent, whose session ends', async ({ page, browser }) => {
	await linkStubIdentity(ISSUER_URL, ANA_SUB, ANA.username);
	const other = await otherBrowser(browser);
	const ana = await other.newPage();
	await signIn(ana, ANA);

	await signIn(page, ADMIN);
	const row = page.getByRole('listitem').filter({ has: page.getByText('ana', { exact: true }) });
	await expect(row.getByText('Single sign-on', { exact: true })).toBeVisible();
	await row.locator('summary', { hasText: 'Unlink single sign-on' }).click();
	await expect(row.getByRole('listitem')).toHaveText([
		'Ana is signed out everywhere and their notification devices are removed.',
		'Their password still works, if they have one.'
	]);
	await row.getByRole('button', { name: /^Unlink single sign-on/ }).click();
	await expect(page.getByRole('status')).toHaveText(UNLINKED);
	await expect(row.getByText('Single sign-on', { exact: true })).toHaveCount(0);
	await expect(row.locator('summary', { hasText: 'Unlink single sign-on' })).toHaveCount(0);

	await ana.goto('/account');
	await expect(ana).toHaveURL(/\/login/);
	await other.close();
});

test('the password change answers "Too many attempts" when limited', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto('/account');
	await setScenario('rate_limited');
	await page.getByLabel('Current password').fill(ANA.password);
	await page.getByLabel('New password', { exact: true }).fill('a long enough password');
	await page.getByLabel('Confirm new password').fill('a long enough password');
	await page.getByRole('button', { name: 'Change password' }).click();
	await expect(
		page.getByRole('region', { name: 'Password' }).getByRole('alert').filter({ hasText: TOO_MANY })
	).toBeVisible();
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('a parent links', async ({ page }) => {
		await toConfirmation(page);
		await page.getByRole('button', { name: 'Confirm' }).click();
		await expect(ssoSection(page)).toContainText(LINKED);
	});

	test('a wrong password shows on the link page', async ({ page }) => {
		await signIn(page, ANA);
		await page.goto('/account');
		await ssoSection(page).getByLabel('Password', { exact: true }).fill('not the password');
		await ssoSection(page).getByRole('button', { name: LINK }).click();
		await expectFieldError(
			ssoSection(page).getByLabel('Password', { exact: true }),
			'The password is incorrect'
		);
	});

	test('the admin unlinks a parent', async ({ page }) => {
		await linkStubIdentity(ISSUER_URL, ANA_SUB, ANA.username);
		await signIn(page, ADMIN);
		const row = page.getByRole('listitem').filter({ has: page.getByText('ana', { exact: true }) });
		await row.locator('summary', { hasText: 'Unlink single sign-on' }).click();
		await row.getByRole('button', { name: /^Unlink single sign-on/ }).click();
		await expect(page.getByRole('status')).toHaveText(UNLINKED);
	});
});

for (const colorScheme of ['light', 'dark'] as const) {
	test.describe(`accessibility, ${colorScheme}`, () => {
		test.use({ colorScheme });

		test('/account, linkable then linked', async ({ page }) => {
			await signIn(page, ANA);
			await page.goto('/account');
			await expect(ssoSection(page).getByRole('button', { name: LINK })).toBeVisible();
			await expectAccessible(page);
			await linkStubIdentity(ISSUER_URL, ANA_SUB, ANA.username);
			await page.goto('/account');
			await expect(ssoSection(page)).toContainText(LINKED);
			await expectAccessible(page);
		});

		test('/account/link-sso with a refusal', async ({ page }) => {
			await signIn(page, ANA);
			await page.goto('/account');
			await ssoSection(page).getByLabel('Password', { exact: true }).fill('not the password');
			await ssoSection(page).getByRole('button', { name: LINK }).click();
			await expect(page).toHaveURL(/\/account\/link-sso$/);
			await expectAccessible(page);
		});

		test('the link confirmation', async ({ page }) => {
			await toConfirmation(page);
			await expectAccessible(page);
		});

		test('/admin/account and /admin/account/link-sso', async ({ page }) => {
			await signIn(page, ADMIN);
			await page.goto('/admin/account');
			await expect(ssoSection(page).getByRole('button', { name: LINK })).toBeVisible();
			await expectAccessible(page);
			await ssoSection(page).getByLabel('Password', { exact: true }).fill('not the password');
			await ssoSection(page).getByRole('button', { name: LINK }).click();
			await expect(page).toHaveURL(/\/admin\/account\/link-sso$/);
			await expect(page.getByRole('heading', { level: 1 })).toHaveText(LINK);
			await expectAccessible(page);
		});

		test('/admin/parents with a linked row and its confirm open', async ({ page }) => {
			await linkStubIdentity(ISSUER_URL, ANA_SUB, ANA.username);
			await signIn(page, ADMIN);
			const row = page
				.getByRole('listitem')
				.filter({ has: page.getByText('ana', { exact: true }) });
			await row.locator('summary', { hasText: 'Unlink single sign-on' }).click();
			await expect(row.getByRole('button', { name: /^Unlink single sign-on/ })).toBeVisible();
			await expectAccessible(page);
		});
	});
}
