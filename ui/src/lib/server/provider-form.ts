import { fail } from '@sveltejs/kit';
import {
	NPI_NAME_MAX_LEN,
	NPI_NAME_MIN_LEN,
	NPI_NAME_PATTERN,
	NPI_ORG_PATTERN,
	NPI_POSTAL_CODE_LEN,
	NPI_STATES,
	VISIT_PROVIDER_MAX_LEN
} from '../constants';
import type { ProviderPage, ProviderQuery, ProviderResult } from '../types';
import type { Api } from './api';
import { failPage } from './outcome';
import type { VisitFormValues } from './visit-form';

// The panel's form names are npi_<key>; keys are the API's body keys, in its order of checks.
const KEYS = [
	'kind',
	'last_name',
	'first_name',
	'organization',
	'state',
	'city',
	'postal_code'
] as const;
export type ProviderQueryValues = Record<(typeof KEYS)[number], string>;
export type ProviderPanel = {
	query: ProviderQueryValues;
	page?: ProviderPage;
	error?: 'unavailable' | 'rate_limited';
	field?: string;
	message?: string;
};

const letters = `${NPI_NAME_MIN_LEN} to ${NPI_NAME_MAX_LEN} letters`;
// Keyed by the API's field names, and choice for "Use this provider" without one; never a typed term.
export const PROVIDER_FIELD_MESSAGES: Record<string, string> = {
	last_name: `Enter a last name of ${letters}`,
	first_name: `Enter a name of ${letters}`,
	organization: `Enter a name of ${letters}`,
	state: 'Choose a state',
	city: `Enter a city of ${letters}`,
	postal_code: `Enter a ${NPI_POSTAL_CODE_LEN}-digit ZIP code`,
	choice: 'Choose a provider'
};

const POSTAL = new RegExp(`^\\d{${NPI_POSTAL_CODE_LEN}}$`);

// The API's checks (plan 10 B4): a doctor sends names, a clinic its name; a blank optional term is not sent.
export function readProviderQuery(form: FormData): {
	query: ProviderQueryValues;
	input: ProviderQuery | null;
	field: string | null;
} {
	const query = Object.fromEntries(
		KEYS.map((k) => [k, String(form.get(`npi_${k}`) ?? '')])
	) as ProviderQueryValues;
	const refuse = (field: string) => ({ query, input: null, field });
	const kind = query.kind === 'organization' ? 'organization' : 'person';
	const input: ProviderQuery = { kind, state: '' };
	const t = (k: keyof ProviderQueryValues) => query[k].trim();
	if (kind === 'person') {
		if (!NPI_NAME_PATTERN.test(t('last_name'))) return refuse('last_name');
		input.lastName = t('last_name');
		if (t('first_name') !== '') {
			if (!NPI_NAME_PATTERN.test(t('first_name'))) return refuse('first_name');
			input.firstName = t('first_name');
		}
	} else {
		if (!NPI_ORG_PATTERN.test(t('organization'))) return refuse('organization');
		input.organization = t('organization');
	}
	const state = t('state').toUpperCase();
	if (!(NPI_STATES as readonly string[]).includes(state)) return refuse('state');
	input.state = state;
	if (t('city') !== '') {
		if (!NPI_NAME_PATTERN.test(t('city'))) return refuse('city');
		input.city = t('city');
	}
	if (t('postal_code') !== '') {
		if (!POSTAL.test(t('postal_code'))) return refuse('postal_code');
		input.postalCode = t('postal_code');
	}
	return { query, input, field: null };
}

const refused = (values: VisitFormValues, providers: ProviderPanel, field: string) =>
	fail(422, {
		values,
		providers: { ...providers, field, message: PROVIDER_FIELD_MESSAGES[field] }
	});

// ?/find: nothing is saved; the visit values come back as typed, with the results or why there are none.
export async function findProviders(
	api: Pick<Api, 'providers'>,
	form: FormData,
	values: VisitFormValues
) {
	const { query, input, field } = readProviderQuery(form);
	if (!input) return refused(values, { query }, field!);
	const r = await api.providers.search(input);
	if (r.ok) return { values, providers: { query, page: r.data } as ProviderPanel };
	if (r.reason === 'unauthorized') failPage(r);
	if (r.reason === 'invalid' && r.field && Object.hasOwn(PROVIDER_FIELD_MESSAGES, r.field)) {
		return refused(values, { query }, r.field);
	}
	const error = r.reason === 'rate_limited' ? 'rate_limited' : 'unavailable';
	return { values, providers: { query, error } as ProviderPanel };
}

// The results the panel posted back in its hidden fields, in their order.
function postedPage(form: FormData): ProviderPage {
	const results: ProviderResult[] = [];
	const get = (i: number, k: string) => String(form.get(`npi_r${i}_${k}`) ?? '');
	const orNull = (s: string) => (s === '' ? null : s);
	for (let i = 0; form.has(`npi_r${i}_npi`); i++) {
		results.push({
			npi: get(i, 'npi'),
			name: get(i, 'name'),
			credential: orNull(get(i, 'credential')),
			taxonomy: orNull(get(i, 'taxonomy')),
			address: get(i, 'address'),
			phone: orNull(get(i, 'phone'))
		});
	}
	return { results, more: form.get('npi_more') === '1' };
}

// ?/pick: fills the provider, the place and the registry number from the chosen row; Save sends them.
export function pickProvider(form: FormData, values: VisitFormValues) {
	const { query } = readProviderQuery(form);
	const page = postedPage(form);
	const choice = String(form.get('npi_choice') ?? '');
	const row = /^\d+$/.test(choice) ? page.results[Number(choice)] : undefined;
	if (!row) return refused(values, { query, page }, 'choice');
	const withCredential = row.credential ? `${row.name}, ${row.credential}` : row.name;
	const provider = [...withCredential].length <= VISIT_PROVIDER_MAX_LEN ? withCredential : row.name;
	return {
		values: {
			...values,
			provider,
			location: row.address,
			provider_npi: row.npi,
			provider_npi_for: provider
		},
		providers: { query } as ProviderPanel
	};
}
