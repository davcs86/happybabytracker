import {
	DAYS_PER_MONTH,
	GRAMS_PER_KG,
	HUNDREDTHS_PER_INCH,
	HUNDREDTHS_PER_OZ,
	MM_PER_CM,
	OZ_PER_LB
} from './constants';
import type { CurvePoint, Curves, GrowthRow, Measure, Units } from './types';

// viewBox units; the SVG scales to its box, so these fix proportions and text size only.
export const CHART = { width: 340, height: 250, left: 36, right: 38, top: 24, bottom: 40 };
const X_TICKS = 6;
const Y_TICKS = 5;
const LABEL_GAP = 13;
// Line styles by distance from the median line: dash pattern, never colour alone.
const DASHES = ['', '6 3', '2 3', '8 3 2 3', '1 3'];
const COORD_DECIMALS = 2;

type Domain = { min: number; max: number };
type Pt = { x: number; y: number; gap?: boolean };
export type Tick = { value: number; pos: number };

export const MEASURE_LABEL: Record<Measure, string> = {
	weight: 'Weight',
	length: 'Length',
	head: 'Head circumference',
	bmi: 'BMI'
};

export function linearScale(d: Domain, range: [number, number]) {
	const [r0, r1] = range;
	if (d.max === d.min) return () => (r0 + r1) / 2;
	return (v: number) => r0 + ((v - d.min) / (d.max - d.min)) * (r1 - r0);
}

function niceStep(span: number, target: number) {
	const raw = span / target;
	const mag = 10 ** Math.floor(Math.log10(raw));
	const norm = raw / mag;
	return (norm <= 1 ? 1 : norm <= 2 ? 2 : norm <= 5 ? 5 : 10) * mag;
}

const stepDecimals = (step: number) => Math.max(0, -Math.floor(Math.log10(step)));
const roundTo = (v: number, decimals: number) => Number(v.toFixed(decimals));

// Multiples of a 1, 2 or 5 step inside [min, max].
export function ticks(min: number, max: number, target: number): number[] {
	if (!Number.isFinite(min) || !Number.isFinite(max) || min > max) return [];
	if (min === max) return [min];
	const step = niceStep(max - min, target);
	const decimals = stepDecimals(step);
	const out: number[] = [];
	for (let k = Math.ceil(min / step); k * step <= max + step * 1e-9; k++) {
		out.push(roundTo(k * step, decimals));
	}
	return out;
}

// The extent widened to whole steps; equal values open by one step each side.
export function niceDomain(values: number[], target: number): Domain {
	if (values.length === 0) return { min: 0, max: 1 };
	let min = Math.min(...values);
	let max = Math.max(...values);
	if (min === max) {
		const pad = niceStep(Math.abs(min) || 1, target);
		min -= pad;
		max += pad;
	}
	const step = niceStep(max - min, target);
	const decimals = stepDecimals(step);
	return {
		min: roundTo(Math.floor(min / step) * step, decimals),
		max: roundTo(Math.ceil(max / step) * step, decimals)
	};
}

const coord = (v: number) => roundTo(v, COORD_DECIMALS);

// A point with gap starts a new subpath: a line is never joined across a table seam.
export function linePath(points: Pt[], sx: (v: number) => number, sy: (v: number) => number) {
	return points
		.map((p, i) => `${i === 0 || p.gap ? 'M' : 'L'}${coord(sx(p.x))} ${coord(sy(p.y))}`)
		.join('');
}

// Linear between neighbours; null outside the line or across a gap.
export function valueAt(points: CurvePoint[], x: number, value: (p: CurvePoint) => number) {
	for (let i = 0; i < points.length; i++) {
		const p = points[i];
		if (p.ageMonths === x) return value(p);
		const next = points[i + 1];
		if (next && p.ageMonths < x && x < next.ageMonths) {
			if (next.gap) return null;
			const t = (x - p.ageMonths) / (next.ageMonths - p.ageMonths);
			return value(p) + t * (value(next) - value(p));
		}
	}
	return null;
}

// Label positions at least gap apart, in [top, bottom] where they fit, in the input order.
export function spreadLabels(ys: number[], gap: number, top: number, bottom: number) {
	const order = ys.map((_, i) => i).sort((a, b) => ys[a] - ys[b]);
	const placed = order.map((i) => ys[i]);
	for (let k = 1; k < placed.length; k++) placed[k] = Math.max(placed[k], placed[k - 1] + gap);
	const over = (placed.at(-1) ?? bottom) - bottom;
	if (over > 0) {
		placed[placed.length - 1] -= over;
		for (let k = placed.length - 2; k >= 0; k--) {
			placed[k] = Math.min(placed[k], placed[k + 1] - gap);
		}
	}
	const under = top - (placed[0] ?? top);
	if (under > 0) for (let k = 0; k < placed.length; k++) placed[k] += under;
	const out: number[] = [];
	order.forEach((i, k) => (out[i] = placed[k]));
	return out;
}

export function ordinal(n: number) {
	const tens = n % 100;
	if (tens >= 11 && tens <= 13) return `${n}th`;
	return `${n}${{ 1: 'st', 2: 'nd', 3: 'rd' }[n % 10] ?? 'th'}`;
}

// The age the row is scored at, in months; null before the expected term (never plotted).
export function scoringMonths(row: Pick<GrowthRow, 'ageDays' | 'correctedDays'>) {
	const days = row.correctedDays ?? row.ageDays;
	return days < 0 ? null : days / DAYS_PER_MONTH;
}

// The row's own column in the parent's unit; BMI has no imperial unit.
function rowValue(row: GrowthRow, measure: Measure, units: Pick<Units, 'weight' | 'length'>) {
	if (measure === 'bmi') return row.scores.bmi?.value ?? null;
	if (measure === 'weight') {
		if (!row.weight) return null;
		return units.weight === 'lb'
			? row.weight.ozHundredths / (HUNDREDTHS_PER_OZ * OZ_PER_LB)
			: row.weight.grams / GRAMS_PER_KG;
	}
	const l = measure === 'length' ? row.length : row.head;
	if (!l) return null;
	return units.length === 'in' ? l.inHundredths / HUNDREDTHS_PER_INCH : l.mm / MM_PER_CM;
}

export function unitLabel(measure: Measure, units: Pick<Units, 'weight' | 'length'>) {
	if (measure === 'bmi') return 'kg/m²';
	return measure === 'weight' ? units.weight : units.length;
}

// The sentence of spec 8.2: no curves at all, or a shown point the source has no table for.
export function lacksTable(covered: boolean, rows: GrowthRow[], measure: Measure) {
	return (
		!covered ||
		rows.some((r) => {
			const score = r.scores[measure];
			return score?.scored === false && score.reason === 'not_covered';
		})
	);
}

export type ChartInput = {
	curves: Curves;
	rows: GrowthRow[];
	measure: Measure;
	units: Pick<Units, 'weight' | 'length'>;
};

export function buildChart({ curves, rows, measure, units }: ChartInput) {
	// A one-point line draws nothing: shown as points without curves.
	const curveLines = curves.lines.filter((l) => l.points.length >= 2);
	const imperial = measure === 'weight' ? units.weight === 'lb' : units.length === 'in';
	const curveValue = (p: CurvePoint) => (imperial && p.imperial !== null ? p.imperial : p.metric);
	const points = rows
		.map((r) => ({
			id: r.id,
			measuredOn: r.measuredOn,
			x: scoringMonths(r),
			y: rowValue(r, measure, units),
			corrected: r.correctedDays !== null
		}))
		.filter(
			(p): p is { id: string; measuredOn: string; x: number; y: number; corrected: boolean } =>
				p.x !== null && p.y !== null
		)
		.sort((a, b) => a.x - b.x);

	const xs = [curves.fromMonths, curves.toMonths, ...points.map((p) => p.x)];
	const xDomain = { min: Math.min(...xs), max: Math.max(...xs) };
	const yDomain = niceDomain(
		[...curveLines.flatMap((l) => l.points.map(curveValue)), ...points.map((p) => p.y)],
		Y_TICKS
	);
	const plot = {
		left: CHART.left,
		right: CHART.width - CHART.right,
		top: CHART.top,
		bottom: CHART.height - CHART.bottom
	};
	const sx = linearScale(xDomain, [plot.left, plot.right]);
	const sy = linearScale(yDomain, [plot.bottom, plot.top]);
	const xTicks = ticks(xDomain.min, xDomain.max, X_TICKS);
	const yTicks = ticks(yDomain.min, yDomain.max, Y_TICKS);

	// Past the second birthday the axis is chronological, but earlier rows still plot at corrected age.
	const mixedAxis = !curves.correctedAgeAxis && points.some((p) => p.corrected);
	const xLabel = curves.correctedAgeAxis
		? 'Corrected age (months)'
		: mixedAxis
			? 'Age (months; corrected before the second birthday)'
			: 'Age (months)';

	const median = curveLines.findIndex((l) => l.percentile === 50);
	const centre = median >= 0 ? median : Math.floor(curveLines.length / 2);
	const ends = curveLines.map((l) => {
		const last = l.points.at(-1);
		return last ? sy(curveValue(last)) : plot.bottom;
	});
	const labelYs = spreadLabels(ends, LABEL_GAP, plot.top, plot.bottom);
	const lines = curveLines.map((l, i) => ({
		percentile: l.percentile,
		label: ordinal(l.percentile),
		d: linePath(
			l.points.map((p) => ({ x: p.ageMonths, y: curveValue(p), gap: p.gap })),
			sx,
			sy
		),
		dash: DASHES[Math.min(Math.abs(i - centre), DASHES.length - 1)],
		labelY: coord(labelYs[i])
	}));

	return {
		width: CHART.width,
		height: CHART.height,
		plot,
		xTicks: xTicks.map((value) => ({ value, pos: coord(sx(value)) })),
		yTicks: yTicks.map((value) => ({ value, pos: coord(sy(value)) })),
		lines,
		path: linePath(points, sx, sy),
		points: points.map((p) => ({ ...p, cx: coord(sx(p.x)), cy: coord(sy(p.y)) })),
		xLabel,
		pointAgeLabel: mixedAxis ? 'Plotted age (months)' : xLabel,
		yLabel: `${MEASURE_LABEL[measure]} (${unitLabel(measure, units)})`,
		table: {
			ticks: xTicks,
			lines: curveLines.map((l) => ({
				label: ordinal(l.percentile),
				values: xTicks.map((t) => valueAt(l.points, t, curveValue))
			}))
		}
	};
}

export type Chart = ReturnType<typeof buildChart>;
