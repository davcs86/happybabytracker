import { describe, expect, it } from 'vitest';
import { MEDICATION_TEXT_MAX_LEN, NOTE_MAX_LEN } from '../constants';
import type { MedicationDose } from '../types';
import {
	DOSE_FIELD_MESSAGES,
	DOSE_LIMIT_MESSAGE,
	doseApiFailure,
	doseEditValues,
	doseFieldFailure,
	readDoseForm
} from './medication-form';

type Fields = Record<string, string>;
const form = (fields: Fields) => {
	const data = new FormData();
	for (const [k, v] of Object.entries(fields)) data.append(k, v);
	return data;
};

const LOG = { kind: 'log' } as const;
const log = (fields: Fields) => readDoseForm(form(fields), LOG);

describe('readDoseForm: the one-tap dose', () => {
	it('sends the course alone when no amount is typed', () => {
		expect(log({ medication: 'm-1' })).toMatchObject({ input: { medication: 'm-1' }, field: null });
		const r = log({ medication: 'm-1', amount: '' });
		expect(r.input).toEqual({ medication: 'm-1' });
		expect(r.values).toEqual({ medication: 'm-1', amount: '' });
	});

	it('sends a trimmed amount and keeps the typed text in the values', () => {
		const r = log({ medication: 'm-1', amount: '  2.5 ml  ' });
		expect(r.input).toEqual({ medication: 'm-1', amount: '2.5 ml' });
		expect(r.values.amount).toBe('  2.5 ml  ');
	});

	it('reads whitespace alone as no amount', () => {
		expect(log({ medication: 'm-1', amount: ' \t ' }).input).toEqual({ medication: 'm-1' });
	});

	it('refuses a missing, empty or malformed course before the amount', () => {
		for (const medication of ['', ' ', 'a b', '../x', '<script>', 'm\n1', 'é', 'x'.repeat(65)]) {
			expect(log({ medication, amount: 'x'.repeat(500) })).toMatchObject({
				input: null,
				field: 'medication'
			});
		}
		expect(readDoseForm(form({ amount: '1 ml' }), LOG)).toMatchObject({
			input: null,
			field: 'medication'
		});
	});

	it('limits the amount to the API length in characters, not bytes', () => {
		const at = MEDICATION_TEXT_MAX_LEN;
		expect(log({ medication: 'm-1', amount: 'a'.repeat(at) }).field).toBeNull();
		expect(log({ medication: 'm-1', amount: 'é'.repeat(at) }).field).toBeNull();
		expect(log({ medication: 'm-1', amount: '😀'.repeat(at) }).field).toBeNull();
		expect(log({ medication: 'm-1', amount: ` ${'a'.repeat(at)} ` }).field).toBeNull();
		for (const amount of ['a'.repeat(at + 1), 'é'.repeat(at + 1), '😀'.repeat(at + 1)]) {
			expect(log({ medication: 'm-1', amount })).toMatchObject({ input: null, field: 'amount' });
		}
	});

	it('reads nothing but the course and the amount', () => {
		const r = log({
			medication: 'm-1',
			amount: '1 ml',
			given_at: '2020-01-01T00:00',
			note: 'x',
			amount_original: '',
			id: 'd-1'
		});
		expect(r.input).toEqual({ medication: 'm-1', amount: '1 ml' });
	});
});

const STORED: MedicationDose = {
	id: 'd-1',
	medication: 'm-1',
	givenAt: '2026-11-20T08:30:45Z',
	amount: '2.5 ml',
	note: 'with food'
};
// An edit form as loaded, with the hidden originals doseEditValues writes.
const loaded = (zone = 'UTC'): Fields => ({ ...doseEditValues(STORED, zone) }) as Fields;
const edit = (change: Fields, drop: string[] = [], zone = 'UTC') => {
	const fields = { ...loaded(zone), ...change };
	for (const k of drop) delete fields[k];
	return readDoseForm(form(fields), { kind: 'edit', zone });
};

describe('doseEditValues', () => {
	it('shows the time on the kid clock to the minute and keeps what was loaded', () => {
		expect(doseEditValues(STORED, 'UTC')).toEqual({
			given_at: '2026-11-20T08:30',
			amount: '2.5 ml',
			note: 'with food',
			given_at_original: '2026-11-20T08:30',
			amount_original: '2.5 ml',
			note_original: 'with food'
		});
		expect(doseEditValues(STORED, 'Europe/Madrid').given_at).toBe('2026-11-20T09:30');
	});

	it('shows no amount and no note as empty text', () => {
		expect(doseEditValues({ ...STORED, amount: null, note: null }, 'UTC')).toMatchObject({
			amount: '',
			note: '',
			amount_original: '',
			note_original: ''
		});
	});
});

describe('readDoseForm: editing a dose', () => {
	it('sends nothing for a form left as loaded, even with seconds in the stored time', () => {
		expect(edit({})).toMatchObject({ input: {}, field: null });
		expect(edit({}, [], 'Europe/Madrid').input).toEqual({});
	});

	it('sends only the note when only the note changed', () => {
		expect(edit({ note: 'after lunch' }).input).toEqual({ note: 'after lunch' });
		expect(edit({ note: '' }).input).toEqual({ note: null });
		expect(edit({ note: '  with food ' }).input).toEqual({});
	});

	it('sends only the amount when only the amount changed, and null when cleared', () => {
		expect(edit({ amount: '5 ml' }).input).toEqual({ amount: '5 ml' });
		expect(edit({ amount: '  5 ml ' }).input).toEqual({ amount: '5 ml' });
		expect(edit({ amount: '' }).input).toEqual({ amount: null });
		expect(edit({ amount: '   ' }).input).toEqual({ amount: null });
		expect(edit({ amount: ' 2.5 ml ' }).input).toEqual({});
	});

	it('sends the new time as an instant read on the kid clock', () => {
		expect(edit({ given_at: '2026-11-20T09:15' }).input).toEqual({
			given_at: '2026-11-20T09:15:00Z'
		});
		expect(edit({ given_at: '2026-11-20T10:15' }, [], 'Europe/Madrid').input).toEqual({
			given_at: '2026-11-20T09:15:00Z'
		});
		expect(edit({ given_at: '2026-11-20T08:30', note: 'x' }).input).toEqual({ note: 'x' });
	});

	it('keeps the stored time when the time is left blank', () => {
		expect(edit({ given_at: '', note: 'x' }).input).toEqual({ note: 'x' });
	});

	it('sends a change of every field together, never the course', () => {
		const r = edit({
			given_at: '2026-11-20T07:00',
			amount: '1 ml',
			note: '',
			medication: 'm-other'
		});
		expect(r.input).toEqual({ given_at: '2026-11-20T07:00:00Z', amount: '1 ml', note: null });
	});

	it('refuses a time that is not a date and time, on the time', () => {
		for (const given_at of [
			'garbage',
			'2026-02-30T10:00',
			'2026-11-20T25:00',
			'2026-11-20T10:60',
			'2026-11-20 10:00',
			'2026-11-20T10:00:30',
			'2026-11-20',
			'10:00',
			'2026-13-01T10:00',
			'+2026-11-20T10:00'
		]) {
			expect(edit({ given_at })).toMatchObject({ input: null, field: 'given_at' });
		}
	});

	it('checks the time, then the amount, then the note', () => {
		const long = 'x'.repeat(NOTE_MAX_LEN + 1);
		expect(edit({ given_at: 'bad', amount: long, note: long }).field).toBe('given_at');
		expect(edit({ amount: long, note: long }).field).toBe('amount');
		expect(edit({ note: long }).field).toBe('note');
	});

	it('limits the amount and the note to their own lengths', () => {
		expect(edit({ amount: 'a'.repeat(MEDICATION_TEXT_MAX_LEN) }).field).toBeNull();
		expect(edit({ amount: 'a'.repeat(MEDICATION_TEXT_MAX_LEN + 1) }).field).toBe('amount');
		expect(edit({ amount: '😀'.repeat(MEDICATION_TEXT_MAX_LEN + 1) }).field).toBe('amount');
		expect(edit({ note: 'n'.repeat(NOTE_MAX_LEN) }).field).toBeNull();
		expect(edit({ note: 'n'.repeat(NOTE_MAX_LEN + 1) }).field).toBe('note');
		expect(edit({ note: '😀'.repeat(NOTE_MAX_LEN + 1) }).field).toBe('note');
	});

	it('fails closed on a missing original, in the order of the fields', () => {
		expect(edit({}, ['given_at_original'])).toMatchObject({ input: null, field: 'given_at' });
		expect(edit({}, ['amount_original'])).toMatchObject({ input: null, field: 'amount' });
		expect(edit({}, ['note_original'])).toMatchObject({ input: null, field: 'note' });
		expect(edit({}, ['given_at_original', 'note_original']).field).toBe('given_at');
		// Not a guess from the typed value, even when it would equal the stored one.
		expect(edit({ note: 'with food' }, ['note_original']).field).toBe('note');
	});

	it('keeps the typed texts and the loaded originals in the values', () => {
		const r = edit({ amount: 'x'.repeat(MEDICATION_TEXT_MAX_LEN + 1), note: 'typed' });
		expect(r.field).toBe('amount');
		expect(r.values).toMatchObject({
			amount: 'x'.repeat(MEDICATION_TEXT_MAX_LEN + 1),
			note: 'typed',
			given_at: '2026-11-20T08:30',
			amount_original: '2.5 ml',
			note_original: 'with food'
		});
	});
});

describe('dose failures', () => {
	const values = { amount: '2.5 ml' };
	it('lands a refused field on its input with a message that holds no value', () => {
		expect(doseFieldFailure(values, 'amount', { form: 'log' })).toMatchObject({
			status: 422,
			data: { form: 'log', values, field: 'amount', message: DOSE_FIELD_MESSAGES.amount }
		});
		for (const message of Object.values(DOSE_FIELD_MESSAGES)) {
			expect(message).not.toMatch(/2\.5|m-1/);
		}
		expect(DOSE_FIELD_MESSAGES.amount).toBe(
			`Enter an amount of up to ${MEDICATION_TEXT_MAX_LEN} characters`
		);
	});

	it('maps the API refusals, the dose cap and an unreachable API', () => {
		expect(
			doseApiFailure(values, { ok: false, status: 422, reason: 'invalid', field: 'given_at' }, {})
		).toMatchObject({
			status: 422,
			data: { field: 'given_at', message: DOSE_FIELD_MESSAGES.given_at }
		});
		expect(
			doseApiFailure(values, { ok: false, status: 422, reason: 'invalid', field: 'medication' }, {})
		).toMatchObject({
			status: 422,
			data: { values, field: 'medication', message: DOSE_FIELD_MESSAGES.medication }
		});
		expect(
			doseApiFailure(values, { ok: false, status: 409, reason: 'conflict', code: 'limit' }, {})
		).toMatchObject({ status: 409, data: { values, field: null, message: DOSE_LIMIT_MESSAGE } });
		expect(DOSE_LIMIT_MESSAGE).toBe('This child has the maximum number of logged doses');
		expect(
			doseApiFailure(values, { ok: false, status: 503, reason: 'unavailable' }, {})
		).toMatchObject({ status: 503, data: { values, field: null } });
		expect(doseApiFailure(values, { ok: false, status: 500, reason: 'error' }, {})).toMatchObject({
			status: 500,
			data: { values, field: null }
		});
	});

	it('answers an unknown field of the API like any other failure', () => {
		expect(
			doseApiFailure(values, { ok: false, status: 422, reason: 'invalid', field: 'zzz' }, {})
		).toMatchObject({ status: 500, data: { values, field: null } });
	});
});
