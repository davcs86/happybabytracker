import { describe, expect, it } from 'vitest';
import { kidValues, readKidForm } from './kid-form';

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
			role: 'guardian'
		} as const;
		expect(kidValues(stored).growthSource).toBe('cdc');
	});
});
