import { afterEach, beforeEach, describe, expect, it, vi, type MockInstance } from 'vitest';
import { MODULES, TRACKER_KINDS } from '../constants';
import { createApi } from './api';
import type { Env } from './env';
import type { ExportInput } from '../types';

const TOKEN = 'internal-' + 'x'.repeat(40);
const SESSION = 'session-' + 's'.repeat(35);
const env: Env = {
	internalApiToken: TOKEN,
	apiBaseUrl: 'http://api:9210/',
	origin: 'https://x.test',
	port: 9212,
	oidc: null
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
					units: { weight: 'kg', length: 'cm', volume: 'ml', temp: 'c' },
					passwordChangedAt: null
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

	it('reports the reminders health status without reading a body', async () => {
		const f = mockFetch(async () => json({ status: 'ok' }));
		expect(await createApi(env, asFetch(f)).remindersHealth()).toEqual({ ok: true, data: null });
		expect(f.mock.calls[0][0]).toBe('http://api:9210/healthz/reminders');
		const bad = createApi(
			env,
			asFetch(mockFetch(async () => json({ status: 'unavailable' }, 503)))
		);
		expect(await bad.remindersHealth()).toMatchObject({ ok: false, status: 503 });
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

describe('createApi redesign data', () => {
	const act = (body: unknown) =>
		createApi(env, asFetch(mockFetch(async () => json(body))), SESSION);
	const HOME_WITH = (last: unknown) => ({
		age_months: 8,
		corrected_birth_date: null,
		band: '0-11',
		modules: [{ module: 'bottles', enabled: true, default: true, overridden: false }],
		running: { sleep: null, nursing: null },
		today: {
			date: '2026-11-20',
			bottles: { count: 0, ml: 0, floz_hundredths: 0 },
			nursing: { count: 0, seconds: 0 },
			pumping: { count: 0, ml: 0, floz_hundredths: 0 },
			diapers: { total: 0, wet: 0, dirty: 0, both: 0, dry: 0 },
			sleep: { seconds: 0 },
			meals: { count: 0 }
		},
		due: [],
		...(last === undefined ? {} : { last })
	});

	it('reads last activity and the unused link of a listed parent, and tolerates their absence', async () => {
		const base = {
			id: 'p-1',
			username: 'eve',
			display_name: 'Eve',
			disabled: false,
			has_password: false,
			created_at: '2026-11-21T00:00:00Z'
		};
		const link = {
			kind: 'reset',
			issued_at: '2026-11-25T09:00:00Z',
			expires_at: '2026-11-25T10:00:00Z'
		};
		const r = await act({
			parents: [
				{ ...base, last_active_at: '2026-11-25T08:00:00Z', link },
				{ ...base, id: 'p-2', last_active_at: null, link: null },
				{ ...base, id: 'p-3' }
			]
		}).listParents();
		expect(r).toMatchObject({
			ok: true,
			data: [
				{
					lastActiveAt: '2026-11-25T08:00:00Z',
					link: { kind: 'reset', issuedAt: link.issued_at, expiresAt: link.expires_at }
				},
				{ lastActiveAt: null, link: null },
				{ lastActiveAt: null, link: null }
			]
		});
	});

	it('reads the newest entry of each enabled tracker from home, null when it has none', async () => {
		const wire = { type: 'diaper', at: '2026-11-25T08:00:00Z', id: 'd-1', kind: 'wet', note: null };
		const r = await act(HOME_WITH({ diaper: wire, nursing: null })).home('k-1');
		expect(r).toMatchObject({
			ok: true,
			data: { last: { nursing: null, diaper: { kind: 'diapers', entry: { id: 'd-1' } } } }
		});
		const old = await act(HOME_WITH(undefined)).home('k-1');
		expect(old).toMatchObject({ ok: true, data: { last: {} } });
	});

	it('sends the device label only when there is one', async () => {
		const f = mockFetch(async () => json({ token: 't', kind: 'parent', expires_at: 'e' }));
		const api = createApi(env, asFetch(f), undefined, '1.2.3.4');
		await api.login('ana', 'pw', 'Safari on iPhone');
		await api.login('ana', 'pw');
		expect(JSON.parse(String(f.mock.calls[0][1]?.body))).toEqual({
			username: 'ana',
			password: 'pw',
			device_label: 'Safari on iPhone'
		});
		expect(JSON.parse(String(f.mock.calls[1][1]?.body))).toEqual({
			username: 'ana',
			password: 'pw'
		});
	});

	it('reads the entry count of a kid', async () => {
		const f = mockFetch(async () => json({ entries: 42 }));
		expect(await createApi(env, asFetch(f), SESSION).kidCounts('k-1')).toEqual({
			ok: true,
			data: { entries: 42 }
		});
		expect(f.mock.calls[0][0]).toBe('http://api:9210/v1/kids/k-1/counts');
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
		vaccine_schema: null,
		reminder_overdue_minutes: 60,
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
		vaccineSchema: null,
		reminderOverdueMinutes: 60,
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

	it('reads a chosen schedule and sends vaccine_schema only when the input has one', async () => {
		const f = mockFetch(async () => json({ ...KID_WIRE, vaccine_schema: 'uk' }));
		const api = createApi(env, asFetch(f), SESSION);
		const input = {
			name: 'Mia',
			birthDate: '2026-07-08',
			sex: 'female',
			gestationWeeks: null,
			gestationDays: null,
			timeZone: 'UTC'
		};
		expect(await api.getKid('k-1')).toMatchObject({ ok: true, data: { vaccineSchema: 'uk' } });
		await api.updateKid('k-1', { ...input, vaccineSchema: 'us' });
		await api.updateKid('k-1', { ...input, vaccineSchema: null });
		await api.updateKid('k-1', input);
		expect(sent(f, 1).body).toMatchObject({ vaccine_schema: 'us' });
		expect(sent(f, 2).body).toHaveProperty('vaccine_schema', null);
		expect(sent(f, 3).body).not.toHaveProperty('vaccine_schema');
	});

	it('reads the overdue minutes and sends them only when the input has them', async () => {
		const f = mockFetch(async () => json({ ...KID_WIRE, reminder_overdue_minutes: 90 }));
		const api = createApi(env, asFetch(f), SESSION);
		const input = {
			name: 'Mia',
			birthDate: '2026-07-08',
			sex: 'female',
			gestationWeeks: null,
			gestationDays: null,
			timeZone: 'UTC'
		};
		expect(await api.getKid('k-1')).toMatchObject({
			ok: true,
			data: { reminderOverdueMinutes: 90 }
		});
		await api.updateKid('k-1', { ...input, reminderOverdueMinutes: 15 });
		await api.createKid({ ...input, reminderOverdueMinutes: 15 });
		await api.updateKid('k-1', input);
		expect(sent(f, 1).body).toMatchObject({ reminder_overdue_minutes: 15 });
		expect(sent(f, 2).body).toMatchObject({ reminder_overdue_minutes: 15 });
		expect(sent(f, 3).body).not.toHaveProperty('reminder_overdue_minutes');
	});

	it.each([
		['missing', undefined],
		['null', null],
		['a string', '60'],
		['a fraction', 60.5]
	])('refuses a kid whose overdue minutes are %s', async (_name, value) => {
		const f = mockFetch(async () => json({ ...KID_WIRE, reminder_overdue_minutes: value }));
		expect(await createApi(env, asFetch(f), SESSION).getKid('k-1')).toMatchObject({
			ok: false,
			reason: 'error'
		});
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
			{ growth_source: undefined },
			{ vaccine_schema: undefined },
			{ vaccine_schema: 1 }
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
			data: [
				{ parentId: 'p-1', username: 'ana', displayName: 'Ana', role: 'guardian', disabled: false }
			]
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
					createdAt: '2026-11-21T00:00:00Z',
					lastActiveAt: null,
					link: null
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
					units: { weight: 'lb', length: 'in', volume: 'floz', temp: 'f' },
					password_changed_at: '2026-11-20T10:00:00Z'
				});
			if (u.endsWith('/sessions') && i?.method === 'GET')
				return json({
					sessions: [
						{
							id: 's-1',
							created_at: 'c',
							last_seen_at: 'l',
							current: true,
							device_label: 'Safari on iPhone'
						},
						{ id: 's-2', created_at: 'c', last_seen_at: 'l', current: false, device_label: null }
					]
				});
			return new Response(null, { status: 204 });
		});
		const api = createApi(env, asFetch(f), SESSION);
		const units = { weight: 'lb', length: 'in', volume: 'floz', temp: 'f' } as const;
		expect(await api.updateProfile('Ana B', units)).toEqual({
			ok: true,
			data: {
				id: 'p-1',
				username: 'ana',
				displayName: 'Ana B',
				units,
				passwordChangedAt: '2026-11-20T10:00:00Z'
			}
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
			data: [
				{
					id: 's-1',
					createdAt: 'c',
					lastSeenAt: 'l',
					current: true,
					deviceLabel: 'Safari on iPhone'
				},
				{ id: 's-2', createdAt: 'c', lastSeenAt: 'l', current: false, deviceLabel: null }
			]
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
				sources: [],
				note: null
			},
			entry: {
				id: 'e1',
				at: T0,
				contents: ['breast_milk', 'formula'],
				volume: VOLUME,
				sources: [],
				note: null
			}
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
				sides: [],
				note: null
			},
			entry: {
				id: 'e1',
				startedAt: T0,
				durationSeconds: 600,
				volume: VOLUME,
				side: 'both',
				sides: [],
				note: null
			}
		},
		sleep: {
			wire: { id: 'e1', started_at: T0, ended_at: null, note: null },
			entry: { id: 'e1', startedAt: T0, endedAt: null, note: null }
		},
		symptoms: {
			wire: {
				id: 'e1',
				at: T0,
				symptom: 'fever',
				severity: 'mild',
				other_text: null,
				temperature: { c_hundredths: 3840, f_hundredths: 10112, typed_unit: 'c' },
				method: 'ear',
				note: null
			},
			entry: {
				id: 'e1',
				at: T0,
				symptom: 'fever',
				severity: 'mild',
				otherText: null,
				temperature: { cHundredths: 3840, fHundredths: 10112, typedUnit: 'c' },
				method: 'ear',
				note: null
			}
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
				note: null,
				weight_before: null,
				weight_after: null,
				milk: null
			},
			entry: {
				id: 'e1',
				startedAt: T0,
				endedAt: null,
				leftSeconds: 60,
				rightSeconds: 0,
				side: 'right',
				sideSince: T1,
				note: null,
				weightBefore: null,
				weightAfter: null,
				milk: null
			}
		}
	} as const;
	const KINDS = Object.keys(ENTRIES) as (keyof typeof ENTRIES)[];

	const without = (o: Record<string, unknown>, key: string) =>
		Object.fromEntries(Object.entries(o).filter(([k]) => k !== key));
	const SOURCES_WIRE = [
		{ source: 'breast_milk', volume: { ml: 60, floz_hundredths: 203, typed_unit: 'ml' } },
		{ source: 'formula', volume: { ml: 90, floz_hundredths: 304, typed_unit: 'floz' } }
	];
	const SOURCES = SOURCES_WIRE;
	const WEIGHT_WIRE = { grams: 3400, oz_hundredths: 11993, typed_unit: 'kg' };
	const MILK_WIRE = {
		gain: { grams: 62, oz_hundredths: 219 },
		volume: { ml: 60, floz_hundredths: 204 }
	};

	describe('advanced bottles', () => {
		const wire = { ...ENTRIES.bottles.wire, sources: SOURCES_WIRE };

		it('parses the sources of a bottle in the order the API sent them', async () => {
			const f = respond({ bottles: [wire], next_cursor: null });
			expect(await api(f).tracker('bottles').list(KID)).toEqual({
				ok: true,
				data: { items: [{ ...ENTRIES.bottles.entry, sources: SOURCES }], nextCursor: null }
			});
		});

		it('creates and updates with a sources body as given and parses the answer', async () => {
			const f = respond(wire, 201);
			const body = { sources: { unit: 'ml', amounts: { breast_milk: 60, formula: 90 } } };
			const t = api(f).tracker('bottles');
			expect(await t.create(KID, body)).toMatchObject({ ok: true, data: { sources: SOURCES } });
			expect(await t.update(KID, 'e1', body)).toMatchObject({ ok: true });
			expect([sent(f, 0).body, sent(f, 1).body]).toEqual([body, body]);
		});

		it('a refused sources body names the field', async () => {
			const f = respond({ error: 'invalid', field: 'sources' }, 422);
			expect(await api(f).tracker('bottles').create(KID, { sources: {} })).toMatchObject({
				ok: false,
				reason: 'invalid',
				field: 'sources'
			});
		});

		it.each([
			['no sources key', without(wire, 'sources')],
			['sources that is not an array', { ...wire, sources: 'none' }],
			['sources null', { ...wire, sources: null }],
			['a source that is not an object', { ...wire, sources: [3] }],
			[
				'a source with an unknown name',
				{ ...wire, sources: [{ ...SOURCES_WIRE[0], source: 'juice' }] }
			],
			['a source without a name', { ...wire, sources: [{ volume: VOLUME_WIRE }] }],
			['a source without a volume', { ...wire, sources: [{ source: 'formula' }] }],
			[
				'a source with a fractional volume',
				{ ...wire, sources: [{ source: 'formula', volume: { ...VOLUME_WIRE, ml: 1.5 } }] }
			],
			[
				'a source with an unknown typed unit',
				{ ...wire, sources: [{ source: 'formula', volume: { ...VOLUME_WIRE, typed_unit: 'cup' } }] }
			]
		])('maps a bottle with %s to error', async (_name, body) => {
			const f = respond(body);
			expect(await api(f).tracker('bottles').get(KID, 'e1')).toEqual({
				ok: false,
				status: 200,
				reason: 'error'
			});
		});
	});

	describe('advanced pumping', () => {
		const SIDES_WIRE = [
			{ side: 'left', volume: { ml: 60, floz_hundredths: 203, typed_unit: 'ml' } },
			{ side: 'right', volume: { ml: 50, floz_hundredths: 169, typed_unit: 'floz' } }
		];
		const wire = { ...ENTRIES.pumping.wire, sides: SIDES_WIRE };

		it('parses the sides of a session in the order the API sent them', async () => {
			const f = respond({ pumping: [wire], next_cursor: null });
			expect(await api(f).tracker('pumping').list(KID)).toEqual({
				ok: true,
				data: { items: [{ ...ENTRIES.pumping.entry, sides: SIDES_WIRE }], nextCursor: null }
			});
		});

		it('creates and updates with a sides body as given and parses the answer', async () => {
			const f = respond(wire, 201);
			const body = { sides: { unit: 'ml', amounts: { left: 60, right: 50 } } };
			const t = api(f).tracker('pumping');
			expect(await t.create(KID, body)).toMatchObject({ ok: true, data: { sides: SIDES_WIRE } });
			expect(await t.update(KID, 'e1', body)).toMatchObject({ ok: true });
			expect([sent(f, 0).body, sent(f, 1).body]).toEqual([body, body]);
		});

		it('a refused sides body names the field', async () => {
			const f = respond({ error: 'invalid', field: 'sides' }, 422);
			expect(await api(f).tracker('pumping').create(KID, { sides: {} })).toMatchObject({
				ok: false,
				reason: 'invalid',
				field: 'sides'
			});
		});

		it.each([
			['no sides key', without(wire, 'sides')],
			['sides that is not an array', { ...wire, sides: 'none' }],
			['sides null', { ...wire, sides: null }],
			['a side that is not an object', { ...wire, sides: [3] }],
			['a side named both', { ...wire, sides: [{ ...SIDES_WIRE[0], side: 'both' }] }],
			['a side with an unknown name', { ...wire, sides: [{ ...SIDES_WIRE[0], side: 'middle' }] }],
			['a side without a name', { ...wire, sides: [{ volume: VOLUME_WIRE }] }],
			['a side without a volume', { ...wire, sides: [{ side: 'left' }] }],
			[
				'a side with a fractional volume',
				{ ...wire, sides: [{ side: 'left', volume: { ...VOLUME_WIRE, ml: 1.5 } }] }
			],
			[
				'a side with an unknown typed unit',
				{ ...wire, sides: [{ side: 'left', volume: { ...VOLUME_WIRE, typed_unit: 'cup' } }] }
			]
		])('maps a pumping session with %s to error', async (_name, body) => {
			const f = respond(body);
			expect(await api(f).tracker('pumping').get(KID, 'e1')).toEqual({
				ok: false,
				status: 200,
				reason: 'error'
			});
		});
	});

	describe('weighed nursing', () => {
		const wire = {
			...ENTRIES.nursing.wire,
			ended_at: T1,
			side: null,
			side_since: null,
			weight_before: WEIGHT_WIRE,
			weight_after: { grams: 3462, oz_hundredths: 12212, typed_unit: 'lb' },
			milk: MILK_WIRE
		};
		const entry = {
			...ENTRIES.nursing.entry,
			endedAt: T1,
			side: null,
			sideSince: null,
			weightBefore: { grams: 3400, ozHundredths: 11993, typedUnit: 'kg' },
			weightAfter: { grams: 3462, ozHundredths: 12212, typedUnit: 'lb' },
			milk: { gain: { grams: 62, ozHundredths: 219 }, volume: { ml: 60, flozHundredths: 204 } }
		};

		it('parses both weights and the estimate into camel case', async () => {
			const f = respond({ nursing: [wire], next_cursor: null });
			expect(await api(f).tracker('nursing').list(KID)).toEqual({
				ok: true,
				data: { items: [entry], nextCursor: null }
			});
		});

		it('parses one weight alone with no estimate', async () => {
			const f = respond({ ...wire, weight_after: null, milk: null });
			expect(await api(f).tracker('nursing').get(KID, 'e1')).toEqual({
				ok: true,
				data: { ...entry, weightAfter: null, milk: null }
			});
		});

		it('creates and updates with weights, and null clears one', async () => {
			const f = respond(wire, 201);
			const t = api(f).tracker('nursing');
			const body = { weight_before: { unit: 'kg', amount: 3400 }, weight_after: null };
			expect(await t.create(KID, { ended_at: T1, ...body })).toMatchObject({ ok: true });
			expect(await t.update(KID, 'e1', body)).toMatchObject({ ok: true });
			expect(sent(f, 1).body).toEqual(body);
		});

		it('a refused weight names the field', async () => {
			const f = respond({ error: 'invalid', field: 'weight_after' }, 422);
			expect(await api(f).tracker('nursing').update(KID, 'e1', { weight_after: 5 })).toMatchObject({
				ok: false,
				reason: 'invalid',
				field: 'weight_after'
			});
		});

		it.each([
			['no weight_before key', without(wire, 'weight_before')],
			['no milk key', without(wire, 'milk')],
			['a bare number for a weight', { ...wire, weight_before: 3400 }],
			[
				'a weight with fractional grams',
				{ ...wire, weight_before: { ...WEIGHT_WIRE, grams: 1.5 } }
			],
			[
				'a weight with an unknown unit',
				{ ...wire, weight_after: { ...WEIGHT_WIRE, typed_unit: 'st' } }
			],
			['a weight without ounces', { ...wire, weight_after: { grams: 3, typed_unit: 'kg' } }],
			['a milk without its gain', { ...wire, milk: { volume: MILK_WIRE.volume } }],
			['a milk without its volume', { ...wire, milk: { gain: MILK_WIRE.gain } }],
			[
				'a milk with a fractional volume',
				{ ...wire, milk: { ...MILK_WIRE, volume: { ml: 1.5, floz_hundredths: 5 } } }
			],
			['a milk that is not an object', { ...wire, milk: 'some' }]
		])('maps a nursing entry with %s to error', async (_name, body) => {
			const f = respond(body);
			expect(await api(f).tracker('nursing').get(KID, 'e1')).toEqual({
				ok: false,
				status: 200,
				reason: 'error'
			});
		});
	});

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
			['meal', 'meals'],
			['symptom', 'symptoms']
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

		it('reads a visit row with its linked counts', async () => {
			const visit = {
				id: 'v-1',
				starts_at: T0,
				duration_min: 30,
				kind: 'sick',
				status: 'cancelled',
				cancelled_at: T1,
				provider: null,
				location: null,
				provider_npi: null,
				note: null,
				outcome_notes: null,
				revision: 2,
				linked: { growth: 0, vaccinations: 1, medications: 2 }
			};
			const f = respond({ items: [{ type: 'visit', at: T0, ...visit }], next_cursor: null });
			expect(await api(f).timeline(KID)).toEqual({
				ok: true,
				data: {
					items: [
						{
							kind: 'visit',
							at: T0,
							entry: {
								id: 'v-1',
								startsAt: T0,
								durationMin: 30,
								kind: 'sick',
								status: 'cancelled',
								cancelledAt: T1,
								provider: null,
								location: null,
								providerNpi: null,
								note: null,
								outcomeNotes: null,
								revision: 2,
								linked: { growth: 0, vaccinations: 1, medications: 2 }
							}
						}
					],
					nextCursor: null
				}
			});
			for (const bad of [
				{ ...visit, linked: { growth: [], vaccinations: [], medications: [] } },
				{ ...visit, status: 'missed' }
			]) {
				const g = respond({ items: [{ type: 'visit', at: T0, ...bad }], next_cursor: null });
				expect(await api(g).timeline(KID)).toMatchObject({ ok: false, reason: 'error' });
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
			today: DAY,
			due: [{ kind: 'vaccine', from: { n: 4, unit: 'm' }, status: 'overdue', count: 3 }]
		};
		const MED_DUE_WIRE = {
			kind: 'medication',
			medication: { id: 'm-1', name: 'Ibuprofen' },
			due_at: T1,
			state: 'overdue'
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
					today: DAY,
					due: [{ kind: 'vaccine', from: { n: 4, unit: 'm' }, status: 'overdue', count: 3 }],
					last: {}
				}
			});
			expect(sent(f).url).toBe('http://api:9210/v1/kids/k-1/home');
		});

		it('reads a medication entry beside a vaccine group, in the API order', async () => {
			const f = respond({ ...HOME, due: [MED_DUE_WIRE, ...HOME.due] });
			expect(await api(f).home(KID)).toMatchObject({
				ok: true,
				data: {
					due: [
						{
							kind: 'medication',
							medication: { id: 'm-1', name: 'Ibuprofen' },
							dueAt: T1,
							state: 'overdue'
						},
						{ kind: 'vaccine', from: { n: 4, unit: 'm' }, status: 'overdue', count: 3 }
					]
				}
			});
		});

		const VISIT_DUE_WIRE = { kind: 'visit', id: 'v-1', starts_at: T1, visit_kind: 'dental' };

		it('reads a visit entry after the others, in the API order', async () => {
			const f = respond({ ...HOME, due: [MED_DUE_WIRE, ...HOME.due, VISIT_DUE_WIRE] });
			expect(await api(f).home(KID)).toMatchObject({
				ok: true,
				data: {
					due: [
						{ kind: 'medication' },
						{ kind: 'vaccine' },
						{ kind: 'visit', id: 'v-1', startsAt: T1, visitKind: 'dental' }
					]
				}
			});
		});

		it('reads a running nursing session with its weight and no estimate', async () => {
			const weighed = {
				...ENTRIES.nursing.wire,
				weight_before: WEIGHT_WIRE,
				weight_after: null,
				milk: null
			};
			const f = respond({ ...HOME, running: { sleep: null, nursing: weighed } });
			expect(await api(f).home(KID)).toMatchObject({
				ok: true,
				data: {
					running: {
						nursing: {
							weightBefore: { grams: 3400, ozHundredths: 11993, typedUnit: 'kg' },
							weightAfter: null,
							milk: null
						}
					}
				}
			});
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
			['a missing corrected birth date', { ...HOME, corrected_birth_date: undefined }],
			['a missing due list', { ...HOME, due: undefined }],
			['a due entry of an unknown kind', { ...HOME, due: [{ ...HOME.due[0], kind: 'medicine' }] }],
			['a due entry that is upcoming', { ...HOME, due: [{ ...HOME.due[0], status: 'upcoming' }] }],
			['a due entry in hours', { ...HOME, due: [{ ...HOME.due[0], from: { n: 4, unit: 'h' } }] }],
			['a fractional due count', { ...HOME, due: [{ ...HOME.due[0], count: 1.5 }] }],
			[
				'a medication entry with an unknown state',
				{ ...HOME, due: [{ ...MED_DUE_WIRE, state: 'upcoming' }] }
			],
			[
				'a medication entry without its course',
				{ ...HOME, due: [{ ...MED_DUE_WIRE, medication: null }] }
			],
			[
				'a medication entry whose course has no name',
				{ ...HOME, due: [{ ...MED_DUE_WIRE, medication: { id: 'm-1' } }] }
			],
			[
				'a medication entry without a due time',
				{ ...HOME, due: [{ ...MED_DUE_WIRE, due_at: undefined }] }
			],
			[
				'a visit entry of an unknown kind of visit',
				{ ...HOME, due: [{ ...VISIT_DUE_WIRE, visit_kind: 'checkup' }] }
			],
			['a visit entry without its id', { ...HOME, due: [{ ...VISIT_DUE_WIRE, id: undefined }] }],
			[
				'a visit entry without its start',
				{ ...HOME, due: [{ ...VISIT_DUE_WIRE, starts_at: null }] }
			]
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
		visit_id: null,
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
		visitId: null,
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

	it('reads the visit a measurement was recorded at', async () => {
		const f = respond({ ...ROW_WIRE, visit_id: 'v-1' });
		expect(await api(f).getGrowth(KID, 'g1')).toEqual({
			ok: true,
			data: { ...ROW, visitId: 'v-1' }
		});
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
		['a missing visit id', { visit_id: undefined }],
		['a numeric visit id', { visit_id: 7 }],
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

describe('createApi vaccination calls', () => {
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
	const MALFORMED = { ok: false, reason: 'error' };

	const SCHEMA_WIRE = {
		id: 'uk',
		name: 'United Kingdom (England)',
		authority: 'UKHSA',
		edition: 'From 1 September 2026',
		retrieved: '2026-11-22',
		origin_url: 'https://www.gov.uk/x',
		licence: 'OGL v3.0',
		attribution: 'Contains public sector information'
	};
	const SCHEMA = {
		id: 'uk',
		name: 'United Kingdom (England)',
		authority: 'UKHSA',
		edition: 'From 1 September 2026',
		retrieved: '2026-11-22',
		originUrl: 'https://www.gov.uk/x',
		licence: 'OGL v3.0',
		attribution: 'Contains public sector information'
	};
	const STATUSES = ['given', 'due', 'upcoming', 'overdue', 'dismissed', 'available'] as const;
	const lineWire = (status: string, i: number) => ({
		id: `hib-${i + 1}`,
		group: '6-in-1',
		antigen: 'hib',
		series: i + 1,
		kind: status === 'available' ? 'may_not_be_needed' : 'routine',
		status,
		from_date: '2026-03-01',
		until_date: '2026-04-01',
		given_on: status === 'given' ? '2026-03-02' : null,
		administration: status === 'given' ? 'a1' : null,
		dismissal: status === 'dismissed' ? { reason: 'not_needed' } : null,
		note: 'Source text'
	});
	const line = (status: string, i: number) => ({
		id: `hib-${i + 1}`,
		group: '6-in-1',
		antigen: 'hib',
		series: i + 1,
		kind: status === 'available' ? 'may_not_be_needed' : 'routine',
		status,
		fromDate: '2026-03-01',
		untilDate: '2026-04-01',
		givenOn: status === 'given' ? '2026-03-02' : null,
		administration: status === 'given' ? 'a1' : null,
		dismissal: status === 'dismissed' ? { reason: 'not_needed' } : null,
		note: 'Source text'
	});
	const VACCINES_WIRE = {
		schema: { ...SCHEMA_WIRE, omitted: ['Pregnancy'] },
		today: '2026-11-22',
		groups: [
			{
				from: { n: 8, unit: 'w' },
				from_date: '2026-03-01',
				status: 'overdue',
				lines: STATUSES.map(lineWire)
			}
		],
		also: [{ ...lineWire('available', 0), id: 'rsv-1', kind: 'supplementary' }],
		recurring: [
			{
				antigen: 'influenza',
				line_id: 'influenza-1',
				doses: [{ administration: 'a2', given_on: '2026-10-01' }]
			}
		],
		other: [
			{ administration: 'a3', antigen: 'typhoid', given_on: '2026-09-01', product: null },
			{ administration: 'a4', antigen: 'other', given_on: '2026-09-02', product: 'Travel' }
		]
	};
	const VACCINES = {
		schema: { ...SCHEMA, omitted: ['Pregnancy'] },
		today: '2026-11-22',
		groups: [
			{
				from: { n: 8, unit: 'w' },
				fromDate: '2026-03-01',
				status: 'overdue',
				lines: STATUSES.map(line)
			}
		],
		also: [{ ...line('available', 0), id: 'rsv-1', kind: 'supplementary' }],
		recurring: [
			{
				antigen: 'influenza',
				lineId: 'influenza-1',
				doses: [{ administration: 'a2', givenOn: '2026-10-01' }]
			}
		],
		other: [
			{ administration: 'a3', antigen: 'typhoid', givenOn: '2026-09-01', product: null },
			{ administration: 'a4', antigen: 'other', givenOn: '2026-09-02', product: 'Travel' }
		]
	};
	const ADMIN_WIRE = {
		id: 'a1',
		visit_id: null,
		given_on: '2026-03-02',
		antigens: ['diphtheria', 'hib'],
		lot: null,
		product: '6-in-1',
		note: null
	};
	const ADMIN = {
		id: 'a1',
		visitId: null,
		givenOn: '2026-03-02',
		antigens: ['diphtheria', 'hib'],
		lot: null,
		product: '6-in-1',
		note: null
	};

	it('reads the schedules and the catalogue', async () => {
		const f = respond({ schemas: [SCHEMA_WIRE] });
		expect(await api(f).vaccineSchemas()).toEqual({ ok: true, data: [SCHEMA] });
		expect(sent(f).url).toBe('http://api:9210/v1/vaccine-schemas');
		const catalogue = {
			antigens: [{ id: 'hib', name: 'Hib' }],
			products: [{ id: 'mmr', name: 'MMR', antigens: ['measles', 'mumps', 'rubella'] }]
		};
		const g = respond(catalogue);
		expect(await api(g).vaccineCatalogue()).toEqual({ ok: true, data: catalogue });
		expect(sent(g).url).toBe('http://api:9210/v1/vaccine-catalogue');
	});

	it("reads a kid's vaccines with every line status, and one without a schedule", async () => {
		const f = respond(VACCINES_WIRE);
		expect(await api(f).vaccines(KID)).toEqual({ ok: true, data: VACCINES });
		expect(sent(f)).toMatchObject({ url: 'http://api:9210/v1/kids/k-1/vaccines', method: 'GET' });
		const none = {
			schema: null,
			today: '2026-11-22',
			groups: [],
			also: [],
			recurring: [],
			other: []
		};
		expect(await api(respond(none)).vaccines(KID)).toEqual({ ok: true, data: none });
	});

	const group = VACCINES_WIRE.groups[0];
	it.each([
		['an unknown line status', { groups: [{ ...group, lines: [lineWire('late', 0)] }] }],
		['an unknown group status', { groups: [{ ...group, status: 'late' }] }],
		['an unknown kind', { groups: [{ ...group, lines: [{ ...lineWire('due', 0), kind: 'x' }] }] }],
		['an unknown age unit', { groups: [{ ...group, from: { n: 8, unit: 'h' } }] }],
		['a fractional age', { groups: [{ ...group, from: { n: 1.5, unit: 'm' } }] }],
		[
			'an unknown dismissal reason',
			{ groups: [{ ...group, lines: [{ ...lineWire('due', 0), dismissal: { reason: 'x' } }] }] }
		],
		['a schema without omitted', { schema: SCHEMA_WIRE }],
		['a missing recurring list', { recurring: undefined }],
		[
			'an other dose without a date',
			{ other: [{ administration: 'a', antigen: 'x', product: null }] }
		],
		['a missing today', { today: undefined }]
	])('maps vaccines with %s to error', async (_name, bad) => {
		expect(await api(respond({ ...VACCINES_WIRE, ...bad })).vaccines(KID)).toMatchObject(MALFORMED);
	});

	it('lists, reads, creates, updates and deletes administrations', async () => {
		const l = respond({ vaccinations: [ADMIN_WIRE] });
		expect(await api(l).listVaccinations(KID)).toEqual({ ok: true, data: [ADMIN] });
		expect(sent(l).url).toBe('http://api:9210/v1/kids/k-1/vaccinations');
		const g = respond(ADMIN_WIRE);
		expect(await api(g).getVaccination(KID, 'a1')).toEqual({ ok: true, data: ADMIN });
		expect(sent(g).url).toBe('http://api:9210/v1/kids/k-1/vaccinations/a1');
		const input = { given_on: '2026-03-02', antigens: ['diphtheria', 'hib'], product: '6-in-1' };
		const c = respond(ADMIN_WIRE, 201);
		expect(await api(c).createVaccination(KID, input)).toEqual({ ok: true, data: ADMIN });
		expect(sent(c)).toEqual({
			url: 'http://api:9210/v1/kids/k-1/vaccinations',
			method: 'POST',
			body: input
		});
		const u = respond(ADMIN_WIRE);
		await api(u).updateVaccination(KID, 'a1', { lot: null });
		expect(sent(u)).toEqual({
			url: 'http://api:9210/v1/kids/k-1/vaccinations/a1',
			method: 'PATCH',
			body: { lot: null }
		});
		const d = respond(null, 204);
		expect(await api(d).deleteVaccination(KID, 'a1')).toEqual({ ok: true, data: null });
		expect(sent(d)).toMatchObject({
			url: 'http://api:9210/v1/kids/k-1/vaccinations/a1',
			method: 'DELETE'
		});
	});

	it('reads the visit an administration was recorded at', async () => {
		const f = respond({ ...ADMIN_WIRE, visit_id: 'v-1' });
		expect(await api(f).getVaccination(KID, 'a1')).toEqual({
			ok: true,
			data: { ...ADMIN, visitId: 'v-1' }
		});
	});

	it('maps a refusal to its field, the cap to limit, 404 and 503', async () => {
		const input = { given_on: '2026-03-02', antigens: ['other'] };
		expect(
			await api(respond({ error: 'invalid', field: 'product' }, 422)).createVaccination(KID, input)
		).toEqual({ ok: false, status: 422, reason: 'invalid', code: 'invalid', field: 'product' });
		expect(await api(respond({ error: 'limit' }, 409)).createVaccination(KID, input)).toEqual({
			ok: false,
			status: 409,
			reason: 'conflict',
			code: 'limit'
		});
		expect(
			await api(respond({ error: 'not_found' }, 404)).updateVaccination(KID, 'a1', input)
		).toMatchObject({ ok: false, status: 404, reason: 'not_found' });
		expect(await api(respond({ error: 'unavailable' }, 503)).vaccines(KID)).toMatchObject({
			ok: false,
			reason: 'unavailable',
			code: 'unavailable'
		});
	});

	it.each([
		['a missing lot', { lot: undefined }],
		['a numeric product', { product: 3 }],
		['antigens that are not a list', { antigens: 'hib' }],
		['a missing date', { given_on: undefined }],
		['a missing visit id', { visit_id: undefined }]
	])('maps an administration with %s to error', async (_name, bad) => {
		expect(
			await api(respond({ vaccinations: [{ ...ADMIN_WIRE, ...bad }] })).listVaccinations(KID)
		).toMatchObject(MALFORMED);
		expect(await api(respond({ vaccinations: {} })).listVaccinations(KID)).toMatchObject(MALFORMED);
	});

	it('sets, clears and bulk-sets dismissals', async () => {
		const s = respond({ line: 'hib-1', reason: 'not_needed' });
		expect(await api(s).setDismissal(KID, 'hib-1', 'not_needed')).toEqual({
			ok: true,
			data: { line: 'hib-1', reason: 'not_needed' }
		});
		expect(sent(s)).toEqual({
			url: 'http://api:9210/v1/kids/k-1/vaccine-dismissals/hib-1',
			method: 'PUT',
			body: { reason: 'not_needed' }
		});
		const c = respond(null, 204);
		expect(await api(c).clearDismissal(KID, 'measles-2-b202407')).toEqual({ ok: true, data: null });
		expect(sent(c)).toMatchObject({
			url: 'http://api:9210/v1/kids/k-1/vaccine-dismissals/measles-2-b202407',
			method: 'DELETE'
		});
		const b = respond({ dismissed: 4 });
		expect(await api(b).dismissBefore(KID, '2026-01-01', 'given_unknown')).toEqual({
			ok: true,
			data: 4
		});
		expect(sent(b)).toEqual({
			url: 'http://api:9210/v1/kids/k-1/vaccine-dismissals',
			method: 'POST',
			body: { before: '2026-01-01', reason: 'given_unknown' }
		});
		expect(await api(respond({ dismissed: '4' })).dismissBefore(KID, 'x', 'y')).toMatchObject(
			MALFORMED
		);
		expect(
			await api(respond({ line: 'hib-1', reason: 'other' })).setDismissal(KID, 'hib-1', 'x')
		).toMatchObject(MALFORMED);
		expect(
			await api(respond({ error: 'invalid', field: 'reason' }, 422)).setDismissal(KID, 'hib-1', 'x')
		).toMatchObject({ reason: 'invalid', field: 'reason' });
	});

	it('an id or line outside its pattern is not_found without a request', async () => {
		const f = respond(ADMIN_WIRE);
		const a = api(f);
		for (const r of [
			await a.vaccines('../x'),
			await a.listVaccinations('k 1'),
			await a.getVaccination(KID, '../x'),
			await a.createVaccination('', {}),
			await a.updateVaccination(KID, 'a/1', {}),
			await a.deleteVaccination(KID, '..'),
			await a.setDismissal(KID, '../hib-1', 'not_needed'),
			await a.setDismissal('k/1', 'hib-1', 'not_needed'),
			await a.clearDismissal(KID, 'hib-1?x'),
			await a.clearDismissal(KID, ''),
			await a.dismissBefore('../x', '2026-01-01', 'not_needed')
		]) {
			expect(r).toEqual(NOT_FOUND);
		}
		expect(f).not.toHaveBeenCalled();
	});

	it('logs a vaccination call by its endpoint name, never the kid, the line or the body', async () => {
		await api(respond({ error: 'invalid', field: 'lot' }, 422)).updateVaccination(KID, 'a-secret', {
			lot: 'LOT-SECRET'
		});
		expect(JSON.parse(String(logSpy.mock.calls[0][0]))).toEqual({
			level: 'error',
			msg: 'api call failed',
			endpoint: 'update_vaccination',
			status: 422
		});
		await api(respond({}, 404)).setDismissal(KID, 'hib-9', 'not_needed');
		expect(JSON.parse(String(logSpy.mock.calls[1][0]))).toMatchObject({
			endpoint: 'set_dismissal'
		});
		expect(logged()).not.toContain('a-secret');
		expect(logged()).not.toContain('LOT-SECRET');
		expect(logged()).not.toContain('hib-9');
	});
});

describe('createApi symptom shapes', () => {
	const KID = 'k-1';
	const T0 = '2026-11-20T10:00:00Z';
	const FEVER = {
		id: 'e1',
		at: T0,
		symptom: 'fever',
		severity: null,
		other_text: null,
		temperature: { c_hundredths: 3840, f_hundredths: 10112, typed_unit: 'f' },
		method: 'oral',
		note: null
	};
	const COUGH = { ...FEVER, symptom: 'cough', temperature: null, method: null };
	const api = (body: unknown) =>
		createApi(env, asFetch(mockFetch(async () => json(body))), SESSION);

	it('reads a fever with its temperature and a cough without one', async () => {
		expect(await api(FEVER).tracker('symptoms').get(KID, 'e1')).toEqual({
			ok: true,
			data: {
				id: 'e1',
				at: T0,
				symptom: 'fever',
				severity: null,
				otherText: null,
				temperature: { cHundredths: 3840, fHundredths: 10112, typedUnit: 'f' },
				method: 'oral',
				note: null
			}
		});
		expect(await api(COUGH).tracker('symptoms').get(KID, 'e1')).toMatchObject({
			ok: true,
			data: { symptom: 'cough', temperature: null, method: null }
		});
	});

	it.each([
		['an unknown symptom', { ...FEVER, symptom: 'teething' }],
		['an unknown severity', { ...FEVER, severity: 'awful' }],
		['an unknown method', { ...FEVER, method: 'toe' }],
		['a missing method', { ...FEVER, method: undefined }],
		[
			'a fractional temperature',
			{ ...FEVER, temperature: { ...FEVER.temperature, c_hundredths: 1.5 } }
		],
		[
			'a temperature in kelvin',
			{ ...FEVER, temperature: { ...FEVER.temperature, typed_unit: 'k' } }
		],
		['a temperature that is not an object', { ...FEVER, temperature: 38 }],
		['a missing other text', { ...FEVER, other_text: undefined }]
	])('maps %s to error', async (_name, body) => {
		expect(await api(body).tracker('symptoms').get(KID, 'e1')).toMatchObject({
			ok: false,
			reason: 'error'
		});
	});

	it('maps a symptom list with a malformed entry to error', async () => {
		const bad = api({ symptoms: [FEVER, { ...FEVER, at: 5 }], next_cursor: null });
		expect(await bad.tracker('symptoms').list(KID)).toMatchObject({ ok: false, reason: 'error' });
	});
});

describe('createApi medication and dose calls', () => {
	const KID = 'k-1';
	const T0 = '2026-11-20T10:00:00Z';
	const T1 = '2026-11-20T20:00:00Z';
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

	const COURSE_WIRE = {
		id: 'm-1',
		name: 'Ibuprofen',
		dose: '2.5 ml',
		schedule: { kind: 'times', times: ['08:00', '20:00'] },
		start_at: T0,
		end_on: '2026-11-30',
		active: true,
		note: null,
		last_dose: { id: 'd-1', given_at: T0, amount: null },
		next_due: { due_at: T1 },
		dose_count: 3,
		visit_id: null
	};
	const COURSE = {
		id: 'm-1',
		name: 'Ibuprofen',
		dose: '2.5 ml',
		schedule: { kind: 'times', times: ['08:00', '20:00'] },
		startAt: T0,
		endOn: '2026-11-30',
		active: true,
		note: null,
		lastDose: { id: 'd-1', givenAt: T0, amount: null },
		nextDue: { dueAt: T1 },
		doseCount: 3,
		visitId: null
	};
	const DOSE_WIRE = { id: 'd-1', medication: 'm-1', given_at: T0, amount: '2.5 ml', note: null };
	const DOSE = { id: 'd-1', medication: 'm-1', givenAt: T0, amount: '2.5 ml', note: null };

	it('lists, gets, creates, updates and removes a course', async () => {
		const f = mockFetch(async (u, init) =>
			init?.method === 'DELETE'
				? new Response(null, { status: 204 })
				: json(
						init?.method === 'GET' && u.endsWith('/medications')
							? { medications: [COURSE_WIRE] }
							: COURSE_WIRE
					)
		);
		const m = api(f).medications;
		expect(await m.list(KID)).toEqual({ ok: true, data: [COURSE] });
		expect(await m.get(KID, 'm-1')).toEqual({ ok: true, data: COURSE });
		expect(await m.create(KID, { name: 'Ibuprofen' })).toEqual({ ok: true, data: COURSE });
		expect(await m.update(KID, 'm-1', { note: null })).toEqual({ ok: true, data: COURSE });
		expect(await m.remove(KID, 'm-1')).toEqual({ ok: true, data: null });
		expect([0, 1, 2, 3, 4].map((i) => sent(f, i))).toEqual([
			{ url: 'http://api:9210/v1/kids/k-1/medications', method: 'GET', body: undefined },
			{ url: 'http://api:9210/v1/kids/k-1/medications/m-1', method: 'GET', body: undefined },
			{
				url: 'http://api:9210/v1/kids/k-1/medications',
				method: 'POST',
				body: { name: 'Ibuprofen' }
			},
			{
				url: 'http://api:9210/v1/kids/k-1/medications/m-1',
				method: 'PATCH',
				body: { note: null }
			},
			{ url: 'http://api:9210/v1/kids/k-1/medications/m-1', method: 'DELETE', body: undefined }
		]);
	});

	it('reads the visit a course was recorded at', async () => {
		const f = respond({ ...COURSE_WIRE, visit_id: 'v-1' });
		expect(await api(f).medications.get(KID, 'm-1')).toEqual({
			ok: true,
			data: { ...COURSE, visitId: 'v-1' }
		});
	});

	it('reads the three schedule kinds and a course with no dose, no next due and no end', async () => {
		const bare = {
			...COURSE_WIRE,
			end_on: null,
			last_dose: null,
			next_due: null,
			dose_count: 0
		};
		for (const [schedule, want] of [
			[{ kind: 'none' }, { kind: 'none' }],
			[
				{ kind: 'every', hours: 6 },
				{ kind: 'every', hours: 6 }
			],
			[
				{ kind: 'times', times: ['08:00'] },
				{ kind: 'times', times: ['08:00'] }
			]
		] as const) {
			const r = await api(respond({ ...bare, schedule })).medications.get(KID, 'm-1');
			expect(r).toMatchObject({
				ok: true,
				data: { schedule: want, endOn: null, lastDose: null, nextDue: null, doseCount: 0 }
			});
		}
	});

	it.each([
		['an unknown schedule kind', { ...COURSE_WIRE, schedule: { kind: 'weekly' } }],
		['times without a list', { ...COURSE_WIRE, schedule: { kind: 'times' } }],
		['times that are not strings', { ...COURSE_WIRE, schedule: { kind: 'times', times: [8] } }],
		['every without hours', { ...COURSE_WIRE, schedule: { kind: 'every' } }],
		['fractional hours', { ...COURSE_WIRE, schedule: { kind: 'every', hours: 1.5 } }],
		['a missing schedule', { ...COURSE_WIRE, schedule: undefined }],
		['an active flag that is a string', { ...COURSE_WIRE, active: 'yes' }],
		['a missing dose count', { ...COURSE_WIRE, dose_count: undefined }],
		['a fractional dose count', { ...COURSE_WIRE, dose_count: 1.5 }],
		['a last dose without an id', { ...COURSE_WIRE, last_dose: { given_at: T0, amount: null } }],
		['a next due that is not an object', { ...COURSE_WIRE, next_due: T1 }],
		['a missing next due', { ...COURSE_WIRE, next_due: undefined }],
		['a missing visit id', { ...COURSE_WIRE, visit_id: undefined }]
	])('maps a course with %s to error', async (_name, body) => {
		expect(await api(respond(body)).medications.get(KID, 'm-1')).toMatchObject({
			ok: false,
			reason: 'error'
		});
	});

	it('maps a course list without its key, or with a bad course, to error', async () => {
		for (const body of [{ items: [] }, { medications: [{ ...COURSE_WIRE, name: 4 }] }]) {
			expect(await api(respond(body)).medications.list(KID)).toMatchObject({
				ok: false,
				reason: 'error'
			});
		}
	});

	it('lists doses with the query, the course filter and the cursor', async () => {
		const f = respond({ doses: [DOSE_WIRE], next_cursor: 'c_-1' });
		const r = await api(f).doses.list(KID, {
			medication: 'm-1',
			from: T0,
			to: T1,
			limit: 5,
			cursor: 'a/b'
		});
		expect(r).toEqual({ ok: true, data: { items: [DOSE], nextCursor: 'c_-1' } });
		expect(sent(f)).toEqual({
			url: `http://api:9210/v1/kids/k-1/medication-doses?from=${encodeURIComponent(T0)}&to=${encodeURIComponent(T1)}&limit=5&cursor=a%2Fb&medication=m-1`,
			method: 'GET',
			body: undefined
		});
		const all = respond({ doses: [], next_cursor: null });
		await api(all).doses.list(KID);
		expect(sent(all).url).toBe('http://api:9210/v1/kids/k-1/medication-doses');
	});

	it('gets, creates, updates and removes a dose', async () => {
		const f = mockFetch(async (_u, init) =>
			init?.method === 'DELETE' ? new Response(null, { status: 204 }) : json(DOSE_WIRE, 200)
		);
		const d = api(f).doses;
		expect(await d.get(KID, 'd-1')).toEqual({ ok: true, data: DOSE });
		expect(await d.create(KID, { medication: 'm-1' })).toEqual({ ok: true, data: DOSE });
		expect(await d.update(KID, 'd-1', { amount: null })).toEqual({ ok: true, data: DOSE });
		expect(await d.remove(KID, 'd-1')).toEqual({ ok: true, data: null });
		expect([0, 1, 2, 3].map((i) => sent(f, i))).toEqual([
			{ url: 'http://api:9210/v1/kids/k-1/medication-doses/d-1', method: 'GET', body: undefined },
			{
				url: 'http://api:9210/v1/kids/k-1/medication-doses',
				method: 'POST',
				body: { medication: 'm-1' }
			},
			{
				url: 'http://api:9210/v1/kids/k-1/medication-doses/d-1',
				method: 'PATCH',
				body: { amount: null }
			},
			{
				url: 'http://api:9210/v1/kids/k-1/medication-doses/d-1',
				method: 'DELETE',
				body: undefined
			}
		]);
	});

	it.each([
		['a missing medication', { ...DOSE_WIRE, medication: undefined }],
		['a given time that is a number', { ...DOSE_WIRE, given_at: 5 }],
		['an amount that is a number', { ...DOSE_WIRE, amount: 2.5 }],
		['a missing note', { ...DOSE_WIRE, note: undefined }]
	])('maps a dose with %s to error', async (_name, body) => {
		expect(await api(respond(body)).doses.get(KID, 'd-1')).toMatchObject({
			ok: false,
			reason: 'error'
		});
	});

	it('maps a dose list without its key to error', async () => {
		expect(await api(respond({ items: [], next_cursor: null })).doses.list(KID)).toMatchObject({
			ok: false,
			reason: 'error'
		});
	});

	it('a refused write names its field and code', async () => {
		const f = respond({ error: 'invalid', field: 'medication' }, 422);
		expect(await api(f).doses.create(KID, {})).toEqual({
			ok: false,
			status: 422,
			reason: 'invalid',
			code: 'invalid',
			field: 'medication'
		});
	});

	it('an id outside the pattern is not_found without a request', async () => {
		const f = respond({});
		const { medications, doses } = api(f);
		for (const r of [
			await medications.list('a/b'),
			await medications.get(KID, '..?x'),
			await medications.get('a/b', 'm-1'),
			await medications.create('', {}),
			await medications.update(KID, 'a b', {}),
			await medications.remove('a/b', 'm-1'),
			await doses.list('a/b'),
			await doses.list(KID, { medication: '../x' }),
			await doses.get(KID, '..?x'),
			await doses.create('', {}),
			await doses.update(KID, 'a b', {}),
			await doses.remove('a/b', 'd-1')
		]) {
			expect(r).toEqual({ ok: false, status: 404, reason: 'not_found' });
		}
		expect(f).not.toHaveBeenCalled();
	});

	it('logs a course or dose call by its endpoint name only', async () => {
		await api(respond({ error: 'invalid', field: 'name' }, 422)).medications.update(
			KID,
			'm-secret',
			{ name: 'Secret syrup' }
		);
		await api(respond({ error: 'invalid', field: 'amount' }, 422)).doses.create(KID, {
			amount: '99 ml secret'
		});
		expect(logSpy.mock.calls.map((c) => JSON.parse(String(c[0])))).toEqual([
			{ level: 'error', msg: 'api call failed', endpoint: 'update_medication', status: 422 },
			{ level: 'error', msg: 'api call failed', endpoint: 'create_dose', status: 422 }
		]);
		expect(logged()).not.toContain('m-secret');
		expect(logged()).not.toContain('Secret syrup');
		expect(logged()).not.toContain('99 ml');
	});
});

describe('createApi push device and ntfy calls', () => {
	const KID = 'k-1';
	const DEVICE = 'dev-1';
	const ENDPOINT = 'https://push.example/secret-endpoint-token';
	const P256DH = 'BP-secret-p256dh-key';
	const AUTH = 'secret-auth-key';
	const TOPIC = 'hbt-SecretTopic0123456789ab';
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
	const URL_ = 'http://api:9210';

	const DEVICE_WIRE = {
		id: DEVICE,
		label: 'Ana phone',
		created_at: '2026-11-23T10:00:00Z',
		kids: [KID]
	};
	const DEVICE_VIEW = {
		id: DEVICE,
		label: 'Ana phone',
		createdAt: '2026-11-23T10:00:00Z',
		kids: [KID]
	};
	const INPUT = { endpoint: ENDPOINT, keys: { p256dh: P256DH, auth: AUTH }, label: 'Ana phone' };

	describe('Web Push state', () => {
		it('reads {enabled:false} as off', async () => {
			const f = respond({ enabled: false });
			expect(await api(f).push.state()).toEqual({ ok: true, data: { enabled: false } });
			expect(sent(f)).toEqual({ url: `${URL_}/v1/push`, method: 'GET', body: undefined });
		});

		it('reads the public key when on', async () => {
			expect(await api(respond({ enabled: true, public_key: 'BKey-_09' })).push.state()).toEqual({
				ok: true,
				data: { enabled: true, publicKey: 'BKey-_09' }
			});
		});

		it.each([
			['no enabled', {}],
			['a string enabled', { enabled: 'yes' }],
			['on without a key', { enabled: true }],
			['on with an empty key', { enabled: true, public_key: '' }],
			['on with a numeric key', { enabled: true, public_key: 5 }]
		])('maps %s to error', async (_name, body) => {
			expect(await api(respond(body)).push.state()).toMatchObject({ ok: false, reason: 'error' });
		});

		it('maps a 503 to unavailable', async () => {
			expect(await api(respond({ error: 'unavailable' }, 503)).push.state()).toMatchObject({
				ok: false,
				status: 503,
				reason: 'unavailable'
			});
		});
	});

	describe('devices', () => {
		it('lists the parent devices with their kids', async () => {
			const f = respond({ devices: [DEVICE_WIRE, { ...DEVICE_WIRE, id: 'dev-2', kids: [] }] });
			expect(await api(f).push.devices()).toEqual({
				ok: true,
				data: [DEVICE_VIEW, { ...DEVICE_VIEW, id: 'dev-2', kids: [] }]
			});
			expect(sent(f)).toEqual({
				url: `${URL_}/v1/me/push-devices`,
				method: 'GET',
				body: undefined
			});
		});

		it.each([
			['no devices key', { items: [] }],
			['a device without kids', { devices: [{ ...DEVICE_WIRE, kids: undefined }] }],
			['a device with a numeric kid', { devices: [{ ...DEVICE_WIRE, kids: [5] }] }],
			['a device without a label', { devices: [{ ...DEVICE_WIRE, label: undefined }] }],
			['a device without a time', { devices: [{ ...DEVICE_WIRE, created_at: 5 }] }]
		])('maps %s to error', async (_name, body) => {
			expect(await api(respond(body)).push.devices()).toMatchObject({
				ok: false,
				reason: 'error'
			});
		});

		it('never returns an endpoint or a key the API should not have sent', async () => {
			const wire = { ...DEVICE_WIRE, endpoint: ENDPOINT, keys: { p256dh: P256DH, auth: AUTH } };
			const r = await api(respond({ devices: [wire] })).push.devices();
			expect(r).toEqual({ ok: true, data: [DEVICE_VIEW] });
			expect(JSON.stringify(r)).not.toContain('secret');
		});

		it('upserts with exactly the endpoint, the keys and the label in the body', async () => {
			const f = respond({ device: DEVICE_WIRE }, 201);
			const extra = { ...INPUT, expirationTime: 5, keys: { ...INPUT.keys, other: 'x' } };
			expect(await api(f).push.upsertDevice(extra)).toEqual({ ok: true, data: DEVICE_VIEW });
			expect(sent(f)).toEqual({
				url: `${URL_}/v1/me/push-devices`,
				method: 'PUT',
				body: INPUT
			});
			expect(Object.keys(sent(f).body)).toEqual(['endpoint', 'keys', 'label']);
			expect(Object.keys(sent(f).body.keys)).toEqual(['p256dh', 'auth']);
		});

		it('sends no label key when none is given', async () => {
			const f = respond({ device: DEVICE_WIRE }, 200);
			const { endpoint, keys } = INPUT;
			expect(await api(f).push.upsertDevice({ endpoint, keys })).toEqual({
				ok: true,
				data: DEVICE_VIEW
			});
			expect(Object.keys(sent(f).body)).toEqual(['endpoint', 'keys']);
		});

		it('reads the id of a device that moved (201) and of an existing one (200) alike', async () => {
			const moved = { ...DEVICE_WIRE, id: 'dev-new', kids: [] };
			expect(await api(respond({ device: moved }, 201)).push.upsertDevice(INPUT)).toEqual({
				ok: true,
				data: { ...DEVICE_VIEW, id: 'dev-new', kids: [] }
			});
			expect(await api(respond({ device: DEVICE_WIRE }, 200)).push.upsertDevice(INPUT)).toEqual({
				ok: true,
				data: DEVICE_VIEW
			});
		});

		it('maps an upsert response without its device to error', async () => {
			expect(await api(respond(DEVICE_WIRE, 201)).push.upsertDevice(INPUT)).toMatchObject({
				ok: false,
				reason: 'error'
			});
		});

		it.each(['endpoint', 'keys', 'label'])('names the refused %s field', async (field) => {
			const f = respond({ error: 'invalid', field }, 422);
			expect(await api(f).push.upsertDevice(INPUT)).toEqual({
				ok: false,
				status: 422,
				reason: 'invalid',
				code: 'invalid',
				field
			});
		});

		it('maps 409 push_unavailable to a conflict that carries the code', async () => {
			const f = respond({ error: 'push_unavailable' }, 409);
			expect(await api(f).push.upsertDevice(INPUT)).toEqual({
				ok: false,
				status: 409,
				reason: 'conflict',
				code: 'push_unavailable'
			});
		});

		it('maps 409 limit to a conflict that carries the code', async () => {
			expect(await api(respond({ error: 'limit' }, 409)).push.upsertDevice(INPUT)).toMatchObject({
				ok: false,
				reason: 'conflict',
				code: 'limit'
			});
		});

		it('renames a device with only the label', async () => {
			const f = respond({ device: { ...DEVICE_WIRE, label: 'Kitchen' } });
			expect(await api(f).push.renameDevice(DEVICE, 'Kitchen')).toEqual({
				ok: true,
				data: { ...DEVICE_VIEW, label: 'Kitchen' }
			});
			expect(sent(f)).toEqual({
				url: `${URL_}/v1/me/push-devices/${DEVICE}`,
				method: 'PATCH',
				body: { label: 'Kitchen' }
			});
			expect(
				await api(respond({ error: 'push_unavailable' }, 409)).push.renameDevice(DEVICE, 'x')
			).toMatchObject({ ok: false, reason: 'conflict', code: 'push_unavailable' });
			expect(
				await api(respond({ error: 'not_found' }, 404)).push.renameDevice(DEVICE, 'x')
			).toMatchObject({ ok: false, status: 404, reason: 'not_found' });
		});

		it('removes a device, which still works with Web Push off', async () => {
			const f = respond(null, 204);
			expect(await api(f).push.removeDevice(DEVICE)).toEqual({ ok: true, data: null });
			expect(sent(f)).toEqual({
				url: `${URL_}/v1/me/push-devices/${DEVICE}`,
				method: 'DELETE',
				body: undefined
			});
			expect(
				await api(respond({ error: 'not_found' }, 404)).push.removeDevice(DEVICE)
			).toMatchObject({ ok: false, status: 404, reason: 'not_found' });
		});

		it('opts a device in and out for a kid', async () => {
			const f = respond(null, 204);
			expect(await api(f).push.optIn(KID, DEVICE)).toEqual({ ok: true, data: null });
			expect(await api(f).push.optOut(KID, DEVICE)).toEqual({ ok: true, data: null });
			expect([0, 1].map((i) => sent(f, i))).toEqual([
				{
					url: `${URL_}/v1/kids/${KID}/push-devices/${DEVICE}`,
					method: 'PUT',
					body: undefined
				},
				{
					url: `${URL_}/v1/kids/${KID}/push-devices/${DEVICE}`,
					method: 'DELETE',
					body: undefined
				}
			]);
		});

		it('maps an opt-in refused for Web Push off, and a device or kid that is not theirs', async () => {
			expect(
				await api(respond({ error: 'push_unavailable' }, 409)).push.optIn(KID, DEVICE)
			).toMatchObject({ ok: false, status: 409, reason: 'conflict', code: 'push_unavailable' });
			expect(await api(respond({ error: 'not_found' }, 404)).push.optIn(KID, DEVICE)).toMatchObject(
				{ ok: false, status: 404, reason: 'not_found' }
			);
			expect(
				await api(respond({ error: 'not_found' }, 404)).push.optOut(KID, DEVICE)
			).toMatchObject({ ok: false, status: 404, reason: 'not_found' });
		});
	});

	describe('ntfy', () => {
		const NTFY_WIRE = { enabled: true, server: 'https://ntfy.example', topic: TOPIC, kids: [KID] };
		const NTFY_VIEW = { enabled: true, server: 'https://ntfy.example', topic: TOPIC, kids: [KID] };

		it.each([true, false])('reads off with has_topic %s, and never a topic', async (hasTopic) => {
			const f = respond({ enabled: false, has_topic: hasTopic, topic: TOPIC });
			expect(await api(f).ntfy.state()).toEqual({ ok: true, data: { enabled: false, hasTopic } });
			expect(sent(f)).toEqual({ url: `${URL_}/v1/me/ntfy`, method: 'GET', body: undefined });
		});

		it('reads the server, the topic and the kids when on', async () => {
			expect(await api(respond(NTFY_WIRE)).ntfy.state()).toEqual({ ok: true, data: NTFY_VIEW });
		});

		it('reads a null topic as turned off for this parent', async () => {
			expect(await api(respond({ ...NTFY_WIRE, topic: null, kids: [] })).ntfy.state()).toEqual({
				ok: true,
				data: { ...NTFY_VIEW, topic: null, kids: [] }
			});
		});

		it.each([
			['no enabled', {}],
			['off without has_topic', { enabled: false }],
			['off with a string has_topic', { enabled: false, has_topic: 'true' }],
			['on without a server', { ...NTFY_WIRE, server: undefined }],
			['on without a topic key', { ...NTFY_WIRE, topic: undefined }],
			['a numeric topic', { ...NTFY_WIRE, topic: 5 }],
			['on without kids', { ...NTFY_WIRE, kids: undefined }],
			['a numeric kid', { ...NTFY_WIRE, kids: [5] }]
		])('maps %s to error', async (_name, body) => {
			expect(await api(respond(body)).ntfy.state()).toMatchObject({ ok: false, reason: 'error' });
		});

		it('turns on with PUT, whether the topic is new (201) or kept (200)', async () => {
			const created = respond(NTFY_WIRE, 201);
			expect(await api(created).ntfy.enable()).toEqual({ ok: true, data: NTFY_VIEW });
			expect(sent(created)).toEqual({ url: `${URL_}/v1/me/ntfy`, method: 'PUT', body: undefined });
			expect(await api(respond(NTFY_WIRE, 200)).ntfy.enable()).toEqual({
				ok: true,
				data: NTFY_VIEW
			});
		});

		it('regenerates with POST', async () => {
			const f = respond({ ...NTFY_WIRE, topic: 'hbt-New' });
			expect(await api(f).ntfy.regenerate()).toEqual({
				ok: true,
				data: { ...NTFY_VIEW, topic: 'hbt-New' }
			});
			expect(sent(f)).toEqual({
				url: `${URL_}/v1/me/ntfy/regenerate`,
				method: 'POST',
				body: undefined
			});
			expect(await api(respond({ error: 'not_found' }, 404)).ntfy.regenerate()).toMatchObject({
				ok: false,
				status: 404,
				reason: 'not_found'
			});
		});

		it('turns off with DELETE', async () => {
			const f = respond(null, 204);
			expect(await api(f).ntfy.disable()).toEqual({ ok: true, data: null });
			expect(sent(f)).toEqual({ url: `${URL_}/v1/me/ntfy`, method: 'DELETE', body: undefined });
		});

		it('opts a kid in and out', async () => {
			const f = respond(null, 204);
			expect(await api(f).ntfy.optIn(KID)).toEqual({ ok: true, data: null });
			expect(await api(f).ntfy.optOut(KID)).toEqual({ ok: true, data: null });
			expect([0, 1].map((i) => sent(f, i))).toEqual([
				{ url: `${URL_}/v1/kids/${KID}/ntfy`, method: 'PUT', body: undefined },
				{ url: `${URL_}/v1/kids/${KID}/ntfy`, method: 'DELETE', body: undefined }
			]);
		});

		it('maps 409 ntfy_unavailable to a conflict that carries the code on every write', async () => {
			const refused = () => respond({ error: 'ntfy_unavailable' }, 409);
			for (const r of [
				await api(refused()).ntfy.enable(),
				await api(refused()).ntfy.regenerate(),
				await api(refused()).ntfy.optIn(KID)
			]) {
				expect(r).toEqual({
					ok: false,
					status: 409,
					reason: 'conflict',
					code: 'ntfy_unavailable'
				});
			}
		});

		it('maps a kid or topic that is not theirs to not_found', async () => {
			const f = () => respond({ error: 'not_found' }, 404);
			expect(await api(f()).ntfy.optIn(KID)).toMatchObject({ ok: false, reason: 'not_found' });
			expect(await api(f()).ntfy.optOut(KID)).toMatchObject({ ok: false, reason: 'not_found' });
			expect(await api(f()).ntfy.disable()).toMatchObject({ ok: false, reason: 'not_found' });
		});
	});

	it('an id outside the pattern is not_found without a request', async () => {
		const f = respond({});
		const { push, ntfy } = api(f);
		for (const r of [
			await push.renameDevice('a/b', 'x'),
			await push.removeDevice('..?x'),
			await push.optIn('a/b', DEVICE),
			await push.optIn(KID, '..?x'),
			await push.optOut('', DEVICE),
			await push.optOut(KID, 'a b'),
			await ntfy.optIn('a/b'),
			await ntfy.optOut('')
		]) {
			expect(r).toEqual({ ok: false, status: 404, reason: 'not_found' });
		}
		expect(f).not.toHaveBeenCalled();
	});

	it('logs a call by its endpoint name only: never an endpoint, key, topic or label', async () => {
		await api(respond({ error: 'invalid', field: 'keys' }, 422)).push.upsertDevice({
			...INPUT,
			label: 'Secret label'
		});
		await api(respond({ error: 'push_unavailable' }, 409)).push.optIn(KID, DEVICE);
		await api(respond({ error: 'ntfy_unavailable' }, 409)).ntfy.enable();
		await api(respond({ ...{ enabled: true, server: 's', topic: TOPIC }, kids: 5 })).ntfy.state();
		expect(logSpy.mock.calls.map((c) => JSON.parse(String(c[0])))).toEqual([
			{ level: 'error', msg: 'api call failed', endpoint: 'upsert_push_device', status: 422 },
			{ level: 'error', msg: 'api call failed', endpoint: 'set_push_optin', status: 409 },
			{ level: 'error', msg: 'api call failed', endpoint: 'enable_ntfy', status: 409 },
			{ level: 'error', msg: 'api call failed', endpoint: 'ntfy_state', status: 200 }
		]);
		for (const secret of [ENDPOINT, P256DH, AUTH, TOPIC, 'Secret label', 'secret']) {
			expect(logged()).not.toContain(secret);
		}
	});

	it('a failed upsert returns no endpoint, key or label', async () => {
		for (const f of [
			respond({ error: 'invalid', field: 'endpoint' }, 422),
			respond({ error: 'push_unavailable' }, 409),
			respond('not json', 500),
			mockFetch(async () => {
				throw new Error(`boom ${ENDPOINT} ${P256DH} ${AUTH}`);
			})
		]) {
			const r = await api(f).push.upsertDevice(INPUT);
			expect(r.ok).toBe(false);
			for (const secret of [ENDPOINT, P256DH, AUTH]) {
				expect(JSON.stringify(r)).not.toContain(secret);
				expect(logged()).not.toContain(secret);
			}
		}
	});
});

describe('createApi milestone calls', () => {
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

	const CATALOG_WIRE = {
		id: 'cdc',
		name: 'Checklists',
		authority: 'CDC',
		edition: '2022',
		retrieved: '2026-11-24',
		origin_url: 'https://www.cdc.gov/act-early/milestones/index.html',
		licence: 'Public domain',
		attribution: 'Source: CDC',
		screening_note: 'Not a screening tool.'
	};
	const VIEW_WIRE = {
		catalog: CATALOG_WIRE,
		age: { chronological_months: 3, corrected_months: 1, basis_months: 1 },
		catalog_shown: true,
		checklists: [
			{
				age_months: 2,
				position: 'next',
				items: [
					{
						id: '2m-se-a',
						domain: 'social_emotional',
						text: 'Item A',
						reached: { id: 'r-1', reached_on: '2026-11-01', note: null }
					},
					{ id: '2m-mp-b', domain: 'movement_physical', text: 'Item B', reached: null }
				]
			}
		],
		custom: [{ id: 'r-2', text: 'Waves', reached_on: '2026-10-01', note: 'park' }],
		retired: [{ id: 'r-3', item: '4m-cg-old', text: 'Old', reached_on: '2026-09-01', note: null }],
		reached: [
			{
				id: 'r-4',
				item: '60m-lc-c',
				age_months: 60,
				domain: 'language_communication',
				text: 'Item C',
				reached_on: '2026-08-01',
				note: null
			}
		]
	};
	const VIEW = {
		catalog: {
			id: 'cdc',
			name: 'Checklists',
			authority: 'CDC',
			edition: '2022',
			retrieved: '2026-11-24',
			originUrl: 'https://www.cdc.gov/act-early/milestones/index.html',
			licence: 'Public domain',
			attribution: 'Source: CDC',
			screeningNote: 'Not a screening tool.'
		},
		age: { chronologicalMonths: 3, correctedMonths: 1, basisMonths: 1 },
		catalogShown: true,
		checklists: [
			{
				ageMonths: 2,
				position: 'next',
				items: [
					{
						id: '2m-se-a',
						domain: 'social_emotional',
						text: 'Item A',
						reached: { id: 'r-1', reachedOn: '2026-11-01', note: null }
					},
					{ id: '2m-mp-b', domain: 'movement_physical', text: 'Item B', reached: null }
				]
			}
		],
		custom: [{ id: 'r-2', text: 'Waves', reachedOn: '2026-10-01', note: 'park' }],
		retired: [{ id: 'r-3', item: '4m-cg-old', text: 'Old', reachedOn: '2026-09-01', note: null }],
		reached: [
			{
				id: 'r-4',
				item: '60m-lc-c',
				ageMonths: 60,
				domain: 'language_communication',
				text: 'Item C',
				reachedOn: '2026-08-01',
				note: null
			}
		]
	};
	const ENTRY_WIRE = {
		id: 'r-1',
		item: '2m-se-a',
		text: 'Item A',
		reached_on: '2026-11-01',
		note: null
	};
	const ENTRY = { id: 'r-1', item: '2m-se-a', text: 'Item A', reachedOn: '2026-11-01', note: null };

	it('lists, gets, creates, updates and removes', async () => {
		const f = mockFetch(async (u, init) =>
			init?.method === 'DELETE'
				? new Response(null, { status: 204 })
				: json(init?.method === 'GET' && u.endsWith('/milestones') ? VIEW_WIRE : ENTRY_WIRE)
		);
		const m = api(f).milestones;
		expect(await m.list(KID)).toEqual({ ok: true, data: VIEW });
		expect(await m.get(KID, 'r-1')).toEqual({ ok: true, data: ENTRY });
		expect(await m.create(KID, { item: '2m-se-a', reached_on: '2026-11-01' })).toEqual({
			ok: true,
			data: ENTRY
		});
		expect(await m.update(KID, 'r-1', { note: null })).toEqual({ ok: true, data: ENTRY });
		expect(await m.remove(KID, 'r-1')).toEqual({ ok: true, data: null });
		expect([0, 1, 2, 3, 4].map((i) => sent(f, i))).toEqual([
			{ url: 'http://api:9210/v1/kids/k-1/milestones', method: 'GET', body: undefined },
			{ url: 'http://api:9210/v1/kids/k-1/milestones/r-1', method: 'GET', body: undefined },
			{
				url: 'http://api:9210/v1/kids/k-1/milestones',
				method: 'POST',
				body: { item: '2m-se-a', reached_on: '2026-11-01' }
			},
			{
				url: 'http://api:9210/v1/kids/k-1/milestones/r-1',
				method: 'PATCH',
				body: { note: null }
			},
			{ url: 'http://api:9210/v1/kids/k-1/milestones/r-1', method: 'DELETE', body: undefined }
		]);
	});

	it('reads a custom entry and a view past the catalog span', async () => {
		const custom = { ...ENTRY_WIRE, item: null, text: 'Waves' };
		expect(await api(respond(custom)).milestones.get(KID, 'r-1')).toEqual({
			ok: true,
			data: { ...ENTRY, item: null, text: 'Waves' }
		});
		const past = {
			...VIEW_WIRE,
			age: { chronological_months: 60, corrected_months: null, basis_months: 60 },
			catalog_shown: false,
			checklists: []
		};
		expect(await api(respond(past)).milestones.list(KID)).toMatchObject({
			ok: true,
			data: {
				age: { chronologicalMonths: 60, correctedMonths: null, basisMonths: 60 },
				catalogShown: false,
				checklists: []
			}
		});
	});

	it('a refused create names its field; a second mark and the cap are conflicts with their code', async () => {
		expect(
			await api(respond({ error: 'invalid', field: 'reached_on' }, 422)).milestones.create(KID, {})
		).toEqual({ ok: false, status: 422, reason: 'invalid', code: 'invalid', field: 'reached_on' });
		for (const code of ['already_reached', 'limit']) {
			expect(await api(respond({ error: code }, 409)).milestones.create(KID, {})).toEqual({
				ok: false,
				status: 409,
				reason: 'conflict',
				code
			});
		}
	});

	const checklist = VIEW_WIRE.checklists[0];
	const item = checklist.items[0];
	it.each([
		['no catalog', { ...VIEW_WIRE, catalog: undefined }],
		[
			'a catalog without its screening note',
			{ ...VIEW_WIRE, catalog: { ...CATALOG_WIRE, screening_note: undefined } }
		],
		[
			'a corrected age that is a string',
			{ ...VIEW_WIRE, age: { ...VIEW_WIRE.age, corrected_months: '1' } }
		],
		['a fractional basis', { ...VIEW_WIRE, age: { ...VIEW_WIRE.age, basis_months: 1.5 } }],
		['a catalog_shown that is a string', { ...VIEW_WIRE, catalog_shown: 'true' }],
		['no checklists key', { ...VIEW_WIRE, checklists: undefined }],
		[
			'a checklist age outside the catalog',
			{ ...VIEW_WIRE, checklists: [{ ...checklist, age_months: 7 }] }
		],
		['an unknown position', { ...VIEW_WIRE, checklists: [{ ...checklist, position: 'behind' }] }],
		[
			'an unknown domain',
			{ ...VIEW_WIRE, checklists: [{ ...checklist, items: [{ ...item, domain: 'fine_motor' }] }] }
		],
		[
			'a reached that is not an object',
			{ ...VIEW_WIRE, checklists: [{ ...checklist, items: [{ ...item, reached: '2026-11-01' }] }] }
		],
		[
			'a missing reached',
			{ ...VIEW_WIRE, checklists: [{ ...checklist, items: [{ ...item, reached: undefined }] }] }
		],
		[
			'a custom row without text',
			{ ...VIEW_WIRE, custom: [{ ...VIEW_WIRE.custom[0], text: undefined }] }
		],
		[
			'a retired row without its item',
			{ ...VIEW_WIRE, retired: [{ ...VIEW_WIRE.retired[0], item: null }] }
		],
		[
			'a reached row with a number date',
			{ ...VIEW_WIRE, reached: [{ ...VIEW_WIRE.reached[0], reached_on: 20260801 }] }
		],
		['no reached key', { ...VIEW_WIRE, reached: undefined }]
	])('maps a view with %s to error', async (_name, body) => {
		expect(await api(respond(body)).milestones.list(KID)).toMatchObject({
			ok: false,
			reason: 'error'
		});
	});

	it.each([
		['an item that is a number', { ...ENTRY_WIRE, item: 7 }],
		['a missing item', { ...ENTRY_WIRE, item: undefined }],
		['a missing text', { ...ENTRY_WIRE, text: undefined }],
		['a missing date', { ...ENTRY_WIRE, reached_on: undefined }],
		['a missing note', { ...ENTRY_WIRE, note: undefined }],
		['an array', [ENTRY_WIRE]]
	])('maps an entry with %s to error', async (_name, body) => {
		expect(await api(respond(body)).milestones.get(KID, 'r-1')).toMatchObject({
			ok: false,
			reason: 'error'
		});
	});

	it('maps a body that is not JSON to error', async () => {
		const f = mockFetch(async () => new Response('<html>', { status: 200 }));
		expect(await api(f).milestones.create(KID, {})).toMatchObject({ ok: false, reason: 'error' });
	});

	it('an id outside the pattern is not_found without a request', async () => {
		const f = respond({});
		const m = api(f).milestones;
		for (const r of [
			await m.list('a/b'),
			await m.get(KID, '..?x'),
			await m.get('a/b', 'r-1'),
			await m.create('', {}),
			await m.update(KID, 'a b', {}),
			await m.update('a/b', 'r-1', {}),
			await m.remove(KID, '../r-1')
		]) {
			expect(r).toEqual({ ok: false, status: 404, reason: 'not_found' });
		}
		expect(f).not.toHaveBeenCalled();
	});

	it('logs a milestone call by its endpoint name only', async () => {
		await api(respond({ error: 'invalid', field: 'text' }, 422)).milestones.update(
			KID,
			'r-secret',
			{
				text: 'Secret words'
			}
		);
		expect(logSpy.mock.calls.map((c) => JSON.parse(String(c[0])))).toEqual([
			{ level: 'error', msg: 'api call failed', endpoint: 'update_milestone', status: 422 }
		]);
		expect(logged()).not.toContain('r-secret');
		expect(logged()).not.toContain('Secret words');
	});
});

describe('createApi export', () => {
	const KID = 'k-1';
	const INPUT: ExportInput = {
		from: '2026-11-01',
		to: '2026-11-24',
		modules: ['bottles', 'growth']
	};
	const DISPOSITION = 'attachment; filename="happybabytracker-2026-11-01-2026-11-24.pdf"';
	const pdf = (headers: Record<string, string> = {}) => {
		let pulled = false;
		const body = new ReadableStream<Uint8Array>(
			{
				pull(c) {
					pulled = true;
					c.enqueue(new TextEncoder().encode('%PDF-1.4 stub'));
					c.close();
				}
			},
			{ highWaterMark: 0 }
		);
		const res = new Response(body, {
			status: 200,
			headers: {
				'content-type': 'application/pdf',
				'content-disposition': DISPOSITION,
				'cache-control': 'no-store',
				...headers
			}
		});
		return { res, pulled: () => pulled };
	};

	it('posts the range and modules as JSON and hands back the body unread, with its headers', async () => {
		const { res, pulled } = pdf();
		const jsonSpy = vi.spyOn(res, 'json');
		const f = mockFetch(async () => res);
		const r = await createApi(env, asFetch(f), SESSION).exportPdf(KID, INPUT);
		const [url, init] = f.mock.calls[0];
		expect(url).toBe('http://api:9210/v1/kids/k-1/export');
		expect(init?.method).toBe('POST');
		expect(JSON.parse(String(init?.body))).toEqual(INPUT);
		const h = new Headers(init?.headers);
		expect(h.get('authorization')).toBe(`Bearer ${TOKEN}`);
		expect(h.get('x-session-token')).toBe(SESSION);
		expect(h.get('content-type')).toBe('application/json');
		expect(r.ok).toBe(true);
		if (!r.ok) return;
		expect(r.contentType).toBe('application/pdf');
		expect(r.disposition).toBe(DISPOSITION);
		expect(jsonSpy).not.toHaveBeenCalled();
		expect(pulled()).toBe(false);
		expect(r.body).toBe(res.body);
		expect(await new Response(r.body).text()).toBe('%PDF-1.4 stub');
		expect(logSpy).not.toHaveBeenCalled();
	});

	it('aborts at EXPORT_TIMEOUT_MS (25000) while the headers have not arrived', async () => {
		vi.useFakeTimers();
		try {
			const spy = vi.spyOn(AbortSignal, 'timeout');
			const f = mockFetch(
				(_u, init) =>
					new Promise((_resolve, reject) =>
						init?.signal?.addEventListener('abort', () => reject(init.signal?.reason))
					)
			);
			let settled = false;
			const r = createApi(env, asFetch(f), SESSION)
				.exportPdf(KID, INPUT)
				.finally(() => (settled = true));
			await vi.advanceTimersByTimeAsync(24_999);
			expect(settled).toBe(false);
			await vi.advanceTimersByTimeAsync(1);
			expect(await r).toEqual({ ok: false, status: 0, reason: 'unavailable' });
			expect(spy).not.toHaveBeenCalled();
			expect(JSON.parse(String(logSpy.mock.calls[0][0]))).toMatchObject({ status: 'timeout' });
		} finally {
			vi.useRealTimers();
		}
	});

	it('stops the timer once the headers arrive, so the body streams past it', async () => {
		vi.useFakeTimers();
		try {
			const f = mockFetch(async () => pdf().res);
			const r = await createApi(env, asFetch(f), SESSION).exportPdf(KID, INPUT);
			await vi.advanceTimersByTimeAsync(2 * 25_000);
			expect(f.mock.calls[0][1]?.signal?.aborted).toBe(false);
			expect(r.ok).toBe(true);
			if (!r.ok) return;
			expect(await new Response(r.body).text()).toBe('%PDF-1.4 stub');
		} finally {
			vi.useRealTimers();
		}
	});

	it('maps a 422 to invalid with its field, and a 503 to unavailable with its code', async () => {
		const respond = (body: unknown, status: number) =>
			createApi(env, asFetch(mockFetch(async () => json(body, status))), SESSION);
		expect(
			await respond({ error: 'invalid', field: 'modules' }, 422).exportPdf(KID, INPUT)
		).toEqual({ ok: false, status: 422, reason: 'invalid', code: 'invalid', field: 'modules' });
		expect(await respond({ error: 'busy' }, 503).exportPdf(KID, INPUT)).toEqual({
			ok: false,
			status: 503,
			reason: 'unavailable',
			code: 'busy'
		});
		expect(await respond({ error: 'not_found' }, 404).exportPdf(KID, INPUT)).toEqual({
			ok: false,
			status: 404,
			reason: 'not_found',
			code: 'not_found'
		});
	});

	it('maps a network error and a timeout to unavailable', async () => {
		for (const err of [new TypeError('fetch failed'), new DOMException('t', 'TimeoutError')]) {
			const f = mockFetch(async () => {
				throw err;
			});
			expect(await createApi(env, asFetch(f), SESSION).exportPdf(KID, INPUT)).toEqual({
				ok: false,
				status: 0,
				reason: 'unavailable'
			});
		}
	});

	it.each([
		['a JSON 200', { 'content-type': 'application/json' }],
		['an HTML 200', { 'content-type': 'text/html' }],
		['an inline disposition', { 'content-disposition': 'inline' }],
		['a path in the name', { 'content-disposition': 'attachment; filename="../x.pdf"' }],
		['no disposition', { 'content-disposition': '' }]
	])('refuses %s as a shape error and never streams it', async (_name, headers) => {
		const { res, pulled } = pdf(headers);
		const r = await createApi(env, asFetch(mockFetch(async () => res)), SESSION).exportPdf(
			KID,
			INPUT
		);
		expect(r).toEqual({ ok: false, status: 200, reason: 'error' });
		expect(pulled()).toBe(false);
	});

	it('answers 404 without a call for a kid id outside ID_PATTERN', async () => {
		const f = mockFetch(async () => pdf().res);
		expect(await createApi(env, asFetch(f), SESSION).exportPdf('a/b', INPUT)).toEqual({
			ok: false,
			status: 404,
			reason: 'not_found'
		});
		expect(f).not.toHaveBeenCalled();
	});

	it('logs a fixed-shape line: never the range, a module or the kid', async () => {
		await createApi(
			env,
			asFetch(mockFetch(async () => json({ error: 'invalid', field: 'to' }, 422))),
			SESSION
		).exportPdf('k-secret', INPUT);
		await createApi(
			env,
			asFetch(
				mockFetch(async () => {
					throw new DOMException('timed out', 'TimeoutError');
				})
			),
			SESSION
		).exportPdf('k-secret', INPUT);
		expect(logSpy.mock.calls.map((c) => JSON.parse(String(c[0])))).toEqual([
			{ level: 'error', msg: 'api call failed', endpoint: 'export_pdf', status: 422 },
			{ level: 'error', msg: 'api call failed', endpoint: 'export_pdf', status: 'timeout' }
		]);
		for (const needle of ['k-secret', '2026-11-01', '2026-11-24', 'bottles', 'growth']) {
			expect(logged()).not.toContain(needle);
		}
	});
});

describe('createApi visit calls', () => {
	const KID = 'k-1';
	const T0 = '2026-11-20T09:30:00Z';
	const T1 = '2026-11-21T08:00:00Z';
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

	const BASE_WIRE = {
		id: 'v-1',
		starts_at: T0,
		duration_min: 30,
		kind: 'well_child',
		status: 'past',
		cancelled_at: null,
		provider: 'Dr. Lee',
		location: '12 Main St',
		provider_npi: '9000000015',
		note: '4-month check',
		outcome_notes: 'Healthy.',
		revision: 3
	};
	const BASE = {
		id: 'v-1',
		startsAt: T0,
		durationMin: 30,
		kind: 'well_child',
		status: 'past',
		cancelledAt: null,
		provider: 'Dr. Lee',
		location: '12 Main St',
		providerNpi: '9000000015',
		note: '4-month check',
		outcomeNotes: 'Healthy.',
		revision: 3
	};
	const VISIT_WIRE = {
		...BASE_WIRE,
		linked: {
			growth: [{ id: 'g-1', measured_on: '2026-11-20' }],
			vaccinations: [{ id: 'a-1', given_on: '2026-11-20', antigens: ['dtap', 'ipv'] }],
			medications: [{ id: 'm-1', name: 'Amoxicillin', dose_count: 14 }]
		}
	};
	const VISIT = {
		...BASE,
		linked: {
			growth: [{ id: 'g-1', measuredOn: '2026-11-20' }],
			vaccinations: [{ id: 'a-1', givenOn: '2026-11-20', antigens: ['dtap', 'ipv'] }],
			medications: [{ id: 'm-1', name: 'Amoxicillin', doseCount: 14 }]
		}
	};
	const ROW_WIRE = { ...BASE_WIRE, linked: { growth: 1, vaccinations: 1, medications: 1 } };
	const ROW = { ...BASE, linked: { growth: 1, vaccinations: 1, medications: 1 } };

	it('lists a page with its query, then gets, creates, updates, removes and records an outcome', async () => {
		const f = mockFetch(async (u, init) =>
			init?.method === 'DELETE'
				? new Response(null, { status: 204 })
				: json(
						init?.method === 'GET' && u.includes('/visits?')
							? { visits: [ROW_WIRE], next_cursor: 'n' }
							: VISIT_WIRE,
						init?.method === 'POST' && u.endsWith('/visits') ? 201 : 200
					)
		);
		const v = api(f).visits;
		expect(await v.list(KID, { from: T0, to: T1, limit: 10, cursor: 'c', order: 'asc' })).toEqual({
			ok: true,
			data: { items: [ROW], nextCursor: 'n' }
		});
		expect(await v.get(KID, 'v-1')).toEqual({ ok: true, data: VISIT });
		expect(await v.create(KID, { kind: 'sick' })).toEqual({ ok: true, data: VISIT });
		expect(await v.update(KID, 'v-1', { cancelled: true })).toEqual({ ok: true, data: VISIT });
		expect(await v.remove(KID, 'v-1', 'keep')).toEqual({ ok: true, data: null });
		expect(await v.remove(KID, 'v-1', 'delete')).toEqual({ ok: true, data: null });
		expect(await v.outcome(KID, 'v-1', { revision: 3, outcome_notes: 'Healthy.' })).toEqual({
			ok: true,
			data: VISIT
		});
		const base = 'http://api:9210/v1/kids/k-1/visits';
		expect([0, 1, 2, 3, 4, 5, 6].map((i) => sent(f, i))).toEqual([
			{
				url: `${base}?from=${encodeURIComponent(T0)}&to=${encodeURIComponent(T1)}&limit=10&cursor=c&order=asc`,
				method: 'GET',
				body: undefined
			},
			{ url: `${base}/v-1`, method: 'GET', body: undefined },
			{ url: base, method: 'POST', body: { kind: 'sick' } },
			{ url: `${base}/v-1`, method: 'PATCH', body: { cancelled: true } },
			{ url: `${base}/v-1?linked=keep`, method: 'DELETE', body: undefined },
			{ url: `${base}/v-1?linked=delete`, method: 'DELETE', body: undefined },
			{
				url: `${base}/v-1/outcome`,
				method: 'POST',
				body: { revision: 3, outcome_notes: 'Healthy.' }
			}
		]);
	});

	it('lists without a query, so the API applies its defaults', async () => {
		const f = respond({ visits: [], next_cursor: null });
		expect(await api(f).visits.list(KID)).toEqual({
			ok: true,
			data: { items: [], nextCursor: null }
		});
		expect(sent(f).url).toBe('http://api:9210/v1/kids/k-1/visits');
	});

	it('reads every kind and status, and a visit with nothing set or linked', async () => {
		for (const kind of ['well_child', 'sick', 'specialist', 'dental', 'emergency', 'other']) {
			for (const status of ['upcoming', 'past', 'cancelled']) {
				const bare = {
					...BASE_WIRE,
					kind,
					status,
					cancelled_at: status === 'cancelled' ? T1 : null,
					provider: null,
					location: null,
					provider_npi: null,
					note: null,
					outcome_notes: null,
					linked: { growth: [], vaccinations: [], medications: [] }
				};
				expect(await api(respond(bare)).visits.get(KID, 'v-1')).toEqual({
					ok: true,
					data: {
						...BASE,
						kind,
						status,
						cancelledAt: status === 'cancelled' ? T1 : null,
						provider: null,
						location: null,
						providerNpi: null,
						note: null,
						outcomeNotes: null,
						linked: { growth: [], vaccinations: [], medications: [] }
					}
				});
			}
		}
	});

	it.each([
		['an unknown kind', { kind: 'checkup' }],
		['an unknown status', { status: 'missed' }],
		['a fractional revision', { revision: 1.5 }],
		['a missing duration', { duration_min: undefined }],
		['a missing outcome note', { outcome_notes: undefined }],
		['a missing provider NPI', { provider_npi: undefined }],
		['a numeric provider NPI', { provider_npi: 9000000015 }],
		['linked counts in place of rows', { linked: { growth: 1, vaccinations: 0, medications: 0 } }],
		[
			'a linked course without its dose count',
			{ linked: { ...VISIT_WIRE.linked, medications: [{ id: 'm-1', name: 'X' }] } }
		],
		[
			'linked antigens that are not a list',
			{
				linked: {
					...VISIT_WIRE.linked,
					vaccinations: [{ id: 'a-1', given_on: '2026-11-20', antigens: 'dtap' }]
				}
			}
		]
	])('maps a visit with %s to error', async (_name, bad) => {
		expect(await api(respond({ ...VISIT_WIRE, ...bad })).visits.get(KID, 'v-1')).toMatchObject({
			ok: false,
			reason: 'error'
		});
	});

	it('maps a list row with linked rows in place of counts to error', async () => {
		const f = respond({ visits: [VISIT_WIRE], next_cursor: null });
		expect(await api(f).visits.list(KID)).toMatchObject({ ok: false, reason: 'error' });
	});

	it.each([
		['stale', 409, 'conflict', undefined],
		['not_past', 409, 'conflict', undefined],
		['limit', 409, 'conflict', undefined],
		['has_outcome', 409, 'conflict', undefined],
		['invalid', 422, 'invalid', 'growth.weight'],
		['invalid', 422, 'invalid', 'vaccinations[1].lot'],
		['invalid', 422, 'invalid', 'medications[0].schedule'],
		['invalid', 422, 'invalid', 'revision'],
		['invalid', 422, 'invalid', 'outcome'],
		['invalid', 422, 'invalid', 'outcome_notes']
	])(
		'maps an outcome refused with %s (%i, field %s) to its code',
		async (code, status, reason, field) => {
			const body = field === undefined ? { error: code } : { error: code, field };
			expect(await api(respond(body, status)).visits.outcome(KID, 'v-1', { revision: 3 })).toEqual({
				ok: false,
				status,
				reason,
				code,
				...(field === undefined ? {} : { field })
			});
		}
	);

	it('maps has_outcome on an update and limit on a create to conflict', async () => {
		expect(
			await api(respond({ error: 'has_outcome' }, 409)).visits.update(KID, 'v-1', {
				cancelled: true
			})
		).toEqual({ ok: false, status: 409, reason: 'conflict', code: 'has_outcome' });
		expect(await api(respond({ error: 'limit' }, 409)).visits.create(KID, {})).toEqual({
			ok: false,
			status: 409,
			reason: 'conflict',
			code: 'limit'
		});
	});

	it('answers not_found without a request for an id outside the pattern or an unknown linked choice', async () => {
		const f = respond(VISIT_WIRE);
		const v = api(f).visits;
		for (const r of [
			await v.list('../x'),
			await v.get(KID, '../x'),
			await v.get('k/1', 'v-1'),
			await v.create('', {}),
			await v.update(KID, 'v 1', {}),
			await v.remove(KID, '..', 'keep'),
			await v.remove(KID, 'v-1', 'all' as never),
			await v.outcome(KID, 'v/1', { revision: 0 }),
			await v.calendar(KID, '..', 'generic'),
			await v.calendar(KID, 'v-1', 'private' as never)
		]) {
			expect(r).toEqual(NOT_FOUND);
		}
		expect(f).not.toHaveBeenCalled();
	});

	it('logs a visit call by its endpoint name, never the kid, the visit or the body', async () => {
		const refuse = api(respond({ error: 'invalid', field: 'outcome_notes' }, 422)).visits;
		await refuse.outcome('k-secret', 'v-secret', { revision: 1, outcome_notes: 'private note' });
		await refuse.update('k-secret', 'v-secret', { provider: 'Dr. Secret' });
		expect(logSpy.mock.calls.map((c) => JSON.parse(String(c[0])))).toEqual([
			{ level: 'error', msg: 'api call failed', endpoint: 'visit_outcome', status: 422 },
			{ level: 'error', msg: 'api call failed', endpoint: 'update_visit', status: 422 }
		]);
		for (const needle of ['k-secret', 'v-secret', 'private note', 'Dr. Secret']) {
			expect(logged()).not.toContain(needle);
		}
	});

	describe('calendar', () => {
		const DISPOSITION = 'attachment; filename="visit-2026-11-20.ics"';
		const ics = (headers: Record<string, string> = {}) => {
			let pulled = false;
			const body = new ReadableStream<Uint8Array>(
				{
					pull(c) {
						pulled = true;
						c.enqueue(new TextEncoder().encode('BEGIN:VCALENDAR\r\n'));
						c.close();
					}
				},
				{ highWaterMark: 0 }
			);
			const res = new Response(body, {
				status: 200,
				headers: {
					'content-type': 'text/calendar; charset=utf-8',
					'content-disposition': DISPOSITION,
					'cache-control': 'no-store',
					...headers
				}
			});
			return { res, pulled: () => pulled };
		};

		it('gets the file with the detail asked for and hands back the body unread, with its headers', async () => {
			for (const detail of ['generic', 'full'] as const) {
				const { res, pulled } = ics();
				const f = mockFetch(async () => res);
				const r = await api(f).visits.calendar(KID, 'v-1', detail);
				const [url, init] = f.mock.calls[0];
				expect(url).toBe(`http://api:9210/v1/kids/k-1/visits/v-1/calendar?detail=${detail}`);
				expect(init?.method).toBe('GET');
				expect(init?.body).toBeUndefined();
				const h = new Headers(init?.headers);
				expect(h.get('authorization')).toBe(`Bearer ${TOKEN}`);
				expect(h.get('x-session-token')).toBe(SESSION);
				expect(h.get('accept')).toBe('text/calendar');
				expect(r.ok).toBe(true);
				if (!r.ok) return;
				expect(r.contentType).toBe('text/calendar; charset=utf-8');
				expect(r.disposition).toBe(DISPOSITION);
				expect(pulled()).toBe(false);
				expect(await new Response(r.body).text()).toBe('BEGIN:VCALENDAR\r\n');
			}
			expect(logSpy).not.toHaveBeenCalled();
		});

		it.each([
			['a JSON 200', { 'content-type': 'application/json' }],
			['a calendar without its charset', { 'content-type': 'text/calendar' }],
			['an inline disposition', { 'content-disposition': 'inline' }],
			['a path in the name', { 'content-disposition': 'attachment; filename="../visit.ics"' }],
			['a name in the file name', { 'content-disposition': 'attachment; filename="ivy.ics"' }],
			['no disposition', { 'content-disposition': '' }]
		])('refuses %s as a shape error and never streams it', async (_name, headers) => {
			const { res, pulled } = ics(headers);
			const r = await api(mockFetch(async () => res)).visits.calendar(KID, 'v-1', 'generic');
			expect(r).toEqual({ ok: false, status: 200, reason: 'error' });
			expect(pulled()).toBe(false);
			expect(JSON.parse(String(logSpy.mock.calls[0][0]))).toEqual({
				level: 'error',
				msg: 'api call failed',
				endpoint: 'visit_calendar',
				status: 200
			});
		});

		it('maps a refusal as call() does, and a network error to unavailable', async () => {
			expect(
				await api(respond({ error: 'not_found' }, 404)).visits.calendar(KID, 'v-1', 'full')
			).toEqual({ ok: false, status: 404, reason: 'not_found', code: 'not_found' });
			expect(
				await api(respond({ error: 'invalid', field: 'detail' }, 422)).visits.calendar(
					KID,
					'v-1',
					'full'
				)
			).toEqual({ ok: false, status: 422, reason: 'invalid', code: 'invalid', field: 'detail' });
			const down = mockFetch(async () => {
				throw new TypeError('fetch failed');
			});
			expect(await api(down).visits.calendar(KID, 'v-1', 'generic')).toEqual({
				ok: false,
				status: 0,
				reason: 'unavailable'
			});
		});
	});
});

describe('createApi provider search calls', () => {
	const sent = (f: ReturnType<typeof mockFetch>, i = 0) => ({
		url: f.mock.calls[i][0],
		method: f.mock.calls[i][1]?.method,
		body:
			f.mock.calls[i][1]?.body === undefined
				? undefined
				: JSON.parse(String(f.mock.calls[i][1]?.body))
	});
	const respond = (body: unknown, status = 200) => mockFetch(async () => json(body, status));
	const api = (f: ReturnType<typeof mockFetch>) => createApi(env, asFetch(f), SESSION);
	const RESULT_WIRE = {
		npi: '9000000015',
		name: 'AVERY QUILL',
		credential: 'MD',
		taxonomy: 'Pediatrics',
		address: '12 PINE ST, SEATTLE, WA 98101',
		phone: '206-555-0100'
	};
	const RESULT = { ...RESULT_WIRE };
	const BARE_WIRE = { ...RESULT_WIRE, credential: null, taxonomy: null, phone: null };

	it('reads whether the search is on', async () => {
		for (const enabled of [true, false]) {
			const f = respond({ search: { enabled } });
			expect(await api(f).providers.state()).toEqual({ ok: true, data: { enabled } });
			expect(sent(f)).toEqual({
				url: 'http://api:9210/v1/providers',
				method: 'GET',
				body: undefined
			});
		}
	});

	it.each([
		['no search object', {}],
		['a string flag', { search: { enabled: 'true' } }]
	])('maps a state with %s to error', async (_name, body) => {
		expect(await api(respond(body)).providers.state()).toMatchObject({
			ok: false,
			reason: 'error'
		});
	});

	it('posts a person search with its terms in the body and reads the page', async () => {
		const f = respond({ results: [RESULT_WIRE, BARE_WIRE], more: true });
		const r = await api(f).providers.search({
			kind: 'person',
			lastName: 'Quill',
			firstName: 'Av',
			state: 'WA',
			city: 'Seattle',
			postalCode: '98101'
		});
		expect(r).toEqual({
			ok: true,
			data: {
				results: [RESULT, { ...RESULT, credential: null, taxonomy: null, phone: null }],
				more: true
			}
		});
		expect(sent(f)).toEqual({
			url: 'http://api:9210/v1/providers/search',
			method: 'POST',
			body: {
				kind: 'person',
				last_name: 'Quill',
				first_name: 'Av',
				state: 'WA',
				city: 'Seattle',
				postal_code: '98101'
			}
		});
	});

	it('sends only the terms given for a clinic search, and reads an empty page', async () => {
		const f = respond({ results: [], more: false });
		expect(
			await api(f).providers.search({
				kind: 'organization',
				organization: 'Quill & Co',
				state: 'OR'
			})
		).toEqual({ ok: true, data: { results: [], more: false } });
		expect(sent(f).body).toEqual({ kind: 'organization', organization: 'Quill & Co', state: 'OR' });
	});

	it.each([
		['no more flag', { results: [] }],
		[
			'a result without an address',
			{ results: [{ ...RESULT_WIRE, address: undefined }], more: false }
		],
		['a numeric NPI', { results: [{ ...RESULT_WIRE, npi: 9000000015 }], more: false }],
		['a missing phone', { results: [{ ...RESULT_WIRE, phone: undefined }], more: false }]
	])('maps a page with %s to error', async (_name, body) => {
		expect(
			await api(respond(body)).providers.search({ kind: 'person', lastName: 'Q', state: 'WA' })
		).toMatchObject({ ok: false, reason: 'error' });
	});

	it.each([
		[429, { error: 'rate_limited' }, { reason: 'rate_limited', code: 'rate_limited' }],
		[503, { error: 'unavailable' }, { reason: 'unavailable', code: 'unavailable' }],
		[404, { error: 'not_found' }, { reason: 'not_found', code: 'not_found' }],
		[
			422,
			{ error: 'invalid', field: 'postal_code' },
			{ reason: 'invalid', code: 'invalid', field: 'postal_code' }
		]
	])('maps a search answered %i to its reason', async (status, body, failure) => {
		expect(
			await api(respond(body, status)).providers.search({
				kind: 'person',
				lastName: 'Quill',
				state: 'WA'
			})
		).toEqual({ ok: false, status, ...failure });
	});

	it('logs a refused search by its endpoint name, never a term', async () => {
		await api(respond({ error: 'unavailable' }, 503)).providers.search({
			kind: 'person',
			lastName: 'Secretname',
			state: 'WA',
			city: 'Hiddentown'
		});
		expect(logSpy.mock.calls.map((c) => JSON.parse(String(c[0])))).toEqual([
			{ level: 'error', msg: 'api call failed', endpoint: 'provider_search', status: 503 }
		]);
		for (const needle of ['Secretname', 'Hiddentown']) expect(logged()).not.toContain(needle);
	});
});
