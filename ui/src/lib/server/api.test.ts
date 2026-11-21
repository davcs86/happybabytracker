import { afterEach, beforeEach, describe, expect, it, vi, type MockInstance } from 'vitest';
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
