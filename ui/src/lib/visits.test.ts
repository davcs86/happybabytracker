import { describe, expect, it } from 'vitest';
import type { VisitDetail } from './types';
import { hasOutcome, visitRecordDate, visitRowName, visitWhen, visitWhenLong } from './visits';

describe('visit dates', () => {
	it('writes a row on the kid’s clock, not the server’s', () => {
		expect(visitWhen('2026-12-03T18:30:00Z', 'America/Los_Angeles')).toBe('Thu 3 Dec 2026, 10:30');
		expect(visitWhen('2026-12-03T23:30:00Z', 'Europe/Madrid')).toBe('Fri 4 Dec 2026, 00:30');
	});

	it('writes the visit page’s line with the length', () => {
		expect(
			visitWhenLong({ startsAt: '2026-12-03T18:30:00Z', durationMin: 45 }, 'America/Los_Angeles')
		).toBe('Thursday 3 December 2026, 10:30 · 45 min');
	});

	it('names a row by its kind and kid-local date', () => {
		expect(visitRowName({ kind: 'well_child', startsAt: '2026-12-04T06:00:00Z' }, 'UTC')).toBe(
			'Well-child visit, 4 Dec 2026'
		);
		expect(
			visitRowName({ kind: 'dental', startsAt: '2026-12-04T06:00:00Z' }, 'America/Los_Angeles')
		).toBe('Dental visit, 3 Dec 2026');
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
