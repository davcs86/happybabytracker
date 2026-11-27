import { expect, test } from '@playwright/test';
import {
	ADMIN,
	ANA,
	DAN,
	INVITEE,
	STUB_TOKEN,
	STUB_URL,
	TOKENS,
	linkStubIdentity,
	moveStubClock,
	openSession,
	resetStub,
	setAuthMode,
	setScenario
} from './stub-api.js';

// The stub's SSO routes answer as httpapi/{authmode,oidc,session,me,admin}.go do: the same order (decode, mode,
// device_label, iss, sub, then the rest), codes and fields, mirroring oidc_test.go and me_test.go.

test.beforeEach(async () => {
	await resetStub();
});

const ISS = 'https://issuer.example';
const PARENT_TTL_S = 30 * 24 * 3600;
const ADMIN_TTL_S = 12 * 3600;
const REAUTH_WINDOW_S = 300;

type Answer = { status: number; body: unknown; retryAfter: string | null };

async function call(
	method: string,
	path: string,
	body?: unknown,
	session?: string,
	internal = true
): Promise<Answer> {
	const headers: Record<string, string> = { 'content-type': 'application/json' };
	if (internal) headers.authorization = `Bearer ${STUB_TOKEN}`;
	if (session) headers['x-session-token'] = session;
	const res = await fetch(`${STUB_URL}${path}`, {
		method,
		headers,
		body: body === undefined ? undefined : typeof body === 'string' ? body : JSON.stringify(body)
	});
	const text = await res.text();
	return {
		status: res.status,
		body: text ? JSON.parse(text) : null,
		retryAfter: res.headers.get('retry-after')
	};
}

const sb = (a: Answer) => ({ status: a.status, body: a.body });
const refused = (status: number, error: string) => ({ status, body: { error } });
const invalid = (field: string) => ({ status: 422, body: { error: 'invalid', field } });
const NO_CONTENT = { status: 204, body: null };
const UNKNOWN_X = { status: 422, body: { error: 'unknown_field', field: 'x' } };
const pair = (sub: string, extra: Record<string, string> = {}) => ({ iss: ISS, sub, ...extra });

const oidcLogin = (body: unknown) => call('POST', '/v1/session/oidc', body);
const redeemOidc = (token: string, body: unknown) =>
	call('POST', `/v1/tokens/${token}/redeem-oidc`, body);
const reauth = (session: string, password: string, admin = false) =>
	call('POST', admin ? '/v1/admin/me/reauth' : '/v1/me/reauth', { password }, session);
const link = (session: string, sub: string, admin = false, iss = ISS) =>
	call('POST', admin ? '/v1/admin/me/identities' : '/v1/me/identities', { iss, sub }, session);
const getSession = (session: string) => call('GET', '/v1/session', undefined, session);

// A session token from a 200 login answer, its expiry checked against the TTL.
function tokenOf(a: Answer, kind: 'parent' | 'admin'): string {
	expect(a.status).toBe(200);
	const b = a.body as { token: string; kind: string; expires_at: string };
	expect(Object.keys(b).sort()).toEqual(['expires_at', 'kind', 'token']);
	expect(b.kind).toBe(kind);
	const ttl = kind === 'parent' ? PARENT_TTL_S : ADMIN_TTL_S;
	const left = Date.parse(b.expires_at) / 1000 - Date.now() / 1000;
	expect(Math.abs(left - ttl)).toBeLessThan(5);
	return b.token;
}

// A fresh invitee through the admin route; the token is the invite URL's last segment.
async function invite(admin: string, username: string): Promise<{ id: string; token: string }> {
	const a = await call('POST', '/v1/admin/parents', { username, display_name: username }, admin);
	expect(a.status).toBe(201);
	const b = a.body as { parent: { id: string }; invite_url: string };
	return { id: b.parent.id, token: b.invite_url.split('/').pop() ?? '' };
}

const currentLabel = async (session: string) => {
	const a = await call('GET', '/v1/me/sessions', undefined, session);
	const rows = (a.body as { sessions: { current: boolean; device_label: string | null }[] })
		.sessions;
	return rows.find((r) => r.current)?.device_label;
};

test('GET /v1/auth answers each mode, local by default, and needs the internal token', async () => {
	expect(sb(await call('GET', '/v1/auth'))).toEqual({ status: 200, body: { mode: 'local' } });
	for (const mode of ['oidc', 'both', 'local'] as const) {
		await setAuthMode(mode);
		expect(sb(await call('GET', '/v1/auth'))).toEqual({ status: 200, body: { mode } });
	}
	expect((await call('GET', '/v1/auth', undefined, undefined, false)).status).toBe(401);
});

test('POST /v1/session: refused in oidc before device_label, after the body', async () => {
	await setAuthMode('oidc');
	const login = { username: ANA.username, password: ANA.password };
	expect(sb(await call('POST', '/v1/session', login))).toEqual(refused(403, 'mode_refused'));
	const bad = { ...login, device_label: 'bad!' };
	expect(sb(await call('POST', '/v1/session', bad))).toEqual(refused(403, 'mode_refused'));
	expect(sb(await call('POST', '/v1/session', '{"x":1}'))).toEqual(UNKNOWN_X);
	await setScenario('rate_limited');
	expect(sb(await call('POST', '/v1/session', login))).toEqual(refused(403, 'mode_refused'));
	await setScenario('normal');

	await setAuthMode('both');
	expect(sb(await call('POST', '/v1/session', bad))).toEqual(invalid('device_label'));
	tokenOf(await call('POST', '/v1/session', login), 'parent');
});

test('a password redemption refused in oidc leaves the invite and the reset usable', async () => {
	await setAuthMode('oidc');
	const redeem = (token: string, body: unknown = { password: 'invite password 1' }) =>
		call('POST', `/v1/tokens/${token}/redeem`, body);
	expect(sb(await redeem(TOKENS.invite))).toEqual(refused(403, 'mode_refused'));
	expect(sb(await redeem(TOKENS.invite, '{"x":1}'))).toEqual(UNKNOWN_X);
	expect(sb(await redeem(TOKENS.unknown))).toEqual(refused(403, 'mode_refused'));
	expect(sb(await redeem(TOKENS.reset))).toEqual(refused(403, 'mode_refused'));
	await setAuthMode('both');
	tokenOf(await redeem(TOKENS.invite), 'parent');
	tokenOf(await redeem(TOKENS.reset, { password: 'reset password 12' }), 'parent');
});

test('POST /v1/session/oidc by mode: parent and admin sessions; unknown and disabled alike', async () => {
	await linkStubIdentity(ISS, 'sub-ana', ANA.username);
	await linkStubIdentity(ISS, 'sub-admin', ADMIN.username);
	await linkStubIdentity(ISS, 'sub-dan', DAN.username);
	expect(sb(await oidcLogin(pair('sub-ana')))).toEqual(refused(403, 'mode_refused'));

	for (const mode of ['oidc', 'both'] as const) {
		await setAuthMode(mode);
		const parent = tokenOf(await oidcLogin(pair('sub-ana')), 'parent');
		const ps = await getSession(parent);
		expect(ps.status).toBe(200);
		expect((ps.body as { parent: { id: string } }).parent.id).toBe('p-ana');
		const admin = tokenOf(await oidcLogin(pair('sub-admin')), 'admin');
		const as = await getSession(admin);
		expect(as.status).toBe(200);
		expect((as.body as { kind: string }).kind).toBe('admin');
	}

	const unknown = sb(await oidcLogin(pair('sub-nobody')));
	expect(unknown).toEqual(refused(403, 'not_linked'));
	const dan = tokenOf(await oidcLogin(pair('sub-dan')), 'parent');
	const admin = await openSession(ADMIN.username, ADMIN.password);
	expect((await call('POST', '/v1/admin/parents/p-dan/disable', undefined, admin)).status).toBe(
		204
	);
	expect(sb(await oidcLogin(pair('sub-dan')))).toEqual(unknown);
	expect(sb(await getSession(dan))).toEqual(refused(401, 'no_session'));
});

test('POST /v1/tokens/{token}/redeem-oidc by mode; the token stays usable after a refusal', async () => {
	const admin = await openSession(ADMIN.username, ADMIN.password);
	expect(sb(await redeemOidc(TOKENS.invite, pair('sub-bo')))).toEqual(refused(403, 'mode_refused'));
	const local = await call('POST', `/v1/tokens/${TOKENS.invite}/redeem`, {
		password: 'invite password 1'
	});
	tokenOf(local, 'parent');

	for (const mode of ['oidc', 'both'] as const) {
		await setAuthMode(mode);
		const bo = await invite(admin, `bo-${mode}`);
		const token = tokenOf(await redeemOidc(bo.token, pair(`sub-${mode}`)), 'parent');
		expect((await getSession(token)).status).toBe(200);
		expect(sb(await redeemOidc(bo.token, pair('sub-again')))).toEqual(refused(404, 'not_found'));
		const cy = await invite(admin, `cy-${mode}`);
		expect(sb(await redeemOidc(cy.token, pair(`sub-${mode}`)))).toEqual(
			refused(409, 'already_linked')
		);
		tokenOf(await redeemOidc(cy.token, pair(`sub-cy-${mode}`)), 'parent');
	}
	for (const t of [
		TOKENS.unknown,
		TOKENS.reset,
		TOKENS.used,
		TOKENS.expired,
		TOKENS.disabledParent
	]) {
		expect(sb(await redeemOidc(t, pair('sub-z')))).toEqual(refused(404, 'not_found'));
	}
});

test('redeem-oidc ends the sessions the parent held before', async () => {
	await setAuthMode('both');
	await linkStubIdentity('https://other.example', 'sub-before', INVITEE);
	const before = tokenOf(
		await oidcLogin({ iss: 'https://other.example', sub: 'sub-before' }),
		'parent'
	);
	expect((await getSession(before)).status).toBe(200);
	const ana = await openSession(ANA.username, ANA.password);
	const token = tokenOf(await redeemOidc(TOKENS.invite, pair('sub-bo')), 'parent');
	expect(sb(await getSession(before))).toEqual(refused(401, 'no_session'));
	expect((await getSession(token)).status).toBe(200);
	expect((await getSession(ana)).status).toBe(200);
});

test('validation of both SSO session routes', async () => {
	await setAuthMode('oidc');
	const prefix = 'https://issuer.example/';
	const long = prefix + 'a'.repeat(512 - prefix.length + 1);
	const cases: [string, Record<string, string>][] = [
		['iss', { iss: '', sub: 's' }],
		['iss', { sub: 's' }],
		['iss', { iss: 'http://issuer.example', sub: 's' }],
		['iss', { iss: long, sub: 's' }],
		['iss', { iss: 'local', sub: 's' }],
		['iss', { iss: 'https://', sub: 's' }],
		['sub', { iss: ISS, sub: '' }],
		['sub', { iss: ISS, sub: 's'.repeat(256) }],
		['device_label', { iss: ISS, sub: 's', device_label: 'bad!' }],
		['device_label', { iss: ISS, sub: 's', device_label: 'a'.repeat(49) }]
	];
	for (const send of [oidcLogin, (b: unknown) => redeemOidc(TOKENS.invite, b)]) {
		for (const [field, body] of cases) expect(sb(await send(body))).toEqual(invalid(field));
	}
	const edge = prefix + 'a'.repeat(512 - prefix.length);
	expect(sb(await oidcLogin({ iss: edge, sub: 's'.repeat(255) }))).toEqual(
		refused(403, 'not_linked')
	);
	const atLimits = { iss: edge, sub: 's'.repeat(255), device_label: 'a'.repeat(48) };
	tokenOf(await redeemOidc(TOKENS.invite, atLimits), 'parent');
});

test('validation order: body, mode, device_label, iss, sub; the token survives the refusals', async () => {
	const bad = { iss: 'http://x', sub: '', device_label: 'bad!' };
	const goodLabel = { ...bad, device_label: 'Firefox on Mac' };
	const goodIss = { ...goodLabel, iss: ISS };
	for (const send of [oidcLogin, (b: unknown) => redeemOidc(TOKENS.invite, b)]) {
		await setAuthMode('local');
		expect(sb(await send(bad))).toEqual(refused(403, 'mode_refused'));
		expect(sb(await send('{"x":1}'))).toEqual(UNKNOWN_X);
		await setAuthMode('oidc');
		expect(sb(await send(bad))).toEqual(invalid('device_label'));
		expect(sb(await send(goodLabel))).toEqual(invalid('iss'));
		expect(sb(await send(goodIss))).toEqual(invalid('sub'));
	}
	await setAuthMode('oidc');
	tokenOf(await redeemOidc(TOKENS.invite, pair('sub-bo')), 'parent');
});

test('both SSO session routes store the device label, and none without one', async () => {
	await setAuthMode('both');
	const body = pair('sub-bo', { device_label: 'Firefox on Mac' });
	const redeemed = tokenOf(await redeemOidc(TOKENS.invite, body), 'parent');
	expect(await currentLabel(redeemed)).toBe('Firefox on Mac');
	const labelled = tokenOf(await oidcLogin(body), 'parent');
	expect(await currentLabel(labelled)).toBe('Firefox on Mac');
	const plain = tokenOf(await oidcLogin(pair('sub-bo')), 'parent');
	expect(await currentLabel(plain)).toBeNull();

	const admin = await openSession(ADMIN.username, ADMIN.password);
	const bob = await invite(admin, 'bob');
	const unlabelled = tokenOf(await redeemOidc(bob.token, pair('sub-bob')), 'parent');
	expect(await currentLabel(unlabelled)).toBeNull();
});

test('an admin session from SSO is refused on kid routes', async () => {
	await setAuthMode('oidc');
	await linkStubIdentity(ISS, 'sub-admin', ADMIN.username);
	const admin = tokenOf(await oidcLogin(pair('sub-admin')), 'admin');
	for (const path of ['/v1/kids', '/v1/kids/k-mia']) {
		expect((await call('GET', path, undefined, admin)).status).toBe(403);
	}
});

test('re-auth by mode: refused in local and oidc, 204 in both', async () => {
	const parent = await openSession(ANA.username, ANA.password);
	const admin = await openSession(ADMIN.username, ADMIN.password);
	for (const mode of ['local', 'oidc'] as const) {
		await setAuthMode(mode);
		expect(sb(await reauth(parent, ANA.password))).toEqual(refused(403, 'mode_refused'));
		expect(sb(await reauth(admin, ADMIN.password, true))).toEqual(refused(403, 'mode_refused'));
		expect(sb(await call('POST', '/v1/me/reauth', '{"x":1}', parent))).toEqual(UNKNOWN_X);
	}
	await setAuthMode('both');
	expect(sb(await reauth(parent, ANA.password))).toEqual(NO_CONTENT);
	expect(sb(await reauth(admin, ADMIN.password, true))).toEqual(NO_CONTENT);
});

test('re-auth: a wrong password, and a parent without one, are 403 wrong_password', async () => {
	await setAuthMode('both');
	const parent = await openSession(ANA.username, ANA.password);
	const admin = await openSession(ADMIN.username, ADMIN.password);
	expect(sb(await reauth(parent, 'not the password'))).toEqual(refused(403, 'wrong_password'));
	expect(sb(await reauth(admin, ANA.password, true))).toEqual(refused(403, 'wrong_password'));
	await linkStubIdentity('https://other.example', 'sub-bo', INVITEE);
	const bo = tokenOf(await oidcLogin({ iss: 'https://other.example', sub: 'sub-bo' }), 'parent');
	expect(sb(await reauth(bo, ''))).toEqual(refused(403, 'wrong_password'));
	expect(sb(await link(bo, 'sub-bo'))).toEqual(refused(403, 'reauth_required'));
});

test('re-auth and the password change count against the limiter (the rate_limited scenario)', async () => {
	await setAuthMode('both');
	const parent = await openSession(ANA.username, ANA.password);
	const admin = await openSession(ADMIN.username, ADMIN.password);
	await setScenario('rate_limited');
	for (const a of [
		await reauth(parent, ANA.password),
		await reauth(admin, ADMIN.password, true),
		await call(
			'POST',
			'/v1/me/password',
			{ current: ANA.password, new: 'a brand new password' },
			parent
		)
	]) {
		expect(sb(a)).toEqual(refused(429, 'rate_limited'));
		expect(a.retryAfter).toBe('60');
	}
	expect(sb(await call('POST', '/v1/me/password', '{"x":1}', parent))).toEqual(UNKNOWN_X);
	await setAuthMode('local');
	expect(sb(await reauth(parent, ANA.password))).toEqual(refused(403, 'mode_refused'));
});

test('link window: 204 within five minutes of the re-auth, reauth_required after or without', async () => {
	await setAuthMode('both');
	const never = await openSession(ANA.username, ANA.password);
	const admin = await openSession(ADMIN.username, ADMIN.password);
	expect(sb(await link(never, 'sub-never'))).toEqual(refused(403, 'reauth_required'));
	expect(sb(await link(admin, 'sub-never', true))).toEqual(refused(403, 'reauth_required'));

	const within = await openSession(ANA.username, ANA.password);
	const late = await openSession(DAN.username, DAN.password);
	expect(sb(await reauth(within, ANA.password))).toEqual(NO_CONTENT);
	expect(sb(await reauth(late, DAN.password))).toEqual(NO_CONTENT);
	expect(sb(await reauth(admin, ADMIN.password, true))).toEqual(NO_CONTENT);
	await moveStubClock(REAUTH_WINDOW_S - 10);
	expect(sb(await link(within, 'sub-ana'))).toEqual(NO_CONTENT);
	expect(sb(await link(admin, 'sub-admin', true))).toEqual(NO_CONTENT);
	await moveStubClock(REAUTH_WINDOW_S + 1);
	expect(sb(await link(late, 'sub-dan'))).toEqual(refused(403, 'reauth_required'));
	expect(sb(await link(within, 'sub-ana-2', false, 'https://other.example'))).toEqual(
		refused(403, 'reauth_required')
	);
});

test('a link keeps the re-auth window for a second issuer', async () => {
	await setAuthMode('both');
	const ana = await openSession(ANA.username, ANA.password);
	expect(sb(await reauth(ana, ANA.password))).toEqual(NO_CONTENT);
	expect(sb(await link(ana, 'sub-ana'))).toEqual(NO_CONTENT);
	expect(sb(await link(ana, 'sub-ana', false, 'https://other.example'))).toEqual(NO_CONTENT);
});

test('link: already_linked when anyone holds the pair or the caller has the issuer', async () => {
	await setAuthMode('both');
	await linkStubIdentity(ISS, 'sub-admin', ADMIN.username);
	await linkStubIdentity(ISS, 'sub-held', DAN.username);
	const ana = await openSession(ANA.username, ANA.password);
	expect(sb(await reauth(ana, ANA.password))).toEqual(NO_CONTENT);
	expect(sb(await link(ana, 'sub-held'))).toEqual(refused(409, 'already_linked'));
	expect(sb(await link(ana, 'sub-admin'))).toEqual(refused(409, 'already_linked'));
	expect(sb(await link(ana, 'sub-ana'))).toEqual(NO_CONTENT);
	expect(sb(await link(ana, 'sub-ana-2'))).toEqual(refused(409, 'already_linked'));

	const admin = await openSession(ADMIN.username, ADMIN.password);
	expect(sb(await reauth(admin, ADMIN.password, true))).toEqual(NO_CONTENT);
	expect(sb(await link(admin, 'sub-held', true))).toEqual(refused(409, 'already_linked'));
	expect(sb(await link(admin, 'sub-admin-2', true))).toEqual(refused(409, 'already_linked'));
});

test('a re-auth is per session', async () => {
	await setAuthMode('both');
	const a = await openSession(ANA.username, ANA.password);
	const b = await openSession(ANA.username, ANA.password);
	expect(sb(await reauth(a, ANA.password))).toEqual(NO_CONTENT);
	expect(sb(await link(b, 'sub-ana'))).toEqual(refused(403, 'reauth_required'));
	expect(sb(await link(a, 'sub-ana'))).toEqual(NO_CONTENT);
});

test('link: the mode, then the window, then iss and sub; no device_label field', async () => {
	await setAuthMode('both');
	const ana = await openSession(ANA.username, ANA.password);
	expect(sb(await reauth(ana, ANA.password))).toEqual(NO_CONTENT);
	const badPair = { iss: 'http://x', sub: '' };
	for (const mode of ['local', 'oidc'] as const) {
		await setAuthMode(mode);
		expect(sb(await link(ana, 'sub-ana'))).toEqual(refused(403, 'mode_refused'));
		expect(sb(await call('POST', '/v1/me/identities', badPair, ana))).toEqual(
			refused(403, 'mode_refused')
		);
	}
	await setAuthMode('both');
	const never = await openSession(DAN.username, DAN.password);
	expect(sb(await call('POST', '/v1/me/identities', badPair, never))).toEqual(
		refused(403, 'reauth_required')
	);
	expect(sb(await link(ana, 's', false, 'local'))).toEqual(invalid('iss'));
	expect(sb(await link(ana, 's'.repeat(256)))).toEqual(invalid('sub'));
	const labelled = { iss: ISS, sub: 's', device_label: 'Phone' };
	expect(sb(await call('POST', '/v1/me/identities', labelled, ana))).toEqual({
		status: 422,
		body: { error: 'unknown_field', field: 'device_label' }
	});
});

test('re-auth and link decode the body before the mode', async () => {
	const parent = await openSession(ANA.username, ANA.password);
	const admin = await openSession(ADMIN.username, ADMIN.password);
	const probe = { status: 422, body: { error: 'unknown_field', field: 'matrix_probe' } };
	for (const mode of ['local', 'oidc'] as const) {
		await setAuthMode(mode);
		for (const [path, s] of [
			['/v1/me/reauth', parent],
			['/v1/admin/me/reauth', admin],
			['/v1/me/identities', parent],
			['/v1/admin/me/identities', admin]
		]) {
			expect(sb(await call('POST', path, '{"matrix_probe":true}', s))).toEqual(probe);
		}
	}
});

test('the access class of the re-auth and identity routes', async () => {
	await setAuthMode('both');
	const parent = await openSession(ANA.username, ANA.password);
	const admin = await openSession(ADMIN.username, ADMIN.password);
	for (const path of ['/v1/me/reauth', '/v1/me/identities']) {
		expect(sb(await call('POST', path, {}, admin))).toEqual(refused(403, 'forbidden'));
		expect(sb(await call('POST', path, {}))).toEqual(refused(401, 'no_session'));
	}
	for (const path of [
		'/v1/admin/me/reauth',
		'/v1/admin/me/identities',
		'/v1/admin/parents/p-ana/unlink-sso'
	]) {
		expect(sb(await call('POST', path, {}, parent))).toEqual(refused(403, 'forbidden'));
		expect(sb(await call('POST', path, {}))).toEqual(refused(401, 'no_session'));
	}
});

test('an admin link goes to the admin identities: SSO then opens an admin session', async () => {
	await setAuthMode('both');
	const admin = await openSession(ADMIN.username, ADMIN.password);
	expect(sb(await reauth(admin, ADMIN.password, true))).toEqual(NO_CONTENT);
	expect(sb(await link(admin, 'sub-admin', true))).toEqual(NO_CONTENT);
	tokenOf(await oidcLogin(pair('sub-admin')), 'admin');
});

test('the session views carry sso_linked, and the admin view its admin object', async () => {
	await setAuthMode('both');
	const ana = await openSession(ANA.username, ANA.password);
	const parentLinked = async () =>
		((await getSession(ana)).body as { parent: { sso_linked: boolean } }).parent.sso_linked;
	const view = (await getSession(ana)).body as { parent: Record<string, unknown> };
	expect(Object.keys(view.parent).sort()).toEqual(
		['display_name', 'id', 'password_changed_at', 'sso_linked', 'units', 'username'].sort()
	);
	expect(await parentLinked()).toBe(false);
	expect(sb(await reauth(ana, ANA.password))).toEqual(NO_CONTENT);
	expect(sb(await link(ana, 'sub-ana'))).toEqual(NO_CONTENT);
	expect(await parentLinked()).toBe(true);
	const patched = await call('PATCH', '/v1/me', { display_name: 'Ana' }, ana);
	expect(patched.status).toBe(200);
	expect((patched.body as { sso_linked: boolean }).sso_linked).toBe(true);

	const admin = await openSession(ADMIN.username, ADMIN.password);
	const adminView = async () => (await getSession(admin)).body as Record<string, unknown>;
	const before = await adminView();
	expect(Object.keys(before).sort()).toEqual(['admin', 'expires_at', 'kind']);
	expect(before.admin).toEqual({ username: ADMIN.username, sso_linked: false });
	expect(sb(await reauth(admin, ADMIN.password, true))).toEqual(NO_CONTENT);
	expect(sb(await link(admin, 'sub-admin', true))).toEqual(NO_CONTENT);
	expect((await adminView()).admin).toEqual({ username: ADMIN.username, sso_linked: true });
});

test('the admin unlinks a parent: identity and sessions gone, the password still works', async () => {
	await setAuthMode('both');
	const admin = await openSession(ADMIN.username, ADMIN.password);
	await linkStubIdentity(ISS, 'sub-ana', ANA.username);
	const created = await call(
		'POST',
		'/v1/admin/parents',
		{ username: 'eve', display_name: 'Eve' },
		admin
	);
	expect((created.body as { parent: { sso_linked: boolean } }).parent.sso_linked).toBe(false);
	const listed = async () =>
		(
			(await call('GET', '/v1/admin/parents', undefined, admin)).body as {
				parents: { id: string; sso_linked: boolean }[];
			}
		).parents.find((p) => p.id === 'p-ana')?.sso_linked;
	expect(await listed()).toBe(true);
	const s1 = tokenOf(await oidcLogin(pair('sub-ana')), 'parent');
	const s2 = await openSession(ANA.username, ANA.password);

	expect(sb(await call('POST', '/v1/admin/parents/p-ana/unlink-sso', undefined, admin))).toEqual(
		NO_CONTENT
	);
	for (const s of [s1, s2]) expect(sb(await getSession(s))).toEqual(refused(401, 'no_session'));
	expect(await listed()).toBe(false);
	tokenOf(
		await call('POST', '/v1/session', { username: ANA.username, password: ANA.password }),
		'parent'
	);
	expect(sb(await oidcLogin(pair('sub-ana')))).toEqual(refused(403, 'not_linked'));
	expect(sb(await call('POST', '/v1/admin/parents/p-dan/unlink-sso', undefined, admin))).toEqual(
		NO_CONTENT
	);
});

test('unlinking an unknown parent is the reset-link 404', async () => {
	const admin = await openSession(ADMIN.username, ADMIN.password);
	const unlink = sb(
		await call('POST', '/v1/admin/parents/no-such-parent/unlink-sso', undefined, admin)
	);
	expect(unlink).toEqual(refused(404, 'not_found'));
	expect(
		sb(await call('POST', '/v1/admin/parents/no-such-parent/reset-link', undefined, admin))
	).toEqual(unlink);
});

test('/__reset restores local and clears the identities', async () => {
	await setAuthMode('both');
	await linkStubIdentity(ISS, 'sub-ana', ANA.username);
	tokenOf(await oidcLogin(pair('sub-ana')), 'parent');
	await resetStub();
	expect(sb(await call('GET', '/v1/auth'))).toEqual({ status: 200, body: { mode: 'local' } });
	await setAuthMode('both');
	expect(sb(await oidcLogin(pair('sub-ana')))).toEqual(refused(403, 'not_linked'));
});
