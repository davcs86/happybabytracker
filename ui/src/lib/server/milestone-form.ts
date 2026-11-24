import { fail } from '@sveltejs/kit';
import { MILESTONE_TEXT_MAX, NOTE_MAX_LEN } from '../constants';
import { milestonesPath } from '../paths';
import { todayInOrUtc } from '../time-zone';
import type { EntryBody, MilestoneEntry } from '../types';
import { MILESTONE_ITEM_PATTERN } from './constants';
import { ENTRY_FIELD_MESSAGES } from './entry-failure';
import { failAction, type ApiFailure } from './outcome';

const KEPT = ['text', 'reached_on', 'note'] as const;
type Original = `${(typeof KEPT)[number]}_original`;

export type MilestoneFormValues = {
	item?: string;
	text?: string;
	reached_on: string;
	note: string;
} & Partial<Record<Original, string>>;
// Only a custom milestone has a text to edit; a catalog row never changes its item.
export type MilestoneMode = { kind: 'create' } | { kind: 'edit'; custom: boolean };

// Keyed by the API's field names; never a typed value in a message.
export const MILESTONE_FIELD_MESSAGES: Record<string, string> = {
	item: 'This milestone is no longer in the list',
	text: `Enter a milestone of up to ${MILESTONE_TEXT_MAX} characters`,
	reached_on: 'Enter a date that is not in the future and not before the birth date',
	note: ENTRY_FIELD_MESSAGES.note
};

function validDate(s: string): boolean {
	const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s);
	if (!m) return false;
	const d = new Date(Date.UTC(+m[1], +m[2] - 1, +m[3]));
	return d.getUTCFullYear() === +m[1] && d.getUTCMonth() === +m[2] - 1 && d.getUTCDate() === +m[3];
}

const textOK = (text: string) => text !== '' && [...text].length <= MILESTONE_TEXT_MAX;

type Outcome = { input: EntryBody } | { field: string };

// The API's order: item or text, reached_on, note.
function buildCreate(v: MilestoneFormValues, today: string): Outcome {
	const input: EntryBody = {};
	if ((v.item === undefined) === (v.text === undefined)) return { field: 'item' };
	if (v.item !== undefined) {
		if (!MILESTONE_ITEM_PATTERN.test(v.item)) return { field: 'item' };
		input.item = v.item;
	} else {
		const text = (v.text ?? '').trim();
		if (!textOK(text)) return { field: 'text' };
		input.text = text;
	}
	const date = v.reached_on.trim() || today;
	if (!validDate(date) || date > today) return { field: 'reached_on' };
	input.reached_on = date;
	const note = v.note.trim();
	if ([...note].length > NOTE_MAX_LEN) return { field: 'note' };
	if (note !== '') input.note = note;
	return { input };
}

// Sends only what differs from its hidden original, and fails closed when one is missing.
function buildEdit(v: MilestoneFormValues, custom: boolean, today: string): Outcome {
	const input: EntryBody = {};
	if (custom) {
		if (v.text_original === undefined) return { field: 'text' };
		const text = (v.text ?? '').trim();
		if (text !== v.text_original.trim()) {
			if (!textOK(text)) return { field: 'text' };
			input.text = text;
		}
	}
	if (v.reached_on_original === undefined) return { field: 'reached_on' };
	const date = v.reached_on.trim();
	if (date !== v.reached_on_original.trim()) {
		if (!validDate(date) || date > today) return { field: 'reached_on' };
		input.reached_on = date;
	}
	if (v.note_original === undefined) return { field: 'note' };
	const note = v.note.trim();
	if (note !== v.note_original.trim()) {
		if ([...note].length > NOTE_MAX_LEN) return { field: 'note' };
		input.note = note === '' ? null : note;
	}
	return { input };
}

// zone is the kid's: a create without a date records the kid-local today, and no date may be after it.
export function readMilestoneForm(
	form: FormData,
	mode: MilestoneMode,
	zone: string
): {
	values: MilestoneFormValues;
	input: EntryBody | null;
	field: string | null;
	message: string | null;
} {
	const text = (k: string) => String(form.get(k) ?? '');
	const values: MilestoneFormValues = { reached_on: text('reached_on'), note: text('note') };
	for (const k of ['item', 'text'] as const) {
		const v = form.get(k);
		if (v !== null) values[k] = String(v);
	}
	if (mode.kind === 'edit') {
		for (const k of KEPT) {
			const o = form.get(`${k}_original`);
			if (o !== null) values[`${k}_original`] = String(o);
		}
	}
	const today = todayInOrUtc(zone);
	const r =
		mode.kind === 'edit' ? buildEdit(values, mode.custom, today) : buildCreate(values, today);
	return 'field' in r
		? { values, input: null, field: r.field, message: MILESTONE_FIELD_MESSAGES[r.field] }
		: { values, input: r.input, field: null, message: null };
}

export const milestoneCreateValues = (today: string, item?: string): MilestoneFormValues =>
	item ? { item, reached_on: today, note: '' } : { text: '', reached_on: today, note: '' };

export function milestoneEditValues(e: MilestoneEntry): MilestoneFormValues {
	const shown = {
		...(e.item === null ? { text: e.text } : {}),
		reached_on: e.reachedOn,
		note: e.note ?? ''
	};
	const originals = Object.fromEntries(Object.entries(shown).map(([k, v]) => [`${k}_original`, v]));
	return { ...shown, ...originals };
}

// The query opens the item's list without JavaScript; item is a checked id, which needs no escaping.
export const milestoneReachedPath = (kidId: string, item: string) =>
	`${milestonesPath(kidId)}?open=${item}#item-${item}`;

export function readOpenItem(url: URL): string | null {
	const item = url.searchParams.get('open');
	return item !== null && MILESTONE_ITEM_PATTERN.test(item) ? item : null;
}

export const MILESTONE_REACHED_MESSAGE = 'Already marked as reached';
export const MILESTONE_LIMIT_MESSAGE = 'This child has the maximum number of recorded milestones';

export function milestoneFieldFailure<T extends object>(
	values: MilestoneFormValues,
	field: string,
	extra: T,
	message: string | null = null
) {
	return fail(422, {
		...extra,
		values,
		field,
		message: message ?? MILESTONE_FIELD_MESSAGES[field]
	});
}

// Only a field in MILESTONE_FIELD_MESSAGES may land on an input; any other refusal stays off the inputs.
export function milestoneApiFailure<T extends object>(
	values: MilestoneFormValues,
	r: ApiFailure,
	extra: T
) {
	if (r.reason === 'invalid' && r.field && Object.hasOwn(MILESTONE_FIELD_MESSAGES, r.field)) {
		return milestoneFieldFailure(values, r.field, extra);
	}
	if (r.reason === 'conflict' && (r.code === 'already_reached' || r.code === 'limit')) {
		const message = r.code === 'limit' ? MILESTONE_LIMIT_MESSAGE : MILESTONE_REACHED_MESSAGE;
		return fail(409, { ...extra, values, field: null as string | null, message });
	}
	return failAction(r, { ...extra, values, field: null as string | null });
}
