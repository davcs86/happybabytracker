import { describe, expect, it } from 'vitest';
import {
	entryApiFailure,
	entryFieldFailure,
	ENTRY_FIELD_MESSAGES,
	FINISHED_SIDE_MESSAGE,
	sourcesTotalMessage,
	staleTimerFailure,
	TIMER_ENDED_MESSAGE,
	totalRangeMessage,
	volumeRefusedMessage
} from './entry-failure';
import { MEAL_TEXT_MAX_LEN, SYMPTOM_OTHER_MAX_LEN } from '../constants';
import type { ApiFailure } from './outcome';

const values = { at: '2026-11-20T10:30', note: 'kept' };
const extra = { form: 'save' as const };
const refused = (over: Partial<ApiFailure>): ApiFailure =>
	({ ok: false, status: 422, reason: 'invalid', ...over }) as ApiFailure;

describe('entryFieldFailure', () => {
	it('answers 422 with the values, the field and its message', () => {
		const r = entryFieldFailure(values, 'contents', extra);
		expect(r.status).toBe(422);
		expect(r.data).toEqual({
			form: 'save',
			values,
			field: 'contents',
			message: ENTRY_FIELD_MESSAGES.contents
		});
	});
});

describe('entryApiFailure', () => {
	it('puts a refused field on that field', () => {
		const r = entryApiFailure('bottles', values, refused({ field: 'at' }), extra);
		expect(r.status).toBe(422);
		expect(r.data).toMatchObject({ field: 'at', message: ENTRY_FIELD_MESSAGES.at, values });
	});

	it('names every field the forms can be refused on', () => {
		for (const field of [
			'at',
			'started_at',
			'ended_at',
			'contents',
			'volume',
			'kind',
			'what',
			'reaction',
			'note',
			'duration_seconds',
			'side',
			'left_seconds',
			'right_seconds',
			'symptom',
			'severity',
			'other_text',
			'temperature',
			'method'
		]) {
			expect(ENTRY_FIELD_MESSAGES[field]).toBeTruthy();
		}
	});

	it('shows a second timer as a message on the page, not on a field', () => {
		const conflict = {
			ok: false,
			status: 409,
			reason: 'conflict',
			code: 'already_running'
		} as const;
		const nursing = entryApiFailure('nursing', values, conflict, extra);
		expect(nursing.status).toBe(409);
		expect(nursing.data).toMatchObject({
			field: null,
			running: true,
			message: 'A nursing timer is already running'
		});
		expect(entryApiFailure('sleep', values, conflict, extra).data).toMatchObject({
			message: 'A sleep timer is already running'
		});
	});

	it('does not invent a message for a field it does not know, or a conflict it does not expect', () => {
		const unknown = entryApiFailure('bottles', values, refused({ field: 'volume.unit' }), extra);
		expect(unknown.status).toBe(500);
		expect(unknown.data).toMatchObject({ field: null, values });
		const other = entryApiFailure(
			'bottles',
			values,
			{ ok: false, status: 409, reason: 'conflict', code: 'already_running' },
			extra
		);
		expect(other.status).toBe(500);
		const proto = entryApiFailure('bottles', values, refused({ field: 'toString' }), extra);
		expect(proto.status).toBe(500);
	});

	it('answers 503 when the API cannot be reached', () => {
		const r = entryApiFailure(
			'bottles',
			values,
			{ ok: false, status: 503, reason: 'unavailable' },
			extra
		);
		expect(r.status).toBe(503);
		expect(r.data).toMatchObject({ values, field: null });
	});
});

describe('staleTimerFailure', () => {
	it('answers 409 with the fresh values and a message that belongs to no field', () => {
		const r = staleTimerFailure(values, extra);
		expect(r.status).toBe(409);
		expect(r.data).toEqual({
			form: 'save',
			values,
			field: null,
			message: TIMER_ENDED_MESSAGE,
			stale: true
		});
	});
});

describe('final-wave messages', () => {
	it('a field failure may carry a message of its own', () => {
		expect(entryFieldFailure(values, 'side', extra, FINISHED_SIDE_MESSAGE).data).toMatchObject({
			field: 'side',
			message: FINISHED_SIDE_MESSAGE
		});
		expect(entryFieldFailure(values, 'side', extra, null).data).toMatchObject({
			message: ENTRY_FIELD_MESSAGES.side
		});
	});

	it('an API volume refusal states the range and the format, per tracker', () => {
		const bottle = entryApiFailure('bottles', values, refused({ field: 'volume' }), extra);
		expect(bottle.data).toMatchObject({
			field: 'volume',
			message: volumeRefusedMessage('bottles')
		});
		expect(volumeRefusedMessage('bottles')).toContain('1000 ml');
		expect(volumeRefusedMessage('pumping')).toContain('2000 ml');
		expect(volumeRefusedMessage('bottles')).toContain('two decimals');
	});

	it('an advanced bottle total refused by the API states the range of the total only', () => {
		const r = entryApiFailure(
			'bottles',
			{ ...values, advanced: '1' },
			refused({ field: 'volume' }),
			extra
		);
		expect(r.data).toMatchObject({ field: 'volume', message: sourcesTotalMessage() });
		expect(sourcesTotalMessage()).toBe('The total must be from 1 to 1000 ml');
		const simple = entryApiFailure(
			'bottles',
			{ ...values, advanced: '' },
			refused({ field: 'volume' }),
			extra
		);
		expect(simple.data).toMatchObject({ message: volumeRefusedMessage('bottles') });
	});

	it('the sources message is plain and carries no value', () => {
		expect(ENTRY_FIELD_MESSAGES.sources).toBe(
			'Enter a volume for each source you ticked, in whole millilitres or ounces with up to two decimals'
		);
		expect(ENTRY_FIELD_MESSAGES.sources).not.toMatch(/\d/);
	});

	it('an advanced pumping total refused by the API states the range of the total only', () => {
		const r = entryApiFailure(
			'pumping',
			{ ...values, advanced: '1' },
			refused({ field: 'volume' }),
			extra
		);
		expect(r.data).toMatchObject({ field: 'volume', message: totalRangeMessage('pumping') });
		expect(totalRangeMessage('pumping')).toBe('The total must be from 1 to 2000 ml');
		const simple = entryApiFailure('pumping', values, refused({ field: 'volume' }), extra);
		expect(simple.data).toMatchObject({ message: volumeRefusedMessage('pumping') });
	});

	it('a refused sides field has a plain message that carries no value', () => {
		expect(ENTRY_FIELD_MESSAGES.sides).toBe(
			'Enter a volume for at least one side, in whole millilitres or ounces with up to two decimals'
		);
		expect(ENTRY_FIELD_MESSAGES.sides).not.toMatch(/\d/);
		expect(
			entryApiFailure('pumping', values, refused({ field: 'sides' }), extra).data
		).toMatchObject({ field: 'sides', message: ENTRY_FIELD_MESSAGES.sides });
	});

	it('an API what refusal states both rules', () => {
		expect(ENTRY_FIELD_MESSAGES.what).toContain(`${MEAL_TEXT_MAX_LEN} characters`);
	});

	it('a refused temperature states the range in both units and the decimals, with no value', () => {
		expect(ENTRY_FIELD_MESSAGES.temperature).toBe(
			'Enter a temperature from 30 to 45 °C (86 to 113 °F), with up to two decimals'
		);
		const r = entryApiFailure('symptoms', values, refused({ field: 'temperature' }), extra);
		expect(r.status).toBe(422);
		expect(r.data).toMatchObject({
			field: 'temperature',
			message: ENTRY_FIELD_MESSAGES.temperature
		});
	});

	it('the other symptom text names its limit, the others name no value', () => {
		expect(ENTRY_FIELD_MESSAGES.other_text).toContain(`${SYMPTOM_OTHER_MAX_LEN} characters`);
		for (const field of ['symptom', 'severity', 'method']) {
			expect(ENTRY_FIELD_MESSAGES[field]).not.toMatch(/\d/);
		}
	});
});
