import { fail } from '@sveltejs/kit';
import { MEDICATION_TEXT_MAX_LEN, NOTE_MAX_LEN } from '../constants';
import { instantToLocalInput, zonedLocalToInstant } from '../time-zone';
import type { EntryBody, MedicationDose } from '../types';
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

export function doseFieldFailure<T extends object>(
	values: DoseFormValues,
	field: string,
	extra: T
) {
	return fail(422, { ...extra, values, field, message: DOSE_FIELD_MESSAGES[field] });
}

// A refused field lands on its input, the dose cap on the form, anything else as failAction decides.
export function doseApiFailure<T extends object>(values: DoseFormValues, r: ApiFailure, extra: T) {
	if (r.reason === 'invalid' && r.field && Object.hasOwn(DOSE_FIELD_MESSAGES, r.field)) {
		return doseFieldFailure(values, r.field, extra);
	}
	if (r.reason === 'conflict' && r.code === 'limit') {
		return fail(409, {
			...extra,
			values,
			field: null as string | null,
			message: DOSE_LIMIT_MESSAGE
		});
	}
	return failAction(r, { ...extra, values, field: null as string | null });
}
