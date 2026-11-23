import { describe, expect, it } from 'vitest';
import {
	antigenName,
	dueLabel,
	groupTitle,
	schemaChosen,
	statusLabel,
	windowLabel
} from './vaccines';
import type { VaccineSchemaInfo } from './types';

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

	it('labels a home due entry by its visit and its status label, without the line count', () => {
		const entry = { kind: 'vaccine' } as const;
		expect(dueLabel({ ...entry, from: { n: 4, unit: 'm' }, status: 'overdue', count: 7 })).toBe(
			'4-month vaccines · Overdue'
		);
		expect(dueLabel({ ...entry, from: { n: 0, unit: 'd' }, status: 'due', count: 1 })).toBe(
			'Birth · Due'
		);
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
