import { DIAPER_KINDS } from './constants';
import { formatDuration, formatVolume } from './format';
import type { DaySummary, Home, ModuleId, TimelineItem, TimelineType, VolumeUnit } from './types';

// lastAt is when the tracker's newest entry happened, for the four trackers the Today card says "Last:" for.
// A finished session counts from when it ended; anything else from its time column.
export const sinceAt = (item: TimelineItem): string =>
	'endedAt' in item.entry && typeof item.entry.endedAt === 'string' ? item.entry.endedAt : item.at;

export type TodayRow = { label: string; value: string; lastAt?: string };

const NONE = 'None yet';

const rows: {
	module: ModuleId;
	label: string;
	last?: TimelineType;
	value: (d: DaySummary, unit: VolumeUnit) => string | null;
}[] = [
	{
		module: 'bottles',
		label: 'Bottles',
		last: 'bottle',
		value: (d, unit) =>
			d.bottles.count === 0 ? null : `${d.bottles.count} · ${formatVolume(d.bottles, unit)}`
	},
	{
		module: 'nursing',
		label: 'Nursing',
		last: 'nursing',
		value: (d) =>
			d.nursing.count === 0 ? null : `${d.nursing.count} · ${formatDuration(d.nursing.seconds)}`
	},
	{
		module: 'diapers',
		label: 'Diapers',
		last: 'diaper',
		value: (d) => {
			if (d.diapers.total === 0) return null;
			const kinds = DIAPER_KINDS.filter((k) => d.diapers[k] > 0).map((k) => `${d.diapers[k]} ${k}`);
			return `${d.diapers.total} (${kinds.join(', ')})`;
		}
	},
	{
		module: 'sleep',
		label: 'Sleep',
		last: 'sleep',
		value: (d) => (d.sleep.seconds === 0 ? null : formatDuration(d.sleep.seconds))
	}
];

const valued = (day: DaySummary, enabled: ReadonlySet<ModuleId>, unit: VolumeUnit) =>
	rows
		.filter((r) => enabled.has(r.module))
		.map((r) => ({ label: r.label, value: r.value(day, unit), last: r.last }));

// One row per enabled module that has a tracker summary; the volume is in the parent's unit.
export function todayRows(
	day: DaySummary,
	enabled: ReadonlySet<ModuleId>,
	unit: VolumeUnit,
	last: Home['last'] = {}
): TodayRow[] {
	return valued(day, enabled, unit).map((r) => {
		const item = r.last && last[r.last];
		const at = item && sinceAt(item);
		return { label: r.label, value: r.value ?? NONE, ...(at ? { lastAt: at } : {}) };
	});
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
