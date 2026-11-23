import { describe, expect, it } from 'vitest';
import { KID_FIELD_MESSAGES, kidValues, readKidForm, scheduleOptions } from './kid-form';
import type { VaccineSchemaInfo } from '../types';

const form = (fields: Record<string, string>) => {
	const data = new FormData();
	for (const [k, v] of Object.entries(fields)) data.append(k, v);
	return data;
};
const kid = { name: 'Mia', birth_date: '2026-07-08', sex: 'female', time_zone: 'UTC' };

describe('readKidForm growth source', () => {
	it('sends the chosen source as typed; the API names an unknown one', () => {
		expect(readKidForm(form({ ...kid, growth_source: 'cdc' })).input).toMatchObject({
			growthSource: 'cdc'
		});
		expect(readKidForm(form({ ...kid, growth_source: 'x' })).input).toMatchObject({
			growthSource: 'x'
		});
	});

	it('sends no source for a form without one, so the stored one is kept', () => {
		const r = readKidForm(form(kid));
		expect(r.input).not.toHaveProperty('growthSource');
		expect(r.values.growthSource).toBe('');
	});
});

describe('readKidForm overdue minutes', () => {
	const typed = (text: string, original = '60') =>
		readKidForm(
			form({ ...kid, reminder_overdue_minutes: text, reminder_overdue_minutes_original: original })
		);

	it('sends the changed value, accepting both ends of the range', () => {
		expect(typed('15').input).toMatchObject({ reminderOverdueMinutes: 15 });
		expect(typed(' 5 ').input).toMatchObject({ reminderOverdueMinutes: 5 });
		expect(typed('1440').input).toMatchObject({ reminderOverdueMinutes: 1440 });
	});

	it('refuses blank, non-integer and out-of-range text on its field with the typed value kept', () => {
		for (const text of ['', '  ', '4', '1441', '0', '-15', '60.5', '1e2', '15 min']) {
			const r = typed(text);
			expect(r.input, text).toBeNull();
			expect(r.field, text).toBe('reminder_overdue_minutes');
			expect(r.values.reminderOverdue).toBe(text.trim());
			expect(r.values.reminderOverdueOriginal).toBe('60');
		}
		expect(KID_FIELD_MESSAGES.reminder_overdue_minutes).toBe('Enter minutes from 5 to 1440');
	});

	it('sends nothing when unchanged, even a stored value outside the range', () => {
		expect(typed('60').input).not.toHaveProperty('reminderOverdueMinutes');
		expect(typed('2000', '2000').input).not.toHaveProperty('reminderOverdueMinutes');
		expect(typed('2000', '2000').field).toBeNull();
	});

	it('sends nothing for a form without the field', () => {
		const r = readKidForm(form(kid));
		expect(r.input).not.toHaveProperty('reminderOverdueMinutes');
		expect(r.values.reminderOverdue).toBe('');
	});

	it('refuses a blank value that arrives without its original', () => {
		const r = readKidForm(form({ ...kid, reminder_overdue_minutes: '' }));
		expect(r.field).toBe('reminder_overdue_minutes');
		expect(r.input).toBeNull();
	});

	it('validates a value that arrives without its original', () => {
		expect(readKidForm(form({ ...kid, reminder_overdue_minutes: '15' })).input).toMatchObject({
			reminderOverdueMinutes: 15
		});
		expect(readKidForm(form({ ...kid, reminder_overdue_minutes: '4' })).field).toBe(
			'reminder_overdue_minutes'
		);
	});
});

describe('kidValues overdue minutes', () => {
	it('shows the stored minutes and remembers them as the original; a new kid has neither', () => {
		const stored = {
			id: 'k-1',
			name: 'Mia',
			birthDate: '2026-07-08',
			sex: 'female',
			gestationWeeks: null,
			gestationDays: null,
			timeZone: 'UTC',
			growthSource: 'who',
			vaccineSchema: null,
			reminderOverdueMinutes: 90,
			role: 'guardian'
		} as const;
		expect(kidValues(stored)).toMatchObject({
			reminderOverdue: '90',
			reminderOverdueOriginal: '90'
		});
		expect(kidValues()).toMatchObject({ reminderOverdue: '', reminderOverdueOriginal: '' });
	});
});

describe('kidValues growth source', () => {
	it("defaults a new kid to who and shows a stored kid's own source", () => {
		expect(kidValues().growthSource).toBe('who');
		const stored = {
			id: 'k-1',
			name: 'Mia',
			birthDate: '2026-07-08',
			sex: 'female',
			gestationWeeks: null,
			gestationDays: null,
			timeZone: 'UTC',
			growthSource: 'cdc',
			vaccineSchema: null,
			reminderOverdueMinutes: 60,
			role: 'guardian'
		} as const;
		expect(kidValues(stored).growthSource).toBe('cdc');
	});
});

describe('readKidForm vaccine schedule', () => {
	it('sends the chosen schedule, null for "none", and nothing for a form without the field', () => {
		expect(readKidForm(form({ ...kid, vaccine_schema: 'uk' })).input).toMatchObject({
			vaccineSchema: 'uk'
		});
		const none = readKidForm(form({ ...kid, vaccine_schema: '' }));
		expect(none.input).toHaveProperty('vaccineSchema', null);
		expect(none.values.vaccineSchema).toBe('');
		expect(readKidForm(form(kid)).input).not.toHaveProperty('vaccineSchema');
	});
});

describe('kidValues vaccine schedule', () => {
	it("shows no schedule for a new kid and a stored kid's own one", () => {
		expect(kidValues().vaccineSchema).toBe('');
		const stored = {
			id: 'k-1',
			name: 'Mia',
			birthDate: '2026-07-08',
			sex: 'female',
			gestationWeeks: null,
			gestationDays: null,
			timeZone: 'UTC',
			growthSource: 'who',
			vaccineSchema: 'us',
			reminderOverdueMinutes: 60,
			role: 'guardian'
		} as const;
		expect(kidValues(stored).vaccineSchema).toBe('us');
		expect(kidValues({ ...stored, vaccineSchema: null }).vaccineSchema).toBe('');
	});
});

describe('scheduleOptions', () => {
	const info = (id: string, name: string): VaccineSchemaInfo => ({
		id,
		name,
		authority: 'CDC',
		edition: '2025',
		retrieved: '2026-11-22',
		originUrl: 'https://example.test',
		licence: 'l',
		attribution: 'a'
	});
	it('offers exactly the API list in its order, named with authority and edition', () => {
		expect(scheduleOptions([info('us', 'United States'), info('uk', 'England')])).toEqual([
			{ id: 'us', label: 'United States — CDC, 2025' },
			{ id: 'uk', label: 'England — CDC, 2025' }
		]);
	});
});
