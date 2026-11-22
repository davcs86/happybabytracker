import {
	BOTTLE_CONTENTS,
	DIAPER_KINDS,
	HUNDREDTHS_PER_FLOZ,
	NURSING_SIDES,
	PUMPING_SIDES,
	SECONDS_PER_MINUTE,
	VOLUME_UNITS
} from '../constants';
import { zonedLocalToInstant } from '../time-zone';
import type { EntryBody, TrackerKind, VolumeUnit } from '../types';

export type EntryFormValues = Record<string, string | string[]>;

const TEXT_FIELDS: Record<TrackerKind, string[]> = {
	bottles: ['at', 'volume', 'volume_unit', 'note'],
	diapers: ['at', 'kind', 'note'],
	meals: ['at', 'what', 'reaction', 'note'],
	pumping: ['started_at', 'duration_minutes', 'volume', 'volume_unit', 'side', 'note'],
	sleep: ['started_at', 'ended_at', 'note'],
	nursing: [
		'started_at',
		'ended_at',
		'left_minutes',
		'right_minutes',
		'left_seconds_original',
		'right_seconds_original',
		'side',
		'note'
	]
};

// create, or an edit of a running (running: true) or finished nursing session.
export type EntryMode = { kind: 'create' } | { kind: 'edit'; running: boolean };

// The minutes a finished session's seconds pre-fill; never rounded up, so an unchanged edit fits its span.
export const floorMinutes = (seconds: number) => Math.floor(seconds / SECONDS_PER_MINUTE);

type Outcome = { input: EntryBody } | { field: string };

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

// Whole millilitres, or fluid ounces with up to two decimals as hundredths of an ounce.
function amount(unit: VolumeUnit, text: string): number | null {
	const m = (unit === 'ml' ? /^(\d+)$/ : /^(\d+)(?:\.(\d{1,2}))?$/).exec(text.trim());
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

// Fields are checked in the API's order, so the one named is the one the API would name.
function build(kind: TrackerKind, v: EntryFormValues, zone: string, mode: EntryMode): Outcome {
	const s = (k: string) => v[k] as string;
	const input: EntryBody = {};
	const bad = (field: string): Outcome => ({ field });
	const timeKey =
		kind === 'bottles' || kind === 'diapers' || kind === 'meals' ? 'at' : 'started_at';
	if (!putTime(input, timeKey, s(timeKey), zone)) return bad(timeKey);

	switch (kind) {
		case 'bottles': {
			const contents = v.contents as string[];
			if (contents.length === 0 || !contents.every((c) => includes(BOTTLE_CONTENTS, c))) {
				return bad('contents');
			}
			const vol = volume(s('volume_unit'), s('volume'));
			if (!vol) return bad('volume');
			Object.assign(input, { contents, volume: vol });
			break;
		}
		case 'diapers':
			if (!includes(DIAPER_KINDS, s('kind'))) return bad('kind');
			input.kind = s('kind');
			break;
		case 'meals': {
			const what = s('what').trim();
			if (what === '') return bad('what');
			Object.assign(input, { what, reaction: orNull(s('reaction')) });
			break;
		}
		case 'pumping': {
			const duration = seconds(s('duration_minutes'), false);
			if (duration === null) return bad('duration_seconds');
			const vol = volume(s('volume_unit'), s('volume'));
			if (!vol) return bad('volume');
			if (!includes(PUMPING_SIDES, s('side'))) return bad('side');
			Object.assign(input, { duration_seconds: duration, volume: vol, side: s('side') });
			break;
		}
		case 'sleep':
			if (!putTime(input, 'ended_at', s('ended_at'), zone)) return bad('ended_at');
			break;
		case 'nursing': {
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
				input.side = s('side');
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
): { values: EntryFormValues; input: EntryBody | null; field: string | null } {
	const values: EntryFormValues = Object.fromEntries(
		TEXT_FIELDS[kind].map((k) => [k, String(form.get(k) ?? '')])
	);
	if (kind === 'bottles') values.contents = form.getAll('contents').map(String);
	const r = build(kind, values, zone, mode);
	return 'field' in r
		? { values, input: null, field: r.field }
		: { values, input: r.input, field: null };
}
