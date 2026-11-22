import { describe, expect, it } from 'vitest';
import { entryApiFailure, entryFieldFailure, ENTRY_FIELD_MESSAGES } from './entry-failure';
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
			'right_seconds'
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
