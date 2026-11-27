import { localDate, temperatureText } from './entry-edit';
import {
	BOTTLE_CONTENT_LABEL,
	MODULES,
	SUMMARY_MAX_DAYS,
	SYMPTOM_LABEL,
	TEMP_METHOD_LABEL,
	TEMP_UNIT_LABEL,
	TIMELINE_KINDS,
	TIMELINE_TYPE,
	TRACKER_LABEL,
	TRACKER_ROUTE,
	VISIT_KIND_LABEL,
	VISIT_STATUS_LABEL
} from './constants';
import { formatAgo, formatDuration, formatMilk, formatVolume } from './format';
import { sinceAt } from './today';
import { formatClock, zonedLocalToInstant } from './time-zone';
import type {
	DaySummary,
	Home,
	ModuleId,
	ModuleState,
	NursingSide,
	TimelineItem,
	TimelineKind,
	TimelineQuery,
	TimelineType,
	TrackerTimelineItem,
	Units,
	VisitRow,
	VolumeUnit
} from './types';

const MS_PER_DAY = 24 * 60 * 60 * 1000;
const MS_PER_SECOND = 1000;
const WEEK_DAYS = 7;
const SIDE_INITIAL: Record<NursingSide, string> = { left: 'L', right: 'R' };

export const INVALID_DATE = 'Enter a valid date';
export const INVALID_CURSOR = 'That page link no longer works';
const REVERSED_RANGE = 'The end date is before the start date';
const LONG_RANGE = `Pick at most ${SUMMARY_MAX_DAYS} days`;

// One line per entry. A point entry leads with its clock; a sleep carries its own range.
export function timelineLine(
	item: TrackerTimelineItem,
	zone: string,
	units: Pick<Units, 'weight' | 'volume' | 'temp'>
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
		case 'symptoms': {
			const { symptom, severity, otherText, temperature, method } = item.entry;
			const name =
				symptom === 'other' && otherText
					? `${SYMPTOM_LABEL.other}: ${otherText}`
					: SYMPTOM_LABEL[symptom];
			const how = method ? ` (${TEMP_METHOD_LABEL[method].toLowerCase()})` : '';
			const reading = temperature
				? ` ${temperatureText(temperature, units.temp)} ${TEMP_UNIT_LABEL[units.temp]}${how}`
				: '';
			return `${clock} ${name}${reading}${severity ? `, ${severity}` : ''}`;
		}
	}
}

// A visit row: its kind, and "Cancelled" when it is.
export function visitTimelineLine(v: Pick<VisitRow, 'kind' | 'status'>): {
	title: string;
	detail: string | null;
} {
	return {
		title: VISIT_KIND_LABEL[v.kind],
		detail: v.status === 'cancelled' ? VISIT_STATUS_LABEL.cancelled : null
	};
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

const moduleKind = (module: ModuleId): TimelineKind | undefined =>
	module === 'visits' ? 'visit' : TRACKER_ROUTE[module];

// The kinds a kid's enabled modules switch on, in filter order.
export function defaultKinds(modules: readonly ModuleState[]): TimelineKind[] {
	return TIMELINE_KINDS.filter((k) => modules.some((m) => m.enabled && moduleKind(m.module) === k));
}

export function shownModules(kinds: readonly TimelineKind[]): Set<ModuleId> {
	return new Set(
		MODULES.filter((m) => {
			const kind = moduleKind(m);
			return kind !== undefined && kinds.includes(kind);
		})
	);
}

export type TimelineFilter = {
	from: string;
	to: string;
	// What the boxes show: the typed selection, else the enabled kinds.
	kinds: TimelineKind[];
	// True when the selection differs from the enabled kinds, so the API is told.
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
const sameKinds = (a: readonly TimelineKind[], b: readonly TimelineKind[]) =>
	a.length === b.length && a.every((k) => b.includes(k));

// from and to are the kid's calendar dates; the API gets [start of from, start of the day after to).
export function readFilter(
	params: URLSearchParams,
	ctx: { zone: string; today: string; defaults: readonly TimelineKind[] }
): FilterRead {
	const from = params.get('from') || ctx.today;
	const to = params.get('to') || ctx.today;
	const names = params.getAll('types').flatMap((v) => v.split(','));
	const typed = TIMELINE_KINDS.filter((k) => names.includes(TIMELINE_TYPE[k]));
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

const addDays = (date: string, n: number) =>
	new Date(Date.parse(`${date}T00:00:00Z`) + n * MS_PER_DAY).toISOString().slice(0, 10);

// The week strip's days: the seven ending today, oldest first.
export const weekDates = (today: string): string[] =>
	Array.from({ length: WEEK_DAYS }, (_, i) => addDays(today, i - (WEEK_DAYS - 1)));

export type WeekCell = {
	date: string;
	dow: string;
	label: string;
	segs: { type: TimelineType; count: number }[];
};

const countOf = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`;

// What a day's summary says happened, for the strip's accessible name; volumes in the parent's unit.
function dayParts(d: DaySummary, unit: VolumeUnit): string[] {
	return [
		d.bottles.count > 0 &&
			`${countOf(d.bottles.count, 'bottle', 'bottles')} (${formatVolume(d.bottles, unit)})`,
		d.nursing.count > 0 && `${d.nursing.count} nursing (${formatDuration(d.nursing.seconds)})`,
		d.pumping.count > 0 && `${d.pumping.count} pumping (${formatVolume(d.pumping, unit)})`,
		d.diapers.total > 0 && countOf(d.diapers.total, 'diaper', 'diapers'),
		d.meals.count > 0 && countOf(d.meals.count, 'meal', 'meals'),
		d.sleep.seconds > 0 && `sleep ${formatDuration(d.sleep.seconds)}`
	].filter((p): p is string => p !== false);
}

export function weekCells(
	days: readonly DaySummary[],
	today: string,
	unit: VolumeUnit
): WeekCell[] {
	return days.map((d) => {
		const parts = dayParts(d, unit);
		const heading =
			d.date === today
				? 'Today'
				: d.date === addDays(today, -1)
					? 'Yesterday'
					: formatDayHeading(d.date);
		const dow =
			d.date === today
				? 'Today'
				: d.date === addDays(today, -1)
					? 'Yest.'
					: formatDayHeading(d.date).split(' ')[0];
		const segs: WeekCell['segs'] = [
			{ type: 'bottle' as const, count: d.bottles.count },
			{ type: 'nursing' as const, count: d.nursing.count },
			{ type: 'pumping' as const, count: d.pumping.count },
			{ type: 'sleep' as const, count: d.sleep.seconds > 0 ? 1 : 0 },
			{ type: 'diaper' as const, count: d.diapers.total },
			{ type: 'meal' as const, count: d.meals.count }
		].filter((s) => s.count > 0);
		return {
			date: d.date,
			dow,
			label: `${heading}: ${parts.length === 0 ? 'nothing logged' : parts.join(' · ')}`,
			segs
		};
	});
}

export type SinceCell = { label: string; value: string };

const NONE_YET = 'None yet';

// Time since the last feed, diaper and sleep; a cell appears only while its trackers are on.
export function sinceLast(last: Home['last'], running: Home['running'], now: Date): SinceCell[] {
	const ago = (...types: TimelineType[]) => {
		const times = types
			.map((t) => last[t])
			.filter((i): i is TimelineItem => !!i)
			.map(sinceAt);
		if (times.length === 0) return NONE_YET;
		return formatAgo(
			times.reduce((a, b) => (a > b ? a : b)),
			now
		);
	};
	const on = (...types: TimelineType[]) => types.some((t) => t in last);
	const cells: SinceCell[] = [];
	if (on('bottle', 'nursing')) cells.push({ label: 'Last feed', value: ago('bottle', 'nursing') });
	if (on('diaper')) cells.push({ label: 'Last diaper', value: ago('diaper') });
	if (running.sleep) {
		const seconds = (now.getTime() - Date.parse(running.sleep.startedAt)) / MS_PER_SECOND;
		cells.push({ label: 'Asleep for', value: formatDuration(seconds) });
	} else if (on('sleep')) {
		cells.push({ label: 'Last sleep', value: ago('sleep') });
	}
	return cells;
}
