import { fail } from '@sveltejs/kit';
import {
	MAX_OUTCOME_MEDICATIONS,
	MAX_OUTCOME_VACCINATIONS,
	VISIT_OUTCOME_NOTES_MAX_LEN
} from '../constants';
import { instantToLocalInput } from '../time-zone';
import type { EntryBody, Units, Visit } from '../types';
import {
	GROWTH_FIELD_MESSAGES,
	growthCreateValues,
	growthRangeMessage,
	readGrowthForm,
	type GrowthFormValues
} from './growth-form';
import {
	COURSE_FIELD_MESSAGES,
	courseCreateValues,
	readMedicationForm,
	type CourseFormValues
} from './medication-form';
import { failAction, type ApiFailure } from './outcome';
import {
	VACCINE_FIELD_MESSAGES,
	readVaccineForm,
	vaccineCreateValues,
	type VaccineFormValues
} from './vaccine-form';
import { VISIT_REFUSALS } from './visit-form';

export type OutcomeSection = 'growth' | 'vaccinations' | 'medications';
// A section is offered only while its module is on for the kid.
export type OutcomeSections = Record<OutcomeSection, boolean>;

// Each block's values are its module's, unprefixed; the form names carry the prefixes below.
export type OutcomeFormValues = {
	outcome_notes: string;
	add: OutcomeSections;
	growth: GrowthFormValues;
	vaccinations: VaccineFormValues[];
	medications: CourseFormValues[];
};
// The blank blocks a new section or an added block starts from.
export type OutcomeBlanks = {
	growth: GrowthFormValues;
	vaccination: VaccineFormValues;
	medication: CourseFormValues;
};
export type OutcomeMode = { zone: string; sections: OutcomeSections };

export const GROWTH_PREFIX = 'growth.';
export const vaccinePrefix = (i: number) => `vaccinations.${i}.`;
export const medicationPrefix = (i: number) => `medications.${i}.`;
const ADD: Record<OutcomeSection, string> = {
	growth: 'add_growth',
	vaccinations: 'add_vaccinations',
	medications: 'add_medications'
};
export const addName = (s: OutcomeSection) => ADD[s];

// Keyed by the API's field names; never a typed value in a message.
export const OUTCOME_FIELD_MESSAGES: Record<string, string> = {
	outcome_notes: `Enter notes of up to ${VISIT_OUTCOME_NOTES_MAX_LEN.toLocaleString('en-US')} characters`,
	outcome: 'Write notes from the visit, or tick something to add'
};
export const OUTCOME_LIMIT_MESSAGE =
	'This child has the maximum number of measurements, vaccinations or medication courses';

// The visit's date and start on the kid's clock prefill the blocks.
export function outcomeBlanks(
	visit: Pick<Visit, 'startsAt'>,
	zone: string,
	units: Pick<Units, 'weight' | 'length'>
): OutcomeBlanks {
	const start = instantToLocalInput(visit.startsAt, zone);
	const day = start.slice(0, 10);
	return {
		growth: growthCreateValues(day, units),
		vaccination: vaccineCreateValues(day),
		medication: { ...courseCreateValues(), start_at: start }
	};
}

// The notes field shows the stored notes, so a save that keeps them sends nothing for them.
export const outcomeCreateValues = (
	b: OutcomeBlanks,
	v: Pick<Visit, 'outcomeNotes'>
): OutcomeFormValues => ({
	outcome_notes: v.outcomeNotes ?? '',
	add: { growth: false, vaccinations: false, medications: false },
	growth: b.growth,
	vaccinations: [b.vaccination],
	medications: [b.medication]
});

const growthPath = (f: string) => `growth.${f}`;
const vaccinePath = (i: number, f: string) => `vaccinations[${i}].${f}`;
const medicationPath = (i: number, f: string) => `medications[${i}].${f}`;
const MEASURES = ['weight', 'length', 'head'];

// A field path of the API's outcome, named in its block: "Vaccine 2: …". null for a path no form field shows.
export function outcomeFieldMessage(path: string, { api = false } = {}): string | null {
	if (Object.hasOwn(OUTCOME_FIELD_MESSAGES, path)) return OUTCOME_FIELD_MESSAGES[path];
	const growth = /^growth\.(\w+)$/.exec(path);
	if (growth && Object.hasOwn(GROWTH_FIELD_MESSAGES, growth[1])) {
		const f = growth[1] as 'weight' | 'length' | 'head';
		return api && MEASURES.includes(f) ? growthRangeMessage(f) : GROWTH_FIELD_MESSAGES[f];
	}
	const block = /^(vaccinations|medications)\[(\d+)\]\.(\w+)$/.exec(path);
	if (!block) return null;
	const [, list, index, f] = block;
	const messages = list === 'vaccinations' ? VACCINE_FIELD_MESSAGES : COURSE_FIELD_MESSAGES;
	if (!Object.hasOwn(messages, f)) return null;
	return `${list === 'vaccinations' ? 'Vaccine' : 'Medication'} ${Number(index) + 1}: ${messages[f]}`;
}

// Blocks are numbered from 0 with no gap, as the page adds them; reading stops at the first missing one or the cap.
function blocks<T>(
	form: FormData,
	cap: number,
	probe: (i: number) => string,
	read: (i: number) => T
) {
	const out: T[] = [];
	for (let i = 0; i < cap && form.has(probe(i)); i++) out.push(read(i));
	return out;
}

type Read = {
	values: OutcomeFormValues;
	revision: number | null;
	input: EntryBody | null;
	field: string | null;
	message: string | null;
};

// Checked in the API's order: revision, notes, growth, vaccinations, medications, then something to record.
// Notes are sent only when they differ from the ones the page showed; blank keeps the stored ones.
export function readOutcomeForm(form: FormData, mode: OutcomeMode): Read {
	const text = (k: string) => String(form.get(k) ?? '');
	const add = Object.fromEntries(
		(Object.keys(ADD) as OutcomeSection[]).map((s) => [
			s,
			mode.sections[s] && form.get(ADD[s]) === 'on'
		])
	) as OutcomeSections;
	const growth = readGrowthForm(form, { kind: 'create' }, { prefix: GROWTH_PREFIX });
	const vaccinations = blocks(
		form,
		MAX_OUTCOME_VACCINATIONS,
		(i) => `${vaccinePrefix(i)}given_on`,
		(i) => readVaccineForm(form, { kind: 'create' }, { prefix: vaccinePrefix(i) })
	);
	const medications = blocks(
		form,
		MAX_OUTCOME_MEDICATIONS,
		(i) => `${medicationPrefix(i)}name`,
		(i) =>
			readMedicationForm(form, { kind: 'create', zone: mode.zone }, { prefix: medicationPrefix(i) })
	);
	const values: OutcomeFormValues = {
		outcome_notes: text('outcome_notes'),
		add,
		growth: growth.values,
		vaccinations: vaccinations.map((v) => v.values),
		medications: medications.map((m) => m.values)
	};
	const refused = (field: string, message: string | null = null): Read => ({
		values,
		revision: null,
		input: null,
		field,
		message: message ?? outcomeFieldMessage(field)
	});

	const rev = text('revision');
	if (!/^\d{1,15}$/.test(rev)) return refused('revision');
	const input: EntryBody = { revision: Number(rev) };

	const notes = values.outcome_notes.trim();
	if (notes !== '' && notes !== text('outcome_notes_original').trim()) {
		if ([...notes].length > VISIT_OUTCOME_NOTES_MAX_LEN) return refused('outcome_notes');
		input.outcome_notes = notes;
	}

	if (add.growth) {
		if (!growth.input) return refused(growthPath(growth.field!), growth.message);
		input.growth = growth.input;
	}
	if (add.vaccinations) {
		const bad = vaccinations.findIndex((v) => !v.input);
		if (bad >= 0) return refused(vaccinePath(bad, vaccinations[bad].field!));
		if (vaccinations.length > 0) input.vaccinations = vaccinations.map((v) => v.input);
	}
	if (add.medications) {
		const bad = medications.findIndex((m) => !m.input);
		if (bad >= 0) return refused(medicationPath(bad, medications[bad].field!));
		if (medications.length > 0) input.medications = medications.map((m) => m.input);
	}
	if (Object.keys(input).length === 1) return refused('outcome');
	return { values, revision: input.revision as number, input, field: null, message: null };
}

// The visit as the page should now show it: its revision and the notes a new save compares with.
export type OutcomeCurrent = { revision: number; notes: string };
export const outcomeCurrent = (v: Pick<Visit, 'revision' | 'outcomeNotes'>): OutcomeCurrent => ({
	revision: v.revision,
	notes: v.outcomeNotes ?? ''
});

export const outcomeFieldFailure = <T extends object>(
	values: OutcomeFormValues,
	field: string,
	message: string | null,
	extra: T
) => fail(422, { ...extra, values, field, message, current: null as OutcomeCurrent | null });

// A stale form re-renders at current's revision with every typed value.
export const outcomeStale = <T extends object>(
	values: OutcomeFormValues,
	current: OutcomeCurrent,
	extra: T
) =>
	fail(409, {
		...extra,
		values,
		field: null as string | null,
		message: VISIT_REFUSALS.stale,
		current
	});

// A refused field lands in its block; a visit rule or a cap on the form.
export function outcomeApiFailure<T extends object>(
	values: OutcomeFormValues,
	r: ApiFailure,
	current: OutcomeCurrent,
	extra: T
) {
	if (r.reason === 'invalid' && r.field) {
		const message = outcomeFieldMessage(r.field, { api: true });
		if (message !== null) return outcomeFieldFailure(values, r.field, message, extra);
		if (r.field === 'revision') return outcomeStale(values, current, extra);
	}
	if (r.reason === 'conflict' && r.code === 'stale') return outcomeStale(values, current, extra);
	const base = {
		...extra,
		values,
		field: null as string | null,
		current: null as OutcomeCurrent | null
	};
	if (r.reason === 'conflict' && r.code === 'not_past') {
		return fail(409, { ...base, message: VISIT_REFUSALS.not_past });
	}
	if (r.reason === 'conflict' && r.code === 'limit') {
		return fail(409, { ...base, message: OUTCOME_LIMIT_MESSAGE });
	}
	return failAction(r, base);
}
