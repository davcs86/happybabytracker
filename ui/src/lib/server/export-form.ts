import { EXPORT_MAX_DAYS, EXPORT_MODULES } from '../constants';
import { todayInOrUtc } from '../time-zone';
import type { ExportInput, ModuleId } from '../types';
import { EXPORT_DEFAULT_DAYS } from './constants';

export type ExportField = 'from' | 'to' | 'modules';
export type ExportError = ExportField | 'busy';

const MS_PER_DAY = 24 * 60 * 60 * 1000;

// Never a typed value in a message.
export const EXPORT_MESSAGES: Record<ExportError, string> = {
	from: 'Enter a start date that is not before the birth date',
	to: `Enter an end date from the start date to today, at most ${EXPORT_MAX_DAYS} days in all`,
	modules: 'Choose at least one module',
	busy: 'The PDF cannot be made right now, try again in a moment'
};
const ERRORS = Object.keys(EXPORT_MESSAGES) as ExportError[];

function validDate(s: string): boolean {
	const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s);
	if (!m) return false;
	const d = new Date(Date.UTC(+m[1], +m[2] - 1, +m[3]));
	return d.getUTCFullYear() === +m[1] && d.getUTCMonth() === +m[2] - 1 && d.getUTCDate() === +m[3];
}

const addDays = (date: string, n: number) =>
	new Date(Date.parse(`${date}T00:00:00Z`) + n * MS_PER_DAY).toISOString().slice(0, 10);
const isModule = (id: string): id is ModuleId => (EXPORT_MODULES as readonly string[]).includes(id);
// Known ids only, de-duplicated, in EXPORT_MODULES order.
const ordered = (ids: readonly string[]) => EXPORT_MODULES.filter((m) => ids.includes(m));

// Kid-local: the last EXPORT_DEFAULT_DAYS days, never starting before the birth date (P-31).
export function exportDefaults(birthDate: string, zone: string): { from: string; to: string } {
	const to = todayInOrUtc(zone);
	const from = addDays(to, -(EXPORT_DEFAULT_DAYS - 1));
	return { from: from < birthDate ? birthDate : from, to };
}

// The API's order: from, to, modules. The API alone checks from against the birth date.
export function readExportForm(form: FormData, zone: string): ExportInput | { field: ExportField } {
	const from = String(form.get('from') ?? '').trim();
	const to = String(form.get('to') ?? '').trim();
	if (!validDate(from)) return { field: 'from' };
	const days = (Date.parse(to) - Date.parse(from)) / MS_PER_DAY + 1;
	if (!validDate(to) || days < 1 || to > todayInOrUtc(zone) || days > EXPORT_MAX_DAYS) {
		return { field: 'to' };
	}
	const ids = form.getAll('modules').map(String);
	if (ids.length === 0 || !ids.every(isModule)) return { field: 'modules' };
	return { from, to, modules: ordered(ids) };
}

// The redirect after a refusal carries only enumerated values: the error, well-formed dates, known ids.
export function exportRetrySearch(error: ExportError, form: FormData): string {
	const p = new URLSearchParams({ error });
	for (const k of ['from', 'to'] as const) {
		const v = String(form.get(k) ?? '').trim();
		if (validDate(v)) p.set(k, v);
	}
	const modules = ordered(form.getAll('modules').map(String));
	return `${p}&modules=${modules.join(',')}`;
}

// null: the parameter is absent or not one the redirect writes, so the page's default applies.
export function readExportSearch(params: URLSearchParams) {
	const error = params.get('error');
	const date = (k: string) => {
		const v = params.get(k);
		return v !== null && validDate(v) ? v : null;
	};
	const modules = params.get('modules');
	return {
		error: ERRORS.find((e) => e === error) ?? null,
		from: date('from'),
		to: date('to'),
		modules: modules === null ? null : ordered(modules.split(','))
	};
}
