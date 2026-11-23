import { floorMinutes } from '../entry-edit';
import type { EntryBody, TrackerKind } from '../types';

const TIME_FIELDS: Record<TrackerKind, string[]> = {
	bottles: ['at'],
	diapers: ['at'],
	meals: ['at'],
	symptoms: ['at'],
	pumping: ['started_at'],
	sleep: ['started_at', 'ended_at'],
	nursing: ['started_at', 'ended_at']
};

const text = (form: FormData, name: string) => String(form.get(name) ?? '').trim();

// Holds only for a form that carries the original it was loaded with.
const unchanged = (form: FormData, name: string) => {
	const original = form.get(`${name}_original`);
	return original !== null && text(form, name) === String(original).trim();
};

// A form shows a time to the minute, a volume in one unit and a length in whole minutes; sending those back
// unchanged would rewrite what the API stored more exactly.
export function sendOnlyChanged(kind: TrackerKind, form: FormData, input: EntryBody): EntryBody {
	const out = { ...input };
	for (const name of TIME_FIELDS[kind]) if (unchanged(form, name)) delete out[name];
	// A simple form over a bottle with sources, or a session with sides, must send its total, or they would stay.
	const hadParts =
		(kind === 'bottles' && text(form, 'sources_original') !== '') ||
		(kind === 'pumping' && text(form, 'sides_original') !== '');
	if (
		(kind === 'bottles' || kind === 'pumping') &&
		!hadParts &&
		unchanged(form, 'volume') &&
		unchanged(form, 'volume_unit')
	) {
		delete out.volume;
	}
	const stored = text(form, 'duration_seconds_original');
	if (
		kind === 'pumping' &&
		/^\d+$/.test(stored) &&
		text(form, 'duration_minutes') === String(floorMinutes(Number(stored)))
	) {
		delete out.duration_seconds;
	}
	return out;
}

// Only a finished session's form carries the stored seconds, so a nursing form without them came from a running page.
export const fromRunningPage = (kind: TrackerKind, form: FormData) =>
	kind === 'nursing' &&
	form.get('left_seconds_original') === null &&
	form.get('right_seconds_original') === null;
