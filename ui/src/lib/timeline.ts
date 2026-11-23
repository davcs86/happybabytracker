import { localDate } from './entry-edit';
import {
	BOTTLE_CONTENT_LABEL,
	SUMMARY_MAX_DAYS,
	TIMELINE_TYPE,
	TRACKER_KINDS,
	TRACKER_LABEL,
	TRACKER_ROUTE
} from './constants';
import { formatDuration, formatMilk, formatVolume } from './format';
import { formatClock, zonedLocalToInstant } from './time-zone';
import type {
	ModuleId,
	ModuleState,
	NursingSide,
	TimelineItem,
	TimelineQuery,
	TrackerKind,
	Units
} from './types';

const MS_PER_DAY = 24 * 60 * 60 * 1000;
const MS_PER_SECOND = 1000;
const SIDE_INITIAL: Record<NursingSide, string> = { left: 'L', right: 'R' };

export const INVALID_DATE = 'Enter a valid date';
export const INVALID_CURSOR = 'That page link no longer works';
const REVERSED_RANGE = 'The end date is before the start date';
const LONG_RANGE = `Pick at most ${SUMMARY_MAX_DAYS} days`;

// One line per entry. A point entry leads with its clock; a sleep carries its own range.
export function timelineLine(
	item: TimelineItem,
	zone: string,
	units: Pick<Units, 'weight' | 'volume'>
): string {
	const clock = formatClock(item.at, zone);
	const label = TRACKER_LABEL[item.kind];
	switch (item.kind) {
		case 'bottles': {
			const contents = item.entry.contents
				.map((c) => BOTTLE_CONTENT_LABEL[c].toLowerCase())
				.join(' + ');
			const volume = formatVolume(item.entry.volume, units.volume);
			return `${clock} ${label} ${volume}${contents ? `, ${contents}` : ''}`;
		}
		case 'nursing': {
			const { leftSeconds, rightSeconds, endedAt, side, milk } = item.entry;
			const sides = [
				...(leftSeconds > 0 ? [`${SIDE_INITIAL.left} ${formatDuration(leftSeconds)}`] : []),
				...(rightSeconds > 0 ? [`${SIDE_INITIAL.right} ${formatDuration(rightSeconds)}`] : [])
			];
			const running = endedAt === null ? `, running${side ? ` on ${SIDE_INITIAL[side]}` : ''}` : '';
			const estimate = milk ? `, ${formatMilk(milk, units)}` : '';
			return `${clock} ${label}${sides.length > 0 ? ` ${sides.join(' · ')}` : ''}${running}${estimate}`;
		}
		case 'pumping': {
			const { durationSeconds, volume, side, sides } = item.entry;
			const where =
				sides.length > 0
					? sides
							.map((s) => `${SIDE_INITIAL[s.side]} ${formatVolume(s.volume, units.volume)}`)
							.join(' · ')
					: side;
			return `${clock} ${label} ${formatDuration(durationSeconds)}, ${formatVolume(volume, units.volume)} (${where})`;
		}
		case 'sleep': {
			const { startedAt, endedAt } = item.entry;
			if (endedAt === null) return `${label} since ${clock}`;
			const seconds = (Date.parse(endedAt) - Date.parse(startedAt)) / MS_PER_SECOND;
			const days =
				(Date.parse(localDate(endedAt, zone)) - Date.parse(localDate(startedAt, zone))) /
				MS_PER_DAY;
			const later = days > 0 ? ` +${days} day${days === 1 ? '' : 's'}` : '';
			return `${label} ${clock}–${formatClock(endedAt, zone)}${later} (${formatDuration(seconds)})`;
		}
		case 'diapers':
			return `${clock} ${label} ${item.entry.kind}`;
		case 'meals': {
			const { what, reaction } = item.entry;
			return `${clock} ${label}: ${what}${reaction ? ` · ${reaction}` : ''}`;
		}
	}
}

export type DayGroup = { date: string; items: TimelineItem[] };

// The page is already newest first, so the days of one page are consecutive runs.
export function groupByDay(items: readonly TimelineItem[], zone: string): DayGroup[] {
	const groups: DayGroup[] = [];
	for (const item of items) {
		const date = localDate(item.at, zone);
		const last = groups.at(-1);
		if (last?.date === date) last.items.push(item);
		else groups.push({ date, items: [item] });
	}
	return groups;
}

// date is YYYY-MM-DD; formatted in UTC so neither the server's zone nor the locale moves it.
export function formatDayHeading(date: string): string {
	const parts = new Intl.DateTimeFormat('en-US', {
		timeZone: 'UTC',
		weekday: 'short',
		day: 'numeric',
		month: 'short',
		year: 'numeric'
	}).formatToParts(new Date(`${date}T00:00:00Z`));
	const part = (type: string) => parts.find((p) => p.type === type)?.value;
	return `${part('weekday')} ${part('day')} ${part('month')} ${part('year')}`;
}

// The trackers a kid's enabled modules switch on, in tracker order.
export function defaultKinds(modules: readonly ModuleState[]): TrackerKind[] {
	return TRACKER_KINDS.filter((k) =>
		modules.some((m) => m.enabled && TRACKER_ROUTE[m.module] === k)
	);
}

export function shownModules(kinds: readonly TrackerKind[]): Set<ModuleId> {
	return new Set(
		(Object.entries(TRACKER_ROUTE) as [ModuleId, TrackerKind][])
			.filter(([, kind]) => kinds.includes(kind))
			.map(([module]) => module)
	);
}

export type TimelineFilter = {
	from: string;
	to: string;
	// What the boxes show: the typed selection, else the enabled trackers.
	kinds: TrackerKind[];
	// True when the selection differs from the enabled trackers, so the API is told.
	narrowed: boolean;
	cursor: string | null;
};
export type FilterError = { field: 'from' | 'to' | 'cursor'; message: string };
export type FilterRead = {
	filter: TimelineFilter;
	query: TimelineQuery | null;
	error: FilterError | null;
};

const dayAfter = (date: string) =>
	new Date(Date.parse(`${date}T00:00:00Z`) + MS_PER_DAY).toISOString().slice(0, 10);
const startOf = (date: string, zone: string) => zonedLocalToInstant(`${date}T00:00`, zone);
const sameKinds = (a: readonly TrackerKind[], b: readonly TrackerKind[]) =>
	a.length === b.length && a.every((k) => b.includes(k));

// from and to are the kid's calendar dates; the API gets [start of from, start of the day after to).
export function readFilter(
	params: URLSearchParams,
	ctx: { zone: string; today: string; defaults: readonly TrackerKind[] }
): FilterRead {
	const from = params.get('from') || ctx.today;
	const to = params.get('to') || ctx.today;
	const names = params.getAll('types').flatMap((v) => v.split(','));
	const typed = TRACKER_KINDS.filter((k) => names.includes(TIMELINE_TYPE[k]));
	const kinds = typed.length > 0 ? typed : [...ctx.defaults];
	const narrowed = typed.length > 0 && !sameKinds(typed, ctx.defaults);
	const filter: TimelineFilter = {
		from,
		to,
		kinds,
		narrowed,
		cursor: params.get('cursor') || null
	};
	const refuse = (field: 'from' | 'to', message: string): FilterRead => ({
		filter,
		query: null,
		error: { field, message }
	});

	const start = startOf(from, ctx.zone);
	if (start === null) return refuse('from', INVALID_DATE);
	if (startOf(to, ctx.zone) === null) return refuse('to', INVALID_DATE);
	const end = startOf(dayAfter(to), ctx.zone);
	if (end === null) return refuse('to', INVALID_DATE);
	const days = (Date.parse(to) - Date.parse(from)) / MS_PER_DAY + 1;
	if (days < 1) return refuse('to', REVERSED_RANGE);
	if (days > SUMMARY_MAX_DAYS) return refuse('to', LONG_RANGE);
	return {
		filter,
		error: null,
		query: {
			from: start,
			to: end,
			types: narrowed ? kinds.map((k) => TIMELINE_TYPE[k]) : undefined,
			cursor: filter.cursor ?? undefined
		}
	};
}

// The query a link carries, in the shape the filter form itself submits.
export function timelineSearch(f: TimelineFilter, cursor: string | null = f.cursor): string {
	const p = new URLSearchParams({ from: f.from, to: f.to });
	if (f.narrowed) for (const k of f.kinds) p.append('types', TIMELINE_TYPE[k]);
	if (cursor) p.set('cursor', cursor);
	return p.toString();
}
