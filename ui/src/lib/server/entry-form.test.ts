import { describe, expect, it } from 'vitest';
import { MEAL_TEXT_MAX_LEN } from '../constants';
import {
	ENTRY_FIELD_MESSAGES,
	FINISHED_SIDE_MESSAGE,
	REOPEN_MESSAGE,
	volumeRangeMessage,
	WHAT_MISSING_MESSAGE,
	WHAT_TOO_LONG_MESSAGE
} from './entry-failure';
import { floorMinutes, readEntryForm, type EntryMode } from './entry-form';
import { growthRangeMessage } from './growth-form';

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

describe('readEntryForm final-wave refusals', () => {
	const EDIT_FINISHED: EntryMode = { kind: 'edit', running: false };
	const EDIT_RUNNING: EntryMode = { kind: 'edit', running: true };

	it.each(['sleep', 'nursing'] as const)(
		'%s: clearing the end of a finished session is refused, not silently kept',
		(kind) => {
			const fields = {
				started_at: '2026-11-20T10:00',
				ended_at: '',
				ended_at_original: '2026-11-20T10:30',
				left_minutes: '0',
				right_minutes: '0',
				left_seconds_original: '0',
				right_seconds_original: '0'
			};
			expect(readEntryForm(kind, form(fields), MADRID, EDIT_FINISHED)).toMatchObject({
				input: null,
				field: 'ended_at',
				message: REOPEN_MESSAGE
			});
			const running = { ...fields, ended_at_original: '' };
			expect(readEntryForm(kind, form(running), MADRID, EDIT_RUNNING).field).toBeNull();
		}
	);

	it('nursing create: a side next to an end time is refused with its own message', () => {
		const r = readEntryForm(
			'nursing',
			form({
				started_at: '2026-11-20T10:00',
				ended_at: '2026-11-20T10:30',
				left_minutes: '10',
				right_minutes: '',
				side: 'left'
			}),
			MADRID,
			CREATE
		);
		expect(r).toMatchObject({ input: null, field: 'side', message: FINISHED_SIDE_MESSAGE });
		expect(FINISHED_SIDE_MESSAGE).not.toBe(ENTRY_FIELD_MESSAGES.side);
		const unknown = readEntryForm(
			'nursing',
			form({ started_at: '', side: 'both' }),
			MADRID,
			CREATE
		);
		expect(unknown).toMatchObject({ field: 'side', message: null });
	});

	it('meals: a blank what and a too long one get different messages', () => {
		expect(readEntryForm('meals', form({ what: ' ' }), MADRID, CREATE)).toMatchObject({
			field: 'what',
			message: WHAT_MISSING_MESSAGE
		});
		const long = 'a'.repeat(MEAL_TEXT_MAX_LEN + 1);
		expect(readEntryForm('meals', form({ what: long }), MADRID, CREATE)).toMatchObject({
			field: 'what',
			message: WHAT_TOO_LONG_MESSAGE
		});
		const fits = 'ñ'.repeat(MEAL_TEXT_MAX_LEN);
		expect(readEntryForm('meals', form({ what: fits }), MADRID, CREATE).field).toBeNull();
	});

	it.each([
		['bottles', '0', true],
		['bottles', '1001', true],
		['bottles', '1000', false],
		['pumping', '2001', true],
		['pumping', '2000', false],
		['pumping', '0', true]
	] as const)('%s: %s ml out of range says so', (kind, amount, refused) => {
		const fields = {
			at: '',
			started_at: '',
			contents: ['formula'],
			duration_minutes: '10',
			side: 'both',
			volume: amount,
			volume_unit: 'ml'
		};
		const r = readEntryForm(kind, form(fields), MADRID, CREATE);
		if (refused) {
			expect(r).toMatchObject({ field: 'volume', message: volumeRangeMessage(kind) });
		} else {
			expect(r.field).toBeNull();
		}
	});

	it('a malformed volume keeps the format message', () => {
		const r = readEntryForm(
			'bottles',
			form({ contents: ['formula'], volume: '1.5', volume_unit: 'ml' }),
			MADRID,
			CREATE
		);
		expect(r).toMatchObject({ field: 'volume', message: null });
	});

	it('ounces take a decimal comma as well as a point', () => {
		for (const [typed, hundredths] of [
			['4,25', 425],
			['4.25', 425],
			['4,5', 450]
		] as const) {
			const r = readEntryForm(
				'bottles',
				form({ contents: ['formula'], volume: typed, volume_unit: 'floz' }),
				MADRID,
				CREATE
			);
			expect(r.input).toMatchObject({ volume: { unit: 'floz', amount: hundredths } });
		}
		for (const typed of ['4,,25', '4.2.5', '4,', '4,567']) {
			const r = readEntryForm(
				'bottles',
				form({ contents: ['formula'], volume: typed, volume_unit: 'floz' }),
				MADRID,
				CREATE
			);
			expect(r).toMatchObject({ input: null, field: 'volume' });
		}
		expect(
			readEntryForm(
				'bottles',
				form({ contents: ['formula'], volume: '12,5', volume_unit: 'ml' }),
				MADRID,
				CREATE
			).field
		).toBe('volume');
	});
});

describe('readEntryForm advanced bottles', () => {
	const EDIT: EntryMode = { kind: 'edit', running: false };
	const valid = {
		advanced: '1',
		at: '2026-11-20T10:30',
		contents: ['breast_milk', 'formula'],
		volume_unit: 'ml',
		sources_breast_milk: '60',
		sources_formula: '90',
		sources_cow_milk: '',
		sources_other: '',
		note: ' warm '
	};
	// A bottle stored with breast milk 60 and formula 90, shown in ml as the edit page loads it.
	const stored = {
		...valid,
		at_original: '2026-11-20T10:30',
		volume_unit_original: 'ml',
		sources_breast_milk_original: '60',
		sources_formula_original: '90',
		sources_cow_milk_original: '',
		sources_other_original: '',
		sources_original: 'breast_milk,formula'
	};
	const sent = (fields: Record<string, string | string[]>, mode: EntryMode = CREATE) =>
		readEntryForm('bottles', form(fields), MADRID, mode);

	it('sends one amount per ticked source in the unit typed, and neither volume nor contents', () => {
		const r = sent(valid);
		expect(r.field).toBeNull();
		expect(r.input).toEqual({
			at: '2026-11-20T09:30:00Z',
			sources: { unit: 'ml', amounts: { breast_milk: 60, formula: 90 } },
			note: 'warm'
		});
		expect(r.values).toMatchObject({
			advanced: '1',
			sources_breast_milk: '60',
			sources_formula: '90',
			sources_cow_milk: '',
			volume_unit: 'ml'
		});
	});

	it('turns ounces into hundredths of an ounce, a decimal comma included', () => {
		const r = sent({
			...valid,
			volume_unit: 'floz',
			sources_breast_milk: '2,25',
			sources_formula: '3'
		});
		expect(r.input).toMatchObject({
			sources: { unit: 'floz', amounts: { breast_milk: 225, formula: 300 } }
		});
	});

	it('ignores an amount typed for a source whose box is not ticked', () => {
		const r = sent({
			...valid,
			contents: ['formula'],
			sources_breast_milk: 'junk',
			sources_other: '7'
		});
		expect(r.input).toMatchObject({ sources: { unit: 'ml', amounts: { formula: 90 } } });
	});

	it('sends the sources in the fixed order whatever order the boxes came in', () => {
		const r = sent({ ...valid, contents: ['formula', 'breast_milk'] });
		expect(Object.keys((r.input?.sources as { amounts: object }).amounts)).toEqual([
			'breast_milk',
			'formula'
		]);
	});

	it.each([
		['blank', ''],
		['spaces', '   '],
		['zero', '0'],
		['exponent', '1e3'],
		['hex', '0x10'],
		['double minus', '--1'],
		['negative', '-5'],
		['fraction in ml', '1.5'],
		['too many decimals in ounces', '4.567'],
		['unsafe integer', '9007199254740993']
	])('names sources for a ticked source typed as %s', (_name, text) => {
		for (const unit of ['ml', 'floz']) {
			const r = sent({ ...valid, volume_unit: unit, sources_formula: text });
			if (unit === 'ml' || !['1.5'].includes(text)) {
				expect(r).toMatchObject({ input: null, field: 'sources', message: null });
			}
			expect(r.values.sources_formula).toBe(text);
		}
	});

	it('names sources for a unit that is not one', () => {
		for (const unit of ['', 'liter']) {
			expect(sent({ ...valid, volume_unit: unit })).toMatchObject({
				input: null,
				field: 'sources'
			});
		}
	});

	it('names contents when no box is ticked or one is unknown, before any amount', () => {
		expect(sent({ ...valid, contents: [], sources_formula: '' })).toMatchObject({
			field: 'contents'
		});
		expect(sent({ ...valid, contents: ['formula', 'juice'] })).toMatchObject({ field: 'contents' });
		expect(sent({ ...valid, at: 'x', contents: [] })).toMatchObject({ field: 'at' });
	});

	it('refuses a total over the bottle range in millilitres, naming volume', () => {
		const r = sent({ ...valid, sources_breast_milk: '500', sources_formula: '501' });
		expect(r).toMatchObject({
			input: null,
			field: 'volume',
			message: volumeRangeMessage('bottles')
		});
		expect(sent({ ...valid, sources_breast_milk: '500', sources_formula: '500' }).field).toBeNull();
	});

	it('leaves a total in ounces to the API', () => {
		const r = sent({
			...valid,
			volume_unit: 'floz',
			sources_breast_milk: '90',
			sources_formula: '90'
		});
		expect(r.field).toBeNull();
	});

	describe('edit', () => {
		it('sends neither sources, volume nor contents when nothing changed', () => {
			const r = sent(stored, EDIT);
			expect(r.field).toBeNull();
			expect(r.input).toEqual({ at: '2026-11-20T09:30:00Z', note: 'warm' });
		});

		it('sends the whole set when one amount changed', () => {
			const r = sent({ ...stored, sources_formula: '95' }, EDIT);
			expect(r.input).toMatchObject({
				sources: { unit: 'ml', amounts: { breast_milk: 60, formula: 95 } }
			});
			expect(r.input).not.toHaveProperty('volume');
			expect(r.input).not.toHaveProperty('contents');
		});

		it('sends the set when only the unit changed', () => {
			const r = sent({ ...stored, volume_unit: 'floz' }, EDIT);
			expect(r.input).toMatchObject({
				sources: { unit: 'floz', amounts: { breast_milk: 60 * 100, formula: 90 * 100 } }
			});
		});

		it('sends the set without a source that was unticked, and with one that was added', () => {
			expect(sent({ ...stored, contents: ['breast_milk'] }, EDIT).input).toMatchObject({
				sources: { unit: 'ml', amounts: { breast_milk: 60 } }
			});
			const added = sent(
				{ ...stored, contents: ['breast_milk', 'formula', 'other'], sources_other: '30' },
				EDIT
			);
			expect(added.input).toMatchObject({
				sources: { unit: 'ml', amounts: { breast_milk: 60, formula: 90, other: 30 } }
			});
		});

		it('a text typed for an unticked source changes nothing', () => {
			const r = sent({ ...stored, sources_other: '44' }, EDIT);
			expect(r.input).not.toHaveProperty('sources');
		});

		it('a ticked source left blank is a sources error, never a silent drop', () => {
			expect(sent({ ...stored, sources_formula: '' }, EDIT)).toMatchObject({
				input: null,
				field: 'sources'
			});
		});

		it('advanced over a simple bottle sends the typed sources, and refuses blank ones', () => {
			const simple = {
				...stored,
				sources_breast_milk_original: '',
				sources_formula_original: '',
				sources_original: ''
			};
			expect(sent(simple, EDIT).input).toMatchObject({
				sources: { unit: 'ml', amounts: { breast_milk: 60, formula: 90 } }
			});
			expect(sent({ ...simple, sources_formula: '' }, EDIT)).toMatchObject({
				input: null,
				field: 'sources'
			});
		});

		it.each([
			'sources_original',
			'volume_unit_original',
			'sources_breast_milk_original',
			'sources_cow_milk_original'
		])('fails closed on a missing %s', (name) => {
			const fields: Record<string, string | string[]> = { ...stored };
			delete fields[name];
			expect(sent(fields, EDIT)).toMatchObject({ input: null, field: 'sources' });
		});

		it('fails closed on a stored source name it does not know', () => {
			expect(sent({ ...stored, sources_original: 'breast_milk,juice' }, EDIT)).toMatchObject({
				input: null,
				field: 'sources'
			});
		});

		it('the simple form over a bottle with sources still sends its volume', () => {
			const r = sent(
				{
					at: '2026-11-20T10:30',
					at_original: '2026-11-20T10:30',
					contents: ['breast_milk', 'formula'],
					volume: '150',
					volume_original: '150',
					volume_unit: 'ml',
					volume_unit_original: 'ml',
					sources_original: 'breast_milk,formula',
					note: ''
				},
				EDIT
			);
			expect(r.input).toMatchObject({
				contents: ['breast_milk', 'formula'],
				volume: { unit: 'ml', amount: 150 }
			});
			expect(r.input).not.toHaveProperty('sources');
		});
	});

	it('the simple form never reads a source text', () => {
		const r = sent({
			at: '',
			contents: ['formula'],
			volume: '90',
			volume_unit: 'ml',
			sources_formula: '5'
		});
		expect(r.input).toEqual({
			contents: ['formula'],
			volume: { unit: 'ml', amount: 90 },
			note: null
		});
	});
});

describe('readEntryForm nursing weigh-ins', () => {
	const EDIT_FINISHED: EntryMode = { kind: 'edit', running: false };
	const EDIT_RUNNING: EntryMode = { kind: 'edit', running: true };
	const finished = {
		advanced: '1',
		started_at: '2026-11-20T10:00',
		ended_at: '2026-11-20T10:30',
		left_minutes: '10',
		right_minutes: '5',
		weight_before_unit: 'kg',
		weight_before_kg: '3.4',
		weight_before_lb: '',
		weight_before_oz: '',
		weight_after_unit: 'kg',
		weight_after_kg: '3.462',
		weight_after_lb: '',
		weight_after_oz: ''
	};
	const sent = (fields: Record<string, string>, mode: EntryMode = CREATE) =>
		readEntryForm('nursing', form(fields), MADRID, mode);

	it('reads kilograms as whole grams', () => {
		const r = sent(finished);
		expect(r.field).toBeNull();
		expect(r.input).toMatchObject({
			weight_before: { unit: 'kg', amount: 3400 },
			weight_after: { unit: 'kg', amount: 3462 },
			left_seconds: 600,
			right_seconds: 300
		});
		expect(r.values).toMatchObject({ weight_before_kg: '3.4', weight_after_kg: '3.462' });
	});

	it('reads pounds and ounces as hundredths of an ounce', () => {
		const r = sent({
			...finished,
			weight_before_unit: 'lb',
			weight_before_kg: '',
			weight_before_lb: '7',
			weight_before_oz: '8.5',
			weight_after_unit: 'lb',
			weight_after_kg: '',
			weight_after_lb: '7',
			weight_after_oz: '9,25'
		});
		expect(r.input).toMatchObject({
			weight_before: { unit: 'lb', amount: 12050 },
			weight_after: { unit: 'lb', amount: 12125 }
		});
	});

	it('sends one weight alone, and none when both are blank', () => {
		const only = sent({ ...finished, weight_after_kg: '' });
		expect(only.input).toHaveProperty('weight_before');
		expect(only.input).not.toHaveProperty('weight_after');
		const none = sent({ ...finished, weight_before_kg: '', weight_after_kg: '' });
		expect(none.field).toBeNull();
		expect(none.input).not.toHaveProperty('weight_before');
		expect(none.input).not.toHaveProperty('weight_after');
	});

	it('reads weights on a running timer too', () => {
		const r = sent({
			advanced: '1',
			started_at: '',
			ended_at: '',
			side: 'left',
			weight_before_unit: 'kg',
			weight_before_kg: '3.4'
		});
		expect(r.input).toEqual({
			side: 'left',
			weight_before: { unit: 'kg', amount: 3400 },
			note: null
		});
	});

	it('does not read a weight from a form that is not advanced', () => {
		const r = sent({ ...finished, advanced: '' });
		expect(r.input).not.toHaveProperty('weight_before');
		expect(r.input).not.toHaveProperty('weight_after');
	});

	it.each([
		['exponent', '1e3'],
		['hex', '0x10'],
		['double minus', '--1'],
		['negative', '-3'],
		['four decimals', '3.4567'],
		['words', 'heavy']
	])('names the weight for a kilogram text typed as %s', (_name, text) => {
		expect(sent({ ...finished, weight_before_kg: text })).toMatchObject({
			input: null,
			field: 'weight_before',
			message: null
		});
		expect(sent({ ...finished, weight_after_kg: text })).toMatchObject({
			input: null,
			field: 'weight_after',
			message: null
		});
	});

	it('refuses a kilogram weight outside 200 g to 250 kg with the growth range message', () => {
		for (const kg of ['0.1', '0.199', '250.001', '999']) {
			expect(sent({ ...finished, weight_before_kg: kg })).toMatchObject({
				input: null,
				field: 'weight_before',
				message: growthRangeMessage('weight')
			});
		}
		expect(sent({ ...finished, weight_before_kg: '0.2', weight_after_kg: '250' }).field).toBeNull();
	});

	it('refuses 16 ounces and a unit that is not one', () => {
		expect(
			sent({ ...finished, weight_before_unit: 'lb', weight_before_lb: '7', weight_before_oz: '16' })
		).toMatchObject({ field: 'weight_before' });
		expect(sent({ ...finished, weight_before_unit: 'st' })).toMatchObject({
			field: 'weight_before'
		});
	});

	it('names weight_before before weight_after, and both after side', () => {
		expect(sent({ ...finished, weight_before_kg: 'x', weight_after_kg: 'y' }).field).toBe(
			'weight_before'
		);
		expect(sent({ started_at: '', side: 'both', advanced: '1', weight_before_kg: 'x' }).field).toBe(
			'side'
		);
	});

	describe('edit', () => {
		const stored = {
			...finished,
			left_seconds_original: '600',
			right_seconds_original: '300',
			weight_before_unit_original: 'kg',
			weight_before_kg_original: '3.4',
			weight_before_lb_original: '7',
			weight_before_oz_original: '7.99',
			weight_after_unit_original: 'kg',
			weight_after_kg_original: '3.462',
			weight_after_lb_original: '7',
			weight_after_oz_original: '10.12'
		};

		it('sends no weight when its texts are as loaded', () => {
			const r = sent(stored, EDIT_FINISHED);
			expect(r.field).toBeNull();
			expect(r.input).not.toHaveProperty('weight_before');
			expect(r.input).not.toHaveProperty('weight_after');
		});

		it('sends only the weight whose text changed', () => {
			const r = sent({ ...stored, weight_after_kg: '3.5' }, EDIT_FINISHED);
			expect(r.input).toMatchObject({ weight_after: { unit: 'kg', amount: 3500 } });
			expect(r.input).not.toHaveProperty('weight_before');
		});

		it('sends the other unit when the unit was flipped, from its own pre-filled texts', () => {
			const r = sent(
				{
					...stored,
					weight_before_unit: 'lb',
					weight_before_lb: '7',
					weight_before_oz: '7.99'
				},
				EDIT_FINISHED
			);
			expect(r.input).toMatchObject({ weight_before: { unit: 'lb', amount: 7 * 1600 + 799 } });
		});

		it('clears a weight with null only when the original was filled', () => {
			const cleared = sent({ ...stored, weight_after_kg: '' }, EDIT_FINISHED);
			expect(cleared.input).toMatchObject({ weight_after: null });
			const empty = {
				...stored,
				weight_after_kg: '',
				weight_after_kg_original: '',
				weight_after_lb_original: '',
				weight_after_oz_original: ''
			};
			expect(sent(empty, EDIT_FINISHED).input).not.toHaveProperty('weight_after');
		});

		it('adds a weight to a session that had none', () => {
			const none = {
				...stored,
				weight_after_kg: '3.5',
				weight_after_kg_original: '',
				weight_after_lb_original: '',
				weight_after_oz_original: ''
			};
			expect(sent(none, EDIT_FINISHED).input).toMatchObject({
				weight_after: { unit: 'kg', amount: 3500 }
			});
		});

		it('reads weights on a running session and never sends seconds there', () => {
			const r = sent(
				{
					...stored,
					ended_at: '',
					weight_after_kg: '3.5',
					weight_after_kg_original: '',
					weight_after_lb_original: '',
					weight_after_oz_original: ''
				},
				EDIT_RUNNING
			);
			expect(r.input).toMatchObject({ weight_after: { unit: 'kg', amount: 3500 } });
			expect(r.input).not.toHaveProperty('left_seconds');
		});

		it.each([
			'weight_before_unit_original',
			'weight_before_kg_original',
			'weight_before_lb_original',
			'weight_before_oz_original'
		])('fails closed on a missing %s', (name) => {
			const fields: Record<string, string> = { ...stored };
			delete fields[name];
			expect(sent(fields, EDIT_FINISHED)).toMatchObject({
				input: null,
				field: 'weight_before'
			});
		});

		it('fails closed on an original unit it does not know', () => {
			expect(sent({ ...stored, weight_after_unit_original: 'st' }, EDIT_FINISHED)).toMatchObject({
				input: null,
				field: 'weight_after'
			});
		});

		it('a simple form (not advanced) over a weighed session sends no weight', () => {
			const r = sent({ ...stored, advanced: '' }, EDIT_FINISHED);
			expect(r.input).not.toHaveProperty('weight_before');
			expect(r.input).not.toHaveProperty('weight_after');
		});
	});
});
