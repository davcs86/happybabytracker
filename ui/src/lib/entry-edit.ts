import {
	HUNDREDTHS_PER_FLOZ,
	MAX_SESSION_HOURS,
	SECONDS_PER_HOUR,
	SECONDS_PER_MINUTE
} from './constants';
import { instantToLocalInput } from './time-zone';
import type { TrackerEntries, TrackerKind, Volume, VolumeUnit } from './types';

// What the entry form shows: the API's own field names, each a text (the contents a list).
export type EntryValues = Record<string, string | string[]>;

type Entry = TrackerEntries[TrackerKind];

// A form for a new entry, for a timer still running, or for one that is done.
export type EntryStage = 'new' | 'running' | 'done';

const MS_PER_SECOND = 1000;

// The minutes a finished session's seconds pre-fill; never rounded up, so an unchanged edit fits its span.
export const floorMinutes = (seconds: number) => Math.floor(seconds / SECONDS_PER_MINUTE);

// The route matcher admits only the six tracker names, so a path parameter that got here is one.
export const trackerOf = (param: string) => param as TrackerKind;

const POINT_KINDS: readonly TrackerKind[] = ['bottles', 'diapers', 'meals'];

// The time the timeline files an entry under.
export function entryInstant(kind: TrackerKind, entry: Entry): string {
	return POINT_KINDS.includes(kind)
		? (entry as TrackerEntries['bottles']).at
		: (entry as TrackerEntries['sleep']).startedAt;
}

// A timer runs while it has no end.
export function isRunning(kind: TrackerKind, entry: Entry): boolean {
	return (
		(kind === 'sleep' || kind === 'nursing') && (entry as TrackerEntries['sleep']).endedAt === null
	);
}

// The kid-local date of an instant, YYYY-MM-DD.
export const localDate = (instant: string, zone: string) =>
	instantToLocalInput(instant, zone).slice(0, 10);

// The API refuses to end a timer once it has run longer than the session limit.
export const ranOver = (startedAt: string, now: number) =>
	now - Date.parse(startedAt) > MAX_SESSION_HOURS * SECONDS_PER_HOUR * MS_PER_SECOND;

// Whole millilitres, or ounces without trailing zeros: the form reads both back.
export function volumeText(v: Pick<Volume, 'ml' | 'floz_hundredths'>, unit: VolumeUnit): string {
	return unit === 'ml'
		? String(v.ml)
		: String(Number((v.floz_hundredths / HUNDREDTHS_PER_FLOZ).toFixed(2)));
}

export function createValues(
	kind: TrackerKind,
	now: string,
	zone: string,
	unit: VolumeUnit
): EntryValues {
	const at = instantToLocalInput(now, zone);
	switch (kind) {
		case 'bottles':
			return { at, contents: [], volume: '', volume_unit: unit, note: '' };
		case 'diapers':
			return { at, kind: '', note: '' };
		case 'meals':
			return { at, what: '', reaction: '', note: '' };
		case 'pumping':
			return {
				started_at: at,
				duration_minutes: '',
				volume: '',
				volume_unit: unit,
				side: '',
				note: ''
			};
		case 'sleep':
			return { started_at: at, ended_at: '', note: '' };
		case 'nursing':
			return {
				started_at: at,
				ended_at: '',
				left_minutes: '',
				right_minutes: '',
				side: '',
				note: ''
			};
	}
}

// A field the form can leave alone keeps its shown text beside it as <name>_original.
const withOriginal = (name: string, text: string): EntryValues => ({
	[name]: text,
	[`${name}_original`]: text
});

const volumeValues = (v: Volume, unit: VolumeUnit): EntryValues => ({
	...withOriginal('volume', volumeText(v, unit)),
	...withOriginal('volume_unit', unit)
});

const instantValues = (name: string, instant: string | null, zone: string): EntryValues =>
	withOriginal(name, instant === null ? '' : instantToLocalInput(instant, zone));

// The volume is shown in the parent's unit, whichever unit it was typed in.
export function editValues(
	kind: TrackerKind,
	entry: Entry,
	zone: string,
	unit: VolumeUnit
): EntryValues {
	const note = { note: entry.note ?? '' };
	switch (kind) {
		case 'bottles': {
			const e = entry as TrackerEntries['bottles'];
			return {
				...instantValues('at', e.at, zone),
				contents: [...e.contents],
				...volumeValues(e.volume, unit),
				...note
			};
		}
		case 'diapers': {
			const e = entry as TrackerEntries['diapers'];
			return { ...instantValues('at', e.at, zone), kind: e.kind, ...note };
		}
		case 'meals': {
			const e = entry as TrackerEntries['meals'];
			return {
				...instantValues('at', e.at, zone),
				what: e.what,
				reaction: e.reaction ?? '',
				...note
			};
		}
		case 'pumping': {
			const e = entry as TrackerEntries['pumping'];
			return {
				...instantValues('started_at', e.startedAt, zone),
				duration_minutes: String(floorMinutes(e.durationSeconds)),
				duration_seconds_original: String(e.durationSeconds),
				...volumeValues(e.volume, unit),
				side: e.side,
				...note
			};
		}
		case 'sleep': {
			const e = entry as TrackerEntries['sleep'];
			return {
				...instantValues('started_at', e.startedAt, zone),
				...instantValues('ended_at', e.endedAt, zone),
				...note
			};
		}
		case 'nursing': {
			const e = entry as TrackerEntries['nursing'];
			const times = {
				...instantValues('started_at', e.startedAt, zone),
				...instantValues('ended_at', e.endedAt, zone)
			};
			// A running session has no minutes to show or keep.
			if (e.endedAt === null) return { ...times, ...note };
			return {
				...times,
				left_minutes: String(floorMinutes(e.leftSeconds)),
				right_minutes: String(floorMinutes(e.rightSeconds)),
				left_seconds_original: String(e.leftSeconds),
				right_seconds_original: String(e.rightSeconds),
				...note
			};
		}
	}
}
