import { DIAPER_KINDS } from './constants';
import { formatDuration, formatVolume } from './format';
import type { DaySummary, ModuleId, VolumeUnit } from './types';

export type TodayRow = { label: string; value: string };

const NONE = 'None yet';

const rows: {
	module: ModuleId;
	label: string;
	value: (d: DaySummary, unit: VolumeUnit) => string | null;
}[] = [
	{
		module: 'bottles',
		label: 'Bottles',
		value: (d, unit) =>
			d.bottles.count === 0 ? null : `${d.bottles.count} · ${formatVolume(d.bottles, unit)}`
	},
	{
		module: 'nursing',
		label: 'Nursing',
		value: (d) =>
			d.nursing.count === 0 ? null : `${d.nursing.count} · ${formatDuration(d.nursing.seconds)}`
	},
	{
		module: 'diapers',
		label: 'Diapers',
		value: (d) => {
			if (d.diapers.total === 0) return null;
			const kinds = DIAPER_KINDS.filter((k) => d.diapers[k] > 0).map((k) => `${d.diapers[k]} ${k}`);
			return `${d.diapers.total} (${kinds.join(', ')})`;
		}
	},
	{
		module: 'sleep',
		label: 'Sleep',
		value: (d) => (d.sleep.seconds === 0 ? null : formatDuration(d.sleep.seconds))
	}
];

const valued = (day: DaySummary, enabled: ReadonlySet<ModuleId>, unit: VolumeUnit) =>
	rows
		.filter((r) => enabled.has(r.module))
		.map((r) => ({ label: r.label, value: r.value(day, unit) }));

// One row per enabled module that has a tracker summary; the volume is in the parent's unit.
export function todayRows(
	day: DaySummary,
	enabled: ReadonlySet<ModuleId>,
	unit: VolumeUnit
): TodayRow[] {
	return valued(day, enabled, unit).map((r) => ({ label: r.label, value: r.value ?? NONE }));
}

// The same rows for a past day, leaving out a tracker with nothing on that day.
export function dayRows(
	day: DaySummary,
	shown: ReadonlySet<ModuleId>,
	unit: VolumeUnit
): TodayRow[] {
	return valued(day, shown, unit).flatMap((r) =>
		r.value === null ? [] : [{ label: r.label, value: r.value }]
	);
}
