import { describe, expect, it } from 'vitest';
import { SUMMARY_MAX_DAYS } from './constants';
import {
	defaultKinds,
	formatDayHeading,
	groupByDay,
	readFilter,
	shownModules,
	timelineLine,
	timelineSearch
} from './timeline';
import type { ModuleId, ModuleState, TimelineItem, TrackerKind } from './types';

const vol = (ml: number, hundredths: number) => ({
	ml,
	floz_hundredths: hundredths,
	typed_unit: 'ml' as const
});
const bottle = (at: string, contents: string[], ml = 120): TimelineItem =>
	({
		kind: 'bottles',
		at,
		entry: { id: 'b1', at, contents, volume: vol(ml, Math.round(ml / 0.295735)), note: null }
	}) as TimelineItem;
const sleep = (startedAt: string, endedAt: string | null): TimelineItem =>
	({
		kind: 'sleep',
		at: startedAt,
		entry: { id: 's1', startedAt, endedAt, note: null }
	}) as TimelineItem;
const nursing = (over: Record<string, unknown>): TimelineItem =>
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
			...over
		}
	}) as TimelineItem;
const diaper = (at: string, kind = 'wet'): TimelineItem =>
	({ kind: 'diapers', at, entry: { id: `d-${at}`, at, kind, note: null } }) as TimelineItem;

describe('timelineLine', () => {
	it('names a bottle with its volume in the parent unit and its contents', () => {
		const b = bottle('2026-10-14T13:05:00Z', ['breast_milk', 'formula']);
		expect(timelineLine(b, 'UTC', 'ml')).toBe('13:05 Bottle 120 ml, breast milk + formula');
		expect(timelineLine(b, 'UTC', 'floz')).toBe('13:05 Bottle 4.06 fl oz, breast milk + formula');
		expect(timelineLine(bottle('2026-10-14T13:05:00Z', ['cow_milk', 'other']), 'UTC', 'ml')).toBe(
			"13:05 Bottle 120 ml, cow's milk + other"
		);
	});

	it('shows the clock of the kid zone, never UTC', () => {
		expect(timelineLine(bottle('2026-03-28T23:30:00Z', ['formula']), 'Europe/Madrid', 'ml')).toBe(
			'00:30 Bottle 120 ml, formula'
		);
		expect(timelineLine(bottle('2026-03-29T21:30:00Z', ['formula']), 'Europe/Madrid', 'ml')).toBe(
			'23:30 Bottle 120 ml, formula'
		);
	});

	it('shows a finished sleep as a range with its length, a running one as since', () => {
		expect(timelineLine(sleep('2026-10-14T13:05:00Z', '2026-10-14T14:40:00Z'), 'UTC', 'ml')).toBe(
			'Sleep 13:05–14:40 (1 h 35 min)'
		);
		expect(timelineLine(sleep('2026-10-14T13:05:00Z', null), 'UTC', 'ml')).toBe(
			'Sleep since 13:05'
		);
		expect(
			timelineLine(sleep('2026-03-28T23:30:00Z', '2026-03-29T00:30:00Z'), 'Europe/Madrid', 'ml')
		).toBe('Sleep 00:30–01:30 (1 h)');
	});

	it('shows nursing minutes per side and the running side while it runs', () => {
		expect(timelineLine(nursing({}), 'UTC', 'ml')).toBe('13:05 Nursing L 8 min · R 5 min');
		expect(timelineLine(nursing({ rightSeconds: 0 }), 'UTC', 'ml')).toBe('13:05 Nursing L 8 min');
		expect(timelineLine(nursing({ leftSeconds: 0, rightSeconds: 0 }), 'UTC', 'ml')).toBe(
			'13:05 Nursing'
		);
		expect(
			timelineLine(nursing({ endedAt: null, rightSeconds: 0, side: 'right' }), 'UTC', 'ml')
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
				note: null
			}
		} as TimelineItem;
		expect(timelineLine(pumping, 'UTC', 'ml')).toBe('09:00 Pumping 20 min, 90 ml (both)');
		expect(timelineLine(diaper('2026-10-14T07:30:00Z'), 'UTC', 'ml')).toBe('07:30 Diaper wet');
		const meal = (reaction: string | null) =>
			({
				kind: 'meals',
				at: '2026-10-14T12:15:00Z',
				entry: { id: 'm1', at: '2026-10-14T12:15:00Z', what: 'pear', reaction, note: null }
			}) as TimelineItem;
		expect(timelineLine(meal(null), 'UTC', 'ml')).toBe('12:15 Solids: pear');
		expect(timelineLine(meal('liked it'), 'UTC', 'ml')).toBe('12:15 Solids: pear · liked it');
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

	it('maps shown trackers back to their modules', () => {
		expect([...shownModules(['meals', 'sleep'])].sort()).toEqual(['feeding', 'sleep']);
	});
});
