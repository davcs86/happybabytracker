import { describe, expect, it } from 'vitest';
import { TRACKER_KINDS } from '../lib/constants';
import { params } from './index';

const accepts = (param: string) => {
	const result = params.tracker['~standard'].validate(param);
	if (result instanceof Promise) throw new Error('matchers are synchronous');
	return !result.issues;
};

describe('tracker param matcher', () => {
	it('accepts exactly the seven tracker route names', () => {
		expect([...TRACKER_KINDS].sort()).toEqual([
			'bottles',
			'diapers',
			'meals',
			'nursing',
			'pumping',
			'sleep',
			'symptoms'
		]);
		for (const kind of TRACKER_KINDS) expect(accepts(kind)).toBe(true);
	});

	it.each([
		'',
		'bottle',
		'diaper',
		'meal',
		'symptom',
		'sickness',
		'medications',
		'growth',
		'Bottles',
		'sleep/',
		'..',
		'constructor'
	])('refuses %j', (param) => {
		expect(accepts(param)).toBe(false);
	});
});
