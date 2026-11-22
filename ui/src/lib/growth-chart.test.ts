import { describe, expect, it } from 'vitest';
import {
	buildChart,
	lacksTable,
	linePath,
	linearScale,
	niceDomain,
	ordinal,
	scoringMonths,
	spreadLabels,
	ticks,
	valueAt
} from './growth-chart';
import type { CurvePoint, Curves, GrowthRow, Units } from './types';

const METRIC: Units = { weight: 'kg', length: 'cm', volume: 'ml', temp: 'c' };
const IMPERIAL: Units = { weight: 'lb', length: 'in', volume: 'floz', temp: 'f' };

const curves = (over: Partial<Curves> = {}): Curves => ({
	source: 'who',
	measure: 'weight',
	sex: 'female',
	covered: true,
	reason: null,
	fromMonths: 0,
	toMonths: 24,
	correctedAgeAxis: false,
	lines: [3, 50, 97].map((percentile, i) => ({
		percentile,
		points: [0, 12, 24].map((ageMonths) => ({
			ageMonths,
			metric: 3 + i + ageMonths / 4,
			imperial: (3 + i + ageMonths / 4) * 2,
			gap: false
		}))
	})),
	...over
});

const row = (over: Partial<GrowthRow> = {}): GrowthRow => ({
	id: 'g1',
	measuredOn: '2026-03-29',
	weight: { grams: 7000, ozHundredths: 24692, typedUnit: 'kg' },
	length: { mm: 650, inHundredths: 2559, typedUnit: 'cm' },
	head: null,
	note: null,
	ageDays: 183,
	correctedDays: null,
	scores: {
		weight: { scored: true, z: 0.1, percentile: 54 },
		length: { scored: true, z: 0.1, percentile: 54 },
		head: null,
		bmi: { scored: true, z: 0.1, percentile: 54, value: 16.57 }
	},
	...over
});

const finite = (s: string) => !/NaN|Infinity/.test(s);

describe('linearScale', () => {
	it('maps the domain onto the range, inverted ranges included', () => {
		const s = linearScale({ min: 0, max: 10 }, [100, 0]);
		expect(s(0)).toBe(100);
		expect(s(10)).toBe(0);
		expect(s(5)).toBe(50);
	});

	it('puts every value in the middle of an empty domain', () => {
		const s = linearScale({ min: 4, max: 4 }, [0, 200]);
		expect(s(4)).toBe(100);
		expect(s(9)).toBe(100);
	});
});

describe('ticks', () => {
	it('chooses a 1, 2 or 5 step and stays inside the range', () => {
		expect(ticks(0, 24, 6)).toEqual([0, 5, 10, 15, 20]);
		expect(ticks(2.5, 9.5, 4)).toEqual([4, 6, 8]);
		expect(ticks(0, 1, 5)).toEqual([0, 0.2, 0.4, 0.6, 0.8, 1]);
	});

	it('is one tick for equal ends and none for a broken range', () => {
		expect(ticks(7, 7, 5)).toEqual([7]);
		expect(ticks(Number.NaN, 1, 5)).toEqual([]);
		expect(ticks(3, 1, 5)).toEqual([]);
	});
});

describe('niceDomain', () => {
	it('widens to whole steps', () => {
		expect(niceDomain([3.2, 11.7], 5)).toEqual({ min: 2, max: 12 });
	});

	it('opens equal values into a range around them', () => {
		const d = niceDomain([7, 7], 5);
		expect(d.min).toBeLessThan(7);
		expect(d.max).toBeGreaterThan(7);
		const zero = niceDomain([0, 0], 5);
		expect(zero.min).toBeLessThan(0);
		expect(zero.max).toBeGreaterThan(0);
	});

	it('falls back to 0..1 with no values', () => {
		expect(niceDomain([], 5)).toEqual({ min: 0, max: 1 });
	});
});

describe('linePath', () => {
	const id = (v: number) => v;

	it('is empty for no points and a lone move for one', () => {
		expect(linePath([], id, id)).toBe('');
		expect(linePath([{ x: 1, y: 2 }], id, id)).toBe('M1 2');
	});

	it('joins points and starts a new subpath at a gap', () => {
		expect(
			linePath(
				[
					{ x: 0, y: 0 },
					{ x: 1, y: 1 },
					{ x: 2, y: 5, gap: true },
					{ x: 3, y: 6 }
				],
				id,
				id
			)
		).toBe('M0 0L1 1M2 5L3 6');
	});

	it('rounds coordinates to two decimals', () => {
		expect(linePath([{ x: 1 / 3, y: 2 / 3 }], id, id)).toBe('M0.33 0.67');
	});
});

describe('valueAt', () => {
	const pts: CurvePoint[] = [
		{ ageMonths: 0, metric: 2, imperial: null, gap: false },
		{ ageMonths: 2, metric: 4, imperial: null, gap: false },
		{ ageMonths: 4, metric: 9, imperial: null, gap: true }
	];
	const metric = (p: CurvePoint) => p.metric;

	it('interpolates inside a segment and is exact on a point', () => {
		expect(valueAt(pts, 1, metric)).toBe(3);
		expect(valueAt(pts, 2, metric)).toBe(4);
		expect(valueAt(pts, 4, metric)).toBe(9);
	});

	it('is null outside the line and across a gap', () => {
		expect(valueAt(pts, -1, metric)).toBeNull();
		expect(valueAt(pts, 5, metric)).toBeNull();
		expect(valueAt(pts, 3, metric)).toBeNull();
	});
});

describe('spreadLabels', () => {
	it('keeps labels that already fit', () => {
		expect(spreadLabels([10, 50, 90], 12, 0, 100)).toEqual([10, 50, 90]);
	});

	it('pushes crowded labels apart in their order and keeps them inside', () => {
		const ys = spreadLabels([50, 49, 51, 50], 12, 0, 100);
		const sorted = [...ys].sort((a, b) => a - b);
		for (let i = 1; i < sorted.length; i++)
			expect(sorted[i] - sorted[i - 1]).toBeGreaterThanOrEqual(12);
		expect(ys[1]).toBeLessThan(ys[0]);
		expect(ys[2]).toBeGreaterThan(ys[3]);
		const low = spreadLabels([98, 99, 100], 12, 0, 100);
		expect(Math.max(...low)).toBeLessThanOrEqual(100);
	});
});

describe('ordinal', () => {
	it('names percentile lines', () => {
		expect([3, 5, 15, 50, 85, 97, 1, 2, 11, 12, 13, 21].map(ordinal)).toEqual([
			'3rd',
			'5th',
			'15th',
			'50th',
			'85th',
			'97th',
			'1st',
			'2nd',
			'11th',
			'12th',
			'13th',
			'21st'
		]);
	});
});

describe('scoringMonths', () => {
	it('uses the corrected days when set and only the day count (no dates, no DST)', () => {
		expect(scoringMonths(row({ ageDays: 61, correctedDays: null }))).toBe(61 / 30.4375);
		expect(scoringMonths(row({ ageDays: 120, correctedDays: 30 }))).toBe(30 / 30.4375);
		expect(scoringMonths(row({ measuredOn: '2026-03-29', ageDays: 10 }))).toBe(
			scoringMonths(row({ measuredOn: '2026-10-25', ageDays: 10 }))
		);
	});

	it('is null before the expected term', () => {
		expect(scoringMonths(row({ ageDays: 10, correctedDays: -20 }))).toBeNull();
	});
});

describe('lacksTable', () => {
	const notCovered = { scored: false as const, reason: 'not_covered' as const };
	const beforeTerm = { scored: false as const, reason: 'before_term' as const };

	it('is true when the curves are not covered', () => {
		expect(lacksTable(false, [], 'weight')).toBe(true);
	});

	it('is true when one row of the shown measure has no table', () => {
		const rows = [row(), row({ id: 'g2', scores: { ...row().scores, head: notCovered } })];
		expect(lacksTable(true, rows, 'head')).toBe(true);
		expect(lacksTable(true, rows, 'weight')).toBe(false);
	});

	it('is false when every row is scored, before term or without the measure', () => {
		const rows = [
			row(),
			row({ id: 'g2', scores: { ...row().scores, weight: beforeTerm } }),
			row({ id: 'g3', weight: null, scores: { ...row().scores, weight: null } })
		];
		expect(lacksTable(true, rows, 'weight')).toBe(false);
		expect(lacksTable(true, [], 'bmi')).toBe(false);
	});
});

describe('buildChart', () => {
	it('draws one labelled, dashed path per percentile line and the kid’s points', () => {
		const c = buildChart({ curves: curves(), rows: [row()], measure: 'weight', units: METRIC });
		expect(c.lines.map((l) => l.label)).toEqual(['3rd', '50th', '97th']);
		expect(c.lines.every((l) => l.d.startsWith('M'))).toBe(true);
		expect(c.lines[1].dash).toBe('');
		expect(c.lines[0].dash).not.toBe('');
		expect(c.lines[0].dash).toBe(c.lines[2].dash);
		expect(c.points).toHaveLength(1);
		expect(c.points[0].y).toBe(7);
		expect(c.xLabel).toBe('Age (months)');
		expect(c.yLabel).toBe('Weight (kg)');
	});

	it('reads the imperial column of the curves and the row, and metric for BMI', () => {
		const lb = buildChart({ curves: curves(), rows: [row()], measure: 'weight', units: IMPERIAL });
		expect(lb.points[0].y).toBeCloseTo(24692 / 1600, 10);
		expect(lb.table.lines[0].values[0]).toBe(6);
		expect(lb.yLabel).toBe('Weight (lb)');
		const len = buildChart({
			curves: curves({ measure: 'length' }),
			rows: [row()],
			measure: 'length',
			units: IMPERIAL
		});
		expect(len.points[0].y).toBe(25.59);
		const bmiCurves = curves({
			measure: 'bmi',
			lines: curves().lines.map((l) => ({
				...l,
				points: l.points.map((p) => ({ ...p, imperial: null }))
			}))
		});
		const bmi = buildChart({ curves: bmiCurves, rows: [row()], measure: 'bmi', units: IMPERIAL });
		expect(bmi.points[0].y).toBe(16.57);
		expect(bmi.table.lines[0].values[0]).toBe(3);
		expect(bmi.yLabel).toBe('BMI (kg/m²)');
	});

	it('labels the corrected age axis and leaves out rows before term or without the measure', () => {
		const c = buildChart({
			curves: curves({ correctedAgeAxis: true }),
			rows: [
				row({ id: 'a', ageDays: 200, correctedDays: 150 }),
				row({ id: 'b', ageDays: 20, correctedDays: -30 }),
				row({ id: 'c', weight: null })
			],
			measure: 'weight',
			units: METRIC
		});
		expect(c.xLabel).toBe('Corrected age (months)');
		expect(c.points.map((p) => p.id)).toEqual(['a']);
		expect(c.points[0].x).toBe(150 / 30.4375);
	});

	it('joins the points in age order', () => {
		const c = buildChart({
			curves: curves(),
			rows: [row({ id: 'late', ageDays: 300 }), row({ id: 'early', ageDays: 30 })],
			measure: 'weight',
			units: METRIC
		});
		expect(c.points.map((p) => p.id)).toEqual(['early', 'late']);
		expect(c.path.match(/M/g)).toHaveLength(1);
		expect(c.path.match(/L/g)).toHaveLength(1);
	});

	it('breaks a percentile line at a gap', () => {
		const gapped = curves();
		gapped.lines[0].points[2].gap = true;
		const c = buildChart({ curves: gapped, rows: [], measure: 'weight', units: METRIC });
		expect(c.lines[0].d.match(/M/g)).toHaveLength(2);
	});

	it('is a valid empty chart with no curves and no points', () => {
		const c = buildChart({
			curves: curves({ covered: false, reason: 'not_covered', lines: [] }),
			rows: [],
			measure: 'head',
			units: METRIC
		});
		expect(c.lines).toEqual([]);
		expect(c.points).toEqual([]);
		expect(c.path).toBe('');
		expect(c.xTicks.length).toBeGreaterThan(0);
		expect(c.yTicks.length).toBeGreaterThan(0);
		expect(finite(JSON.stringify(c))).toBe(true);
	});

	it('handles one point with equal values and no curves', () => {
		const c = buildChart({
			curves: curves({ covered: false, lines: [], fromMonths: 6, toMonths: 6 }),
			rows: [row({ ageDays: 183 })],
			measure: 'weight',
			units: METRIC
		});
		expect(c.points).toHaveLength(1);
		const p = c.points[0];
		expect(p.cx).toBeGreaterThanOrEqual(c.plot.left);
		expect(p.cx).toBeLessThanOrEqual(c.plot.right);
		expect(p.cy).toBeGreaterThanOrEqual(c.plot.top);
		expect(p.cy).toBeLessThanOrEqual(c.plot.bottom);
		expect(finite(JSON.stringify(c))).toBe(true);
	});

	it('keeps an extreme value inside the plot by widening the axis', () => {
		const c = buildChart({
			curves: curves(),
			rows: [row({ weight: { grams: 250000, ozHundredths: 881849, typedUnit: 'kg' } })],
			measure: 'weight',
			units: METRIC
		});
		const p = c.points[0];
		expect(p.cy).toBeGreaterThanOrEqual(c.plot.top);
		expect(c.yTicks.at(-1)!.value).toBeGreaterThanOrEqual(250);
	});

	it('gives the table each line’s value at each x tick', () => {
		const c = buildChart({ curves: curves(), rows: [], measure: 'weight', units: METRIC });
		expect(c.table.ticks).toEqual(c.xTicks.map((t) => t.value));
		expect(c.table.lines.map((l) => l.label)).toEqual(['3rd', '50th', '97th']);
		expect(c.table.lines[1].values[0]).toBe(4);
		expect(c.table.lines[1].values[c.table.ticks.indexOf(20)]).toBe(9);
	});
	it('drops lines with fewer than two points, with their labels and table columns', () => {
		const one = curves({
			fromMonths: 24,
			toMonths: 24,
			lines: [3, 50].map((percentile) => ({
				percentile,
				points: [{ ageMonths: 24, metric: 15, imperial: null, gap: false }]
			}))
		});
		const c = buildChart({ curves: one, rows: [row()], measure: 'bmi', units: METRIC });
		expect(c.lines).toEqual([]);
		expect(c.table.lines).toEqual([]);
		expect(c.points).toHaveLength(1);

		const mixed = curves();
		mixed.lines[0].points = mixed.lines[0].points.slice(0, 1);
		const m = buildChart({ curves: mixed, rows: [], measure: 'weight', units: METRIC });
		expect(m.lines.map((l) => l.label)).toEqual(['50th', '97th']);
		expect(m.table.lines.map((l) => l.label)).toEqual(['50th', '97th']);
		expect(m.lines[0].dash).toBe('');
	});
});
