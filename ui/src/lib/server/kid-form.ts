import { fail } from '@sveltejs/kit';
import type { ApiResult } from './api';
import { failAction } from './outcome';
import {
	DEFAULT_GROWTH_SOURCE,
	DEFAULT_TIME_ZONE,
	GESTATION_DAYS,
	GESTATION_WEEKS,
	KID_MIN_BIRTH_DATE,
	KID_NAME_MAX_LEN
} from '../constants';
import { coverageHint } from '../growth';
import type { GrowthSourceInfo, Kid, KidInput, VaccineSchemaInfo } from '../types';

export type KidField =
	| 'name'
	| 'birth_date'
	| 'sex'
	| 'gestation_weeks'
	| 'gestation_days'
	| 'time_zone'
	| 'growth_source'
	| 'vaccine_schema';

export type KidFormValues = {
	name: string;
	birthDate: string;
	sex: string;
	bornEarly: boolean;
	weeks: string;
	days: string;
	timeZone: string;
	growthSource: string;
	vaccineSchema: string;
};

export const KID_FIELD_MESSAGES: Record<KidField, string> = {
	name: `Enter a name of 1 to ${KID_NAME_MAX_LEN} characters`,
	birth_date: `Enter a birth date from ${KID_MIN_BIRTH_DATE} to today`,
	sex: 'Choose female or male',
	gestation_weeks: `Enter weeks from ${GESTATION_WEEKS.min} to ${GESTATION_WEEKS.max}`,
	gestation_days: `Enter days from ${GESTATION_DAYS.min} to ${GESTATION_DAYS.max}`,
	time_zone: 'Choose a time zone from the list',
	growth_source: 'Choose a growth reference from the list',
	vaccine_schema: 'Choose a vaccination schedule from the list'
};

const isKidField = (f: string | undefined): f is KidField =>
	f !== undefined && Object.hasOwn(KID_FIELD_MESSAGES, f);

export function kidValues(kid?: Kid): KidFormValues {
	return {
		name: kid?.name ?? '',
		birthDate: kid?.birthDate ?? '',
		sex: kid?.sex ?? '',
		bornEarly: kid ? kid.gestationWeeks !== null : false,
		weeks: kid?.gestationWeeks?.toString() ?? '',
		days: kid?.gestationDays?.toString() ?? '',
		timeZone: kid?.timeZone ?? DEFAULT_TIME_ZONE,
		growthSource: kid?.growthSource ?? DEFAULT_GROWTH_SOURCE,
		vaccineSchema: kid?.vaccineSchema ?? ''
	};
}

const wholeNumber = (s: string) => (/^\d{1,3}$/.test(s) ? Number(s) : null);

// With "Born early?" off the gestation is sent as null, which clears it on PATCH; a form without a growth
// source or schedule field keeps the stored one, and a blank schedule sends null (no schedule).
export function readKidForm(form: FormData): {
	values: KidFormValues;
	input: KidInput | null;
	field: KidField | null;
} {
	const text = (k: string) => String(form.get(k) ?? '');
	const values: KidFormValues = {
		name: text('name'),
		birthDate: text('birth_date'),
		sex: text('sex'),
		bornEarly: form.get('born_early') !== null,
		weeks: text('gestation_weeks').trim(),
		days: text('gestation_days').trim(),
		timeZone: text('time_zone'),
		growthSource: text('growth_source'),
		vaccineSchema: text('vaccine_schema')
	};
	const weeks = values.bornEarly ? wholeNumber(values.weeks) : null;
	const days = values.bornEarly ? wholeNumber(values.days) : null;
	if (values.bornEarly && weeks === null) return { values, input: null, field: 'gestation_weeks' };
	if (values.bornEarly && days === null) return { values, input: null, field: 'gestation_days' };
	const input: KidInput = {
		name: values.name,
		birthDate: values.birthDate,
		sex: values.sex,
		gestationWeeks: weeks,
		gestationDays: days,
		timeZone: values.timeZone
	};
	if (values.growthSource !== '') input.growthSource = values.growthSource;
	if (form.get('vaccine_schema') !== null) input.vaccineSchema = values.vaccineSchema || null;
	return { values, input, field: null };
}

export function kidFieldFailure<T extends object>(
	values: KidFormValues,
	field: KidField,
	extra: T
) {
	return fail(400, { ...extra, values, field, message: KID_FIELD_MESSAGES[field] });
}

export function kidApiFailure<T extends object>(
	values: KidFormValues,
	r: Extract<ApiResult<unknown>, { ok: false }>,
	extra: T
) {
	if (r.reason === 'invalid' && isKidField(r.field)) return kidFieldFailure(values, r.field, extra);
	return failAction(r, { ...extra, values, field: null as KidField | null });
}

// The source select offers exactly the API's list, each with its coverage on one line.
export const sourceOptions = (sources: GrowthSourceInfo[]) =>
	sources.map((s) => ({ id: s.id, name: s.name, hint: coverageHint(s.coverage) }));

// The schedule select offers exactly the API's list after "None yet".
export const scheduleOptions = (schemas: VaccineSchemaInfo[]) =>
	schemas.map((s) => ({ id: s.id, label: `${s.name} — ${s.authority}, ${s.edition}` }));
