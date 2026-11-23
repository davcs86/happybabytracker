import { describe, expect, it } from 'vitest';
import { courseEnded, scheduleLabel, whenLabel } from './medication';

describe('scheduleLabel', () => {
	const START = '2026-11-20T08:00:00Z';

	it('says as needed for a course with no schedule, whatever the start', () => {
		expect(scheduleLabel({ kind: 'none' }, START, 'UTC')).toBe('As needed');
		expect(scheduleLabel({ kind: 'none' }, 'not an instant', 'UTC')).toBe('As needed');
	});

	it('lists the times as the API sent them, never reordered', () => {
		expect(scheduleLabel({ kind: 'times', times: ['08:00', '20:00'] }, START, 'UTC')).toBe(
			'08:00, 20:00'
		);
		expect(scheduleLabel({ kind: 'times', times: ['08:00'] }, START, 'UTC')).toBe('08:00');
		expect(scheduleLabel({ kind: 'times', times: ['20:00', '08:00'] }, START, 'UTC')).toBe(
			'20:00, 08:00'
		);
		const twelve = Array.from({ length: 12 }, (_, i) => `${String(i * 2).padStart(2, '0')}:30`);
		expect(scheduleLabel({ kind: 'times', times: twelve }, START, 'UTC')).toBe(twelve.join(', '));
		expect(scheduleLabel({ kind: 'times', times: ['08:00'] }, 'not an instant', 'UTC')).toBe(
			'08:00'
		);
	});

	it('names the grid and the clock time of its start in the kid zone', () => {
		expect(scheduleLabel({ kind: 'every', hours: 6 }, START, 'UTC')).toBe(
			'Every 6 hours from 08:00'
		);
		expect(scheduleLabel({ kind: 'every', hours: 1 }, START, 'UTC')).toBe('Every hour from 08:00');
		expect(scheduleLabel({ kind: 'every', hours: 168 }, START, 'UTC')).toBe(
			'Every 168 hours from 08:00'
		);
	});

	it('reads the start on the kid clock, winter and summer', () => {
		expect(scheduleLabel({ kind: 'every', hours: 8 }, START, 'Europe/Madrid')).toBe(
			'Every 8 hours from 09:00'
		);
		expect(
			scheduleLabel({ kind: 'every', hours: 8 }, '2026-07-01T08:00:00Z', 'Europe/Madrid')
		).toBe('Every 8 hours from 10:00');
		// 07:30Z on the spring-forward day is 03:30 EDT: the skipped 02:30 never shows.
		expect(
			scheduleLabel({ kind: 'every', hours: 4 }, '2026-03-08T07:30:00Z', 'America/New_York')
		).toBe('Every 4 hours from 03:30');
		expect(scheduleLabel({ kind: 'every', hours: 4 }, '2026-11-20T23:30:00Z', 'Asia/Kolkata')).toBe(
			'Every 4 hours from 05:00'
		);
	});

	it('reads a zone the runtime cannot format as UTC', () => {
		expect(scheduleLabel({ kind: 'every', hours: 6 }, START, 'Not/AZone')).toBe(
			'Every 6 hours from 08:00'
		);
		expect(scheduleLabel({ kind: 'every', hours: 6 }, START, '')).toBe('Every 6 hours from 08:00');
	});
});

describe('whenLabel', () => {
	it('shows the clock time alone for today on the kid clock', () => {
		expect(whenLabel('2026-11-23T14:05:00Z', 'UTC', '2026-11-23')).toBe('14:05');
		expect(whenLabel('2026-11-23T00:00:00Z', 'UTC', '2026-11-23')).toBe('00:00');
		expect(whenLabel('2026-11-23T23:59:59Z', 'UTC', '2026-11-23')).toBe('23:59');
	});

	it('adds the kid-local date for any other day, so a time is never read as today', () => {
		expect(whenLabel('2026-11-22T14:05:00Z', 'UTC', '2026-11-23')).toBe('2026-11-22 14:05');
		expect(whenLabel('2026-11-24T08:00:00Z', 'UTC', '2026-11-23')).toBe('2026-11-24 08:00');
		expect(whenLabel('2025-12-31T23:59:00Z', 'UTC', '2026-01-01')).toBe('2025-12-31 23:59');
	});

	it('compares the day on the kid clock, not on UTC', () => {
		// 23:30Z is already 00:30 the next day in Madrid.
		expect(whenLabel('2026-11-23T23:30:00Z', 'Europe/Madrid', '2026-11-24')).toBe('00:30');
		expect(whenLabel('2026-11-23T23:30:00Z', 'Europe/Madrid', '2026-11-23')).toBe(
			'2026-11-24 00:30'
		);
		expect(whenLabel('2026-11-23T23:30:00Z', 'Not/AZone', '2026-11-23')).toBe('23:30');
	});
});

describe('courseEnded', () => {
	it('is ended only once its last day is before today', () => {
		expect(courseEnded(null, '2026-11-23')).toBe(false);
		expect(courseEnded('2026-11-23', '2026-11-23')).toBe(false);
		expect(courseEnded('2026-11-24', '2026-11-23')).toBe(false);
		expect(courseEnded('2026-11-22', '2026-11-23')).toBe(true);
		expect(courseEnded('2025-12-31', '2026-01-01')).toBe(true);
	});
});
