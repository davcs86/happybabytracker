import { MODULE_LABEL, MODULES, QUICK_LOG_FIRST, TIMELINE_TYPE, TRACKER_ROUTE } from './constants';
import { sinceAt } from './today';
import { growthNewPath, medicationPath, milestonesPath, vaccinesPath } from './paths';
import type { AgeBand, Home, ModuleId, ModuleState, TrackerKind } from './types';

// Modules whose entry point is a page of their own rather than a tracker's log form.
export const PAGE_ROUTE: Partial<Record<ModuleId, (kidId: string) => string>> = {
	milestones: milestonesPath,
	growth: growthNewPath,
	vaccination: vaccinesPath,
	medication: medicationPath
};

// kind is null for a module with a page of its own (PAGE_ROUTE).
// lastAt is when the tracker's newest entry happened.
export type QuickLogEntry = {
	module: ModuleId;
	kind: TrackerKind | null;
	label: string;
	lastAt?: string;
};

// Enabled modules with a tracker or a page, in the band's order; a running timer replaces its start entry.
export function quickLog(
	band: AgeBand,
	modules: ModuleState[],
	running: Home['running'],
	last: Home['last'] = {}
): QuickLogEntry[] {
	const on = new Set(modules.filter((m) => m.enabled).map((m) => m.module));
	const first = QUICK_LOG_FIRST[band];
	const ordered = [...first, ...MODULES.filter((m) => !first.includes(m))];
	const busy = (kind: TrackerKind) => (kind === 'sleep' || kind === 'nursing') && !!running[kind];
	return ordered.flatMap((module) => {
		const kind = TRACKER_ROUTE[module] ?? null;
		if (!on.has(module) || (kind ? busy(kind) : !PAGE_ROUTE[module])) return [];
		const item = kind && last[TIMELINE_TYPE[kind]];
		const at = item && sinceAt(item);
		return [{ module, kind, label: MODULE_LABEL[module], ...(at ? { lastAt: at } : {}) }];
	});
}
