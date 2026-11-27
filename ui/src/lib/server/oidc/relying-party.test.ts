import {
	afterAll,
	beforeAll,
	beforeEach,
	describe,
	expect,
	it,
	vi,
	type MockInstance
} from 'vitest';
import { createIssuer, type IssuerBehaviour } from '../../../../e2e/stub-issuer';
import { OIDC_LABEL_MAX_LEN } from '../constants';
import type { OidcEnv } from '../env';
import { logSsoRefusal, type SsoCode } from './log';
import { createRelyingParty } from './relying-party';
import type { OidcPurpose, OidcStart } from './types';

const ISSUER = 'https://issuer.test';
const ORIGIN = 'https://app.test';
const OIDC: OidcEnv = {
	issuer: ISSUER,
	clientId: 'fake-client',
	clientSecret: 'fake-client-secret-000000000000',
	scopes: 'openid profile',
	sealKey: 'fake-seal-key-00000000000000000000'
};
const USER = { sub: 'subject-ana-0001', preferredUsername: 'ana.sso' };

const issuer = createIssuer({
	issuer: ISSUER,
	clientId: OIDC.clientId,
	clientSecret: OIDC.clientSecret
});

// Every value a log line must never hold, gathered over the whole file.
const secrets = new Set<string>([OIDC.clientSecret, USER.sub, USER.preferredUsername]);
const spies: MockInstance[] = [];
const calls: string[] = [];

type Fetch = typeof fetch;
const toIssuer: Fetch = (input, init) => issuer.handle(new Request(input, init));

function recording(): { fetch: Fetch; urls: string[] } {
	const urls: string[] = [];
	return {
		urls,
		fetch: (input, init) => {
			urls.push(new Request(input, init).url);
			return toIssuer(input, init);
		}
	};
}

const nowS = () => Math.floor(Date.now() / 1000);

// One flow: start, the browser's visit to the issuer, and the callback URL it lands on.
async function begin(
	rp: ReturnType<typeof createRelyingParty>,
	purpose: OidcPurpose,
	startedAt = nowS()
): Promise<{ callback: URL; txn: OidcStart }> {
	const s = await rp.start(purpose);
	if (!s.ok) throw new Error('start failed');
	const res = await issuer.handle(new Request(s.url));
	const location = res.headers.get('location') ?? '';
	const callback = new URL(location);
	for (const v of [s.state, s.nonce, s.verifier, callback.searchParams.get('code') ?? '']) {
		if (v) secrets.add(v);
	}
	const txn: OidcStart = {
		stage: 'start',
		purpose,
		state: s.state,
		nonce: s.nonce,
		verifier: s.verifier,
		startedAt,
		next: null,
		binding: null,
		username: null,
		kind: null,
		deviceLabel: null,
		exp: nowS() + 600
	};
	return { callback, txn };
}

async function flow(
	behaviour: Partial<IssuerBehaviour>,
	purpose: OidcPurpose = 'login',
	startedAt?: number
) {
	issuer.configure(behaviour);
	const rp = createRelyingParty(OIDC, ORIGIN, { fetch: toIssuer });
	const { callback, txn } = await begin(rp, purpose, startedAt);
	return rp.finish(callback, txn);
}

beforeAll(() => {
	for (const m of ['log', 'info', 'warn', 'error', 'debug'] as const) {
		spies.push(
			vi.spyOn(console, m).mockImplementation((...args: unknown[]) => {
				calls.push(args.map(String).join(' '));
			})
		);
	}
});

afterAll(() => {
	for (const s of spies) s.mockRestore();
});

beforeEach(() => {
	issuer.reset();
	issuer.configure({ user: USER });
});

describe('createRelyingParty', () => {
	it('start builds the authorization URL', async () => {
		const rp = createRelyingParty(OIDC, ORIGIN, { fetch: toIssuer });
		for (const purpose of ['login', 'invite', 'link'] as const) {
			const s = await rp.start(purpose);
			if (!s.ok) throw new Error('start failed');
			const url = new URL(s.url);
			expect(url.origin + url.pathname).toBe(`${ISSUER}/authorize`);
			const q = Object.fromEntries(url.searchParams);
			const reauth = purpose === 'login' ? {} : { prompt: 'login', max_age: '1' };
			const challenge = Buffer.from(
				await crypto.subtle.digest('SHA-256', new TextEncoder().encode(s.verifier))
			).toString('base64url');
			expect(q).toEqual({
				response_type: 'code',
				client_id: OIDC.clientId,
				redirect_uri: `${ORIGIN}/auth/callback`,
				scope: 'openid profile',
				state: s.state,
				nonce: s.nonce,
				code_challenge: challenge,
				code_challenge_method: 'S256',
				response_mode: 'query',
				...reauth
			});
			expect([...url.searchParams.keys()].length).toBe(Object.keys(q).length);
		}
	});

	it('a full flow returns iss, sub and the label', async () => {
		expect(await flow({})).toEqual({
			ok: true,
			iss: 'https://issuer.test',
			sub: USER.sub,
			label: USER.preferredUsername
		});
	});

	it('bad state', async () => {
		const rec = recording();
		const rp = createRelyingParty(OIDC, ORIGIN, { fetch: rec.fetch });
		const { callback, txn } = await begin(rp, 'login');
		callback.searchParams.set('state', 'not-the-state');
		expect(await rp.finish(callback, txn)).toEqual({ ok: false, code: 'state' });
		expect(rec.urls.some((u) => u.endsWith('/token'))).toBe(false);
	});

	it.each<[string, IssuerBehaviour['idToken']]>([
		['wrong iss', { iss: 'https://other.test' }],
		['wrong aud', { aud: 'other-client' }],
		['expired token', { expOffsetS: -120 }],
		['wrong nonce', { nonce: 'not-the-nonce' }],
		['alg HS256', { alg: 'HS256' }],
		['alg none', { alg: 'none' }],
		['bad signature', { badSignature: true }]
	])('%s', async (_, idToken) => {
		expect(await flow({ idToken })).toEqual({ ok: false, code: 'id_token' });
	});

	describe.each(['link', 'invite'] as const)('auth_time on %s', (purpose) => {
		const S = nowS();
		it('passes at S - 30', async () => {
			const r = await flow({ idToken: { authTime: S - 30 } }, purpose, S);
			expect(r).toMatchObject({ ok: true, sub: USER.sub });
		});
		it('refuses S - 31', async () => {
			const r = await flow({ idToken: { authTime: S - 31 } }, purpose, S);
			expect(r).toEqual({ ok: false, code: 'auth_time' });
		});
		it('refuses no claim', async () => {
			const r = await flow({ idToken: { authTime: null } }, purpose, S);
			expect(r).toEqual({ ok: false, code: 'auth_time' });
		});
	});

	it('login ignores auth_time', async () => {
		expect(await flow({ idToken: { authTime: null } }, 'login')).toMatchObject({ ok: true });
	});

	it('a sign-in slower than 30 s still passes the auth_time check', async () => {
		const S = nowS() - 120;
		expect(await flow({ idToken: { authTime: S } }, 'link', S)).toMatchObject({ ok: true });
	});

	describe('key set older than 300 s, refetch fails', () => {
		async function withCache(age: number) {
			const res = await issuer.handle(new Request(`${ISSUER}/jwks.json`));
			const jwks = (await res.json()) as { keys: object[] };
			const rp = createRelyingParty(OIDC, ORIGIN, {
				fetch: toIssuer,
				jwksCache: { jwks, uat: nowS() - age }
			});
			const { callback, txn } = await begin(rp, 'login');
			issuer.configure({ jwksFails: true });
			return rp.finish(callback, txn);
		}

		it('refuses at 301 s', async () => {
			expect(await withCache(301)).toEqual({ ok: false, code: 'id_token' });
		});

		it('uses the cached key at 10 s', async () => {
			expect(await withCache(10)).toMatchObject({ ok: true, sub: USER.sub });
		});

		it('refuses when the first fetch fails', async () => {
			expect(await flow({ jwksFails: true })).toEqual({ ok: false, code: 'id_token' });
		});
	});

	it('replayed code', async () => {
		const rp = createRelyingParty(OIDC, ORIGIN, { fetch: toIssuer });
		const { callback, txn } = await begin(rp, 'login');
		expect(await rp.finish(callback, txn)).toMatchObject({ ok: true });
		expect(await rp.finish(callback, txn)).toEqual({ ok: false, code: 'exchange' });
	});

	it('IdP error', async () => {
		expect(await flow({ deny: true })).toEqual({ ok: false, code: 'idp_error' });
	});

	it('link while another IdP session is live', async () => {
		const other = { sub: 'subject-other-0002', preferredUsername: 'other.sso' };
		secrets.add(other.sub).add(other.preferredUsername);
		const r = await flow({ reauthUser: other }, 'link');
		expect(r).toEqual({ ok: true, iss: ISSUER, sub: other.sub, label: other.preferredUsername });
		expect(issuer.authorizeLog).toEqual([{ prompt: 'login', maxAge: '1' }]);
	});

	describe('the label', () => {
		const bare = { sub: USER.sub };

		// The stub's ID token and userinfo carry the same user; this answers userinfo apart.
		function userinfoAnswering(body: object): Fetch {
			return (input, init) => {
				const req = new Request(input, init);
				if (req.url.endsWith('/userinfo')) {
					return Promise.resolve(Response.json(body));
				}
				return issuer.handle(req);
			};
		}

		async function viaUserinfo(body: object) {
			issuer.configure({ user: bare });
			const rp = createRelyingParty(OIDC, ORIGIN, { fetch: userinfoAnswering(body) });
			const { callback, txn } = await begin(rp, 'login');
			return rp.finish(callback, txn);
		}

		it('label from userinfo', async () => {
			const r = await viaUserinfo({ sub: USER.sub, preferred_username: 'from.userinfo' });
			secrets.add('from.userinfo');
			expect(r).toEqual({ ok: true, iss: ISSUER, sub: USER.sub, label: 'from.userinfo' });
		});

		it('userinfo for another sub gives null', async () => {
			const r = await viaUserinfo({ sub: 'someone-else', preferred_username: 'from.userinfo' });
			expect(r).toEqual({ ok: true, iss: ISSUER, sub: USER.sub, label: null });
		});

		it('userinfo fails', async () => {
			const r = await flow({ user: bare, userinfoFails: true });
			expect(r).toEqual({ ok: true, iss: ISSUER, sub: USER.sub, label: null });
		});

		it('no preferred_username', async () => {
			const r = await flow({ user: bare });
			expect(r).toEqual({ ok: true, iss: ISSUER, sub: USER.sub, label: null });
		});

		it.each<[string, string, string | null]>([
			['300 characters', 'a'.repeat(300), 'a'.repeat(OIDC_LABEL_MAX_LEN)],
			['300 astral characters', '\u{1F600}'.repeat(300), '\u{1F600}'.repeat(OIDC_LABEL_MAX_LEN)],
			['markup', '<script>alert(1)</script>', '<script>alert(1)</script>'],
			['U+202E and BEL', 'ana‮\u0007.sso', 'ana.sso'],
			['spaces only', '    ', null],
			['format characters only', '​‮', null]
		])('odd label: %s', async (_, preferredUsername, label) => {
			secrets.add(preferredUsername);
			const r = await flow({ user: { sub: USER.sub, preferredUsername } });
			expect(r).toEqual({ ok: true, iss: ISSUER, sub: USER.sub, label });
		});
	});

	it('discovery failure', async () => {
		const rp = createRelyingParty(OIDC, ORIGIN, { fetch: toIssuer });
		issuer.configure({ down: true });
		expect(await rp.start('login')).toEqual({ ok: false });
		issuer.configure({ down: false });
		expect(await rp.start('login')).toMatchObject({ ok: true });
	});

	it('finish without a discovery gives discovery', async () => {
		const rp = createRelyingParty(OIDC, ORIGIN, { fetch: toIssuer });
		const { callback, txn } = await begin(rp, 'login');
		const fresh = createRelyingParty(OIDC, ORIGIN, { fetch: toIssuer });
		issuer.configure({ down: true });
		expect(await fresh.finish(callback, txn)).toEqual({ ok: false, code: 'discovery' });
	});

	it('nothing secret is logged', () => {
		const codes: SsoCode[] = [
			'discovery',
			'idp_error',
			'state',
			'exchange',
			'id_token',
			'auth_time',
			'not_linked',
			'mode_refused',
			'already_linked',
			'reauth_required',
			'binding',
			'not_found',
			'oidc_not_configured'
		];
		const before = calls.length;
		for (const code of codes) logSsoRefusal(code);
		expect(calls.slice(before)).toEqual(
			codes.map((code) => JSON.stringify({ level: 'warn', msg: 'sso refused', code }))
		);
		expect(secrets.size).toBeGreaterThan(10);
		for (const line of calls) {
			for (const s of secrets) expect(line).not.toContain(s);
		}
	});
});
