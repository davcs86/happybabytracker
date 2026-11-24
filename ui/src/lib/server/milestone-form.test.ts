import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { MILESTONE_FIELD_MESSAGES, readMilestoneForm } from './milestone-form';

const CREATE = { kind: 'create' } as const;
const EDIT_ITEM = { kind: 'edit', custom: false } as const;
const EDIT_CUSTOM = { kind: 'edit', custom: true } as const;
const UTC = 'UTC';
// 2026-11-24 12:00 UTC is already 25 November in Auckland.
const NOW = Date.UTC(2026, 10, 24, 12);
const form = (fields: Record<string, string>) => {
	const data = new FormData();
	for (const [k, v] of Object.entries(fields)) data.append(k, v);
	return data;
};
const create = (fields: Record<string, string>, zone = UTC) =>
	readMilestoneForm(form(fields), CREATE, zone);

const STORED = {
	reached_on: '2026-11-01',
	reached_on_original: '2026-11-01',
	note: 'first time',
	note_original: 'first time'
};
const CUSTOM = { ...STORED, text: 'Waves bye', text_original: 'Waves bye' };

beforeEach(() => {
	vi.useFakeTimers({ now: NOW, toFake: ['Date'] });
});
afterEach(() => {
	vi.useRealTimers();
});

describe('readMilestoneForm create', () => {
	it('marks an item reached today in the kid zone when no date is given', () => {
		expect(create({ item: '9m-se-stub-1' })).toEqual({
			values: { item: '9m-se-stub-1', reached_on: '', note: '' },
			input: { item: '9m-se-stub-1', reached_on: '2026-11-24' },
			field: null,
			message: null
		});
		expect(create({ item: '9m-se-stub-1', reached_on: '  ' }, 'Pacific/Auckland').input).toEqual({
			item: '9m-se-stub-1',
			reached_on: '2026-11-25'
		});
	});

	it('sends a chosen date, a trimmed note and a trimmed custom text', () => {
		expect(
			create({ text: '  Says “mama”  ', reached_on: ' 2026-03-04 ', note: ' at the park ' }).input
		).toEqual({ text: 'Says “mama”', reached_on: '2026-03-04', note: 'at the park' });
	});

	it('accepts a custom text of exactly the maximum, counted in characters', () => {
		const text = '😀'.repeat(200);
		expect(create({ text }).input).toMatchObject({ text });
	});

	it.each([
		['blank', '   '],
		['empty', ''],
		['over the maximum', 'a'.repeat(201)],
		['over the maximum in characters', 'é'.repeat(201)]
	])('refuses a %s text with field text', (_name, text) => {
		expect(create({ text })).toMatchObject({
			input: null,
			field: 'text',
			message: MILESTONE_FIELD_MESSAGES.text
		});
	});

	it('refuses a form with neither or both of item and text, as the API does', () => {
		expect(create({ reached_on: '2026-11-01' }).field).toBe('item');
		expect(create({ item: '9m-se-stub-1', text: 'x' }).field).toBe('item');
	});

	it.each(['', '9m-xx-stub', '../x', '9m-se-STUB', '123m-se-a', '9m-se-a b', '9m-se-a?x=1'])(
		'refuses a hostile item %j',
		(item) => {
			expect(create({ item })).toMatchObject({ input: null, field: 'item' });
		}
	);

	it('refuses a date after the kid-local today', () => {
		expect(create({ item: '9m-se-stub-1', reached_on: '2026-11-25' })).toMatchObject({
			input: null,
			field: 'reached_on',
			message: MILESTONE_FIELD_MESSAGES.reached_on
		});
		expect(
			create({ item: '9m-se-stub-1', reached_on: '2026-11-25' }, 'Pacific/Auckland').input
		).toMatchObject({ reached_on: '2026-11-25' });
	});

	it.each([
		'2026-02-30',
		'2026-13-01',
		'2026-1-04',
		'04/03/2026',
		'2026-03-04T00:00',
		'+02026-03-04',
		'2026-03-04 x',
		'٢٠٢٦-٠٣-٠٤',
		'yesterday'
	])('refuses hostile date text %j', (reached_on) => {
		expect(create({ item: '9m-se-stub-1', reached_on })).toMatchObject({
			input: null,
			field: 'reached_on'
		});
	});

	it('refuses a note over the maximum', () => {
		expect(create({ item: '9m-se-stub-1', note: 'n'.repeat(501) }).field).toBe('note');
	});

	it('checks the fields in the API order: item or text, reached_on, note', () => {
		expect(create({ text: ' ', reached_on: 'x', note: 'n'.repeat(501) }).field).toBe('text');
		expect(create({ text: 'ok', reached_on: 'x', note: 'n'.repeat(501) }).field).toBe('reached_on');
	});

	it('never puts a typed value in a message', () => {
		const typed = 'Secret 2099-01-01 <script>';
		const cases: Record<string, string>[] = [
			{ text: typed.repeat(20) },
			{ item: typed },
			{ item: '9m-se-stub-1', reached_on: typed },
			{ item: '9m-se-stub-1', reached_on: '2099-01-01' },
			{ item: '9m-se-stub-1', note: typed.repeat(30) }
		];
		for (const fields of cases) {
			const r = create(fields);
			expect(r.field).not.toBeNull();
			expect(r.message).toBe(MILESTONE_FIELD_MESSAGES[r.field as string]);
			expect(r.message).not.toMatch(/Secret|2099|script/);
		}
	});
});

describe('readMilestoneForm edit', () => {
	it('sends nothing when nothing changed', () => {
		expect(readMilestoneForm(form(STORED), EDIT_ITEM, UTC)).toMatchObject({
			input: {},
			field: null
		});
		expect(readMilestoneForm(form(CUSTOM), EDIT_CUSTOM, UTC).input).toEqual({});
	});

	it('sends only the changed fields; a cleared note is null', () => {
		expect(
			readMilestoneForm(form({ ...STORED, reached_on: '2026-10-30' }), EDIT_ITEM, UTC).input
		).toEqual({ reached_on: '2026-10-30' });
		expect(readMilestoneForm(form({ ...STORED, note: '  ' }), EDIT_ITEM, UTC).input).toEqual({
			note: null
		});
		expect(
			readMilestoneForm(form({ ...CUSTOM, text: ' Waves hello ' }), EDIT_CUSTOM, UTC).input
		).toEqual({ text: 'Waves hello' });
	});

	it('never sends a text or an item for a catalog row', () => {
		const r = readMilestoneForm(
			form({ ...STORED, text: 'changed', item: '2m-se-stub-1' }),
			EDIT_ITEM,
			UTC
		);
		expect(r.input).toEqual({});
	});

	it('refuses a changed date that is blank, malformed or after today', () => {
		for (const reached_on of ['', '2026-11-31', '2026-11-25']) {
			expect(readMilestoneForm(form({ ...STORED, reached_on }), EDIT_ITEM, UTC).field).toBe(
				'reached_on'
			);
		}
	});

	it('refuses a changed custom text that is blank or too long', () => {
		for (const text of [' ', 'a'.repeat(201)]) {
			expect(readMilestoneForm(form({ ...CUSTOM, text }), EDIT_CUSTOM, UTC).field).toBe('text');
		}
	});

	it.each([
		['reached_on', EDIT_ITEM, STORED],
		['note', EDIT_ITEM, STORED],
		['text', EDIT_CUSTOM, CUSTOM],
		['reached_on', EDIT_CUSTOM, CUSTOM],
		['note', EDIT_CUSTOM, CUSTOM]
	] as const)('fails closed without %s_original', (field, mode, fields) => {
		const rest: Record<string, string> = { ...fields };
		delete rest[`${field}_original`];
		expect(readMilestoneForm(form(rest), mode, UTC)).toMatchObject({ input: null, field });
	});
});
