import { describe, expect, it } from 'vitest';
import { readEntryForm } from './entry-form';

const MADRID = 'Europe/Madrid';
const form = (fields: Record<string, string | string[]>) => {
	const data = new FormData();
	for (const [k, v] of Object.entries(fields)) {
		for (const one of Array.isArray(v) ? v : [v]) data.append(k, one);
	}
	return data;
};

describe('readEntryForm bottles', () => {
	const valid = {
		at: '2026-11-20T10:30',
		contents: ['breast_milk', 'formula'],
		volume: '120',
		volume_unit: 'ml',
		note: ' warm '
	};

	it('reads a valid bottle with several contents, the time in the kid zone', () => {
		const r = readEntryForm('bottles', form(valid), MADRID);
		expect(r.field).toBeNull();
		expect(r.input).toEqual({
			at: '2026-11-20T09:30:00Z',
			contents: ['breast_milk', 'formula'],
			volume: { unit: 'ml', amount: 120 },
			note: 'warm'
		});
		expect(r.values).toMatchObject({ ...valid, contents: ['breast_milk', 'formula'] });
	});

	it('names contents when no box is ticked, keeping what was typed', () => {
		const r = readEntryForm('bottles', form({ ...valid, contents: [] }), MADRID);
		expect(r).toMatchObject({ input: null, field: 'contents' });
		expect(r.values.contents).toEqual([]);
		expect(r.values.volume).toBe('120');
	});

	it('names contents for an unknown content', () => {
		const r = readEntryForm('bottles', form({ ...valid, contents: ['formula', 'juice'] }), MADRID);
		expect(r).toMatchObject({ input: null, field: 'contents' });
	});

	it('turns a volume typed in ounces into hundredths of a fluid ounce', () => {
		for (const [typed, hundredths] of [
			['4', 400],
			['4.5', 450],
			['4.06', 406],
			['0.03', 3]
		] as const) {
			const r = readEntryForm(
				'bottles',
				form({ ...valid, volume: typed, volume_unit: 'floz' }),
				MADRID
			);
			expect(r.input).toMatchObject({ volume: { unit: 'floz', amount: hundredths } });
		}
	});

	it.each([
		['', 'ml'],
		['abc', 'ml'],
		['12.5', 'ml'],
		['-3', 'ml'],
		['4.567', 'floz'],
		['4,5', 'floz'],
		['120', 'liter'],
		['120', '']
	])('names volume for %j in %j', (volume, unit) => {
		const r = readEntryForm('bottles', form({ ...valid, volume, volume_unit: unit }), MADRID);
		expect(r).toMatchObject({ input: null, field: 'volume' });
	});

	it('omits a blank time so the API takes now, and sends a blank note as null', () => {
		const r = readEntryForm('bottles', form({ ...valid, at: '', note: '  ' }), MADRID);
		expect(r.input).not.toHaveProperty('at');
		expect(r.input).toMatchObject({ note: null });
	});

	it('names at for a time that is not a local date and time', () => {
		expect(readEntryForm('bottles', form({ ...valid, at: 'soon' }), MADRID)).toMatchObject({
			input: null,
			field: 'at'
		});
	});

	it('reports the first invalid field in the API order: at, then contents, then volume', () => {
		const r = readEntryForm(
			'bottles',
			form({ at: 'x', contents: [], volume: '', volume_unit: 'ml' }),
			MADRID
		);
		expect(r.field).toBe('at');
		const r2 = readEntryForm(
			'bottles',
			form({ contents: [], volume: '', volume_unit: 'ml' }),
			MADRID
		);
		expect(r2.field).toBe('contents');
	});
});

describe('readEntryForm diapers', () => {
	it('reads a valid diaper', () => {
		const r = readEntryForm('diapers', form({ at: '', kind: 'dirty', note: '' }), MADRID);
		expect(r).toMatchObject({ field: null, input: { kind: 'dirty', note: null } });
	});

	it.each(['', 'soggy'])('names kind for %j', (kind) => {
		expect(readEntryForm('diapers', form({ kind }), MADRID)).toMatchObject({
			input: null,
			field: 'kind'
		});
	});
});

describe('readEntryForm meals', () => {
	it('reads a valid meal, trimming the text and nulling a blank reaction', () => {
		const r = readEntryForm(
			'meals',
			form({ at: '2026-07-20T12:00', what: '  pear  ', reaction: '', note: 'n' }),
			MADRID
		);
		expect(r.input).toEqual({
			at: '2026-07-20T10:00:00Z',
			what: 'pear',
			reaction: null,
			note: 'n'
		});
	});

	it.each(['', '   '])('names what for %j', (what) => {
		expect(readEntryForm('meals', form({ what }), MADRID)).toMatchObject({
			input: null,
			field: 'what'
		});
	});
});

describe('readEntryForm pumping', () => {
	const valid = {
		started_at: '2026-11-20T08:00',
		duration_minutes: '15',
		volume: '90',
		volume_unit: 'ml',
		side: 'both',
		note: ''
	};

	it('reads a valid session, minutes becoming seconds', () => {
		expect(readEntryForm('pumping', form(valid), MADRID).input).toEqual({
			started_at: '2026-11-20T07:00:00Z',
			duration_seconds: 900,
			volume: { unit: 'ml', amount: 90 },
			side: 'both',
			note: null
		});
	});

	it.each([
		['duration_minutes', '', 'duration_seconds'],
		['duration_minutes', '1.5', 'duration_seconds'],
		['duration_minutes', 'x', 'duration_seconds'],
		['volume', '', 'volume'],
		['side', 'middle', 'side'],
		['side', '', 'side'],
		['started_at', 'x', 'started_at']
	])('names %s=%j as the field %s', (key, value, field) => {
		expect(readEntryForm('pumping', form({ ...valid, [key]: value }), MADRID)).toMatchObject({
			input: null,
			field
		});
	});
});

describe('readEntryForm sleep', () => {
	it('reads a running timer: no end time is sent', () => {
		const r = readEntryForm(
			'sleep',
			form({ started_at: '2026-11-20T22:00', ended_at: '' }),
			MADRID
		);
		expect(r.input).toEqual({ started_at: '2026-11-20T21:00:00Z', note: null });
	});

	it('reads a finished session', () => {
		const r = readEntryForm(
			'sleep',
			form({ started_at: '2026-11-20T22:00', ended_at: '2026-11-21T06:30', note: 'woke once' }),
			MADRID
		);
		expect(r.input).toEqual({
			started_at: '2026-11-20T21:00:00Z',
			ended_at: '2026-11-21T05:30:00Z',
			note: 'woke once'
		});
	});

	it('names ended_at for an end time that is not a local date and time', () => {
		expect(
			readEntryForm('sleep', form({ started_at: '2026-11-20T22:00', ended_at: '6am' }), MADRID)
		).toMatchObject({ input: null, field: 'ended_at' });
	});
});

describe('readEntryForm nursing', () => {
	it('reads a timer started on a side', () => {
		const r = readEntryForm(
			'nursing',
			form({ started_at: '', ended_at: '', side: 'left' }),
			MADRID
		);
		expect(r.input).toEqual({ side: 'left', note: null });
	});

	it('reads a finished session, minutes becoming seconds, a blank side counting as none', () => {
		const r = readEntryForm(
			'nursing',
			form({
				started_at: '2026-11-20T10:00',
				ended_at: '2026-11-20T10:30',
				left_minutes: '12',
				right_minutes: ''
			}),
			MADRID
		);
		expect(r.input).toEqual({
			started_at: '2026-11-20T09:00:00Z',
			ended_at: '2026-11-20T09:30:00Z',
			left_seconds: 720,
			right_seconds: 0,
			note: null
		});
	});

	it.each([
		[{ side: 'both' }, 'side'],
		[{ ended_at: '2026-11-20T10:30', left_minutes: 'x' }, 'left_seconds'],
		[{ ended_at: '2026-11-20T10:30', right_minutes: '-1' }, 'right_seconds']
	])('names the field for %j', (extra, field) => {
		expect(
			readEntryForm('nursing', form({ started_at: '2026-11-20T10:00', ...extra }), MADRID)
		).toMatchObject({ input: null, field });
	});
});
