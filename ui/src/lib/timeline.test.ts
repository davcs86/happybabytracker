import { describe, expect, it } from 'vitest';
import { SUMMARY_MAX_DAYS } from './constants';
import {
	defaultKinds,
	formatDayHeading,
	groupByDay,
	sinceLast,
	weekCells,
	weekDates,
	readFilter,
	shownModules,
	timelineLine,
	timelineSearch,
	visitTimelineLine
} from './timeline';
import type {
	DaySummary,
	ModuleId,
	ModuleState,
	TimelineItem,
	TimelineKind,
	TrackerTimelineItem,
	TrackerKind,
	VisitRow
} from './types';

const METRIC = { weight: 'kg', volume: 'ml', temp: 'c' } as const;
const IMPERIAL = { weight: 'lb', volume: 'floz', temp: 'f' } as const;
const vol = (ml: number, hundredths: number) => ({
	ml,
	floz_hundredths: hundredths,
	typed_unit: 'ml' as const
});
const bottle = (at: string, contents: string[], ml = 120): TrackerTimelineItem =>
	({
		kind: 'bottles',
		at,
		entry: {
			id: 'b1',
			at,
			contents,
			volume: vol(ml, Math.round(ml / 0.295735)),
			sources: [],
			note: null
		}
	}) as TrackerTimelineItem;
const sleep = (startedAt: string, endedAt: string | null): TrackerTimelineItem =>
	({
		kind: 'sleep',
		at: startedAt,
		entry: { id: 's1', startedAt, endedAt, note: null }
	}) as TrackerTimelineItem;
const nursing = (over: Record<string, unknown>): TrackerTimelineItem =>
	({
		kind: 'nursing',
		at: '2026-10-14T13:05:00Z',
		entry: {
			id: 'n1',
			startedAt: '2026-10-14T13:05:00Z',
			endedAt: '2026-10-14T13:25:00Z',
			leftSeconds: 480,
			rightSeconds: 300,
			side: null,
			sideSince: null,
			note: null,
			weightBefore: null,
			weightAfter: null,
			milk: null,
			...over
		}
	}) as TrackerTimelineItem;
const diaper = (at: string, kind = 'wet'): TrackerTimelineItem =>
	({ kind: 'diapers', at, entry: { id: `d-${at}`, at, kind, note: null } }) as TrackerTimelineItem;

describe('timelineLine', () => {
	it('appends the milk estimate to a weighed nursing session, in the parent units', () => {
		const milk = {
			gain: { grams: 62, ozHundredths: 219 },
			volume: { ml: 60, flozHundredths: 204 }
		};
		const weighed = nursing({ milk });
		expect(timelineLine(weighed, 'UTC', METRIC)).toBe(
			'13:05 Nursing L 8 min · R 5 min, weight gain 62 g, about 60 ml'
		);
		expect(timelineLine(weighed, 'UTC', IMPERIAL)).toBe(
			'13:05 Nursing L 8 min · R 5 min, weight gain 2.19 oz, about 2.04 fl oz'
		);
	});

	it('shows a zero estimate, and nothing for a session with weights but no estimate', () => {
		const zero = { gain: { grams: 0, ozHundredths: 0 }, volume: { ml: 0, flozHundredths: 0 } };
		expect(timelineLine(nursing({ milk: zero }), 'UTC', METRIC)).toBe(
			'13:05 Nursing L 8 min · R 5 min, weight gain 0 g, about 0 ml'
		);
		const grams = { grams: 3400, ozHundredths: 11992, typedUnit: 'kg' };
		const lighter = nursing({
			weightBefore: grams,
			weightAfter: { ...grams, grams: 3300 },
			milk: null
		});
		expect(timelineLine(lighter, 'UTC', METRIC)).toBe('13:05 Nursing L 8 min · R 5 min');
	});

	it('leaves an unweighed session exactly as it was', () => {
		expect(timelineLine(nursing({}), 'UTC', METRIC)).toBe('13:05 Nursing L 8 min · R 5 min');
	});

	it('appends the estimate after a running tag too', () => {
		const milk = { gain: { grams: 5, ozHundredths: 18 }, volume: { ml: 5, flozHundredths: 16 } };
		const running = nursing({ endedAt: null, rightSeconds: 0, side: 'right', milk });
		expect(timelineLine(running, 'UTC', METRIC)).toBe(
			'13:05 Nursing L 8 min, running on R, weight gain 5 g, about 5 ml'
		);
	});

	it('names a bottle with its volume in the parent unit and its contents', () => {
		const b = bottle('2026-10-14T13:05:00Z', ['breast_milk', 'formula']);
		expect(timelineLine(b, 'UTC', METRIC)).toBe('13:05 Bottle 120 ml, breast milk + formula');
		expect(timelineLine(b, 'UTC', { weight: 'kg', volume: 'floz', temp: 'c' })).toBe(
			'13:05 Bottle 4.06 fl oz, breast milk + formula'
		);
		expect(timelineLine(bottle('2026-10-14T13:05:00Z', ['cow_milk', 'other']), 'UTC', METRIC)).toBe(
			"13:05 Bottle 120 ml, cow's milk + other"
		);
	});

	it('shows the clock of the kid zone, never UTC', () => {
		expect(timelineLine(bottle('2026-03-28T23:30:00Z', ['formula']), 'Europe/Madrid', METRIC)).toBe(
			'00:30 Bottle 120 ml, formula'
		);
		expect(timelineLine(bottle('2026-03-29T21:30:00Z', ['formula']), 'Europe/Madrid', METRIC)).toBe(
			'23:30 Bottle 120 ml, formula'
		);
	});

	it('shows a finished sleep as a range with its length, a running one as since', () => {
		expect(timelineLine(sleep('2026-10-14T13:05:00Z', '2026-10-14T14:40:00Z'), 'UTC', METRIC)).toBe(
			'Sleep 13:05–14:40 (1 h 35 min)'
		);
		expect(timelineLine(sleep('2026-10-14T13:05:00Z', null), 'UTC', METRIC)).toBe(
			'Sleep since 13:05'
		);
		expect(
			timelineLine(sleep('2026-03-28T23:30:00Z', '2026-03-29T00:30:00Z'), 'Europe/Madrid', METRIC)
		).toBe('Sleep 00:30–01:30 (1 h)');
	});

	it('marks the end of a sleep that ends on a later kid-local day', () => {
		expect(
			timelineLine(sleep('2026-11-20T21:00:00Z', '2026-11-21T05:00:00Z'), 'Europe/Madrid', METRIC)
		).toBe('Sleep 22:00–06:00 +1 day (8 h)');
		expect(
			timelineLine(sleep('2026-11-20T10:00:00Z', '2026-11-21T10:00:00Z'), 'Europe/Madrid', METRIC)
		).toBe('Sleep 11:00–11:00 +1 day (24 h)');
		expect(timelineLine(sleep('2026-11-20T21:00:00Z', '2026-11-20T22:30:00Z'), 'UTC', METRIC)).toBe(
			'Sleep 21:00–22:30 (1 h 30 min)'
		);
	});

	it('shows nursing minutes per side and the running side while it runs', () => {
		expect(timelineLine(nursing({}), 'UTC', METRIC)).toBe('13:05 Nursing L 8 min · R 5 min');
		expect(timelineLine(nursing({ rightSeconds: 0 }), 'UTC', METRIC)).toBe('13:05 Nursing L 8 min');
		expect(timelineLine(nursing({ leftSeconds: 0, rightSeconds: 0 }), 'UTC', METRIC)).toBe(
			'13:05 Nursing'
		);
		expect(
			timelineLine(nursing({ endedAt: null, rightSeconds: 0, side: 'right' }), 'UTC', METRIC)
		).toBe('13:05 Nursing L 8 min, running on R');
	});

	it('shows pumping, a diaper and a solid meal', () => {
		const pumping = {
			kind: 'pumping',
			at: '2026-10-14T09:00:00Z',
			entry: {
				id: 'p1',
				startedAt: '2026-10-14T09:00:00Z',
				durationSeconds: 1200,
				volume: vol(90, 304),
				side: 'both',
				sides: [],
				note: null
			}
		} as TrackerTimelineItem;
		expect(timelineLine(pumping, 'UTC', METRIC)).toBe('09:00 Pumping 20 min, 90 ml (both)');
		expect(timelineLine(diaper('2026-10-14T07:30:00Z'), 'UTC', METRIC)).toBe('07:30 Diaper wet');
		const meal = (reaction: string | null) =>
			({
				kind: 'meals',
				at: '2026-10-14T12:15:00Z',
				entry: { id: 'm1', at: '2026-10-14T12:15:00Z', what: 'pear', reaction, note: null }
			}) as TrackerTimelineItem;
		expect(timelineLine(meal(null), 'UTC', METRIC)).toBe('12:15 Solids: pear');
		expect(timelineLine(meal('liked it'), 'UTC', METRIC)).toBe('12:15 Solids: pear · liked it');
	});
});

describe('timelineLine symptoms', () => {
	const symptom = (over: Record<string, unknown>): TrackerTimelineItem =>
		({
			kind: 'symptoms',
			at: '2026-10-14T09:12:00Z',
			entry: {
				id: 'y1',
				at: '2026-10-14T09:12:00Z',
				symptom: 'cough',
				severity: null,
				otherText: null,
				temperature: null,
				method: null,
				note: null,
				...over
			}
		}) as TrackerTimelineItem;
	const fever = (over: Record<string, unknown> = {}) =>
		symptom({
			symptom: 'fever',
			temperature: { cHundredths: 3840, fHundredths: 10112, typedUnit: 'c' },
			method: 'ear',
			...over
		});

	it('names a fever with its temperature in the parent unit and its method', () => {
		expect(timelineLine(fever(), 'UTC', METRIC)).toBe('09:12 Fever 38.4 °C (ear)');
		expect(timelineLine(fever(), 'UTC', IMPERIAL)).toBe('09:12 Fever 101.12 °F (ear)');
	});

	it('shows the column of the parent unit, so a temperature typed in °F reads back exactly', () => {
		const typedF = fever({
			temperature: { cHundredths: 3850, fHundredths: 10130, typedUnit: 'f' }
		});
		expect(timelineLine(typedF, 'UTC', IMPERIAL)).toBe('09:12 Fever 101.3 °F (ear)');
	});

	it('writes the method as the form names it', () => {
		expect(timelineLine(fever({ method: 'axillary' }), 'UTC', METRIC)).toBe(
			'09:12 Fever 38.4 °C (armpit)'
		);
	});

	it('names any other symptom, with its severity after a comma', () => {
		expect(timelineLine(symptom({}), 'UTC', METRIC)).toBe('09:12 Cough');
		expect(timelineLine(symptom({ severity: 'moderate' }), 'UTC', METRIC)).toBe(
			'09:12 Cough, moderate'
		);
		expect(timelineLine(symptom({ symptom: 'ear_pain', severity: 'mild' }), 'UTC', METRIC)).toBe(
			'09:12 Ear pain, mild'
		);
		expect(timelineLine(fever({ severity: 'severe' }), 'UTC', METRIC)).toBe(
			'09:12 Fever 38.4 °C (ear), severe'
		);
	});

	it('shows the text of an other symptom', () => {
		expect(timelineLine(symptom({ symptom: 'other', otherText: 'hiccups' }), 'UTC', METRIC)).toBe(
			'09:12 Other: hiccups'
		);
	});

	it('shows the clock of the kid zone', () => {
		expect(timelineLine(symptom({}), 'Europe/Madrid', METRIC)).toBe('11:12 Cough');
	});
});

describe('timelineLine pumping with sides', () => {
	const sided = (side: string, sides: [string, number, number][]) =>
		({
			kind: 'pumping',
			at: '2026-10-14T09:00:00Z',
			entry: {
				id: 'p1',
				startedAt: '2026-10-14T09:00:00Z',
				durationSeconds: 1200,
				volume: vol(
					sides.reduce((n, s) => n + s[1], 0),
					sides.reduce((n, s) => n + s[2], 0)
				),
				side,
				sides: sides.map(([s, ml, floz]) => ({ side: s, volume: vol(ml, floz) })),
				note: null
			}
		}) as TrackerTimelineItem;

	it('gives each side its own volume in the parent unit instead of the side name', () => {
		const both = sided('both', [
			['left', 60, 203],
			['right', 50, 169]
		]);
		expect(timelineLine(both, 'UTC', METRIC)).toBe(
			'09:00 Pumping 20 min, 110 ml (L 60 ml · R 50 ml)'
		);
		expect(timelineLine(both, 'UTC', IMPERIAL)).toBe(
			'09:00 Pumping 20 min, 3.72 fl oz (L 2.03 fl oz · R 1.69 fl oz)'
		);
	});

	it('a single side shows that side alone', () => {
		expect(timelineLine(sided('left', [['left', 60, 203]]), 'UTC', METRIC)).toBe(
			'09:00 Pumping 20 min, 60 ml (L 60 ml)'
		);
		expect(timelineLine(sided('right', [['right', 50, 169]]), 'UTC', METRIC)).toBe(
			'09:00 Pumping 20 min, 50 ml (R 50 ml)'
		);
	});
});

const visitItem = (startsAt: string, over: Partial<VisitRow> = {}): TimelineItem => ({
	kind: 'visit',
	at: startsAt,
	entry: {
		id: `v-${startsAt}`,
		note: null,
		startsAt,
		durationMin: 30,
		kind: 'sick',
		status: 'past',
		cancelledAt: null,
		provider: null,
		location: null,
		outcomeNotes: null,
		revision: 0,
		linked: { growth: 0, vaccinations: 0, medications: 0 },
		...over
	} as VisitRow
});

describe('visitTimelineLine', () => {
	it('names a visit by its kind, with Cancelled as its detail only when it is cancelled', () => {
		expect(visitTimelineLine({ kind: 'sick', status: 'past' })).toEqual({
			title: 'Sick visit',
			detail: null
		});
		expect(visitTimelineLine({ kind: 'well_child', status: 'upcoming' })).toEqual({
			title: 'Well-child visit',
			detail: null
		});
		expect(visitTimelineLine({ kind: 'dental', status: 'cancelled' })).toEqual({
			title: 'Dental visit',
			detail: 'Cancelled'
		});
	});
});

describe('groupByDay', () => {
	it('groups consecutive entries by the kid-local date and keeps the order', () => {
		const items = [
			diaper('2026-03-29T21:30:00Z', 'dirty'),
			diaper('2026-03-28T23:30:00Z'),
			diaper('2026-03-28T21:00:00Z')
		];
		const madrid = groupByDay(items, 'Europe/Madrid');
		expect(madrid.map((g) => [g.date, g.items.length])).toEqual([
			['2026-03-29', 2],
			['2026-03-28', 1]
		]);
		expect(groupByDay(items, 'UTC').map((g) => [g.date, g.items.length])).toEqual([
			['2026-03-29', 1],
			['2026-03-28', 2]
		]);
		expect(groupByDay([], 'UTC')).toEqual([]);
	});

	it('puts a visit under the kid-local day it starts on, among the tracker entries', () => {
		const items = [
			diaper('2026-03-29T21:30:00Z'),
			visitItem('2026-03-28T23:30:00Z'),
			diaper('2026-03-28T21:00:00Z')
		];
		expect(
			groupByDay(items, 'Europe/Madrid').map((g) => [g.date, g.items.map((i) => i.kind)])
		).toEqual([
			['2026-03-29', ['diapers', 'visit']],
			['2026-03-28', ['diapers']]
		]);
	});
});

describe('formatDayHeading', () => {
	it('writes the weekday and the date without the locale or the server zone', () => {
		expect(formatDayHeading('2026-03-29')).toBe('Sun 29 Mar 2026');
		expect(formatDayHeading('2026-10-14')).toBe('Wed 14 Oct 2026');
	});
});

const ALL: TrackerKind[] = ['bottles', 'nursing', 'pumping', 'sleep', 'diapers', 'meals'];
const ctx = { zone: 'UTC', today: '2026-11-22', defaults: ALL };
const params = (s: string) => new URLSearchParams(s);

describe('readFilter', () => {
	it('defaults to the kid’s today and every enabled tracker, sending no types', () => {
		const r = readFilter(params(''), ctx);
		expect(r.error).toBeNull();
		expect(r.filter).toEqual({
			from: '2026-11-22',
			to: '2026-11-22',
			kinds: ALL,
			narrowed: false,
			cursor: null
		});
		expect(r.query).toEqual({
			from: '2026-11-22T00:00:00Z',
			to: '2026-11-23T00:00:00Z',
			types: undefined,
			cursor: undefined
		});
	});

	it('converts the dates to UTC bounds with the kid zone, on a 23 and a 25 hour day', () => {
		const spring = readFilter(params('from=2026-03-29&to=2026-03-29'), {
			...ctx,
			zone: 'Europe/Madrid'
		});
		expect(spring.query).toMatchObject({
			from: '2026-03-28T23:00:00Z',
			to: '2026-03-29T22:00:00Z'
		});
		const autumn = readFilter(params('from=2026-10-25&to=2026-10-25'), {
			...ctx,
			zone: 'Europe/Madrid'
		});
		expect(autumn.query).toMatchObject({
			from: '2026-10-24T22:00:00Z',
			to: '2026-10-25T23:00:00Z'
		});
		const range = readFilter(params('from=2026-12-30&to=2027-01-01'), ctx);
		expect(range.query).toMatchObject({
			from: '2026-12-30T00:00:00Z',
			to: '2027-01-02T00:00:00Z'
		});
	});

	it('treats a blank or missing date as today', () => {
		const r = readFilter(params('from=&to=2026-11-24'), ctx);
		expect(r.filter).toMatchObject({ from: '2026-11-22', to: '2026-11-24' });
	});

	it('names the field of an invalid date, a reversed range and a range over the limit', () => {
		const bad = ['garbage', '2026-02-30', '2026-13-01', '22-11-2026', '2026-11-22T10:00'];
		for (const v of bad) {
			expect(readFilter(params(`from=${v}`), ctx)).toMatchObject({
				query: null,
				error: { field: 'from', message: 'Enter a valid date' }
			});
			expect(readFilter(params(`to=${v}`), ctx)).toMatchObject({
				query: null,
				error: { field: 'to', message: 'Enter a valid date' }
			});
		}
		expect(readFilter(params('from=2026-11-20&to=2026-11-19'), ctx)).toMatchObject({
			query: null,
			error: { field: 'to', message: 'The end date is before the start date' }
		});
		expect(SUMMARY_MAX_DAYS).toBe(93);
		expect(readFilter(params('from=2026-01-01&to=2026-04-03'), ctx).error).toBeNull();
		expect(readFilter(params('from=2026-01-01&to=2026-04-04'), ctx)).toMatchObject({
			query: null,
			error: { field: 'to', message: 'Pick at most 93 days' }
		});
	});

	it('keeps the typed values in the filter when it is refused', () => {
		const r = readFilter(params('from=2026-11-20&to=2026-11-19&types=bottle'), ctx);
		expect(r.filter).toMatchObject({ from: '2026-11-20', to: '2026-11-19', kinds: ['bottles'] });
	});

	it('sends types only when the selection differs from the enabled trackers', () => {
		const one = readFilter(params('types=sleep&types=bottle'), ctx);
		expect(one.filter).toMatchObject({ kinds: ['bottles', 'sleep'], narrowed: true });
		expect(one.query?.types).toEqual(['bottle', 'sleep']);
		const all = readFilter(params(ALL.map((k) => `types=${k.replace(/s$/, '')}`).join('&')), ctx);
		expect(all.query?.types).toBeUndefined();
		const off = { ...ctx, defaults: ['sleep', 'diapers'] as TrackerKind[] };
		expect(readFilter(params(''), off).filter).toMatchObject({
			kinds: ['sleep', 'diapers'],
			narrowed: false
		});
		expect(readFilter(params('types=diaper&types=sleep'), off).query?.types).toBeUndefined();
		const extra = readFilter(params('types=diaper&types=sleep&types=bottle'), off);
		expect(extra.query?.types).toEqual(['bottle', 'sleep', 'diaper']);
	});

	it('reads a comma list, drops unknown names and falls back to the default', () => {
		expect(readFilter(params('types=bottle,diaper'), ctx).query?.types).toEqual([
			'bottle',
			'diaper'
		]);
		expect(readFilter(params('types=bottle,nope'), ctx).query?.types).toEqual(['bottle']);
		const none = readFilter(params('types=nope&types='), ctx);
		expect(none.filter).toMatchObject({ kinds: ALL, narrowed: false });
		expect(none.query?.types).toBeUndefined();
	});

	it('reads the visit type like a tracker, after the trackers', () => {
		const withVisits = { ...ctx, defaults: [...ALL, 'visit'] as TimelineKind[] };
		expect(readFilter(params(''), withVisits).filter).toMatchObject({
			kinds: [...ALL, 'visit'],
			narrowed: false
		});
		const only = readFilter(params('types=visit'), withVisits);
		expect(only.filter).toMatchObject({ kinds: ['visit'], narrowed: true });
		expect(only.query?.types).toEqual(['visit']);
		expect(readFilter(params('types=visit,bottle'), ctx).query?.types).toEqual(['bottle', 'visit']);
		expect(readFilter(params('types=visits'), ctx).filter.kinds).toEqual(ALL);
	});

	it('passes the cursor on untouched', () => {
		const r = readFilter(params('cursor=MTIzOmFiYw'), ctx);
		expect(r.filter.cursor).toBe('MTIzOmFiYw');
		expect(r.query?.cursor).toBe('MTIzOmFiYw');
		expect(readFilter(params('cursor='), ctx).filter.cursor).toBeNull();
	});
});

describe('timelineSearch', () => {
	it('writes the dates, a narrowed type list and the cursor, in the form’s own shape', () => {
		const base = {
			from: '2026-10-14',
			to: '2026-10-15',
			kinds: ALL,
			narrowed: false,
			cursor: null
		};
		expect(timelineSearch(base)).toBe('from=2026-10-14&to=2026-10-15');
		expect(
			timelineSearch({ ...base, kinds: ['bottles', 'meals'], narrowed: true, cursor: 'a b' })
		).toBe('from=2026-10-14&to=2026-10-15&types=bottle&types=meal&cursor=a+b');
		expect(timelineSearch(base, 'next')).toBe('from=2026-10-14&to=2026-10-15&cursor=next');
		expect(timelineSearch({ ...base, cursor: 'old' }, null)).toBe('from=2026-10-14&to=2026-10-15');
		expect(timelineSearch({ ...base, kinds: ['sleep', 'visit'], narrowed: true })).toBe(
			'from=2026-10-14&to=2026-10-15&types=sleep&types=visit'
		);
	});
});

describe('defaultKinds and shownModules', () => {
	const state = (module: ModuleId, enabled: boolean): ModuleState => ({
		module,
		enabled,
		default: enabled,
		overridden: false
	});

	it('follows the enabled modules that have a tracker, in tracker order', () => {
		const modules = [
			state('feeding', true),
			state('diapers', true),
			state('bottles', false),
			state('growth', true),
			state('sleep', true)
		];
		expect(defaultKinds(modules)).toEqual(['sleep', 'diapers', 'meals']);
		expect(defaultKinds([])).toEqual([]);
	});

	it('adds visits, last, while the Medical visits module is on', () => {
		expect(defaultKinds([state('visits', true), state('sleep', true)])).toEqual(['sleep', 'visit']);
		expect(defaultKinds([state('visits', false), state('sleep', true)])).toEqual(['sleep']);
	});

	it('maps shown trackers back to their modules', () => {
		expect([...shownModules(['meals', 'sleep'])].sort()).toEqual(['feeding', 'sleep']);
		expect([...shownModules(['visit', 'diapers'])].sort()).toEqual(['diapers', 'visits']);
	});
});

describe('weekDates', () => {
	it('is the seven days ending today, oldest first, across a month end', () => {
		expect(weekDates('2026-12-02')).toEqual([
			'2026-11-26',
			'2026-11-27',
			'2026-11-28',
			'2026-11-29',
			'2026-11-30',
			'2026-12-01',
			'2026-12-02'
		]);
	});
});

describe('weekCells', () => {
	const day = (date: string, over: Partial<DaySummary> = {}): DaySummary => ({
		date,
		bottles: { count: 0, ml: 0, floz_hundredths: 0 },
		nursing: { count: 0, seconds: 0 },
		pumping: { count: 0, ml: 0, floz_hundredths: 0 },
		diapers: { total: 0, wet: 0, dirty: 0, both: 0, dry: 0 },
		sleep: { seconds: 0 },
		meals: { count: 0 },
		...over
	});
	const days = [
		day('2026-11-23'),
		day('2026-11-24', {
			bottles: { count: 2, ml: 250, floz_hundredths: 845 },
			nursing: { count: 1, seconds: 26 * 60 },
			diapers: { total: 3, wet: 2, dirty: 1, both: 0, dry: 0 },
			sleep: { seconds: 5400 }
		}),
		day('2026-11-25', { meals: { count: 1 } })
	];
	it('names Today and Yesterday, counts every tracker and says what happened in the label', () => {
		const cells = weekCells(days, '2026-11-25', 'ml');
		expect(cells.map((c) => c.dow)).toEqual(['Mon', 'Yest.', 'Today']);
		expect(cells[1].label).toBe(
			'Yesterday: 2 bottles (250 ml) · 1 nursing (26 min) · 3 diapers · sleep 1 h 30 min'
		);
		expect(cells[0].label).toBe('Mon 23 Nov 2026: nothing logged');
		expect(cells[2].label).toBe('Today: 1 meal');
		// The summary has no session count, so a day with any sleep shows one sleep segment.
		expect(cells[1].segs).toEqual([
			{ type: 'bottle', count: 2 },
			{ type: 'nursing', count: 1 },
			{ type: 'sleep', count: 1 },
			{ type: 'diaper', count: 3 }
		]);
	});
	it('reads volumes in the parent unit', () => {
		expect(weekCells(days, '2026-11-25', 'floz')[1].label).toContain('2 bottles (8.45 fl oz)');
	});
});

describe('sinceLast', () => {
	const now = new Date('2026-11-25T12:00:00Z');
	const at = (minutes: number) => new Date(now.getTime() - minutes * 60_000).toISOString();
	const item = (kind: string, minutes: number, endedAt: number | null = null) =>
		({
			kind,
			at: at(minutes),
			entry: { id: kind, endedAt: endedAt === null ? null : at(endedAt) }
		}) as never;
	const none = { sleep: null, nursing: null };
	it('says when the last feed, diaper and sleep were; a feed is the newer of bottle and nursing', () => {
		const cells = sinceLast(
			{
				bottle: item('bottles', 94),
				nursing: item('nursing', 200, 40),
				diaper: item('diapers', 10),
				sleep: item('sleep', 300, 120)
			},
			none,
			now
		);
		expect(cells).toEqual([
			{ label: 'Last feed', value: '40 min ago' },
			{ label: 'Last diaper', value: '10 min ago' },
			{ label: 'Last sleep', value: '2 h ago' }
		]);
	});
	it('says Asleep for while a sleep runs, and None yet for a tracker with no entry', () => {
		const sleeping = { sleep: { startedAt: at(95) } as never, nursing: null };
		expect(sinceLast({ bottle: null, diaper: null, sleep: null }, sleeping, now)).toEqual([
			{ label: 'Last feed', value: 'None yet' },
			{ label: 'Last diaper', value: 'None yet' },
			{ label: 'Asleep for', value: '1 h 35 min' }
		]);
	});
	it('leaves out a cell whose trackers are off', () => {
		expect(sinceLast({ diaper: item('diapers', 10) }, none, now)).toEqual([
			{ label: 'Last diaper', value: '10 min ago' }
		]);
		expect(sinceLast({}, none, now)).toEqual([]);
	});
});
