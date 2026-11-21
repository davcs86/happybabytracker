import { randomBytes } from 'node:crypto';
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
export const CLEO = { username: 'cleo', password: 'fake-cleo-password-0000', displayName: 'Cleo' };
export const DAN = { username: 'dan', password: 'fake-dan-password-0000', displayName: 'Dan' };

// Ana guards Mia alone and is Cleo's caregiver for Leo; Dan has no kids.
export const MIA = {
	id: 'k-mia',
	name: 'Mia',
	birthDate: '2026-07-08',
	timeZone: 'UTC'
} as const;
export const LEO = {
	id: 'k-leo',
	name: 'Leo',
	birthDate: '2024-08-20',
	timeZone: 'Europe/Madrid'
} as const;

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

export const SCENARIOS = [
	'normal',
	'down',
	'unavailable',
	'slow',
	'rate_limited',
	'leave_unavailable',
	'leave_rate_limited',
	'leave_forbidden'
] as const;
export type Scenario = (typeof SCENARIOS)[number];
// Longer than the UI's 5 s API timeout.
const SLOW_MS = 6000;
const RETRY_AFTER_S = '60';
const PARENT_TTL_S = 30 * 24 * 3600;
const ADMIN_TTL_S = 12 * 3600;
const INVITE_TTL_S = 72 * 3600;
const RESET_TTL_S = 3600;
const MIN_PASSWORD = 12;
const MAX_PASSWORD = 256;
// The API's limits (constants.go), mirrored for the stub only.
const USERNAME_PATTERN = /^[a-z0-9._-]{3,32}$/;
const DISPLAY_NAME = { min: 1, max: 64 };
const KID_NAME = { min: 1, max: 64 };
const KID_MIN_BIRTH_DATE = '1900-01-01';
const WEEKS = { min: 22, max: 44 };
const DAYS = { min: 0, max: 6 };
const UNIT_SETS = {
	weight: ['kg', 'lb'],
	length: ['cm', 'in'],
	volume: ['ml', 'floz'],
	temp: ['c', 'f']
} as const;

export type LoggedRequest = {
	method: string;
	path: string;
	authorized: boolean;
	session: string | null;
	clientAddress: string | null;
};

type Units = { weight: string; length: string; volume: string; temp: string };
type Role = 'guardian' | 'caregiver';
type Account = {
	id: string;
	username: string;
	displayName: string;
	password: string | null;
	disabled: boolean;
	units: Units;
	createdAt: number;
};
type SessionRow = {
	id: string;
	kind: 'admin' | 'parent';
	parentId: string | null;
	expiresAt: number;
	createdAt: number;
	lastSeenAt: number;
};
type TokenRow = { kind: 'invite' | 'reset'; parentId: string; usable: boolean };
type KidRow = {
	id: string;
	name: string;
	birthDate: string;
	sex: string;
	gestationWeeks: number | null;
	gestationDays: number | null;
	timeZone: string;
	seq: number;
};
type Grant = { kidId: string; parentId: string; role: Role };

type State = {
	scenario: Scenario;
	requests: LoggedRequest[];
	parents: Map<string, Account>;
	sessions: Map<string, SessionRow>;
	tokens: Map<string, TokenRow>;
	kids: Map<string, KidRow>;
	grants: Grant[];
	nextId: number;
};

const iso = (unixS: number) => new Date(unixS * 1000).toISOString().replace(/\.\d{3}Z$/, 'Z');
const now = () => Math.floor(Date.now() / 1000);
const DEFAULT_UNITS: Units = { weight: 'kg', length: 'cm', volume: 'ml', temp: 'c' };

function account(id: string, username: string, displayName: string, password: string | null) {
	return {
		id,
		username,
		displayName,
		password,
		disabled: false,
		units: { ...DEFAULT_UNITS },
		createdAt: now()
	};
}

function freshState(): Omit<State, 'scenario' | 'requests'> {
	const ana: Account = account('p-ana', ANA.username, ANA.displayName, ANA.password);
	const bo: Account = account('p-bo', INVITEE, 'Bo', null);
	const cleo: Account = account('p-cleo', CLEO.username, CLEO.displayName, CLEO.password);
	const dan: Account = account('p-dan', DAN.username, DAN.displayName, DAN.password);
	const dee: Account = { ...account('p-dee', 'dee', 'Dee', null), disabled: true };
	const mia: KidRow = {
		...MIA,
		sex: 'female',
		gestationWeeks: null,
		gestationDays: null,
		seq: 1
	};
	const leo: KidRow = { ...LEO, sex: 'male', gestationWeeks: 35, gestationDays: 2, seq: 2 };
	return {
		parents: new Map([ana, bo, cleo, dan, dee].map((p) => [p.id, p])),
		sessions: new Map(),
		// Used, expired and disabled-parent links all answer 404, as the API does.
		tokens: new Map<string, TokenRow>([
			[TOKENS.invite, { kind: 'invite', parentId: bo.id, usable: true }],
			[TOKENS.reset, { kind: 'reset', parentId: ana.id, usable: true }],
			[TOKENS.used, { kind: 'invite', parentId: bo.id, usable: false }],
			[TOKENS.expired, { kind: 'reset', parentId: ana.id, usable: false }],
			[TOKENS.disabledParent, { kind: 'invite', parentId: dee.id, usable: false }]
		]),
		kids: new Map([mia, leo].map((k) => [k.id, k])),
		grants: [
			{ kidId: mia.id, parentId: ana.id, role: 'guardian' },
			{ kidId: leo.id, parentId: cleo.id, role: 'guardian' },
			{ kidId: leo.id, parentId: ana.id, role: 'caregiver' }
		],
		nextId: 1
	};
}

const nextId = (state: State, prefix: string) =>
	`${prefix}-${String(state.nextId++).padStart(12, '0')}`;

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

// respondNoKid: one body for an unknown kid and for no grant.
const noKid = (res: ServerResponse) => error(res, 404, 'not_found');

async function readBody(req: IncomingMessage): Promise<string> {
	let data = '';
	for await (const chunk of req) data += chunk;
	return data;
}

type FieldKind = 'string' | 'int' | 'units';
type Body = Record<string, unknown>;

// decodeJSON in respond.go: unknown key (also nested) -> 422 unknown_field, a wrong type or bad JSON -> 400.
async function decodeBody(
	req: IncomingMessage,
	res: ServerResponse,
	fields: Record<string, FieldKind>
): Promise<Body | null> {
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
	const isObject = (v: unknown) => typeof v === 'object' && v !== null && !Array.isArray(v);
	for (const [k, v] of Object.entries(value)) {
		const kind = fields[k];
		if (!kind) {
			error(res, 422, 'unknown_field', k);
			return null;
		}
		if (v === null) continue;
		const typed =
			kind === 'string'
				? typeof v === 'string'
				: kind === 'int'
					? Number.isInteger(v)
					: isObject(v) &&
						Object.values(v as Body).every((u) => u === null || typeof u === 'string');
		if (kind === 'units' && isObject(v)) {
			const unknown = Object.keys(v as Body).find((u) => !(u in UNIT_SETS));
			if (unknown) {
				error(res, 422, 'unknown_field', unknown);
				return null;
			}
		}
		if (!typed) {
			error(res, 400, 'bad_request');
			return null;
		}
	}
	return value as Body;
}

// A null JSON string leaves a Go string at "".
async function decode(
	req: IncomingMessage,
	res: ServerResponse,
	keys: string[]
): Promise<Record<string, string> | null> {
	const body = await decodeBody(req, res, Object.fromEntries(keys.map((k) => [k, 'string'])));
	if (!body) return null;
	return Object.fromEntries(Object.entries(body).map(([k, v]) => [k, (v as string | null) ?? '']));
}

const header = (req: IncomingMessage, name: string) => {
	const v = req.headers[name];
	return typeof v === 'string' ? v : null;
};
const codePoints = (s: string) => [...s].length;

function newSession(state: State, kind: 'admin' | 'parent', parentId: string | null) {
	const token = `stub-session-${String(state.nextId++).padStart(30, '0')}`;
	const t = now();
	const expiresAt = t + (kind === 'admin' ? ADMIN_TTL_S : PARENT_TTL_S);
	const id = nextId(state, 's');
	state.sessions.set(token, { id, kind, parentId, expiresAt, createdAt: t, lastSeenAt: t });
	return { token, kind, expires_at: iso(expiresAt) };
}

function parentView(p: Account) {
	return { id: p.id, username: p.username, display_name: p.displayName, units: p.units };
}

function adminParentView(p: Account) {
	return {
		id: p.id,
		username: p.username,
		display_name: p.displayName,
		disabled: p.disabled,
		has_password: p.password !== null,
		created_at: iso(p.createdAt)
	};
}

function kidView(k: KidRow, role: Role) {
	return {
		id: k.id,
		name: k.name,
		birth_date: k.birthDate,
		sex: k.sex,
		gestation_weeks: k.gestationWeeks,
		gestation_days: k.gestationDays,
		time_zone: k.timeZone,
		role
	};
}

// auth.NormalizeUsername: ASCII only, trimmed, lower-cased, then the pattern.
function normalizeUsername(s: string): string | null {
	if ([...s].some((c) => c.charCodeAt(0) >= 0x80)) return null;
	const u = s.trim().toLowerCase();
	return USERNAME_PATTERN.test(u) ? u : null;
}

function cleanDisplayName(s: string): string | null {
	const t = s.trim();
	const n = codePoints(t);
	return n >= DISPLAY_NAME.min && n <= DISPLAY_NAME.max ? t : null;
}

function todayIn(timeZone: string): string {
	const p = new Intl.DateTimeFormat('en-CA', {
		timeZone,
		year: 'numeric',
		month: '2-digit',
		day: '2-digit'
	}).formatToParts(new Date());
	const part = (t: string) => p.find((x) => x.type === t)?.value;
	return `${part('year')}-${part('month')}-${part('day')}`;
}

// kidLocation: "" and "Local" are refused; anything time.LoadLocation cannot load is refused.
function validZone(name: string): boolean {
	if (name === '' || name === 'Local' || /^[+-]/.test(name)) return false;
	try {
		new Intl.DateTimeFormat('en-US', { timeZone: name });
		return true;
	} catch {
		return false;
	}
}

function validDate(s: string): boolean {
	const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s);
	if (!m) return false;
	const d = new Date(Date.UTC(+m[1], +m[2] - 1, +m[3]));
	return d.getUTCFullYear() === +m[1] && d.getUTCMonth() === +m[2] - 1 && d.getUTCDate() === +m[3];
}

const inRange = (n: number | null, r: { min: number; max: number }) =>
	n !== null && n >= r.min && n <= r.max;

// invalidKidField in kids.go, same order: name, birth_date, sex, gestation_weeks, gestation_days, time_zone.
function invalidKidField(k: KidRow): string {
	const n = codePoints(k.name);
	if (n < KID_NAME.min || n > KID_NAME.max) return 'name';
	const zoneOK = validZone(k.timeZone);
	if (
		!validDate(k.birthDate) ||
		k.birthDate < KID_MIN_BIRTH_DATE ||
		(zoneOK && k.birthDate > todayIn(k.timeZone))
	)
		return 'birth_date';
	if (!['female', 'male'].includes(k.sex)) return 'sex';
	const w = k.gestationWeeks;
	const d = k.gestationDays;
	if ((w === null && d !== null) || (w !== null && !inRange(w, WEEKS))) return 'gestation_weeks';
	if ((d === null && w !== null) || (d !== null && !inRange(d, DAYS))) return 'gestation_days';
	if (!zoneOK) return 'time_zone';
	return '';
}

// mergeOnto: absent or null strings keep the stored value; gestation null clears, absent keeps.
function mergeKid(k: KidRow, body: Body): KidRow {
	const out = { ...k };
	for (const [key, prop] of [
		['name', 'name'],
		['birth_date', 'birthDate'],
		['sex', 'sex'],
		['time_zone', 'timeZone']
	] as const) {
		if (typeof body[key] === 'string') out[prop] = body[key] as string;
	}
	if ('gestation_weeks' in body) out.gestationWeeks = body.gestation_weeks as number | null;
	if ('gestation_days' in body) out.gestationDays = body.gestation_days as number | null;
	out.name = out.name.trim();
	return out;
}

const KID_FIELDS: Record<string, FieldKind> = {
	name: 'string',
	birth_date: 'string',
	sex: 'string',
	gestation_weeks: 'int',
	gestation_days: 'int',
	time_zone: 'string'
};

const roleFor = (state: State, kidId: string, parentId: string) =>
	state.grants.find((g) => g.kidId === kidId && g.parentId === parentId)?.role;

const activeGuardians = (state: State, kidId: string) =>
	state.grants.filter(
		(g) => g.kidId === kidId && g.role === 'guardian' && !state.parents.get(g.parentId)?.disabled
	).length;

function issueLink(state: State, kind: TokenRow['kind'], parentId: string) {
	const token = randomBytes(32).toString('base64url');
	state.tokens.set(token, { kind, parentId, usable: true });
	const ttl = kind === 'invite' ? INVITE_TTL_S : RESET_TTL_S;
	return { url: `${UI_ORIGIN}/${kind}/${token}`, expiresAt: iso(now() + ttl) };
}

type Access = 'session' | 'admin' | 'parent' | 'caregiver' | 'guardian';
type Principal = { token: string; session: SessionRow; parentId: string; role: Role | null };

// admit in middleware.go: no session 401, wrong kind 403, no grant 404, caregiver on a guardian route 403.
function admit(
	state: State,
	res: ServerResponse,
	token: string | null,
	access: Access,
	kidId?: string
): Principal | null {
	const session = token ? state.sessions.get(token) : undefined;
	if (!token || !session || session.expiresAt <= now()) {
		error(res, 401, 'no_session');
		return null;
	}
	const perKid = access === 'caregiver' || access === 'guardian';
	if (
		(access === 'admin' && session.kind !== 'admin') ||
		((access === 'parent' || perKid) && session.kind !== 'parent')
	) {
		error(res, 403, 'forbidden');
		return null;
	}
	const parentId = session.parentId ?? '';
	let role: Role | null = null;
	if (perKid) {
		role = roleFor(state, kidId ?? '', parentId) ?? null;
		if (!role || !state.kids.has(kidId ?? '')) {
			noKid(res);
			return null;
		}
		if (access === 'guardian' && role !== 'guardian') {
			error(res, 403, 'forbidden');
			return null;
		}
	}
	if (session.kind === 'parent') session.lastSeenAt = now();
	return { token, session, parentId, role };
}

type Handler = (
	state: State,
	req: IncomingMessage,
	res: ServerResponse,
	p: Principal,
	params: string[]
) => unknown;
type RouteDef = {
	method: string;
	pattern: RegExp;
	access: Access;
	kidParam?: boolean;
	run: Handler;
};

const routes: RouteDef[] = [
	{
		method: 'GET',
		pattern: /^\/v1\/admin\/parents$/,
		access: 'admin',
		run: (state, _req, res) => {
			const rows = [...state.parents.values()].sort((a, b) => a.username.localeCompare(b.username));
			send(res, 200, { parents: rows.map(adminParentView) });
		}
	},
	{
		method: 'POST',
		pattern: /^\/v1\/admin\/parents$/,
		access: 'admin',
		run: async (state, req, res) => {
			const body = await decode(req, res, ['username', 'display_name']);
			if (!body) return;
			const username = normalizeUsername(body.username ?? '');
			if (!username || username === ADMIN.username) return error(res, 422, 'invalid', 'username');
			const name = cleanDisplayName(body.display_name ?? '');
			if (name === null) return error(res, 422, 'invalid', 'display_name');
			if ([...state.parents.values()].some((p) => p.username === username)) {
				return error(res, 409, 'username_taken');
			}
			const parent = account(nextId(state, 'p'), username, name, null);
			state.parents.set(parent.id, parent);
			const link = issueLink(state, 'invite', parent.id);
			send(res, 201, {
				parent: adminParentView(parent),
				invite_url: link.url,
				expires_at: link.expiresAt
			});
		}
	},
	{
		method: 'POST',
		pattern: /^\/v1\/admin\/parents\/([^/]+)\/reset-link$/,
		access: 'admin',
		run: (state, _req, res, _p, [id]) => {
			const parent = state.parents.get(id);
			if (!parent) return error(res, 404, 'not_found');
			const kind = parent.password === null ? 'invite' : 'reset';
			const link = issueLink(state, kind, parent.id);
			send(res, 200, { [`${kind}_url`]: link.url, expires_at: link.expiresAt });
		}
	},
	{
		method: 'POST',
		pattern: /^\/v1\/admin\/parents\/([^/]+)\/(disable|enable)$/,
		access: 'admin',
		run: (state, _req, res, _p, [id, action]) => {
			const parent = state.parents.get(id);
			if (!parent) return error(res, 404, 'not_found');
			if (action === 'enable') {
				parent.disabled = false;
				return send(res, 204);
			}
			const guarded = state.grants.filter((g) => g.parentId === id && g.role === 'guardian');
			parent.disabled = true;
			if (guarded.some((g) => activeGuardians(state, g.kidId) === 0)) {
				parent.disabled = false;
				// The admin learns nothing about which kid.
				return send(res, 409, { error: 'sole_guardian' });
			}
			for (const [t, s] of state.sessions) if (s.parentId === id) state.sessions.delete(t);
			send(res, 204);
		}
	},
	{
		method: 'PATCH',
		pattern: /^\/v1\/me$/,
		access: 'parent',
		run: async (state, req, res, p) => {
			const body = await decodeBody(req, res, { display_name: 'string', units: 'units' });
			if (!body) return;
			const parent = state.parents.get(p.parentId);
			if (!parent) return error(res, 401, 'no_session');
			let name = parent.displayName;
			if (typeof body.display_name === 'string') {
				const clean = cleanDisplayName(body.display_name);
				if (clean === null) return error(res, 422, 'invalid', 'display_name');
				name = clean;
			}
			const units = { ...parent.units };
			const patch = (body.units ?? {}) as Record<string, string | null>;
			for (const key of ['weight', 'length', 'volume', 'temp'] as const) {
				const v = patch[key];
				if (v === undefined || v === null) continue;
				if (!(UNIT_SETS[key] as readonly string[]).includes(v)) {
					return error(res, 422, 'invalid', `units.${key}`);
				}
				units[key] = v;
			}
			parent.displayName = name;
			parent.units = units;
			send(res, 200, parentView(parent));
		}
	},
	{
		method: 'POST',
		pattern: /^\/v1\/me\/password$/,
		access: 'parent',
		run: async (state, req, res, p) => {
			const body = await decode(req, res, ['current', 'new']);
			if (!body) return;
			const parent = state.parents.get(p.parentId);
			if (!parent || parent.password === null || parent.password !== body.current) {
				return error(res, 403, 'wrong_password');
			}
			const n = codePoints(body.new ?? '');
			if (n < MIN_PASSWORD || n > MAX_PASSWORD) return error(res, 422, 'invalid', 'new');
			parent.password = body.new;
			for (const [t, s] of state.sessions) {
				if (s.parentId === parent.id && t !== p.token) state.sessions.delete(t);
			}
			send(res, 204);
		}
	},
	{
		method: 'GET',
		pattern: /^\/v1\/me\/sessions$/,
		access: 'parent',
		run: (state, _req, res, p) => {
			const rows = [...state.sessions.values()]
				.filter((s) => s.parentId === p.parentId && s.expiresAt > now())
				.sort((a, b) => b.createdAt - a.createdAt || b.id.localeCompare(a.id));
			send(res, 200, {
				sessions: rows.map((s) => ({
					id: s.id,
					created_at: iso(s.createdAt),
					last_seen_at: iso(s.lastSeenAt),
					current: s.id === p.session.id
				}))
			});
		}
	},
	{
		method: 'DELETE',
		pattern: /^\/v1\/me\/sessions\/([^/]+)$/,
		access: 'parent',
		run: (state, _req, res, p, [id]) => {
			for (const [t, s] of state.sessions) {
				if (s.id === id && s.parentId === p.parentId) {
					state.sessions.delete(t);
					return send(res, 204);
				}
			}
			error(res, 404, 'not_found');
		}
	},
	{
		method: 'GET',
		pattern: /^\/v1\/kids$/,
		access: 'parent',
		run: (state, _req, res, p) => {
			const rows = state.grants
				.filter((g) => g.parentId === p.parentId)
				.map((g) => ({ kid: state.kids.get(g.kidId), role: g.role }))
				.filter((r): r is { kid: KidRow; role: Role } => r.kid !== undefined)
				.sort((a, b) => a.kid.seq - b.kid.seq);
			send(res, 200, { kids: rows.map((r) => kidView(r.kid, r.role)) });
		}
	},
	{
		method: 'POST',
		pattern: /^\/v1\/kids$/,
		access: 'parent',
		run: async (state, req, res, p) => {
			const body = await decodeBody(req, res, KID_FIELDS);
			if (!body) return;
			const empty: KidRow = {
				id: '',
				name: '',
				birthDate: '',
				sex: '',
				gestationWeeks: null,
				gestationDays: null,
				timeZone: '',
				seq: 0
			};
			const kid = mergeKid(empty, body);
			const field = invalidKidField(kid);
			if (field) return error(res, 422, 'invalid', field);
			kid.id = nextId(state, 'k');
			kid.seq = state.nextId++;
			state.kids.set(kid.id, kid);
			state.grants.push({ kidId: kid.id, parentId: p.parentId, role: 'guardian' });
			send(res, 201, kidView(kid, 'guardian'));
		}
	},
	{
		method: 'GET',
		pattern: /^\/v1\/kids\/([^/]+)$/,
		access: 'caregiver',
		kidParam: true,
		run: (state, _req, res, p, [id]) => {
			const kid = state.kids.get(id);
			if (!kid) return noKid(res);
			send(res, 200, kidView(kid, p.role ?? 'caregiver'));
		}
	},
	{
		method: 'PATCH',
		pattern: /^\/v1\/kids\/([^/]+)$/,
		access: 'guardian',
		kidParam: true,
		run: async (state, req, res, p, [id]) => {
			const body = await decodeBody(req, res, KID_FIELDS);
			if (!body) return;
			const stored = state.kids.get(id);
			if (!stored) return noKid(res);
			const kid = mergeKid(stored, body);
			const field = invalidKidField(kid);
			if (field) return error(res, 422, 'invalid', field);
			state.kids.set(id, kid);
			send(res, 200, kidView(kid, p.role ?? 'guardian'));
		}
	},
	{
		method: 'DELETE',
		pattern: /^\/v1\/kids\/([^/]+)$/,
		access: 'guardian',
		kidParam: true,
		run: async (state, req, res, _p, [id]) => {
			const body = await decode(req, res, ['confirm_name']);
			if (!body) return;
			const kid = state.kids.get(id);
			if (!kid) return noKid(res);
			if ((body.confirm_name ?? '') !== kid.name) {
				return error(res, 422, 'invalid', 'confirm_name');
			}
			state.kids.delete(id);
			state.grants = state.grants.filter((g) => g.kidId !== id);
			send(res, 204);
		}
	},
	{
		method: 'GET',
		pattern: /^\/v1\/kids\/([^/]+)\/access$/,
		access: 'guardian',
		kidParam: true,
		run: (state, _req, res, _p, [id]) => {
			const rows = state.grants
				.filter((g) => g.kidId === id)
				.map((g) => ({ g, parent: state.parents.get(g.parentId)! }))
				.sort(
					(a, b) =>
						Number(a.g.role !== 'guardian') - Number(b.g.role !== 'guardian') ||
						a.parent.username.localeCompare(b.parent.username)
				);
			send(res, 200, {
				grants: rows.map(({ g, parent }) => ({
					parent_id: parent.id,
					username: parent.username,
					display_name: parent.displayName,
					role: g.role
				}))
			});
		}
	},
	{
		method: 'PUT',
		pattern: /^\/v1\/kids\/([^/]+)\/access\/([^/]+)$/,
		access: 'guardian',
		kidParam: true,
		run: async (state, req, res, _p, [id, rawUsername]) => {
			const body = await decode(req, res, ['role']);
			if (!body) return;
			const role = body.role;
			if (role !== 'guardian' && role !== 'caregiver') return error(res, 422, 'invalid', 'role');
			const username = normalizeUsername(rawUsername);
			const target = [...state.parents.values()].find(
				(p) => p.username === username && !p.disabled
			);
			if (!username || !target) return error(res, 404, 'parent_not_found');
			const before = state.grants.map((g) => ({ ...g }));
			const existing = state.grants.find((g) => g.kidId === id && g.parentId === target.id);
			if (existing) existing.role = role;
			else state.grants.push({ kidId: id, parentId: target.id, role });
			if (activeGuardians(state, id) === 0) {
				state.grants = before;
				return error(res, 409, 'last_guardian');
			}
			send(res, 204);
		}
	},
	{
		method: 'DELETE',
		pattern: /^\/v1\/kids\/([^/]+)\/access\/([^/]+)$/,
		access: 'caregiver',
		kidParam: true,
		run: (state, _req, res, p, [id, target]) => {
			if (target !== p.parentId && p.role !== 'guardian') return error(res, 403, 'forbidden');
			const before = state.grants;
			const kept = state.grants.filter((g) => !(g.kidId === id && g.parentId === target));
			if (kept.length === before.length) return noKid(res);
			state.grants = kept;
			if (activeGuardians(state, id) === 0) {
				state.grants = before;
				return error(res, 409, 'last_guardian');
			}
			send(res, 204);
		}
	}
];

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
		if (
			!parent ||
			parent.disabled ||
			parent.password === null ||
			parent.password !== body.password
		) {
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
		live.lastSeenAt = now();
		return send(res, 200, {
			kind: 'parent',
			expires_at: iso(live.expiresAt),
			parent: parentView(parent)
		});
	}

	const tokenMatch = /^\/v1\/tokens\/([^/]+)(\/redeem)?$/.exec(path);
	if (tokenMatch) {
		const row = state.tokens.get(decodeURIComponent(tokenMatch[1]));
		const usable = row?.usable && !state.parents.get(row.parentId)?.disabled ? row : undefined;
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
			const length = codePoints(body.password ?? '');
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

	for (const r of routes) {
		const m = r.method === method ? r.pattern.exec(path) : null;
		if (!m) continue;
		const params = m.slice(1).map((s) => decodeURIComponent(s));
		const p = admit(state, res, sessionToken, r.access, r.kidParam ? params[0] : undefined);
		if (!p) return;
		// Failures of one access removal, for the leave action's error paths.
		if (method === 'DELETE' && r.pattern.source.includes('access')) {
			if (state.scenario === 'leave_unavailable') return error(res, 503, 'unavailable');
			if (state.scenario === 'leave_forbidden') return error(res, 403, 'forbidden');
			if (state.scenario === 'leave_rate_limited') {
				return send(res, 429, { error: 'rate_limited' }, { 'retry-after': RETRY_AFTER_S });
			}
		}
		return r.run(state, req, res, p, params);
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

// A second session for a parent, as another device would open it.
export async function openSession(username: string, password: string): Promise<string> {
	const res = await fetch(`${STUB_URL}/v1/session`, {
		method: 'POST',
		headers: { authorization: `Bearer ${STUB_TOKEN}`, 'content-type': 'application/json' },
		body: JSON.stringify({ username, password })
	});
	if (res.status !== 200) throw new Error(`openSession: ${res.status}`);
	return ((await res.json()) as { token: string }).token;
}

if (process.argv[1] && resolve(process.argv[1]) === import.meta.filename) startStub();
