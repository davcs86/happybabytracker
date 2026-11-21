import { createServer, type IncomingMessage, type ServerResponse } from 'node:http';
import { resolve } from 'node:path';

// A stub keeps UI tests green on a broken contract: scripts/smoke.sh (real API + real UI) is the final check.

export const STUB_PORT = 4181;
export const STUB_URL = `http://127.0.0.1:${STUB_PORT}`;
export const UI_PORT = 4173;
export const UI_ORIGIN = `http://127.0.0.1:${UI_PORT}`;
export const STUB_TOKEN = 'fake-internal-api-token-000000000000';
export const ADMIN = { username: 'admin', password: 'fake-admin-password-0000' };
export const ANA = { username: 'ana', password: 'fake-ana-password-0000', displayName: 'Ana' };

// One-time tokens, 43 base64url characters like the API's.
const tok = (c: string) => c.repeat(43);
export const TOKENS = {
	invite: tok('i'),
	reset: tok('r'),
	used: tok('u'),
	expired: tok('e'),
	disabledParent: tok('d'),
	unknown: tok('x')
} as const;
export const INVITEE = 'bo';

export const SCENARIOS = ['normal', 'down', 'unavailable', 'slow', 'rate_limited'] as const;
export type Scenario = (typeof SCENARIOS)[number];
// Longer than the UI's 5 s API timeout.
const SLOW_MS = 6000;
const RETRY_AFTER_S = '60';
const PARENT_TTL_S = 30 * 24 * 3600;
const ADMIN_TTL_S = 12 * 3600;
const MIN_PASSWORD = 12;
const MAX_PASSWORD = 256;

export type LoggedRequest = {
	method: string;
	path: string;
	authorized: boolean;
	session: string | null;
	clientAddress: string | null;
};

type Account = { id: string; username: string; displayName: string; password: string | null };
type SessionRow = { kind: 'admin' | 'parent'; parentId: string | null; expiresAt: number };
type TokenRow = { kind: 'invite' | 'reset'; parentId: string; usable: boolean };

type State = {
	scenario: Scenario;
	requests: LoggedRequest[];
	parents: Map<string, Account>;
	sessions: Map<string, SessionRow>;
	tokens: Map<string, TokenRow>;
	nextSession: number;
};

function freshState(): Omit<State, 'scenario' | 'requests'> {
	const ana: Account = { id: 'p-ana', ...ANA };
	const bo: Account = { id: 'p-bo', username: INVITEE, displayName: 'Bo', password: null };
	const dee: Account = { id: 'p-dee', username: 'dee', displayName: 'Dee', password: null };
	return {
		parents: new Map([ana, bo, dee].map((p) => [p.id, p])),
		sessions: new Map(),
		// Used, expired and disabled-parent links all answer 404, as the API does.
		tokens: new Map<string, TokenRow>([
			[TOKENS.invite, { kind: 'invite', parentId: bo.id, usable: true }],
			[TOKENS.reset, { kind: 'reset', parentId: ana.id, usable: true }],
			[TOKENS.used, { kind: 'invite', parentId: bo.id, usable: false }],
			[TOKENS.expired, { kind: 'reset', parentId: ana.id, usable: false }],
			[TOKENS.disabledParent, { kind: 'invite', parentId: dee.id, usable: false }]
		]),
		nextSession: 1
	};
}

function send(res: ServerResponse, status: number, body?: unknown, headers = {}) {
	if (body === undefined) {
		res.writeHead(status, headers);
		return res.end();
	}
	res.writeHead(status, { 'content-type': 'application/json', ...headers });
	res.end(JSON.stringify(body));
}

const error = (res: ServerResponse, status: number, code: string, field?: string) =>
	send(res, status, field === undefined ? { error: code } : { error: code, field });

async function readBody(req: IncomingMessage): Promise<string> {
	let data = '';
	for await (const chunk of req) data += chunk;
	return data;
}

// decodeJSON in respond.go: unknown key -> 422 unknown_field, anything malformed -> 400 bad_request.
async function decode(
	req: IncomingMessage,
	res: ServerResponse,
	keys: string[]
): Promise<Record<string, string> | null> {
	let value: unknown;
	try {
		value = JSON.parse(await readBody(req));
	} catch {
		error(res, 400, 'bad_request');
		return null;
	}
	if (typeof value !== 'object' || value === null || Array.isArray(value)) {
		error(res, 400, 'bad_request');
		return null;
	}
	const out: Record<string, string> = {};
	for (const [k, v] of Object.entries(value)) {
		if (!keys.includes(k)) {
			error(res, 422, 'unknown_field', k);
			return null;
		}
		if (typeof v !== 'string') {
			error(res, 400, 'bad_request');
			return null;
		}
		out[k] = v;
	}
	return out;
}

const iso = (unixS: number) => new Date(unixS * 1000).toISOString().replace(/\.\d{3}Z$/, 'Z');
const now = () => Math.floor(Date.now() / 1000);
const header = (req: IncomingMessage, name: string) => {
	const v = req.headers[name];
	return typeof v === 'string' ? v : null;
};

function newSession(state: State, kind: 'admin' | 'parent', parentId: string | null) {
	const token = `stub-session-${String(state.nextSession++).padStart(30, '0')}`;
	const expiresAt = now() + (kind === 'admin' ? ADMIN_TTL_S : PARENT_TTL_S);
	state.sessions.set(token, { kind, parentId, expiresAt });
	return { token, kind, expires_at: iso(expiresAt) };
}

function parentView(p: Account) {
	return {
		id: p.id,
		username: p.username,
		display_name: p.displayName,
		units: { weight: 'kg', length: 'cm', volume: 'ml', temp: 'c' }
	};
}

async function route(state: State, req: IncomingMessage, res: ServerResponse, url: URL) {
	const method = req.method ?? 'GET';
	const path = url.pathname;
	if (method === 'GET' && path === '/healthz') return send(res, 200, { status: 'ok' });
	if (req.headers.authorization !== `Bearer ${STUB_TOKEN}`) return error(res, 401, 'unauthorized');

	const sessionToken = header(req, 'x-session-token');
	const session = sessionToken ? state.sessions.get(sessionToken) : undefined;
	const live = session && session.expiresAt > now() ? session : undefined;

	if (path === '/v1/session' && method === 'POST') {
		const body = await decode(req, res, ['username', 'password']);
		if (!body) return;
		if (state.scenario === 'rate_limited') {
			return send(res, 429, { error: 'rate_limited' }, { 'retry-after': RETRY_AFTER_S });
		}
		const username = (body.username ?? '').trim().toLowerCase();
		if (username === ADMIN.username && body.password === ADMIN.password) {
			return send(res, 200, newSession(state, 'admin', null));
		}
		const parent = [...state.parents.values()].find((p) => p.username === username);
		if (!parent || parent.password === null || parent.password !== body.password) {
			return error(res, 401, 'invalid_credentials');
		}
		return send(res, 200, newSession(state, 'parent', parent.id));
	}

	if (path === '/v1/session' && (method === 'GET' || method === 'DELETE')) {
		if (!live || !sessionToken) return error(res, 401, 'no_session');
		if (method === 'DELETE') {
			state.sessions.delete(sessionToken);
			return send(res, 204);
		}
		if (live.kind === 'admin')
			return send(res, 200, { kind: 'admin', expires_at: iso(live.expiresAt) });
		const parent = state.parents.get(live.parentId ?? '');
		if (!parent) return error(res, 401, 'no_session');
		return send(res, 200, {
			kind: 'parent',
			expires_at: iso(live.expiresAt),
			parent: parentView(parent)
		});
	}

	const tokenMatch = /^\/v1\/tokens\/([^/]+)(\/redeem)?$/.exec(path);
	if (tokenMatch) {
		const row = state.tokens.get(decodeURIComponent(tokenMatch[1]));
		const usable = row?.usable ? row : undefined;
		if (!tokenMatch[2] && method === 'GET') {
			if (!usable) return error(res, 404, 'not_found');
			const parent = state.parents.get(usable.parentId);
			return send(res, 200, { kind: usable.kind, username: parent?.username });
		}
		if (tokenMatch[2] && method === 'POST') {
			const body = await decode(req, res, ['password']);
			if (!body) return;
			if (state.scenario === 'rate_limited') {
				return send(res, 429, { error: 'rate_limited' }, { 'retry-after': RETRY_AFTER_S });
			}
			if (!usable) return error(res, 404, 'not_found');
			const length = [...(body.password ?? '')].length;
			if (length < MIN_PASSWORD || length > MAX_PASSWORD)
				return error(res, 422, 'invalid', 'password');
			usable.usable = false;
			const parent = state.parents.get(usable.parentId);
			if (parent) parent.password = body.password;
			for (const [t, s] of state.sessions)
				if (s.parentId === usable.parentId) state.sessions.delete(t);
			return send(res, 200, newSession(state, 'parent', usable.parentId));
		}
	}
	return error(res, 404, 'not_found');
}

export function startStub(port = STUB_PORT) {
	const state: State = { scenario: 'normal', requests: [], ...freshState() };
	const server = createServer(async (req, res) => {
		const url = new URL(req.url ?? '/', STUB_URL);
		if (url.pathname.startsWith('/__')) return control(state, req, res, url);
		state.requests.push({
			method: req.method ?? 'GET',
			path: url.pathname,
			authorized: req.headers.authorization === `Bearer ${STUB_TOKEN}`,
			session: header(req, 'x-session-token'),
			clientAddress: header(req, 'x-client-address')
		});
		if (state.scenario === 'down') return void req.socket.destroy();
		if (state.scenario === 'unavailable') return error(res, 503, 'unavailable');
		if (state.scenario === 'slow') {
			setTimeout(() => void route(state, req, res, url), SLOW_MS);
			return;
		}
		return route(state, req, res, url);
	});
	server.listen(port, '127.0.0.1');
	return server;
}

async function control(state: State, req: IncomingMessage, res: ServerResponse, url: URL) {
	if (req.method === 'POST' && url.pathname === '/__scenario') {
		const body = JSON.parse((await readBody(req)) || '{}') as { name?: string };
		if (!(SCENARIOS as readonly string[]).includes(body.name ?? '')) return send(res, 400);
		state.scenario = body.name as Scenario;
		return send(res, 204);
	}
	if (req.method === 'POST' && url.pathname === '/__reset') {
		Object.assign(state, { scenario: 'normal', requests: [], ...freshState() });
		return send(res, 204);
	}
	if (req.method === 'GET' && url.pathname === '/__requests') return send(res, 200, state.requests);
	return send(res, 404);
}

export async function setScenario(name: Scenario): Promise<void> {
	const res = await fetch(`${STUB_URL}/__scenario`, {
		method: 'POST',
		body: JSON.stringify({ name })
	});
	if (res.status !== 204) throw new Error(`setScenario ${name}: ${res.status}`);
}

export async function resetStub(): Promise<void> {
	const res = await fetch(`${STUB_URL}/__reset`, { method: 'POST' });
	if (res.status !== 204) throw new Error(`resetStub: ${res.status}`);
}

export async function receivedRequests(): Promise<LoggedRequest[]> {
	return (await (await fetch(`${STUB_URL}/__requests`)).json()) as LoggedRequest[];
}

if (process.argv[1] && resolve(process.argv[1]) === import.meta.filename) startStub();
