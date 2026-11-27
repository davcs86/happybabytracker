import { fail } from '@sveltejs/kit';
import {
	MAX_EVERY_HOURS,
	MAX_SCHEDULE_TIMES,
	MEDICATION_TEXT_MAX_LEN,
	MIN_EVERY_HOURS,
	NOTE_MAX_LEN
} from '../constants';
import { instantToLocalInput, zonedLocalToInstant } from '../time-zone';
import type { EntryBody, Medication, MedicationDose, Schedule } from '../types';
import { ID_PATTERN } from './constants';
import { ENTRY_FIELD_MESSAGES } from './entry-failure';
import { failAction, type ApiFailure } from './outcome';

const KEPT = ['given_at', 'amount', 'note'] as const;
type Original = `${(typeof KEPT)[number]}_original`;

export type DoseFormValues = {
	amount: string;
	medication?: string;
	given_at?: string;
	note?: string;
} & Partial<Record<Original, string>>;
// The one-tap form names its course; an edit reads the dose's time on the kid's clock.
export type DoseMode = { kind: 'log' } | { kind: 'edit'; zone: string };

// Keyed by the API's field names; never a typed value in a message.
export const DOSE_FIELD_MESSAGES: Record<string, string> = {
	medication: 'This course is no longer in the list',
	given_at: ENTRY_FIELD_MESSAGES.at,
	amount: `Enter an amount of up to ${MEDICATION_TEXT_MAX_LEN} characters`,
	note: ENTRY_FIELD_MESSAGES.note
};
export const DOSE_LIMIT_MESSAGE = 'This child has the maximum number of logged doses';

export function doseEditValues(d: MedicationDose, zone: string): DoseFormValues {
	const shown = {
		given_at: instantToLocalInput(d.givenAt, zone),
		amount: d.amount ?? '',
		note: d.note ?? ''
	};
	return {
		...shown,
		given_at_original: shown.given_at,
		amount_original: shown.amount,
		note_original: shown.note
	};
}

const tooLong = (text: string, max: number) => [...text].length > max;

type Outcome = { input: EntryBody } | { field: string };

// The time, the amount and the note are checked in the API's order. An edit sends only what differs from its
// hidden original and fails closed when one is missing; a blank time keeps the stored one.
function buildEdit(v: DoseFormValues & { given_at: string; note: string }, zone: string): Outcome {
	const input: EntryBody = {};

	if (v.given_at_original === undefined) return { field: 'given_at' };
	const at = v.given_at.trim();
	if (at !== '' && at !== v.given_at_original.trim()) {
		const instant = zonedLocalToInstant(at, zone);
		if (instant === null) return { field: 'given_at' };
		input.given_at = instant;
	}

	if (v.amount_original === undefined) return { field: 'amount' };
	const amount = v.amount.trim();
	if (amount !== v.amount_original.trim()) {
		if (tooLong(amount, MEDICATION_TEXT_MAX_LEN)) return { field: 'amount' };
		input.amount = amount === '' ? null : amount;
	}

	if (v.note_original === undefined) return { field: 'note' };
	const note = v.note.trim();
	if (note !== v.note_original.trim()) {
		if (tooLong(note, NOTE_MAX_LEN)) return { field: 'note' };
		input.note = note === '' ? null : note;
	}
	return { input };
}

function buildLog(v: DoseFormValues & { medication: string }): Outcome {
	if (!ID_PATTERN.test(v.medication)) return { field: 'medication' };
	const amount = v.amount.trim();
	if (tooLong(amount, MEDICATION_TEXT_MAX_LEN)) return { field: 'amount' };
	return { input: { medication: v.medication, ...(amount === '' ? {} : { amount }) } };
}

export function readDoseForm(
	form: FormData,
	mode: DoseMode
): { values: DoseFormValues; input: EntryBody | null; field: string | null } {
	const text = (k: string) => String(form.get(k) ?? '');
	if (mode.kind === 'log') {
		const values = { medication: text('medication'), amount: text('amount') };
		const r = buildLog(values);
		return 'field' in r
			? { values, input: null, field: r.field }
			: { values, input: r.input, field: null };
	}
	const values: DoseFormValues & { given_at: string; note: string } = {
		given_at: text('given_at'),
		amount: text('amount'),
		note: text('note')
	};
	for (const k of KEPT) {
		const o = form.get(`${k}_original`);
		if (o !== null) values[`${k}_original`] = String(o);
	}
	const r = buildEdit(values, mode.zone);
	return 'field' in r
		? { values, input: null, field: r.field }
		: { values, input: r.input, field: null };
}

const refusedField = <V, T extends object>(
	messages: Record<string, string>,
	values: V,
	field: string,
	extra: T
) => fail(422, { ...extra, values, field, message: messages[field] });

// A refused field lands on its input, a cap on the form, anything else as failAction decides.
function apiRefusal<V, T extends object>(
	messages: Record<string, string>,
	limit: string,
	values: V,
	r: ApiFailure,
	extra: T
) {
	if (r.reason === 'invalid' && r.field && Object.hasOwn(messages, r.field)) {
		return refusedField(messages, values, r.field, extra);
	}
	if (r.reason === 'conflict' && r.code === 'limit') {
		return fail(409, { ...extra, values, field: null as string | null, message: limit });
	}
	return failAction(r, { ...extra, values, field: null as string | null });
}

export const doseFieldFailure = <T extends object>(
	values: DoseFormValues,
	field: string,
	extra: T
) => refusedField(DOSE_FIELD_MESSAGES, values, field, extra);

export const doseApiFailure = <T extends object>(values: DoseFormValues, r: ApiFailure, extra: T) =>
	apiRefusal(DOSE_FIELD_MESSAGES, DOSE_LIMIT_MESSAGE, values, r, extra);

const TEXT_ORIGINALS = [
	'name',
	'dose',
	'schedule',
	'start_at',
	'end_on',
	'active',
	'note'
] as const;
type CourseOriginal = `${(typeof TEXT_ORIGINALS)[number]}_original`;

// kind and hours are what the radios and the hours input post; times repeats, one per slot.
export type CourseFormValues = {
	name: string;
	dose: string;
	kind: string;
	times: string[];
	hours: string;
	start_at: string;
	end_on: string;
	active: boolean;
	note: string;
} & Partial<Record<CourseOriginal, string>>;
export type CourseMode = { kind: 'create' | 'edit'; zone: string };

// Keyed by the API's field names; never a typed value in a message.
export const COURSE_FIELD_MESSAGES: Record<string, string> = {
	name: `Enter a name of up to ${MEDICATION_TEXT_MAX_LEN} characters`,
	dose: `Enter a dose of up to ${MEDICATION_TEXT_MAX_LEN} characters`,
	schedule: `Choose as needed, up to ${MAX_SCHEDULE_TIMES} different times of day, or a whole number of hours from ${MIN_EVERY_HOURS} to ${MAX_EVERY_HOURS}`,
	start_at: 'Enter a start that is not before the birth date',
	end_on: 'Enter a last day that is a real date and not before the start',
	active: 'Choose whether the course is active',
	note: ENTRY_FIELD_MESSAGES.note
};
export const COURSE_LIMIT_MESSAGE = 'This child has the maximum number of medication courses';

export const courseCreateValues = (): CourseFormValues => ({
	name: '',
	dose: '',
	kind: 'none',
	times: [],
	hours: '',
	start_at: '',
	end_on: '',
	active: true,
	note: ''
});

// The stored schedule as one string: none, times:HH:MM,HH:MM or every:N.
const scheduleKey = (s: Schedule): string =>
	s.kind === 'times'
		? `times:${[...s.times].sort().join(',')}`
		: s.kind === 'every'
			? `every:${s.hours}`
			: 'none';

export function courseEditValues(c: Medication, zone: string) {
	const shown = {
		name: c.name,
		dose: c.dose,
		kind: c.schedule.kind as string,
		times: c.schedule.kind === 'times' ? [...c.schedule.times] : [],
		hours: c.schedule.kind === 'every' ? String(c.schedule.hours) : '',
		start_at: instantToLocalInput(c.startAt, zone),
		end_on: c.endOn ?? '',
		active: c.active,
		note: c.note ?? ''
	};
	return {
		...shown,
		name_original: shown.name,
		dose_original: shown.dose,
		schedule_original: scheduleKey(c.schedule),
		start_at_original: shown.start_at,
		end_on_original: shown.end_on,
		active_original: String(shown.active),
		note_original: shown.note
	};
}

const CLOCK = /^([01]\d|2[0-3]):[0-5]\d$/;
const HOURS = /^\d{1,3}$/;
const validDate = (s: string) => {
	const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s);
	if (!m) return false;
	const d = new Date(Date.UTC(+m[1], +m[2] - 1, +m[3]));
	return d.getUTCFullYear() === +m[1] && d.getUTCMonth() === +m[2] - 1 && d.getUTCDate() === +m[3];
};
const textOK = (s: string) => [...s].length >= 1 && [...s].length <= MEDICATION_TEXT_MAX_LEN;
const normHours = (h: string) => (HOURS.test(h) ? String(Number(h)) : h);

type Typed = { kind: string; times: string[]; hours: string };

function sameSchedule(t: Typed, original: string): boolean {
	const colon = original.indexOf(':');
	const kind = colon < 0 ? original : original.slice(0, colon);
	const rest = original.slice(colon + 1);
	if (kind !== t.kind) return false;
	if (kind === 'none') return colon < 0;
	if (kind === 'times') {
		const stored = rest.split(',');
		return stored.length === t.times.length && stored.every((v, i) => v === t.times[i]);
	}
	return kind === 'every' && rest === t.hours;
}

// The API's schedule body for what was typed, or null; only the chosen kind's fields count.
function scheduleBody(t: Typed): EntryBody | null {
	if (t.kind === 'none') return { kind: 'none' };
	if (t.kind === 'times') {
		const ok =
			t.times.length >= 1 &&
			t.times.length <= MAX_SCHEDULE_TIMES &&
			t.times.every((v, i) => CLOCK.test(v) && t.times.indexOf(v) === i);
		return ok ? { kind: 'times', times: t.times } : null;
	}
	const hours = HOURS.test(t.hours) ? Number(t.hours) : NaN;
	return t.kind === 'every' && hours >= MIN_EVERY_HOURS && hours <= MAX_EVERY_HOURS
		? { kind: 'every', hours }
		: null;
}

type CourseOutcome = { input: EntryBody } | { field: string };

// Checked in the API's order: name, dose, schedule, start_at, end_on, active, note. An edit sends only what differs
// from its hidden original (compared before any range check) and fails closed when one is missing.
function buildCourse(v: CourseFormValues, mode: CourseMode, activeOK: boolean): CourseOutcome {
	const edit = mode.kind === 'edit';
	const input: EntryBody = {};

	for (const f of ['name', 'dose'] as const) {
		const text = v[f].trim();
		if (edit) {
			const o = v[`${f}_original`];
			if (o === undefined) return { field: f };
			if (text === o.trim()) continue;
		}
		if (!textOK(text)) return { field: f };
		input[f] = text;
	}

	const typed: Typed = {
		kind: v.kind,
		times: v.times
			.map((t) => t.trim())
			.filter((t) => t !== '')
			.sort(),
		hours: normHours(v.hours.trim())
	};
	if (edit && v.schedule_original === undefined) return { field: 'schedule' };
	if (!edit || !sameSchedule(typed, v.schedule_original as string)) {
		const schedule = scheduleBody(typed);
		if (!schedule) return { field: 'schedule' };
		input.schedule = schedule;
	}

	const start = v.start_at.trim();
	if (edit && v.start_at_original === undefined) return { field: 'start_at' };
	if (start !== '' && (!edit || start !== v.start_at_original?.trim())) {
		const instant = zonedLocalToInstant(start, mode.zone);
		if (instant === null) return { field: 'start_at' };
		input.start_at = instant;
	}

	const end = v.end_on.trim();
	if (edit && v.end_on_original === undefined) return { field: 'end_on' };
	if (edit ? end !== v.end_on_original?.trim() : end !== '') {
		if (end === '') input.end_on = null;
		else if (!validDate(end)) return { field: 'end_on' };
		else input.end_on = end;
	}

	if (!activeOK) return { field: 'active' };
	if (edit) {
		const o = v.active_original;
		if (o !== 'true' && o !== 'false') return { field: 'active' };
		if (String(v.active) !== o) input.active = v.active;
	} else if (!v.active) {
		input.active = false;
	}

	const note = v.note.trim();
	if (edit && v.note_original === undefined) return { field: 'note' };
	if (!edit || note !== v.note_original?.trim()) {
		if ([...note].length > NOTE_MAX_LEN) return { field: 'note' };
		if (note !== '') input.note = note;
		else if (edit) input.note = null;
	}
	return { input };
}

export function readMedicationForm(
	form: FormData,
	mode: CourseMode,
	{ prefix = '' }: { prefix?: string } = {}
): { values: CourseFormValues; input: EntryBody | null; field: string | null } {
	const text = (k: string) => String(form.get(`${prefix}${k}`) ?? '');
	// An unticked box is absent; anything but on is not what the form posts.
	const box = form.get(`${prefix}active`);
	const values: CourseFormValues = {
		name: text('name'),
		dose: text('dose'),
		kind: text('kind'),
		times: form.getAll(`${prefix}times`).map(String),
		hours: text('hours'),
		start_at: text('start_at'),
		end_on: text('end_on'),
		active: box === 'on',
		note: text('note')
	};
	if (mode.kind === 'edit') {
		for (const k of TEXT_ORIGINALS) {
			const o = form.get(`${prefix}${k}_original`);
			if (o !== null) values[`${k}_original`] = String(o);
		}
	}
	const r = buildCourse(values, mode, box === null || box === 'on');
	return 'field' in r
		? { values, input: null, field: r.field }
		: { values, input: r.input, field: null };
}

export const courseFieldFailure = <T extends object>(
	values: CourseFormValues,
	field: string,
	extra: T
) => refusedField(COURSE_FIELD_MESSAGES, values, field, extra);

export const courseApiFailure = <T extends object>(
	values: CourseFormValues,
	r: ApiFailure,
	extra: T
) => apiRefusal(COURSE_FIELD_MESSAGES, COURSE_LIMIT_MESSAGE, values, r, extra);
