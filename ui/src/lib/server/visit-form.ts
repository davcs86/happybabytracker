import { fail } from '@sveltejs/kit';
import {
	DEFAULT_VISIT_DURATION_MIN,
	MAX_VISITS_PER_KID,
	NOTE_MAX_LEN,
	VISIT_DURATION_MIN,
	VISIT_HORIZON_DAYS,
	VISIT_KINDS,
	VISIT_LOCATION_MAX_LEN,
	VISIT_PROVIDER_MAX_LEN
} from '../constants';
import { instantToLocalInput, zonedLocalToInstant } from '../time-zone';
import type { EntryBody, Visit } from '../types';
import { failAction, type ApiFailure } from './outcome';

// The form's names are the API's keys, in the API's order of checks.
const FIELDS = ['starts_at', 'duration_min', 'kind', 'provider', 'location', 'note'] as const;
type Name = (typeof FIELDS)[number];
type Original = `${Name}_original`;

export type VisitFormValues = Record<Name, string> & Partial<Record<Original, string>>;
export type VisitMode = { kind: 'create' | 'edit'; zone: string };

// Keyed by the API's field names; never a typed value in a message.
export const VISIT_FIELD_MESSAGES: Record<string, string> = {
	starts_at: `Enter a date and time from the birth date to ${VISIT_HORIZON_DAYS} days from now`,
	duration_min: `Enter a length from ${VISIT_DURATION_MIN.min} to ${VISIT_DURATION_MIN.max} minutes`,
	kind: 'Choose a kind',
	provider: `Enter a doctor or clinic of up to ${VISIT_PROVIDER_MAX_LEN} characters`,
	location: `Enter a place of up to ${VISIT_LOCATION_MAX_LEN} characters`,
	note: `Enter a reason of up to ${NOTE_MAX_LEN} characters`
};

// The page spec's refusals, keyed by the API's 409 codes.
export const VISIT_REFUSALS: Record<string, string> = {
	has_outcome:
		'This visit has notes or records from the visit. Remove them first, or delete the visit.',
	not_past: 'This visit hasn’t happened yet.',
	stale: 'Someone changed this visit on another phone. It’s shown now; your entries are kept.',
	limit: `This child has the most visits the app keeps (${MAX_VISITS_PER_KID.toLocaleString('en-US')}).`
};

export const visitCreateValues = (now: Date, zone: string): VisitFormValues => ({
	starts_at: instantToLocalInput(now.toISOString(), zone),
	duration_min: String(DEFAULT_VISIT_DURATION_MIN),
	kind: '',
	provider: '',
	location: '',
	note: ''
});

export function visitEditValues(v: Visit, zone: string): VisitFormValues {
	const shown: Record<Name, string> = {
		starts_at: instantToLocalInput(v.startsAt, zone),
		duration_min: String(v.durationMin),
		kind: v.kind,
		provider: v.provider ?? '',
		location: v.location ?? '',
		note: v.note ?? ''
	};
	const originals = Object.fromEntries(FIELDS.map((f) => [`${f}_original`, shown[f]]));
	return { ...shown, ...originals };
}

const MINUTES = /^\d{1,4}$/;
const TEXT_MAX: Record<'provider' | 'location' | 'note', number> = {
	provider: VISIT_PROVIDER_MAX_LEN,
	location: VISIT_LOCATION_MAX_LEN,
	note: NOTE_MAX_LEN
};

type Outcome = { input: EntryBody } | { field: string };

// An edit sends only what differs from its hidden original and fails closed when one is missing; a create sends
// every required key and the optional texts that are not blank.
function build(v: VisitFormValues, mode: VisitMode): Outcome {
	const edit = mode.kind === 'edit';
	const input: EntryBody = {};
	const changed = (f: Name) => {
		if (!edit) return true;
		const o = v[`${f}_original`];
		return o === undefined ? null : v[f].trim() !== o.trim();
	};

	for (const f of FIELDS) {
		const c = changed(f);
		if (c === null) return { field: f };
		if (!c) continue;
		const text = v[f].trim();
		if (f === 'starts_at') {
			const instant = zonedLocalToInstant(text, mode.zone);
			if (instant === null) return { field: f };
			input.starts_at = instant;
		} else if (f === 'duration_min') {
			const n = MINUTES.test(text) ? Number(text) : NaN;
			if (!(n >= VISIT_DURATION_MIN.min && n <= VISIT_DURATION_MIN.max)) return { field: f };
			input.duration_min = n;
		} else if (f === 'kind') {
			if (!(VISIT_KINDS as readonly string[]).includes(text)) return { field: f };
			input.kind = text;
		} else {
			if ([...text].length > TEXT_MAX[f]) return { field: f };
			if (text !== '') input[f] = text;
			else if (edit) input[f] = null;
		}
	}
	return { input };
}

export function readVisitForm(
	form: FormData,
	mode: VisitMode
): { values: VisitFormValues; input: EntryBody | null; field: string | null } {
	const text = (k: string) => String(form.get(k) ?? '');
	const values = Object.fromEntries(FIELDS.map((f) => [f, text(f)])) as VisitFormValues;
	if (mode.kind === 'edit') {
		for (const f of FIELDS) {
			const o = form.get(`${f}_original`);
			if (o !== null) values[`${f}_original`] = String(o);
		}
	}
	const r = build(values, mode);
	return 'field' in r
		? { values, input: null, field: r.field }
		: { values, input: r.input, field: null };
}

export const visitFieldFailure = <T extends object>(
	values: VisitFormValues,
	field: string,
	extra: T
) => fail(422, { ...extra, values, field, message: VISIT_FIELD_MESSAGES[field] });

// A visit rule (409) on the form, anything else as failAction decides.
export function visitRefusal<T extends object>(r: ApiFailure, extra: T) {
	if (r.reason === 'conflict' && r.code && Object.hasOwn(VISIT_REFUSALS, r.code)) {
		return fail(409, { ...extra, message: VISIT_REFUSALS[r.code] });
	}
	return failAction(r, extra);
}

// A refused field lands on its input, a visit rule or a cap on the form.
export function visitApiFailure<T extends object>(
	values: VisitFormValues,
	r: ApiFailure,
	extra: T
) {
	if (r.reason === 'invalid' && r.field && Object.hasOwn(VISIT_FIELD_MESSAGES, r.field)) {
		return visitFieldFailure(values, r.field, extra);
	}
	return visitRefusal(r, { ...extra, values, field: null as string | null });
}
