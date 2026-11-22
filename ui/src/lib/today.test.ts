import { describe, expect, it } from 'vitest';
import { MODULES } from './constants';
import { todayRows } from './today';
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
