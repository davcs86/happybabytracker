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
	nursing: ['started_at', 'ended_at', 'left_minutes', 'right_minutes', 'side', 'note']
};

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

const orNull = (text: string) => text.trim() || null;

// Fields are checked in the API's order, so the one named is the one the API would name.
function build(kind: TrackerKind, v: EntryFormValues, zone: string): Outcome {
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
			// The seconds belong to a finished session; a running one takes its side instead.
			if (input.ended_at !== undefined) {
				const left = seconds(s('left_minutes'), true);
				if (left === null) return bad('left_seconds');
				const right = seconds(s('right_minutes'), true);
				if (right === null) return bad('right_seconds');
				Object.assign(input, { left_seconds: left, right_seconds: right });
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
	zone: string
): { values: EntryFormValues; input: EntryBody | null; field: string | null } {
	const values: EntryFormValues = Object.fromEntries(
		TEXT_FIELDS[kind].map((k) => [k, String(form.get(k) ?? '')])
	);
	if (kind === 'bottles') values.contents = form.getAll('contents').map(String);
	const r = build(kind, values, zone);
	return 'field' in r
		? { values, input: null, field: r.field }
		: { values, input: r.input, field: null };
}
