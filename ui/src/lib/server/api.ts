import { API_TIMEOUT_MS, ID_PATTERN, TOKEN_PATTERN } from './constants';
import type { Env } from './env';
import {
	AGE_BANDS,
	BOTTLE_CONTENTS,
	DIAPER_KINDS,
	MODULES,
	NURSING_SIDES,
	PUMPING_SIDES,
	TIMELINE_TYPE,
	TIMER_KINDS,
	TRACKER_KINDS,
	VOLUME_UNITS
} from '../constants';
import type {
	AdminParent,
	Bottle,
	DaySummary,
	Diaper,
	EntryBody,
	Grant,
	Home,
	Kid,
	KidInput,
	ListQuery,
	Meal,
	ModuleId,
	ModuleState,
	NewSession,
	Nursing,
	NursingSide,
	Page,
	Parent,
	ParentLink,
	Pumping,
	Role,
	Session,
	SessionInfo,
	SessionKind,
	Sleep,
	TimelineItem,
	TimelineQuery,
	TimerKind,
	TokenInfo,
	TrackerEntries,
	TrackerKind,
	Units,
	Volume,
	VolumeTotal
} from '../types';

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

type Endpoint =
	| 'health'
	| 'login'
	| 'session'
	| 'logout'
	| 'peek_token'
	| 'redeem_token'
	| 'list_parents'
	| 'create_parent'
	| 'parent_link'
	| 'disable_parent'
	| 'enable_parent'
	| 'update_profile'
	| 'change_password'
	| 'list_sessions'
	| 'delete_session'
	| 'list_kids'
	| 'create_kid'
	| 'get_kid'
	| 'update_kid'
	| 'delete_kid'
	| 'list_access'
	| 'set_access'
	| 'remove_access'
	| `${'list' | 'get' | 'create' | 'update' | 'delete'}_${TrackerKind}`
	| 'end_timer'
	| 'switch_side'
	| 'timeline'
	| 'summary'
	| 'list_modules'
	| 'set_module'
	| 'home';

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

function boolean(v: unknown): boolean {
	if (typeof v !== 'boolean') throw new ShapeError();
	return v;
}

function intOrNull(v: unknown): number | null {
	if (v === null) return null;
	if (!Number.isInteger(v)) throw new ShapeError();
	return v as number;
}

function int(v: unknown): number {
	if (!Number.isInteger(v)) throw new ShapeError();
	return v as number;
}

function stringOrNull(v: unknown): string | null {
	return v === null ? null : string(v);
}

function array<T>(v: unknown, item: (row: unknown) => T): T[] {
	if (!Array.isArray(v)) throw new ShapeError();
	return v.map(item);
}

function list<T>(v: unknown, key: string, item: (row: unknown) => T): T[] {
	return array(object(v)[key], item);
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

const ROLES = ['guardian', 'caregiver'] as const satisfies readonly Role[];

function toKid(v: unknown): Kid {
	const k = object(v);
	return {
		id: string(k.id),
		name: string(k.name),
		birthDate: string(k.birth_date),
		sex: oneOf(k.sex, ['female', 'male']),
		gestationWeeks: intOrNull(k.gestation_weeks),
		gestationDays: intOrNull(k.gestation_days),
		timeZone: string(k.time_zone),
		role: oneOf(k.role, ROLES)
	};
}

function toGrant(v: unknown): Grant {
	const g = object(v);
	return {
		parentId: string(g.parent_id),
		username: string(g.username),
		displayName: string(g.display_name),
		role: oneOf(g.role, ROLES)
	};
}

function toAdminParent(v: unknown): AdminParent {
	const p = object(v);
	return {
		id: string(p.id),
		username: string(p.username),
		displayName: string(p.display_name),
		disabled: boolean(p.disabled),
		hasPassword: boolean(p.has_password),
		createdAt: string(p.created_at)
	};
}

// The response key says which link it is: a parent who never set a password gets an invite.
function toParentLink(v: unknown): ParentLink {
	const l = object(v);
	const expiresAt = string(l.expires_at);
	if (typeof l.reset_url === 'string') return { kind: 'reset', url: l.reset_url, expiresAt };
	return { kind: 'invite', url: string(l.invite_url), expiresAt };
}

function toSessionInfo(v: unknown): SessionInfo {
	const s = object(v);
	return {
		id: string(s.id),
		createdAt: string(s.created_at),
		lastSeenAt: string(s.last_seen_at),
		current: boolean(s.current)
	};
}

function toVolume(v: unknown): Volume {
	const o = object(v);
	return {
		ml: int(o.ml),
		floz_hundredths: int(o.floz_hundredths),
		typed_unit: oneOf(o.typed_unit, VOLUME_UNITS)
	};
}

function toBottle(v: unknown): Bottle {
	const o = object(v);
	const contents = array(o.contents, (c) => oneOf(c, BOTTLE_CONTENTS));
	if (contents.length === 0) throw new ShapeError();
	return {
		id: string(o.id),
		at: string(o.at),
		contents,
		volume: toVolume(o.volume),
		note: stringOrNull(o.note)
	};
}

function toDiaper(v: unknown): Diaper {
	const o = object(v);
	return {
		id: string(o.id),
		at: string(o.at),
		kind: oneOf(o.kind, DIAPER_KINDS),
		note: stringOrNull(o.note)
	};
}

function toMeal(v: unknown): Meal {
	const o = object(v);
	return {
		id: string(o.id),
		at: string(o.at),
		what: string(o.what),
		reaction: stringOrNull(o.reaction),
		note: stringOrNull(o.note)
	};
}

function toPumping(v: unknown): Pumping {
	const o = object(v);
	return {
		id: string(o.id),
		startedAt: string(o.started_at),
		durationSeconds: int(o.duration_seconds),
		volume: toVolume(o.volume),
		side: oneOf(o.side, PUMPING_SIDES),
		note: stringOrNull(o.note)
	};
}

function toSleep(v: unknown): Sleep {
	const o = object(v);
	return {
		id: string(o.id),
		startedAt: string(o.started_at),
		endedAt: stringOrNull(o.ended_at),
		note: stringOrNull(o.note)
	};
}

function toNursing(v: unknown): Nursing {
	const o = object(v);
	return {
		id: string(o.id),
		startedAt: string(o.started_at),
		endedAt: stringOrNull(o.ended_at),
		leftSeconds: int(o.left_seconds),
		rightSeconds: int(o.right_seconds),
		side: o.side === null ? null : oneOf(o.side, NURSING_SIDES),
		sideSince: stringOrNull(o.side_since),
		note: stringOrNull(o.note)
	};
}

const PARSERS: { [K in TrackerKind]: (v: unknown) => TrackerEntries[K] } = {
	bottles: toBottle,
	nursing: toNursing,
	pumping: toPumping,
	sleep: toSleep,
	diapers: toDiaper,
	meals: toMeal
};

function toPage<K extends TrackerKind>(kind: K, v: unknown): Page<TrackerEntries[K]> {
	return { items: list(v, kind, PARSERS[kind]), nextCursor: stringOrNull(object(v).next_cursor) };
}

// An item is its tracker's own view plus type and at; type is the singular name, kind the route name.
function toTimelineItem(v: unknown): TimelineItem {
	const o = object(v);
	const kind = TRACKER_KINDS.find((k) => TIMELINE_TYPE[k] === o.type);
	if (!kind) throw new ShapeError();
	return { kind, at: string(o.at), entry: PARSERS[kind](o) } as TimelineItem;
}

function toVolumeTotal(v: unknown): VolumeTotal {
	const o = object(v);
	return { count: int(o.count), ml: int(o.ml), floz_hundredths: int(o.floz_hundredths) };
}

function toDaySummary(v: unknown): DaySummary {
	const o = object(v);
	const nursing = object(o.nursing);
	const diapers = object(o.diapers);
	return {
		date: string(o.date),
		bottles: toVolumeTotal(o.bottles),
		nursing: { count: int(nursing.count), seconds: int(nursing.seconds) },
		pumping: toVolumeTotal(o.pumping),
		diapers: {
			total: int(diapers.total),
			wet: int(diapers.wet),
			dirty: int(diapers.dirty),
			both: int(diapers.both),
			dry: int(diapers.dry)
		},
		sleep: { seconds: int(object(o.sleep).seconds) },
		meals: { count: int(object(o.meals).count) }
	};
}

function toModuleState(v: unknown): ModuleState {
	const o = object(v);
	return {
		module: oneOf(o.module, MODULES),
		enabled: boolean(o.enabled),
		default: boolean(o.default),
		overridden: boolean(o.overridden)
	};
}

function toHome(v: unknown): Home {
	const o = object(v);
	const running = object(o.running);
	return {
		ageMonths: int(o.age_months),
		band: oneOf(o.band, AGE_BANDS),
		modules: list(o, 'modules', toModuleState),
		running: {
			sleep: running.sleep === null ? null : toSleep(running.sleep),
			nursing: running.nursing === null ? null : toNursing(running.nursing)
		},
		today: toDaySummary(o.today)
	};
}

function queryString(q: TimelineQuery): string {
	const p = new URLSearchParams();
	if (q.from !== undefined) p.set('from', q.from);
	if (q.to !== undefined) p.set('to', q.to);
	if (q.limit !== undefined) p.set('limit', String(q.limit));
	if (q.cursor !== undefined) p.set('cursor', q.cursor);
	if (q.types?.length) p.set('types', q.types.join(','));
	const s = p.toString();
	return s === '' ? '' : `?${s}`;
}

const kidBody = (k: KidInput) => ({
	name: k.name,
	birth_date: k.birthDate,
	sex: k.sex,
	gestation_weeks: k.gestationWeeks,
	gestation_days: k.gestationDays,
	time_zone: k.timeZone
});

const NOT_FOUND = { ok: false, status: 404, reason: 'not_found' } as const;
const PARENT_NOT_FOUND = { ...NOT_FOUND, code: 'parent_not_found' } as const;

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

	// An id outside ID_PATTERN answers 404 without a call, like an unknown id.
	function withId<T>(id: string, run: (id: string) => Promise<ApiResult<T>>) {
		return ID_PATTERN.test(id) ? run(id) : Promise.resolve<ApiResult<T>>(NOT_FOUND);
	}

	// A kind or module outside its list never reaches a path, whatever the caller cast.
	const known = <T>(ok: boolean, run: () => Promise<ApiResult<T>>) =>
		ok ? run() : Promise.resolve<ApiResult<T>>(NOT_FOUND);

	const withKidEntry = <T>(
		kidId: string,
		id: string,
		run: (kidId: string, id: string) => Promise<ApiResult<T>>
	) => withId(kidId, (k) => withId(id, (i) => run(k, i)));

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
		},

		listParents: () =>
			call('list_parents', 'GET', '/v1/admin/parents', (b) => list(b, 'parents', toAdminParent)),
		createParent: (username: string, displayName: string) =>
			call(
				'create_parent',
				'POST',
				'/v1/admin/parents',
				(b) => ({ parent: toAdminParent(object(b).parent), link: toParentLink(b) }),
				{ username, display_name: displayName }
			),
		parentLink: (id: string) =>
			withId(id, (i) =>
				call('parent_link', 'POST', `/v1/admin/parents/${i}/reset-link`, toParentLink)
			),
		setParentDisabled: (id: string, disabled: boolean) =>
			withId(id, (i) =>
				disabled
					? call<null>('disable_parent', 'POST', `/v1/admin/parents/${i}/disable`, null)
					: call<null>('enable_parent', 'POST', `/v1/admin/parents/${i}/enable`, null)
			),

		updateProfile: (displayName: string, units: Units) =>
			call('update_profile', 'PATCH', '/v1/me', toParent, { display_name: displayName, units }),
		changePassword: (current: string, next: string) =>
			call<null>('change_password', 'POST', '/v1/me/password', null, { current, new: next }),
		listSessions: () =>
			call('list_sessions', 'GET', '/v1/me/sessions', (b) => list(b, 'sessions', toSessionInfo)),
		deleteSession: (id: string) =>
			withId(id, (i) => call<null>('delete_session', 'DELETE', `/v1/me/sessions/${i}`, null)),

		listKids: () => call('list_kids', 'GET', '/v1/kids', (b) => list(b, 'kids', toKid)),
		createKid: (kid: KidInput) => call('create_kid', 'POST', '/v1/kids', toKid, kidBody(kid)),
		getKid: (id: string) => withId(id, (i) => call('get_kid', 'GET', `/v1/kids/${i}`, toKid)),
		// Sends every field: gestation null clears it, as PATCH reads an explicit null.
		updateKid: (id: string, kid: KidInput) =>
			withId(id, (i) => call('update_kid', 'PATCH', `/v1/kids/${i}`, toKid, kidBody(kid))),
		deleteKid: (id: string, confirmName: string) =>
			withId(id, (i) =>
				call<null>('delete_kid', 'DELETE', `/v1/kids/${i}`, null, { confirm_name: confirmName })
			),
		listAccess: (kidId: string) =>
			withId(kidId, (i) =>
				call('list_access', 'GET', `/v1/kids/${i}/access`, (b) => list(b, 'grants', toGrant))
			),
		// The API normalises the username; a dot segment would be rewritten by URL parsing.
		setAccess(kidId: string, username: string, role: Role): Promise<ApiResult<null>> {
			const name = username.trim();
			if (name === '' || name === '.' || name === '..') return Promise.resolve(PARENT_NOT_FOUND);
			return withId(kidId, (i) =>
				call<null>('set_access', 'PUT', `/v1/kids/${i}/access/${encodeURIComponent(name)}`, null, {
					role
				})
			);
		},
		removeAccess: (kidId: string, parentId: string) =>
			withId(kidId, (k) =>
				withId(parentId, (p) =>
					call<null>('remove_access', 'DELETE', `/v1/kids/${k}/access/${p}`, null)
				)
			),

		// input is the API's own JSON body, as readEntryForm builds it.
		tracker<K extends TrackerKind>(kind: K) {
			const parse = PARSERS[kind];
			const path = (k: string) => `/v1/kids/${k}/${kind}`;
			const ok = (TRACKER_KINDS as readonly string[]).includes(kind);
			return {
				list: (kidId: string, query: ListQuery = {}) =>
					known(ok, () =>
						withId(kidId, (k) =>
							call(`list_${kind}`, 'GET', `${path(k)}${queryString(query)}`, (b) => toPage(kind, b))
						)
					),
				get: (kidId: string, id: string) =>
					known(ok, () =>
						withKidEntry(kidId, id, (k, i) => call(`get_${kind}`, 'GET', `${path(k)}/${i}`, parse))
					),
				create: (kidId: string, input: EntryBody) =>
					known(ok, () =>
						withId(kidId, (k) => call(`create_${kind}`, 'POST', path(k), parse, input))
					),
				update: (kidId: string, id: string, input: EntryBody) =>
					known(ok, () =>
						withKidEntry(kidId, id, (k, i) =>
							call(`update_${kind}`, 'PATCH', `${path(k)}/${i}`, parse, input)
						)
					),
				remove: (kidId: string, id: string) =>
					known(ok, () =>
						withKidEntry(kidId, id, (k, i) =>
							call<null>(`delete_${kind}`, 'DELETE', `${path(k)}/${i}`, null)
						)
					)
			};
		},
		// at is an RFC 3339 instant; left out it is now.
		endTimer: <K extends TimerKind>(kind: K, kidId: string, id: string, at?: string) =>
			known((TIMER_KINDS as readonly string[]).includes(kind), () =>
				withKidEntry(kidId, id, (k, i) =>
					call(
						'end_timer',
						'POST',
						`/v1/kids/${k}/${kind}/${i}/end`,
						PARSERS[kind],
						at === undefined ? {} : { at }
					)
				)
			),
		switchNursingSide: (kidId: string, id: string, side: NursingSide, at?: string) =>
			withKidEntry(kidId, id, (k, i) =>
				call(
					'switch_side',
					'POST',
					`/v1/kids/${k}/nursing/${i}/side`,
					toNursing,
					at === undefined ? { side } : { side, at }
				)
			),
		timeline: (kidId: string, query: TimelineQuery = {}) =>
			withId(kidId, (k) =>
				call('timeline', 'GET', `/v1/kids/${k}/timeline${queryString(query)}`, (b) => ({
					items: list(b, 'items', toTimelineItem),
					nextCursor: stringOrNull(object(b).next_cursor)
				}))
			),
		// from and to are YYYY-MM-DD in the kid's zone.
		summary: (kidId: string, from: string, to: string) =>
			withId(kidId, (k) =>
				call('summary', 'GET', `/v1/kids/${k}/summary?${new URLSearchParams({ from, to })}`, (b) =>
					list(b, 'days', toDaySummary)
				)
			),
		modules: (kidId: string) =>
			withId(kidId, (k) =>
				call('list_modules', 'GET', `/v1/kids/${k}/modules`, (b) =>
					list(b, 'modules', toModuleState)
				)
			),
		setModule: (kidId: string, module: ModuleId, enabled: boolean) =>
			known((MODULES as readonly string[]).includes(module), () =>
				withId(kidId, (k) =>
					call<null>('set_module', 'PUT', `/v1/kids/${k}/modules/${module}`, null, { enabled })
				)
			),
		home: (kidId: string) => withId(kidId, (k) => call('home', 'GET', `/v1/kids/${k}/home`, toHome))
	};
}
