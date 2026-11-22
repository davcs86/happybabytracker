import {
	NOTE_MAX_LEN,
	VACCINE_DISMISS_REASONS,
	VACCINE_LOT_MAX_LEN,
	VACCINE_PRODUCT_MAX_LEN
} from '../constants';
import type { Administration, AdministrationInput, VaccineDismissReason } from '../types';
import { ANTIGEN_PATTERN } from './constants';

const TEXTS = ['lot', 'product', 'note'] as const;
type Text = (typeof TEXTS)[number];
const MAX_LEN: Record<Text, number> = {
	lot: VACCINE_LOT_MAX_LEN,
	product: VACCINE_PRODUCT_MAX_LEN,
	note: NOTE_MAX_LEN
};
const KEPT = ['given_on', 'antigens', ...TEXTS] as const;
type Original = `${(typeof KEPT)[number]}_original`;

// antigens_original is the stored set joined by commas.
export type VaccineFormValues = { given_on: string; antigens: string[] } & Record<Text, string> &
	Partial<Record<Original, string>>;
export type VaccineMode = { kind: 'create' } | { kind: 'edit' };

// Keyed by the API's field names; never a typed value in a message.
export const VACCINE_FIELD_MESSAGES: Record<string, string> = {
	given_on: 'Enter a date that is not in the future and not before the birth date',
	antigens: 'Choose at least one vaccine from the list',
	lot: `Enter a lot number of up to ${VACCINE_LOT_MAX_LEN} characters`,
	product: `Enter a product name of up to ${VACCINE_PRODUCT_MAX_LEN} characters; Other needs one`,
	note: 'The note is too long'
};

export const DISMISS_FIELD_MESSAGES: Record<string, string> = {
	before: 'Enter a date',
	reason: 'Choose a reason from the list'
};

export const vaccineCreateValues = (today: string): VaccineFormValues => ({
	given_on: today,
	antigens: [],
	lot: '',
	product: '',
	note: ''
});

export function vaccineEditValues(a: Administration): VaccineFormValues {
	const shown = {
		given_on: a.givenOn,
		antigens: [...a.antigens],
		lot: a.lot ?? '',
		product: a.product ?? '',
		note: a.note ?? ''
	};
	return {
		...shown,
		given_on_original: shown.given_on,
		antigens_original: shown.antigens.join(','),
		lot_original: shown.lot,
		product_original: shown.product,
		note_original: shown.note
	};
}

function validDate(s: string): boolean {
	const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s);
	if (!m) return false;
	const d = new Date(Date.UTC(+m[1], +m[2] - 1, +m[3]));
	return d.getUTCFullYear() === +m[1] && d.getUTCMonth() === +m[2] - 1 && d.getUTCDate() === +m[3];
}

const sameSet = (a: string[], b: string[]) =>
	a.length === b.length && [...a].sort().every((v, i) => v === [...b].sort()[i]);

type Outcome = { input: AdministrationInput } | { field: string };

// Fields are checked in the API's order: given_on, antigens, lot, product, note. An edit sends only what
// differs from its hidden _original, and fails closed when one is missing.
function build(v: VaccineFormValues, mode: VaccineMode): Outcome {
	const edit = mode.kind === 'edit';
	const input: AdministrationInput = {};

	const date = v.given_on.trim();
	if (edit && v.given_on_original === undefined) return { field: 'given_on' };
	if (!edit || date !== v.given_on_original?.trim()) {
		if (!validDate(date)) return { field: 'given_on' };
		input.given_on = date;
	}

	if (edit && v.antigens_original === undefined) return { field: 'antigens' };
	const stored = (v.antigens_original ?? '').split(',').filter((a) => a !== '');
	if (!edit || !sameSet(v.antigens, stored)) {
		const ok = v.antigens.every(
			(a, i) => ANTIGEN_PATTERN.test(a) && !v.antigens.includes(a, i + 1)
		);
		if (v.antigens.length === 0 || !ok) return { field: 'antigens' };
		input.antigens = [...v.antigens].sort();
	}

	for (const t of TEXTS) {
		const original = v[`${t}_original`];
		if (edit && original === undefined) return { field: t };
		const text = v[t].trim();
		if (edit && text === original?.trim()) continue;
		if ([...text].length > MAX_LEN[t]) return { field: t };
		if (text !== '') input[t] = text;
		else if (edit) input[t] = null;
	}
	return { input };
}

// message is null when the field's own message (VACCINE_FIELD_MESSAGES) applies.
export function readVaccineForm(
	form: FormData,
	mode: VaccineMode
): {
	values: VaccineFormValues;
	input: AdministrationInput | null;
	field: string | null;
	message: string | null;
} {
	const text = (k: string) => String(form.get(k) ?? '');
	const values: VaccineFormValues = {
		given_on: text('given_on'),
		antigens: form.getAll('antigens').map(String),
		lot: text('lot'),
		product: text('product'),
		note: text('note')
	};
	if (mode.kind === 'edit') {
		for (const k of KEPT) {
			const o = form.get(`${k}_original`);
			if (o !== null) values[`${k}_original`] = String(o);
		}
	}
	const r = build(values, mode);
	return 'field' in r
		? { values, input: null, field: r.field, message: null }
		: { values, input: r.input, field: null, message: null };
}

const isReason = (v: string): v is VaccineDismissReason =>
	(VACCINE_DISMISS_REASONS as readonly string[]).includes(v);

export function readDismissForm(form: FormData): {
	input: { reason: VaccineDismissReason } | null;
	field: 'reason' | null;
} {
	const reason = String(form.get('reason') ?? '');
	return isReason(reason) ? { input: { reason }, field: null } : { input: null, field: 'reason' };
}

// before is checked first, as the API does.
export function readDismissBeforeForm(form: FormData): {
	values: { before: string; reason: string };
	input: { before: string; reason: VaccineDismissReason } | null;
	field: 'before' | 'reason' | null;
} {
	const values = {
		before: String(form.get('before') ?? '').trim(),
		reason: String(form.get('reason') ?? '')
	};
	if (!validDate(values.before)) return { values, input: null, field: 'before' };
	if (!isReason(values.reason)) return { values, input: null, field: 'reason' };
	return { values, input: { before: values.before, reason: values.reason }, field: null };
}
