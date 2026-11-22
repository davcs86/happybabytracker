import { describe, expect, it } from 'vitest';
import {
	createValues,
	editValues,
	entryInstant,
	isRunning,
	localDate,
	ranOver,
	volumeText
} from './entry-edit';
import type { Bottle, Diaper, Meal, Nursing, Pumping, Sleep } from './types';

const MADRID = 'Europe/Madrid';
const HOUR_MS = 3_600_000;

const bottle = (over: Partial<Bottle> = {}): Bottle => ({
	id: 'e1',
	at: '2026-11-20T09:30:00Z',
	contents: ['breast_milk', 'formula'],
	volume: { ml: 120, floz_hundredths: 406, typed_unit: 'ml' },
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
		expect(createValues('bottles', now, MADRID, 'floz')).toEqual({
			at: '2026-03-29T01:45',
			contents: [],
			volume: '',
			volume_unit: 'floz',
			note: ''
		});
		expect(createValues('diapers', now, 'UTC', 'ml')).toEqual({
			at: '2026-03-29T00:45',
			kind: '',
			note: ''
		});
		expect(createValues('meals', now, 'UTC', 'ml')).toEqual({
			at: '2026-03-29T00:45',
			what: '',
			reaction: '',
			note: ''
		});
		expect(createValues('pumping', now, 'UTC', 'ml')).toEqual({
			started_at: '2026-03-29T00:45',
			duration_minutes: '',
			volume: '',
			volume_unit: 'ml',
			side: '',
			note: ''
		});
		expect(createValues('sleep', now, 'UTC', 'ml')).toEqual({
			started_at: '2026-03-29T00:45',
			ended_at: '',
			note: ''
		});
		expect(createValues('nursing', now, 'UTC', 'ml')).toEqual({
			started_at: '2026-03-29T00:45',
			ended_at: '',
			left_minutes: '',
			right_minutes: '',
			side: '',
			note: ''
		});
	});

	it('carries no stored originals', () => {
		for (const kind of ['bottles', 'nursing', 'pumping', 'sleep', 'diapers', 'meals'] as const) {
			expect(
				Object.keys(createValues(kind, now, 'UTC', 'ml')).filter((k) => k.endsWith('_original'))
			).toEqual([]);
		}
	});
});

describe('editValues', () => {
	it('shows a bottle in the preferred unit and keeps the stored values as originals', () => {
		const v = editValues('bottles', bottle(), MADRID, 'floz');
		expect(v).toEqual({
			at: '2026-11-20T10:30',
			at_original: '2026-11-20T10:30',
			contents: ['breast_milk', 'formula'],
			volume: '4.06',
			volume_original: '4.06',
			volume_unit: 'floz',
			volume_unit_original: 'floz',
			note: ''
		});
	});

	it('shows the typed column when it matches the preference', () => {
		const typed = bottle({ volume: { ml: 126, floz_hundredths: 425, typed_unit: 'floz' } });
		expect(editValues('bottles', typed, 'UTC', 'floz')).toMatchObject({
			volume: '4.25',
			volume_unit: 'floz'
		});
		expect(editValues('bottles', typed, 'UTC', 'ml')).toMatchObject({
			volume: '126',
			volume_unit: 'ml'
		});
	});

	it('floors a finished nursing session to minutes and keeps the raw seconds', () => {
		const v = editValues('nursing', nursing(), 'UTC', 'ml');
		expect(v).toEqual({
			started_at: '2026-11-20T10:00',
			started_at_original: '2026-11-20T10:00',
			ended_at: '2026-11-20T10:12',
			ended_at_original: '2026-11-20T10:12',
			left_minutes: '12',
			right_minutes: '1',
			left_seconds_original: '723',
			right_seconds_original: '61',
			note: 'ok'
		});
	});

	it('gives a running nursing session no minutes, no originals and no side', () => {
		const v = editValues(
			'nursing',
			nursing({ endedAt: null, side: 'left', sideSince: '2026-11-20T10:00:00Z', leftSeconds: 0 }),
			'UTC',
			'ml'
		);
		expect(v).toEqual({
			started_at: '2026-11-20T10:00',
			started_at_original: '2026-11-20T10:00',
			ended_at: '',
			ended_at_original: '',
			note: 'ok'
		});
	});

	it('floors a pumping duration and keeps the raw seconds', () => {
		expect(editValues('pumping', pumping(), 'UTC', 'ml')).toEqual({
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
		expect(editValues('sleep', sleep(), 'UTC', 'ml')).toMatchObject({
			ended_at: '',
			ended_at_original: ''
		});
		expect(
			editValues('sleep', sleep({ endedAt: '2026-11-20T11:30:00Z' }), MADRID, 'ml')
		).toMatchObject({ started_at: '2026-11-20T11:00', ended_at: '2026-11-20T12:30' });
	});

	it('shows a diaper and a meal; a null reaction is blank', () => {
		expect(editValues('diapers', diaper, 'UTC', 'ml')).toEqual({
			at: '2026-11-20T10:00',
			at_original: '2026-11-20T10:00',
			kind: 'both',
			note: 'rash'
		});
		expect(editValues('meals', meal, 'UTC', 'ml')).toEqual({
			at: '2026-11-20T10:00',
			at_original: '2026-11-20T10:00',
			what: 'rice',
			reaction: '',
			note: ''
		});
	});

	it('reads the time on the kid clock across the spring change', () => {
		const v = editValues('diapers', { ...diaper, at: '2026-03-29T01:30:00Z' }, MADRID, 'ml');
		expect(v.at).toBe('2026-03-29T03:30');
	});
});
