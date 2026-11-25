import { describe, expect, it } from 'vitest';
import {
	NOTE_MAX_LEN,
	VISIT_DURATION_MIN,
	VISIT_KINDS,
	VISIT_LOCATION_MAX_LEN,
	VISIT_PROVIDER_MAX_LEN
} from '../constants';
import type { Visit } from '../types';
import {
	VISIT_FIELD_MESSAGES,
	VISIT_REFUSALS,
	readVisitForm,
	visitApiFailure,
	visitCreateValues,
	visitEditValues,
	visitFieldFailure,
	visitRefusal
} from './visit-form';

const form = (fields: Record<string, string>) => {
	const data = new FormData();
	for (const [k, v] of Object.entries(fields)) data.append(k, v);
	return data;
};

const LA = 'America/Los_Angeles';
const CREATE = { kind: 'create', zone: LA } as const;
const EDIT = { kind: 'edit', zone: LA } as const;
const create = (fields: Record<string, string>) => readVisitForm(form(fields), CREATE);
const filled = {
	starts_at: '2026-12-03T10:30',
	duration_min: '30',
	kind: 'well_child',
	provider: '',
	location: '',
	note: ''
};

const visit: Visit = {
	id: 'v-1',
	startsAt: '2026-12-03T18:30:00Z',
	durationMin: 45,
	kind: 'dental',
	status: 'upcoming',
	cancelledAt: null,
	provider: 'Dr. Lee',
	location: null,
	note: '4-month check',
	outcomeNotes: null,
	revision: 2
};
const stored = visitEditValues(visit, LA);
const edit = (changes: Record<string, string>, values: Record<string, string> = stored) =>
	readVisitForm(form({ ...values, ...changes }), EDIT);

describe('visitCreateValues', () => {
	it('starts at the kid-local minute, with the default length and no kind', () => {
		expect(visitCreateValues(new Date('2026-12-03T18:30:42Z'), LA)).toEqual({
			starts_at: '2026-12-03T10:30',
			duration_min: '30',
			kind: '',
			provider: '',
			location: '',
			note: ''
		});
	});
});

describe('readVisitForm: create', () => {
	it('sends the instant in the kid’s zone, the length, the kind and no blank text', () => {
		const r = create(filled);
		expect(r).toMatchObject({ field: null });
		expect(r.input).toEqual({
			starts_at: '2026-12-03T18:30:00Z',
			duration_min: 30,
			kind: 'well_child'
		});
	});

	it('sends trimmed texts and keeps what was typed in the values', () => {
		const r = create({ ...filled, provider: ' Dr. Lee ', location: '12 Main St', note: ' Fever ' });
		expect(r.input).toMatchObject({ provider: 'Dr. Lee', location: '12 Main St', note: 'Fever' });
		expect(r.values.provider).toBe(' Dr. Lee ');
	});

	it('reads whitespace alone as no text', () => {
		expect(create({ ...filled, provider: '  ', note: '\t' }).input).not.toHaveProperty('provider');
		expect(create({ ...filled, note: '\t' }).input).not.toHaveProperty('note');
	});

	it('refuses the fields in the API’s order', () => {
		const bad = {
			starts_at: '',
			duration_min: '4',
			kind: 'checkup',
			provider: 'x'.repeat(VISIT_PROVIDER_MAX_LEN + 1),
			location: 'x'.repeat(VISIT_LOCATION_MAX_LEN + 1),
			note: 'x'.repeat(NOTE_MAX_LEN + 1)
		};
		const order = ['starts_at', 'duration_min', 'kind', 'provider', 'location', 'note'];
		order.forEach((field, i) => {
			const fields = {
				...filled,
				...Object.fromEntries(order.slice(i).map((f) => [f, bad[f as keyof typeof bad]]))
			};
			expect(create(fields)).toMatchObject({ input: null, field });
		});
	});

	it('refuses a missing or impossible time', () => {
		for (const starts_at of ['', '2026-02-30T10:00', '2026-12-03', '10:30', '2026-12-03T24:00']) {
			expect(create({ ...filled, starts_at })).toMatchObject({ field: 'starts_at' });
		}
	});

	it('takes a whole number of minutes in range only', () => {
		for (const ok of [String(VISIT_DURATION_MIN.min), String(VISIT_DURATION_MIN.max), ' 45 ']) {
			expect(create({ ...filled, duration_min: ok }).field).toBeNull();
		}
		for (const duration_min of [
			'',
			String(VISIT_DURATION_MIN.min - 1),
			String(VISIT_DURATION_MIN.max + 1),
			'30.5',
			'-30',
			'1e2',
			'thirty'
		]) {
			expect(create({ ...filled, duration_min })).toMatchObject({ field: 'duration_min' });
		}
	});

	it('takes every kind of the API and nothing else', () => {
		for (const kind of VISIT_KINDS)
			expect(create({ ...filled, kind }).input).toMatchObject({ kind });
		for (const kind of ['', 'Dental', 'well-child']) {
			expect(create({ ...filled, kind })).toMatchObject({ field: 'kind' });
		}
	});

	it('limits the texts to the API length in characters, not bytes', () => {
		expect(create({ ...filled, provider: 'é'.repeat(VISIT_PROVIDER_MAX_LEN) }).field).toBeNull();
		expect(create({ ...filled, location: '💉'.repeat(VISIT_LOCATION_MAX_LEN) }).field).toBeNull();
		expect(create({ ...filled, note: 'é'.repeat(NOTE_MAX_LEN) }).field).toBeNull();
	});

	it('ignores posted originals', () => {
		const r = create({ ...filled, kind_original: 'dental' });
		expect(r.values).not.toHaveProperty('kind_original');
		expect(r.input).toMatchObject({ kind: 'well_child' });
	});
});

describe('visitEditValues', () => {
	it('shows the visit on the kid’s clock, with every original', () => {
		expect(stored).toEqual({
			starts_at: '2026-12-03T10:30',
			duration_min: '45',
			kind: 'dental',
			provider: 'Dr. Lee',
			location: '',
			note: '4-month check',
			starts_at_original: '2026-12-03T10:30',
			duration_min_original: '45',
			kind_original: 'dental',
			provider_original: 'Dr. Lee',
			location_original: '',
			note_original: '4-month check'
		});
	});
});

describe('readVisitForm: edit', () => {
	it('sends nothing when nothing changed, even with spaces around', () => {
		expect(edit({})).toEqual({ values: expect.anything(), input: {}, field: null });
		expect(edit({ provider: ' Dr. Lee ' }).input).toEqual({});
	});

	it('sends only what changed', () => {
		expect(edit({ starts_at: '2026-12-04T09:00', kind: 'sick' }).input).toEqual({
			starts_at: '2026-12-04T17:00:00Z',
			kind: 'sick'
		});
		expect(edit({ duration_min: '60', location: 'Clinic' }).input).toEqual({
			duration_min: 60,
			location: 'Clinic'
		});
	});

	it('clears an emptied text with null', () => {
		expect(edit({ provider: '', note: '  ' }).input).toEqual({ provider: null, note: null });
	});

	it('checks only a changed field', () => {
		const old = { ...stored, duration_min: '3', duration_min_original: '3' };
		expect(edit({}, old).input).toEqual({});
		expect(edit({ duration_min: '4' }, old)).toMatchObject({ field: 'duration_min' });
		expect(edit({ starts_at: '' })).toMatchObject({ field: 'starts_at' });
		expect(edit({ kind: 'x' })).toMatchObject({ field: 'kind' });
	});

	it('fails closed on a missing original', () => {
		for (const f of ['starts_at', 'duration_min', 'kind', 'provider', 'location', 'note']) {
			const values = Object.fromEntries(
				Object.entries(stored).filter(([k]) => k !== `${f}_original`)
			) as Record<string, string>;
			expect(edit({}, values)).toMatchObject({ input: null, field: f });
		}
	});
});

describe('failures', () => {
	const values = visitCreateValues(new Date('2026-12-03T18:30:00Z'), LA);

	it('puts a refused field on its input', () => {
		expect(visitFieldFailure(values, 'kind', { form: 'save' })).toMatchObject({
			status: 422,
			data: { form: 'save', values, field: 'kind', message: VISIT_FIELD_MESSAGES.kind }
		});
		for (const field of Object.keys(VISIT_FIELD_MESSAGES)) {
			expect(
				visitApiFailure(values, { ok: false, status: 422, reason: 'invalid', field }, {})
			).toMatchObject({
				status: 422,
				data: { values, field, message: VISIT_FIELD_MESSAGES[field] }
			});
		}
	});

	it('says the page spec’s sentences for the visit rules and the cap', () => {
		expect(VISIT_REFUSALS).toEqual({
			has_outcome:
				'This visit has notes or records from the visit. Remove them first, or delete the visit.',
			not_past: 'This visit hasn’t happened yet.',
			stale: 'Someone changed this visit on another phone. It’s shown now; your entries are kept.',
			limit: 'This child has the most visits the app keeps (2,000).'
		});
		for (const code of ['has_outcome', 'not_past', 'limit']) {
			expect(
				visitApiFailure(values, { ok: false, status: 409, reason: 'conflict', code }, {})
			).toMatchObject({
				status: 409,
				data: { values, field: null, message: VISIT_REFUSALS[code] }
			});
		}
		expect(
			visitRefusal(
				{ ok: false, status: 409, reason: 'conflict', code: 'has_outcome' },
				{
					form: 'status'
				}
			)
		).toMatchObject({ status: 409, data: { form: 'status', message: VISIT_REFUSALS.has_outcome } });
	});

	it('answers anything else as a page failure', () => {
		expect(
			visitApiFailure(values, { ok: false, status: 422, reason: 'invalid', field: 'zzz' }, {})
		).toMatchObject({ status: 500, data: { values, field: null } });
		expect(
			visitApiFailure(values, { ok: false, status: 409, reason: 'conflict', code: 'zzz' }, {})
		).toMatchObject({ status: 500 });
		expect(
			visitApiFailure(values, { ok: false, status: 503, reason: 'unavailable' }, {})
		).toMatchObject({ status: 503, data: { values, field: null } });
	});

	it('never puts a typed value in a message', () => {
		const typed = 'Dr. Secret';
		const r = create({ ...filled, provider: typed.repeat(VISIT_PROVIDER_MAX_LEN) });
		expect(r.field).toBe('provider');
		expect(VISIT_FIELD_MESSAGES.provider).not.toContain(typed);
	});
});
