import { describe, expect, it } from 'vitest';
import { MODULES } from './constants';
import { quickLog } from './quick-log';
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
		expect(labels('0-11')).toEqual(['Bottle', 'Nursing', 'Diaper', 'Sleep', 'Pumping', 'Feeding']);
	});

	it('12-23 months: feeding, bottle, sleep, diaper, then the rest', () => {
		expect(labels('12-23')).toEqual(['Feeding', 'Bottle', 'Sleep', 'Diaper', 'Nursing', 'Pumping']);
	});

	it('24+ months: the four later modules have no tracker yet, so only re-enabled ones show', () => {
		expect(labels('24+', ['sickness', 'medication', 'growth', 'vaccination'])).toEqual([]);
		expect(labels('24+', ['growth', 'sleep', 'bottles'])).toEqual(['Bottle', 'Sleep']);
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
			['pumping', 'pumping']
		]);
	});

	it('never lists a module without a tracker in this build', () => {
		const kinds = new Set<string>(quickLog('0-11', states(), none).map((e) => e.module));
		for (const m of ['milestones', 'growth', 'vaccination', 'sickness', 'medication']) {
			expect(kinds.has(m)).toBe(false);
		}
	});

	it('a running timer replaces its own start button only', () => {
		const sleep = { id: 's', startedAt: '2026-11-22T10:00:00Z', endedAt: null, note: null };
		expect(labels('0-11', undefined, { sleep, nursing: null })).toEqual([
			'Bottle',
			'Nursing',
			'Diaper',
			'Pumping',
			'Feeding'
		]);
	});
});
