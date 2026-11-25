import { describe, expect, it } from 'vitest';
import { sectionOrHome, switchTarget } from './switch-target';

describe('switchTarget', () => {
	it.each([
		['/kids/k1/timeline', 'timeline'],
		['/kids/k1/growth/new', 'growth'],
		['/kids/k1/vaccines/a1', 'vaccines'],
		['/kids/k1/medication/doses/d1', 'medication'],
		['/kids/k1/milestones', 'milestones'],
		['/kids/k1/export/pdf', 'export'],
		['/', ''],
		['/kids/k1', ''],
		['/kids/k1/settings', ''],
		['/kids/k1/log/bottles', ''],
		['/account', '']
	])('%s returns to %j', (path, expected) => {
		expect(switchTarget(path)).toBe(expected);
	});
});

describe('sectionOrHome', () => {
	it.each([
		['timeline', 'timeline'],
		['export', 'export'],
		['settings', ''],
		['../account', ''],
		['', ''],
		[null, '']
	])('%j returns %j', (raw, expected) => {
		expect(sectionOrHome(raw)).toBe(expected);
	});
});
