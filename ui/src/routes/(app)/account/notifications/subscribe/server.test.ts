import { describe, expect, it, vi } from 'vitest';
import { UNREACHABLE_MESSAGE, PUSH_DEVICE_COOKIE } from '#lib/server/constants.js';
import type { ApiResult } from '#lib/server/api.js';
import type { PushDevice } from '#lib/types.js';
import { POST } from './+server';

const device = (id: string, kids: string[], label = 'Night phone'): PushDevice => ({
	id,
	label,
	createdAt: '2026-11-23T10:00:00Z',
	kids
});
const ok = <T>(data: T): ApiResult<T> => ({ ok: true, data });

describe('POST subscribe, a renewed subscription', () => {
	it('keeps the old device and the cookie when a kid cannot be carried over', async () => {
		const old = device('d-old', ['k-a', 'k-b']);
		const push = {
			state: vi.fn(async () => ok({ enabled: true, publicKey: 'key' })),
			devices: vi.fn(async () => ok([old])),
			upsertDevice: vi.fn(async () => ok(device('d-new', [], old.label))),
			optIn: vi.fn(async (): Promise<ApiResult<null>> => ({
				ok: false,
				status: 503,
				reason: 'unavailable'
			})),
			removeDevice: vi.fn(async (): Promise<ApiResult<null>> => ok(null))
		};
		const cookies = { get: vi.fn(() => old.id), set: vi.fn(), delete: vi.fn() };
		const request = new Request('http://localhost/account/notifications/subscribe', {
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify({ endpoint: 'https://push.example/x', keys: { p256dh: 'p', auth: 'a' } })
		});
		const res = await POST({ request, locals: { api: { push } }, cookies } as never);
		expect(res.status).toBe(503);
		expect(await res.json()).toEqual({ message: UNREACHABLE_MESSAGE });
		expect(cookies.get).toHaveBeenCalledWith(PUSH_DEVICE_COOKIE);
		expect(cookies.set).not.toHaveBeenCalled();
		expect(push.optIn).toHaveBeenCalledTimes(1);
		expect(push.removeDevice).not.toHaveBeenCalled();
	});
});
