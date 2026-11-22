import { describe, expect, it } from 'vitest';
import { floorMinutes, readEntryForm, type EntryMode } from './entry-form';

const MADRID = 'Europe/Madrid';
const CREATE: EntryMode = { kind: 'create' };
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
		const r = readEntryForm('bottles', form(valid), MADRID, CREATE);
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
		const r = readEntryForm('bottles', form({ ...valid, contents: [] }), MADRID, CREATE);
		expect(r).toMatchObject({ input: null, field: 'contents' });
		expect(r.values.contents).toEqual([]);
		expect(r.values.volume).toBe('120');
	});

	it('names contents for an unknown content', () => {
		const r = readEntryForm(
			'bottles',
			form({ ...valid, contents: ['formula', 'juice'] }),
			MADRID,
			CREATE
		);
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
				MADRID,
				CREATE
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
		const r = readEntryForm(
			'bottles',
			form({ ...valid, volume, volume_unit: unit }),
			MADRID,
			CREATE
		);
		expect(r).toMatchObject({ input: null, field: 'volume' });
	});

	it('omits a blank time so the API takes now, and sends a blank note as null', () => {
		const r = readEntryForm('bottles', form({ ...valid, at: '', note: '  ' }), MADRID, CREATE);
		expect(r.input).not.toHaveProperty('at');
		expect(r.input).toMatchObject({ note: null });
	});

	it('names at for a time that is not a local date and time', () => {
		expect(readEntryForm('bottles', form({ ...valid, at: 'soon' }), MADRID, CREATE)).toMatchObject({
			input: null,
			field: 'at'
		});
	});

	it('reports the first invalid field in the API order: at, then contents, then volume', () => {
		const r = readEntryForm(
			'bottles',
			form({ at: 'x', contents: [], volume: '', volume_unit: 'ml' }),
			MADRID,
			CREATE
		);
		expect(r.field).toBe('at');
		const r2 = readEntryForm(
			'bottles',
			form({ contents: [], volume: '', volume_unit: 'ml' }),
			MADRID,
			CREATE
		);
		expect(r2.field).toBe('contents');
	});
});

describe('readEntryForm diapers', () => {
	it('reads a valid diaper', () => {
		const r = readEntryForm('diapers', form({ at: '', kind: 'dirty', note: '' }), MADRID, CREATE);
		expect(r).toMatchObject({ field: null, input: { kind: 'dirty', note: null } });
	});

	it.each(['', 'soggy'])('names kind for %j', (kind) => {
		expect(readEntryForm('diapers', form({ kind }), MADRID, CREATE)).toMatchObject({
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
			MADRID,
			CREATE
		);
		expect(r.input).toEqual({
			at: '2026-07-20T10:00:00Z',
			what: 'pear',
			reaction: null,
			note: 'n'
		});
	});

	it.each(['', '   '])('names what for %j', (what) => {
		expect(readEntryForm('meals', form({ what }), MADRID, CREATE)).toMatchObject({
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
		expect(readEntryForm('pumping', form(valid), MADRID, CREATE).input).toEqual({
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
		expect(
			readEntryForm('pumping', form({ ...valid, [key]: value }), MADRID, CREATE)
		).toMatchObject({
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
			MADRID,
			CREATE
		);
		expect(r.input).toEqual({ started_at: '2026-11-20T21:00:00Z', note: null });
	});

	it('reads a finished session', () => {
		const r = readEntryForm(
			'sleep',
			form({ started_at: '2026-11-20T22:00', ended_at: '2026-11-21T06:30', note: 'woke once' }),
			MADRID,
			CREATE
		);
		expect(r.input).toEqual({
			started_at: '2026-11-20T21:00:00Z',
			ended_at: '2026-11-21T05:30:00Z',
			note: 'woke once'
		});
	});

	it('names ended_at for an end time that is not a local date and time', () => {
		expect(
			readEntryForm(
				'sleep',
				form({ started_at: '2026-11-20T22:00', ended_at: '6am' }),
				MADRID,
				CREATE
			)
		).toMatchObject({ input: null, field: 'ended_at' });
	});
});

describe('readEntryForm nursing', () => {
	it('reads a timer started on a side', () => {
		const r = readEntryForm(
			'nursing',
			form({ started_at: '', ended_at: '', side: 'left' }),
			MADRID,
			CREATE
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
			MADRID,
			CREATE
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
			readEntryForm('nursing', form({ started_at: '2026-11-20T10:00', ...extra }), MADRID, CREATE)
		).toMatchObject({ input: null, field });
	});
});

describe('floorMinutes', () => {
	it('rounds down, never up', () => {
		expect([0, 59, 60, 719, 723, 1023, 3600].map(floorMinutes)).toEqual([0, 0, 1, 11, 12, 17, 60]);
	});
});

describe('readEntryForm nursing seconds by mode', () => {
	const EDIT_FINISHED: EntryMode = { kind: 'edit', running: false };
	const EDIT_RUNNING: EntryMode = { kind: 'edit', running: true };
	// 723 s on the left (12 min 3 s), 300 s on the right, as the timer credited them.
	const stored = { left_seconds_original: '723', right_seconds_original: '300' };
	const finished = {
		started_at: '2026-11-20T10:00',
		ended_at: '2026-11-20T10:30',
		note: 'x',
		...stored
	};

	it('create: a finished session sends minutes x 60, a blank count is 0', () => {
		const r = readEntryForm(
			'nursing',
			form({ ...finished, left_minutes: '12', right_minutes: '' }),
			MADRID,
			CREATE
		);
		expect(r.input).toMatchObject({ left_seconds: 720, right_seconds: 0 });
	});

	it('create: a running timer sends no seconds even if minutes are present', () => {
		const r = readEntryForm(
			'nursing',
			form({ side: 'left', left_minutes: '5', right_minutes: '5' }),
			MADRID,
			CREATE
		);
		expect(r.input).toEqual({ side: 'left', note: null });
	});

	it('edit finished, minutes untouched: no seconds, so the stored 723 s survive a note edit', () => {
		const r = readEntryForm(
			'nursing',
			form({ ...finished, left_minutes: '12', right_minutes: '5' }),
			MADRID,
			EDIT_FINISHED
		);
		expect(r.field).toBeNull();
		expect(r.input).not.toHaveProperty('left_seconds');
		expect(r.input).not.toHaveProperty('right_seconds');
		expect(r.input).toMatchObject({ note: 'x' });
	});

	it('edit finished, one side touched: only that side is sent', () => {
		const left = readEntryForm(
			'nursing',
			form({ ...finished, left_minutes: '10', right_minutes: '5' }),
			MADRID,
			EDIT_FINISHED
		);
		expect(left.input).toMatchObject({ left_seconds: 600 });
		expect(left.input).not.toHaveProperty('right_seconds');
		const right = readEntryForm(
			'nursing',
			form({ ...finished, left_minutes: '12', right_minutes: '6' }),
			MADRID,
			EDIT_FINISHED
		);
		expect(right.input).toMatchObject({ right_seconds: 360 });
		expect(right.input).not.toHaveProperty('left_seconds');
	});

	it('edit finished, a side cleared to blank is touched and means 0', () => {
		const r = readEntryForm(
			'nursing',
			form({ ...finished, left_minutes: '', right_minutes: '5' }),
			MADRID,
			EDIT_FINISHED
		);
		expect(r.input).toMatchObject({ left_seconds: 0 });
	});

	it('edit finished with a blank end time still decides the seconds by the minutes', () => {
		const r = readEntryForm(
			'nursing',
			form({ ...finished, ended_at: '', left_minutes: '13', right_minutes: '5' }),
			MADRID,
			EDIT_FINISHED
		);
		expect(r.input).toMatchObject({ left_seconds: 780 });
		expect(r.input).not.toHaveProperty('ended_at');
	});

	it('edit running: never seconds, minute fields are not even read', () => {
		const r = readEntryForm(
			'nursing',
			form({
				started_at: '2026-11-20T10:00',
				ended_at: '2026-11-20T10:30',
				left_minutes: 'junk',
				right_minutes: '-4'
			}),
			MADRID,
			EDIT_RUNNING
		);
		expect(r.field).toBeNull();
		expect(r.input).toEqual({
			started_at: '2026-11-20T09:00:00Z',
			ended_at: '2026-11-20T09:30:00Z',
			note: null
		});
	});

	it.each([
		['missing left', { left_seconds_original: undefined }, 'left_seconds'],
		['non-integer left', { left_seconds_original: '12.5' }, 'left_seconds'],
		['blank left', { left_seconds_original: '' }, 'left_seconds'],
		['negative left', { left_seconds_original: '-1' }, 'left_seconds'],
		['missing right', { right_seconds_original: undefined }, 'right_seconds'],
		['non-integer right', { right_seconds_original: 'abc' }, 'right_seconds']
	])('edit finished with a %s stored value is a field error, never 0', (_name, patch, field) => {
		const fields: Record<string, string> = {
			...finished,
			left_minutes: '12',
			right_minutes: '5'
		};
		for (const [k, v] of Object.entries(patch)) {
			if (v === undefined) delete fields[k];
			else fields[k] = v;
		}
		expect(readEntryForm('nursing', form(fields), MADRID, EDIT_FINISHED)).toMatchObject({
			input: null,
			field
		});
	});

	it('edit finished with a bad minutes entry names the side', () => {
		expect(
			readEntryForm(
				'nursing',
				form({ ...finished, left_minutes: 'x', right_minutes: '5' }),
				MADRID,
				EDIT_FINISHED
			)
		).toMatchObject({ input: null, field: 'left_seconds' });
	});
});
