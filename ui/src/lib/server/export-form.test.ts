import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { EXPORT_MAX_DAYS, EXPORT_MODULES } from '../constants';
import { EXPORT_DEFAULT_DAYS } from './constants';
import {
	EXPORT_MESSAGES,
	exportDefaults,
	exportRetrySearch,
	readExportForm,
	readExportSearch
} from './export-form';

const UTC = 'UTC';
// 2026-11-24 12:00 UTC is already 25 November in Auckland.
const NOW = Date.UTC(2026, 10, 24, 12);
const TODAY = '2026-11-24';
const MS_PER_DAY = 86_400_000;
const addDays = (date: string, n: number) =>
	new Date(Date.parse(`${date}T00:00:00Z`) + n * MS_PER_DAY).toISOString().slice(0, 10);
const form = (fields: [string, string][]) => {
	const data = new FormData();
	for (const [k, v] of fields) data.append(k, v);
	return data;
};
const read = (fields: [string, string][], zone = UTC) => readExportForm(form(fields), zone);
const RANGE: [string, string][] = [
	['from', '2026-11-01'],
	['to', '2026-11-24']
];

beforeEach(() => {
	vi.useFakeTimers({ now: NOW, toFake: ['Date'] });
});
afterEach(() => {
	vi.useRealTimers();
});

describe('constants', () => {
	it('mirrors the API: 93 days, the twelve module ids in model.Modules order', () => {
		expect(EXPORT_MAX_DAYS).toBe(93);
		expect(EXPORT_DEFAULT_DAYS).toBe(30);
		expect(EXPORT_MODULES).toEqual([
			'bottles',
			'nursing',
			'pumping',
			'diapers',
			'sleep',
			'feeding',
			'milestones',
			'growth',
			'vaccination',
			'sickness',
			'medication',
			'visits'
		]);
	});
});

describe('exportDefaults', () => {
	it('is the last EXPORT_DEFAULT_DAYS kid-local days, today included', () => {
		expect(exportDefaults('2025-01-01', UTC)).toEqual({ from: '2026-10-26', to: TODAY });
		expect(exportDefaults('2025-01-01', 'Pacific/Auckland')).toEqual({
			from: '2026-10-27',
			to: '2026-11-25'
		});
	});

	it('starts on the birth date for a kid younger than EXPORT_DEFAULT_DAYS days (P-31)', () => {
		expect(exportDefaults(addDays(TODAY, -10), UTC)).toEqual({
			from: addDays(TODAY, -10),
			to: TODAY
		});
		expect(exportDefaults(TODAY, UTC)).toEqual({ from: TODAY, to: TODAY });
	});
});

describe('readExportForm', () => {
	it('reads the range and the modules', () => {
		expect(read([...RANGE, ['modules', 'bottles'], ['modules', 'growth']])).toEqual({
			from: '2026-11-01',
			to: '2026-11-24',
			modules: ['bottles', 'growth']
		});
	});

	it('normalises the order to EXPORT_MODULES and drops a duplicate', () => {
		expect(
			read([
				...RANGE,
				['modules', 'medication'],
				['modules', 'growth'],
				['modules', 'bottles'],
				['modules', 'growth']
			])
		).toEqual({
			from: '2026-11-01',
			to: '2026-11-24',
			modules: ['bottles', 'growth', 'medication']
		});
	});

	it('refuses no module as modules', () => {
		expect(read(RANGE)).toEqual({ field: 'modules' });
		expect(read([...RANGE, ['modules', '']])).toEqual({ field: 'modules' });
	});

	it.each(['bogus', 'Bottles', '../x', 'bottles,growth', ' bottles', '__proto__', 'toString'])(
		'refuses the unknown or hostile id %j as modules, never sending it',
		(id) => {
			expect(read([...RANGE, ['modules', 'bottles'], ['modules', id]])).toEqual({
				field: 'modules'
			});
		}
	);

	it('accepts exactly EXPORT_MAX_DAYS days and refuses one more as to', () => {
		const from = addDays(TODAY, -(EXPORT_MAX_DAYS - 1));
		expect(
			read([
				['from', from],
				['to', TODAY],
				['modules', 'sleep']
			])
		).toEqual({ from, to: TODAY, modules: ['sleep'] });
		expect(
			read([
				['from', addDays(from, -1)],
				['to', TODAY],
				['modules', 'sleep']
			])
		).toEqual({ field: 'to' });
	});

	it('refuses an end before the start or after the kid-local today as to', () => {
		const range = (from: string, to: string, zone = UTC) =>
			read(
				[
					['from', from],
					['to', to],
					['modules', 'sleep']
				],
				zone
			);
		expect(range('2026-11-10', '2026-11-09')).toEqual({ field: 'to' });
		expect(range('2026-11-10', '2026-11-25')).toEqual({ field: 'to' });
		expect(range('2026-11-10', '2026-11-25', 'Pacific/Auckland')).toEqual({
			from: '2026-11-10',
			to: '2026-11-25',
			modules: ['sleep']
		});
	});

	it.each([
		'',
		'2026-02-30',
		'2026-1-01',
		'26-11-01',
		'2026-11-01T00:00',
		'2026-11-01<script>',
		'../2026-11-01',
		'2026/11/01'
	])('refuses the hostile date %j in the API order: from, then to', (date) => {
		expect(
			read([
				['from', date],
				['to', 'nonsense'],
				['modules', 'bogus']
			])
		).toEqual({ field: 'from' });
		expect(
			read([
				['from', '2026-11-01'],
				['to', date],
				['modules', 'bogus']
			])
		).toEqual({ field: 'to' });
	});

	it('trims a date', () => {
		expect(
			read([
				['from', ' 2026-11-01 '],
				['to', '2026-11-02 '],
				['modules', 'sleep']
			])
		).toEqual({ from: '2026-11-01', to: '2026-11-02', modules: ['sleep'] });
	});

	it('has one value-free message per field and for busy', () => {
		expect(Object.keys(EXPORT_MESSAGES).sort()).toEqual(['busy', 'from', 'modules', 'to']);
		expect(EXPORT_MESSAGES.to).toContain(String(EXPORT_MAX_DAYS));
	});
});

describe('the validation redirect', () => {
	it('carries the error, valid dates and the known ids in EXPORT_MODULES order', () => {
		expect(
			exportRetrySearch(
				'to',
				form([...RANGE, ['modules', 'growth'], ['modules', 'bottles'], ['modules', 'growth']])
			)
		).toBe('error=to&from=2026-11-01&to=2026-11-24&modules=bottles,growth');
		expect(exportRetrySearch('modules', form(RANGE))).toBe(
			'error=modules&from=2026-11-01&to=2026-11-24&modules='
		);
	});

	it('never carries a typed value that is not a date or a module id', () => {
		const search = exportRetrySearch(
			'from',
			form([
				['from', 'Secret words'],
				['to', '2026-11-24<x>'],
				['modules', 'bogus'],
				['modules', 'sleep']
			])
		);
		expect(search).toBe('error=from&modules=sleep');
	});

	it('reads back only an enumerated error, valid dates and known ids', () => {
		expect(
			readExportSearch(
				new URLSearchParams('error=to&from=2026-11-01&to=2026-11-24&modules=growth,bottles,bogus')
			)
		).toEqual({
			error: 'to',
			from: '2026-11-01',
			to: '2026-11-24',
			modules: ['bottles', 'growth']
		});
		expect(readExportSearch(new URLSearchParams('error=modules&modules='))).toEqual({
			error: 'modules',
			from: null,
			to: null,
			modules: []
		});
		expect(readExportSearch(new URLSearchParams('error=<b>&from=x&to=2026-02-30'))).toEqual({
			error: null,
			from: null,
			to: null,
			modules: null
		});
		expect(readExportSearch(new URLSearchParams('error=busy'))).toMatchObject({ error: 'busy' });
	});
});
