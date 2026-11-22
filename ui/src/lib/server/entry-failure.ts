import { fail } from '@sveltejs/kit';
import { MAX_SESSION_HOURS } from '../constants';
import type { EntryValues } from '../entry-edit';
import type { TimerKind } from '../types';
import { failAction, type ApiFailure } from './outcome';

const TIME = 'Enter a time that is not in the future and not before the birth date';

// Keyed by the API's field names, which the form's inputs share.
export const ENTRY_FIELD_MESSAGES: Record<string, string> = {
	at: TIME,
	started_at: TIME,
	ended_at: `Enter an end after the start, within ${MAX_SESSION_HOURS} hours of it and not in the future`,
	contents: 'Choose at least one',
	volume: 'Enter a volume: whole millilitres, or ounces with up to two decimals',
	kind: 'Choose wet, dirty, both or dry',
	what: 'Enter what was eaten',
	reaction: 'The reaction is too long',
	note: 'The note is too long',
	duration_seconds: `Enter whole minutes, up to ${MAX_SESSION_HOURS} hours`,
	side: 'Choose a side',
	left_seconds: 'Enter whole minutes that fit within the session',
	right_seconds: 'Enter whole minutes that fit within the session'
};

const ALREADY_RUNNING: Record<TimerKind, string> = {
	sleep: 'A sleep timer is already running',
	nursing: 'A nursing timer is already running'
};

const isEntryField = (f: string | undefined): f is string =>
	f !== undefined && Object.hasOwn(ENTRY_FIELD_MESSAGES, f);

export function entryFieldFailure<T extends object>(values: EntryValues, field: string, extra: T) {
	return fail(422, { ...extra, values, field, message: ENTRY_FIELD_MESSAGES[field] });
}

// A refused field lands on its input, a second timer on the page, anything else on the form.
export function entryApiFailure<T extends object>(
	kind: string,
	values: EntryValues,
	r: ApiFailure,
	extra: T
) {
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
