import { describe, expect, it } from 'vitest';
import { MODULES, TRACKER_ROUTE } from './constants';
import { PAGE_ROUTE, quickLog } from './quick-log';
import type { AgeBand, Home, ModuleId, ModuleState } from './types';

const states = (on: readonly ModuleId[] = MODULES): ModuleState[] =>
	MODULES.map((module) => ({
		module,
		enabled: on.includes(module),
		default: true,
		overridden: false
	}));
const none: Home['running'] = { sleep: null, nursing: null };
const labels = (band: AgeBand, on?: readonly ModuleId[], running = none) =>
	quickLog(band, states(on), running).map((e) => e.label);

describe('quickLog', () => {
	it('0-11 months: bottle, nursing, diaper, sleep, then the rest in module order', () => {
		expect(labels('0-11')).toEqual([
			'Bottle',
			'Nursing',
			'Diaper',
			'Sleep',
			'Pumping',
			'Feeding',
			'Growth',
			'Vaccination',
			'Sickness',
			'Medication'
		]);
	});

	it('12-23 months: feeding, bottle, sleep, diaper, then the rest', () => {
		expect(labels('12-23')).toEqual([
			'Feeding',
			'Bottle',
			'Sleep',
			'Diaper',
			'Nursing',
			'Pumping',
			'Growth',
			'Vaccination',
			'Sickness',
			'Medication'
		]);
	});

	it('24+ months: sickness, medication, growth, vaccination, then the rest in module order', () => {
		expect(labels('24+')).toEqual([
			'Sickness',
			'Medication',
			'Growth',
			'Vaccination',
			'Bottle',
			'Nursing',
			'Pumping',
			'Diaper',
			'Sleep',
			'Feeding'
		]);
		expect(labels('24+', ['vaccination', 'growth', 'sleep', 'bottles'])).toEqual([
			'Growth',
			'Vaccination',
			'Bottle',
			'Sleep'
		]);
		expect(labels('24+', ['medication', 'sickness'])).toEqual(['Sickness', 'Medication']);
	});

	it('leaves out a disabled module and keeps the order of the rest', () => {
		expect(labels('0-11', ['sleep', 'bottles', 'feeding'])).toEqual(['Bottle', 'Sleep', 'Feeding']);
	});

	it('maps each entry to its tracker route', () => {
		const entries = quickLog('12-23', states(), none);
		expect(entries.map((e) => [e.module, e.kind])).toEqual([
			['feeding', 'meals'],
			['bottles', 'bottles'],
			['sleep', 'sleep'],
			['diapers', 'diapers'],
			['nursing', 'nursing'],
			['pumping', 'pumping'],
			['growth', null],
			['vaccination', null],
			['sickness', 'symptoms'],
			['medication', null]
		]);
	});

	it('growth opens its page, never a tracker route', () => {
		expect(TRACKER_ROUTE.growth).toBeUndefined();
		expect(PAGE_ROUTE.growth?.('k-1')).toBe('/kids/k-1/growth/new');
	});

	it('vaccination opens the Vaccines page, never a tracker route', () => {
		expect(TRACKER_ROUTE.vaccination).toBeUndefined();
		expect(PAGE_ROUTE.vaccination?.('k-1')).toBe('/kids/k-1/vaccines');
	});

	it('sickness opens the symptoms tracker', () => {
		expect(TRACKER_ROUTE.sickness).toBe('symptoms');
		expect(PAGE_ROUTE.sickness).toBeUndefined();
	});

	it('medication opens the Medication page, never a tracker route', () => {
		expect(TRACKER_ROUTE.medication).toBeUndefined();
		expect(PAGE_ROUTE.medication?.('k-1')).toBe('/kids/k-1/medication');
	});

	it('never lists a module without a tracker or a page in this build', () => {
		const kinds = new Set<string>(quickLog('0-11', states(), none).map((e) => e.module));
		expect(kinds.has('milestones')).toBe(false);
	});

	it('a running timer replaces its own start button only', () => {
		const sleep = { id: 's', startedAt: '2026-11-22T10:00:00Z', endedAt: null, note: null };
		expect(labels('0-11', undefined, { sleep, nursing: null })).toEqual([
			'Bottle',
			'Nursing',
			'Diaper',
			'Pumping',
			'Feeding',
			'Growth',
			'Vaccination',
			'Sickness',
			'Medication'
		]);
	});
});
