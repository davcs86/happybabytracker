import { describe, expect, it } from 'vitest';
import {
	advancedToggle,
	createValues,
	editValues,
	entryInstant,
	isAdvanced,
	isRunning,
	localDate,
	milkStatus,
	ranOver,
	volumeText
} from './entry-edit';
import type { Bottle, Diaper, Meal, Nursing, Pumping, Sleep } from './types';

const MADRID = 'Europe/Madrid';
const ML = { weight: 'kg', volume: 'ml' } as const;
const FLOZ = { weight: 'lb', volume: 'floz' } as const;
const HOUR_MS = 3_600_000;

const NO_SOURCES = {
	sources_breast_milk: '',
	sources_formula: '',
	sources_cow_milk: '',
	sources_other: ''
};
const blankWeight = (name: string, unit: string, texts = { kg: '', lb: '', oz: '' }) => ({
	[`${name}_unit`]: unit,
	[`${name}_kg`]: texts.kg,
	[`${name}_lb`]: texts.lb,
	[`${name}_oz`]: texts.oz
});
const BLANK_WEIGHTS_KG = {
	...blankWeight('weight_before', 'kg'),
	...blankWeight('weight_after', 'kg')
};

// The same texts again under their _original names, as the edit page loads them.
const originals = (texts: Record<string, string>) =>
	Object.fromEntries(
		Object.entries(texts).flatMap(([k, v]) => [
			[k, v],
			[`${k}_original`, v]
		])
	);

const bottle = (over: Partial<Bottle> = {}): Bottle => ({
	id: 'e1',
	at: '2026-11-20T09:30:00Z',
	contents: ['breast_milk', 'formula'],
	volume: { ml: 120, floz_hundredths: 406, typed_unit: 'ml' },
	sources: [],
	note: null,
	...over
});
const nursing = (over: Partial<Nursing> = {}): Nursing => ({
	id: 'e2',
	startedAt: '2026-11-20T10:00:00Z',
	endedAt: '2026-11-20T10:12:03Z',
	leftSeconds: 723,
	rightSeconds: 61,
	side: null,
	sideSince: null,
	note: 'ok',
	weightBefore: null,
	weightAfter: null,
	milk: null,
	...over
});
const pumping = (over: Partial<Pumping> = {}): Pumping => ({
	id: 'e3',
	startedAt: '2026-11-20T10:00:00Z',
	durationSeconds: 1203,
	volume: { ml: 80, floz_hundredths: 270, typed_unit: 'floz' },
	side: 'both',
	note: null,
	...over
});
const sleep = (over: Partial<Sleep> = {}): Sleep => ({
	id: 'e4',
	startedAt: '2026-11-20T10:00:00Z',
	endedAt: null,
	note: null,
	...over
});
const diaper: Diaper = { id: 'e5', at: '2026-11-20T10:00:00Z', kind: 'both', note: 'rash' };
const meal: Meal = {
	id: 'e6',
	at: '2026-11-20T10:00:00Z',
	what: 'rice',
	reaction: null,
	note: null
};

describe('entryInstant and isRunning', () => {
	it('takes the time of an entry from its own field', () => {
		expect(entryInstant('bottles', bottle())).toBe('2026-11-20T09:30:00Z');
		expect(entryInstant('diapers', diaper)).toBe(diaper.at);
		expect(entryInstant('meals', meal)).toBe(meal.at);
		expect(entryInstant('pumping', pumping())).toBe('2026-11-20T10:00:00Z');
		expect(entryInstant('sleep', sleep())).toBe('2026-11-20T10:00:00Z');
		expect(entryInstant('nursing', nursing())).toBe('2026-11-20T10:00:00Z');
	});

	it('runs only a timer without an end', () => {
		expect(isRunning('sleep', sleep())).toBe(true);
		expect(isRunning('sleep', sleep({ endedAt: '2026-11-20T11:00:00Z' }))).toBe(false);
		expect(isRunning('nursing', nursing({ endedAt: null, side: 'left' }))).toBe(true);
		expect(isRunning('nursing', nursing())).toBe(false);
		expect(isRunning('bottles', bottle())).toBe(false);
		expect(isRunning('pumping', pumping())).toBe(false);
	});
});

describe('localDate', () => {
	it('is the date on the kid clock, not on UTC', () => {
		expect(localDate('2026-11-22T23:30:00Z', MADRID)).toBe('2026-11-23');
		expect(localDate('2026-11-22T23:30:00Z', 'UTC')).toBe('2026-11-22');
		expect(localDate('2026-11-23T00:30:00Z', 'America/Los_Angeles')).toBe('2026-11-22');
	});
});

describe('ranOver', () => {
	const start = '2026-11-20T10:00:00Z';
	const at = (hours: number) => Date.parse(start) + hours * HOUR_MS;

	it('is true only past the session limit', () => {
		expect(ranOver(start, at(25))).toBe(true);
		expect(ranOver(start, at(24) + 1000)).toBe(true);
		expect(ranOver(start, at(24))).toBe(false);
		expect(ranOver(start, at(1))).toBe(false);
	});
});

describe('volumeText', () => {
	it('shows whole millilitres, or ounces without trailing zeros', () => {
		const v = (ml: number, floz_hundredths: number) => ({ ml, floz_hundredths });
		expect(volumeText(v(120, 406), 'ml')).toBe('120');
		expect(volumeText(v(126, 425), 'floz')).toBe('4.25');
		expect(volumeText(v(120, 406), 'floz')).toBe('4.06');
		expect(volumeText(v(133, 450), 'floz')).toBe('4.5');
		expect(volumeText(v(118, 400), 'floz')).toBe('4');
		expect(volumeText(v(1, 3), 'floz')).toBe('0.03');
	});
});

describe('createValues', () => {
	const now = '2026-03-29T00:45:00Z';

	it('prefills the time to now on the kid clock and the unit to the preference', () => {
		expect(createValues('bottles', now, MADRID, FLOZ)).toEqual({
			at: '2026-03-29T01:45',
			contents: [],
			volume: '',
			volume_unit: 'floz',
			...NO_SOURCES,
			note: ''
		});
		expect(createValues('diapers', now, 'UTC', ML)).toEqual({
			at: '2026-03-29T00:45',
			kind: '',
			note: ''
		});
		expect(createValues('meals', now, 'UTC', ML)).toEqual({
			at: '2026-03-29T00:45',
			what: '',
			reaction: '',
			note: ''
		});
		expect(createValues('pumping', now, 'UTC', ML)).toEqual({
			started_at: '2026-03-29T00:45',
			duration_minutes: '',
			volume: '',
			volume_unit: 'ml',
			side: '',
			note: ''
		});
		expect(createValues('sleep', now, 'UTC', ML)).toEqual({
			started_at: '2026-03-29T00:45',
			ended_at: '',
			note: ''
		});
		expect(createValues('nursing', now, 'UTC', ML)).toEqual({
			started_at: '2026-03-29T00:45',
			ended_at: '',
			left_minutes: '',
			right_minutes: '',
			side: '',
			...BLANK_WEIGHTS_KG,
			note: ''
		});
	});

	it('prefills the weight units to the weight preference', () => {
		expect(createValues('nursing', now, 'UTC', FLOZ)).toMatchObject({
			weight_before_unit: 'lb',
			weight_after_unit: 'lb'
		});
	});

	it('carries no stored originals', () => {
		for (const kind of ['bottles', 'nursing', 'pumping', 'sleep', 'diapers', 'meals'] as const) {
			expect(
				Object.keys(createValues(kind, now, 'UTC', ML)).filter((k) => k.endsWith('_original'))
			).toEqual([]);
		}
	});
});

describe('editValues', () => {
	it('shows a bottle in the preferred unit and keeps the stored values as originals', () => {
		const v = editValues('bottles', bottle(), MADRID, FLOZ);
		expect(v).toEqual({
			at: '2026-11-20T10:30',
			at_original: '2026-11-20T10:30',
			contents: ['breast_milk', 'formula'],
			volume: '4.06',
			volume_original: '4.06',
			volume_unit: 'floz',
			volume_unit_original: 'floz',
			...originals(NO_SOURCES),
			sources_original: '',
			note: ''
		});
	});

	it('shows the typed column when it matches the preference', () => {
		const typed = bottle({ volume: { ml: 126, floz_hundredths: 425, typed_unit: 'floz' } });
		expect(editValues('bottles', typed, 'UTC', FLOZ)).toMatchObject({
			volume: '4.25',
			volume_unit: 'floz'
		});
		expect(editValues('bottles', typed, 'UTC', ML)).toMatchObject({
			volume: '126',
			volume_unit: 'ml'
		});
	});

	it('floors a finished nursing session to minutes and keeps the raw seconds', () => {
		const v = editValues('nursing', nursing(), 'UTC', ML);
		expect(v).toEqual({
			started_at: '2026-11-20T10:00',
			started_at_original: '2026-11-20T10:00',
			ended_at: '2026-11-20T10:12',
			ended_at_original: '2026-11-20T10:12',
			left_minutes: '12',
			right_minutes: '1',
			left_seconds_original: '723',
			right_seconds_original: '61',
			...originals(BLANK_WEIGHTS_KG),
			note: 'ok'
		});
	});

	it('gives a running nursing session no minutes, no originals and no side', () => {
		const v = editValues(
			'nursing',
			nursing({ endedAt: null, side: 'left', sideSince: '2026-11-20T10:00:00Z', leftSeconds: 0 }),
			'UTC',
			ML
		);
		expect(v).toEqual({
			started_at: '2026-11-20T10:00',
			started_at_original: '2026-11-20T10:00',
			ended_at: '',
			ended_at_original: '',
			...originals(BLANK_WEIGHTS_KG),
			note: 'ok'
		});
	});

	it('floors a pumping duration and keeps the raw seconds', () => {
		expect(editValues('pumping', pumping(), 'UTC', ML)).toEqual({
			started_at: '2026-11-20T10:00',
			started_at_original: '2026-11-20T10:00',
			duration_minutes: '20',
			duration_seconds_original: '1203',
			volume: '80',
			volume_original: '80',
			volume_unit: 'ml',
			volume_unit_original: 'ml',
			side: 'both',
			note: ''
		});
	});

	it('shows a running sleep with a blank end and a finished one with its end', () => {
		expect(editValues('sleep', sleep(), 'UTC', ML)).toMatchObject({
			ended_at: '',
			ended_at_original: ''
		});
		expect(
			editValues('sleep', sleep({ endedAt: '2026-11-20T11:30:00Z' }), MADRID, ML)
		).toMatchObject({ started_at: '2026-11-20T11:00', ended_at: '2026-11-20T12:30' });
	});

	it('shows a diaper and a meal; a null reaction is blank', () => {
		expect(editValues('diapers', diaper, 'UTC', ML)).toEqual({
			at: '2026-11-20T10:00',
			at_original: '2026-11-20T10:00',
			kind: 'both',
			note: 'rash'
		});
		expect(editValues('meals', meal, 'UTC', ML)).toEqual({
			at: '2026-11-20T10:00',
			at_original: '2026-11-20T10:00',
			what: 'rice',
			reaction: '',
			note: ''
		});
	});

	it('reads the time on the kid clock across the spring change', () => {
		const v = editValues('diapers', { ...diaper, at: '2026-03-29T01:30:00Z' }, MADRID, ML);
		expect(v.at).toBe('2026-03-29T03:30');
	});

	const SOURCES = [
		{ source: 'breast_milk', volume: { ml: 60, floz_hundredths: 203, typed_unit: 'ml' } },
		{ source: 'formula', volume: { ml: 90, floz_hundredths: 304, typed_unit: 'ml' } }
	] as const;
	const advancedBottle = () =>
		bottle({
			volume: { ml: 150, floz_hundredths: 507, typed_unit: 'ml' },
			sources: [...SOURCES]
		});

	it('shows each source of an advanced bottle in the preferred unit, with its original', () => {
		expect(editValues('bottles', advancedBottle(), 'UTC', ML)).toMatchObject({
			volume: '150',
			volume_unit: 'ml',
			sources_breast_milk: '60',
			sources_breast_milk_original: '60',
			sources_formula: '90',
			sources_formula_original: '90',
			sources_cow_milk: '',
			sources_cow_milk_original: '',
			sources_other: '',
			sources_other_original: '',
			sources_original: 'breast_milk,formula'
		});
		expect(editValues('bottles', advancedBottle(), 'UTC', FLOZ)).toMatchObject({
			volume: '5.07',
			volume_unit: 'floz',
			sources_breast_milk: '2.03',
			sources_breast_milk_original: '2.03',
			sources_formula: '3.04',
			sources_original: 'breast_milk,formula'
		});
	});

	it('shows the stored total of an advanced bottle in the preferred unit, and none for a simple one', () => {
		expect(editValues('bottles', advancedBottle(), 'UTC', ML).sources_total).toBe('150 ml');
		expect(editValues('bottles', advancedBottle(), 'UTC', FLOZ).sources_total).toBe('5.07 fl oz');
		expect(editValues('bottles', bottle(), 'UTC', ML)).not.toHaveProperty('sources_total');
	});

	it('shows a simple bottle with no source text and an empty sources_original', () => {
		expect(editValues('bottles', bottle(), 'UTC', ML)).toMatchObject({
			sources_formula: '',
			sources_formula_original: '',
			sources_original: ''
		});
	});

	it('shows a weight in its typed unit and pre-fills the other from its own column', () => {
		const kg = { grams: 3400, ozHundredths: 11993, typedUnit: 'kg' } as const;
		const lb = { grams: 3402, ozHundredths: 12000, typedUnit: 'lb' } as const;
		const v = editValues('nursing', nursing({ weightBefore: kg, weightAfter: lb }), 'UTC', ML);
		expect(v).toMatchObject({
			weight_before_unit: 'kg',
			weight_before_unit_original: 'kg',
			weight_before_kg: '3.4',
			weight_before_kg_original: '3.4',
			weight_before_lb: '7',
			weight_before_lb_original: '7',
			weight_before_oz: '7.93',
			weight_before_oz_original: '7.93',
			weight_after_unit: 'lb',
			weight_after_kg: '3.402',
			weight_after_lb: '7',
			weight_after_oz: '8'
		});
	});

	it('shows a missing weight blank, in the preferred weight unit', () => {
		const v = editValues('nursing', nursing(), 'UTC', FLOZ);
		expect(v).toMatchObject({
			weight_before_unit: 'lb',
			weight_before_unit_original: 'lb',
			weight_before_kg: '',
			weight_before_kg_original: '',
			weight_after_unit: 'lb',
			weight_after_oz_original: ''
		});
	});

	it('shows the weights of a running session too', () => {
		const kg = { grams: 3400, ozHundredths: 11993, typedUnit: 'kg' } as const;
		const running = nursing({ endedAt: null, side: 'left', weightBefore: kg });
		expect(editValues('nursing', running, 'UTC', ML)).toMatchObject({
			weight_before_kg: '3.4',
			weight_after_kg: ''
		});
	});
});

describe('isAdvanced', () => {
	const withSources = bottle({
		sources: [{ source: 'formula', volume: { ml: 60, floz_hundredths: 203, typed_unit: 'ml' } }]
	});

	it('opens a new bottle advanced only on ?advanced=1', () => {
		expect(isAdvanced('bottles', '1')).toBe(true);
		for (const param of [null, '', '0', '2', 'true']) {
			expect(isAdvanced('bottles', param)).toBe(false);
		}
	});

	it('opens an edit advanced on ?advanced=1 or by itself when the bottle has sources', () => {
		expect(isAdvanced('bottles', '1', bottle())).toBe(true);
		expect(isAdvanced('bottles', null, withSources)).toBe(true);
		expect(isAdvanced('bottles', 'x', withSources)).toBe(true);
		expect(isAdvanced('bottles', null, bottle())).toBe(false);
	});

	it('?advanced=0 opens the simple form over a bottle with sources', () => {
		expect(isAdvanced('bottles', '0', withSources)).toBe(false);
	});

	it('reads no advanced data from an entry of a kind without a data check', () => {
		expect(isAdvanced('diapers', null, diaper)).toBe(false);
		expect(isAdvanced('meals', '0', meal)).toBe(false);
		expect(isAdvanced('pumping', null, pumping())).toBe(false);
	});

	it('is never advanced for a tracker without an advanced form', () => {
		for (const kind of ['diapers', 'meals', 'pumping', 'sleep'] as const) {
			expect(isAdvanced(kind, '1')).toBe(false);
		}
	});

	describe('nursing', () => {
		const kg = { grams: 3400, ozHundredths: 11993, typedUnit: 'kg' } as const;

		it('opens a new session advanced only on ?advanced=1', () => {
			expect(isAdvanced('nursing', '1')).toBe(true);
			for (const param of [null, '', '0', '2']) expect(isAdvanced('nursing', param)).toBe(false);
		});

		it('opens an edit advanced by itself when either weight is set', () => {
			expect(isAdvanced('nursing', null, nursing())).toBe(false);
			expect(isAdvanced('nursing', null, nursing({ weightBefore: kg }))).toBe(true);
			expect(isAdvanced('nursing', null, nursing({ weightAfter: kg }))).toBe(true);
			expect(isAdvanced('nursing', '1', nursing())).toBe(true);
		});

		it('?advanced=0 opens the simple form over a weighed session', () => {
			expect(isAdvanced('nursing', '0', nursing({ weightBefore: kg }))).toBe(false);
		});
	});
});

describe('advancedToggle', () => {
	it('offers the advanced form from the simple one, and one total from the advanced one', () => {
		expect(advancedToggle('bottles', false)).toEqual({
			href: '?advanced=1',
			label: 'Advanced: volume per source'
		});
		expect(advancedToggle('bottles', true)).toEqual({
			href: '?advanced=0',
			label: 'Use one total'
		});
	});

	it('offers the weigh-ins to nursing, and a way back that does not say one total', () => {
		expect(advancedToggle('nursing', false)).toEqual({
			href: '?advanced=1',
			label: 'Advanced: weigh before and after'
		});
		expect(advancedToggle('nursing', true)).toEqual({
			href: '?advanced=0',
			label: 'Hide weigh-ins'
		});
	});

	it('offers nothing for a tracker without an advanced form', () => {
		expect(advancedToggle('diapers', false)).toBeNull();
		expect(advancedToggle('pumping', false)).toBeNull();
	});
});

describe('milkStatus', () => {
	const w = (grams: number, ozHundredths: number) =>
		({ grams, ozHundredths, typedUnit: 'kg' }) as const;
	const weighed = nursing({
		weightBefore: w(3400, 11993),
		weightAfter: w(3462, 12212),
		milk: { gain: { grams: 62, ozHundredths: 219 }, volume: { ml: 60, flozHundredths: 204 } }
	});
	const BASIS =
		'The volume assumes about 1.03 g per ml, an approximation in the range commonly used (1.0\u20131.03 g/ml); not measured. A single-feed reading from a home scale can be off by several grams.';

	it('states the gain and the volume from the API in the parent units, then the basis', () => {
		expect(milkStatus('nursing', weighed, ML)).toBe(`Weight gain 62 g, about 60 ml. ${BASIS}`);
		expect(milkStatus('nursing', weighed, FLOZ)).toBe(
			`Weight gain 2.19 oz, about 2.04 fl oz. ${BASIS}`
		);
	});

	it('says "No estimate" when both weights are set and the API gave no estimate', () => {
		const lighter = nursing({ weightBefore: w(3462, 12212), weightAfter: w(3400, 11993) });
		expect(milkStatus('nursing', lighter, ML)).toBe('No estimate');
	});

	it('says nothing while a weight is missing', () => {
		expect(milkStatus('nursing', nursing({ weightBefore: w(3400, 11993) }), ML)).toBeNull();
		expect(milkStatus('nursing', nursing({ weightAfter: w(3400, 11993) }), ML)).toBeNull();
		expect(milkStatus('nursing', nursing(), ML)).toBeNull();
	});

	it('says nothing for another tracker, and no judgement wording anywhere', () => {
		expect(milkStatus('bottles', bottle(), ML)).toBeNull();
		for (const text of [milkStatus('nursing', weighed, ML), 'No estimate']) {
			expect(text).not.toMatch(/good|low|enough|concern|normal/i);
		}
	});
});
