import { describe, expect, it } from 'vitest';
import {
	MAX_EVERY_HOURS,
	MAX_SCHEDULE_TIMES,
	MEDICATION_TEXT_MAX_LEN,
	MIN_EVERY_HOURS,
	NOTE_MAX_LEN
} from '../constants';
import type { Medication, MedicationDose } from '../types';
import {
	COURSE_FIELD_MESSAGES,
	COURSE_LIMIT_MESSAGE,
	DOSE_FIELD_MESSAGES,
	DOSE_LIMIT_MESSAGE,
	courseApiFailure,
	courseCreateValues,
	courseEditValues,
	courseFieldFailure,
	doseApiFailure,
	doseEditValues,
	doseFieldFailure,
	readDoseForm,
	readMedicationForm
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

// A form as the browser posts it: times repeat, an unticked box is absent.
type Posted = Record<string, string | string[] | undefined>;
const posted = (fields: Posted) => {
	const data = new FormData();
	for (const [k, v] of Object.entries(fields)) {
		if (v === undefined) continue;
		for (const one of Array.isArray(v) ? v : [v]) data.append(k, one);
	}
	return data;
};

const MADRID = 'Europe/Madrid';
const create = (fields: Posted, zone = 'UTC') =>
	readMedicationForm(posted(fields), { kind: 'create', zone });
const OK = { name: 'Calpol', dose: '2.5 ml', kind: 'none', active: 'on' } as const;

describe('readMedicationForm: a new course', () => {
	it('sends the name, the dose and the schedule, and nothing the API defaults', () => {
		expect(create(OK)).toMatchObject({
			input: { name: 'Calpol', dose: '2.5 ml', schedule: { kind: 'none' } },
			field: null
		});
		expect(Object.keys(create(OK).input as object).sort()).toEqual(['dose', 'name', 'schedule']);
	});

	it('trims the texts it sends and keeps what was typed in the values', () => {
		const r = create({ ...OK, name: '  Calpol ', dose: ' 2.5 ml  ', note: '  with food ' });
		expect(r.input).toEqual({
			name: 'Calpol',
			dose: '2.5 ml',
			schedule: { kind: 'none' },
			note: 'with food'
		});
		expect(r.values).toMatchObject({ name: '  Calpol ', dose: ' 2.5 ml  ', note: '  with food ' });
	});

	it('sends active false only for an unticked box', () => {
		expect(create({ ...OK, active: undefined }).input).toMatchObject({ active: false });
		expect(create(OK).input).not.toHaveProperty('active');
		expect(create({ ...OK, active: undefined }).values.active).toBe(false);
		expect(create(OK).values.active).toBe(true);
	});

	it('refuses any other value of the box', () => {
		for (const active of ['', 'yes', 'true', 'ON', '1', 'on ']) {
			expect(create({ ...OK, active })).toMatchObject({ input: null, field: 'active' });
		}
	});

	it('sends fixed times sorted, dropping the blank slots', () => {
		const times = ['20:00', '', '08:00', '  ', '12:30'];
		expect(create({ ...OK, kind: 'times', times }).input).toMatchObject({
			schedule: { kind: 'times', times: ['08:00', '12:30', '20:00'] }
		});
		expect(create({ ...OK, kind: 'times', times: [' 08:00 '] }).input).toMatchObject({
			schedule: { kind: 'times', times: ['08:00'] }
		});
	});

	it('allows as many times as the API and refuses one more', () => {
		const some = (n: number) =>
			Array.from({ length: n }, (_, i) => `${String(i).padStart(2, '0')}:15`);
		expect(create({ ...OK, kind: 'times', times: some(MAX_SCHEDULE_TIMES) }).field).toBeNull();
		expect(create({ ...OK, kind: 'times', times: some(MAX_SCHEDULE_TIMES + 1) })).toMatchObject({
			input: null,
			field: 'schedule'
		});
	});

	it('refuses no time, a repeated time and anything that is not a strict HH:MM', () => {
		for (const times of [[], [''], ['', ' '], ['08:00', '08:00'], ['08:00', ' 08:00 ']]) {
			expect(create({ ...OK, kind: 'times', times })).toMatchObject({
				input: null,
				field: 'schedule'
			});
		}
		for (const bad of [
			'8:00',
			'08:0',
			'24:00',
			'12:60',
			'99:99',
			'12:00:00',
			'12.00',
			'12-00',
			'noon',
			'12:00pm',
			'12:00\n13:00',
			'-1:00',
			'+1:00',
			'0x:00',
			'１２:００',
			'١٢:٠٠',
			'<b>',
			"08:00'; drop",
			'08:00,20:00'
		]) {
			expect(create({ ...OK, kind: 'times', times: ['07:00', bad] })).toMatchObject({
				input: null,
				field: 'schedule'
			});
		}
	});

	it('sends the hours as a number inside the API range', () => {
		expect(create({ ...OK, kind: 'every', hours: '6' }).input).toMatchObject({
			schedule: { kind: 'every', hours: 6 }
		});
		expect(create({ ...OK, kind: 'every', hours: ' 006 ' }).input).toMatchObject({
			schedule: { kind: 'every', hours: 6 }
		});
		for (const hours of [MIN_EVERY_HOURS, MAX_EVERY_HOURS]) {
			expect(create({ ...OK, kind: 'every', hours: String(hours) }).field).toBeNull();
		}
	});

	it('refuses hours outside the range or not a whole number', () => {
		for (const hours of [
			String(MIN_EVERY_HOURS - 1),
			String(MAX_EVERY_HOURS + 1),
			'',
			' ',
			'-1',
			'+6',
			'1.5',
			'6.0',
			'6,5',
			'1e2',
			'0x10',
			'six',
			'６',
			'٦',
			'6 h',
			'6\n7',
			'1000',
			'9'.repeat(40),
			'Infinity',
			'NaN'
		]) {
			expect(create({ ...OK, kind: 'every', hours })).toMatchObject({
				input: null,
				field: 'schedule'
			});
		}
		expect(create({ ...OK, kind: 'every' }).field).toBe('schedule');
	});

	it('refuses a missing or unknown kind, and reads only the fields of the chosen kind', () => {
		for (const kind of ['', 'None', 'daily', 'times ', 'every\n', '<x>', '__proto__']) {
			expect(create({ ...OK, kind })).toMatchObject({ input: null, field: 'schedule' });
		}
		expect(create({ ...OK, kind: undefined }).field).toBe('schedule');
		const stray = { times: ['99:99'], hours: 'x' };
		expect(create({ ...OK, ...stray }).input).toMatchObject({ schedule: { kind: 'none' } });
		expect(create({ ...OK, kind: 'every', hours: '4', times: ['99:99'] }).input).toMatchObject({
			schedule: { kind: 'every', hours: 4 }
		});
		expect(create({ ...OK, kind: 'times', times: ['09:00'], hours: 'x' }).input).toMatchObject({
			schedule: { kind: 'times', times: ['09:00'] }
		});
	});

	it('reads the start on the kid clock and leaves a blank start to the API', () => {
		expect(create(OK).input).not.toHaveProperty('start_at');
		expect(create({ ...OK, start_at: '' }).input).not.toHaveProperty('start_at');
		expect(create({ ...OK, start_at: '  ' }).input).not.toHaveProperty('start_at');
		expect(create({ ...OK, start_at: '2026-11-22T10:30' }).input).toMatchObject({
			start_at: '2026-11-22T10:30:00Z'
		});
		expect(create({ ...OK, start_at: '2026-11-22T10:30' }, MADRID).input).toMatchObject({
			start_at: '2026-11-22T09:30:00Z'
		});
	});

	it('refuses a start that is not a date and time', () => {
		for (const start_at of [
			'garbage',
			'2026-02-30T10:00',
			'2026-11-22T25:00',
			'2026-11-22T10:60',
			'2026-11-22 10:00',
			'2026-11-22T10:00:30',
			'2026-11-22',
			'10:00',
			'2026-13-01T10:00',
			'+2026-11-22T10:00'
		]) {
			expect(create({ ...OK, start_at })).toMatchObject({ input: null, field: 'start_at' });
		}
	});

	it('sends the last day as a plain date, and nothing for a blank one', () => {
		expect(create(OK).input).not.toHaveProperty('end_on');
		expect(create({ ...OK, end_on: ' ' }).input).not.toHaveProperty('end_on');
		expect(create({ ...OK, end_on: ' 2026-12-01 ' }).input).toMatchObject({ end_on: '2026-12-01' });
	});

	it('refuses a last day that is not a real date', () => {
		for (const end_on of [
			'2026-02-30',
			'2026-13-01',
			'2026-1-1',
			'22/11/2026',
			'2026-11-22T10:00',
			'x',
			'2026-12-01Z',
			'+2026-12-01',
			'２０２６-12-01'
		]) {
			expect(create({ ...OK, end_on })).toMatchObject({ input: null, field: 'end_on' });
		}
	});

	it('limits the name and the dose to the API length in characters, not bytes', () => {
		const at = MEDICATION_TEXT_MAX_LEN;
		for (const field of ['name', 'dose'] as const) {
			for (const unit of ['a', 'é', '😀']) {
				expect(create({ ...OK, [field]: unit.repeat(at) }).field).toBeNull();
				expect(create({ ...OK, [field]: ` ${unit.repeat(at)} ` }).field).toBeNull();
				expect(create({ ...OK, [field]: unit.repeat(at + 1) })).toMatchObject({
					input: null,
					field
				});
			}
			for (const blank of ['', ' ', ' \t\n ']) {
				expect(create({ ...OK, [field]: blank })).toMatchObject({ input: null, field });
			}
		}
		expect(create({ ...OK, name: undefined }).field).toBe('name');
		expect(create({ ...OK, dose: undefined }).field).toBe('dose');
	});

	it('limits the note to the API length and sends nothing for a blank one', () => {
		expect(create({ ...OK, note: 'n'.repeat(NOTE_MAX_LEN) }).field).toBeNull();
		expect(create({ ...OK, note: 'n'.repeat(NOTE_MAX_LEN + 1) }).field).toBe('note');
		expect(create({ ...OK, note: '😀'.repeat(NOTE_MAX_LEN + 1) }).field).toBe('note');
		expect(create({ ...OK, note: '   ' }).input).not.toHaveProperty('note');
	});

	it('checks the fields in the API order: name, dose, schedule, start, end, active, note', () => {
		const all = {
			name: '',
			dose: '',
			kind: 'x',
			start_at: 'x',
			end_on: 'x',
			active: 'x',
			note: 'n'.repeat(NOTE_MAX_LEN + 1)
		};
		const order = ['name', 'dose', 'schedule', 'start_at', 'end_on', 'active', 'note'];
		const fix: Posted[] = [
			{ name: 'a' },
			{ dose: 'a' },
			{ kind: 'none' },
			{ start_at: '' },
			{ end_on: '' },
			{ active: 'on' }
		];
		let fields: Posted = all;
		for (const [i, name] of order.entries()) {
			expect(create(fields).field).toBe(name);
			fields = { ...fields, ...(fix[i] ?? {}) };
		}
	});

	it('reads nothing but its own fields', () => {
		const r = create({
			...OK,
			id: 'm-1',
			dose_count: '3',
			name_original: 'Other',
			schedule: '{"kind":"every"}',
			start_at_original: 'x'
		});
		expect(r.input).toEqual({ name: 'Calpol', dose: '2.5 ml', schedule: { kind: 'none' } });
	});
});

const COURSE: Medication = {
	id: 'm-1',
	visitId: null,
	name: 'Calpol',
	dose: '2.5 ml',
	schedule: { kind: 'times', times: ['08:00', '20:00'] },
	startAt: '2026-11-20T08:30:45Z',
	endOn: '2026-12-01',
	active: true,
	note: 'with food',
	lastDose: null,
	nextDue: null,
	doseCount: 12
};

// What the browser posts for the form courseEditValues loaded: the repeated times, the box ticked or absent.
const asPosted = (values: ReturnType<typeof courseEditValues>): Posted => {
	const { active, times, ...rest } = values;
	return { ...rest, times, active: active ? 'on' : undefined };
};
const loadedCourse = (c: Medication = COURSE, zone = 'UTC') => asPosted(courseEditValues(c, zone));
const edit2 = (change: Posted, drop: string[] = [], c: Medication = COURSE, zone = 'UTC') => {
	const fields = { ...loadedCourse(c, zone), ...change };
	for (const k of drop) delete fields[k];
	return readMedicationForm(posted(fields), { kind: 'edit', zone });
};

describe('courseCreateValues', () => {
	it('starts as needed, active and empty', () => {
		expect(courseCreateValues()).toEqual({
			name: '',
			dose: '',
			kind: 'none',
			times: [],
			hours: '',
			start_at: '',
			end_on: '',
			active: true,
			note: ''
		});
	});
});

describe('courseEditValues', () => {
	it('shows the start on the kid clock to the minute and keeps what was loaded', () => {
		expect(courseEditValues(COURSE, 'UTC')).toEqual({
			name: 'Calpol',
			dose: '2.5 ml',
			kind: 'times',
			times: ['08:00', '20:00'],
			hours: '',
			start_at: '2026-11-20T08:30',
			end_on: '2026-12-01',
			active: true,
			note: 'with food',
			name_original: 'Calpol',
			dose_original: '2.5 ml',
			schedule_original: 'times:08:00,20:00',
			start_at_original: '2026-11-20T08:30',
			end_on_original: '2026-12-01',
			active_original: 'true',
			note_original: 'with food'
		});
		expect(courseEditValues(COURSE, MADRID).start_at).toBe('2026-11-20T09:30');
	});

	it('shows each kind with its own fields', () => {
		const every = courseEditValues({ ...COURSE, schedule: { kind: 'every', hours: 6 } }, 'UTC');
		expect(every).toMatchObject({
			kind: 'every',
			hours: '6',
			times: [],
			schedule_original: 'every:6'
		});
		const none = courseEditValues({ ...COURSE, schedule: { kind: 'none' } }, 'UTC');
		expect(none).toMatchObject({ kind: 'none', hours: '', times: [], schedule_original: 'none' });
	});

	it('shows no end, no note and an inactive course as empty text and false', () => {
		expect(
			courseEditValues({ ...COURSE, endOn: null, note: null, active: false }, 'UTC')
		).toMatchObject({
			end_on: '',
			note: '',
			active: false,
			end_on_original: '',
			note_original: '',
			active_original: 'false'
		});
	});
});

describe('readMedicationForm: editing a course', () => {
	it('sends nothing for a form left as loaded, even with seconds in the stored start', () => {
		expect(edit2({})).toMatchObject({ input: {}, field: null });
		expect(edit2({}, [], COURSE, MADRID).input).toEqual({});
	});

	it('sends only the note when only the note changed, and null when it is cleared', () => {
		expect(edit2({ note: 'after lunch' }).input).toEqual({ note: 'after lunch' });
		expect(edit2({ note: '  after lunch ' }).input).toEqual({ note: 'after lunch' });
		expect(edit2({ note: '' }).input).toEqual({ note: null });
		expect(edit2({ note: ' with food ' }).input).toEqual({});
	});

	it('sends only the name or the dose that changed', () => {
		expect(edit2({ name: 'Ibuprofen' }).input).toEqual({ name: 'Ibuprofen' });
		expect(edit2({ dose: ' 5 ml ' }).input).toEqual({ dose: '5 ml' });
		expect(edit2({ name: ' Calpol ' }).input).toEqual({});
	});

	it('refuses a blank or long name and dose that changed', () => {
		for (const field of ['name', 'dose'] as const) {
			for (const bad of ['', '   ', 'x'.repeat(MEDICATION_TEXT_MAX_LEN + 1)]) {
				expect(edit2({ [field]: bad })).toMatchObject({ input: null, field });
			}
		}
	});

	it('sends the whole schedule when any of it changed', () => {
		expect(edit2({ times: ['08:00', '21:00'] }).input).toEqual({
			schedule: { kind: 'times', times: ['08:00', '21:00'] }
		});
		expect(edit2({ times: ['20:00', '07:00', ''] }).input).toEqual({
			schedule: { kind: 'times', times: ['07:00', '20:00'] }
		});
		expect(edit2({ kind: 'every', hours: '8' }).input).toEqual({
			schedule: { kind: 'every', hours: 8 }
		});
		expect(edit2({ kind: 'none' }).input).toEqual({ schedule: { kind: 'none' } });
		expect(edit2({ times: ['08:00', '20:00', '12:00'], note: 'x' }).input).toEqual({
			schedule: { kind: 'times', times: ['08:00', '12:00', '20:00'] },
			note: 'x'
		});
	});

	it('reads the same times in other slots, or other hours of an unused kind, as unchanged', () => {
		expect(edit2({ times: ['', '20:00', ' ', '08:00'] }).input).toEqual({});
		expect(edit2({ hours: 'junk' }).input).toEqual({});
		const every = { ...COURSE, schedule: { kind: 'every', hours: 6 } } as const;
		expect(edit2({ hours: ' 006 ', times: ['99:99'] }, [], every).input).toEqual({});
		expect(edit2({ hours: '7', times: ['99:99'] }, [], every).input).toEqual({
			schedule: { kind: 'every', hours: 7 }
		});
	});

	it('compares the schedule before checking its range, so a stored course stays editable', () => {
		const odd = { ...COURSE, schedule: { kind: 'every', hours: MAX_EVERY_HOURS + 32 } } as const;
		expect(edit2({ note: 'x' }, [], odd).input).toEqual({ note: 'x' });
		expect(edit2({ hours: String(MAX_EVERY_HOURS + 33) }, [], odd)).toMatchObject({
			input: null,
			field: 'schedule'
		});
	});

	it('refuses a changed schedule that is not valid', () => {
		expect(edit2({ times: [] })).toMatchObject({ input: null, field: 'schedule' });
		expect(edit2({ times: ['08:00', '25:00'] }).field).toBe('schedule');
		expect(edit2({ times: ['08:00', '08:00'] }).field).toBe('schedule');
		expect(edit2({ kind: 'every', hours: '0' }).field).toBe('schedule');
		expect(edit2({ kind: 'every', hours: '' }).field).toBe('schedule');
		expect(edit2({ kind: 'weekly' }).field).toBe('schedule');
		expect(edit2({ kind: undefined }).field).toBe('schedule');
	});

	it('sends the new start as an instant read on the kid clock, and keeps the stored one when blank', () => {
		expect(edit2({ start_at: '2026-11-21T10:15' }).input).toEqual({
			start_at: '2026-11-21T10:15:00Z'
		});
		expect(edit2({ start_at: '2026-11-21T10:15' }, [], COURSE, MADRID).input).toEqual({
			start_at: '2026-11-21T09:15:00Z'
		});
		expect(edit2({ start_at: '2026-11-20T08:30', note: 'x' }).input).toEqual({ note: 'x' });
		expect(edit2({ start_at: '', note: 'x' }).input).toEqual({ note: 'x' });
		expect(edit2({ start_at: '2026-02-30T10:00' })).toMatchObject({
			input: null,
			field: 'start_at'
		});
	});

	it('sends a new last day, and null when it is cleared', () => {
		expect(edit2({ end_on: '2026-12-15' }).input).toEqual({ end_on: '2026-12-15' });
		expect(edit2({ end_on: '' }).input).toEqual({ end_on: null });
		expect(edit2({ end_on: '  ' }).input).toEqual({ end_on: null });
		expect(edit2({ end_on: ' 2026-12-01 ' }).input).toEqual({});
		expect(edit2({ end_on: '2026-02-30' })).toMatchObject({ input: null, field: 'end_on' });
		const open = { ...COURSE, endOn: null };
		expect(edit2({}, [], open).input).toEqual({});
		expect(edit2({ end_on: '2026-12-15' }, [], open).input).toEqual({ end_on: '2026-12-15' });
	});

	it('sends the active flag only when it changed', () => {
		expect(edit2({ active: undefined }).input).toEqual({ active: false });
		const paused = { ...COURSE, active: false };
		expect(edit2({}, [], paused).input).toEqual({});
		expect(edit2({ active: 'on' }, [], paused).input).toEqual({ active: true });
		expect(edit2({ active: 'maybe' })).toMatchObject({ input: null, field: 'active' });
	});

	it('checks name, dose, schedule, start, end, active and note in that order', () => {
		const long = 'x'.repeat(NOTE_MAX_LEN + 1);
		const all = {
			name: '',
			dose: '',
			kind: 'x',
			start_at: 'x',
			end_on: 'x',
			active: 'x',
			note: long
		};
		const order = ['name', 'dose', 'schedule', 'start_at', 'end_on', 'active', 'note'];
		const fix: Posted[] = [
			{ name: 'a' },
			{ dose: 'a' },
			{ kind: 'none' },
			{ start_at: '' },
			{ end_on: '' },
			{ active: 'on' }
		];
		let fields: Posted = all;
		for (const [i, name] of order.entries()) {
			expect(edit2(fields).field).toBe(name);
			fields = { ...fields, ...(fix[i] ?? {}) };
		}
	});

	it('limits a changed note to the API length', () => {
		expect(edit2({ note: 'n'.repeat(NOTE_MAX_LEN) }).field).toBeNull();
		expect(edit2({ note: 'n'.repeat(NOTE_MAX_LEN + 1) }).field).toBe('note');
		expect(edit2({ note: '😀'.repeat(NOTE_MAX_LEN + 1) }).field).toBe('note');
	});

	it('fails closed on a missing original, in the order of the fields', () => {
		const missing: [string, string][] = [
			['name_original', 'name'],
			['dose_original', 'dose'],
			['schedule_original', 'schedule'],
			['start_at_original', 'start_at'],
			['end_on_original', 'end_on'],
			['active_original', 'active'],
			['note_original', 'note']
		];
		for (const [original, field] of missing) {
			expect(edit2({}, [original])).toMatchObject({ input: null, field });
		}
		expect(edit2({}, ['dose_original', 'note_original']).field).toBe('dose');
		// Not a guess from the typed value, even when it would equal the stored one.
		expect(edit2({ note: 'with food' }, ['note_original']).field).toBe('note');
		expect(edit2({ active: 'on' }, ['active_original']).field).toBe('active');
	});

	it('refuses an original of the box that is neither true nor false', () => {
		for (const active_original of ['', 'on', 'yes', 'TRUE']) {
			expect(edit2({ active_original })).toMatchObject({ input: null, field: 'active' });
		}
	});

	it('keeps the typed values and the loaded originals when it refuses', () => {
		const r = edit2({ name: 'Typed', kind: 'every', hours: '999', note: 'typed note' });
		expect(r.field).toBe('schedule');
		expect(r.values).toMatchObject({
			name: 'Typed',
			kind: 'every',
			hours: '999',
			note: 'typed note',
			name_original: 'Calpol',
			schedule_original: 'times:08:00,20:00',
			start_at: '2026-11-20T08:30',
			active: true
		});
	});

	it('never sends the id or the dose count', () => {
		expect(edit2({ id: 'm-9', dose_count: '0', note: 'x' }).input).toEqual({ note: 'x' });
	});
});

describe('course failures', () => {
	const values = courseEditValues(COURSE, 'UTC');
	const API_FIELDS = ['name', 'dose', 'schedule', 'start_at', 'end_on', 'active', 'note'];

	it('has a value-free message for every field the API names', () => {
		for (const f of API_FIELDS) expect(COURSE_FIELD_MESSAGES[f]).toEqual(expect.any(String));
		for (const message of Object.values(COURSE_FIELD_MESSAGES)) {
			expect(message).not.toMatch(/Calpol|2\.5|m-1|08:00|with food/);
		}
		expect(COURSE_FIELD_MESSAGES.name).toBe(
			`Enter a name of up to ${MEDICATION_TEXT_MAX_LEN} characters`
		);
		expect(COURSE_FIELD_MESSAGES.dose).toBe(
			`Enter a dose of up to ${MEDICATION_TEXT_MAX_LEN} characters`
		);
	});

	it('lands a refused field on its input', () => {
		expect(courseFieldFailure(values, 'name', { form: 'save' })).toMatchObject({
			status: 422,
			data: { form: 'save', values, field: 'name', message: COURSE_FIELD_MESSAGES.name }
		});
	});

	it('maps every field the API names, the course cap and an unreachable API', () => {
		for (const field of API_FIELDS) {
			expect(
				courseApiFailure(values, { ok: false, status: 422, reason: 'invalid', field }, {})
			).toMatchObject({
				status: 422,
				data: { values, field, message: COURSE_FIELD_MESSAGES[field] }
			});
		}
		expect(
			courseApiFailure(values, { ok: false, status: 409, reason: 'conflict', code: 'limit' }, {})
		).toMatchObject({ status: 409, data: { values, field: null, message: COURSE_LIMIT_MESSAGE } });
		expect(COURSE_LIMIT_MESSAGE).toBe('This child has the maximum number of medication courses');
		expect(
			courseApiFailure(values, { ok: false, status: 503, reason: 'unavailable' }, {})
		).toMatchObject({ status: 503, data: { values, field: null } });
		expect(courseApiFailure(values, { ok: false, status: 500, reason: 'error' }, {})).toMatchObject(
			{
				status: 500,
				data: { values, field: null }
			}
		);
	});

	it('answers an unknown field of the API like any other failure', () => {
		expect(
			courseApiFailure(values, { ok: false, status: 422, reason: 'invalid', field: 'zzz' }, {})
		).toMatchObject({ status: 500, data: { values, field: null } });
	});
});

describe('readMedicationForm with a prefix', () => {
	const behind = (fields: Posted) =>
		Object.fromEntries(Object.entries(fields).map(([k, v]) => [`v1.${k}`, v]));

	it('reads the names behind the prefix, the times, the box and originals included, and ignores the others', () => {
		const block = {
			...loadedCourse(),
			kind: 'times',
			times: ['09:00', '21:00'],
			active: undefined
		};
		const r = readMedicationForm(
			posted({ ...OK, ...behind(block) }),
			{ kind: 'edit', zone: 'UTC' },
			{ prefix: 'v1.' }
		);
		expect(r).toEqual(readMedicationForm(posted(block), { kind: 'edit', zone: 'UTC' }));
		expect(r.values).toMatchObject({ times: ['09:00', '21:00'], active: false });
		expect(r.input).toMatchObject({ schedule: { kind: 'times', times: ['09:00', '21:00'] } });
		expect(
			readMedicationForm(posted(behind(OK)), { kind: 'create', zone: 'UTC' }, { prefix: 'v1.' })
		).toEqual(create(OK));
	});
});
