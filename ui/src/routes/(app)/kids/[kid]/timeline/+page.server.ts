import { dayRows, type TodayRow } from '#lib/today.js';
import { todayInOrUtc } from '#lib/time-zone.js';
import {
	INVALID_CURSOR,
	INVALID_DATE,
	defaultKinds,
	groupByDay,
	readFilter,
	shownModules,
	timelineSearch,
	weekCells,
	weekDates,
	type FilterError
} from '#lib/timeline.js';
import { setKidCookie } from '#lib/server/cookie.js';
import { failPage } from '#lib/server/outcome.js';
import type { TimelineItem } from '#lib/types.js';
import type { PageServerLoad } from './$types';

type Day = { date: string; items: TimelineItem[]; totals: TodayRow[] };

export const load: PageServerLoad = async ({ locals, params, url, cookies, parent }) => {
	const [kid, modules, home, { session }] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.modules(params.kid),
		locals.api.home(params.kid),
		parent()
	]);
	if (!kid.ok) failPage(kid);
	if (!modules.ok) failPage(modules);
	setKidCookie(cookies, kid.data.id);
	const zone = kid.data.timeZone;
	const unit = session?.parent?.units.volume ?? 'ml';
	const today = todayInOrUtc(zone);
	const { filter, query, error } = readFilter(url.searchParams, {
		zone,
		today,
		defaults: defaultKinds(modules.data)
	});
	// The strip, Since last and the running cards are extras: a failed call leaves them out.
	const days7 = weekDates(today);
	const weekR = await locals.api.summary(params.kid, days7[0], days7[6]);
	const day = (date: string) =>
		timelineSearch({ ...filter, from: date, to: date, cursor: null }, null);
	const view = {
		week: weekR.ok
			? weekCells(weekR.data, today, unit).map((c) => ({
					...c,
					query: day(c.date),
					current: filter.from === c.date && filter.to === c.date
				}))
			: null,
		// Showing every day of the strip is the link back from a single day.
		allDays:
			filter.from === filter.to
				? timelineSearch({ ...filter, from: days7[0], to: today, cursor: null }, null)
				: null,
		last: home.ok ? home.data.last : null,
		running: home.ok ? home.data.running : null,
		kid: kid.data,
		units: {
			weight: session?.parent?.units.weight ?? 'kg',
			volume: unit,
			temp: session?.parent?.units.temp ?? 'c'
		},
		filter,
		newest: filter.cursor ? timelineSearch(filter, null) : null,
		days: [] as Day[],
		older: null as string | null,
		error: error as FilterError | null
	};
	if (!query) return view;

	// One summary call covers every day of the range.
	const [page, summary] = await Promise.all([
		locals.api.timeline(params.kid, query),
		locals.api.summary(params.kid, filter.from, filter.to)
	]);
	if (!page.ok) {
		// The form fields were checked before the call; the API alone judges the cursor.
		if (page.reason === 'invalid' && page.field === 'cursor') {
			return { ...view, error: { field: 'cursor', message: INVALID_CURSOR } };
		}
		if (page.reason === 'invalid' && (page.field === 'from' || page.field === 'to')) {
			return { ...view, error: { field: page.field, message: INVALID_DATE } };
		}
		failPage(page);
	}
	if (page.data.items.length > 0 && !summary.ok) failPage(summary);
	const shown = shownModules(filter.kinds);
	const totals = new Map(
		(summary.ok ? summary.data : []).map((d) => [d.date, dayRows(d, shown, unit)])
	);
	return {
		...view,
		days: groupByDay(page.data.items, zone).map((g) => ({
			...g,
			totals: totals.get(g.date) ?? []
		})),
		older: page.data.nextCursor ? timelineSearch(filter, page.data.nextCursor) : null
	};
};
