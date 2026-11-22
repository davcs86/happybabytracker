import { afterEach, beforeEach, describe, expect, it, vi, type MockInstance } from 'vitest';
import { MODULES, TRACKER_KINDS } from '../constants';
import { createApi } from './api';
import type { Env } from './env';

const TOKEN = 'internal-' + 'x'.repeat(40);
const SESSION = 'session-' + 's'.repeat(35);
const env: Env = {
	internalApiToken: TOKEN,
	apiBaseUrl: 'http://api:9210/',
	origin: 'https://x.test',
	port: 9212
};
const json = (body: unknown, status = 200, headers: Record<string, string> = {}) =>
	new Response(JSON.stringify(body), {
		status,
		headers: { 'content-type': 'application/json', ...headers }
	});
const mockFetch = (impl: (u: string, i?: RequestInit) => Promise<Response>) =>
	vi.fn<(u: string, i?: RequestInit) => Promise<Response>>(impl);
const asFetch = (f: unknown) => f as typeof fetch;

let logSpy: MockInstance;
beforeEach(() => {
	logSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
});
afterEach(() => vi.restoreAllMocks());

const logged = () => logSpy.mock.calls.map((c) => c.map(String).join(' ')).join('\n');

describe('createApi headers', () => {
	it('sends authorization, x-session-token and x-client-address', async () => {
		const f = mockFetch(async () => json({ kind: 'admin', expires_at: '2026-11-21T12:00:00Z' }));
		await createApi(env, asFetch(f), SESSION, '203.0.113.9').session();
		const [url, init] = f.mock.calls[0];
		expect(url).toBe('http://api:9210/v1/session');
		expect(init?.method).toBe('GET');
		const h = new Headers(init?.headers);
		expect(h.get('authorization')).toBe(`Bearer ${TOKEN}`);
		expect(h.get('x-session-token')).toBe(SESSION);
		expect(h.get('x-client-address')).toBe('203.0.113.9');
	});

	it('omits the session and client headers when not given', async () => {
		const f = mockFetch(async () => json({ status: 'ok' }));
		await createApi(env, asFetch(f)).health();
		const h = new Headers(f.mock.calls[0][1]?.headers);
		expect(h.get('authorization')).toBe(`Bearer ${TOKEN}`);
		expect(h.has('x-session-token')).toBe(false);
		expect(h.has('x-client-address')).toBe(false);
	});

	it('applies a 5000 ms abort signal to every call', async () => {
		const spy = vi.spyOn(AbortSignal, 'timeout');
		const f = mockFetch(async () => new Response(null, { status: 204 }));
		const api = createApi(env, asFetch(f), SESSION, '1.2.3.4');
		await api.health();
		await api.session();
		await api.logout();
		await api.login('ana', 'pw');
		await api.peekToken('abc');
		await api.redeemToken('abc', 'pw');
		expect(spy).toHaveBeenCalledTimes(6);
		for (const call of spy.mock.calls) expect(call).toEqual([5000]);
		for (const [, init] of f.mock.calls) expect(init?.signal).toBeInstanceOf(AbortSignal);
	});
});

describe('createApi requests and shapes', () => {
	it('logs in with a JSON body and maps the session token', async () => {
		const f = mockFetch(async () =>
			json({ token: 't1', kind: 'parent', expires_at: '2026-12-21T00:00:00Z' })
		);
		const r = await createApi(env, asFetch(f), undefined, '1.2.3.4').login('Ana', 'secret pw');
		expect(r).toEqual({
			ok: true,
			data: { token: 't1', kind: 'parent', expiresAt: '2026-12-21T00:00:00Z' }
		});
		const [url, init] = f.mock.calls[0];
		expect(url).toBe('http://api:9210/v1/session');
		expect(init?.method).toBe('POST');
		expect(new Headers(init?.headers).get('content-type')).toBe('application/json');
		expect(JSON.parse(String(init?.body))).toEqual({ username: 'Ana', password: 'secret pw' });
	});

	it('maps a parent session to camelCase', async () => {
		const f = mockFetch(async () =>
			json({
				kind: 'parent',
				expires_at: '2026-12-21T00:00:00Z',
				parent: {
					id: 'p1',
					username: 'ana',
					display_name: 'Ana',
					units: { weight: 'kg', length: 'cm', volume: 'ml', temp: 'c' }
				}
			})
		);
		expect(await createApi(env, asFetch(f), SESSION).session()).toEqual({
			ok: true,
			data: {
				kind: 'parent',
				parent: {
					id: 'p1',
					username: 'ana',
					displayName: 'Ana',
					units: { weight: 'kg', length: 'cm', volume: 'ml', temp: 'c' }
				}
			}
		});
	});

	it('maps an admin session without a parent', async () => {
		const f = mockFetch(async () => json({ kind: 'admin', expires_at: '2026-11-21T12:00:00Z' }));
		expect(await createApi(env, asFetch(f), SESSION).session()).toEqual({
			ok: true,
			data: { kind: 'admin' }
		});
	});

	it('logs out with DELETE /v1/session', async () => {
		const f = mockFetch(async () => new Response(null, { status: 204 }));
		expect(await createApi(env, asFetch(f), SESSION).logout()).toEqual({ ok: true, data: null });
		expect(f.mock.calls[0][0]).toBe('http://api:9210/v1/session');
		expect(f.mock.calls[0][1]?.method).toBe('DELETE');
	});

	it('peeks and redeems a token', async () => {
		const f = mockFetch(async (u) =>
			u.endsWith('/redeem')
				? json({ token: 's2', kind: 'parent', expires_at: '2026-12-21T00:00:00Z' })
				: json({ kind: 'invite', username: 'bo' })
		);
		const api = createApi(env, asFetch(f), undefined, '1.2.3.4');
		expect(await api.peekToken('Ab_9-')).toEqual({
			ok: true,
			data: { kind: 'invite', username: 'bo' }
		});
		expect(await api.redeemToken('Ab_9-', 'a long password')).toEqual({
			ok: true,
			data: { token: 's2', kind: 'parent', expiresAt: '2026-12-21T00:00:00Z' }
		});
		expect(f.mock.calls[0][0]).toBe('http://api:9210/v1/tokens/Ab_9-');
		expect(f.mock.calls[1][0]).toBe('http://api:9210/v1/tokens/Ab_9-/redeem');
		expect(f.mock.calls[1][1]?.method).toBe('POST');
		expect(JSON.parse(String(f.mock.calls[1][1]?.body))).toEqual({ password: 'a long password' });
	});

	it.each(['..', '.', '', 'a/b', 'a?b', 'a%2Fb', 'a'.repeat(129)])(
		'a malformed token %j is not_found without a request',
		async (token) => {
			const f = mockFetch(async () => json({}));
			const api = createApi(env, asFetch(f));
			expect(await api.peekToken(token)).toEqual({ ok: false, status: 404, reason: 'not_found' });
			expect(await api.redeemToken(token, 'x'.repeat(12))).toEqual({
				ok: false,
				status: 404,
				reason: 'not_found'
			});
			expect(f).not.toHaveBeenCalled();
		}
	);

	it('reports the healthz status', async () => {
		const ok = createApi(env, asFetch(mockFetch(async () => json({ status: 'ok' }))));
		expect(await ok.health()).toEqual({ ok: true, data: null });
		const f = mockFetch(async () => json({ status: 'ok' }));
		await createApi(env, asFetch(f)).health();
		expect(f.mock.calls[0][0]).toBe('http://api:9210/healthz');
	});
});

describe('createApi failure mapping', () => {
	const failing = (status: number, body: unknown = { error: 'some_code' }, headers = {}) =>
		createApi(env, asFetch(mockFetch(async () => json(body, status, headers))), SESSION);

	it.each([
		[401, 'unauthorized'],
		[403, 'forbidden'],
		[404, 'not_found'],
		[409, 'conflict'],
		[429, 'rate_limited'],
		[400, 'error'],
		[413, 'error'],
		[500, 'error'],
		[502, 'unavailable'],
		[503, 'unavailable'],
		[504, 'unavailable']
	] as const)('maps %i to %s with the code', async (status, reason) => {
		expect(await failing(status).session()).toEqual({
			ok: false,
			status,
			reason,
			code: 'some_code'
		});
	});

	it('maps 422 to invalid with the field', async () => {
		expect(await failing(422, { error: 'invalid', field: 'password' }).session()).toEqual({
			ok: false,
			status: 422,
			reason: 'invalid',
			code: 'invalid',
			field: 'password'
		});
	});

	it('keeps the reason when the error body is not JSON', async () => {
		const api = createApi(
			env,
			asFetch(mockFetch(async () => new Response('<html>', { status: 401 }))),
			SESSION
		);
		expect(await api.session()).toEqual({ ok: false, status: 401, reason: 'unauthorized' });
	});

	it('maps a network error and a timeout to unavailable', async () => {
		const net = createApi(
			env,
			asFetch(async () => {
				throw new TypeError('fetch failed');
			})
		);
		expect(await net.session()).toEqual({ ok: false, status: 0, reason: 'unavailable' });
		const timeout = createApi(
			env,
			asFetch(async () => {
				throw new DOMException('timed out', 'TimeoutError');
			})
		);
		expect(await timeout.session()).toEqual({ ok: false, status: 0, reason: 'unavailable' });
	});

	it('maps a body read that times out to unavailable', async () => {
		const res = json({});
		vi.spyOn(res, 'json').mockRejectedValue(new DOMException('timed out', 'TimeoutError'));
		const api = createApi(env, asFetch(mockFetch(async () => res)), SESSION);
		expect(await api.session()).toEqual({ ok: false, status: 0, reason: 'unavailable' });
	});

	it('maps an unparsable or malformed 2xx body to error', async () => {
		const html = createApi(
			env,
			asFetch(mockFetch(async () => new Response('<html>', { status: 200 }))),
			SESSION
		);
		expect(await html.session()).toEqual({ ok: false, status: 200, reason: 'error' });
		const wrong = createApi(env, asFetch(mockFetch(async () => json({ kind: 'root' }))), SESSION);
		expect(await wrong.session()).toEqual({ ok: false, status: 200, reason: 'error' });
	});

	it('aborts after exactly 5 s: pending at 4999 ms, unavailable at 5000 ms', async () => {
		vi.useFakeTimers();
		try {
			vi.spyOn(AbortSignal, 'timeout').mockImplementation((ms) => {
				const c = new AbortController();
				setTimeout(() => c.abort(new DOMException('timed out', 'TimeoutError')), ms);
				return c.signal;
			});
			const hang = ((_u: string, init?: RequestInit) =>
				new Promise((_resolve, reject) => {
					init?.signal?.addEventListener('abort', () => reject(init.signal?.reason));
				})) as unknown as typeof fetch;
			let settled: unknown;
			const p = createApi(env, hang, SESSION)
				.session()
				.then((r) => (settled = r));
			await vi.advanceTimersByTimeAsync(4999);
			expect(settled).toBeUndefined();
			await vi.advanceTimersByTimeAsync(1);
			await p;
			expect(settled).toEqual({ ok: false, status: 0, reason: 'unavailable' });
		} finally {
			vi.useRealTimers();
		}
	});
});

describe('createApi logging', () => {
	it('writes one fixed-shape line per failure: endpoint name and status only', async () => {
		const api = createApi(
			env,
			asFetch(
				mockFetch(async () =>
					json({ error: 'invalid_credentials', echo: 'secret pw' }, 401, { 'x-echo': SESSION })
				)
			),
			SESSION,
			'203.0.113.9'
		);
		await api.login('ana', 'secret pw');
		expect(logSpy).toHaveBeenCalledTimes(1);
		expect(JSON.parse(String(logSpy.mock.calls[0][0]))).toEqual({
			level: 'error',
			msg: 'api call failed',
			endpoint: 'login',
			status: 401
		});
	});

	it.each([
		['network', new TypeError('fetch failed http://api:9210/v1/tokens/SECRETTOKEN')],
		['timeout', new DOMException('timed out', 'TimeoutError')]
	])('logs %s failures without the error text', async (status, err) => {
		const f = mockFetch(async () => {
			throw err;
		});
		await createApi(env, asFetch(f), SESSION, '203.0.113.9').peekToken('SECRETTOKEN');
		expect(JSON.parse(String(logSpy.mock.calls[0][0]))).toEqual({
			level: 'error',
			msg: 'api call failed',
			endpoint: 'peek_token',
			status
		});
	});

	it('never logs a token, a body, a header value or a query', async () => {
		const f = mockFetch(async () => json({ error: 'not_found', detail: 'secret pw' }, 404));
		const api = createApi(env, asFetch(f), SESSION, '203.0.113.9');
		await api.peekToken('SECRETTOKEN');
		await api.redeemToken('SECRETTOKEN', 'secret pw');
		await api.login('ana', 'secret pw');
		await api.session();
		const out = logged();
		expect(logSpy).toHaveBeenCalledTimes(4);
		for (const needle of [
			TOKEN,
			SESSION,
			'SECRETTOKEN',
			'secret pw',
			'ana',
			'203.0.113.9',
			'?',
			'http'
		]) {
			expect(out).not.toContain(needle);
		}
	});

	it('logs nothing on success', async () => {
		const f = mockFetch(async () => json({ status: 'ok' }));
		await createApi(env, asFetch(f)).health();
		expect(logSpy).not.toHaveBeenCalled();
	});
});

describe('createApi admin, account, kid and access calls', () => {
	const KID_WIRE = {
		id: 'k-1',
		name: 'Mia',
		birth_date: '2026-07-08',
		sex: 'female',
		gestation_weeks: 35,
		gestation_days: null,
		time_zone: 'Europe/Madrid',
		growth_source: 'cdc',
		role: 'caregiver'
	};
	const KID = {
		id: 'k-1',
		name: 'Mia',
		birthDate: '2026-07-08',
		sex: 'female',
		gestationWeeks: 35,
		gestationDays: null,
		timeZone: 'Europe/Madrid',
		growthSource: 'cdc',
		role: 'caregiver'
	};
	const PARENT_WIRE = {
		id: 'p-1',
		username: 'eve',
		display_name: 'Eve',
		disabled: false,
		has_password: false,
		created_at: '2026-11-21T00:00:00Z'
	};
	const sent = (f: ReturnType<typeof mockFetch>, i = 0) => ({
		url: f.mock.calls[i][0],
		method: f.mock.calls[i][1]?.method,
		body:
			f.mock.calls[i][1]?.body === undefined
				? undefined
				: JSON.parse(String(f.mock.calls[i][1]?.body))
	});

	it('maps a kid and a kid list to camelCase', async () => {
		const f = mockFetch(async (u) => json(u.endsWith('/kids') ? { kids: [KID_WIRE] } : KID_WIRE));
		const api = createApi(env, asFetch(f), SESSION);
		expect(await api.listKids()).toEqual({ ok: true, data: [KID] });
		expect(await api.getKid('k-1')).toEqual({ ok: true, data: KID });
		expect(sent(f, 1)).toMatchObject({ url: 'http://api:9210/v1/kids/k-1', method: 'GET' });
	});

	it('sends every kid field on create and update, gestation as explicit null', async () => {
		const f = mockFetch(async () => json(KID_WIRE));
		const api = createApi(env, asFetch(f), SESSION);
		const input = {
			name: 'Mia',
			birthDate: '2026-07-08',
			sex: 'female',
			gestationWeeks: null,
			gestationDays: null,
			timeZone: 'UTC'
		};
		const body = {
			name: 'Mia',
			birth_date: '2026-07-08',
			sex: 'female',
			gestation_weeks: null,
			gestation_days: null,
			time_zone: 'UTC'
		};
		await api.createKid(input);
		await api.updateKid('k-1', input);
		expect(sent(f, 0)).toEqual({ url: 'http://api:9210/v1/kids', method: 'POST', body });
		expect(sent(f, 1)).toEqual({ url: 'http://api:9210/v1/kids/k-1', method: 'PATCH', body });
	});

	it('sends growth_source only when the input has one', async () => {
		const f = mockFetch(async () => json(KID_WIRE));
		const input = {
			name: 'Mia',
			birthDate: '2026-07-08',
			sex: 'female',
			gestationWeeks: null,
			gestationDays: null,
			timeZone: 'UTC',
			growthSource: 'cdc'
		};
		await createApi(env, asFetch(f), SESSION).updateKid('k-1', input);
		expect(sent(f).body).toMatchObject({ growth_source: 'cdc' });
	});

	it('deletes a kid with confirm_name', async () => {
		const f = mockFetch(async () => new Response(null, { status: 204 }));
		expect(await createApi(env, asFetch(f), SESSION).deleteKid('k-1', 'Mia')).toEqual({
			ok: true,
			data: null
		});
		expect(sent(f)).toEqual({
			url: 'http://api:9210/v1/kids/k-1',
			method: 'DELETE',
			body: { confirm_name: 'Mia' }
		});
	});

	it('refuses a kid with an unknown sex, role or a fractional gestation', async () => {
		for (const bad of [
			{ sex: 'x' },
			{ role: 'owner' },
			{ gestation_weeks: 35.5 },
			{ growth_source: 'uk-who' },
			{ growth_source: undefined }
		]) {
			const f = mockFetch(async () => json({ ...KID_WIRE, ...bad }));
			expect(await createApi(env, asFetch(f), SESSION).getKid('k-1')).toMatchObject({
				ok: false,
				reason: 'error'
			});
		}
	});

	it('lists, sets and removes access', async () => {
		const f = mockFetch(async (_u, i) =>
			i?.method === 'GET'
				? json({
						grants: [{ parent_id: 'p-1', username: 'ana', display_name: 'Ana', role: 'guardian' }]
					})
				: new Response(null, { status: 204 })
		);
		const api = createApi(env, asFetch(f), SESSION);
		expect(await api.listAccess('k-1')).toEqual({
			ok: true,
			data: [{ parentId: 'p-1', username: 'ana', displayName: 'Ana', role: 'guardian' }]
		});
		await api.setAccess('k-1', '  Dan ', 'caregiver');
		await api.removeAccess('k-1', 'p-2');
		expect(sent(f, 1)).toEqual({
			url: 'http://api:9210/v1/kids/k-1/access/Dan',
			method: 'PUT',
			body: { role: 'caregiver' }
		});
		expect(sent(f, 2)).toMatchObject({
			url: 'http://api:9210/v1/kids/k-1/access/p-2',
			method: 'DELETE'
		});
	});

	it('encodes the username as one path segment', async () => {
		const f = mockFetch(async () => new Response(null, { status: 204 }));
		await createApi(env, asFetch(f), SESSION).setAccess('k-1', 'a/b?c', 'guardian');
		expect(sent(f).url).toBe('http://api:9210/v1/kids/k-1/access/a%2Fb%3Fc');
	});

	it.each(['', '   ', '.', '..'])(
		'a username %j is parent_not_found without a request',
		async (username) => {
			const f = mockFetch(async () => json({}));
			expect(
				await createApi(env, asFetch(f), SESSION).setAccess('k-1', username, 'guardian')
			).toEqual({ ok: false, status: 404, reason: 'not_found', code: 'parent_not_found' });
			expect(f).not.toHaveBeenCalled();
		}
	);

	it.each(['..', '.', '', 'a/b', 'a.b', 'a%2Fb', 'a'.repeat(65)])(
		'an id %j is not_found without a request',
		async (id) => {
			const f = mockFetch(async () => json({}));
			const api = createApi(env, asFetch(f), SESSION);
			const nf = { ok: false, status: 404, reason: 'not_found' };
			expect(await api.getKid(id)).toEqual(nf);
			expect(await api.updateKid(id, {} as never)).toEqual(nf);
			expect(await api.deleteKid(id, 'x')).toEqual(nf);
			expect(await api.listAccess(id)).toEqual(nf);
			expect(await api.setAccess(id, 'ana', 'guardian')).toEqual(nf);
			expect(await api.removeAccess(id, 'p-1')).toEqual(nf);
			expect(await api.removeAccess('k-1', id)).toEqual(nf);
			expect(await api.deleteSession(id)).toEqual(nf);
			expect(await api.parentLink(id)).toEqual(nf);
			expect(await api.setParentDisabled(id, true)).toEqual(nf);
			expect(f).not.toHaveBeenCalled();
		}
	);

	it('creates a parent and returns the invite link', async () => {
		const f = mockFetch(async () =>
			json(
				{
					parent: PARENT_WIRE,
					invite_url: 'https://x.test/invite/t',
					expires_at: '2026-11-24T00:00:00Z'
				},
				201
			)
		);
		expect(await createApi(env, asFetch(f), SESSION).createParent('eve', 'Eve')).toEqual({
			ok: true,
			data: {
				parent: {
					id: 'p-1',
					username: 'eve',
					displayName: 'Eve',
					disabled: false,
					hasPassword: false,
					createdAt: '2026-11-21T00:00:00Z'
				},
				link: { kind: 'invite', url: 'https://x.test/invite/t', expiresAt: '2026-11-24T00:00:00Z' }
			}
		});
		expect(sent(f)).toEqual({
			url: 'http://api:9210/v1/admin/parents',
			method: 'POST',
			body: { username: 'eve', display_name: 'Eve' }
		});
	});

	it('tells a reset link from an invite by the response key', async () => {
		const reset = mockFetch(async () =>
			json({ reset_url: 'https://x.test/reset/r', expires_at: 'e' })
		);
		const invite = mockFetch(async () =>
			json({ invite_url: 'https://x.test/invite/i', expires_at: 'e' })
		);
		expect(await createApi(env, asFetch(reset), SESSION).parentLink('p-1')).toEqual({
			ok: true,
			data: { kind: 'reset', url: 'https://x.test/reset/r', expiresAt: 'e' }
		});
		expect(await createApi(env, asFetch(invite), SESSION).parentLink('p-1')).toEqual({
			ok: true,
			data: { kind: 'invite', url: 'https://x.test/invite/i', expiresAt: 'e' }
		});
		expect(sent(reset)).toMatchObject({
			url: 'http://api:9210/v1/admin/parents/p-1/reset-link',
			method: 'POST'
		});
	});

	it('lists, disables and enables parents; keeps the sole_guardian code', async () => {
		const f = mockFetch(async (u) =>
			u.endsWith('/parents')
				? json({ parents: [PARENT_WIRE] })
				: u.endsWith('/disable')
					? json({ error: 'sole_guardian' }, 409)
					: new Response(null, { status: 204 })
		);
		const api = createApi(env, asFetch(f), SESSION);
		expect((await api.listParents()).ok).toBe(true);
		expect(await api.setParentDisabled('p-1', true)).toEqual({
			ok: false,
			status: 409,
			reason: 'conflict',
			code: 'sole_guardian'
		});
		expect(await api.setParentDisabled('p-1', false)).toEqual({ ok: true, data: null });
		expect(sent(f, 2).url).toBe('http://api:9210/v1/admin/parents/p-1/enable');
	});

	it('updates the profile, changes the password and manages sessions', async () => {
		const f = mockFetch(async (u, i) => {
			if (u.endsWith('/v1/me'))
				return json({
					id: 'p-1',
					username: 'ana',
					display_name: 'Ana B',
					units: { weight: 'lb', length: 'in', volume: 'floz', temp: 'f' }
				});
			if (u.endsWith('/sessions') && i?.method === 'GET')
				return json({
					sessions: [{ id: 's-1', created_at: 'c', last_seen_at: 'l', current: true }]
				});
			return new Response(null, { status: 204 });
		});
		const api = createApi(env, asFetch(f), SESSION);
		const units = { weight: 'lb', length: 'in', volume: 'floz', temp: 'f' } as const;
		expect(await api.updateProfile('Ana B', units)).toEqual({
			ok: true,
			data: { id: 'p-1', username: 'ana', displayName: 'Ana B', units }
		});
		expect(sent(f, 0)).toEqual({
			url: 'http://api:9210/v1/me',
			method: 'PATCH',
			body: { display_name: 'Ana B', units }
		});
		await api.changePassword('old password!', 'new password!');
		expect(sent(f, 1)).toEqual({
			url: 'http://api:9210/v1/me/password',
			method: 'POST',
			body: { current: 'old password!', new: 'new password!' }
		});
		expect(await api.listSessions()).toEqual({
			ok: true,
			data: [{ id: 's-1', createdAt: 'c', lastSeenAt: 'l', current: true }]
		});
		await api.deleteSession('s-2');
		expect(sent(f, 3)).toMatchObject({
			url: 'http://api:9210/v1/me/sessions/s-2',
			method: 'DELETE'
		});
	});

	it('maps a list without its key to error', async () => {
		const f = mockFetch(async () => json({ items: [] }));
		expect(await createApi(env, asFetch(f), SESSION).listKids()).toMatchObject({
			ok: false,
			reason: 'error'
		});
	});
});

describe('createApi tracker calls', () => {
	const KID = 'k-1';
	const VOLUME_WIRE = { ml: 120, floz_hundredths: 406, typed_unit: 'ml' };
	const VOLUME = VOLUME_WIRE;
	const T0 = '2026-11-20T10:00:00Z';
	const T1 = '2026-11-20T10:30:00Z';
	const sent = (f: ReturnType<typeof mockFetch>, i = 0) => ({
		url: f.mock.calls[i][0],
		method: f.mock.calls[i][1]?.method,
		body:
			f.mock.calls[i][1]?.body === undefined
				? undefined
				: JSON.parse(String(f.mock.calls[i][1]?.body))
	});
	const respond = (body: unknown, status = 200) =>
		mockFetch(async () => (status === 204 ? new Response(null, { status }) : json(body, status)));
	const api = (f: ReturnType<typeof mockFetch>) => createApi(env, asFetch(f), SESSION);

	// The wire shape of each tracker, as the Go views write it, and what the client makes of it.
	const ENTRIES = {
		bottles: {
			wire: {
				id: 'e1',
				at: T0,
				contents: ['breast_milk', 'formula'],
				volume: VOLUME_WIRE,
				note: null
			},
			entry: { id: 'e1', at: T0, contents: ['breast_milk', 'formula'], volume: VOLUME, note: null }
		},
		diapers: {
			wire: { id: 'e1', at: T0, kind: 'both', note: 'rash' },
			entry: { id: 'e1', at: T0, kind: 'both', note: 'rash' }
		},
		meals: {
			wire: { id: 'e1', at: T0, what: 'pear', reaction: null, note: null },
			entry: { id: 'e1', at: T0, what: 'pear', reaction: null, note: null }
		},
		pumping: {
			wire: {
				id: 'e1',
				started_at: T0,
				duration_seconds: 600,
				volume: VOLUME_WIRE,
				side: 'both',
				note: null
			},
			entry: {
				id: 'e1',
				startedAt: T0,
				durationSeconds: 600,
				volume: VOLUME,
				side: 'both',
				note: null
			}
		},
		sleep: {
			wire: { id: 'e1', started_at: T0, ended_at: null, note: null },
			entry: { id: 'e1', startedAt: T0, endedAt: null, note: null }
		},
		nursing: {
			wire: {
				id: 'e1',
				started_at: T0,
				ended_at: null,
				left_seconds: 60,
				right_seconds: 0,
				side: 'right',
				side_since: T1,
				note: null
			},
			entry: {
				id: 'e1',
				startedAt: T0,
				endedAt: null,
				leftSeconds: 60,
				rightSeconds: 0,
				side: 'right',
				sideSince: T1,
				note: null
			}
		}
	} as const;
	const KINDS = Object.keys(ENTRIES) as (keyof typeof ENTRIES)[];

	it.each(KINDS)('lists %s newest first with the query and the cursor', async (kind) => {
		const f = respond({ [kind]: [ENTRIES[kind].wire], next_cursor: 'abc_-9' });
		const r = await api(f)
			.tracker(kind)
			.list(KID, { from: T0, to: T1, limit: 2, cursor: 'prev/cursor' });
		expect(r).toEqual({ ok: true, data: { items: [ENTRIES[kind].entry], nextCursor: 'abc_-9' } });
		expect(sent(f)).toEqual({
			url: `http://api:9210/v1/kids/k-1/${kind}?from=${encodeURIComponent(T0)}&to=${encodeURIComponent(T1)}&limit=2&cursor=prev%2Fcursor`,
			method: 'GET',
			body: undefined
		});
	});

	it('lists without a query string and reads the last page as a null cursor', async () => {
		const f = respond({ diapers: [], next_cursor: null });
		expect(await api(f).tracker('diapers').list(KID)).toEqual({
			ok: true,
			data: { items: [], nextCursor: null }
		});
		expect(sent(f).url).toBe('http://api:9210/v1/kids/k-1/diapers');
	});

	it.each(KINDS)('gets, updates and removes one %s', async (kind) => {
		const f = mockFetch(async (_u, init) =>
			init?.method === 'DELETE' ? new Response(null, { status: 204 }) : json(ENTRIES[kind].wire)
		);
		const t = api(f).tracker(kind);
		expect(await t.get(KID, 'e1')).toEqual({ ok: true, data: ENTRIES[kind].entry });
		expect(await t.update(KID, 'e1', { note: null })).toEqual({
			ok: true,
			data: ENTRIES[kind].entry
		});
		expect(await t.remove(KID, 'e1')).toEqual({ ok: true, data: null });
		expect([sent(f, 0), sent(f, 1), sent(f, 2)]).toEqual([
			{ url: `http://api:9210/v1/kids/k-1/${kind}/e1`, method: 'GET', body: undefined },
			{ url: `http://api:9210/v1/kids/k-1/${kind}/e1`, method: 'PATCH', body: { note: null } },
			{ url: `http://api:9210/v1/kids/k-1/${kind}/e1`, method: 'DELETE', body: undefined }
		]);
	});

	it('creates with the body as given and parses the 201 entry', async () => {
		const f = respond(ENTRIES.diapers.wire, 201);
		const body = { at: T0, kind: 'both', note: 'rash' };
		expect(await api(f).tracker('diapers').create(KID, body)).toEqual({
			ok: true,
			data: ENTRIES.diapers.entry
		});
		expect(sent(f)).toEqual({ url: 'http://api:9210/v1/kids/k-1/diapers', method: 'POST', body });
	});

	it('a refused create is invalid with the API field and code', async () => {
		const f = respond({ error: 'invalid', field: 'contents' }, 422);
		expect(await api(f).tracker('bottles').create(KID, { contents: [] })).toEqual({
			ok: false,
			status: 422,
			reason: 'invalid',
			code: 'invalid',
			field: 'contents'
		});
	});

	it('an id outside the pattern is not_found without a request', async () => {
		const f = respond({});
		const t = api(f).tracker('bottles');
		for (const r of [
			await t.list('a/b'),
			await t.get(KID, '..?x'),
			await t.update('a/b', 'e1', {}),
			await t.remove(KID, 'a b'),
			await t.create('', {})
		]) {
			expect(r).toEqual({ ok: false, status: 404, reason: 'not_found' });
		}
		expect(f).not.toHaveBeenCalled();
	});

	it.each([
		['a body without its list key', { items: [], next_cursor: null }],
		['a list that is not an array', { bottles: 'none', next_cursor: null }],
		[
			'an entry with an unknown content',
			{
				bottles: [{ ...ENTRIES.bottles.wire, contents: ['juice'] }],
				next_cursor: null
			}
		],
		[
			'an entry with no content',
			{
				bottles: [{ ...ENTRIES.bottles.wire, contents: [] }],
				next_cursor: null
			}
		],
		[
			'an entry with a fractional volume',
			{
				bottles: [{ ...ENTRIES.bottles.wire, volume: { ...VOLUME_WIRE, ml: 1.5 } }],
				next_cursor: null
			}
		],
		[
			'an entry without an id',
			{ bottles: [{ ...ENTRIES.bottles.wire, id: 7 }], next_cursor: null }
		],
		['a cursor that is not a string', { bottles: [], next_cursor: 4 }],
		['a missing cursor', { bottles: [] }]
	])('maps %s to error', async (_name, body) => {
		const f = respond(body);
		expect(await api(f).tracker('bottles').list(KID)).toEqual({
			ok: false,
			status: 200,
			reason: 'error'
		});
		expect(JSON.parse(String(logSpy.mock.calls[0][0]))).toEqual({
			level: 'error',
			msg: 'api call failed',
			endpoint: 'list_bottles',
			status: 200
		});
	});

	it('refuses a nursing entry with a side no one can feed on', async () => {
		const f = respond({ ...ENTRIES.nursing.wire, side: 'both' });
		expect(await api(f).tracker('nursing').get(KID, 'e1')).toMatchObject({ reason: 'error' });
	});

	describe('timers', () => {
		it.each(['sleep', 'nursing'] as const)('ends a %s at an instant', async (kind) => {
			const f = respond(ENTRIES[kind].wire);
			expect(await api(f).endTimer(kind, KID, 'e1', T1)).toEqual({
				ok: true,
				data: ENTRIES[kind].entry
			});
			expect(sent(f)).toEqual({
				url: `http://api:9210/v1/kids/k-1/${kind}/e1/end`,
				method: 'POST',
				body: { at: T1 }
			});
		});

		it('ends now when no instant is given', async () => {
			const f = respond(ENTRIES.sleep.wire);
			await api(f).endTimer('sleep', KID, 'e1');
			expect(sent(f).body).toEqual({});
		});

		it('a timer that is not running is a conflict with its code', async () => {
			const f = respond({ error: 'not_running' }, 409);
			expect(await api(f).endTimer('sleep', KID, 'e1')).toEqual({
				ok: false,
				status: 409,
				reason: 'conflict',
				code: 'not_running'
			});
		});

		it('a second timer is a conflict with its code', async () => {
			const f = respond({ error: 'already_running' }, 409);
			expect(await api(f).tracker('nursing').create(KID, { side: 'left' })).toEqual({
				ok: false,
				status: 409,
				reason: 'conflict',
				code: 'already_running'
			});
		});

		it('a timer that ran over a day cannot end: invalid at at', async () => {
			const f = respond({ error: 'invalid', field: 'at' }, 422);
			expect(await api(f).endTimer('nursing', KID, 'e1', T1)).toEqual({
				ok: false,
				status: 422,
				reason: 'invalid',
				code: 'invalid',
				field: 'at'
			});
		});

		it('switches the nursing side, with or without an instant', async () => {
			const f = respond(ENTRIES.nursing.wire);
			expect(await api(f).switchNursingSide(KID, 'e1', 'right')).toEqual({
				ok: true,
				data: ENTRIES.nursing.entry
			});
			await api(f).switchNursingSide(KID, 'e1', 'left', T1);
			expect(sent(f, 0)).toEqual({
				url: 'http://api:9210/v1/kids/k-1/nursing/e1/side',
				method: 'POST',
				body: { side: 'right' }
			});
			expect(sent(f, 1).body).toEqual({ side: 'left', at: T1 });
		});

		it('an unsafe id never reaches an end or side path', async () => {
			const f = respond({});
			expect(await api(f).endTimer('sleep', KID, '../x')).toMatchObject({ reason: 'not_found' });
			expect(await api(f).switchNursingSide('a/b', 'e1', 'left')).toMatchObject({
				reason: 'not_found'
			});
			expect(f).not.toHaveBeenCalled();
		});
	});

	describe('timeline, summary, modules and home', () => {
		const DAY = {
			date: '2026-11-20',
			bottles: { count: 1, ml: 120, floz_hundredths: 406 },
			nursing: { count: 1, seconds: 120 },
			pumping: { count: 0, ml: 0, floz_hundredths: 0 },
			diapers: { total: 2, wet: 1, dirty: 0, both: 1, dry: 0 },
			sleep: { seconds: 1800 },
			meals: { count: 1 }
		};
		const MODULE = { module: 'bottles', enabled: true, default: true, overridden: false };
		const TYPES = [
			['bottle', 'bottles'],
			['nursing', 'nursing'],
			['pumping', 'pumping'],
			['sleep', 'sleep'],
			['diaper', 'diapers'],
			['meal', 'meals']
		] as const;

		it('maps every timeline type to its route name and keeps the entry', async () => {
			const items = TYPES.map(([type, kind]) => ({ type, at: T0, ...ENTRIES[kind].wire }));
			const f = respond({ items, next_cursor: 'c' });
			const r = await api(f).timeline(KID, { from: T0, limit: 6, types: ['bottle', 'meal'] });
			expect(r).toEqual({
				ok: true,
				data: {
					items: TYPES.map(([, kind]) => ({ kind, at: T0, entry: ENTRIES[kind].entry })),
					nextCursor: 'c'
				}
			});
			expect(sent(f).url).toBe(
				`http://api:9210/v1/kids/k-1/timeline?from=${encodeURIComponent(T0)}&limit=6&types=bottle%2Cmeal`
			);
		});

		it('sends no types when none are given, so the API applies the kid modules', async () => {
			const f = respond({ items: [], next_cursor: null });
			await api(f).timeline(KID);
			await api(f).timeline(KID, { types: [] });
			expect(sent(f, 0).url).toBe('http://api:9210/v1/kids/k-1/timeline');
			expect(sent(f, 1).url).toBe('http://api:9210/v1/kids/k-1/timeline');
		});

		it('maps a timeline item of an unknown type, or whose entry is malformed, to error', async () => {
			for (const item of [
				{ type: 'growth', at: T0, id: 'e1' },
				{ type: 'diaper', ...ENTRIES.diapers.wire, kind: 'soggy' },
				{ type: 'sleep', ...ENTRIES.sleep.wire }
			]) {
				const f = respond({ items: [item], next_cursor: null });
				expect(await api(f).timeline(KID)).toMatchObject({ ok: false, reason: 'error' });
			}
		});

		it('reads day summaries for a date range', async () => {
			const f = respond({ days: [DAY] });
			expect(await api(f).summary(KID, '2026-11-20', '2026-11-22')).toEqual({
				ok: true,
				data: [DAY]
			});
			expect(sent(f).url).toBe('http://api:9210/v1/kids/k-1/summary?from=2026-11-20&to=2026-11-22');
			const bad = respond({ days: [{ ...DAY, sleep: {} }] });
			expect(await api(bad).summary(KID, '2026-11-20', '2026-11-20')).toMatchObject({
				reason: 'error'
			});
		});

		it('a range the API refuses names its field', async () => {
			const f = respond({ error: 'invalid', field: 'to' }, 422);
			expect(await api(f).summary(KID, '2026-01-01', '2026-11-20')).toMatchObject({
				reason: 'invalid',
				field: 'to'
			});
		});

		it('reads and sets modules', async () => {
			const f = mockFetch(async (_u, init) =>
				init?.method === 'PUT' ? new Response(null, { status: 204 }) : json({ modules: [MODULE] })
			);
			const a = api(f);
			expect(await a.modules(KID)).toEqual({ ok: true, data: [MODULE] });
			expect(await a.setModule(KID, 'feeding', false)).toEqual({ ok: true, data: null });
			expect(sent(f, 1)).toEqual({
				url: 'http://api:9210/v1/kids/k-1/modules/feeding',
				method: 'PUT',
				body: { enabled: false }
			});
			const bad = respond({ modules: [{ ...MODULE, module: 'teething' }] });
			expect(await api(bad).modules(KID)).toMatchObject({ reason: 'error' });
		});

		it('a caregiver switching a module is forbidden', async () => {
			expect(await api(respond({ error: 'forbidden' }, 403)).setModule(KID, 'sleep', true)).toEqual(
				{
					ok: false,
					status: 403,
					reason: 'forbidden',
					code: 'forbidden'
				}
			);
		});

		const HOME = {
			age_months: 8,
			corrected_birth_date: '2026-03-10',
			band: '0-11',
			modules: [MODULE],
			running: { sleep: ENTRIES.sleep.wire, nursing: null },
			today: DAY
		};

		it('reads the home model, a running timer and an idle one', async () => {
			const f = respond(HOME);
			expect(await api(f).home(KID)).toEqual({
				ok: true,
				data: {
					ageMonths: 8,
					correctedBirthDate: '2026-03-10',
					band: '0-11',
					modules: [MODULE],
					running: { sleep: ENTRIES.sleep.entry, nursing: null },
					today: DAY
				}
			});
			expect(sent(f).url).toBe('http://api:9210/v1/kids/k-1/home');
		});

		it.each([
			['an unknown band', { ...HOME, band: '36+' }],
			['a missing running block', { ...HOME, running: undefined }],
			[
				'a running sleep that is malformed',
				{ ...HOME, running: { sleep: { id: 1 }, nursing: null } }
			],
			['a missing today', { ...HOME, today: undefined }],
			['a fractional age', { ...HOME, age_months: 8.5 }],
			['a missing corrected birth date', { ...HOME, corrected_birth_date: undefined }]
		])('maps %s to error', async (_name, body) => {
			expect(await api(respond(body)).home(KID)).toMatchObject({ ok: false, reason: 'error' });
		});
	});

	describe('path segments that are not a known kind or module', () => {
		const NOT_FOUND = { ok: false, status: 404, reason: 'not_found' };
		// A caller's careless cast must not reach a path: ../.. would walk to another API route.
		const BAD = ['../..', '..', '', 'diaper', 'milestones', 'Bottles', 'bottles/e1', 'sleep?x=1'];

		it.each(BAD)('tracker(%j) answers not_found on every call without a request', async (kind) => {
			const f = respond({});
			const t = api(f).tracker(kind as never);
			for (const r of [
				await t.list(KID),
				await t.get(KID, 'e1'),
				await t.create(KID, {}),
				await t.update(KID, 'e1', {}),
				await t.remove(KID, 'e1')
			]) {
				expect(r).toEqual(NOT_FOUND);
			}
			expect(f).not.toHaveBeenCalled();
		});

		it.each([...BAD, 'bottles', 'diapers', 'meals', 'pumping'])(
			'endTimer(%j) answers not_found for a kind that is not a timer, without a request',
			async (kind) => {
				const f = respond({});
				expect(await api(f).endTimer(kind as never, KID, 'e1')).toEqual(NOT_FOUND);
				expect(f).not.toHaveBeenCalled();
			}
		);

		it.each([...BAD.filter((m) => m !== 'milestones'), 'bottle', 'feedings', 'Growth'])(
			'setModule(%j) answers not_found without a request',
			async (module) => {
				const f = respond({});
				expect(await api(f).setModule(KID, module as never, true)).toEqual(NOT_FOUND);
				expect(f).not.toHaveBeenCalled();
			}
		);

		it('still reaches the API for every known kind, timer kind and module', async () => {
			const f = respond({});
			const a = api(f);
			for (const kind of TRACKER_KINDS) await a.tracker(kind).remove(KID, 'e1');
			for (const kind of ['sleep', 'nursing'] as const) await a.endTimer(kind, KID, 'e1');
			for (const module of MODULES) await a.setModule(KID, module, true);
			expect(f).toHaveBeenCalledTimes(TRACKER_KINDS.length + 2 + MODULES.length);
		});
	});

	it('logs a tracker call by its endpoint name, never the kid, the entry or the body', async () => {
		await api(respond({ error: 'invalid', field: 'note' }, 422))
			.tracker('meals')
			.update(KID, 'e-secret', { note: 'private note' });
		expect(logSpy).toHaveBeenCalledTimes(1);
		expect(JSON.parse(String(logSpy.mock.calls[0][0]))).toEqual({
			level: 'error',
			msg: 'api call failed',
			endpoint: 'update_meals',
			status: 422
		});
		expect(logged()).not.toContain('e-secret');
		expect(logged()).not.toContain('private note');
	});
});

describe('createApi growth calls', () => {
	const KID = 'k-1';
	const sent = (f: ReturnType<typeof mockFetch>, i = 0) => ({
		url: f.mock.calls[i][0],
		method: f.mock.calls[i][1]?.method,
		body:
			f.mock.calls[i][1]?.body === undefined
				? undefined
				: JSON.parse(String(f.mock.calls[i][1]?.body))
	});
	const respond = (body: unknown, status = 200) =>
		mockFetch(async () => (status === 204 ? new Response(null, { status }) : json(body, status)));
	const api = (f: ReturnType<typeof mockFetch>) => createApi(env, asFetch(f), SESSION);
	const NOT_FOUND = { ok: false, status: 404, reason: 'not_found' };

	const ROW_WIRE = {
		id: 'g1',
		measured_on: '2026-11-01',
		weight: { grams: 3728, oz_hundredths: 13150, typed_unit: 'lb' },
		length: { mm: 495, in_hundredths: 1949, typed_unit: 'cm' },
		head: null,
		note: null,
		age: { days: 40, corrected_days: 7 },
		scores: {
			weight: { scored: true, z: 0.21, percentile: 58.2 },
			length: { scored: false, reason: 'not_covered' },
			head: null,
			bmi: { scored: false, reason: 'before_term', value: 15.21 }
		}
	};
	const ROW = {
		id: 'g1',
		measuredOn: '2026-11-01',
		weight: { grams: 3728, ozHundredths: 13150, typedUnit: 'lb' },
		length: { mm: 495, inHundredths: 1949, typedUnit: 'cm' },
		head: null,
		note: null,
		ageDays: 40,
		correctedDays: 7,
		scores: {
			weight: { scored: true, z: 0.21, percentile: 58.2 },
			length: { scored: false, reason: 'not_covered' },
			head: null,
			bmi: { scored: false, reason: 'before_term', value: 15.21 }
		}
	};

	it('lists measurements and reads one', async () => {
		const f = respond({ growth: [ROW_WIRE] });
		expect(await api(f).listGrowth(KID)).toEqual({ ok: true, data: [ROW] });
		expect(sent(f)).toEqual({
			url: 'http://api:9210/v1/kids/k-1/growth',
			method: 'GET',
			body: undefined
		});
		const g = respond(ROW_WIRE);
		expect(await api(g).getGrowth(KID, 'g1')).toEqual({ ok: true, data: ROW });
		expect(sent(g).url).toBe('http://api:9210/v1/kids/k-1/growth/g1');
	});

	it('creates with the body as given; a refusal is invalid with the API field', async () => {
		const input = { measured_on: '2026-11-01', weight: { unit: 'kg', amount: 3200 } };
		const f = respond(ROW_WIRE, 201);
		expect(await api(f).createGrowth(KID, input)).toEqual({ ok: true, data: ROW });
		expect(sent(f)).toEqual({
			url: 'http://api:9210/v1/kids/k-1/growth',
			method: 'POST',
			body: input
		});
		expect(
			await api(respond({ error: 'invalid', field: 'measured_on' }, 422)).createGrowth(KID, input)
		).toEqual({ ok: false, status: 422, reason: 'invalid', code: 'invalid', field: 'measured_on' });
		expect(await api(respond({ error: 'limit' }, 409)).createGrowth(KID, input)).toMatchObject({
			reason: 'conflict',
			code: 'limit'
		});
	});

	it('updates with PATCH and deletes', async () => {
		const f = respond(ROW_WIRE);
		await api(f).updateGrowth(KID, 'g1', { length: null });
		expect(sent(f)).toEqual({
			url: 'http://api:9210/v1/kids/k-1/growth/g1',
			method: 'PATCH',
			body: { length: null }
		});
		const d = respond(null, 204);
		expect(await api(d).deleteGrowth(KID, 'g1')).toEqual({ ok: true, data: null });
		expect(sent(d)).toMatchObject({
			url: 'http://api:9210/v1/kids/k-1/growth/g1',
			method: 'DELETE'
		});
	});

	it('an id outside the pattern is not_found without a request', async () => {
		const f = respond(ROW_WIRE);
		const a = api(f);
		for (const r of [
			await a.listGrowth('../x'),
			await a.getGrowth(KID, '../x'),
			await a.getGrowth('k/1', 'g1'),
			await a.createGrowth('', {}),
			await a.updateGrowth(KID, 'g 1', {}),
			await a.deleteGrowth(KID, '..'),
			await a.curves('../x', { measure: 'weight' })
		]) {
			expect(r).toEqual(NOT_FOUND);
		}
		expect(f).not.toHaveBeenCalled();
	});

	it.each([
		['a fractional gram', { weight: { ...ROW_WIRE.weight, grams: 3.5 } }],
		['an unknown typed unit', { length: { ...ROW_WIRE.length, typed_unit: 'mm' } }],
		['a missing age', { age: undefined }],
		['a fractional corrected age', { age: { days: 40, corrected_days: 1.5 } }],
		[
			'a scored score without a z',
			{ scores: { ...ROW_WIRE.scores, weight: { scored: true, percentile: 5 } } }
		],
		[
			'an unknown reason',
			{ scores: { ...ROW_WIRE.scores, length: { scored: false, reason: 'low' } } }
		],
		[
			'a bmi without its value',
			{ scores: { ...ROW_WIRE.scores, bmi: { scored: false, reason: 'not_covered' } } }
		],
		['a missing scores block', { scores: undefined }],
		[
			'a string percentile',
			{ scores: { ...ROW_WIRE.scores, weight: { scored: true, z: 0, percentile: '5' } } }
		]
	])('maps a row with %s to error', async (_name, bad) => {
		expect(await api(respond({ growth: [{ ...ROW_WIRE, ...bad }] })).listGrowth(KID)).toMatchObject(
			{
				ok: false,
				reason: 'error'
			}
		);
	});

	const CURVES_WIRE = {
		source: 'who',
		measure: 'weight',
		sex: 'male',
		covered: true,
		reason: null,
		from_months: 0,
		to_months: 24,
		lines: [
			{
				percentile: 50,
				points: [
					{ age_months: 0, metric: 3.3, imperial: 7.28 },
					{ age_months: 24.5, metric: 12.2, imperial: 26.9, gap: true }
				]
			}
		],
		corrected_age_axis: true
	};
	const CURVES = {
		source: 'who',
		measure: 'weight',
		sex: 'male',
		covered: true,
		reason: null,
		fromMonths: 0,
		toMonths: 24,
		lines: [
			{
				percentile: 50,
				points: [
					{ ageMonths: 0, metric: 3.3, imperial: 7.28, gap: false },
					{ ageMonths: 24.5, metric: 12.2, imperial: 26.9, gap: true }
				]
			}
		],
		correctedAgeAxis: true
	};

	it('reads curves with the measure and the month range in the query', async () => {
		const f = respond(CURVES_WIRE);
		expect(await api(f).curves(KID, { measure: 'weight', fromMonths: 0, toMonths: 24 })).toEqual({
			ok: true,
			data: CURVES
		});
		expect(sent(f).url).toBe(
			'http://api:9210/v1/kids/k-1/growth/curves?measure=weight&from_months=0&to_months=24'
		);
		const g = respond({
			...CURVES_WIRE,
			measure: 'bmi',
			covered: false,
			reason: 'not_covered',
			lines: []
		});
		expect(await api(g).curves(KID, { measure: 'bmi' })).toMatchObject({
			ok: true,
			data: { covered: false, reason: 'not_covered', lines: [] }
		});
		expect(sent(g).url).toBe('http://api:9210/v1/kids/k-1/growth/curves?measure=bmi');
	});

	it.each(['../..', '', 'Weight', 'height', 'bmi/x', 'weight&x=1'])(
		'curves for measure %j answers not_found without a request',
		async (measure) => {
			const f = respond(CURVES_WIRE);
			expect(await api(f).curves(KID, { measure: measure as never })).toEqual(NOT_FOUND);
			expect(f).not.toHaveBeenCalled();
		}
	);

	it.each([
		['an unknown source', { source: 'uk-who' }],
		[
			'a point without a metric',
			{ lines: [{ percentile: 50, points: [{ age_months: 0, imperial: null }] }] }
		],
		[
			'a string imperial',
			{ lines: [{ percentile: 50, points: [{ age_months: 0, metric: 1, imperial: '2' }] }] }
		],
		['a missing corrected axis', { corrected_age_axis: undefined }]
	])('maps curves with %s to error', async (_name, bad) => {
		expect(
			await api(respond({ ...CURVES_WIRE, ...bad })).curves(KID, { measure: 'weight' })
		).toMatchObject({ ok: false, reason: 'error' });
	});

	const SOURCE_WIRE = {
		id: 'cdc',
		name: 'CDC 2000 Growth Charts',
		attribution: 'Source: CDC',
		licence: 'Public domain',
		version: '2000',
		origin_url: 'https://www.cdc.gov/growthcharts/data/zscore/',
		retrieved: '2026-11-22',
		coverage: [{ measure: 'bmi', from_months: 24, to_months: 240.5 }]
	};

	it('lists the growth sources', async () => {
		const f = respond({ sources: [SOURCE_WIRE] });
		expect(await api(f).growthSources()).toEqual({
			ok: true,
			data: [
				{
					id: 'cdc',
					name: 'CDC 2000 Growth Charts',
					attribution: 'Source: CDC',
					licence: 'Public domain',
					version: '2000',
					originUrl: 'https://www.cdc.gov/growthcharts/data/zscore/',
					retrieved: '2026-11-22',
					coverage: [{ measure: 'bmi', fromMonths: 24, toMonths: 240.5 }]
				}
			]
		});
		expect(sent(f).url).toBe('http://api:9210/v1/growth-sources');
		for (const bad of [
			{ id: 'uk-who' },
			{ coverage: [{ measure: 'height', from_months: 0, to_months: 1 }] }
		]) {
			expect(
				await api(respond({ sources: [{ ...SOURCE_WIRE, ...bad }] })).growthSources()
			).toMatchObject({
				ok: false,
				reason: 'error'
			});
		}
	});

	it('logs a growth call by its endpoint name, never the kid, the row or the body', async () => {
		await api(respond({ error: 'invalid', field: 'note' }, 422)).updateGrowth(KID, 'g-secret', {
			note: 'private note'
		});
		expect(JSON.parse(String(logSpy.mock.calls[0][0]))).toEqual({
			level: 'error',
			msg: 'api call failed',
			endpoint: 'update_growth',
			status: 422
		});
		expect(logged()).not.toContain('g-secret');
		expect(logged()).not.toContain('private note');
	});
});
