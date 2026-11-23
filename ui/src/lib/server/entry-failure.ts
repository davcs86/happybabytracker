import { fail } from '@sveltejs/kit';
import { MAX_SESSION_HOURS, MEAL_TEXT_MAX_LEN, VOLUME_ML_RANGE } from '../constants';
import type { EntryValues } from '../entry-edit';
import type { TimerKind } from '../types';
import { GROWTH_FIELD_MESSAGES } from './growth-form';
import { failAction, type ApiFailure } from './outcome';

const TIME = 'Enter a time that is not in the future and not before the birth date';

// Keyed by the API's field names, which the form's inputs share.
export const ENTRY_FIELD_MESSAGES: Record<string, string> = {
	at: TIME,
	started_at: TIME,
	ended_at: `Enter an end after the start, within ${MAX_SESSION_HOURS} hours of it and not in the future`,
	contents: 'Choose at least one',
	volume: 'Enter a volume: whole millilitres, or ounces with up to two decimals',
	sources:
		'Enter a volume for every ticked source: whole millilitres, or ounces with up to two decimals',
	weight_before: GROWTH_FIELD_MESSAGES.weight,
	weight_after: GROWTH_FIELD_MESSAGES.weight,
	kind: 'Choose wet, dirty, both or dry',
	what: `Enter what was eaten, up to ${MEAL_TEXT_MAX_LEN} characters`,
	reaction: 'The reaction is too long',
	note: 'The note is too long',
	duration_seconds: `Enter whole minutes, up to ${MAX_SESSION_HOURS} hours`,
	side: 'Choose a side',
	left_seconds: 'Enter whole minutes that fit within the session',
	right_seconds: 'Enter whole minutes that fit within the session'
};

// The form's own reader can tell these apart from the field's message above.
export const REOPEN_MESSAGE =
	'A finished session cannot be re-opened; set an end time or delete it';
export const FINISHED_SIDE_MESSAGE = 'A finished session has no side; clear the side or the end';
export const WHAT_MISSING_MESSAGE = 'Enter what was eaten';
export const WHAT_TOO_LONG_MESSAGE = `Keep what was eaten to ${MEAL_TEXT_MAX_LEN} characters`;
export type VolumeKind = keyof typeof VOLUME_ML_RANGE;
export const volumeRangeMessage = (kind: VolumeKind) =>
	`Enter from ${VOLUME_ML_RANGE[kind].min} to ${VOLUME_ML_RANGE[kind].max} ml`;
// The API refuses an amount the reader let through only by its range, in ounces as much as millilitres.
export const volumeRefusedMessage = (kind: VolumeKind) =>
	`${volumeRangeMessage(kind)}, in whole millilitres or ounces with up to two decimals`;

export const TIMER_ENDED_MESSAGE = 'This timer has ended; check the times and save again';

const ALREADY_RUNNING: Record<TimerKind, string> = {
	sleep: 'A sleep timer is already running',
	nursing: 'A nursing timer is already running'
};

const isEntryField = (f: string | undefined): f is string =>
	f !== undefined && Object.hasOwn(ENTRY_FIELD_MESSAGES, f);

export function entryFieldFailure<T extends object>(
	values: EntryValues,
	field: string,
	extra: T,
	message: string | null = null
) {
	return fail(422, { ...extra, values, field, message: message ?? ENTRY_FIELD_MESSAGES[field] });
}

// A refused field lands on its input, a second timer on the page, anything else on the form.
export function entryApiFailure<T extends object>(
	kind: string,
	values: EntryValues,
	r: ApiFailure,
	extra: T
) {
	if (r.reason === 'invalid' && r.field === 'volume' && Object.hasOwn(VOLUME_ML_RANGE, kind))
		return entryFieldFailure(values, r.field, extra, volumeRefusedMessage(kind as VolumeKind));
	if (r.reason === 'invalid' && isEntryField(r.field))
		return entryFieldFailure(values, r.field, extra);
	if (
		r.reason === 'conflict' &&
		r.code === 'already_running' &&
		Object.hasOwn(ALREADY_RUNNING, kind)
	) {
		return fail(409, {
			...extra,
			values,
			field: null as string | null,
			message: ALREADY_RUNNING[kind as TimerKind],
			running: true
		});
	}
	return failAction(r, { ...extra, values, field: null as string | null });
}

// The page was opened while the timer ran; the fresh values replace what it shows.
export function staleTimerFailure<T extends object>(values: EntryValues, extra: T) {
	return fail(409, {
		...extra,
		values,
		field: null as string | null,
		message: TIMER_ENDED_MESSAGE,
		stale: true
	});
}
