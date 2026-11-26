import {
	GRAMS_PER_KG,
	HUNDREDTHS_PER_INCH,
	HUNDREDTHS_PER_OZ,
	MM_PER_CM,
	OZ_PER_LB,
	PERCENTILE_DECIMALS,
	PERCENTILE_SHOWN,
	Z_SCORE_DECIMALS
} from './constants';
import { formatAge, formatAgeDays } from './format';
import { MEASURE_LABEL } from './growth-chart';
import type {
	GrowthLength,
	GrowthRow,
	GrowthScore,
	GrowthSourceInfo,
	GrowthWeight,
	LengthUnit,
	Measure,
	Units,
	WeightUnit
} from './types';

// Ounces are shown to one decimal: hundredths per shown tenth.
const HUNDREDTHS_PER_TENTH = 10;
const TENTHS_PER_LB = OZ_PER_LB * (HUNDREDTHS_PER_OZ / HUNDREDTHS_PER_TENTH);

const roundHalfUp = (n: number, d: number) => Math.floor((2 * n + d) / (2 * d));

// n / scale for a whole n and a power-of-ten scale, without trailing zeros and without float maths.
export function decimal(n: number, scale: number): string {
	const whole = Math.floor(n / scale);
	const frac = String(n % scale)
		.padStart(String(scale).length - 1, '0')
		.replace(/0+$/, '');
	return frac === '' ? String(whole) : `${whole}.${frac}`;
}

export function formatWeight(w: Pick<GrowthWeight, 'grams' | 'ozHundredths'>, unit: WeightUnit) {
	if (unit === 'kg') return `${decimal(w.grams, GRAMS_PER_KG)} kg`;
	const tenths = roundHalfUp(w.ozHundredths, HUNDREDTHS_PER_TENTH);
	const lb = Math.floor(tenths / TENTHS_PER_LB);
	return `${lb} lb ${decimal(tenths % TENTHS_PER_LB, HUNDREDTHS_PER_TENTH)} oz`;
}

export function formatLength(l: Pick<GrowthLength, 'mm' | 'inHundredths'>, unit: LengthUnit) {
	return unit === 'cm'
		? `${decimal(l.mm, MM_PER_CM)} cm`
		: `${decimal(l.inHundredths, HUNDREDTHS_PER_INCH)} in`;
}

const toFixedHalfUp = (x: number, places: number) => {
	const p = 10 ** places;
	const r = Math.round(x * p) / p;
	return (r === 0 ? 0 : r).toFixed(places);
};

// Beyond PERCENTILE_SHOWN the value is shown as a bound, never as 0 or 100.
export function formatPercentile(p: number): string {
	if (p < PERCENTILE_SHOWN.min) return `<${PERCENTILE_SHOWN.min}`;
	if (p > PERCENTILE_SHOWN.max) return `>${PERCENTILE_SHOWN.max}`;
	return toFixedHalfUp(p, PERCENTILE_DECIMALS);
}

export const formatZ = (z: number) => toFixedHalfUp(z, Z_SCORE_DECIMALS);

// Wherever corrected age is used both ages are shown; below zero there is no corrected age yet.
export function correctedAgeLabel(chronologicalDays: number, correctedDays: number | null) {
	const lived = formatAgeDays(chronologicalDays);
	if (correctedDays === null) return lived;
	if (correctedDays < 0) return `${lived} (before expected term)`;
	return `${lived} (corrected ${formatAgeDays(correctedDays)})`;
}

// The header's age; a due date still ahead (ISO dates compare as strings) has no corrected age yet.
export function ageHeader(age: string, correctedBirthDate: string | null, today: string) {
	if (correctedBirthDate === null) return age;
	if (correctedBirthDate > today) return `${age} (before expected term)`;
	return `${age} (corrected ${formatAge(correctedBirthDate, today)})`;
}

const COVERAGE_LABEL = {
	weight: 'weight',
	length: 'length',
	head: 'head circumference',
	bmi: 'BMI'
};

// Whole months inside the API's range, so the hint never claims an age the source lacks.
export const coverageHint = (coverage: GrowthSourceInfo['coverage']) =>
	`Months covered: ${coverage
		.map((c) => `${COVERAGE_LABEL[c.measure]} ${Math.ceil(c.fromMonths)}–${Math.floor(c.toMonths)}`)
		.join(', ')}`;

const NO_SCORE = {
	before_term: 'before expected term',
	not_covered: 'this source has no table for this age'
} as const;

const scoreText = (s: GrowthScore) =>
	s.scored ? `z ${formatZ(s.z)}, percentile ${formatPercentile(s.percentile)}` : NO_SCORE[s.reason];

// One line per measure a row has: its value in the parent's units, then its score or why it has none.
export function rowMeasures(
	r: GrowthRow,
	units: Pick<Units, 'weight' | 'length'>
): { measure: Measure; line: string }[] {
	const parts: { measure: Measure; value: string; score: GrowthScore | null }[] = [];
	if (r.weight) {
		parts.push({
			measure: 'weight',
			value: formatWeight(r.weight, units.weight),
			score: r.scores.weight
		});
	}
	if (r.length) {
		parts.push({
			measure: 'length',
			value: formatLength(r.length, units.length),
			score: r.scores.length
		});
	}
	if (r.head) {
		parts.push({
			measure: 'head',
			value: formatLength(r.head, units.length),
			score: r.scores.head
		});
	}
	// BMI has no imperial unit.
	if (r.scores.bmi)
		parts.push({ measure: 'bmi', value: String(r.scores.bmi.value), score: r.scores.bmi });
	return parts.map((m) => ({
		measure: m.measure,
		line: `${MEASURE_LABEL[m.measure]} ${m.value}${m.score ? `: ${scoreText(m.score)}` : ''}`
	}));
}
