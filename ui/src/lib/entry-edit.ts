import {
	BOTTLE_CONTENTS,
	GRAMS_PER_KG,
	HUNDREDTHS_PER_FLOZ,
	HUNDREDTHS_PER_OZ,
	MAX_SESSION_HOURS,
	OZ_PER_LB,
	SECONDS_PER_HOUR,
	SECONDS_PER_MINUTE
} from './constants';
import { decimal } from './growth';
import { instantToLocalInput } from './time-zone';
import type {
	GrowthWeight,
	Session,
	TrackerEntries,
	TrackerKind,
	Units,
	Volume,
	VolumeUnit
} from './types';

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

type FormUnits = Pick<Units, 'weight' | 'volume'>;

// What a form shows its amounts in; a page without a signed-in parent's preference is metric.
export const formUnits = (session: Session | null | undefined): FormUnits => ({
	weight: session?.parent?.units.weight ?? 'kg',
	volume: session?.parent?.units.volume ?? 'ml'
});

// The two weigh-ins of a nursing form, each a unit and the texts that unit makes active (see growth-form).
export const WEIGHT_NAMES = ['weight_before', 'weight_after'] as const;

const blankSources = () => Object.fromEntries(BOTTLE_CONTENTS.map((c) => [`sources_${c}`, '']));
const blankWeights = (unit: Units['weight']): EntryValues =>
	Object.fromEntries(
		WEIGHT_NAMES.flatMap((n) => [
			[`${n}_unit`, unit],
			[`${n}_kg`, ''],
			[`${n}_lb`, ''],
			[`${n}_oz`, '']
		])
	);

export function createValues(
	kind: TrackerKind,
	now: string,
	zone: string,
	units: FormUnits
): EntryValues {
	const at = instantToLocalInput(now, zone);
	const unit = units.volume;
	switch (kind) {
		case 'bottles':
			return {
				at,
				contents: [],
				volume: '',
				volume_unit: unit,
				...blankSources(),
				note: ''
			};
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
				...blankWeights(units.weight),
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

// A source shown in the parent's unit; a name the bottle has no source for is blank.
function sourceValues(e: TrackerEntries['bottles'], unit: VolumeUnit): EntryValues {
	const shown = BOTTLE_CONTENTS.flatMap((c) => {
		const source = e.sources.find((s) => s.source === c);
		return Object.entries(
			withOriginal(`sources_${c}`, source ? volumeText(source.volume, unit) : '')
		);
	});
	return {
		...Object.fromEntries(shown),
		sources_original: e.sources.map((s) => s.source).join(',')
	};
}

// Each weight in its typed unit; the other unit is pre-filled from its own column, so flipping resends the same weight.
function weightValues(name: string, w: GrowthWeight | null, unit: Units['weight']): EntryValues {
	const ozPerLb = OZ_PER_LB * HUNDREDTHS_PER_OZ;
	return {
		...withOriginal(`${name}_unit`, w?.typedUnit ?? unit),
		...withOriginal(`${name}_kg`, w ? decimal(w.grams, GRAMS_PER_KG) : ''),
		...withOriginal(`${name}_lb`, w ? String(Math.floor(w.ozHundredths / ozPerLb)) : ''),
		...withOriginal(`${name}_oz`, w ? decimal(w.ozHundredths % ozPerLb, HUNDREDTHS_PER_OZ) : '')
	};
}

const instantValues = (name: string, instant: string | null, zone: string): EntryValues =>
	withOriginal(name, instant === null ? '' : instantToLocalInput(instant, zone));

// The volume is shown in the parent's unit, whichever unit it was typed in.
export function editValues(
	kind: TrackerKind,
	entry: Entry,
	zone: string,
	units: FormUnits
): EntryValues {
	const unit = units.volume;
	const note = { note: entry.note ?? '' };
	switch (kind) {
		case 'bottles': {
			const e = entry as TrackerEntries['bottles'];
			return {
				...instantValues('at', e.at, zone),
				contents: [...e.contents],
				...volumeValues(e.volume, unit),
				...sourceValues(e, unit),
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
			const weights = {
				...weightValues('weight_before', e.weightBefore, units.weight),
				...weightValues('weight_after', e.weightAfter, units.weight)
			};
			// A running session has no minutes to show or keep.
			if (e.endedAt === null) return { ...times, ...weights, ...note };
			return {
				...times,
				left_minutes: String(floorMinutes(e.leftSeconds)),
				right_minutes: String(floorMinutes(e.rightSeconds)),
				left_seconds_original: String(e.leftSeconds),
				right_seconds_original: String(e.rightSeconds),
				...weights,
				...note
			};
		}
	}
}
