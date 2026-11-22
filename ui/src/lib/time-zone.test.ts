import { describe, expect, it } from 'vitest';
import {
	acceptsZone,
	instantToLocalInput,
	todayInOrUtc,
	withZone,
	zonedLocalToInstant
} from './time-zone';

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

describe('zonedLocalToInstant', () => {
	it('reads a wall time in the zone as an RFC 3339 UTC instant', () => {
		expect(zonedLocalToInstant('2026-11-20T10:30', 'UTC')).toBe('2026-11-20T10:30:00Z');
		expect(zonedLocalToInstant('2026-11-20T10:30', 'Europe/Madrid')).toBe('2026-11-20T09:30:00Z');
		expect(zonedLocalToInstant('2026-07-20T10:30', 'Europe/Madrid')).toBe('2026-07-20T08:30:00Z');
		expect(zonedLocalToInstant('2026-11-20T10:30', 'Asia/Kolkata')).toBe('2026-11-20T05:00:00Z');
	});

	it('moves a time skipped by the spring-forward gap to the same minute after it', () => {
		// Madrid skips 02:00-03:00 on 2026-03-29, so 02:30 becomes 03:30 CEST.
		expect(zonedLocalToInstant('2026-03-29T02:30', 'Europe/Madrid')).toBe('2026-03-29T01:30:00Z');
		expect(zonedLocalToInstant('2026-03-08T02:30', 'America/Los_Angeles')).toBe(
			'2026-03-08T10:30:00Z'
		);
	});

	it('takes the earlier instant for a time that happens twice', () => {
		// Madrid repeats 02:00-03:00 on 2026-10-25: 00:30Z is CEST, 01:30Z is CET.
		expect(zonedLocalToInstant('2026-10-25T02:30', 'Europe/Madrid')).toBe('2026-10-25T00:30:00Z');
		expect(zonedLocalToInstant('2026-11-01T01:30', 'America/Los_Angeles')).toBe(
			'2026-11-01T08:30:00Z'
		);
	});

	it('leaves the times next to a transition alone', () => {
		expect(zonedLocalToInstant('2026-03-29T01:59', 'Europe/Madrid')).toBe('2026-03-29T00:59:00Z');
		expect(zonedLocalToInstant('2026-03-29T03:00', 'Europe/Madrid')).toBe('2026-03-29T01:00:00Z');
		expect(zonedLocalToInstant('2026-10-25T03:00', 'Europe/Madrid')).toBe('2026-10-25T02:00:00Z');
	});

	it.each([
		'',
		'garbage',
		'2026-11-20',
		'2026-11-20 10:30',
		'2026-11-20T10:30:15',
		'2026-11-20T10:30Z',
		'2026-11-20T24:00',
		'2026-11-20T10:60',
		'2026-13-01T10:30',
		'2026-02-30T10:30',
		'2026-11-31T10:30'
	])('refuses %j', (local) => {
		expect(zonedLocalToInstant(local, 'Europe/Madrid')).toBeNull();
	});

	it('accepts 29 Feb only in a leap year', () => {
		expect(zonedLocalToInstant('2028-02-29T12:00', 'UTC')).toBe('2028-02-29T12:00:00Z');
		expect(zonedLocalToInstant('2026-02-29T12:00', 'UTC')).toBeNull();
	});

	it('reads the wall time in UTC for a zone the runtime cannot format', () => {
		expect(zonedLocalToInstant('2026-11-20T10:30', 'Mars/Olympus')).toBe('2026-11-20T10:30:00Z');
	});
});

describe('instantToLocalInput', () => {
	it('shows an instant as the zone wall time, minutes only', () => {
		expect(instantToLocalInput('2026-11-20T09:30:45Z', 'Europe/Madrid')).toBe('2026-11-20T10:30');
		expect(instantToLocalInput('2026-03-29T01:30:00Z', 'Europe/Madrid')).toBe('2026-03-29T03:30');
		expect(instantToLocalInput('2026-11-20T23:59:00Z', 'Pacific/Auckland')).toBe(
			'2026-11-21T12:59'
		);
		expect(instantToLocalInput('2026-11-20T00:00:00Z', 'UTC')).toBe('2026-11-20T00:00');
	});

	it('shows midnight as 00:00, never 24:00', () => {
		expect(instantToLocalInput('2026-11-19T23:00:00Z', 'Europe/Madrid')).toBe('2026-11-20T00:00');
	});

	it('round-trips every wall minute; a repeated hour reads back one hour earlier', () => {
		for (const zone of ['UTC', 'Europe/Madrid', 'America/Los_Angeles', 'Asia/Kolkata']) {
			for (let h = 0; h < 24 * 400; h += 7) {
				const ms = Date.UTC(2026, 0, 1) + h * 3_600_000 + 17 * 60_000;
				const local = instantToLocalInput(new Date(ms).toISOString(), zone);
				const back = zonedLocalToInstant(local, zone);
				expect([0, 3_600_000]).toContain(ms - Date.parse(back ?? ''));
				expect(instantToLocalInput(back ?? '', zone)).toBe(local);
			}
		}
	});
});
