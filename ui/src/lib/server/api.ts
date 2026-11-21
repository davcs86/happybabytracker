import { API_TIMEOUT_MS, TOKEN_PATTERN } from './constants';
import type { Env } from './env';
import type { NewSession, Parent, Session, SessionKind, TokenInfo, Units } from '../types';

export type ApiReason =
	| 'unavailable'
	| 'unauthorized'
	| 'forbidden'
	| 'not_found'
	| 'invalid'
	| 'conflict'
	| 'rate_limited'
	| 'error';

export type ApiResult<T> =
	| { ok: true; data: T }
	| { ok: false; status: number; reason: ApiReason; code?: string; field?: string };

type Endpoint = 'health' | 'login' | 'session' | 'logout' | 'peek_token' | 'redeem_token';

// Thrown by a wire-shape parser; the call becomes reason 'error'.
class ShapeError extends Error {}

const isAbort = (e: unknown) =>
	e instanceof Error && (e.name === 'AbortError' || e.name === 'TimeoutError');

// One fixed-shape line per failure: never a URL, a header, a body or an error's text.
function logFailure(endpoint: Endpoint, status: number | 'network' | 'timeout'): void {
	console.error(JSON.stringify({ level: 'error', msg: 'api call failed', endpoint, status }));
}

function reasonFor(status: number): ApiReason {
	switch (status) {
		case 401:
			return 'unauthorized';
		case 403:
			return 'forbidden';
		case 404:
			return 'not_found';
		case 409:
			return 'conflict';
		case 422:
			return 'invalid';
		case 429:
			return 'rate_limited';
		case 502:
		case 503:
		case 504:
			return 'unavailable';
		default:
			return 'error';
	}
}

type Wire = Record<string, unknown>;

function object(v: unknown): Wire {
	if (typeof v !== 'object' || v === null || Array.isArray(v)) throw new ShapeError();
	return v as Wire;
}

function string(v: unknown): string {
	if (typeof v !== 'string') throw new ShapeError();
	return v;
}

function oneOf<T extends string>(v: unknown, values: readonly T[]): T {
	if (!values.includes(v as T)) throw new ShapeError();
	return v as T;
}

const KINDS = ['admin', 'parent'] as const satisfies readonly SessionKind[];

function toUnits(v: unknown): Units {
	const u = object(v);
	return {
		weight: oneOf(u.weight, ['kg', 'lb']),
		length: oneOf(u.length, ['cm', 'in']),
		volume: oneOf(u.volume, ['ml', 'floz']),
		temp: oneOf(u.temp, ['c', 'f'])
	};
}

function toParent(v: unknown): Parent {
	const p = object(v);
	return {
		id: string(p.id),
		username: string(p.username),
		displayName: string(p.display_name),
		units: toUnits(p.units)
	};
}

function toSession(v: unknown): Session {
	const s = object(v);
	const kind = oneOf(s.kind, KINDS);
	return kind === 'parent' ? { kind, parent: toParent(s.parent) } : { kind };
}

function toNewSession(v: unknown): NewSession {
	const s = object(v);
	return { token: string(s.token), kind: oneOf(s.kind, KINDS), expiresAt: string(s.expires_at) };
}

function toTokenInfo(v: unknown): TokenInfo {
	const t = object(v);
	return { kind: oneOf(t.kind, ['invite', 'reset']), username: string(t.username) };
}

const NOT_FOUND = { ok: false, status: 404, reason: 'not_found' } as const;

export type Api = ReturnType<typeof createApi>;

export function createApi(
	env: Pick<Env, 'internalApiToken' | 'apiBaseUrl'>,
	fetchFn: typeof fetch,
	session?: string,
	clientAddress?: string
) {
	const base = env.apiBaseUrl.replace(/\/+$/, '');

	async function call<T>(
		endpoint: Endpoint,
		method: string,
		path: string,
		parse: ((body: unknown) => T) | null,
		body?: unknown
	): Promise<ApiResult<T>> {
		const headers: Record<string, string> = {
			authorization: `Bearer ${env.internalApiToken}`,
			accept: 'application/json'
		};
		if (body !== undefined) headers['content-type'] = 'application/json';
		if (session) headers['x-session-token'] = session;
		if (clientAddress) headers['x-client-address'] = clientAddress;

		let res: Response;
		let raw: unknown = undefined;
		let parsed = true;
		try {
			res = await fetchFn(`${base}${path}`, {
				method,
				headers,
				body: body === undefined ? undefined : JSON.stringify(body),
				signal: AbortSignal.timeout(API_TIMEOUT_MS)
			});
			if (res.status !== 204) {
				raw = await res.json().catch((e: unknown) => {
					if (isAbort(e)) throw e;
					parsed = false;
				});
			}
		} catch (e) {
			logFailure(endpoint, isAbort(e) ? 'timeout' : 'network');
			return { ok: false, status: 0, reason: 'unavailable' };
		}

		if (!res.ok) {
			logFailure(endpoint, res.status);
			const failure: ApiResult<T> = {
				ok: false,
				status: res.status,
				reason: reasonFor(res.status)
			};
			const err = typeof raw === 'object' && raw !== null ? (raw as Wire) : {};
			if (typeof err.error === 'string') failure.code = err.error;
			if (res.status === 422 && typeof err.field === 'string') failure.field = err.field;
			return failure;
		}
		if (parse === null) return { ok: true, data: null as T };
		try {
			if (!parsed) throw new ShapeError();
			return { ok: true, data: parse(raw) };
		} catch {
			logFailure(endpoint, res.status);
			return { ok: false, status: res.status, reason: 'error' };
		}
	}

	const tokenPath = (token: string) =>
		TOKEN_PATTERN.test(token) ? `/v1/tokens/${token}` : undefined;

	return {
		health: () => call<null>('health', 'GET', '/healthz', null),
		login: (username: string, password: string) =>
			call('login', 'POST', '/v1/session', toNewSession, { username, password }),
		session: () => call('session', 'GET', '/v1/session', toSession),
		logout: () => call<null>('logout', 'DELETE', '/v1/session', null),
		peekToken(token: string): Promise<ApiResult<TokenInfo>> {
			const path = tokenPath(token);
			if (!path) return Promise.resolve(NOT_FOUND);
			return call('peek_token', 'GET', path, toTokenInfo);
		},
		redeemToken(token: string, password: string): Promise<ApiResult<NewSession>> {
			const path = tokenPath(token);
			if (!path) return Promise.resolve(NOT_FOUND);
			return call('redeem_token', 'POST', `${path}/redeem`, toNewSession, { password });
		}
	};
}
