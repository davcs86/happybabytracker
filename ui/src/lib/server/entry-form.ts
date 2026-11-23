import {
	BOTTLE_CONTENTS,
	DIAPER_KINDS,
	HUNDREDTHS_PER_FLOZ,
	MEAL_TEXT_MAX_LEN,
	NURSING_SIDES,
	PUMPING_SIDES,
	SECONDS_PER_MINUTE,
	VOLUME_ML_RANGE,
	VOLUME_UNITS,
	WEIGHT_UNITS
} from '../constants';
import { floorMinutes, WEIGHT_NAMES } from '../entry-edit';
import { formatVolume } from '../format';
import { zonedLocalToInstant } from '../time-zone';
import type { EntryBody, TrackerKind, VolumeUnit } from '../types';
import {
	FINISHED_SIDE_MESSAGE,
	REOPEN_MESSAGE,
	sourcesTotalMessage,
	totalRangeMessage,
	volumeRangeMessage,
	WHAT_MISSING_MESSAGE,
	WHAT_TOO_LONG_MESSAGE,
	type VolumeKind
} from './entry-failure';
import { weight as readWeight } from './growth-form';

export type EntryFormValues = Record<string, string | string[]>;

const WEIGHT_PARTS = ['unit', 'kg', 'lb', 'oz'] as const;
const WEIGHT_FIELDS = WEIGHT_NAMES.flatMap((n) => WEIGHT_PARTS.map((p) => `${n}_${p}`));
const SOURCE_FIELDS = BOTTLE_CONTENTS.map((c) => `sources_${c}`);
const SIDE_FIELDS = NURSING_SIDES.map((s) => `sides_${s}`);

const TEXT_FIELDS: Record<TrackerKind, string[]> = {
	bottles: ['at', 'volume', 'volume_unit', 'advanced', ...SOURCE_FIELDS, 'note'],
	diapers: ['at', 'kind', 'note'],
	meals: ['at', 'what', 'reaction', 'note'],
	symptoms: ['at', 'note'],
	pumping: [
		'started_at',
		'duration_minutes',
		'volume',
		'volume_unit',
		'side',
		'advanced',
		...SIDE_FIELDS,
		'note'
	],
	sleep: ['started_at', 'ended_at', 'note'],
	nursing: [
		'started_at',
		'ended_at',
		'left_minutes',
		'right_minutes',
		'left_seconds_original',
		'right_seconds_original',
		'side',
		'advanced',
		...WEIGHT_FIELDS,
		'note'
	]
};

// What an edit of an advanced form compares against, as loaded; one that is missing is refused, never guessed.
const ORIGINALS: Partial<Record<TrackerKind, string[]>> = {
	bottles: ['volume_unit', 'sources', ...SOURCE_FIELDS].map((k) => `${k}_original`),
	pumping: ['volume_unit', 'sides', ...SIDE_FIELDS].map((k) => `${k}_original`),
	nursing: WEIGHT_FIELDS.map((k) => `${k}_original`)
};

// create, or an edit of a running (running: true) or finished nursing session.
export type EntryMode = { kind: 'create' } | { kind: 'edit'; running: boolean };

export { floorMinutes };

// message is null when the field's own message (ENTRY_FIELD_MESSAGES) applies.
type Outcome = { input: EntryBody } | { field: string; message: string | null };

const includes = <T extends string>(list: readonly T[], v: string): v is T =>
	(list as readonly string[]).includes(v);

// A blank time is left out, so the API takes now on create and keeps the stored time on update.
function putTime(input: EntryBody, key: string, text: string, zone: string): boolean {
	const t = text.trim();
	if (t === '') return true;
	const instant = zonedLocalToInstant(t, zone);
	if (instant === null) return false;
	input[key] = instant;
	return true;
}

// Whole millilitres, or fluid ounces with up to two decimals (point or comma) as hundredths of an ounce.
function amount(unit: VolumeUnit, text: string): number | null {
	const m = (unit === 'ml' ? /^(\d+)$/ : /^(\d+)(?:[.,](\d{1,2}))?$/).exec(text.trim());
	if (!m) return null;
	const n =
		unit === 'ml'
			? Number(m[1])
			: Number(m[1]) * HUNDREDTHS_PER_FLOZ + Number((m[2] ?? '').padEnd(2, '0'));
	return Number.isSafeInteger(n) ? n : null;
}

function volume(unit: string, text: string): { unit: VolumeUnit; amount: number } | null {
	if (!includes(VOLUME_UNITS, unit)) return null;
	const n = amount(unit, text);
	return n === null ? null : { unit, amount: n };
}

// A blank count is 0 when the field is optional, else not a count.
function seconds(text: string, optional: boolean): number | null {
	const t = text.trim();
	if (t === '') return optional ? 0 : null;
	if (!/^\d+$/.test(t)) return null;
	const n = Number(t) * SECONDS_PER_MINUTE;
	return Number.isSafeInteger(n) ? n : null;
}

// The hidden stored seconds of a finished session; anything but a whole number is refused, never read as 0.
function storedSeconds(text: string): number | null {
	if (!/^\d+$/.test(text)) return null;
	const n = Number(text);
	return Number.isSafeInteger(n) ? n : null;
}

const orNull = (text: string) => text.trim() || null;

// A millilitre amount outside the tracker's range is refused here; ounces are left to the API's conversion.
function volumeRange(kind: VolumeKind, vol: { unit: VolumeUnit; amount: number }): string | null {
	const { min, max } = VOLUME_ML_RANGE[kind];
	return vol.unit === 'ml' && (vol.amount < min || vol.amount > max)
		? volumeRangeMessage(kind)
		: null;
}

type Sources = { unit: VolumeUnit; amounts: Record<string, number> };
type Sides = Sources;
type Refusal = { field: string; message: string | null };

// The advanced bottle: one amount per ticked source in one unit. A source typed for an unticked box is not read.
// An edit sends the set only if a ticked text, the set of boxes or the unit differs from what was loaded.
function readSources(
	v: EntryFormValues,
	ticked: readonly string[],
	edit: boolean
): { send: Sources | null } | Refusal {
	const text = (k: string) => v[k] as string | undefined;
	const refuse = (field = 'sources', message: string | null = null): Refusal => ({
		field,
		message
	});
	const unit = text('volume_unit') ?? '';
	if (!includes(VOLUME_UNITS, unit)) return refuse();
	const amounts: Record<string, number> = {};
	let sum = 0;
	for (const name of BOTTLE_CONTENTS.filter((c) => ticked.includes(c))) {
		const n = amount(unit, text(`sources_${name}`) ?? '');
		if (n === null || n < 1) return refuse();
		amounts[name] = n;
		sum += n;
	}
	if (!Number.isSafeInteger(sum)) return refuse();
	// Millilitres only: an ounce total is left to the API's conversion.
	const { min, max } = VOLUME_ML_RANGE.bottles;
	const outside = unit === 'ml' && (sum < min || sum > max);
	const send = (): { send: Sources } | Refusal =>
		outside ? refuse('volume', sourcesTotalMessage()) : { send: { unit, amounts } };
	if (!edit) return send();

	const kept = text('sources_original');
	const stored: string[] = kept === undefined || kept === '' ? [] : kept.split(',');
	if (
		kept === undefined ||
		text('volume_unit_original') === undefined ||
		stored.some((n) => !BOTTLE_CONTENTS.some((c) => c === n)) ||
		BOTTLE_CONTENTS.some((c) => text(`sources_${c}_original`) === undefined)
	) {
		return refuse();
	}
	const names = Object.keys(amounts);
	const same =
		unit === text('volume_unit_original') &&
		names.length === stored.length &&
		names.every(
			(n) =>
				stored.includes(n) &&
				(text(`sources_${n}`) ?? '').trim() === text(`sources_${n}_original`)?.trim()
		);
	// An untouched form is not range-checked: per-source columns shown in ml can sum past the range.
	return same ? { send: null } : send();
}

// The advanced pumping session: one amount per side typed, in one unit; a blank side is absent. An edit sends the
// set only if a side's text, the set of sides or the unit differs from what was loaded.
function readSides(v: EntryFormValues, edit: boolean): { send: Sides | null } | Refusal {
	const text = (k: string) => v[k] as string | undefined;
	const refuse = (field = 'sides', message: string | null = null): Refusal => ({
		field,
		message
	});
	const unit = text('volume_unit') ?? '';
	if (!includes(VOLUME_UNITS, unit)) return refuse();
	const amounts: Record<string, number> = {};
	let sum = 0;
	for (const name of NURSING_SIDES) {
		if ((text(`sides_${name}`) ?? '').trim() === '') continue;
		const n = amount(unit, text(`sides_${name}`) ?? '');
		if (n === null || n < 1) return refuse();
		amounts[name] = n;
		sum += n;
	}
	const names = Object.keys(amounts);
	if (names.length === 0 || !Number.isSafeInteger(sum)) return refuse();
	// Millilitres only: an ounce total is left to the API's conversion.
	const { min, max } = VOLUME_ML_RANGE.pumping;
	const outside = unit === 'ml' && (sum < min || sum > max);
	const send = (): { send: Sides } | Refusal =>
		outside ? refuse('volume', totalRangeMessage('pumping')) : { send: { unit, amounts } };
	if (!edit) return send();

	const kept = text('sides_original');
	const stored: string[] = kept === undefined || kept === '' ? [] : kept.split(',');
	if (
		kept === undefined ||
		text('volume_unit_original') === undefined ||
		stored.some((n) => !includes(NURSING_SIDES, n)) ||
		NURSING_SIDES.some((n) => text(`sides_${n}_original`) === undefined)
	) {
		return refuse();
	}
	const same =
		unit === text('volume_unit_original') &&
		names.length === stored.length &&
		names.every(
			(n) =>
				stored.includes(n) &&
				(text(`sides_${n}`) ?? '').trim() === text(`sides_${n}_original`)?.trim()
		);
	// An untouched form is not range-checked: per-side columns shown in ml can sum past the range.
	return same ? { send: null } : send();
}

// The sum of the texts that read, as the form shows it; the API's total is the sum of what is sent.
function typedTotal(v: EntryFormValues, keys: readonly string[]): string {
	const unit = v.volume_unit as string;
	if (!includes(VOLUME_UNITS, unit)) return '';
	const sum = keys.reduce((total, k) => total + (amount(unit, v[k] as string) ?? 0), 0);
	return formatVolume(
		unit === 'ml' ? { ml: sum, floz_hundredths: 0 } : { ml: 0, floz_hundredths: sum },
		unit
	);
}

// The texts a weight unit makes active, as in the growth form.
const activeTexts = (unit: string): string[] | null =>
	unit === 'kg' ? ['kg'] : unit === 'lb' ? ['lb', 'oz'] : null;

// One weigh-in: nothing to send ({}), a weight, or null to clear it. An edit compares with the texts as loaded and
// fails closed without them, like the growth form.
function readWeighIn(
	v: EntryFormValues,
	name: string,
	edit: boolean
): { send?: { unit: string; amount: number } | null } | Refusal {
	const t = (p: string) => (v[`${name}_${p}`] as string).trim();
	const o = (p: string) => (v[`${name}_${p}_original`] as string | undefined)?.trim();
	let wasFilled = false;
	if (edit) {
		const shown = activeTexts(o('unit') ?? '');
		if (WEIGHT_PARTS.some((p) => o(p) === undefined) || shown === null) {
			return { field: name, message: null };
		}
		if (!includes(WEIGHT_UNITS, t('unit'))) return { field: name, message: null };
		wasFilled = shown.some((p) => o(p) !== '');
		const active = activeTexts(t('unit')) ?? [];
		if (t('unit') === o('unit') && active.every((p) => t(p) === o(p))) return {};
	}
	const r = readWeight({
		weight_unit: t('unit'),
		weight_kg: t('kg'),
		weight_lb: t('lb'),
		weight_oz: t('oz')
	});
	if (r === 'blank') return edit && wasFilled ? { send: null } : {};
	if ('message' in r) return { field: name, message: r.message };
	return { send: r.body };
}

// Fields are checked in the API's order, so the one named is the one the API would name. endShown is true when
// the form was loaded with an end time, which a blank one would silently keep.
function build(
	kind: TrackerKind,
	v: EntryFormValues,
	zone: string,
	mode: EntryMode,
	endShown: boolean
): Outcome {
	const s = (k: string) => v[k] as string;
	const input: EntryBody = {};
	const bad = (field: string, message: string | null = null): Outcome => ({ field, message });
	const reopened = () => mode.kind === 'edit' && endShown && s('ended_at').trim() === '';
	const timeKey =
		kind === 'bottles' || kind === 'diapers' || kind === 'meals' || kind === 'symptoms'
			? 'at'
			: 'started_at';
	if (!putTime(input, timeKey, s(timeKey), zone)) return bad(timeKey);

	switch (kind) {
		case 'bottles': {
			const contents = v.contents as string[];
			if (contents.length === 0 || !contents.every((c) => includes(BOTTLE_CONTENTS, c))) {
				return bad('contents');
			}
			if (s('advanced') === '1') {
				const r = readSources(v, contents, mode.kind === 'edit');
				if ('field' in r) return r;
				if (r.send) input.sources = r.send;
				break;
			}
			const vol = volume(s('volume_unit'), s('volume'));
			if (!vol) return bad('volume');
			const range = volumeRange(kind, vol);
			if (range) return bad('volume', range);
			Object.assign(input, { contents, volume: vol });
			break;
		}
		case 'diapers':
			if (!includes(DIAPER_KINDS, s('kind'))) return bad('kind');
			input.kind = s('kind');
			break;
		case 'meals': {
			const what = s('what').trim();
			if (what === '') return bad('what', WHAT_MISSING_MESSAGE);
			if ([...what].length > MEAL_TEXT_MAX_LEN) return bad('what', WHAT_TOO_LONG_MESSAGE);
			Object.assign(input, { what, reaction: orNull(s('reaction')) });
			break;
		}
		case 'symptoms':
			break;
		case 'pumping': {
			const duration = seconds(s('duration_minutes'), false);
			if (duration === null) return bad('duration_seconds');
			if (s('advanced') === '1') {
				const r = readSides(v, mode.kind === 'edit');
				if ('field' in r) return r;
				input.duration_seconds = duration;
				if (r.send) input.sides = r.send;
				break;
			}
			const vol = volume(s('volume_unit'), s('volume'));
			if (!vol) return bad('volume');
			const range = volumeRange(kind, vol);
			if (range) return bad('volume', range);
			if (!includes(PUMPING_SIDES, s('side'))) return bad('side');
			Object.assign(input, { duration_seconds: duration, volume: vol, side: s('side') });
			break;
		}
		case 'sleep':
			if (reopened()) return bad('ended_at', REOPEN_MESSAGE);
			if (!putTime(input, 'ended_at', s('ended_at'), zone)) return bad('ended_at');
			break;
		case 'nursing': {
			if (reopened()) return bad('ended_at', REOPEN_MESSAGE);
			if (!putTime(input, 'ended_at', s('ended_at'), zone)) return bad('ended_at');
			// A running edit never sends seconds; a finished edit sends a side's only once its minutes changed.
			const finishedCreate = mode.kind === 'create' && input.ended_at !== undefined;
			const finishedEdit = mode.kind === 'edit' && !mode.running;
			for (const side of finishedCreate || finishedEdit ? (['left', 'right'] as const) : []) {
				const typed = seconds(s(`${side}_minutes`), true);
				if (typed === null) return bad(`${side}_seconds`);
				if (finishedCreate) {
					input[`${side}_seconds`] = typed;
					continue;
				}
				const stored = storedSeconds(s(`${side}_seconds_original`));
				if (stored === null) return bad(`${side}_seconds`);
				if (typed !== floorMinutes(stored) * SECONDS_PER_MINUTE) input[`${side}_seconds`] = typed;
			}
			if (s('side') !== '') {
				if (!includes(NURSING_SIDES, s('side'))) return bad('side');
				if (input.ended_at !== undefined) return bad('side', FINISHED_SIDE_MESSAGE);
				input.side = s('side');
			}
			for (const name of s('advanced') === '1' ? WEIGHT_NAMES : []) {
				const r = readWeighIn(v, name, mode.kind === 'edit');
				if ('field' in r) return r;
				if (r.send !== undefined) input[name] = r.send;
			}
			break;
		}
	}
	input.note = orNull(s('note'));
	return { input };
}

// Form names follow the API's, except durations, typed in minutes as duration_minutes, left_minutes and
// right_minutes; a refused field is named as the API names it (duration_seconds, left_seconds, right_seconds).
export function readEntryForm(
	kind: TrackerKind,
	form: FormData,
	zone: string,
	mode: EntryMode
): {
	values: EntryFormValues;
	input: EntryBody | null;
	field: string | null;
	message: string | null;
} {
	const values: EntryFormValues = Object.fromEntries(
		TEXT_FIELDS[kind].map((k) => [k, String(form.get(k) ?? '')])
	);
	if (kind === 'bottles') values.contents = form.getAll('contents').map(String);
	if (kind === 'bottles' && values.advanced === '1') {
		const ticked = BOTTLE_CONTENTS.filter((c) => (values.contents as string[]).includes(c));
		values.sources_total = typedTotal(
			values,
			ticked.map((c) => `sources_${c}`)
		);
	}
	if (kind === 'pumping' && values.advanced === '1')
		values.sides_total = typedTotal(values, SIDE_FIELDS);
	if (mode.kind === 'edit') {
		for (const k of ORIGINALS[kind] ?? []) {
			const o = form.get(k);
			if (o !== null) values[k] = String(o);
		}
	}
	const endShown = String(form.get('ended_at_original') ?? '').trim() !== '';
	const r = build(kind, values, zone, mode, endShown);
	return 'field' in r
		? { values, input: null, field: r.field, message: r.message }
		: { values, input: r.input, field: null, message: null };
}
