import { describe, expect, it } from 'vitest';
import { growthCreateValues, GROWTH_FIELD_MESSAGES, readGrowthForm } from './growth-form';

const CREATE = { kind: 'create' } as const;
const EDIT = { kind: 'edit' } as const;
const form = (fields: Record<string, string>) => {
	const data = new FormData();
	for (const [k, v] of Object.entries(fields)) data.append(k, v);
	return data;
};

const blank = {
	measured_on: '2026-11-20',
	weight_unit: 'kg',
	weight_kg: '',
	weight_lb: '',
	weight_oz: '',
	length_unit: 'cm',
	length: '',
	head_unit: 'cm',
	head: '',
	note: ''
};

// An edit form as loaded: every shown text beside its _original.
const loaded = (shown: Record<string, string>) => {
	const all = { ...blank, ...shown };
	const originals = Object.fromEntries(
		Object.entries(all)
			.filter(([k]) => k !== 'note')
			.map(([k, v]) => [`${k}_original`, v])
	);
	return { ...all, ...originals };
};

const create = (fields: Record<string, string>) =>
	readGrowthForm(form({ ...blank, ...fields }), CREATE);

describe('growthCreateValues', () => {
	it('prefills today and the units from the preference; head follows the length unit', () => {
		expect(growthCreateValues('2026-11-20', { weight: 'lb', length: 'in' })).toEqual({
			...blank,
			weight_unit: 'lb',
			length_unit: 'in',
			head_unit: 'in'
		});
	});
});

describe('readGrowthForm create', () => {
	it('reads a full row: kilograms to grams, centimetres to millimetres', () => {
		const r = create({ weight_kg: '3.2', length: '49.5', head: '34', note: ' first visit ' });
		expect(r).toMatchObject({ field: null, message: null });
		expect(r.input).toEqual({
			measured_on: '2026-11-20',
			weight: { unit: 'kg', amount: 3200 },
			length: { unit: 'cm', amount: 495 },
			head: { unit: 'cm', amount: 340 },
			note: 'first visit'
		});
	});

	it.each([
		['3.2', 3200],
		['3,2', 3200],
		['3.215', 3215],
		['0,25', 250],
		[' 4 ', 4000]
	])('kg %j is %i g', (typed, grams) => {
		expect(create({ weight_kg: typed }).input).toMatchObject({
			weight: { unit: 'kg', amount: grams }
		});
	});

	it.each([
		['8', '3.5', 13150],
		['8', '3,5', 13150],
		['7', '1', 11300],
		['7', '', 11200],
		['', '8', 800],
		['0', '15.99', 1599],
		['8', '0.05', 12805]
	])('lb %j and oz %j are %i hundredths of an ounce', (lb, oz, amount) => {
		expect(create({ weight_unit: 'lb', weight_lb: lb, weight_oz: oz }).input).toMatchObject({
			weight: { unit: 'lb', amount }
		});
	});

	it.each([
		['49.5', 'cm', 495],
		['49,5', 'cm', 495],
		['50', 'cm', 500],
		['19.5', 'in', 1950],
		['19.53', 'in', 1953],
		['20', 'in', 2000]
	])('length %j %s is %i', (typed, unit, amount) => {
		expect(create({ length: typed, length_unit: unit }).input).toMatchObject({
			length: { unit, amount }
		});
		expect(create({ head: typed, head_unit: unit }).input).toMatchObject({
			head: { unit, amount }
		});
	});

	it('leaves a blank measure out', () => {
		const r = create({ length: '49.5' });
		expect(r.input).toEqual({
			measured_on: '2026-11-20',
			length: { unit: 'cm', amount: 495 },
			note: null
		});
	});

	it('names measurements when nothing is filled', () => {
		expect(create({ note: 'just a note' })).toMatchObject({
			input: null,
			field: 'measurements',
			message: null
		});
	});

	it.each(['', '2026-13-01', '2026-02-30', '20-11-2026', 'today'])(
		'names measured_on for %j',
		(date) => {
			expect(create({ measured_on: date, weight_kg: '3.2' })).toMatchObject({
				input: null,
				field: 'measured_on'
			});
		}
	);

	it.each(['16', '16.00', '15.999', '20', '-1', '1e1'])('refuses %j ounces as weight', (oz) => {
		expect(create({ weight_unit: 'lb', weight_lb: '8', weight_oz: oz })).toMatchObject({
			input: null,
			field: 'weight'
		});
	});

	it('refuses 0 lb 0 oz', () => {
		expect(create({ weight_unit: 'lb', weight_lb: '0', weight_oz: '0' })).toMatchObject({
			field: 'weight'
		});
	});

	it.each(['1e3', '0x10', '3.2.1', '--1', '-1', '3.2345', '+3', 'Infinity', '٣', '3 .2'])(
		'refuses hostile text %j in every measure',
		(typed) => {
			expect(create({ weight_kg: typed }).field).toBe('weight');
			expect(create({ weight_unit: 'lb', weight_lb: typed, weight_oz: '1' }).field).toBe('weight');
			expect(create({ length: typed }).field).toBe('length');
			expect(create({ head: typed, head_unit: 'in' }).field).toBe('head');
		}
	);

	it('refuses a second decimal in centimetres and a third in inches', () => {
		expect(create({ length: '49.55' }).field).toBe('length');
		expect(create({ length: '19.555', length_unit: 'in' }).field).toBe('length');
	});

	it('refuses an unknown unit for a filled measure', () => {
		expect(create({ weight_kg: '3.2', weight_unit: 'st' }).field).toBe('weight');
		expect(create({ length: '50', length_unit: 'mm' }).field).toBe('length');
		expect(create({ head: '34', head_unit: '' }).field).toBe('head');
	});

	it('refuses a metric value outside the range, with a range message', () => {
		for (const [fields, field] of [
			[{ weight_kg: '0.199' }, 'weight'],
			[{ weight_kg: '250.001' }, 'weight'],
			[{ length: '14.9' }, 'length'],
			[{ length: '250.1' }, 'length'],
			[{ head: '14.9' }, 'head'],
			[{ head: '80.1' }, 'head']
		] as const) {
			const r = create(fields);
			expect(r).toMatchObject({ input: null, field });
			expect(r.message).not.toBeNull();
		}
		expect(create({ weight_kg: '0.2', length: '250', head: '80' }).field).toBeNull();
	});

	it('names the first bad field in the API order', () => {
		expect(create({ measured_on: '', weight_kg: 'x', length: 'x' }).field).toBe('measured_on');
		expect(create({ weight_kg: 'x', length: 'x', head: 'x' }).field).toBe('weight');
		expect(create({ length: 'x', head: 'x' }).field).toBe('length');
	});

	it('keeps what was typed in values', () => {
		const r = create({ weight_kg: '3,2x', note: 'n' });
		expect(r.values).toMatchObject({ weight_kg: '3,2x', note: 'n', measured_on: '2026-11-20' });
	});

	it('puts no typed value or date in any message', () => {
		const typed = '987654.321';
		const results = [
			create({ weight_kg: typed }),
			create({ weight_unit: 'lb', weight_lb: '1', weight_oz: '15.999' }),
			create({ length: typed }),
			create({ head: '987.6' }),
			create({ measured_on: '2031-01-02', weight_kg: '3' })
		];
		const messages = [
			...results.map((r) => r.message ?? ''),
			...Object.values(GROWTH_FIELD_MESSAGES)
		];
		for (const m of messages) {
			for (const bad of [typed, '987', '15.999', '2031']) expect(m).not.toContain(bad);
		}
	});
});

describe('readGrowthForm edit', () => {
	const stored = loaded({
		measured_on: '2026-11-01',
		weight_kg: '3.2',
		length: '49.5'
	});
	const edit = (fields: Record<string, string>) =>
		readGrowthForm(form({ ...stored, ...fields }), EDIT);

	it('sends no measure and no date when nothing changed; the note always goes', () => {
		expect(edit({ note: 'seen' })).toMatchObject({
			field: null,
			input: { note: 'seen' }
		});
		expect(Object.keys(edit({}).input ?? {})).toEqual(['note']);
	});

	it('sends a measure only once its typed text changed', () => {
		expect(edit({ weight_kg: '3.25' }).input).toEqual({
			weight: { unit: 'kg', amount: 3250 },
			note: null
		});
		expect(edit({ weight_kg: ' 3.2 ' }).input).toEqual({ note: null });
	});

	it('a changed unit counts as a change', () => {
		expect(edit({ weight_unit: 'lb', weight_lb: '7', weight_oz: '1' }).input).toMatchObject({
			weight: { unit: 'lb', amount: 11300 }
		});
	});

	it('sends a changed date', () => {
		expect(edit({ measured_on: '2026-11-02' }).input).toEqual({
			measured_on: '2026-11-02',
			note: null
		});
	});

	it('clears a measure that was shown and is now blank with null', () => {
		expect(edit({ length: '' }).input).toEqual({ length: null, note: null });
	});

	it('sends nothing for a measure that was blank and still is', () => {
		expect(edit({ head: '' }).input).toEqual({ note: null });
	});

	it('adds a measure that was blank', () => {
		expect(edit({ head: '34.5' }).input).toEqual({
			head: { unit: 'cm', amount: 345 },
			note: null
		});
	});

	it('names measurements when the edit would leave no measure', () => {
		expect(edit({ weight_kg: '', length: '' })).toMatchObject({
			input: null,
			field: 'measurements'
		});
	});

	it('refuses a changed measure it cannot read', () => {
		expect(edit({ weight_kg: '3.2.1' }).field).toBe('weight');
	});

	it('names measured_on when an edit blanks the date', () => {
		expect(edit({ measured_on: '' }).field).toBe('measured_on');
	});

	it.each([
		['measured_on_original', 'measured_on'],
		['weight_unit_original', 'weight'],
		['weight_kg_original', 'weight'],
		['weight_lb_original', 'weight'],
		['weight_oz_original', 'weight'],
		['length_unit_original', 'length'],
		['length_original', 'length'],
		['head_unit_original', 'head'],
		['head_original', 'head']
	])('fails closed without %s', (missing, field) => {
		const fields: Record<string, string> = { ...stored };
		delete fields[missing];
		expect(readGrowthForm(form(fields), EDIT)).toMatchObject({ input: null, field });
	});

	it('fails closed on an original unit it does not know', () => {
		expect(edit({ weight_unit_original: 'st' }).field).toBe('weight');
	});

	it('keeps the originals in values so the form can be shown again', () => {
		expect(edit({ weight_kg: 'x' }).values).toMatchObject({
			weight_kg: 'x',
			weight_kg_original: '3.2',
			measured_on_original: '2026-11-01'
		});
	});
});
