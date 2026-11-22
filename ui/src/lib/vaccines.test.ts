import { describe, expect, it } from 'vitest';
import {
	ageLabel,
	antigenName,
	dueLabel,
	groupTitle,
	schemaChosen,
	statusLabel,
	windowLabel
} from './vaccines';
import type { VaccineSchemaInfo } from './types';

describe('ageLabel', () => {
	it('names the age in its own unit, singular for one, birth for zero', () => {
		expect(ageLabel({ n: 8, unit: 'w' })).toBe('8 weeks');
		expect(ageLabel({ n: 1, unit: 'w' })).toBe('1 week');
		expect(ageLabel({ n: 4, unit: 'm' })).toBe('4 months');
		expect(ageLabel({ n: 1, unit: 'm' })).toBe('1 month');
		expect(ageLabel({ n: 11, unit: 'y' })).toBe('11 years');
		expect(ageLabel({ n: 1, unit: 'y' })).toBe('1 year');
		expect(ageLabel({ n: 3, unit: 'd' })).toBe('3 days');
		expect(ageLabel({ n: 1, unit: 'd' })).toBe('1 day');
		expect(ageLabel({ n: 0, unit: 'd' })).toBe('birth');
	});
});

describe('groupTitle', () => {
	it('titles a visit by its from age', () => {
		expect(groupTitle({ from: { n: 0, unit: 'd' } })).toBe('Birth');
		expect(groupTitle({ from: { n: 8, unit: 'w' } })).toBe('8-week vaccines');
		expect(groupTitle({ from: { n: 4, unit: 'm' } })).toBe('4-month vaccines');
		expect(groupTitle({ from: { n: 1, unit: 'y' } })).toBe('1-year vaccines');
		expect(groupTitle({ from: { n: 11, unit: 'y' } })).toBe('11-year vaccines');
	});
});

describe('status and due labels', () => {
	it('labels every status', () => {
		expect(
			(['given', 'due', 'upcoming', 'overdue', 'dismissed', 'available'] as const).map(statusLabel)
		).toEqual(['Given', 'Due', 'Upcoming', 'Overdue', 'Dismissed', 'Available']);
	});

	it('labels a home due entry by its visit and status', () => {
		const entry = { kind: 'vaccine', count: 3 } as const;
		expect(dueLabel({ ...entry, from: { n: 4, unit: 'm' }, status: 'overdue' })).toBe(
			'4-month vaccines · overdue'
		);
		expect(dueLabel({ ...entry, from: { n: 0, unit: 'd' }, status: 'due' })).toBe('Birth · due');
	});
});

describe('windowLabel', () => {
	const dates = { fromDate: '2025-03-10', untilDate: '2025-04-10' };
	it('names when a line falls due and when it stops being due, by kind', () => {
		expect(windowLabel({ ...dates, kind: 'routine' })).toBe(
			'Due from 2025-03-10 · overdue from 2025-04-10'
		);
		expect(windowLabel({ ...dates, kind: 'may_not_be_needed' })).toBe(
			'Due from 2025-03-10 · no longer due from 2025-04-10'
		);
		expect(windowLabel({ ...dates, kind: 'supplementary' })).toBe('Available from 2025-03-10');
		expect(windowLabel({ ...dates, kind: 'recurring' })).toBe('');
	});
});

describe('antigenName', () => {
	const catalogue = { antigens: [{ id: 'hib', name: 'Hib' }], products: [] };
	it('names a known antigen and falls back to the id for an unknown one', () => {
		expect(antigenName('hib', catalogue)).toBe('Hib');
		expect(antigenName('zzz', catalogue)).toBe('zzz');
	});
});

describe('schemaChosen', () => {
	const schema = (id: string): VaccineSchemaInfo => ({
		id,
		name: id,
		authority: 'a',
		edition: 'e',
		retrieved: '2026-11-22',
		originUrl: 'https://example.test',
		licence: 'l',
		attribution: 'a'
	});
	it('is true only for a schedule the API still ships', () => {
		const schemas = [schema('us'), schema('uk')];
		expect(schemaChosen('us', schemas)).toBe(true);
		expect(schemaChosen(null, schemas)).toBe(false);
		expect(schemaChosen('ch', schemas)).toBe(false);
		expect(schemaChosen('us', [])).toBe(false);
	});
});
