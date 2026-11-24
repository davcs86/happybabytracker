import { describe, expect, it, vi } from 'vitest';
import type { ApiResult } from './api';
import { carryOver } from './push-renew';
import type { PushDevice } from '../types';

const ok: ApiResult<null> = { ok: true, data: null };
const refused: ApiResult<null> = { ok: false, status: 503, reason: 'unavailable' };
const device = (id: string, kids: string[]): PushDevice => ({
	id,
	label: 'Phone',
	createdAt: '2026-11-23T10:00:00Z',
	kids
});

function calls(
	optIn = vi.fn(async (): Promise<ApiResult<null>> => ok),
	removeDevice = vi.fn(async (): Promise<ApiResult<null>> => ok)
) {
	const order: string[] = [];
	return {
		order,
		calls: {
			optIn: vi.fn(async (kid: string, id: string) => {
				order.push(`in ${kid} ${id}`);
				return optIn();
			}),
			removeDevice: vi.fn(async (id: string) => {
				order.push(`remove ${id}`);
				return removeDevice();
			})
		}
	};
}

describe('carryOver', () => {
	it('opts the new device into every kid it lacks, then removes the old one', async () => {
		const { order, calls: c } = calls();
		expect(await carryOver(c, device('old', ['k-a', 'k-b']), device('new', ['k-b']))).toBeNull();
		expect(order).toEqual(['in k-a new', 'remove old']);
	});

	it('removes the old device when it had no kids', async () => {
		const { order, calls: c } = calls();
		expect(await carryOver(c, device('old', []), device('new', []))).toBeNull();
		expect(order).toEqual(['remove old']);
	});

	it('leaves the old device in place when an opt-in is refused', async () => {
		const { order, calls: c } = calls(
			vi
				.fn(async (): Promise<ApiResult<null>> => ok)
				.mockResolvedValueOnce(ok)
				.mockResolvedValueOnce(refused)
		);
		expect(await carryOver(c, device('old', ['k-a', 'k-b', 'k-c']), device('new', []))).toEqual(
			refused
		);
		expect(order).toEqual(['in k-a new', 'in k-b new']);
		expect(c.removeDevice).not.toHaveBeenCalled();
	});

	it('answers the removal refusal, and an old device already gone counts as removed', async () => {
		const { calls: failing } = calls(
			undefined,
			vi.fn(async (): Promise<ApiResult<null>> => refused)
		);
		expect(await carryOver(failing, device('old', ['k-a']), device('new', []))).toEqual(refused);
		const gone: ApiResult<null> = { ok: false, status: 404, reason: 'not_found' };
		const { calls: c } = calls(
			undefined,
			vi.fn(async (): Promise<ApiResult<null>> => gone)
		);
		expect(await carryOver(c, device('old', ['k-a']), device('new', []))).toBeNull();
	});
});
