import { MODULE_LABEL, MODULES, PAGE_ROUTE, QUICK_LOG_FIRST, TRACKER_ROUTE } from './constants';
import type { AgeBand, Home, ModuleId, ModuleState, TrackerKind } from './types';

// kind is null for a module with a page of its own (PAGE_ROUTE).
export type QuickLogEntry = { module: ModuleId; kind: TrackerKind | null; label: string };

// Enabled modules with a tracker or a page, in the band's order; a running timer replaces its start entry.
export function quickLog(
	band: AgeBand,
	modules: ModuleState[],
	running: Home['running']
): QuickLogEntry[] {
	const on = new Set(modules.filter((m) => m.enabled).map((m) => m.module));
	const first = QUICK_LOG_FIRST[band];
	const ordered = [...first, ...MODULES.filter((m) => !first.includes(m))];
	const busy = (kind: TrackerKind) => (kind === 'sleep' || kind === 'nursing') && !!running[kind];
	return ordered.flatMap((module) => {
		const kind = TRACKER_ROUTE[module] ?? null;
		if (!on.has(module) || (kind ? busy(kind) : !PAGE_ROUTE[module])) return [];
		return [{ module, kind, label: MODULE_LABEL[module] }];
	});
}
