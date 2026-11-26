import { describe, expect, it } from 'vitest';
import {
	MAX_OUTCOME_MEDICATIONS,
	MAX_OUTCOME_VACCINATIONS,
	VACCINE_LOT_MAX_LEN,
	VISIT_OUTCOME_NOTES_MAX_LEN
} from '../constants';
import { COURSE_FIELD_MESSAGES } from './medication-form';
import { GROWTH_FIELD_MESSAGES, growthRangeMessage } from './growth-form';
import {
	OUTCOME_FIELD_MESSAGES,
	OUTCOME_LIMIT_MESSAGE,
	medicationPrefix,
	outcomeApiFailure,
	outcomeBlanks,
	outcomeCreateValues,
	outcomeCurrent,
	outcomeFieldMessage,
	outcomeStale,
	readOutcomeForm,
	vaccinePrefix,
	type OutcomeSections
} from './outcome-form';
import { VACCINE_FIELD_MESSAGES } from './vaccine-form';
import { VISIT_REFUSALS } from './visit-form';

const LA = 'America/Los_Angeles';
const ALL: OutcomeSections = { growth: true, vaccinations: true, medications: true };
const read = (fields: [string, string][], sections: OutcomeSections = ALL) => {
	const data = new FormData();
	for (const [k, v] of fields) data.append(k, v);
	return readOutcomeForm(data, { zone: LA, sections });
};

const growth: [string, string][] = [
	['growth.measured_on', '2026-11-20'],
	['growth.weight_unit', 'kg'],
	['growth.weight_kg', '6.4'],
	['growth.weight_lb', ''],
	['growth.weight_oz', ''],
	['growth.length_unit', 'cm'],
	['growth.length', ''],
	['growth.head_unit', 'cm'],
	['growth.head', ''],
	['growth.note', '']
];
const vaccine = (i: number, antigens: string[], lot = ''): [string, string][] => [
	[`${vaccinePrefix(i)}given_on`, '2026-11-20'],
	...antigens.map((a): [string, string] => [`${vaccinePrefix(i)}antigens`, a]),
	[`${vaccinePrefix(i)}lot`, lot],
	[`${vaccinePrefix(i)}product`, ''],
	[`${vaccinePrefix(i)}note`, '']
];
const course = (i: number, name = 'Amoxicillin'): [string, string][] => [
	[`${medicationPrefix(i)}name`, name],
	[`${medicationPrefix(i)}dose`, '5 ml'],
	[`${medicationPrefix(i)}kind`, 'none'],
	[`${medicationPrefix(i)}hours`, ''],
	[`${medicationPrefix(i)}start_at`, ''],
	[`${medicationPrefix(i)}end_on`, ''],
	[`${medicationPrefix(i)}active`, 'on'],
	[`${medicationPrefix(i)}note`, '']
];
const rev: [string, string][] = [
	['revision', '3'],
	['outcome_notes', ''],
	['outcome_notes_original', '']
];

describe('outcomeBlanks and outcomeCreateValues', () => {
	it('prefill the visit’s kid-local date and start, one block each, nothing ticked, the stored notes', () => {
		const blanks = outcomeBlanks({ startsAt: '2026-11-21T06:30:00Z' }, LA, {
			weight: 'lb',
			length: 'in'
		});
		expect(blanks.growth).toMatchObject({
			measured_on: '2026-11-20',
			weight_unit: 'lb',
			length_unit: 'in',
			head_unit: 'in'
		});
		expect(blanks.vaccination).toMatchObject({ given_on: '2026-11-20', antigens: [] });
		expect(blanks.medication).toMatchObject({ start_at: '2026-11-20T22:30', kind: 'none' });
		const values = outcomeCreateValues(blanks, { outcomeNotes: 'Healthy.' });
		expect(values).toMatchObject({
			outcome_notes: 'Healthy.',
			add: { growth: false, vaccinations: false, medications: false }
		});
		expect(values.vaccinations).toEqual([blanks.vaccination]);
		expect(values.medications).toEqual([blanks.medication]);
		expect(outcomeCreateValues(blanks, { outcomeNotes: null }).outcome_notes).toBe('');
	});
});

describe('readOutcomeForm', () => {
	it('composes the three readers behind their prefixes, only for ticked sections', () => {
		const r = read([
			['revision', '3'],
			['outcome_notes', ' Healthy. '],
			['outcome_notes_original', ''],
			['add_growth', 'on'],
			...growth,
			['add_vaccinations', 'on'],
			...vaccine(0, ['diphtheria', 'hib']),
			...vaccine(1, ['measles'], 'AB12'),
			['add_medications', 'on'],
			...course(0)
		]);
		expect(r.field).toBeNull();
		expect(r.input).toEqual({
			revision: 3,
			outcome_notes: 'Healthy.',
			growth: { measured_on: '2026-11-20', weight: { unit: 'kg', amount: 6400 }, note: null },
			vaccinations: [
				{ given_on: '2026-11-20', antigens: ['diphtheria', 'hib'] },
				{ given_on: '2026-11-20', antigens: ['measles'], lot: 'AB12' }
			],
			medications: [{ name: 'Amoxicillin', dose: '5 ml', schedule: { kind: 'none' } }]
		});
		expect(r.revision).toBe(3);
		expect(r.values.add).toEqual(ALL);
		expect(r.values.vaccinations.map((v) => v.antigens)).toEqual([
			['diphtheria', 'hib'],
			['measles']
		]);
	});

	it('a block of an unticked section is kept as typed but never sent or checked', () => {
		const r = read([
			...rev,
			['add_growth', 'on'],
			...growth,
			...vaccine(0, [], 'x'.repeat(VACCINE_LOT_MAX_LEN + 1)),
			...course(0, '')
		]);
		expect(r.field).toBeNull();
		expect(Object.keys(r.input!)).toEqual(['revision', 'growth']);
		expect(r.values.vaccinations[0].lot).toHaveLength(VACCINE_LOT_MAX_LEN + 1);
		expect(r.values.medications[0].name).toBe('');
	});

	it('a section whose module is off is not read, even when its box is posted', () => {
		const r = read([...rev, ['add_vaccinations', 'on'], ...vaccine(0, ['measles'])], {
			...ALL,
			vaccinations: false
		});
		expect(r.values.add.vaccinations).toBe(false);
		expect(r.field).toBe('outcome');
		expect(r.message).toBe(OUTCOME_FIELD_MESSAGES.outcome);
	});

	it('sends the notes only when they differ from the ones shown, and never blank', () => {
		const shown: [string, string][] = [
			['revision', '3'],
			['outcome_notes_original', 'Healthy.'],
			['add_growth', 'on'],
			...growth
		];
		expect(read([...shown, ['outcome_notes', 'Healthy. ']]).input).not.toHaveProperty(
			'outcome_notes'
		);
		expect(read([...shown, ['outcome_notes', '  ']]).input).not.toHaveProperty('outcome_notes');
		expect(read([...shown, ['outcome_notes', 'Next check at 6 months.']]).input).toMatchObject({
			outcome_notes: 'Next check at 6 months.'
		});
		const notesOnly = read([
			['revision', '0'],
			['outcome_notes', 'Healthy.']
		]);
		expect(notesOnly.input).toEqual({ revision: 0, outcome_notes: 'Healthy.' });
	});

	it('refuses notes over the limit, and a save with nothing to record', () => {
		const long = read([
			['revision', '3'],
			['outcome_notes', 'x'.repeat(VISIT_OUTCOME_NOTES_MAX_LEN + 1)]
		]);
		expect(long).toMatchObject({
			input: null,
			field: 'outcome_notes',
			message: OUTCOME_FIELD_MESSAGES.outcome_notes
		});
		expect(read([...rev, ...growth]).field).toBe('outcome');
	});

	it('fails closed on a missing or malformed revision', () => {
		expect(read([['outcome_notes', 'Healthy.']]).field).toBe('revision');
		expect(
			read([
				['revision', '3a'],
				['outcome_notes', 'Healthy.']
			]).field
		).toBe('revision');
	});

	it('maps a refused field to its block, named in it', () => {
		const lot = read([
			...rev,
			['add_vaccinations', 'on'],
			...vaccine(0, ['measles']),
			...vaccine(1, ['measles'], 'x'.repeat(VACCINE_LOT_MAX_LEN + 1))
		]);
		expect(lot).toMatchObject({
			input: null,
			field: 'vaccinations[1].lot',
			message: `Vaccine 2: ${VACCINE_FIELD_MESSAGES.lot}`
		});
		const name = read([...rev, ['add_medications', 'on'], ...course(0), ...course(1, '')]);
		expect(name).toMatchObject({
			field: 'medications[1].name',
			message: `Medication 2: ${COURSE_FIELD_MESSAGES.name}`
		});
		const none = read([
			...rev,
			['add_growth', 'on'],
			...growth.map(([k, v]): [string, string] => [k, k === 'growth.weight_kg' ? '' : v])
		]);
		expect(none).toMatchObject({
			field: 'growth.measurements',
			message: GROWTH_FIELD_MESSAGES.measurements
		});
		const heavy = read([
			...rev,
			['add_growth', 'on'],
			...growth.map(([k, v]): [string, string] => [k, k === 'growth.weight_kg' ? '300' : v])
		]);
		expect(heavy).toMatchObject({ field: 'growth.weight', message: growthRangeMessage('weight') });
	});

	it('reads blocks from 0 with no gap, up to the caps', () => {
		const many = read([
			...rev,
			['add_vaccinations', 'on'],
			...Array.from({ length: MAX_OUTCOME_VACCINATIONS + 2 }, (_, i) =>
				vaccine(i, ['measles'])
			).flat(),
			['add_medications', 'on'],
			...Array.from({ length: MAX_OUTCOME_MEDICATIONS + 1 }, (_, i) => course(i)).flat()
		]);
		expect(many.input!.vaccinations).toHaveLength(MAX_OUTCOME_VACCINATIONS);
		expect(many.input!.medications).toHaveLength(MAX_OUTCOME_MEDICATIONS);
		expect(many.values.vaccinations).toHaveLength(MAX_OUTCOME_VACCINATIONS);

		const gap = read([
			...rev,
			['add_vaccinations', 'on'],
			...vaccine(0, ['measles']),
			...vaccine(2, ['hib'])
		]);
		expect(gap.input!.vaccinations).toEqual([{ given_on: '2026-11-20', antigens: ['measles'] }]);
	});

	it('reads a course start on the kid’s clock', () => {
		const r = read([
			...rev,
			['add_medications', 'on'],
			...course(0).map(([k, v]): [string, string] => [
				k,
				k.endsWith('start_at') ? '2026-11-20T22:30' : v
			])
		]);
		expect(r.input!.medications).toEqual([
			{
				name: 'Amoxicillin',
				dose: '5 ml',
				schedule: { kind: 'none' },
				start_at: '2026-11-21T06:30:00Z'
			}
		]);
	});
});

describe('outcomeFieldMessage', () => {
	it('names the block of every API field path a form field shows, and nothing else', () => {
		expect(outcomeFieldMessage('vaccinations[0].given_on')).toBe(
			`Vaccine 1: ${VACCINE_FIELD_MESSAGES.given_on}`
		);
		expect(outcomeFieldMessage('medications[4].schedule')).toBe(
			`Medication 5: ${COURSE_FIELD_MESSAGES.schedule}`
		);
		expect(outcomeFieldMessage('growth.note')).toBe(GROWTH_FIELD_MESSAGES.note);
		expect(outcomeFieldMessage('growth.head')).toBe(GROWTH_FIELD_MESSAGES.head);
		expect(outcomeFieldMessage('growth.head', { api: true })).toBe(growthRangeMessage('head'));
		expect(outcomeFieldMessage('outcome_notes')).toBe(OUTCOME_FIELD_MESSAGES.outcome_notes);
		for (const path of [
			'revision',
			'vaccinations',
			'growth',
			'vaccinations[0].unknown',
			'medications.0.name'
		]) {
			expect(outcomeFieldMessage(path)).toBeNull();
		}
	});
});

describe('outcomeApiFailure', () => {
	const values = outcomeCreateValues(
		outcomeBlanks({ startsAt: '2026-11-21T06:30:00Z' }, LA, { weight: 'kg', length: 'cm' }),
		{ outcomeNotes: null }
	);
	const current = outcomeCurrent({ revision: 4, outcomeNotes: 'Healthy.' });
	const extra = { form: 'save' as const };

	it('puts a refused field in its block', () => {
		const r = outcomeApiFailure(
			values,
			{ ok: false, status: 422, reason: 'invalid', field: 'vaccinations[1].lot' },
			current,
			extra
		);
		expect(r.status).toBe(422);
		expect(r.data).toMatchObject({
			form: 'save',
			values,
			field: 'vaccinations[1].lot',
			message: `Vaccine 2: ${VACCINE_FIELD_MESSAGES.lot}`,
			current: null
		});
		const weight = outcomeApiFailure(
			values,
			{ ok: false, status: 422, reason: 'invalid', field: 'growth.weight' },
			current,
			extra
		);
		expect(weight.data).toMatchObject({ message: growthRangeMessage('weight') });
	});

	it('re-renders a stale save at the current revision with what was typed', () => {
		const r = outcomeApiFailure(
			values,
			{ ok: false, status: 409, reason: 'conflict', code: 'stale' },
			current,
			extra
		);
		expect(r.status).toBe(409);
		expect(r.data).toMatchObject({
			values,
			field: null,
			message: VISIT_REFUSALS.stale,
			current: { revision: 4, notes: 'Healthy.' }
		});
		expect(outcomeStale(values, current, extra).data).toEqual(r.data);
		const revision = outcomeApiFailure(
			values,
			{ ok: false, status: 422, reason: 'invalid', field: 'revision' },
			current,
			extra
		);
		expect(revision.data).toEqual(r.data);
	});

	it('says not_past and limit on the form', () => {
		const notPast = outcomeApiFailure(
			values,
			{ ok: false, status: 409, reason: 'conflict', code: 'not_past' },
			current,
			extra
		);
		expect(notPast.data).toMatchObject({
			message: VISIT_REFUSALS.not_past,
			field: null,
			current: null
		});
		const limit = outcomeApiFailure(
			values,
			{ ok: false, status: 409, reason: 'conflict', code: 'limit' },
			current,
			extra
		);
		expect(limit.data).toMatchObject({ message: OUTCOME_LIMIT_MESSAGE, field: null });
	});

	it('leaves the rest to failAction', () => {
		const down = outcomeApiFailure(
			values,
			{ ok: false, status: 0, reason: 'unavailable' },
			current,
			extra
		);
		expect(down.status).toBe(503);
		expect(() =>
			outcomeApiFailure(values, { ok: false, status: 404, reason: 'not_found' }, current, extra)
		).toThrow();
	});
});
