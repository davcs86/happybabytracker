import { describe, expect, it } from 'vitest';
import { fromRunningPage, sendOnlyChanged, withoutTimerSide } from './entry-submit';
import type { EntryBody } from '../types';

const form = (fields: Record<string, string | string[]>) => {
	const data = new FormData();
	for (const [k, v] of Object.entries(fields)) {
		for (const one of Array.isArray(v) ? v : [v]) data.append(k, one);
	}
	return data;
};

describe('sendOnlyChanged', () => {
	const bottle = {
		at: '2026-11-20T10:30',
		at_original: '2026-11-20T10:30',
		volume: '4.25',
		volume_original: '4.25',
		volume_unit: 'floz',
		volume_unit_original: 'floz'
	};
	const input = (): EntryBody => ({
		at: '2026-11-20T09:30:00Z',
		volume: { unit: 'floz', amount: 425 },
		contents: ['formula'],
		note: 'n'
	});

	it('leaves out a time and a volume that were not touched', () => {
		expect(sendOnlyChanged('bottles', form(bottle), input())).toEqual({
			contents: ['formula'],
			note: 'n'
		});
	});

	it('keeps a changed amount, a changed unit and a changed time', () => {
		expect(sendOnlyChanged('bottles', form({ ...bottle, volume: '5' }), input())).toHaveProperty(
			'volume'
		);
		expect(
			sendOnlyChanged('bottles', form({ ...bottle, volume_unit: 'ml' }), input())
		).toHaveProperty('volume');
		const moved = sendOnlyChanged('bottles', form({ ...bottle, at: '2026-11-20T10:31' }), input());
		expect(moved).toHaveProperty('at');
		expect(moved).not.toHaveProperty('volume');
	});

	it('keeps everything when the originals are missing', () => {
		const r = sendOnlyChanged('bottles', form({ at: bottle.at, volume: '4.25' }), input());
		expect(r).toEqual(input());
	});

	it('does not change what it was given', () => {
		const given = input();
		sendOnlyChanged('bottles', form(bottle), given);
		expect(given).toEqual(input());
	});

	it('drops the end of a finished session when it was not touched', () => {
		const sleep = {
			started_at: '2026-11-20T10:00',
			started_at_original: '2026-11-20T10:00',
			ended_at: '2026-11-20T11:00',
			ended_at_original: '2026-11-20T11:00'
		};
		const body = { started_at: 'a', ended_at: 'b', note: null };
		expect(sendOnlyChanged('sleep', form(sleep), body)).toEqual({ note: null });
		expect(
			sendOnlyChanged('sleep', form({ ...sleep, ended_at: '2026-11-20T11:05' }), body)
		).toEqual({ ended_at: 'b', note: null });
	});

	it('drops a pumping duration whose minutes are the floor of the stored seconds', () => {
		const pumping = {
			started_at: '2026-11-20T10:00',
			started_at_original: '2026-11-20T10:00',
			duration_minutes: '20',
			duration_seconds_original: '1203',
			volume: '80',
			volume_original: '80',
			volume_unit: 'ml',
			volume_unit_original: 'ml'
		};
		const body = { duration_seconds: 1200, volume: { unit: 'ml', amount: 80 }, side: 'both' };
		expect(sendOnlyChanged('pumping', form(pumping), body)).toEqual({ side: 'both' });
		expect(
			sendOnlyChanged('pumping', form({ ...pumping, duration_minutes: '21' }), {
				...body,
				duration_seconds: 1260
			})
		).toEqual({ duration_seconds: 1260, side: 'both' });
		expect(
			sendOnlyChanged('pumping', form({ ...pumping, duration_seconds_original: 'x' }), body)
		).toHaveProperty('duration_seconds');
	});

	it('leaves the other kinds alone', () => {
		const body = { kind: 'wet', note: null };
		expect(sendOnlyChanged('diapers', form({ kind: 'wet' }), body)).toEqual(body);
		expect(sendOnlyChanged('meals', form({ what: 'rice' }), { what: 'rice' })).toEqual({
			what: 'rice'
		});
	});
});

describe('withoutTimerSide', () => {
	const fields = { started_at: '2026-11-20T10:00', side: 'left', left_minutes: '3' };

	it('drops the timer side when a nursing session is finished', () => {
		const r = withoutTimerSide('nursing', form({ ...fields, ended_at: '2026-11-20T10:10' }));
		expect(r.has('side')).toBe(false);
		expect(r.get('left_minutes')).toBe('3');
		expect(r.get('ended_at')).toBe('2026-11-20T10:10');
	});

	it('keeps the side of a timer, and every other tracker', () => {
		expect(withoutTimerSide('nursing', form({ ...fields, ended_at: '  ' })).get('side')).toBe(
			'left'
		);
		expect(withoutTimerSide('pumping', form({ ...fields, ended_at: 'x' })).get('side')).toBe(
			'left'
		);
	});
});

describe('fromRunningPage', () => {
	const running = { started_at: '2026-11-20T10:00', ended_at: '', note: '' };

	it('is a nursing form without either stored-seconds original', () => {
		expect(fromRunningPage('nursing', form(running))).toBe(true);
	});

	it('is not a finished nursing form, whatever the original holds', () => {
		const done = { ...running, left_seconds_original: '723', right_seconds_original: '61' };
		expect(fromRunningPage('nursing', form(done))).toBe(false);
		expect(fromRunningPage('nursing', form({ ...running, left_seconds_original: '' }))).toBe(false);
		expect(fromRunningPage('nursing', form({ ...running, right_seconds_original: 'x' }))).toBe(
			false
		);
	});

	it('is never true for another tracker', () => {
		expect(fromRunningPage('sleep', form(running))).toBe(false);
		expect(fromRunningPage('pumping', form(running))).toBe(false);
	});
});
