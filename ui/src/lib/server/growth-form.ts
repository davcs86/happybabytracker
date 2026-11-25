import { fail } from '@sveltejs/kit';
import {
	GRAMS_PER_KG,
	GROWTH_RANGES,
	HUNDREDTHS_PER_INCH,
	HUNDREDTHS_PER_OZ,
	LENGTH_UNITS,
	MM_PER_CM,
	OZ_PER_LB,
	WEIGHT_UNITS
} from '../constants';
import { decimal } from '../growth';
import type { EntryBody, GrowthLength, GrowthRow, Measure, Units } from '../types';
import { failAction, type ApiFailure } from './outcome';

export type GrowthFormValues = Record<string, string>;
export type GrowthMode = { kind: 'create' } | { kind: 'edit' };
type Field = 'measured_on' | 'weight' | 'length' | 'head';
type Ranged = keyof typeof GROWTH_RANGES;
const MEASURES: readonly Ranged[] = ['weight', 'length', 'head'];

// Keyed by the API's field names; never a typed value in a message.
export const GROWTH_FIELD_MESSAGES: Record<string, string> = {
	measured_on: 'Enter a date that is not in the future and not before the birth date',
	weight: `Enter a weight: kilograms with up to three decimals, or pounds and 0 to ${OZ_PER_LB - 1}.99 ounces`,
	length: 'Enter a length: centimetres with one decimal, or inches with up to two',
	head: 'Enter a head circumference: centimetres with one decimal, or inches with up to two',
	note: 'The note is too long',
	measurements: 'Enter at least one of weight, length or head circumference'
};

const RANGE_LABEL: Record<Ranged, string> = {
	weight: 'a weight',
	length: 'a length',
	head: 'a head circumference'
};

export const growthRangeMessage = (m: Ranged) =>
	m === 'weight'
		? `Enter ${RANGE_LABEL[m]} from ${GROWTH_RANGES[m].min / GRAMS_PER_KG} to ${GROWTH_RANGES[m].max / GRAMS_PER_KG} kg`
		: `Enter ${RANGE_LABEL[m]} from ${GROWTH_RANGES[m].min / MM_PER_CM} to ${GROWTH_RANGES[m].max / MM_PER_CM} cm`;

const TEXT_FIELDS = [
	'measured_on',
	'weight_unit',
	'weight_kg',
	'weight_lb',
	'weight_oz',
	'length_unit',
	'length',
	'head_unit',
	'head',
	'note'
] as const;
// Every field an edit compares with the text it was loaded with, as <name>_original.
const KEPT_FIELDS = TEXT_FIELDS.filter((f) => f !== 'note');

// The texts a measure's unit makes active; the unit select is compared too.
const ACTIVE: Record<Exclude<Field, 'measured_on'>, (unit: string) => string[] | null> = {
	weight: (u) => (u === 'kg' ? ['weight_kg'] : u === 'lb' ? ['weight_lb', 'weight_oz'] : null),
	length: (u) => (includes(LENGTH_UNITS, u) ? ['length'] : null),
	head: (u) => (includes(LENGTH_UNITS, u) ? ['head'] : null)
};

const includes = <T extends string>(list: readonly T[], v: string): v is T =>
	(list as readonly string[]).includes(v);

export function growthCreateValues(
	today: string,
	units: Pick<Units, 'weight' | 'length'>
): GrowthFormValues {
	const values: GrowthFormValues = Object.fromEntries(TEXT_FIELDS.map((f) => [f, '']));
	return {
		...values,
		measured_on: today,
		weight_unit: units.weight,
		length_unit: units.length,
		head_unit: units.length
	};
}

const lengthText = (l: GrowthLength | null) =>
	l === null
		? ''
		: l.typedUnit === 'cm'
			? decimal(l.mm, MM_PER_CM)
			: decimal(l.inHundredths, HUNDREDTHS_PER_INCH);

// Typed values in their typed unit; the other weight unit is pre-filled from its own column, so flipping
// the unit resends the same weight. A measure the row lacks shows the preferred unit, head the length one.
export function growthEditValues(
	row: GrowthRow,
	units: Pick<Units, 'weight' | 'length'>
): GrowthFormValues {
	const ozPerLb = OZ_PER_LB * HUNDREDTHS_PER_OZ;
	const w = row.weight;
	const shown: GrowthFormValues = {
		measured_on: row.measuredOn,
		weight_unit: w?.typedUnit ?? units.weight,
		weight_kg: w ? decimal(w.grams, GRAMS_PER_KG) : '',
		weight_lb: w ? String(Math.floor(w.ozHundredths / ozPerLb)) : '',
		weight_oz: w ? decimal(w.ozHundredths % ozPerLb, HUNDREDTHS_PER_OZ) : '',
		length_unit: row.length?.typedUnit ?? units.length,
		length: lengthText(row.length),
		head_unit: row.head?.typedUnit ?? units.length,
		head: lengthText(row.head),
		note: row.note ?? ''
	};
	const originals = Object.fromEntries(KEPT_FIELDS.map((k) => [`${k}_original`, shown[k]]));
	return { ...shown, ...originals };
}

// The chart the growth page opens on after a save: the first measure the row has.
export const firstMeasure = (row: Pick<GrowthRow, 'weight' | 'length' | 'head'>): Measure =>
	row.weight ? 'weight' : row.length ? 'length' : row.head ? 'head' : 'weight';

// The units a page without a signed-in parent's preference shows.
export const METRIC_UNITS: Pick<Units, 'weight' | 'length'> = { weight: 'kg', length: 'cm' };

export const GROWTH_LIMIT_MESSAGE = 'This child has the maximum number of measurements';

export function growthFieldFailure<T extends object>(
	values: GrowthFormValues,
	field: string,
	extra: T,
	message: string | null = null
) {
	return fail(422, { ...extra, values, field, message: message ?? GROWTH_FIELD_MESSAGES[field] });
}

// A refused field lands on its input, the row cap on the form, anything else as failAction decides.
// The reader already checked the format, so a refused measure is out of range.
export function growthApiFailure<T extends object>(
	values: GrowthFormValues,
	r: ApiFailure,
	extra: T
) {
	if (r.reason === 'invalid' && r.field && Object.hasOwn(GROWTH_FIELD_MESSAGES, r.field)) {
		const range = includes(MEASURES, r.field) ? growthRangeMessage(r.field) : null;
		return growthFieldFailure(values, r.field, extra, range);
	}
	if (r.reason === 'conflict' && r.code === 'limit') {
		return fail(409, {
			...extra,
			values,
			field: null as string | null,
			message: GROWTH_LIMIT_MESSAGE
		});
	}
	return failAction(r, { ...extra, values, field: null as string | null });
}

// A whole number, or one with up to as many decimals (point or comma) as scale has zeros, times scale.
function scaled(text: string, scale: number): number | null {
	const places = String(scale).length - 1;
	const m = new RegExp(`^(\\d+)(?:[.,](\\d{1,${places}}))?$`).exec(text);
	if (!m) return null;
	const n = Number(m[1]) * scale + Number((m[2] ?? '').padEnd(places, '0'));
	return Number.isSafeInteger(n) ? n : null;
}

function validDate(s: string): boolean {
	const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s);
	if (!m) return false;
	const d = new Date(Date.UTC(+m[1], +m[2] - 1, +m[3]));
	return d.getUTCFullYear() === +m[1] && d.getUTCMonth() === +m[2] - 1 && d.getUTCDate() === +m[3];
}

export type Measured =
	{ body: { unit: string; amount: number } } | 'blank' | { message: string | null };

// A metric amount outside its range is refused here; imperial ones are left to the API's conversion.
function ranged(m: Ranged, unit: string, amount: number): Measured {
	const { min, max } = GROWTH_RANGES[m];
	if ((unit === 'kg' || unit === 'cm') && (amount < min || amount > max)) {
		return { message: growthRangeMessage(m) };
	}
	return amount < 1 ? { message: null } : { body: { unit, amount } };
}

export function weight(v: GrowthFormValues): Measured {
	const unit = v.weight_unit;
	if (!includes(WEIGHT_UNITS, unit)) {
		const typed = [v.weight_kg, v.weight_lb, v.weight_oz].some((t) => t.trim() !== '');
		return typed ? { message: null } : 'blank';
	}
	if (unit === 'lb') {
		const lb = v.weight_lb.trim();
		const oz = v.weight_oz.trim();
		if (lb === '' && oz === '') return 'blank';
		const pounds = lb === '' ? 0 : /^\d+$/.test(lb) ? Number(lb) : null;
		const ounces = oz === '' ? 0 : scaled(oz, HUNDREDTHS_PER_OZ);
		if (pounds === null || ounces === null || ounces >= OZ_PER_LB * HUNDREDTHS_PER_OZ) {
			return { message: null };
		}
		const amount = pounds * OZ_PER_LB * HUNDREDTHS_PER_OZ + ounces;
		return Number.isSafeInteger(amount) ? ranged('weight', unit, amount) : { message: null };
	}
	const kg = v.weight_kg.trim();
	if (kg === '') return 'blank';
	const grams = scaled(kg, GRAMS_PER_KG);
	return grams === null ? { message: null } : ranged('weight', unit, grams);
}

function length(m: 'length' | 'head', v: GrowthFormValues): Measured {
	const text = v[m].trim();
	if (text === '') return 'blank';
	const unit = v[`${m}_unit`];
	if (!includes(LENGTH_UNITS, unit)) return { message: null };
	const amount = scaled(text, unit === 'cm' ? MM_PER_CM : HUNDREDTHS_PER_INCH);
	return amount === null ? { message: null } : ranged(m, unit, amount);
}

type Outcome = { input: EntryBody } | { field: string; message: string | null };

// Fields are checked in the API's order: measured_on, weight, length, head, then measurements.
function build(v: GrowthFormValues, mode: GrowthMode): Outcome {
	const edit = mode.kind === 'edit';
	const original = (k: string) => v[`${k}_original`];
	const same = (k: string) => original(k).trim() === v[k].trim();
	const input: EntryBody = {};

	if (edit && original('measured_on') === undefined) return { field: 'measured_on', message: null };
	const date = v.measured_on.trim();
	if (!edit || !same('measured_on')) {
		if (!validDate(date)) return { field: 'measured_on', message: null };
		input.measured_on = date;
	}

	let present = false;
	for (const m of ['weight', 'length', 'head'] as const) {
		const names = [
			`${m}_unit`,
			...(m === 'weight' ? ['weight_kg', 'weight_lb', 'weight_oz'] : [m])
		];
		const shown = edit ? ACTIVE[m](original(`${m}_unit`) ?? '') : [];
		if (edit && (names.some((n) => original(n) === undefined) || shown === null)) {
			return { field: m, message: null };
		}
		const wasFilled = (shown ?? []).some((n) => original(n).trim() !== '');
		const active = ACTIVE[m](v[`${m}_unit`]) ?? [];
		if (edit && same(`${m}_unit`) && active.every(same)) {
			present ||= wasFilled;
			continue;
		}
		const r = m === 'weight' ? weight(v) : length(m, v);
		if (r === 'blank') {
			if (edit && wasFilled) input[m] = null;
			continue;
		}
		if ('message' in r) return { field: m, message: r.message };
		input[m] = r.body;
		present = true;
	}
	if (!present) return { field: 'measurements', message: null };
	input.note = v.note.trim() || null;
	return { input };
}

// message is null when the field's own message (GROWTH_FIELD_MESSAGES) applies.
export function readGrowthForm(
	form: FormData,
	mode: GrowthMode,
	{ prefix = '' }: { prefix?: string } = {}
): {
	values: GrowthFormValues;
	input: EntryBody | null;
	field: string | null;
	message: string | null;
} {
	const values: GrowthFormValues = Object.fromEntries(
		TEXT_FIELDS.map((k) => [k, String(form.get(`${prefix}${k}`) ?? '')])
	);
	if (mode.kind === 'edit') {
		for (const k of KEPT_FIELDS) {
			const o = form.get(`${prefix}${k}_original`);
			if (o !== null) values[`${k}_original`] = String(o);
		}
	}
	const r = build(values, mode);
	return 'field' in r
		? { values, input: null, field: r.field, message: r.message }
		: { values, input: r.input, field: null, message: null };
}
