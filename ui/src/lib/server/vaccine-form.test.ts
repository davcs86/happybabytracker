import { describe, expect, it } from 'vitest';
import type { Administration } from '../types';
import {
	DISMISS_FIELD_MESSAGES,
	DISMISS_REFUSED_MESSAGE,
	dismissApiFailure,
	prefillAntigens,
	vaccineApiFailure,
	VACCINE_LIMIT_MESSAGE,
	readDismissBeforeForm,
	readDismissForm,
	readVaccineForm,
	VACCINE_FIELD_MESSAGES,
	vaccineCreateValues,
	vaccineEditValues
} from './vaccine-form';

const CREATE = { kind: 'create' } as const;
const EDIT = { kind: 'edit' } as const;
type Fields = Record<string, string | string[]>;
const form = (fields: Fields) => {
	const data = new FormData();
	for (const [k, v] of Object.entries(fields)) {
		for (const one of Array.isArray(v) ? v : [v]) data.append(k, one);
	}
	return data;
};

const base = { given_on: '2026-11-20', antigens: ['measles', 'mumps', 'rubella'] };
const create = (fields: Fields) => readVaccineForm(form({ ...base, ...fields }), CREATE);

const STORED: Administration = {
	id: 'a1',
	visitId: null,
	givenOn: '2026-11-01',
	antigens: ['diphtheria', 'pertussis', 'tetanus'],
	lot: 'LOT-7',
	product: '6-in-1',
	note: 'left thigh'
};
// An edit form as loaded, with the hidden originals vaccineEditValues writes.
const loaded = () => {
	const v = vaccineEditValues(STORED);
	return { ...v, antigens: [...v.antigens] } as Fields;
};
const edit = (change: Fields, drop: string[] = []) => {
	const fields = { ...loaded(), ...change };
	for (const k of drop) delete fields[k];
	return readVaccineForm(form(fields), EDIT);
};

describe('vaccineCreateValues and vaccineEditValues', () => {
	it('prefills today on create, nothing ticked', () => {
		expect(vaccineCreateValues('2026-11-20')).toEqual({
			given_on: '2026-11-20',
			antigens: [],
			lot: '',
			product: '',
			note: ''
		});
	});

	it('shows the stored administration beside its originals', () => {
		expect(vaccineEditValues(STORED)).toEqual({
			given_on: '2026-11-01',
			antigens: ['diphtheria', 'pertussis', 'tetanus'],
			lot: 'LOT-7',
			product: '6-in-1',
			note: 'left thigh',
			given_on_original: '2026-11-01',
			antigens_original: 'diphtheria,pertussis,tetanus',
			lot_original: 'LOT-7',
			product_original: '6-in-1',
			note_original: 'left thigh'
		});
	});
});

describe('readVaccineForm create', () => {
	it('sends the date, the antigens sorted, and leaves blank optional texts out', () => {
		const r = create({ antigens: ['rubella', 'measles', 'mumps'], lot: '  ', note: '' });
		expect(r).toMatchObject({ field: null, message: null });
		expect(r.input).toEqual({ given_on: '2026-11-20', antigens: ['measles', 'mumps', 'rubella'] });
	});

	it('trims and sends lot, product and note', () => {
		expect(create({ lot: ' A1 ', product: ' MMR ', note: ' ok ' }).input).toEqual({
			...base,
			lot: 'A1',
			product: 'MMR',
			note: 'ok'
		});
	});

	it.each([
		['blank', ''],
		['whitespace', '   '],
		['an impossible date', '2025-13-40'],
		['February 30th', '2026-02-30'],
		['a number', '1e3'],
		['an inner space', '2026-11- 20'],
		['a time', '2026-11-20T00:00'],
		['a short year', '26-11-20'],
		['non-ASCII digits', '٢٠٢٦-١١-٢٠']
	])('refuses %s as given_on', (_name, given_on) => {
		expect(create({ given_on })).toMatchObject({ input: null, field: 'given_on', message: null });
	});

	it('refuses no antigen, a duplicate one and one that cannot be an id', () => {
		for (const antigens of [[], ['hib', 'hib'], ['Hib'], ['hib', '../x'], ['']]) {
			expect(create({ antigens })).toMatchObject({ input: null, field: 'antigens' });
		}
	});

	it('refuses texts over their limits by field name, counting characters', () => {
		expect(create({ lot: 'x'.repeat(41) }).field).toBe('lot');
		expect(create({ lot: 'é'.repeat(40) }).field).toBeNull();
		expect(create({ product: 'x'.repeat(81) }).field).toBe('product');
		expect(create({ note: 'x'.repeat(501) }).field).toBe('note');
		expect(create({ note: 'x'.repeat(500) }).field).toBeNull();
		const huge = create({ lot: 'LOTSECRET'.repeat(1111) + 'L' });
		expect(huge).toMatchObject({ input: null, field: 'lot', message: null });
		expect(VACCINE_FIELD_MESSAGES.lot).not.toContain('LOTSECRET');
	});

	it('names the first invalid field in the API order', () => {
		expect(create({ given_on: 'x', antigens: [], lot: 'x'.repeat(41) }).field).toBe('given_on');
		expect(create({ antigens: [], lot: 'x'.repeat(41) }).field).toBe('antigens');
		expect(create({ lot: 'x'.repeat(41), note: 'x'.repeat(501) }).field).toBe('lot');
	});

	it('keeps what was typed for the form to show again', () => {
		const r = create({ given_on: 'nope', lot: 'L', antigens: ['hib'] });
		expect(r.values).toMatchObject({ given_on: 'nope', lot: 'L', antigens: ['hib'] });
	});
});

describe('readVaccineForm edit', () => {
	it('sends nothing for an untouched form', () => {
		expect(edit({})).toEqual({
			values: expect.any(Object),
			input: {},
			field: null,
			message: null
		});
	});

	it('sends only the changed fields', () => {
		expect(edit({ given_on: '2026-11-02', note: 'right thigh' }).input).toEqual({
			given_on: '2026-11-02',
			note: 'right thigh'
		});
	});

	it('clears a stored lot, product or note with null', () => {
		expect(edit({ lot: '', product: ' ', note: '' }).input).toEqual({
			lot: null,
			product: null,
			note: null
		});
	});

	it('compares the antigens as a set, whatever their order', () => {
		expect(edit({ antigens: ['tetanus', 'diphtheria', 'pertussis'] }).input).toEqual({});
		expect(edit({ antigens: ['tetanus', 'diphtheria'] }).input).toEqual({
			antigens: ['diphtheria', 'tetanus']
		});
	});

	it('an untouched stored text is not checked again; a changed one is', () => {
		const long = 'y'.repeat(41);
		const kept = edit({ lot: long, lot_original: long });
		expect(kept).toMatchObject({ field: null, input: {} });
		expect(kept.input).not.toHaveProperty('lot');
		expect(edit({ given_on: 'x' }).field).toBe('given_on');
		expect(edit({ lot: 'y'.repeat(41) }).field).toBe('lot');
		expect(edit({ antigens: [] }).field).toBe('antigens');
	});

	it.each([
		['given_on_original', 'given_on'],
		['antigens_original', 'antigens'],
		['lot_original', 'lot'],
		['product_original', 'product'],
		['note_original', 'note']
	])('fails closed without %s', (original, field) => {
		expect(edit({}, [original])).toMatchObject({ input: null, field });
	});
});

describe('messages', () => {
	it('are fixed texts that never echo a typed value', () => {
		const typed = ['LOT-SECRET-1', 'PRODUCT-SECRET', 'NOTE-SECRET', '2025-13-40'];
		const results = [
			create({ given_on: '2025-13-40' }),
			create({ lot: 'LOT-SECRET-1'.repeat(5) }),
			create({ product: 'PRODUCT-SECRET'.repeat(7) }),
			create({ note: 'NOTE-SECRET'.repeat(50) })
		];
		for (const r of results) {
			const shown = r.message ?? VACCINE_FIELD_MESSAGES[r.field as string];
			expect(shown).toEqual(expect.any(String));
			for (const t of typed) expect(shown).not.toContain(t);
		}
		for (const m of [
			...Object.values(VACCINE_FIELD_MESSAGES),
			...Object.values(DISMISS_FIELD_MESSAGES)
		]) {
			expect(m.length).toBeGreaterThan(0);
		}
	});
});

describe('readDismissForm and readDismissBeforeForm', () => {
	it('reads a reason from the list', () => {
		expect(readDismissForm(form({ reason: 'given_unknown' }))).toEqual({
			input: { reason: 'given_unknown' },
			field: null
		});
		for (const reason of ['', 'other', 'NOT_NEEDED']) {
			expect(readDismissForm(form({ reason }))).toEqual({ input: null, field: 'reason' });
		}
		expect(readDismissForm(new FormData())).toEqual({ input: null, field: 'reason' });
	});

	it('reads a strict date and a reason, the date first', () => {
		expect(readDismissBeforeForm(form({ before: '2026-01-31', reason: 'not_needed' }))).toEqual({
			values: { before: '2026-01-31', reason: 'not_needed' },
			input: { before: '2026-01-31', reason: 'not_needed' },
			field: null
		});
		for (const before of ['', '2026-02-30', '1e3', '2026-1-31']) {
			expect(readDismissBeforeForm(form({ before, reason: 'x' })).field).toBe('before');
		}
		expect(readDismissBeforeForm(form({ before: '2026-01-31', reason: 'x' })).field).toBe('reason');
	});
});

describe('vaccineApiFailure and dismissApiFailure', () => {
	const values = vaccineCreateValues('2026-11-20');
	it('lands a refused field on its input and the dose cap on the form', () => {
		expect(
			vaccineApiFailure(values, { ok: false, status: 422, reason: 'invalid', field: 'product' }, {})
		).toMatchObject({
			status: 422,
			data: { values, field: 'product', message: VACCINE_FIELD_MESSAGES.product }
		});
		expect(
			vaccineApiFailure(values, { ok: false, status: 409, reason: 'conflict', code: 'limit' }, {})
		).toMatchObject({ status: 409, data: { values, field: null, message: VACCINE_LIMIT_MESSAGE } });
		expect(VACCINE_LIMIT_MESSAGE).toBe(
			'This child has the maximum number of recorded vaccinations'
		);
		expect(
			vaccineApiFailure(values, { ok: false, status: 503, reason: 'unavailable' }, {})
		).toMatchObject({ status: 503, data: { values, field: null } });
	});

	it('answers a caregiver with a message and a bad reason on its field', () => {
		expect(
			dismissApiFailure({ ok: false, status: 403, reason: 'forbidden' }, { form: 'dismiss' })
		).toMatchObject({
			status: 403,
			data: { form: 'dismiss', field: null, message: DISMISS_REFUSED_MESSAGE }
		});
		expect(
			dismissApiFailure({ ok: false, status: 422, reason: 'invalid', field: 'before' }, {})
		).toMatchObject({ status: 422, data: { field: 'before', message: 'Enter a date' } });
	});
});

describe('prefillAntigens', () => {
	const catalogue = {
		antigens: [
			{ id: 'measles', name: 'Measles' },
			{ id: 'hib', name: 'Hib' }
		],
		products: []
	};
	it('keeps catalogue ids once, in order', () => {
		expect(prefillAntigens('hib,measles,hib', catalogue)).toEqual(['hib', 'measles']);
		expect(prefillAntigens('measles,BAD,nope,../x,', catalogue)).toEqual(['measles']);
		expect(prefillAntigens(null, catalogue)).toEqual([]);
	});
});
