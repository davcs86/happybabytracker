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

// Ana guards Mia and Ivy alone and is Cleo's caregiver for Leo; Dan has no kids.
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
	'leave_forbidden',
	'patch_unavailable'
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
	// The raw JSON of a write; absent for a request without a body.
	body?: string;
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
	growthSource: string;
	vaccineSchema: string | null;
	seq: number;
};
type Grant = { kidId: string; parentId: string; role: Role };
type TrackerKey = 'bottles' | 'nursing' | 'pumping' | 'sleep' | 'diapers' | 'meals';
// view is the entry as the API answers it; every merge works on it directly.
type Row = { kidId: string; view: Body & { id: string } };

type State = {
	scenario: Scenario;
	requests: LoggedRequest[];
	parents: Map<string, Account>;
	sessions: Map<string, SessionRow>;
	tokens: Map<string, TokenRow>;
	kids: Map<string, KidRow>;
	grants: Grant[];
	entries: Record<TrackerKey, Row[]>;
	growth: GrowthStub[];
	vaccinations: AdministrationStub[];
	// kid id + ':' + schema + ':' + line id -> reason.
	dismissals: Map<string, string>;
	// kid id + ':' + module id; a row is a guardian's choice (or a band-crossing re-enable).
	modules: Map<string, boolean>;
	nextId: number;
};

const iso = (unixS: number) => new Date(unixS * 1000).toISOString().replace(/\.\d{3}Z$/, 'Z');
const now = () => Math.floor(Date.now() / 1000);

// Thirteen whole months old on the stub's clock, whatever day the suite runs.
export const IVY = {
	id: 'k-ivy',
	name: 'Ivy',
	birthDate: monthsBefore(todayIn('UTC'), 13),
	timeZone: 'UTC'
} as const;
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
		growthSource: 'who',
		vaccineSchema: null,
		seq: 1
	};
	const leo: KidRow = {
		...LEO,
		sex: 'male',
		gestationWeeks: 35,
		gestationDays: 2,
		growthSource: 'who',
		vaccineSchema: null,
		seq: 2
	};
	const ivy: KidRow = {
		...IVY,
		sex: 'female',
		gestationWeeks: null,
		gestationDays: null,
		growthSource: 'who',
		vaccineSchema: null,
		seq: 3
	};
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
		kids: new Map([mia, leo, ivy].map((k) => [k.id, k])),
		grants: [
			{ kidId: mia.id, parentId: ana.id, role: 'guardian' },
			{ kidId: leo.id, parentId: cleo.id, role: 'guardian' },
			{ kidId: leo.id, parentId: ana.id, role: 'caregiver' },
			{ kidId: ivy.id, parentId: ana.id, role: 'guardian' }
		],
		entries: { bottles: [], nursing: [], pumping: [], sleep: [], diapers: [], meals: [] },
		growth: leoGrowth(),
		vaccinations: [],
		dismissals: new Map(),
		modules: new Map(),
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

const bodies = new WeakMap<IncomingMessage, string>();

async function readBody(req: IncomingMessage): Promise<string> {
	const read = bodies.get(req);
	if (read !== undefined) return read;
	let data = '';
	for await (const chunk of req) data += chunk;
	bodies.set(req, data);
	return data;
}

type FieldKind = 'string' | 'int' | 'units' | 'bool' | 'time' | 'strings' | 'volume' | 'measure';
type Body = Record<string, unknown>;

const RFC3339 = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$/;
const isObject = (v: unknown) => typeof v === 'object' && v !== null && !Array.isArray(v);
const nullOr = (type: string) => (u: unknown) => u === null || typeof u === type;

// The JSON type Go would decode each kind from; a mismatch is a 400.
function typedAs(kind: FieldKind, v: unknown): boolean {
	switch (kind) {
		case 'string':
			return typeof v === 'string';
		case 'int':
			return Number.isInteger(v);
		case 'bool':
			return typeof v === 'boolean';
		case 'time':
			return typeof v === 'string' && RFC3339.test(v) && !Number.isNaN(Date.parse(v));
		case 'strings':
			return Array.isArray(v) && v.every((u) => typeof u === 'string');
		case 'volume':
		case 'measure':
			return (
				isObject(v) &&
				nullOr('string')((v as Body).unit) &&
				((v as Body).amount == null || Number.isInteger((v as Body).amount))
			);
		case 'units':
			return isObject(v) && Object.values(v as Body).every(nullOr('string'));
	}
}

// Keys a nested object may carry before decodeJSON refuses it.
const NESTED_KEYS: Partial<Record<FieldKind, readonly string[]>> = {
	units: Object.keys(UNIT_SETS),
	volume: ['unit', 'amount'],
	measure: ['unit', 'amount']
};

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
	for (const [k, v] of Object.entries(value)) {
		const kind = fields[k];
		if (!kind) {
			error(res, 422, 'unknown_field', k);
			return null;
		}
		if (v === null) continue;
		const allowed = NESTED_KEYS[kind];
		if (allowed && isObject(v)) {
			const unknown = Object.keys(v as Body).find((u) => !allowed.includes(u));
			if (unknown) {
				error(res, 422, 'unknown_field', unknown);
				return null;
			}
		}
		if (!typedAs(kind, v)) {
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
		growth_source: k.growthSource,
		vaccine_schema: k.vaccineSchema,
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

// invalidKidField in kids.go, same order: name, birth_date, sex, gestation_weeks, gestation_days, time_zone,
// growth_source, then vaccine_schema (newSchemaOK; every stub schedule is shipped).
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
	if (!GROWTH_SOURCE_IDS.includes(k.growthSource)) return 'growth_source';
	if (k.vaccineSchema !== null && !STUB_SCHEDULES.has(k.vaccineSchema)) return 'vaccine_schema';
	return '';
}

// mergeOnto: absent or null strings keep the stored value; gestation null clears, absent keeps.
function mergeKid(k: KidRow, body: Body): KidRow {
	const out = { ...k };
	for (const [key, prop] of [
		['name', 'name'],
		['birth_date', 'birthDate'],
		['sex', 'sex'],
		['time_zone', 'timeZone'],
		['growth_source', 'growthSource']
	] as const) {
		if (typeof body[key] === 'string') out[prop] = body[key] as string;
	}
	if ('gestation_weeks' in body) out.gestationWeeks = body.gestation_weeks as number | null;
	if ('gestation_days' in body) out.gestationDays = body.gestation_days as number | null;
	if ('vaccine_schema' in body) out.vaccineSchema = body.vaccine_schema as string | null;
	out.name = out.name.trim();
	return out;
}

const KID_FIELDS: Record<string, FieldKind> = {
	name: 'string',
	birth_date: 'string',
	sex: 'string',
	gestation_weeks: 'int',
	gestation_days: 'int',
	time_zone: 'string',
	growth_source: 'string',
	vaccine_schema: 'string'
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
				growthSource: 'who',
				vaccineSchema: null,
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

// Trackers, timeline, summary, modules and home: the Go handlers' validation and 404/409 behaviour, in memory.

const TRACKER_KEYS: readonly TrackerKey[] = [
	'bottles',
	'nursing',
	'pumping',
	'sleep',
	'diapers',
	'meals'
];
// The timeline names an entry by its singular type, REST paths by the route name.
const TIMELINE_TYPES: Record<TrackerKey, string> = {
	bottles: 'bottle',
	nursing: 'nursing',
	pumping: 'pumping',
	sleep: 'sleep',
	diapers: 'diaper',
	meals: 'meal'
};
const MODULES = [
	'bottles',
	'nursing',
	'pumping',
	'diapers',
	'sleep',
	'feeding',
	'milestones',
	'growth',
	'vaccination',
	'sickness',
	'medication'
] as const;
const MODULE_OF: Record<TrackerKey, string> = {
	bottles: 'bottles',
	nursing: 'nursing',
	pumping: 'pumping',
	diapers: 'diapers',
	sleep: 'sleep',
	meals: 'feeding'
};
const OLDER_BAND_MODULES = ['growth', 'vaccination', 'sickness', 'medication'];
const BOTTLE_CONTENTS = ['breast_milk', 'formula', 'cow_milk', 'other'];
const DIAPER_KINDS = ['wet', 'dirty', 'both', 'dry'];
const NURSING_SIDES = ['left', 'right'];
const PUMPING_SIDES = ['left', 'right', 'both'];
// The API's limits (constants.go), mirrored for the stub only.
const NOTE_MAX_LEN = 500;
const MEAL_TEXT_MAX_LEN = 200;
const MAX_FUTURE_SKEW_S = 5 * 60;
const MAX_SESSION_S = 24 * 3600;
const BOTTLE_ML = { min: 1, max: 1000 };
const PUMPING_ML = { min: 1, max: 2000 };
const MAX_VOLUME_AMOUNT = 1 << 24;
const FLOZ_NUMERATOR = 295735;
const FLOZ_DENOMINATOR = 10000 * 100;
const PAGE_DEFAULT = 50;
const PAGE_MAX = 100;
const SUMMARY_MAX_DAYS = 93;
const BAND_SECOND_MONTHS = 12;
const BAND_THIRD_MONTHS = 24;
const OVERRIDE_LOOKBACK_S = 60 * 24 * 3600;
const DAY_S = 24 * 3600;
const ENTRY_ID = /^[A-Za-z0-9-]{1,64}$/;
const CURSOR = /^(\d+):([A-Za-z0-9-]{1,64})$/;

const unix = (v: unknown) => Math.floor(Date.parse(v as string) / 1000);
const runes = (s: string) => [...s].length;

function wallOffsetMs(ms: number, zone: string): number {
	const parts = new Intl.DateTimeFormat('en-CA', {
		timeZone: zone,
		hourCycle: 'h23',
		year: 'numeric',
		month: '2-digit',
		day: '2-digit',
		hour: '2-digit',
		minute: '2-digit',
		second: '2-digit'
	}).formatToParts(new Date(ms));
	const n = (t: string) => Number(parts.find((p) => p.type === t)?.value);
	const wall = Date.UTC(n('year'), n('month') - 1, n('day'), n('hour'), n('minute'), n('second'));
	return wall - Math.floor(ms / 1000) * 1000;
}

// Local midnight of a YYYY-MM-DD date, as unix seconds.
function midnight(date: string, zone: string): number {
	const [y, m, d] = date.split('-').map(Number);
	const wall = Date.UTC(y, m - 1, d);
	const offsets = [wall - DAY_S * 1000, wall + DAY_S * 1000].map((t) => wallOffsetMs(t, zone));
	const hits = offsets.map((o) => wall - o).filter((t) => wallOffsetMs(t, zone) === wall - t);
	return Math.floor((hits.length > 0 ? Math.min(...hits) : wall - offsets[0]) / 1000);
}

const addDays = (date: string, days: number) => {
	const [y, m, d] = date.split('-').map(Number);
	return new Date(Date.UTC(y, m - 1, d + days)).toISOString().slice(0, 10);
};

// An overflowing day rolls into the next month, as Go's time.Date does.
function addMonths(date: string, months: number): string {
	const [y, m, d] = date.split('-').map(Number);
	return new Date(Date.UTC(y, m - 1 + months, d)).toISOString().slice(0, 10);
}

// The same day n months earlier, clamped to that month's last day.
function monthsBefore(date: string, n: number): string {
	const [y, m, d] = date.split('-').map(Number);
	const lastDay = new Date(Date.UTC(y, m - 1 - n + 1, 0)).getUTCDate();
	return new Date(Date.UTC(y, m - 1 - n, Math.min(d, lastDay))).toISOString().slice(0, 10);
}

// AgeMonths in kidtime.go: whole calendar months, never negative.
function ageMonths(birthDate: string, today: string): number {
	const [by, bm, bd] = birthDate.split('-').map(Number);
	const [ty, tm, td] = today.split('-').map(Number);
	return Math.max((ty - by) * 12 + (tm - bm) - (td < bd ? 1 : 0), 0);
}

const bandOf = (months: number) =>
	months >= BAND_THIRD_MONTHS ? '24+' : months >= BAND_SECOND_MONTHS ? '12-23' : '0-11';

type Ctx = { now: number; birth: number };
const ctxFor = (kid: KidRow): Ctx => ({ now: now(), birth: midnight(kid.birthDate, kid.timeZone) });
const instantOK = (t: number, c: Ctx) => t <= c.now + MAX_FUTURE_SKEW_S && t >= c.birth;

// A time in the body replaces the stored one (truncated to whole seconds); false when the result is refused.
function mergeTime(view: Body, body: Body, key: string, c: Ctx): boolean {
	if (typeof body[key] === 'string') view[key] = iso(unix(body[key]));
	return instantOK(unix(view[key]), c);
}

// optionalString: absent keeps, null clears, a string replaces.
function mergeNote(view: Body, body: Body): boolean {
	if ('note' in body) view.note = body.note;
	return view.note === null || runes(view.note as string) <= NOTE_MAX_LEN;
}

type Vol = { ml: number; floz_hundredths: number; typed_unit: string };
const roundHalfUp = (n: number, d: number) => Math.floor((2 * n + d) / (2 * d));

// units.NewVolume: whole ml, or hundredths of a US fl oz.
function newVolume(unit: unknown, amount: unknown): Vol | null {
	if ((unit !== 'ml' && unit !== 'floz') || !Number.isInteger(amount)) return null;
	const a = amount as number;
	if (a < 1 || a > MAX_VOLUME_AMOUNT) return null;
	const ml = unit === 'ml' ? a : roundHalfUp(a * FLOZ_NUMERATOR, FLOZ_DENOMINATOR);
	const floz = unit === 'floz' ? a : roundHalfUp(a * FLOZ_DENOMINATOR, FLOZ_NUMERATOR);
	return ml < 1 || floz < 1 ? null : { ml, floz_hundredths: floz, typed_unit: unit };
}

// mergeVolume: absent or the unchanged typed value keeps the stored columns; the range is the tracker's.
function mergeVolume(view: Body, body: Body, range: { min: number; max: number }): boolean {
	const stored = view.volume as Vol;
	const input = body.volume as { unit?: string | null; amount?: number | null } | null | undefined;
	if (input) {
		const typed = stored.typed_unit === 'floz' ? stored.floz_hundredths : stored.ml;
		if (input.unit !== stored.typed_unit || input.amount !== typed) {
			const v = newVolume(input.unit, input.amount);
			if (!v) return false;
			view.volume = v;
		}
	}
	const ml = (view.volume as Vol).ml;
	return ml >= range.min && ml <= range.max;
}

const BLANK_VOLUME: Vol = { ml: 0, floz_hundredths: 0, typed_unit: '' };

// SleepSession.InvalidField and NursingSession.InvalidField in model/tracker.go.
function invalidSession(key: TrackerKey, v: Body): string {
	const started = unix(v.started_at);
	const ended = v.ended_at === null ? null : unix(v.ended_at);
	const span = (end: number) => (end < started || end - started > MAX_SESSION_S ? 'ended_at' : '');
	if (key === 'sleep') return ended === null ? '' : span(ended);
	const left = v.left_seconds as number;
	const right = v.right_seconds as number;
	const fits = (s: number) => left <= s && right <= s - left;
	if (left < 0) return 'left_seconds';
	if (right < 0) return 'right_seconds';
	if (ended !== null) {
		if (v.side !== null || v.side_since !== null) return 'side';
		return span(ended) || (fits(ended - started) ? '' : 'left_seconds');
	}
	if (v.side_since === null || !NURSING_SIDES.includes(v.side as string)) return 'side';
	const since = unix(v.side_since);
	return since < started || !fits(since - started) ? 'started_at' : '';
}

// NursingSession.credit: the seconds since side_since go to the side in use; false when at precedes it.
function credit(v: Body, at: number): boolean {
	if (v.ended_at !== null || v.side_since === null || at < unix(v.side_since)) return false;
	const key = v.side === 'left' ? 'left_seconds' : 'right_seconds';
	v[key] = (v[key] as number) + (at - unix(v.side_since));
	return true;
}

type Spec = {
	timeKey: 'at' | 'started_at';
	fields: Record<string, FieldKind>;
	timer: boolean;
	// The entry a POST starts from; merge validates the body onto it.
	blank: (c: Ctx) => Body;
	// A PATCH applied to the stored entry: the first invalid field, "" when valid.
	merge: (view: Body, body: Body, c: Ctx) => string;
	// A POST that is not a merge onto blank: the entry, or the first invalid field.
	create?: (body: Body, c: Ctx) => Body | string;
};

const NOTE_FIELD: Record<string, FieldKind> = { note: 'string' };

function createEntry(spec: Spec, body: Body, c: Ctx): Body | string {
	if (spec.create) return spec.create(body, c);
	const view = spec.blank(c);
	return spec.merge(view, body, c) || view;
}

const SPECS: Record<TrackerKey, Spec> = {
	bottles: {
		timeKey: 'at',
		timer: false,
		fields: { at: 'time', contents: 'strings', volume: 'volume', ...NOTE_FIELD },
		merge: (view, body, c) => {
			if (!mergeTime(view, body, 'at', c)) return 'at';
			if (Array.isArray(body.contents)) view.contents = body.contents;
			const given = view.contents as string[];
			const canonical = BOTTLE_CONTENTS.filter((n) => given.includes(n));
			if (canonical.length === 0 || canonical.length !== given.length) return 'contents';
			view.contents = canonical;
			if (!mergeVolume(view, body, BOTTLE_ML)) return 'volume';
			return mergeNote(view, body) ? '' : 'note';
		},
		blank: (c) => ({ at: iso(c.now), contents: [], volume: { ...BLANK_VOLUME }, note: null })
	},
	diapers: {
		timeKey: 'at',
		timer: false,
		fields: { at: 'time', kind: 'string', ...NOTE_FIELD },
		merge: (view, body, c) => {
			if (!mergeTime(view, body, 'at', c)) return 'at';
			if (typeof body.kind === 'string') view.kind = body.kind;
			if (!DIAPER_KINDS.includes(view.kind as string)) return 'kind';
			return mergeNote(view, body) ? '' : 'note';
		},
		blank: (c) => ({ at: iso(c.now), kind: '', note: null })
	},
	meals: {
		timeKey: 'at',
		timer: false,
		fields: { at: 'time', what: 'string', reaction: 'string', ...NOTE_FIELD },
		merge: (view, body, c) => {
			if (!mergeTime(view, body, 'at', c)) return 'at';
			if (typeof body.what === 'string') view.what = body.what.trim();
			const what = view.what as string;
			if (what === '' || runes(what) > MEAL_TEXT_MAX_LEN) return 'what';
			if ('reaction' in body) view.reaction = body.reaction;
			if (view.reaction !== null && runes(view.reaction as string) > MEAL_TEXT_MAX_LEN) {
				return 'reaction';
			}
			return mergeNote(view, body) ? '' : 'note';
		},
		blank: (c) => ({ at: iso(c.now), what: '', reaction: null, note: null })
	},
	pumping: {
		timeKey: 'started_at',
		timer: false,
		fields: {
			started_at: 'time',
			duration_seconds: 'int',
			volume: 'volume',
			side: 'string',
			...NOTE_FIELD
		},
		merge: (view, body, c) => {
			if (!mergeTime(view, body, 'started_at', c)) return 'started_at';
			if (typeof body.duration_seconds === 'number') view.duration_seconds = body.duration_seconds;
			const d = view.duration_seconds as number;
			if (d < 1 || d > MAX_SESSION_S) return 'duration_seconds';
			if (!mergeVolume(view, body, PUMPING_ML)) return 'volume';
			if (typeof body.side === 'string') view.side = body.side;
			if (!PUMPING_SIDES.includes(view.side as string)) return 'side';
			return mergeNote(view, body) ? '' : 'note';
		},
		blank: (c) => ({
			started_at: iso(c.now),
			duration_seconds: 0,
			volume: { ...BLANK_VOLUME },
			side: '',
			note: null
		})
	},
	sleep: {
		timeKey: 'started_at',
		timer: true,
		fields: { started_at: 'time', ended_at: 'time', ...NOTE_FIELD },
		merge: (view, body, c) => {
			if (!mergeTime(view, body, 'started_at', c)) return 'started_at';
			if (body.ended_at === null && view.ended_at !== null) return 'ended_at';
			if (typeof body.ended_at === 'string') {
				view.ended_at = iso(unix(body.ended_at));
				if (!instantOK(unix(view.ended_at), c)) return 'ended_at';
			}
			if (!mergeNote(view, body)) return 'note';
			return invalidSession('sleep', view);
		},
		blank: (c) => ({ started_at: iso(c.now), ended_at: null, note: null })
	},
	nursing: {
		timeKey: 'started_at',
		timer: true,
		fields: {
			started_at: 'time',
			ended_at: 'time',
			left_seconds: 'int',
			right_seconds: 'int',
			side: 'string',
			...NOTE_FIELD
		},
		// The seconds are editable once finished, the side only through the side route.
		merge: (view, body, c) => {
			const wasRunning = view.ended_at === null;
			if (!mergeTime(view, body, 'started_at', c)) return 'started_at';
			if (body.ended_at === null && !wasRunning) return 'ended_at';
			if (typeof body.ended_at === 'string') {
				const end = unix(body.ended_at);
				if (!instantOK(end, c)) return 'ended_at';
				if (wasRunning) {
					if (!credit(view, end)) return 'ended_at';
					Object.assign(view, { ended_at: iso(end), side: null, side_since: null });
				} else view.ended_at = iso(end);
			}
			for (const key of ['left_seconds', 'right_seconds']) {
				if (typeof body[key] !== 'number') continue;
				if (wasRunning) return key;
				view[key] = body[key];
			}
			if (typeof body.side === 'string') return 'side';
			if (!mergeNote(view, body)) return 'note';
			return invalidSession('nursing', view);
		},
		blank: (c) => ({
			started_at: iso(c.now),
			ended_at: null,
			left_seconds: 0,
			right_seconds: 0,
			side: null,
			side_since: null,
			note: null
		}),
		create: (body, c) => {
			const started = typeof body.started_at === 'string' ? unix(body.started_at) : c.now;
			if (!instantOK(started, c)) return 'started_at';
			const ended = typeof body.ended_at === 'string' ? unix(body.ended_at) : null;
			if (ended !== null && !instantOK(ended, c)) return 'ended_at';
			const running = ended === null;
			const has = (k: string) => typeof body[k] === 'number';
			if (has('left_seconds') === running) return 'left_seconds';
			if (has('right_seconds') === running) return 'right_seconds';
			if ((typeof body.side === 'string') !== running) return 'side';
			const view: Body = {
				started_at: iso(started),
				ended_at: ended === null ? null : iso(ended),
				left_seconds: running ? 0 : body.left_seconds,
				right_seconds: running ? 0 : body.right_seconds,
				side: running ? body.side : null,
				side_since: running ? iso(started) : null,
				note: null
			};
			if (!mergeNote(view, body)) return 'note';
			return invalidSession('nursing', view) || view;
		}
	}
};

const rowOf = (state: State, key: TrackerKey, kidId: string, id: string) =>
	ENTRY_ID.test(id)
		? state.entries[key].find((r) => r.kidId === kidId && r.view.id === id)
		: undefined;

type Page = { from: number; to: number; limit: number; after: { at: number; id: string } | null };

// parseListQuery in query.go: the first invalid parameter, in the order from, to, limit, cursor.
function parsePage(url: URL): Page | string {
	const q = url.searchParams;
	const time = (k: string, fallback: number) =>
		q.has(k) ? unixStrict(q.get(k) as string) : fallback;
	const from = time('from', 0);
	if (Number.isNaN(from)) return 'from';
	const to = time('to', now() + MAX_FUTURE_SKEW_S);
	if (Number.isNaN(to)) return 'to';
	if (from > to) return 'from';
	let limit = PAGE_DEFAULT;
	if (q.has('limit')) {
		limit = /^\d+$/.test(q.get('limit') ?? '') ? Number(q.get('limit')) : 0;
		if (limit < 1 || limit > PAGE_MAX) return 'limit';
	}
	let after: Page['after'] = null;
	if (q.has('cursor')) {
		const m = CURSOR.exec(Buffer.from(q.get('cursor') ?? '', 'base64url').toString());
		if (!m || String(Number(m[1])) !== m[1]) return 'cursor';
		after = { at: Number(m[1]), id: m[2] };
	}
	return { from, to, limit, after };
}

const unixStrict = (s: string) => (RFC3339.test(s) ? unix(s) : NaN);

type Item = { at: number; id: string; out: Body };

// The list tail in store/entries.go: [from, to), strictly older than the cursor, newest first, one page.
function respondPage(res: ServerResponse, key: string, items: Item[], q: Page) {
	const rows = items
		.filter((i) => i.at >= q.from && i.at < q.to)
		.filter((i) => !q.after || i.at < q.after.at || (i.at === q.after.at && i.id < q.after.id))
		.sort((a, b) => b.at - a.at || (a.id < b.id ? 1 : -1));
	const page = rows.slice(0, q.limit);
	const last = rows.length > q.limit ? page[page.length - 1] : null;
	send(res, 200, {
		[key]: page.map((i) => i.out),
		next_cursor: last ? Buffer.from(`${last.at}:${last.id}`).toString('base64url') : null
	});
}

const itemsOf = (state: State, key: TrackerKey, kidId: string): Item[] =>
	state.entries[key]
		.filter((r) => r.kidId === kidId)
		.map((r) => ({
			at: unix(r.view[SPECS[key].timeKey]),
			id: r.view.id,
			out: { ...r.view }
		}));

function trackerRoutes(key: TrackerKey): RouteDef[] {
	const spec = SPECS[key];
	const base = `^\\/v1\\/kids\\/([^/]+)\\/${key}`;
	const kidOf = (state: State, id: string) => state.kids.get(id) as KidRow;
	// The 404 an unknown entry, another kid's entry and a malformed id all share.
	const entryOr404 = (state: State, res: ServerResponse, kidId: string, id: string) => {
		const row = rowOf(state, key, kidId, id);
		if (!row) noKid(res);
		return row;
	};
	const routes: RouteDef[] = [
		{
			method: 'GET',
			pattern: new RegExp(`${base}$`),
			access: 'caregiver',
			kidParam: true,
			run: (state, req, res, _p, [kidId]) => {
				const q = parsePage(new URL(req.url ?? '/', STUB_URL));
				if (typeof q === 'string') return error(res, 422, 'invalid', q);
				respondPage(res, key, itemsOf(state, key, kidId), q);
			}
		},
		{
			method: 'POST',
			pattern: new RegExp(`${base}$`),
			access: 'caregiver',
			kidParam: true,
			run: async (state, req, res, _p, [kidId]) => {
				const body = await decodeBody(req, res, spec.fields);
				if (!body) return;
				const created = createEntry(spec, body, ctxFor(kidOf(state, kidId)));
				if (typeof created === 'string') return error(res, 422, 'invalid', created);
				const running = state.entries[key].some(
					(r) => r.kidId === kidId && r.view.ended_at === null
				);
				if (spec.timer && created.ended_at === null && running) {
					return error(res, 409, 'already_running');
				}
				const view = { id: nextId(state, 'e'), ...created };
				state.entries[key].push({ kidId, view });
				send(res, 201, view);
			}
		},
		{
			method: 'GET',
			pattern: new RegExp(`${base}\\/([^/]+)$`),
			access: 'caregiver',
			kidParam: true,
			run: (state, _req, res, _p, [kidId, id]) => {
				const row = entryOr404(state, res, kidId, id);
				if (row) send(res, 200, row.view);
			}
		},
		{
			method: 'PATCH',
			pattern: new RegExp(`${base}\\/([^/]+)$`),
			access: 'caregiver',
			kidParam: true,
			run: async (state, req, res, _p, [kidId, id]) => {
				const body = await decodeBody(req, res, spec.fields);
				if (!body) return;
				const row = entryOr404(state, res, kidId, id);
				if (!row) return;
				const merged = structuredClone(row.view);
				const field = spec.merge(merged, body, ctxFor(kidOf(state, kidId)));
				if (field) return error(res, 422, 'invalid', field);
				row.view = merged;
				send(res, 200, merged);
			}
		},
		{
			method: 'DELETE',
			pattern: new RegExp(`${base}\\/([^/]+)$`),
			access: 'caregiver',
			kidParam: true,
			run: (state, _req, res, _p, [kidId, id]) => {
				const row = entryOr404(state, res, kidId, id);
				if (!row) return;
				state.entries[key] = state.entries[key].filter((r) => r !== row);
				send(res, 204);
			}
		}
	];
	if (!spec.timer) return routes;

	// timerAction in entries.go: 404, then a bad body field, then at (default now), then the transition.
	const action = (
		suffix: string,
		fields: Record<string, FieldKind>,
		act: (view: Body, body: Body, at: number) => string | null
	): RouteDef => ({
		method: 'POST',
		pattern: new RegExp(`${base}\\/([^/]+)\\/${suffix}$`),
		access: 'caregiver',
		kidParam: true,
		run: async (state, req, res, _p, [kidId, id]) => {
			const body = await decodeBody(req, res, { at: 'time', ...fields });
			if (!body) return;
			const row = entryOr404(state, res, kidId, id);
			if (!row) return;
			const c = ctxFor(kidOf(state, kidId));
			const at = typeof body.at === 'string' ? unix(body.at) : c.now;
			const next = structuredClone(row.view);
			const refused = act(next, body, at);
			if (refused === 'side') return error(res, 422, 'invalid', 'side');
			if (!instantOK(at, c)) return error(res, 422, 'invalid', 'at');
			if (refused === 'not_running') return error(res, 409, 'not_running');
			if (refused === 'at' || invalidSession(key, next)) return error(res, 422, 'invalid', 'at');
			row.view = next;
			send(res, 200, next);
		}
	});
	routes.push(
		action('end', {}, (view, _body, at) => {
			if (view.ended_at !== null) return 'not_running';
			if (key === 'nursing' && !credit(view, at)) return 'at';
			Object.assign(view, { ended_at: iso(at) });
			if (key === 'nursing') Object.assign(view, { side: null, side_since: null });
			return null;
		})
	);
	if (key === 'nursing') {
		routes.push(
			action('side', { side: 'string' }, (view, body, at) => {
				if (!NURSING_SIDES.includes(body.side as string)) return 'side';
				if (view.ended_at !== null) return 'not_running';
				if (!credit(view, at)) return 'at';
				Object.assign(view, { side: body.side, side_since: iso(at) });
				return null;
			})
		);
	}
	return routes;
}

function moduleEnabledByDefault(module: string, band: string) {
	return band !== '24+' || OLDER_BAND_MODULES.includes(module);
}

// Re-evaluated on every read, as Modules in store/modules.go: a recent entry or a running timer turns an
// off-by-default module on once, and a stored row is final.
function moduleStates(state: State, kid: KidRow) {
	const months = ageMonths(kid.birthDate, todayIn(kid.timeZone));
	const band = bandOf(months);
	const since =
		midnight(addMonths(kid.birthDate, BAND_THIRD_MONTHS), kid.timeZone) - OVERRIDE_LOOKBACK_S;
	for (const m of MODULES) {
		const stored = `${kid.id}:${m}`;
		const tracker = TRACKER_KEYS.find((k) => MODULE_OF[k] === m);
		if (state.modules.has(stored) || moduleEnabledByDefault(m, band) || !tracker) continue;
		const recent = state.entries[tracker].some((r) => {
			if (r.kidId !== kid.id) return false;
			const t = unix(r.view[SPECS[tracker].timeKey]);
			return (t >= since && t <= now()) || (SPECS[tracker].timer && r.view.ended_at === null);
		});
		if (recent) state.modules.set(stored, true);
	}
	return MODULES.map((m) => {
		const stored = state.modules.get(`${kid.id}:${m}`);
		const fallback = moduleEnabledByDefault(m, band);
		return {
			module: m,
			enabled: stored ?? fallback,
			default: fallback,
			overridden: stored !== undefined
		};
	});
}

// One kid-local day, as Summaries in store/summary.go.
function daySummary(state: State, kid: KidRow, date: string) {
	const start = midnight(date, kid.timeZone);
	const end = midnight(addDays(date, 1), kid.timeZone);
	const views = (key: TrackerKey) =>
		state.entries[key].filter((r) => r.kidId === kid.id).map((r) => r.view);
	const inDay = (v: Body, timeKey: string) => unix(v[timeKey]) >= start && unix(v[timeKey]) < end;
	const sum = (vs: Body[], pick: (v: Body) => number) => vs.reduce((n, v) => n + pick(v), 0);
	const volumes = (key: TrackerKey, timeKey: string) => {
		const vs = views(key).filter((v) => inDay(v, timeKey));
		return {
			count: vs.length,
			ml: sum(vs, (v) => (v.volume as Vol).ml),
			floz_hundredths: sum(vs, (v) => (v.volume as Vol).floz_hundredths)
		};
	};
	const nursing = views('nursing').filter((v) => inDay(v, 'started_at'));
	const diapers = views('diapers').filter((v) => inDay(v, 'at'));
	const kinds = (k: string) => diapers.filter((v) => v.kind === k).length;
	const overlap = (v: Body) => {
		const to = Math.min(v.ended_at === null ? now() : unix(v.ended_at), end);
		return Math.max(0, to - Math.max(unix(v.started_at), start));
	};
	return {
		date,
		bottles: volumes('bottles', 'at'),
		nursing: {
			count: nursing.length,
			seconds: sum(nursing, (v) => (v.left_seconds as number) + (v.right_seconds as number))
		},
		pumping: volumes('pumping', 'started_at'),
		diapers: {
			total: diapers.length,
			wet: kinds('wet'),
			dirty: kinds('dirty'),
			both: kinds('both'),
			dry: kinds('dry')
		},
		sleep: { seconds: sum(views('sleep'), overlap) },
		meals: { count: views('meals').filter((v) => inDay(v, 'at')).length }
	};
}

const runningOf = (state: State, key: TrackerKey, kidId: string) =>
	state.entries[key].find((r) => r.kidId === kidId && r.view.ended_at === null)?.view ?? null;

const kidRoute = (
	method: string,
	suffix: string,
	access: 'caregiver' | 'guardian',
	run: (state: State, req: IncomingMessage, res: ServerResponse, kid: KidRow) => unknown
): RouteDef => ({
	method,
	pattern: new RegExp(`^\\/v1\\/kids\\/([^/]+)\\/${suffix}$`),
	access,
	kidParam: true,
	run: (state, req, res, _p, [kidId]) => run(state, req, res, state.kids.get(kidId) as KidRow)
});

const overviewRoutes: RouteDef[] = [
	kidRoute('GET', 'timeline', 'caregiver', (state, req, res, kid) => {
		const url = new URL(req.url ?? '/', STUB_URL);
		const q = parsePage(url);
		if (typeof q === 'string') return error(res, 422, 'invalid', q);
		const given = url.searchParams.getAll('types');
		let keys: TrackerKey[];
		if (given.length === 0) {
			const on = moduleStates(state, kid)
				.filter((m) => m.enabled)
				.map((m) => m.module as string);
			keys = TRACKER_KEYS.filter((k) => on.includes(MODULE_OF[k]));
		} else {
			const names = given.length === 1 ? given[0].split(',') : [''];
			const wanted = names.map((n) => TRACKER_KEYS.find((k) => TIMELINE_TYPES[k] === n));
			if (wanted.some((k, i) => !k || wanted.indexOf(k) !== i)) {
				return error(res, 422, 'invalid', 'types');
			}
			keys = wanted as TrackerKey[];
		}
		const items = keys.flatMap((key) =>
			itemsOf(state, key, kid.id).map((i) => ({
				...i,
				out: { type: TIMELINE_TYPES[key], at: iso(i.at), ...i.out }
			}))
		);
		respondPage(res, 'items', items, q);
	}),
	kidRoute('GET', 'summary', 'caregiver', (state, req, res, kid) => {
		const q = new URL(req.url ?? '/', STUB_URL).searchParams;
		const from = q.get('from') ?? '';
		const to = q.get('to') ?? '';
		if (!validDate(from)) return error(res, 422, 'invalid', 'from');
		if (!validDate(to)) return error(res, 422, 'invalid', 'to');
		const days = (Date.parse(to) - Date.parse(from)) / (DAY_S * 1000) + 1;
		if (days < 1) return error(res, 422, 'invalid', 'from');
		if (days > SUMMARY_MAX_DAYS) return error(res, 422, 'invalid', 'to');
		send(res, 200, {
			days: Array.from({ length: days }, (_, i) => daySummary(state, kid, addDays(from, i)))
		});
	}),
	kidRoute('GET', 'modules', 'caregiver', (state, _req, res, kid) => {
		send(res, 200, { modules: moduleStates(state, kid) });
	}),
	{
		method: 'PUT',
		pattern: /^\/v1\/kids\/([^/]+)\/modules\/([^/]+)$/,
		access: 'guardian',
		kidParam: true,
		run: async (state, req, res, _p, [kidId, module]) => {
			const body = await decodeBody(req, res, { enabled: 'bool' });
			if (!body) return;
			if (!(MODULES as readonly string[]).includes(module)) return noKid(res);
			if (typeof body.enabled !== 'boolean') return error(res, 422, 'invalid', 'enabled');
			state.modules.set(`${kidId}:${module}`, body.enabled);
			send(res, 204);
		}
	},
	kidRoute('GET', 'home', 'caregiver', (state, _req, res, kid) => {
		const today = todayIn(kid.timeZone);
		const months = ageMonths(kid.birthDate, today);
		const correction = correctionOn(kid, today);
		send(res, 200, {
			age_months: months,
			corrected_birth_date: correction > 0 ? addDays(kid.birthDate, correction) : null,
			band: bandOf(months),
			modules: moduleStates(state, kid),
			running: {
				sleep: runningOf(state, 'sleep', kid.id),
				nursing: runningOf(state, 'nursing', kid.id)
			},
			today: daySummary(state, kid, todayIn(kid.timeZone)),
			due: vaccineDue(state, kid)
		});
	})
];

// Growth: the Go handlers' validation, conversions and 404s in memory; scores and curves are fake.

const GROWTH_SOURCE_IDS = ['who', 'cdc'];
const GROWTH_MEASURES = ['weight', 'length', 'head', 'bmi'] as const;
type GrowthMeasure = (typeof GROWTH_MEASURES)[number];
// The API's limits (constants.go), mirrored for the stub only.
const GROWTH_RANGE = {
	weight: { min: 200, max: 250000 },
	length: { min: 150, max: 2500 },
	head: { min: 150, max: 800 }
};
const MAX_GROWTH_PER_KID = 3000;
const MAX_MEASURE_AMOUNT = 1 << 24;
const GRAMS_PER_OZ = { num: 28349523125n, den: 1000000000n * 100n };
const MM_PER_INCH = { num: 254n, den: 10n * 100n };
const TERM_GESTATION_DAYS = 280;
const PRETERM_BELOW_DAYS = 259;
const CORRECTION_ENDS_MONTHS = 24;
const DAYS_PER_MONTH = 30.4375;
const CURVE_MAX_MONTHS = 240;
const CURVE_DEFAULT_MIN_TO = 24;
const CURVE_DEFAULT_AHEAD = 6;
const CURVE_FINE_STEP = 0.5;
const CURVE_FINE_UNTIL = 24;
const POUNDS_PER_KG = 2.2046226218;
const CM_PER_INCH = 2.54;

type Weight = { grams: number; oz_hundredths: number; typed_unit: string };
type Length = { mm: number; in_hundredths: number; typed_unit: string };
type GrowthStub = {
	kidId: string;
	id: string;
	measuredOn: string;
	weight: Weight | null;
	length: Length | null;
	head: Length | null;
	note: string | null;
};

const halfUp = (n: bigint, d: bigint) => Number((2n * n + d) / (2n * d));

// units.NewWeight and units.NewLength: the typed amount is exact, the other column derived half-up.
function newMeasure(
	metric: string,
	imperial: string,
	ratio: { num: bigint; den: bigint },
	unit: unknown,
	amount: unknown
): { typed_unit: string; metric: number; imperial: number } | null {
	if ((unit !== metric && unit !== imperial) || !Number.isInteger(amount)) return null;
	const a = amount as number;
	if (a < 1 || a > MAX_MEASURE_AMOUNT) return null;
	const big = BigInt(a);
	const m = unit === metric ? a : halfUp(big * ratio.num, ratio.den);
	const i = unit === imperial ? a : halfUp(big * ratio.den, ratio.num);
	return m < 1 || i < 1 ? null : { typed_unit: unit, metric: m, imperial: i };
}

function newWeight(unit: unknown, amount: unknown): Weight | null {
	const m = newMeasure('kg', 'lb', GRAMS_PER_OZ, unit, amount);
	return m && { grams: m.metric, oz_hundredths: m.imperial, typed_unit: m.typed_unit };
}

function newLength(unit: unknown, amount: unknown): Length | null {
	const m = newMeasure('cm', 'in', MM_PER_INCH, unit, amount);
	return m && { mm: m.metric, in_hundredths: m.imperial, typed_unit: m.typed_unit };
}

// Leo's rows: one before his expected term, one at corrected age, one after his second birthday.
function leoGrowth(): GrowthStub[] {
	const row = (
		id: string,
		measuredOn: string,
		grams: number,
		mm: number | null,
		head: number | null
	) => ({
		kidId: LEO.id,
		id,
		measuredOn,
		weight: newWeight('kg', grams),
		length: mm === null ? null : newLength('cm', mm),
		head: head === null ? null : newLength('cm', head),
		note: null
	});
	return [
		row('g-leo-term', '2024-09-01', 2500, null, null),
		row('g-leo-baby', '2025-02-20', 7000, 650, 430),
		row('g-leo-two', '2026-09-01', 12500, 880, null)
	];
}

const civilDay = (date: string) => Date.parse(`${date}T00:00:00Z`) / (DAY_S * 1000);

// correctionOn in kidtime.go: the days born early, until the second birthday.
function correctionOn(kid: KidRow, on: string): number {
	const w = kid.gestationWeeks;
	const d = kid.gestationDays;
	if (w === null || d === null || on >= addMonths(kid.birthDate, CORRECTION_ENDS_MONTHS)) return 0;
	const gestation = w * 7 + d;
	return gestation >= PRETERM_BELOW_DAYS ? 0 : TERM_GESTATION_DAYS - gestation;
}

function ageBasis(kid: KidRow, on: string) {
	const days = civilDay(on) - civilDay(kid.birthDate);
	const correction = correctionOn(kid, on);
	return { days, scoring: days - correction, corrected: correction > 0 };
}

// A STUB, not reference data: invented medians and spreads, so scores and curves are deterministic.
const FAKE_REFERENCE: Record<GrowthMeasure, { at0: number; perMonth: number; sd: number }> = {
	weight: { at0: 3.3, perMonth: 0.4, sd: 1.2 },
	length: { at0: 50, perMonth: 1.8, sd: 3 },
	head: { at0: 35, perMonth: 0.6, sd: 1.5 },
	bmi: { at0: 13.5, perMonth: 0.05, sd: 1.5 }
};
// The months each source covers per measure (both sexes), close to the real tables' coverage.
const COVERAGE: Record<string, Record<GrowthMeasure, [number, number]>> = {
	who: { weight: [0, 120], length: [0, 228], head: [0, 60.98], bmi: [0, 228] },
	cdc: { weight: [0, 240], length: [0, 240], head: [0, 36], bmi: [24, 240] }
};
const LINE_Z: Record<number, number> = {
	3: -1.881,
	5: -1.645,
	10: -1.282,
	15: -1.036,
	25: -0.674,
	50: 0,
	75: 0.674,
	85: 1.036,
	90: 1.282,
	95: 1.645,
	97: 1.881
};
const CDC_BMI_EXTRA_LINE = 85;
const LINES: Record<string, number[]> = {
	who: [3, 15, 50, 85, 97],
	cdc: [3, 5, 10, 25, 50, 75, 90, 95, 97]
};

const round = (x: number, places: number) => {
	const r = Math.round(x * 10 ** places) / 10 ** places;
	return r === 0 ? 0 : r;
};
const median = (m: GrowthMeasure, months: number) =>
	FAKE_REFERENCE[m].at0 + FAKE_REFERENCE[m].perMonth * months;
const covers = (source: string, m: GrowthMeasure, months: number) =>
	months >= COVERAGE[source][m][0] && months <= COVERAGE[source][m][1];

function fakeScore(kid: KidRow, m: GrowthMeasure, scoringDays: number, value: number) {
	if (scoringDays < 0) return { scored: false, reason: 'before_term' };
	const months = scoringDays / DAYS_PER_MONTH;
	if (!covers(kid.growthSource, m, months)) return { scored: false, reason: 'not_covered' };
	const z = (value - median(m, months)) / FAKE_REFERENCE[m].sd;
	return { scored: true, z: round(z, 2), percentile: round(100 / (1 + Math.exp(-1.702 * z)), 1) };
}

function growthView(g: GrowthStub, kid: KidRow) {
	const basis = ageBasis(kid, g.measuredOn);
	const score = (m: GrowthMeasure, value: number) => fakeScore(kid, m, basis.scoring, value);
	const bmiValue =
		g.weight && g.length ? (g.weight.grams * 1000) / (g.length.mm * g.length.mm) : null;
	return {
		id: g.id,
		measured_on: g.measuredOn,
		weight: g.weight,
		length: g.length,
		head: g.head,
		note: g.note,
		age: { days: basis.days, corrected_days: basis.corrected ? basis.scoring : null },
		scores: {
			weight: g.weight && score('weight', g.weight.grams / 1000),
			length: g.length && score('length', g.length.mm / 10),
			head: g.head && score('head', g.head.mm / 10),
			bmi: bmiValue === null ? null : { ...score('bmi', bmiValue), value: round(bmiValue, 2) }
		}
	};
}

type MeasureBody = { unit?: unknown; amount?: unknown } | null | undefined;

// mergeWeight and mergeLength: absent keeps, null clears, the stored typed value keeps the columns.
function mergeMeasure<T extends { typed_unit: string }>(
	stored: T | null,
	body: Body,
	key: string,
	typed: (v: T) => number,
	make: (unit: unknown, amount: unknown) => T | null
): { ok: boolean; v: T | null } {
	if (!(key in body)) return { ok: true, v: stored };
	const input = body[key] as MeasureBody;
	if (!input) return { ok: true, v: null };
	if (stored && input.unit === stored.typed_unit && input.amount === typed(stored)) {
		return { ok: true, v: stored };
	}
	const v = make(input.unit ?? '', input.amount ?? 0);
	return { ok: v !== null, v };
}

const inGrowthRange = (n: number | undefined, r: { min: number; max: number }) =>
	n === undefined || (n >= r.min && n <= r.max);

// growthInput.mergeOnto in growth.go: the first invalid field, "" when valid.
function mergeGrowth(g: GrowthStub, body: Body, kid: KidRow): string {
	const stored = g.measuredOn;
	if (typeof body.measured_on === 'string') g.measuredOn = body.measured_on;
	if (g.measuredOn === '' || g.measuredOn !== stored) {
		if (
			!validDate(g.measuredOn) ||
			g.measuredOn > todayIn(kid.timeZone) ||
			g.measuredOn < kid.birthDate
		)
			return 'measured_on';
	}
	const w = mergeMeasure(
		g.weight,
		body,
		'weight',
		(v) => (v.typed_unit === 'lb' ? v.oz_hundredths : v.grams),
		newWeight
	);
	if (!w.ok || !inGrowthRange(w.v?.grams, GROWTH_RANGE.weight)) return 'weight';
	g.weight = w.v;
	const lengthTyped = (v: Length) => (v.typed_unit === 'in' ? v.in_hundredths : v.mm);
	const l = mergeMeasure(g.length, body, 'length', lengthTyped, newLength);
	if (!l.ok || !inGrowthRange(l.v?.mm, GROWTH_RANGE.length)) return 'length';
	g.length = l.v;
	const h = mergeMeasure(g.head, body, 'head', lengthTyped, newLength);
	if (!h.ok || !inGrowthRange(h.v?.mm, GROWTH_RANGE.head)) return 'head';
	g.head = h.v;
	if ('note' in body) g.note = body.note as string | null;
	if (g.note !== null && runes(g.note) > NOTE_MAX_LEN) return 'note';
	if (!g.weight && !g.length && !g.head) return 'measurements';
	return '';
}

const GROWTH_FIELDS: Record<string, FieldKind> = {
	measured_on: 'string',
	weight: 'measure',
	length: 'measure',
	head: 'measure',
	note: 'string'
};

// monthsParam in growth_curves.go: a finite month count in [0, CURVE_MAX_MONTHS]; def when absent.
function monthsParam(q: URLSearchParams, key: string, def: number): number | null {
	if (!q.has(key)) return def;
	const s = q.get(key) ?? '';
	if (!/^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$/.test(s)) return null;
	const v = Number(s);
	return v < 0 || v > CURVE_MAX_MONTHS ? null : v;
}

function curves(kid: KidRow, measure: GrowthMeasure, from: number, to: number) {
	const [lo, hi] = COVERAGE[kid.growthSource][measure];
	const ages: number[] = [];
	for (let a = 0; a <= hi; a += a < CURVE_FINE_UNTIL ? CURVE_FINE_STEP : 1) {
		if (a >= Math.max(from, lo) && a <= Math.min(to, hi)) ages.push(a);
	}
	// Like the API, a line ends on the table's last row (WHO head: 60.98 months).
	if (hi <= to && hi >= Math.max(from, lo) && ages.at(-1) !== hi) ages.push(hi);
	if (ages.length === 0) return null;
	const percentiles = [...LINES[kid.growthSource]];
	if (kid.growthSource === 'cdc' && measure === 'bmi') percentiles.push(CDC_BMI_EXTRA_LINE);
	percentiles.sort((a, b) => a - b);
	const imperial = (v: number) =>
		measure === 'weight' ? v * POUNDS_PER_KG : measure === 'bmi' ? null : v / CM_PER_INCH;
	return {
		from: ages[0],
		to: ages[ages.length - 1],
		lines: percentiles.map((p) => ({
			percentile: p,
			points: ages.map((a) => {
				const metric = median(measure, a) + LINE_Z[p] * FAKE_REFERENCE[measure].sd;
				return { age_months: a, metric, imperial: imperial(metric) };
			})
		}))
	};
}

const SOURCE_INFO: Record<string, Record<string, string>> = {
	who: {
		name: 'WHO Child Growth Standards (2006) and Growth Reference (2007)',
		attribution: 'Stub attribution: World Health Organization',
		licence: 'CC BY-NC-SA 3.0 IGO',
		version: '2006 and 2007',
		origin_url: 'https://www.who.int/tools/child-growth-standards',
		retrieved: '2026-11-22'
	},
	cdc: {
		name: 'CDC 2000 Growth Charts',
		attribution: 'Stub attribution: Source: CDC',
		licence: 'Public domain',
		version: '2000',
		origin_url: 'https://www.cdc.gov/growthcharts/data/zscore/',
		retrieved: '2026-11-22'
	}
};

const growthRowOf = (state: State, kidId: string, id: string) =>
	ENTRY_ID.test(id) ? state.growth.find((g) => g.kidId === kidId && g.id === id) : undefined;

// The 404 an unknown row, another kid's row and a malformed id all share.
const growthEntryRoute = (
	method: string,
	run: (state: State, res: ServerResponse, kid: KidRow, row: GrowthStub, body: Body) => unknown
): RouteDef => ({
	method,
	pattern: /^\/v1\/kids\/([^/]+)\/growth\/([^/]+)$/,
	access: 'caregiver',
	kidParam: true,
	run: async (state, req, res, _p, [kidId, id]) => {
		const body = method === 'PATCH' ? await decodeBody(req, res, GROWTH_FIELDS) : {};
		if (!body) return;
		const row = growthRowOf(state, kidId, id);
		if (!row) return noKid(res);
		return run(state, res, state.kids.get(kidId) as KidRow, row, body);
	}
});

const growthRoutes: RouteDef[] = [
	kidRoute('GET', 'growth', 'caregiver', (state, _req, res, kid) => {
		const rows = state.growth
			.filter((g) => g.kidId === kid.id)
			.sort((a, b) =>
				a.measuredOn === b.measuredOn
					? a.id < b.id
						? 1
						: -1
					: a.measuredOn < b.measuredOn
						? 1
						: -1
			);
		send(res, 200, { growth: rows.map((g) => growthView(g, kid)) });
	}),
	kidRoute('POST', 'growth', 'caregiver', async (state, req, res, kid) => {
		const body = await decodeBody(req, res, GROWTH_FIELDS);
		if (!body) return;
		const g: GrowthStub = {
			kidId: kid.id,
			id: '',
			measuredOn: '',
			weight: null,
			length: null,
			head: null,
			note: null
		};
		const field = mergeGrowth(g, body, kid);
		if (field) return error(res, 422, 'invalid', field);
		if (state.growth.filter((r) => r.kidId === kid.id).length >= MAX_GROWTH_PER_KID) {
			return error(res, 409, 'limit');
		}
		g.id = nextId(state, 'g');
		state.growth.push(g);
		send(res, 201, growthView(g, kid));
	}),
	kidRoute('GET', 'growth\\/curves', 'caregiver', (_state, req, res, kid) => {
		const q = new URL(req.url ?? '/', STUB_URL).searchParams;
		const measure = q.get('measure') ?? '';
		if (!(GROWTH_MEASURES as readonly string[]).includes(measure)) {
			return error(res, 422, 'invalid', 'measure');
		}
		const basis = ageBasis(kid, todayIn(kid.timeZone));
		const defTo = Math.min(
			Math.max(CURVE_DEFAULT_MIN_TO, basis.scoring / DAYS_PER_MONTH + CURVE_DEFAULT_AHEAD),
			CURVE_MAX_MONTHS
		);
		const from = monthsParam(q, 'from_months', 0);
		if (from === null) return error(res, 422, 'invalid', 'from_months');
		const to = monthsParam(q, 'to_months', defTo);
		if (to === null) return error(res, 422, 'invalid', 'to_months');
		if (from > to) return error(res, 422, 'invalid', 'from_months');
		const c = curves(kid, measure as GrowthMeasure, from, to);
		send(res, 200, {
			source: kid.growthSource,
			measure,
			sex: kid.sex,
			covered: c !== null,
			reason: c === null ? 'not_covered' : null,
			from_months: c?.from ?? from,
			to_months: c?.to ?? to,
			lines: c?.lines ?? [],
			corrected_age_axis: basis.corrected
		});
	}),
	growthEntryRoute('GET', (_state, res, kid, row) => send(res, 200, growthView(row, kid))),
	growthEntryRoute('PATCH', (_state, res, kid, row, body) => {
		const merged = structuredClone(row);
		const field = mergeGrowth(merged, body, kid);
		if (field) return error(res, 422, 'invalid', field);
		Object.assign(row, merged);
		send(res, 200, growthView(row, kid));
	}),
	growthEntryRoute('DELETE', (state, res, _kid, row) => {
		state.growth = state.growth.filter((g) => g !== row);
		send(res, 204);
	}),
	{
		method: 'GET',
		pattern: /^\/v1\/growth-sources$/,
		access: 'parent',
		run: (_state, _req, res) => {
			send(res, 200, {
				sources: GROWTH_SOURCE_IDS.map((id) => ({
					id,
					...SOURCE_INFO[id],
					coverage: GROWTH_MEASURES.map((m) => ({
						measure: m,
						from_months: COVERAGE[id][m][0],
						to_months: COVERAGE[id][m][1]
					}))
				}))
			});
		}
	}
];
// Vaccination: the Go handlers' validation, 404s and evaluation rules in memory, on a tiny made-up schedule.

const MAX_VACCINE_ADMINISTRATIONS_PER_KID = 500;
const VACCINE_LOT_MAX_LEN = 40;
const VACCINE_PRODUCT_MAX_LEN = 80;
const VACCINE_DEFAULT_WINDOW_MONTHS = 1;
const DISMISS_REASONS = ['not_needed', 'given_unknown'];
const ANTIGEN_OTHER = 'other';

type VaccineKind = 'routine' | 'may_not_be_needed' | 'supplementary' | 'recurring';
type Age = { n: number; unit: 'd' | 'w' | 'm' | 'y' };
type StubLine = {
	id: string;
	group: string;
	antigen: string;
	series: number;
	from: Age;
	until?: Age;
	kind: VaccineKind;
};
type AdministrationStub = {
	kidId: string;
	id: string;
	givenOn: string;
	antigens: string[];
	lot: string | null;
	product: string | null;
	note: string | null;
};

const age = (text: string): Age => ({
	n: Number(text.slice(0, -1)),
	unit: text.slice(-1) as Age['unit']
});
const stubLine = (
	id: string,
	group: string,
	from: string,
	kind: VaccineKind = 'routine',
	until?: string
) => {
	const [antigen, series] = id.split('-');
	return {
		id,
		group,
		antigen,
		series: Number(series),
		from: age(from),
		until: until ? age(until) : undefined,
		kind
	};
};

// A STUB, not reference data: two made-up schedules for UI tests only; birth-date cohorts are not
// modelled.
type StubSchedule = { info: Record<string, string>; omitted: string[]; lines: readonly StubLine[] };
const STUB_SCHEDULES = new Map<string, StubSchedule>(
	(
		[
			[
				'us',
				'United States (stub)',
				[
					stubLine('diphtheria-1', '6-in-1', '8w'),
					stubLine('hib-1', '6-in-1', '8w'),
					stubLine('rotavirus-1', 'Rotavirus', '8w', 'may_not_be_needed', '15w'),
					stubLine('measles-1', 'MMR', '12m', 'routine', '15m'),
					stubLine('hib-2', 'Hib', '12m', 'routine', '15m'),
					stubLine('diphtheria-2', 'DTaP', '18m'),
					stubLine('influenza-1', 'Flu', '6m', 'recurring'),
					stubLine('rsv-1', 'RSV', '8m', 'supplementary')
				]
			],
			[
				'uk',
				'United Kingdom (stub)',
				[
					stubLine('diphtheria-1', '6-in-1', '8w'),
					stubLine('hib-1', '6-in-1', '8w'),
					stubLine('measles-1', 'MMR', '1y'),
					stubLine('measles-2', 'MMR', '3y'),
					stubLine('influenza-1', 'Flu', '2y', 'recurring')
				]
			]
		] as const
	).map(([id, name, lines]) => [
		id,
		{
			info: {
				id,
				name,
				authority: 'Stub authority',
				edition: 'Stub edition',
				retrieved: '2026-11-22',
				origin_url: 'https://example.test/schedule',
				licence: 'Stub licence',
				attribution: 'Stub attribution'
			},
			omitted: ['Stub omission'],
			lines: lines as readonly StubLine[]
		}
	])
);

const VACCINE_ANTIGENS = [
	['diphtheria', 'Diphtheria'],
	['hib', 'Haemophilus influenzae type b (Hib)'],
	['rotavirus', 'Rotavirus'],
	['measles', 'Measles'],
	['influenza', 'Influenza'],
	['rsv', 'Respiratory syncytial virus (RSV)'],
	['typhoid', 'Typhoid'],
	[ANTIGEN_OTHER, 'Other (product name required)']
].map(([id, name]) => ({ id, name }));
const VACCINE_PRODUCTS = [
	{ id: '6in1', name: '6-in-1 (stub)', antigens: ['diphtheria', 'hib'] },
	{ id: 'mmr', name: 'MMR (stub)', antigens: ['measles'] }
];

function addAge(date: string, a: Age): string {
	if (a.unit === 'd') return addDays(date, a.n);
	if (a.unit === 'w') return addDays(date, 7 * a.n);
	return addMonths(date, a.unit === 'm' ? a.n : 12 * a.n);
}

type LineState = {
	line: StubLine;
	status: string;
	fromDate: string;
	untilDate: string;
	givenOn: string | null;
	administration: string | null;
	reason: string | null;
};
type StubDose = { id: string; administration: string; antigen: string; givenOn: string };
const STATUS_RANK = ['overdue', 'due', 'upcoming', 'available', 'dismissed', 'given'];

const dosesOf = (state: State, kidId: string): StubDose[] =>
	state.vaccinations
		.filter((a) => a.kidId === kidId)
		.flatMap((a) =>
			a.antigens.map((antigen) => ({
				id: `${a.id}:${antigen}`,
				administration: a.id,
				antigen,
				givenOn: a.givenOn
			}))
		)
		.sort((x, y) =>
			x.givenOn === y.givenOn ? (x.id < y.id ? -1 : 1) : x.givenOn < y.givenOn ? -1 : 1
		);

// Evaluate in vaccines/evaluate.go: the Nth dose of an antigen fills its Nth line; given beats dismissed.
function evaluate(state: State, kid: KidRow, schemaId: string | null) {
	const today = todayIn(kid.timeZone);
	const lines = schemaId ? (STUB_SCHEDULES.get(schemaId)?.lines ?? []) : [];
	const doses = dosesOf(state, kid.id);
	const e = {
		lines: [] as LineState[],
		also: [] as LineState[],
		recurring: [] as Body[],
		other: [] as StubDose[]
	};
	const antigens = [...new Set(lines.map((l) => l.antigen))];
	e.other = doses.filter((d) => !antigens.includes(d.antigen));
	for (const antigen of antigens) {
		const series = lines.filter((l) => l.antigen === antigen).sort((a, b) => a.series - b.series);
		const given = doses.filter((d) => d.antigen === antigen);
		if (series[0].kind === 'recurring') {
			e.recurring.push({
				antigen,
				line_id: series[0].id,
				doses: given.map((d) => ({ administration: d.administration, given_on: d.givenOn }))
			});
			continue;
		}
		series.forEach((line, i) => {
			const fromDate = addAge(kid.birthDate, line.from);
			const untilDate = line.until
				? addAge(kid.birthDate, line.until)
				: addMonths(fromDate, VACCINE_DEFAULT_WINDOW_MONTHS);
			let status =
				line.kind === 'supplementary'
					? 'available'
					: today < fromDate
						? 'upcoming'
						: today >= untilDate
							? line.kind === 'routine'
								? 'overdue'
								: 'available'
							: 'due';
			const ls: LineState = {
				line,
				status,
				fromDate,
				untilDate,
				givenOn: null,
				administration: null,
				reason: null
			};
			const reason = state.dismissals.get(`${kid.id}:${schemaId}:${line.id}`);
			if (i < given.length) {
				status = 'given';
				Object.assign(ls, { givenOn: given[i].givenOn, administration: given[i].administration });
			} else if (reason !== undefined) {
				status = 'dismissed';
				ls.reason = reason;
			}
			ls.status = status;
			(line.kind === 'supplementary' ? e.also : e.lines).push(ls);
		});
		e.other.push(...given.slice(series.length));
	}
	e.other.sort((x, y) =>
		x.givenOn === y.givenOn ? (x.id < y.id ? -1 : 1) : x.givenOn < y.givenOn ? -1 : 1
	);
	const cmp = <T extends string | number>(x: T, y: T) => (x < y ? -1 : x > y ? 1 : 0);
	// cmp.Or in Evaluate: from date, then group, antigen and series.
	const order = (a: LineState, b: LineState) =>
		cmp(a.fromDate, b.fromDate) ||
		cmp(a.line.group, b.line.group) ||
		cmp(a.line.antigen, b.line.antigen) ||
		cmp(a.line.series, b.line.series);
	e.lines.sort(order);
	e.also.sort(order);
	return { ...e, today };
}

const kidSchedule = (kid: KidRow) =>
	kid.vaccineSchema !== null && STUB_SCHEDULES.has(kid.vaccineSchema) ? kid.vaccineSchema : null;

function lineView(ls: LineState) {
	return {
		id: ls.line.id,
		group: ls.line.group,
		antigen: ls.line.antigen,
		series: ls.line.series,
		kind: ls.line.kind,
		status: ls.status,
		from_date: ls.fromDate,
		until_date: ls.untilDate,
		given_on: ls.givenOn,
		administration: ls.administration,
		dismissal: ls.reason === null ? null : { reason: ls.reason },
		note: 'Stub note'
	};
}

function vaccinesView(state: State, kid: KidRow) {
	const schemaId = kidSchedule(kid);
	const e = evaluate(state, kid, schemaId);
	const groups: { from: Age; from_date: string; status: string; lines: Body[] }[] = [];
	for (const ls of e.lines) {
		let g = groups.at(-1);
		if (!g || g.from_date !== ls.fromDate) {
			g = { from: ls.line.from, from_date: ls.fromDate, status: ls.status, lines: [] };
			groups.push(g);
		}
		if (STATUS_RANK.indexOf(ls.status) < STATUS_RANK.indexOf(g.status)) g.status = ls.status;
		g.lines.push(lineView(ls));
	}
	const product = (d: StubDose) =>
		state.vaccinations.find((a) => a.id === d.administration)?.product ?? null;
	const schedule = schemaId ? STUB_SCHEDULES.get(schemaId) : undefined;
	return {
		schema: schedule ? { ...schedule.info, omitted: schedule.omitted } : null,
		today: e.today,
		groups,
		also: e.also.map(lineView),
		recurring: e.recurring,
		other: e.other.map((d) => ({
			administration: d.administration,
			antigen: d.antigen,
			given_on: d.givenOn,
			product: product(d)
		}))
	};
}

// vaccineDue in home.go: one entry per visit with a due or overdue line; empty when the module is off.
function vaccineDue(state: State, kid: KidRow) {
	const on = moduleStates(state, kid).some((m) => m.module === 'vaccination' && m.enabled);
	const schemaId = kidSchedule(kid);
	if (!on || schemaId === null) return [];
	const out: { kind: string; from: Age; status: string; count: number }[] = [];
	let fromDate = '';
	for (const ls of evaluate(state, kid, schemaId).lines) {
		if (ls.status !== 'due' && ls.status !== 'overdue') continue;
		if (out.length === 0 || fromDate !== ls.fromDate) {
			out.push({ kind: 'vaccine', from: ls.line.from, status: 'due', count: 0 });
			fromDate = ls.fromDate;
		}
		const g = out[out.length - 1];
		g.count++;
		if (ls.status === 'overdue') g.status = 'overdue';
	}
	return out;
}

const administrationView = (a: AdministrationStub) => ({
	id: a.id,
	given_on: a.givenOn,
	antigens: a.antigens,
	lot: a.lot,
	product: a.product,
	note: a.note
});

const ADMINISTRATION_FIELDS: Record<string, FieldKind> = {
	given_on: 'string',
	antigens: 'strings',
	lot: 'string',
	product: 'string',
	note: 'string'
};

// mergeOnto in vaccines.go: absent keeps, null clears lot, product or note; an unchanged stored date stays valid.
function mergeAdministration(a: AdministrationStub, body: Body, kid: KidRow): string {
	const stored = a.givenOn;
	if ('given_on' in body) {
		if (body.given_on === null) return 'given_on';
		a.givenOn = body.given_on as string;
	}
	if (a.givenOn === '' || a.givenOn !== stored) {
		if (!validDate(a.givenOn) || a.givenOn > todayIn(kid.timeZone) || a.givenOn < kid.birthDate) {
			return 'given_on';
		}
	}
	if ('antigens' in body) {
		const ids = body.antigens as string[] | null;
		const known = VACCINE_ANTIGENS.map((x) => x.id);
		if (
			ids === null ||
			ids.length === 0 ||
			ids.some((id, i) => !known.includes(id) || ids.indexOf(id) !== i)
		) {
			return 'antigens';
		}
		a.antigens = [...ids].sort();
	}
	if (a.antigens.length === 0) return 'antigens';
	const tooLong = (s: string | null, max: number) => s !== null && runes(s) > max;
	if ('lot' in body) a.lot = body.lot as string | null;
	if (tooLong(a.lot, VACCINE_LOT_MAX_LEN)) return 'lot';
	if ('product' in body) a.product = body.product as string | null;
	if (
		tooLong(a.product, VACCINE_PRODUCT_MAX_LEN) ||
		(a.antigens.includes(ANTIGEN_OTHER) && (a.product ?? '').trim() === '')
	) {
		return 'product';
	}
	if ('note' in body) a.note = body.note as string | null;
	if (tooLong(a.note, NOTE_MAX_LEN)) return 'note';
	return '';
}

const administrationOf = (state: State, kidId: string, id: string) =>
	ENTRY_ID.test(id) ? state.vaccinations.find((a) => a.kidId === kidId && a.id === id) : undefined;

// The 404 an unknown administration, another kid's one and a malformed id all share.
const administrationRoute = (
	method: string,
	run: (
		state: State,
		res: ServerResponse,
		kid: KidRow,
		a: AdministrationStub,
		body: Body
	) => unknown
): RouteDef => ({
	method,
	pattern: /^\/v1\/kids\/([^/]+)\/vaccinations\/([^/]+)$/,
	access: 'caregiver',
	kidParam: true,
	run: async (state, req, res, _p, [kidId, id]) => {
		const body = method === 'PATCH' ? await decodeBody(req, res, ADMINISTRATION_FIELDS) : {};
		if (!body) return;
		const a = administrationOf(state, kidId, id);
		if (!a) return noKid(res);
		return run(state, res, state.kids.get(kidId) as KidRow, a, body);
	}
});

// dismissalLine in vaccines.go: 404 unless the kid has a schedule and the line is one of its evaluated lines.
const dismissalRoute = (
	method: string,
	run: (
		state: State,
		res: ServerResponse,
		kid: KidRow,
		key: string,
		line: string,
		body: Body
	) => unknown
): RouteDef => ({
	method,
	pattern: /^\/v1\/kids\/([^/]+)\/vaccine-dismissals\/([^/]+)$/,
	access: 'guardian',
	kidParam: true,
	run: async (state, req, res, _p, [kidId, line]) => {
		const body = method === 'PUT' ? await decodeBody(req, res, { reason: 'string' }) : {};
		if (!body) return;
		const kid = state.kids.get(kidId) as KidRow;
		const schemaId = kidSchedule(kid);
		if (
			schemaId === null ||
			!evaluate(state, kid, schemaId).lines.some((l) => l.line.id === line)
		) {
			return noKid(res);
		}
		return run(state, res, kid, `${kid.id}:${schemaId}:${line}`, line, body);
	}
});

const vaccineRoutes: RouteDef[] = [
	{
		method: 'GET',
		pattern: /^\/v1\/vaccine-schemas$/,
		access: 'parent',
		run: (_state, _req, res) =>
			send(res, 200, { schemas: [...STUB_SCHEDULES.values()].map((s) => s.info) })
	},
	{
		method: 'GET',
		pattern: /^\/v1\/vaccine-catalogue$/,
		access: 'parent',
		run: (_state, _req, res) =>
			send(res, 200, { antigens: VACCINE_ANTIGENS, products: VACCINE_PRODUCTS })
	},
	kidRoute('GET', 'vaccines', 'caregiver', (state, _req, res, kid) =>
		send(res, 200, vaccinesView(state, kid))
	),
	kidRoute('GET', 'vaccinations', 'caregiver', (state, _req, res, kid) => {
		const rows = state.vaccinations
			.filter((a) => a.kidId === kid.id)
			.sort((a, b) =>
				a.givenOn === b.givenOn ? (a.id < b.id ? 1 : -1) : a.givenOn < b.givenOn ? 1 : -1
			);
		send(res, 200, { vaccinations: rows.map(administrationView) });
	}),
	kidRoute('POST', 'vaccinations', 'caregiver', async (state, req, res, kid) => {
		const body = await decodeBody(req, res, ADMINISTRATION_FIELDS);
		if (!body) return;
		const a: AdministrationStub = {
			kidId: kid.id,
			id: '',
			givenOn: '',
			antigens: [],
			lot: null,
			product: null,
			note: null
		};
		const field = mergeAdministration(a, body, kid);
		if (field) return error(res, 422, 'invalid', field);
		const count = state.vaccinations.filter((r) => r.kidId === kid.id).length;
		if (count >= MAX_VACCINE_ADMINISTRATIONS_PER_KID) return error(res, 409, 'limit');
		a.id = nextId(state, 'v');
		state.vaccinations.push(a);
		send(res, 201, administrationView(a));
	}),
	administrationRoute('GET', (_state, res, _kid, a) => send(res, 200, administrationView(a))),
	administrationRoute('PATCH', (_state, res, kid, a, body) => {
		const merged = structuredClone(a);
		const field = mergeAdministration(merged, body, kid);
		if (field) return error(res, 422, 'invalid', field);
		Object.assign(a, merged);
		send(res, 200, administrationView(a));
	}),
	administrationRoute('DELETE', (state, res, _kid, a) => {
		state.vaccinations = state.vaccinations.filter((r) => r !== a);
		send(res, 204);
	}),
	dismissalRoute('PUT', (state, res, _kid, key, line, body) => {
		if (!DISMISS_REASONS.includes(body.reason as string)) {
			return error(res, 422, 'invalid', 'reason');
		}
		state.dismissals.set(key, body.reason as string);
		send(res, 200, { line, reason: body.reason });
	}),
	dismissalRoute('DELETE', (state, res, _kid, key) => {
		state.dismissals.delete(key);
		send(res, 204);
	}),
	kidRoute('POST', 'vaccine-dismissals', 'guardian', async (state, req, res, kid) => {
		const body = await decode(req, res, ['before', 'reason']);
		if (!body) return;
		const schemaId = kidSchedule(kid);
		if (schemaId === null) return noKid(res);
		if (!validDate(body.before)) return error(res, 422, 'invalid', 'before');
		if (!DISMISS_REASONS.includes(body.reason)) return error(res, 422, 'invalid', 'reason');
		const lines = evaluate(state, kid, schemaId).lines.filter(
			(l) =>
				(l.line.kind === 'routine' || l.line.kind === 'may_not_be_needed') &&
				!['given', 'dismissed', 'available'].includes(l.status) &&
				l.fromDate < body.before
		);
		for (const l of lines) state.dismissals.set(`${kid.id}:${schemaId}:${l.line.id}`, body.reason);
		send(res, 200, { dismissed: lines.length });
	})
];

routes.push(
	...TRACKER_KEYS.flatMap(trackerRoutes),
	...overviewRoutes,
	...growthRoutes,
	...vaccineRoutes
);

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
		if (method === 'PATCH' && state.scenario === 'patch_unavailable') {
			return error(res, 503, 'unavailable');
		}
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
		const body = req.method === 'GET' ? '' : await readBody(req);
		state.requests.push({
			method: req.method ?? 'GET',
			path: url.pathname,
			body: body || undefined,
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
	// A stored schedule id the API no longer ships, which PATCH /v1/kids refuses to write.
	if (req.method === 'POST' && url.pathname === '/__vaccine-schema') {
		const body = JSON.parse((await readBody(req)) || '{}') as { kid?: string; schema?: string };
		const kid = state.kids.get(body.kid ?? '');
		if (!kid || typeof body.schema !== 'string') return send(res, 400);
		kid.vaccineSchema = body.schema;
		return send(res, 204);
	}
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

export async function storeVaccineSchema(kid: string, schema: string): Promise<void> {
	const res = await fetch(`${STUB_URL}/__vaccine-schema`, {
		method: 'POST',
		body: JSON.stringify({ kid, schema })
	});
	if (res.status !== 204) throw new Error(`storeVaccineSchema: ${res.status}`);
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

// Calls the stub as a parent, as another device would.
export async function stubAs(who: { username: string; password: string }) {
	const token = await openSession(who.username, who.password);
	return async (method: string, path: string, body?: unknown) => {
		const res = await fetch(`${STUB_URL}${path}`, {
			method,
			headers: {
				authorization: `Bearer ${STUB_TOKEN}`,
				'x-session-token': token,
				'content-type': 'application/json'
			},
			body: body === undefined ? undefined : JSON.stringify(body)
		});
		const text = await res.text();
		return { status: res.status, body: text ? JSON.parse(text) : null };
	};
}

if (process.argv[1] && resolve(process.argv[1]) === import.meta.filename) startStub();
