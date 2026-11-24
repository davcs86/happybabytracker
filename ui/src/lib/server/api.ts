import {
	API_TIMEOUT_MS,
	EXPORT_DISPOSITION_PATTERN,
	EXPORT_TIMEOUT_MS,
	ID_PATTERN,
	TOKEN_PATTERN,
	VACCINE_LINE_PATTERN
} from './constants';
import type { Env } from './env';
import {
	AGE_BANDS,
	BOTTLE_CONTENTS,
	DIAPER_KINDS,
	GROWTH_SCORE_REASONS,
	GROWTH_SOURCES,
	LENGTH_UNITS,
	MEASURES,
	MILESTONE_AGES,
	MILESTONE_DOMAINS,
	MILESTONE_POSITIONS,
	MODULES,
	NURSING_SIDES,
	PUMPING_SIDES,
	SEVERITIES,
	SYMPTOMS,
	TEMP_METHODS,
	TEMP_UNITS,
	TIMELINE_TYPE,
	TIMER_KINDS,
	TRACKER_KINDS,
	VACCINE_AGE_UNITS,
	VACCINE_DISMISS_REASONS,
	VACCINE_KINDS,
	VACCINE_STATUSES,
	VOLUME_UNITS,
	WEIGHT_UNITS
} from '../constants';
import type {
	Administration,
	AdministrationInput,
	AdminParent,
	Bottle,
	BottleSource,
	Curves,
	CurvesQuery,
	DaySummary,
	Diaper,
	EntryBody,
	ExportInput,
	Grant,
	GrowthLength,
	GrowthRow,
	GrowthScore,
	GrowthSourceInfo,
	GrowthWeight,
	DoseQuery,
	Home,
	HomeDue,
	Kid,
	KidInput,
	ListQuery,
	Meal,
	Medication,
	MedicationDose,
	MilestoneChecklist,
	MilestoneEntry,
	MilestoneItem,
	MilestoneReached,
	Milestones,
	Milk,
	ModuleId,
	ModuleState,
	NewSession,
	NtfyState,
	Nursing,
	NursingSide,
	Page,
	Parent,
	ParentLink,
	Pumping,
	PumpingSideVolume,
	PushDevice,
	PushState,
	PushSubscriptionInput,
	Role,
	Schedule,
	Session,
	SessionInfo,
	SessionKind,
	Sleep,
	Symptom,
	Temperature,
	TimelineItem,
	TimelineQuery,
	TimerKind,
	TokenInfo,
	TrackerEntries,
	TrackerKind,
	Units,
	VaccineAge,
	VaccineCatalogue,
	VaccineDismissReason,
	VaccineGroup,
	VaccineLine,
	Vaccines,
	VaccineSchemaInfo,
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

export type ExportResult =
	| { ok: true; body: ReadableStream<Uint8Array>; contentType: string; disposition: string }
	| Extract<ApiResult<never>, { ok: false }>;

type Endpoint =
	| 'health'
	| 'reminders_health'
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
	| `${'list' | 'get' | 'create' | 'update' | 'delete'}_${TrackerKind | 'growth' | 'vaccination' | 'medication' | 'dose' | 'milestone'}`
	| 'push_state'
	| 'list_push_devices'
	| 'upsert_push_device'
	| 'rename_push_device'
	| 'delete_push_device'
	| 'set_push_optin'
	| 'clear_push_optin'
	| 'ntfy_state'
	| 'enable_ntfy'
	| 'regenerate_ntfy'
	| 'disable_ntfy'
	| 'set_ntfy_optin'
	| 'clear_ntfy_optin'
	| 'growth_curves'
	| 'growth_sources'
	| 'end_timer'
	| 'switch_side'
	| 'timeline'
	| 'summary'
	| 'list_modules'
	| 'set_module'
	| 'home'
	| 'vaccines'
	| 'vaccine_schemas'
	| 'vaccine_catalogue'
	| 'set_dismissal'
	| 'clear_dismissal'
	| 'dismiss_before'
	| 'export_pdf';

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

function number(v: unknown): number {
	if (typeof v !== 'number' || !Number.isFinite(v)) throw new ShapeError();
	return v;
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
		growthSource: oneOf(k.growth_source, GROWTH_SOURCES),
		vaccineSchema: stringOrNull(k.vaccine_schema),
		reminderOverdueMinutes: int(k.reminder_overdue_minutes),
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

function toBottleSource(v: unknown): BottleSource {
	const o = object(v);
	return { source: oneOf(o.source, BOTTLE_CONTENTS), volume: toVolume(o.volume) };
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
		sources: array(o.sources, toBottleSource),
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

function toTemperature(v: unknown): Temperature {
	const o = object(v);
	return {
		cHundredths: int(o.c_hundredths),
		fHundredths: int(o.f_hundredths),
		typedUnit: oneOf(o.typed_unit, TEMP_UNITS)
	};
}

function toSymptom(v: unknown): Symptom {
	const o = object(v);
	return {
		id: string(o.id),
		at: string(o.at),
		symptom: oneOf(o.symptom, SYMPTOMS),
		severity: o.severity === null ? null : oneOf(o.severity, SEVERITIES),
		otherText: stringOrNull(o.other_text),
		temperature: nullOr(o.temperature, toTemperature),
		method: o.method === null ? null : oneOf(o.method, TEMP_METHODS),
		note: stringOrNull(o.note)
	};
}

function toPumpingSideVolume(v: unknown): PumpingSideVolume {
	const o = object(v);
	return { side: oneOf(o.side, NURSING_SIDES), volume: toVolume(o.volume) };
}

function toPumping(v: unknown): Pumping {
	const o = object(v);
	return {
		id: string(o.id),
		startedAt: string(o.started_at),
		durationSeconds: int(o.duration_seconds),
		volume: toVolume(o.volume),
		side: oneOf(o.side, PUMPING_SIDES),
		sides: array(o.sides, toPumpingSideVolume),
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

function toMilk(v: unknown): Milk {
	const o = object(v);
	const gain = object(o.gain);
	const volume = object(o.volume);
	return {
		gain: { grams: int(gain.grams), ozHundredths: int(gain.oz_hundredths) },
		volume: { ml: int(volume.ml), flozHundredths: int(volume.floz_hundredths) }
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
		note: stringOrNull(o.note),
		weightBefore: nullOr(o.weight_before, toGrowthWeight),
		weightAfter: nullOr(o.weight_after, toGrowthWeight),
		milk: nullOr(o.milk, toMilk)
	};
}

const PARSERS: { [K in TrackerKind]: (v: unknown) => TrackerEntries[K] } = {
	bottles: toBottle,
	nursing: toNursing,
	pumping: toPumping,
	sleep: toSleep,
	diapers: toDiaper,
	meals: toMeal,
	symptoms: toSymptom
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
		correctedBirthDate: stringOrNull(o.corrected_birth_date),
		band: oneOf(o.band, AGE_BANDS),
		modules: list(o, 'modules', toModuleState),
		running: {
			sleep: running.sleep === null ? null : toSleep(running.sleep),
			nursing: running.nursing === null ? null : toNursing(running.nursing)
		},
		today: toDaySummary(o.today),
		due: list(o, 'due', toHomeDue)
	};
}

const nullOr = <T>(v: unknown, parse: (v: unknown) => T): T | null =>
	v === null ? null : parse(v);

function toVaccineAge(v: unknown): VaccineAge {
	const o = object(v);
	return { n: int(o.n), unit: oneOf(o.unit, VACCINE_AGE_UNITS) };
}

function toHomeDue(v: unknown): HomeDue {
	const o = object(v);
	if (o.kind === 'medication') {
		const m = object(o.medication);
		return {
			kind: 'medication',
			medication: { id: string(m.id), name: string(m.name) },
			dueAt: string(o.due_at),
			state: oneOf(o.state, ['due', 'overdue'])
		};
	}
	return {
		kind: oneOf(o.kind, ['vaccine']),
		from: toVaccineAge(o.from),
		status: oneOf(o.status, ['due', 'overdue']),
		count: int(o.count)
	};
}

function toVaccineSchemaInfo(v: unknown): VaccineSchemaInfo {
	const o = object(v);
	return {
		id: string(o.id),
		name: string(o.name),
		authority: string(o.authority),
		edition: string(o.edition),
		retrieved: string(o.retrieved),
		originUrl: string(o.origin_url),
		licence: string(o.licence),
		attribution: string(o.attribution)
	};
}

const toDismissReason = (v: unknown): VaccineDismissReason => oneOf(v, VACCINE_DISMISS_REASONS);

function toVaccineLine(v: unknown): VaccineLine {
	const o = object(v);
	return {
		id: string(o.id),
		group: string(o.group),
		antigen: string(o.antigen),
		series: int(o.series),
		kind: oneOf(o.kind, VACCINE_KINDS),
		status: oneOf(o.status, VACCINE_STATUSES),
		fromDate: string(o.from_date),
		untilDate: string(o.until_date),
		givenOn: stringOrNull(o.given_on),
		administration: stringOrNull(o.administration),
		dismissal: nullOr(o.dismissal, (d) => ({ reason: toDismissReason(object(d).reason) })),
		note: string(o.note)
	};
}

function toVaccineGroup(v: unknown): VaccineGroup {
	const o = object(v);
	return {
		from: toVaccineAge(o.from),
		fromDate: string(o.from_date),
		status: oneOf(o.status, VACCINE_STATUSES),
		lines: list(o, 'lines', toVaccineLine)
	};
}

function toVaccines(v: unknown): Vaccines {
	const o = object(v);
	return {
		schema: nullOr(o.schema, (s) => ({
			...toVaccineSchemaInfo(s),
			omitted: list(s, 'omitted', string)
		})),
		today: string(o.today),
		groups: list(o, 'groups', toVaccineGroup),
		also: list(o, 'also', toVaccineLine),
		recurring: list(o, 'recurring', (r) => {
			const rec = object(r);
			return {
				antigen: string(rec.antigen),
				lineId: string(rec.line_id),
				doses: list(rec, 'doses', (d) => {
					const dose = object(d);
					return { administration: string(dose.administration), givenOn: string(dose.given_on) };
				})
			};
		}),
		other: list(o, 'other', (d) => {
			const dose = object(d);
			return {
				administration: string(dose.administration),
				antigen: string(dose.antigen),
				givenOn: string(dose.given_on),
				product: stringOrNull(dose.product)
			};
		})
	};
}

function toVaccineCatalogue(v: unknown): VaccineCatalogue {
	return {
		antigens: list(v, 'antigens', (a) => {
			const o = object(a);
			return { id: string(o.id), name: string(o.name) };
		}),
		products: list(v, 'products', (p) => {
			const o = object(p);
			return { id: string(o.id), name: string(o.name), antigens: list(o, 'antigens', string) };
		})
	};
}

function toAdministration(v: unknown): Administration {
	const o = object(v);
	return {
		id: string(o.id),
		givenOn: string(o.given_on),
		antigens: list(o, 'antigens', string),
		lot: stringOrNull(o.lot),
		product: stringOrNull(o.product),
		note: stringOrNull(o.note)
	};
}

function toSchedule(v: unknown): Schedule {
	const o = object(v);
	switch (o.kind) {
		case 'none':
			return { kind: 'none' };
		case 'times':
			return { kind: 'times', times: array(o.times, string) };
		case 'every':
			return { kind: 'every', hours: int(o.hours) };
		default:
			throw new ShapeError();
	}
}

function toMedication(v: unknown): Medication {
	const o = object(v);
	return {
		id: string(o.id),
		name: string(o.name),
		dose: string(o.dose),
		schedule: toSchedule(o.schedule),
		startAt: string(o.start_at),
		endOn: stringOrNull(o.end_on),
		active: boolean(o.active),
		note: stringOrNull(o.note),
		lastDose: nullOr(o.last_dose, (d) => {
			const dose = object(d);
			return {
				id: string(dose.id),
				givenAt: string(dose.given_at),
				amount: stringOrNull(dose.amount)
			};
		}),
		nextDue: nullOr(o.next_due, (d) => ({ dueAt: string(object(d).due_at) })),
		doseCount: int(o.dose_count)
	};
}

function toMedicationDose(v: unknown): MedicationDose {
	const o = object(v);
	return {
		id: string(o.id),
		medication: string(o.medication),
		givenAt: string(o.given_at),
		amount: stringOrNull(o.amount),
		note: stringOrNull(o.note)
	};
}

function toMilestoneReached(v: unknown): MilestoneReached {
	const o = object(v);
	return { id: string(o.id), reachedOn: string(o.reached_on), note: stringOrNull(o.note) };
}

const toMilestoneDomain = (v: unknown) => oneOf(v, MILESTONE_DOMAINS);

function toMilestoneAge(v: unknown): number {
	const age = int(v);
	if (!(MILESTONE_AGES as readonly number[]).includes(age)) throw new ShapeError();
	return age;
}

function toMilestoneItem(v: unknown): MilestoneItem {
	const o = object(v);
	return {
		id: string(o.id),
		domain: toMilestoneDomain(o.domain),
		text: string(o.text),
		reached: nullOr(o.reached, toMilestoneReached)
	};
}

function toMilestoneChecklist(v: unknown): MilestoneChecklist {
	const o = object(v);
	return {
		ageMonths: toMilestoneAge(o.age_months),
		position: oneOf(o.position, MILESTONE_POSITIONS),
		items: list(o, 'items', toMilestoneItem)
	};
}

function toMilestones(v: unknown): Milestones {
	const o = object(v);
	const c = object(o.catalog);
	const age = object(o.age);
	return {
		catalog: {
			id: string(c.id),
			name: string(c.name),
			authority: string(c.authority),
			edition: string(c.edition),
			retrieved: string(c.retrieved),
			originUrl: string(c.origin_url),
			licence: string(c.licence),
			attribution: string(c.attribution),
			screeningNote: string(c.screening_note)
		},
		age: {
			chronologicalMonths: int(age.chronological_months),
			correctedMonths: intOrNull(age.corrected_months),
			basisMonths: int(age.basis_months)
		},
		catalogShown: boolean(o.catalog_shown),
		checklists: list(o, 'checklists', toMilestoneChecklist),
		custom: list(o, 'custom', (r) => ({ ...toMilestoneReached(r), text: string(object(r).text) })),
		retired: list(o, 'retired', (r) => {
			const row = object(r);
			return { ...toMilestoneReached(r), item: string(row.item), text: string(row.text) };
		}),
		reached: list(o, 'reached', (r) => {
			const row = object(r);
			return {
				...toMilestoneReached(r),
				item: string(row.item),
				ageMonths: toMilestoneAge(row.age_months),
				domain: toMilestoneDomain(row.domain),
				text: string(row.text)
			};
		})
	};
}

function toMilestoneEntry(v: unknown): MilestoneEntry {
	const o = object(v);
	return { ...toMilestoneReached(o), item: stringOrNull(o.item), text: string(o.text) };
}

function toPushState(v: unknown): PushState {
	const o = object(v);
	if (!boolean(o.enabled)) return { enabled: false };
	const publicKey = string(o.public_key);
	if (publicKey === '') throw new ShapeError();
	return { enabled: true, publicKey };
}

// Only these four fields are read: an endpoint or key in a reply is never copied out.
function toPushDevice(v: unknown): PushDevice {
	const o = object(v);
	return {
		id: string(o.id),
		label: string(o.label),
		createdAt: string(o.created_at),
		kids: array(o.kids, string)
	};
}

function toNtfyState(v: unknown): NtfyState {
	const o = object(v);
	if (!boolean(o.enabled)) return { enabled: false, hasTopic: boolean(o.has_topic) };
	return {
		enabled: true,
		server: string(o.server),
		topic: stringOrNull(o.topic),
		kids: array(o.kids, string)
	};
}

function toGrowthWeight(v: unknown): GrowthWeight {
	const o = object(v);
	return {
		grams: int(o.grams),
		ozHundredths: int(o.oz_hundredths),
		typedUnit: oneOf(o.typed_unit, WEIGHT_UNITS)
	};
}

function toGrowthLength(v: unknown): GrowthLength {
	const o = object(v);
	return {
		mm: int(o.mm),
		inHundredths: int(o.in_hundredths),
		typedUnit: oneOf(o.typed_unit, LENGTH_UNITS)
	};
}

function toGrowthScore(v: unknown): GrowthScore {
	const o = object(v);
	return boolean(o.scored)
		? { scored: true, z: number(o.z), percentile: number(o.percentile) }
		: { scored: false, reason: oneOf(o.reason, GROWTH_SCORE_REASONS) };
}

function toGrowthRow(v: unknown): GrowthRow {
	const o = object(v);
	const age = object(o.age);
	const scores = object(o.scores);
	return {
		id: string(o.id),
		measuredOn: string(o.measured_on),
		weight: nullOr(o.weight, toGrowthWeight),
		length: nullOr(o.length, toGrowthLength),
		head: nullOr(o.head, toGrowthLength),
		note: stringOrNull(o.note),
		ageDays: int(age.days),
		correctedDays: intOrNull(age.corrected_days),
		scores: {
			weight: nullOr(scores.weight, toGrowthScore),
			length: nullOr(scores.length, toGrowthScore),
			head: nullOr(scores.head, toGrowthScore),
			bmi: nullOr(scores.bmi, (b) => ({
				...toGrowthScore(b),
				value: number(object(b).value)
			}))
		}
	};
}

function toCurves(v: unknown): Curves {
	const o = object(v);
	return {
		source: oneOf(o.source, GROWTH_SOURCES),
		measure: oneOf(o.measure, MEASURES),
		sex: oneOf(o.sex, ['female', 'male']),
		covered: boolean(o.covered),
		reason: stringOrNull(o.reason),
		fromMonths: number(o.from_months),
		toMonths: number(o.to_months),
		lines: array(o.lines, (l) => {
			const line = object(l);
			return {
				percentile: number(line.percentile),
				points: array(line.points, (p) => {
					const point = object(p);
					return {
						ageMonths: number(point.age_months),
						metric: number(point.metric),
						imperial: nullOr(point.imperial, number),
						gap: point.gap === undefined ? false : boolean(point.gap)
					};
				})
			};
		}),
		correctedAgeAxis: boolean(o.corrected_age_axis)
	};
}

function toGrowthSourceInfo(v: unknown): GrowthSourceInfo {
	const o = object(v);
	return {
		id: oneOf(o.id, GROWTH_SOURCES),
		name: string(o.name),
		attribution: string(o.attribution),
		licence: string(o.licence),
		version: string(o.version),
		originUrl: string(o.origin_url),
		retrieved: string(o.retrieved),
		coverage: array(o.coverage, (c) => {
			const cov = object(c);
			return {
				measure: oneOf(cov.measure, MEASURES),
				fromMonths: number(cov.from_months),
				toMonths: number(cov.to_months)
			};
		})
	};
}

function curvesQuery(q: CurvesQuery): string {
	const p = new URLSearchParams({ measure: q.measure });
	if (q.fromMonths !== undefined) p.set('from_months', String(q.fromMonths));
	if (q.toMonths !== undefined) p.set('to_months', String(q.toMonths));
	return `?${p}`;
}

function queryString(q: TimelineQuery & { medication?: string }): string {
	const p = new URLSearchParams();
	if (q.from !== undefined) p.set('from', q.from);
	if (q.to !== undefined) p.set('to', q.to);
	if (q.limit !== undefined) p.set('limit', String(q.limit));
	if (q.cursor !== undefined) p.set('cursor', q.cursor);
	if (q.medication !== undefined) p.set('medication', q.medication);
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
	time_zone: k.timeZone,
	...(k.growthSource === undefined ? {} : { growth_source: k.growthSource }),
	...(k.vaccineSchema === undefined ? {} : { vaccine_schema: k.vaccineSchema }),
	...(k.reminderOverdueMinutes === undefined
		? {}
		: { reminder_overdue_minutes: k.reminderOverdueMinutes })
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

	function requestHeaders(accept: string, json: boolean): Record<string, string> {
		const headers: Record<string, string> = {
			authorization: `Bearer ${env.internalApiToken}`,
			accept
		};
		if (json) headers['content-type'] = 'application/json';
		if (session) headers['x-session-token'] = session;
		if (clientAddress) headers['x-client-address'] = clientAddress;
		return headers;
	}

	async function call<T>(
		endpoint: Endpoint,
		method: string,
		path: string,
		parse: ((body: unknown) => T) | null,
		body?: unknown
	): Promise<ApiResult<T>> {
		const headers = requestHeaders('application/json', body !== undefined);
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

		if (!res.ok) return refused(endpoint, res.status, raw);
		if (parse === null) return { ok: true, data: null as T };
		try {
			if (!parsed) throw new ShapeError();
			return { ok: true, data: parse(raw) };
		} catch {
			logFailure(endpoint, res.status);
			return { ok: false, status: res.status, reason: 'error' };
		}
	}

	function refused(endpoint: Endpoint, status: number, raw: unknown) {
		logFailure(endpoint, status);
		const failure: Extract<ApiResult<never>, { ok: false }> = {
			ok: false,
			status,
			reason: reasonFor(status)
		};
		const err = typeof raw === 'object' && raw !== null ? (raw as Wire) : {};
		if (typeof err.error === 'string') failure.code = err.error;
		if (status === 422 && typeof err.field === 'string') failure.field = err.field;
		return failure;
	}

	// A 200 is streamed unread, and only as a PDF attachment; a refusal's JSON is read as call() reads it.
	async function exportPdf(kidId: string, input: ExportInput): Promise<ExportResult> {
		const endpoint = 'export_pdf';
		let res: Response;
		let raw: unknown = undefined;
		try {
			res = await fetchFn(`${base}/v1/kids/${kidId}/export`, {
				method: 'POST',
				headers: requestHeaders('application/pdf', true),
				body: JSON.stringify(input),
				signal: AbortSignal.timeout(EXPORT_TIMEOUT_MS)
			});
			if (!res.ok) {
				raw = await res.json().catch((e: unknown) => {
					if (isAbort(e)) throw e;
				});
			}
		} catch (e) {
			logFailure(endpoint, isAbort(e) ? 'timeout' : 'network');
			return { ok: false, status: 0, reason: 'unavailable' };
		}
		if (!res.ok) return refused(endpoint, res.status, raw);
		const contentType = res.headers.get('content-type') ?? '';
		const disposition = res.headers.get('content-disposition') ?? '';
		if (
			res.body === null ||
			contentType !== 'application/pdf' ||
			!EXPORT_DISPOSITION_PATTERN.test(disposition)
		) {
			await res.body?.cancel().catch(() => {});
			logFailure(endpoint, res.status);
			return { ok: false, status: res.status, reason: 'error' };
		}
		return { ok: true, body: res.body, contentType, disposition };
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

	// A schedule line id outside VACCINE_LINE_PATTERN answers 404 without a call.
	const withKidLine = <T>(
		kidId: string,
		line: string,
		run: (kidId: string, line: string) => Promise<ApiResult<T>>
	) =>
		withId(kidId, (k) =>
			VACCINE_LINE_PATTERN.test(line) ? run(k, line) : Promise.resolve<ApiResult<T>>(NOT_FOUND)
		);

	return {
		health: () => call<null>('health', 'GET', '/healthz', null),
		remindersHealth: () => call<null>('reminders_health', 'GET', '/healthz/reminders', null),
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
		// input is the API's own JSON body, as the medication form builds it.
		medications: {
			list: (kidId: string) =>
				withId(kidId, (k) =>
					call('list_medication', 'GET', `/v1/kids/${k}/medications`, (b) =>
						list(b, 'medications', toMedication)
					)
				),
			get: (kidId: string, id: string) =>
				withKidEntry(kidId, id, (k, i) =>
					call('get_medication', 'GET', `/v1/kids/${k}/medications/${i}`, toMedication)
				),
			create: (kidId: string, input: EntryBody) =>
				withId(kidId, (k) =>
					call('create_medication', 'POST', `/v1/kids/${k}/medications`, toMedication, input)
				),
			update: (kidId: string, id: string, input: EntryBody) =>
				withKidEntry(kidId, id, (k, i) =>
					call('update_medication', 'PATCH', `/v1/kids/${k}/medications/${i}`, toMedication, input)
				),
			remove: (kidId: string, id: string) =>
				withKidEntry(kidId, id, (k, i) =>
					call<null>('delete_medication', 'DELETE', `/v1/kids/${k}/medications/${i}`, null)
				)
		},
		// input is the API's own JSON body; query.medication filters to one course.
		doses: {
			list: (kidId: string, query: DoseQuery = {}) =>
				known(query.medication === undefined || ID_PATTERN.test(query.medication), () =>
					withId(kidId, (k) =>
						call(
							'list_dose',
							'GET',
							`/v1/kids/${k}/medication-doses${queryString(query)}`,
							(b) => ({
								items: list(b, 'doses', toMedicationDose),
								nextCursor: stringOrNull(object(b).next_cursor)
							})
						)
					)
				),
			get: (kidId: string, id: string) =>
				withKidEntry(kidId, id, (k, i) =>
					call('get_dose', 'GET', `/v1/kids/${k}/medication-doses/${i}`, toMedicationDose)
				),
			create: (kidId: string, input: EntryBody) =>
				withId(kidId, (k) =>
					call('create_dose', 'POST', `/v1/kids/${k}/medication-doses`, toMedicationDose, input)
				),
			update: (kidId: string, id: string, input: EntryBody) =>
				withKidEntry(kidId, id, (k, i) =>
					call(
						'update_dose',
						'PATCH',
						`/v1/kids/${k}/medication-doses/${i}`,
						toMedicationDose,
						input
					)
				),
			remove: (kidId: string, id: string) =>
				withKidEntry(kidId, id, (k, i) =>
					call<null>('delete_dose', 'DELETE', `/v1/kids/${k}/medication-doses/${i}`, null)
				)
		},
		// input is the API's own JSON body, as readMilestoneForm builds it.
		milestones: {
			list: (kidId: string) =>
				withId(kidId, (k) =>
					call('list_milestone', 'GET', `/v1/kids/${k}/milestones`, toMilestones)
				),
			get: (kidId: string, id: string) =>
				withKidEntry(kidId, id, (k, i) =>
					call('get_milestone', 'GET', `/v1/kids/${k}/milestones/${i}`, toMilestoneEntry)
				),
			create: (kidId: string, input: EntryBody) =>
				withId(kidId, (k) =>
					call('create_milestone', 'POST', `/v1/kids/${k}/milestones`, toMilestoneEntry, input)
				),
			update: (kidId: string, id: string, input: EntryBody) =>
				withKidEntry(kidId, id, (k, i) =>
					call(
						'update_milestone',
						'PATCH',
						`/v1/kids/${k}/milestones/${i}`,
						toMilestoneEntry,
						input
					)
				),
			remove: (kidId: string, id: string) =>
				withKidEntry(kidId, id, (k, i) =>
					call<null>('delete_milestone', 'DELETE', `/v1/kids/${k}/milestones/${i}`, null)
				)
		},
		// A write answers 409 with code push_unavailable while Web Push is off; list and removal still work.
		push: {
			state: () => call('push_state', 'GET', '/v1/push', toPushState),
			devices: () =>
				call('list_push_devices', 'GET', '/v1/me/push-devices', (b) =>
					list(b, 'devices', toPushDevice)
				),
			// The only call that carries the endpoint and keys, and only in its body.
			upsertDevice: (input: PushSubscriptionInput) =>
				call(
					'upsert_push_device',
					'PUT',
					'/v1/me/push-devices',
					(b) => toPushDevice(object(b).device),
					{
						endpoint: input.endpoint,
						keys: { p256dh: input.keys.p256dh, auth: input.keys.auth },
						...(input.label === undefined ? {} : { label: input.label })
					}
				),
			renameDevice: (id: string, label: string) =>
				withId(id, (i) =>
					call(
						'rename_push_device',
						'PATCH',
						`/v1/me/push-devices/${i}`,
						(b) => toPushDevice(object(b).device),
						{ label }
					)
				),
			removeDevice: (id: string) =>
				withId(id, (i) =>
					call<null>('delete_push_device', 'DELETE', `/v1/me/push-devices/${i}`, null)
				),
			optIn: (kidId: string, deviceId: string) =>
				withKidEntry(kidId, deviceId, (k, d) =>
					call<null>('set_push_optin', 'PUT', `/v1/kids/${k}/push-devices/${d}`, null)
				),
			optOut: (kidId: string, deviceId: string) =>
				withKidEntry(kidId, deviceId, (k, d) =>
					call<null>('clear_push_optin', 'DELETE', `/v1/kids/${k}/push-devices/${d}`, null)
				)
		},
		// A write answers 409 with code ntfy_unavailable while ntfy is off; the topic never reaches a log.
		ntfy: {
			state: () => call('ntfy_state', 'GET', '/v1/me/ntfy', toNtfyState),
			enable: () => call('enable_ntfy', 'PUT', '/v1/me/ntfy', toNtfyState),
			regenerate: () => call('regenerate_ntfy', 'POST', '/v1/me/ntfy/regenerate', toNtfyState),
			disable: () => call<null>('disable_ntfy', 'DELETE', '/v1/me/ntfy', null),
			optIn: (kidId: string) =>
				withId(kidId, (k) => call<null>('set_ntfy_optin', 'PUT', `/v1/kids/${k}/ntfy`, null)),
			optOut: (kidId: string) =>
				withId(kidId, (k) => call<null>('clear_ntfy_optin', 'DELETE', `/v1/kids/${k}/ntfy`, null))
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
		home: (kidId: string) =>
			withId(kidId, (k) => call('home', 'GET', `/v1/kids/${k}/home`, toHome)),

		// input is the API's own JSON body, as readGrowthForm builds it.
		listGrowth: (kidId: string) =>
			withId(kidId, (k) =>
				call('list_growth', 'GET', `/v1/kids/${k}/growth`, (b) => list(b, 'growth', toGrowthRow))
			),
		getGrowth: (kidId: string, id: string) =>
			withKidEntry(kidId, id, (k, i) =>
				call('get_growth', 'GET', `/v1/kids/${k}/growth/${i}`, toGrowthRow)
			),
		createGrowth: (kidId: string, input: EntryBody) =>
			withId(kidId, (k) =>
				call('create_growth', 'POST', `/v1/kids/${k}/growth`, toGrowthRow, input)
			),
		updateGrowth: (kidId: string, id: string, input: EntryBody) =>
			withKidEntry(kidId, id, (k, i) =>
				call('update_growth', 'PATCH', `/v1/kids/${k}/growth/${i}`, toGrowthRow, input)
			),
		deleteGrowth: (kidId: string, id: string) =>
			withKidEntry(kidId, id, (k, i) =>
				call<null>('delete_growth', 'DELETE', `/v1/kids/${k}/growth/${i}`, null)
			),
		curves: (kidId: string, query: CurvesQuery) =>
			known((MEASURES as readonly string[]).includes(query.measure), () =>
				withId(kidId, (k) =>
					call('growth_curves', 'GET', `/v1/kids/${k}/growth/curves${curvesQuery(query)}`, toCurves)
				)
			),
		growthSources: () =>
			call('growth_sources', 'GET', '/v1/growth-sources', (b) =>
				list(b, 'sources', toGrowthSourceInfo)
			),

		vaccineSchemas: () =>
			call('vaccine_schemas', 'GET', '/v1/vaccine-schemas', (b) =>
				list(b, 'schemas', toVaccineSchemaInfo)
			),
		vaccineCatalogue: () =>
			call('vaccine_catalogue', 'GET', '/v1/vaccine-catalogue', toVaccineCatalogue),
		vaccines: (kidId: string) =>
			withId(kidId, (k) => call('vaccines', 'GET', `/v1/kids/${k}/vaccines`, toVaccines)),
		listVaccinations: (kidId: string) =>
			withId(kidId, (k) =>
				call('list_vaccination', 'GET', `/v1/kids/${k}/vaccinations`, (b) =>
					list(b, 'vaccinations', toAdministration)
				)
			),
		getVaccination: (kidId: string, id: string) =>
			withKidEntry(kidId, id, (k, i) =>
				call('get_vaccination', 'GET', `/v1/kids/${k}/vaccinations/${i}`, toAdministration)
			),
		// input is the API's own JSON body, as readVaccineForm builds it.
		createVaccination: (kidId: string, input: AdministrationInput) =>
			withId(kidId, (k) =>
				call('create_vaccination', 'POST', `/v1/kids/${k}/vaccinations`, toAdministration, input)
			),
		updateVaccination: (kidId: string, id: string, input: AdministrationInput) =>
			withKidEntry(kidId, id, (k, i) =>
				call(
					'update_vaccination',
					'PATCH',
					`/v1/kids/${k}/vaccinations/${i}`,
					toAdministration,
					input
				)
			),
		deleteVaccination: (kidId: string, id: string) =>
			withKidEntry(kidId, id, (k, i) =>
				call<null>('delete_vaccination', 'DELETE', `/v1/kids/${k}/vaccinations/${i}`, null)
			),
		setDismissal: (kidId: string, line: string, reason: string) =>
			withKidLine(kidId, line, (k, l) =>
				call(
					'set_dismissal',
					'PUT',
					`/v1/kids/${k}/vaccine-dismissals/${l}`,
					(b) => {
						const o = object(b);
						return { line: string(o.line), reason: toDismissReason(o.reason) };
					},
					{ reason }
				)
			),
		clearDismissal: (kidId: string, line: string) =>
			withKidLine(kidId, line, (k, l) =>
				call<null>('clear_dismissal', 'DELETE', `/v1/kids/${k}/vaccine-dismissals/${l}`, null)
			),
		// before is YYYY-MM-DD in the kid's zone; the answer is how many lines were dismissed.
		dismissBefore: (kidId: string, before: string, reason: string) =>
			withId(kidId, (k) =>
				call(
					'dismiss_before',
					'POST',
					`/v1/kids/${k}/vaccine-dismissals`,
					(b) => int(object(b).dismissed),
					{ before, reason }
				)
			),
		exportPdf: (kidId: string, input: ExportInput): Promise<ExportResult> =>
			ID_PATTERN.test(kidId) ? exportPdf(kidId, input) : Promise.resolve(NOT_FOUND)
	};
}
