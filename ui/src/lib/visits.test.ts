import { describe, expect, it } from 'vitest';
import type { VisitDetail } from './types';
import {
	hasOutcome,
	visitDueWhen,
	visitRecordDate,
	visitRecordLines,
	visitWhen,
	visitWhenLong
} from './visits';

describe('visit dates', () => {
	it('writes a row on the kid’s clock, not the server’s', () => {
		expect(visitWhen('2026-12-03T18:30:00Z', 'America/Los_Angeles')).toBe('Thu 3 Dec 2026, 10:30');
		expect(visitWhen('2026-12-03T23:30:00Z', 'Europe/Madrid')).toBe('Fri 4 Dec 2026, 00:30');
	});

	it('writes Home’s Due line without the year, on the kid’s clock', () => {
		expect(visitDueWhen('2026-12-03T18:30:00Z', 'America/Los_Angeles')).toBe('Thu 3 Dec, 10:30');
		expect(visitDueWhen('2026-12-31T23:30:00Z', 'Europe/Madrid')).toBe('Fri 1 Jan, 00:30');
	});

	it('writes the visit page’s line with the length', () => {
		expect(
			visitWhenLong({ startsAt: '2026-12-03T18:30:00Z', durationMin: 45 }, 'America/Los_Angeles')
		).toBe('Thursday 3 December 2026, 10:30 · 45 min');
	});

	it('writes a linked record’s date', () => {
		expect(visitRecordDate('2026-01-09')).toBe('9 Jan 2026');
	});
});

describe('hasOutcome', () => {
	const none = { outcomeNotes: null, linked: { growth: [], vaccinations: [], medications: [] } };
	it('is notes or any linked row', () => {
		expect(hasOutcome(none)).toBe(false);
		expect(hasOutcome({ ...none, outcomeNotes: 'ok' })).toBe(true);
		const linked: VisitDetail['linked'] = {
			...none.linked,
			medications: [{ id: 'm', name: 'x', doseCount: 0 }]
		};
		expect(hasOutcome({ ...none, linked })).toBe(true);
	});
});

describe('visitRecordLines', () => {
	const g = (n: number) =>
		Array.from({ length: n }, (_, i) => ({ id: `g${i}`, measuredOn: '2026-12-03' }));
	const v = (n: number) =>
		Array.from({ length: n }, (_, i) => ({
			id: `v${i}`,
			givenOn: '2026-12-03',
			antigens: ['measles']
		}));
	const m = (...doses: number[]) =>
		doses.map((doseCount, i) => ({ id: `m${i}`, name: 'Amoxicillin', doseCount }));

	it('names each module once, singular or plural, in the confirm’s order', () => {
		expect(visitRecordLines({ growth: g(1), vaccinations: v(2), medications: m(14) })).toEqual([
			'1 growth measurement',
			'2 vaccines',
			'1 medication, with its 14 logged doses'
		]);
		expect(visitRecordLines({ growth: g(2), vaccinations: v(1), medications: m(1, 0) })).toEqual([
			'2 growth measurements',
			'1 vaccine',
			'2 medications, with their 1 logged dose'
		]);
	});

	it('leaves out what is not linked, and the doses when there are none', () => {
		expect(visitRecordLines({ growth: [], vaccinations: [], medications: m(0) })).toEqual([
			'1 medication'
		]);
		expect(visitRecordLines({ growth: [], vaccinations: [], medications: [] })).toEqual([]);
	});
});
