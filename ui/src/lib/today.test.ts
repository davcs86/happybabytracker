import { describe, expect, it } from 'vitest';
import { MODULES } from './constants';
import { dayRows, todayRows } from './today';
import type { DaySummary, ModuleId } from './types';

const day: DaySummary = {
	date: '2026-11-22',
	bottles: { count: 5, ml: 650, floz_hundredths: 2198 },
	nursing: { count: 3, seconds: 41 * 60 },
	pumping: { count: 0, ml: 0, floz_hundredths: 0 },
	diapers: { total: 6, wet: 4, dirty: 2, both: 0, dry: 0 },
	sleep: { seconds: 9 * 3600 + 20 * 60 },
	meals: { count: 0 }
};
const all = new Set<ModuleId>(MODULES);

describe('todayRows', () => {
	it('shows each enabled tracker in the parent unit', () => {
		expect(todayRows(day, all, 'ml')).toEqual([
			{ label: 'Bottles', value: '5 · 650 ml' },
			{ label: 'Nursing', value: '3 · 41 min' },
			{ label: 'Diapers', value: '6 (4 wet, 2 dirty)' },
			{ label: 'Sleep', value: '9 h 20 min' }
		]);
		expect(todayRows(day, all, 'floz')[0]).toEqual({ label: 'Bottles', value: '5 · 21.98 fl oz' });
	});

	it('names only the diaper kinds that happened', () => {
		const rows = todayRows(
			{ ...day, diapers: { total: 4, wet: 1, dirty: 0, both: 2, dry: 1 } },
			all,
			'ml'
		);
		expect(rows[2].value).toBe('4 (1 wet, 2 both, 1 dry)');
	});

	it('says none yet for an empty day', () => {
		const empty: DaySummary = {
			...day,
			bottles: { count: 0, ml: 0, floz_hundredths: 0 },
			nursing: { count: 0, seconds: 0 },
			diapers: { total: 0, wet: 0, dirty: 0, both: 0, dry: 0 },
			sleep: { seconds: 0 }
		};
		expect(todayRows(empty, all, 'ml').map((r) => r.value)).toEqual([
			'None yet',
			'None yet',
			'None yet',
			'None yet'
		]);
	});

	it('leaves out a disabled module', () => {
		expect(
			todayRows(day, new Set<ModuleId>(['sleep', 'bottles']), 'ml').map((r) => r.label)
		).toEqual(['Bottles', 'Sleep']);
		expect(todayRows(day, new Set<ModuleId>(['growth']), 'ml')).toEqual([]);
	});
});

describe('todayRows last entries', () => {
	const item = (kind: 'bottles' | 'nursing' | 'diapers' | 'sleep', at: string) =>
		({ kind, at, entry: { id: 'x' } }) as never;
	it('carries when the newest bottle, nursing, diaper and sleep happened, and nothing for a tracker with none', () => {
		const last = {
			bottle: item('bottles', '2026-11-22T10:00:00Z'),
			nursing: null,
			diaper: item('diapers', '2026-11-22T11:00:00Z'),
			sleep: item('sleep', '2026-11-22T09:00:00Z')
		};
		const rows = todayRows(day, all, 'ml', last);
		expect(rows.map((r) => r.lastAt)).toEqual([
			'2026-11-22T10:00:00Z',
			undefined,
			'2026-11-22T11:00:00Z',
			'2026-11-22T09:00:00Z'
		]);
	});
	it('counts a finished sleep or nursing session from when it ended, a running one from its start', () => {
		const finished = (kind: 'sleep' | 'nursing', at: string, endedAt: string | null) =>
			({ kind, at, entry: { id: 'x', endedAt } }) as never;
		const rows = todayRows(day, all, 'ml', {
			nursing: finished('nursing', '2026-11-22T08:00:00Z', '2026-11-22T08:20:00Z'),
			sleep: finished('sleep', '2026-11-22T09:00:00Z', null)
		});
		expect(rows.find((r) => r.label === 'Nursing')?.lastAt).toBe('2026-11-22T08:20:00Z');
		expect(rows.find((r) => r.label === 'Sleep')?.lastAt).toBe('2026-11-22T09:00:00Z');
	});
	it('carries nothing when the API sent no last', () => {
		expect(todayRows(day, all, 'ml').every((r) => r.lastAt === undefined)).toBe(true);
	});
});

describe('dayRows', () => {
	it('shows only the shown trackers that have something on the day', () => {
		const empty = { ...day, bottles: { count: 0, ml: 0, floz_hundredths: 0 } };
		expect(dayRows(empty, all, 'ml').map((r) => r.label)).toEqual(['Nursing', 'Diapers', 'Sleep']);
		expect(dayRows(day, new Set<ModuleId>(['bottles', 'sleep']), 'ml')).toEqual([
			{ label: 'Bottles', value: '5 · 650 ml' },
			{ label: 'Sleep', value: '9 h 20 min' }
		]);
		expect(dayRows(day, new Set<ModuleId>(), 'ml')).toEqual([]);
	});
});
