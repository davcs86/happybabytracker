import { describe, expect, it } from 'vitest';
import { acceptsZone, todayInOrUtc, withZone } from './time-zone';

describe('acceptsZone', () => {
	it('accepts current IANA names and UTC', () => {
		for (const z of ['Asia/Kolkata', 'Europe/Kyiv', 'Asia/Ho_Chi_Minh', 'UTC']) {
			expect(acceptsZone(z)).toBe(true);
		}
	});

	it('refuses unknown names and the empty string', () => {
		expect(acceptsZone('Mars/Olympus')).toBe(false);
		expect(acceptsZone('')).toBe(false);
	});
});

describe('withZone', () => {
	it('adds a zone missing from the list, sorted', () => {
		expect(withZone(['Asia/Calcutta', 'UTC'], 'Asia/Kolkata')).toEqual([
			'Asia/Calcutta',
			'Asia/Kolkata',
			'UTC'
		]);
	});

	it('never duplicates a zone already listed', () => {
		expect(withZone(['Asia/Calcutta', 'UTC'], 'UTC')).toEqual(['Asia/Calcutta', 'UTC']);
	});
});

describe('todayInOrUtc', () => {
	it('is today in the zone, or in UTC for a zone the runtime cannot format', () => {
		const now = new Date('2026-11-20T20:00:00Z');
		expect(todayInOrUtc('Asia/Kolkata', now)).toBe('2026-11-21');
		expect(todayInOrUtc('Mars/Olympus', now)).toBe('2026-11-20');
	});
});
