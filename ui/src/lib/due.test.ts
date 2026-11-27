import { describe, expect, it } from 'vitest';
import { dueLines } from './due';
import { dueLabel } from './vaccines';
import type { HomeDue, HomeDueMedication, HomeDueVaccine, HomeDueVisit } from './types';

const KID = 'k-1';
const med = (n: number, state: 'due' | 'overdue' = 'due', dueAt?: string): HomeDueMedication => ({
	kind: 'medication',
	medication: { id: `m-${n}`, name: `Med ${n}` },
	dueAt: dueAt ?? '2026-11-23T14:00:00Z',
	state
});
const vax = (n: number, status: 'due' | 'overdue' = 'due'): HomeDueVaccine => ({
	kind: 'vaccine',
	from: { n, unit: 'y' },
	status,
	count: 1
});
const visit = (n: number, startsAt = '2026-11-26T09:30:00Z'): HomeDueVisit => ({
	kind: 'visit',
	id: `v-${n}`,
	startsAt,
	visitKind: 'well_child'
});
const lines = (due: HomeDue[], max = 3) => dueLines(due, max, KID, 'UTC', '2026-11-23');
const meds = (n: number) => Array.from({ length: n }, (_, i) => med(i + 1));
const vaxes = (n: number) => Array.from({ length: n }, (_, i) => vax(i + 1));
const visits = (n: number) => Array.from({ length: n }, (_, i) => visit(i + 1));

describe('dueLines', () => {
	it('is empty with nothing due', () => {
		expect(lines([])).toEqual([]);
	});

	it('reads a medication entry as its name and kid-local time, linked to the Medication page', () => {
		expect(lines([med(1)])).toEqual([
			{ kind: 'medication', text: 'Med 1 · 14:00', href: `/kids/${KID}/medication` }
		]);
	});

	it('adds overdue as text after the time, and nothing for due', () => {
		expect(lines([med(1, 'overdue')])[0].text).toBe('Med 1 · 14:00 · overdue');
		expect(lines([med(1, 'due')])[0].text).toBe('Med 1 · 14:00');
	});

	it('shows the time in the kid zone, with the date when it is not the kid-local today', () => {
		const at = '2026-11-23T23:30:00Z';
		expect(dueLines([med(1, 'due', at)], 3, KID, 'Europe/Madrid', '2026-11-23')[0].text).toBe(
			'Med 1 · 2026-11-24 00:30'
		);
		expect(dueLines([med(1, 'due', at)], 3, KID, 'Europe/Madrid', '2026-11-24')[0].text).toBe(
			'Med 1 · 00:30'
		);
	});

	it('keeps a vaccine entry as dueLabel linked to the Vaccines page', () => {
		const entry = vax(2, 'overdue');
		expect(lines([entry])).toEqual([
			{ kind: 'vaccine', text: dueLabel(entry), href: `/kids/${KID}/vaccines` }
		]);
	});

	it('lists vaccine-only entries as the home always did: three, then one more line', () => {
		expect(lines(vaxes(3))).toEqual(
			vaxes(3).map((e) => ({ kind: 'vaccine', text: dueLabel(e), href: `/kids/${KID}/vaccines` }))
		);
		expect(lines(vaxes(5)).map((l) => l.text)).toEqual([
			'1-year vaccines · Due',
			'2-year vaccines · Due',
			'3-year vaccines · Due',
			'2 more on the Vaccines page'
		]);
		expect(lines(vaxes(5)).at(-1)?.href).toBe(`/kids/${KID}/vaccines`);
	});

	it('keeps the API order, medication first, whatever the entries are', () => {
		expect(lines([...meds(2), ...vaxes(3)]).map((l) => l.text)).toEqual([
			'Med 1 · 14:00',
			'Med 2 · 14:00',
			'1-year vaccines · Due',
			'2 more on the Vaccines page'
		]);
		expect(lines([vax(1), med(1)]).map((l) => l.text)).toEqual([
			'1-year vaccines · Due',
			'Med 1 · 14:00'
		]);
	});

	it('has no more line for 0 and 2 medication entries with 0 or 3 vaccine groups when all fit', () => {
		expect(lines(meds(2))).toHaveLength(2);
		expect(lines([...meds(0), ...vaxes(3)])).toHaveLength(3);
	});

	it('says "1 more" for a single hidden entry of either kind', () => {
		expect(lines(meds(4)).at(-1)?.text).toBe('1 more on the Medication page');
		expect(lines(vaxes(4)).at(-1)?.text).toBe('1 more on the Vaccines page');
	});

	it('counts what is hidden per kind: 5 medication entries with no vaccine group', () => {
		expect(lines(meds(5)).map((l) => l.text)).toEqual([
			'Med 1 · 14:00',
			'Med 2 · 14:00',
			'Med 3 · 14:00',
			'2 more on the Medication page'
		]);
		expect(lines(meds(5)).at(-1)?.href).toBe(`/kids/${KID}/medication`);
	});

	it('shows one more line per hidden kind, Medication then Vaccines, each linking its page', () => {
		const out = lines([...meds(5), ...vaxes(3)]);
		expect(out.map((l) => l.text)).toEqual([
			'Med 1 · 14:00',
			'Med 2 · 14:00',
			'Med 3 · 14:00',
			'2 more on the Medication page',
			'3 more on the Vaccines page'
		]);
		expect(out.slice(-2).map((l) => l.href)).toEqual([
			`/kids/${KID}/medication`,
			`/kids/${KID}/vaccines`
		]);
	});

	it('cuts inside the vaccine groups when the medication entries all fit: only a Vaccines line', () => {
		expect(lines([...meds(2), ...vaxes(3)], 3).filter((l) => l.text.includes(' more '))).toEqual([
			{ kind: 'vaccine', text: '2 more on the Vaccines page', href: `/kids/${KID}/vaccines` }
		]);
	});

	it('applies no window of its own: an overdue entry from yesterday is listed', () => {
		expect(lines([med(1, 'overdue', '2026-11-22T23:50:00Z')])[0].text).toBe(
			'Med 1 · 2026-11-22 23:50 · overdue'
		);
	});

	it('follows the max it is given', () => {
		expect(lines(meds(3), 1).map((l) => l.text)).toEqual([
			'Med 1 · 14:00',
			'2 more on the Medication page'
		]);
	});
	it('reads a visit as its kind, weekday, date and time on the kid clock, linked to the visit', () => {
		expect(lines([visit(1)])).toEqual([
			{
				kind: 'visit',
				text: 'Well-child visit, Thu 26 Nov, 09:30',
				href: `/kids/${KID}/visits/v-1`
			}
		]);
		const late: HomeDueVisit = { ...visit(2, '2026-11-26T23:30:00Z'), visitKind: 'dental' };
		expect(dueLines([late], 3, KID, 'Europe/Madrid', '2026-11-25')[0].text).toBe(
			'Dental visit, Fri 27 Nov, 00:30'
		);
		expect(dueLines([late], 3, KID, 'America/Los_Angeles', '2026-11-25')[0].text).toBe(
			'Dental visit, Thu 26 Nov, 15:30'
		);
	});

	it('keeps visits after medication and vaccines, in API order', () => {
		expect(lines([med(1), vax(1), visit(1)]).map((l) => l.kind)).toEqual([
			'medication',
			'vaccine',
			'visit'
		]);
	});

	it('counts hidden visits as more on the Visits page, after the other kinds', () => {
		expect(lines(visits(4)).at(-1)).toEqual({
			kind: 'visit',
			text: '1 more on the Visits page',
			href: `/kids/${KID}/visits`
		});
		const out = lines([...meds(4), ...vaxes(2), ...visits(3)]);
		expect(out.map((l) => l.text).slice(3)).toEqual([
			'1 more on the Medication page',
			'2 more on the Vaccines page',
			'3 more on the Visits page'
		]);
		expect(out.slice(3).map((l) => l.href)).toEqual([
			`/kids/${KID}/medication`,
			`/kids/${KID}/vaccines`,
			`/kids/${KID}/visits`
		]);
	});

	it('has no Visits line when every visit fits', () => {
		expect(lines([med(1), ...visits(2)]).map((l) => l.text)).toEqual([
			'Med 1 · 14:00',
			'Well-child visit, Thu 26 Nov, 09:30',
			'Well-child visit, Thu 26 Nov, 09:30'
		]);
	});
});
