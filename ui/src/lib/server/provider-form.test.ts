import { describe, expect, it, vi } from 'vitest';
import { NPI_STATES, VISIT_PROVIDER_MAX_LEN } from '../constants';
import type { ProviderResult } from '../types';
import {
	PROVIDER_FIELD_MESSAGES,
	findProviders,
	pickProvider,
	readProviderQuery
} from './provider-form';
import type { Api, ApiResult } from './api';
import type { ProviderPage } from '../types';
import { visitCreateValues } from './visit-form';

const form = (fields: Record<string, string>) => {
	const data = new FormData();
	for (const [k, v] of Object.entries(fields)) data.append(k, v);
	return data;
};
const read = (fields: Record<string, string>) => readProviderQuery(form(fields));
const person = { npi_kind: 'person', npi_last_name: 'Quill', npi_state: 'WA' };
const clinic = { npi_kind: 'organization', npi_organization: 'Quill & Sons 2', npi_state: 'OR' };

const AVERY: ProviderResult = {
	npi: '9000000015',
	name: 'AVERY QUILL',
	credential: 'MD',
	taxonomy: 'Pediatrics',
	address: '12 PINE ST, SUITE 4, SEATTLE, WA 98101',
	phone: '206-555-0100'
};
const CLINIC: ProviderResult = {
	npi: '9000000049',
	name: 'QUILLFEATHER PEDIATRICS',
	credential: null,
	taxonomy: null,
	address: '40 ELM AVE, SEATTLE, WA 98101',
	phone: null
};
// The hidden fields a results list posts back.
const rows = (results: ProviderResult[], more = false) => ({
	...(more ? { npi_more: '1' } : {}),
	...Object.fromEntries(
		results.flatMap((r, i) =>
			Object.entries(r).map(([k, v]) => [`npi_r${i}_${k}`, v ?? ''] as [string, string])
		)
	)
});

describe('readProviderQuery', () => {
	it('reads a doctor search, trimmed, with the state upper-cased and blank optional terms left out', () => {
		expect(
			read({
				...person,
				npi_last_name: ' Quill ',
				npi_first_name: ' ',
				npi_state: 'wa',
				npi_city: '',
				npi_postal_code: ''
			})
		).toEqual({
			query: {
				kind: 'person',
				last_name: ' Quill ',
				first_name: ' ',
				organization: '',
				state: 'wa',
				city: '',
				postal_code: ''
			},
			input: { kind: 'person', lastName: 'Quill', state: 'WA' },
			field: null
		});
	});

	it('sends every term given', () => {
		expect(
			read({
				...person,
				npi_first_name: 'Avery',
				npi_city: 'Seattle',
				npi_postal_code: ' 98101 '
			}).input
		).toEqual({
			kind: 'person',
			lastName: 'Quill',
			firstName: 'Avery',
			state: 'WA',
			city: 'Seattle',
			postalCode: '98101'
		});
	});

	it('reads a clinic search with digits and an ampersand, and drops the doctor fields', () => {
		expect(read({ ...clinic, npi_last_name: 'Quill', npi_first_name: 'Avery' }).input).toEqual({
			kind: 'organization',
			organization: 'Quill & Sons 2',
			state: 'OR'
		});
	});

	it('takes letters of any script and the punctuation of names', () => {
		for (const name of ['Núñez', "O'Brien-Smith", 'St. John', 'Ли']) {
			expect(read({ ...person, npi_last_name: name }).field).toBeNull();
		}
	});

	it('searches for a doctor unless a clinic is chosen', () => {
		for (const npi_kind of ['', 'robot']) {
			expect(read({ ...person, npi_kind }).input).toMatchObject({ kind: 'person' });
		}
	});

	it('names the first refused term in the API’s order', () => {
		const cases: [Record<string, string>, string][] = [
			[{ ...person, npi_last_name: '', npi_state: '' }, 'last_name'],
			[{ ...person, npi_last_name: 'Q' }, 'last_name'],
			[{ ...person, npi_last_name: 'Qu*' }, 'last_name'],
			[{ ...person, npi_last_name: '9ill' }, 'last_name'],
			[{ ...person, npi_last_name: 'x'.repeat(61) }, 'last_name'],
			[{ ...person, npi_first_name: 'A', npi_state: '' }, 'first_name'],
			[{ ...clinic, npi_organization: '' }, 'organization'],
			[{ ...clinic, npi_organization: 'Q$ Clinic' }, 'organization'],
			[{ ...person, npi_state: '', npi_city: 'S' }, 'state'],
			[{ ...person, npi_state: 'XX' }, 'state'],
			[{ ...person, npi_city: 'S', npi_postal_code: '1' }, 'city'],
			[{ ...person, npi_city: 'Seattle 9' }, 'city'],
			[{ ...person, npi_postal_code: '9810' }, 'postal_code'],
			[{ ...person, npi_postal_code: '98101-2756' }, 'postal_code']
		];
		for (const [fields, field] of cases) {
			expect(read(fields)).toMatchObject({ input: null, field });
		}
	});

	it('takes every state code of the API', () => {
		for (const npi_state of NPI_STATES) expect(read({ ...person, npi_state }).field).toBeNull();
	});

	it('has the page spec’s field messages, never a typed term', () => {
		expect(PROVIDER_FIELD_MESSAGES).toEqual({
			last_name: 'Enter a last name of 2 to 60 letters',
			first_name: 'Enter a name of 2 to 60 letters',
			organization: 'Enter a name of 2 to 60 letters',
			state: 'Choose a state',
			city: 'Enter a city of 2 to 60 letters',
			postal_code: 'Enter a 5-digit ZIP code',
			choice: 'Choose a provider'
		});
	});
});

describe('findProviders', () => {
	const values = visitCreateValues(new Date('2026-12-03T18:30:00Z'), 'UTC');
	const api = (r: ApiResult<ProviderPage>) => {
		const search = vi.fn(async () => r);
		return { search, api: { providers: { search } } as unknown as Pick<Api, 'providers'> };
	};

	it('returns the page with the typed visit values and query', async () => {
		const { api: a, search } = api({ ok: true, data: { results: [AVERY], more: true } });
		const r = await findProviders(a, form(person), values);
		expect(search).toHaveBeenCalledWith({ kind: 'person', lastName: 'Quill', state: 'WA' });
		expect(r).toEqual({
			values,
			providers: {
				query: expect.objectContaining({ last_name: 'Quill', state: 'WA' }),
				page: { results: [AVERY], more: true }
			}
		});
	});

	it('refuses a term on its field without calling the API', async () => {
		const { api: a, search } = api({ ok: true, data: { results: [], more: false } });
		expect(await findProviders(a, form({ ...person, npi_state: '' }), values)).toMatchObject({
			status: 422,
			data: {
				values,
				providers: { field: 'state', message: 'Choose a state' }
			}
		});
		expect(search).not.toHaveBeenCalled();
	});

	it('says rate limited, and unavailable for anything else, keeping the values', async () => {
		const cases: [ApiResult<ProviderPage>, string][] = [
			[{ ok: false, status: 429, reason: 'rate_limited', code: 'rate_limited' }, 'rate_limited'],
			[{ ok: false, status: 503, reason: 'unavailable', code: 'unavailable' }, 'unavailable'],
			[{ ok: false, status: 0, reason: 'unavailable' }, 'unavailable'],
			[{ ok: false, status: 404, reason: 'not_found', code: 'not_found' }, 'unavailable'],
			[{ ok: false, status: 200, reason: 'error' }, 'unavailable'],
			[{ ok: false, status: 422, reason: 'invalid', field: 'zzz' }, 'unavailable']
		];
		for (const [result, error] of cases) {
			expect(await findProviders(api(result).api, form(person), values)).toEqual({
				values,
				providers: { query: expect.anything(), error }
			});
		}
	});

	it('puts a term the API refused on its field', async () => {
		const r = await findProviders(
			api({ ok: false, status: 422, reason: 'invalid', field: 'city' }).api,
			form(person),
			values
		);
		expect(r).toMatchObject({ status: 422, data: { providers: { field: 'city' } } });
	});
});

describe('pickProvider', () => {
	const values = visitCreateValues(new Date('2026-12-03T18:30:00Z'), 'UTC');
	const typed = { ...values, kind: 'dental', note: 'Check-up', location: 'old place' };

	it('fills the provider with its credential, the place and the registry number, and closes the panel', () => {
		const r = pickProvider(form({ ...person, ...rows([CLINIC, AVERY]), npi_choice: '1' }), typed);
		expect(r).toEqual({
			values: {
				...typed,
				provider: 'AVERY QUILL, MD',
				location: '12 PINE ST, SUITE 4, SEATTLE, WA 98101',
				provider_npi: '9000000015',
				provider_npi_for: 'AVERY QUILL, MD'
			},
			providers: { query: expect.objectContaining({ last_name: 'Quill' }) }
		});
	});

	it('leaves the credential off when there is none or it would pass the provider limit', () => {
		expect(pickProvider(form({ ...rows([CLINIC]), npi_choice: '0' }), typed)).toMatchObject({
			values: { provider: 'QUILLFEATHER PEDIATRICS' }
		});
		const longest = 'A'.repeat(VISIT_PROVIDER_MAX_LEN - 4);
		const fits = { ...AVERY, name: longest, credential: 'MD' };
		expect(pickProvider(form({ ...rows([fits]), npi_choice: '0' }), typed)).toMatchObject({
			values: { provider: `${longest}, MD` }
		});
		const long = { ...AVERY, name: `${longest}A`, credential: 'MD' };
		expect(pickProvider(form({ ...rows([long]), npi_choice: '0' }), typed)).toMatchObject({
			values: { provider: `${longest}A`, provider_npi_for: `${longest}A` }
		});
	});

	it('asks for a choice and shows the same results again', () => {
		for (const npi_choice of [undefined, '', '2', '-1', 'x']) {
			const fields = { ...person, ...rows([AVERY, CLINIC], true) };
			const r = pickProvider(
				form(npi_choice === undefined ? fields : { ...fields, npi_choice }),
				typed
			);
			expect(r).toMatchObject({
				status: 422,
				data: {
					values: typed,
					providers: {
						field: 'choice',
						message: 'Choose a provider',
						page: { results: [AVERY, CLINIC], more: true }
					}
				}
			});
		}
	});
});
