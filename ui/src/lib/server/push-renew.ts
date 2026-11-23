import type { ApiFailure } from './outcome';
import type { ApiResult } from './api';
import type { PushDevice } from '../types';

type RenewCalls = {
	optIn: (kidId: string, deviceId: string) => Promise<ApiResult<null>>;
	removeDevice: (deviceId: string) => Promise<ApiResult<null>>;
};

// The old device goes only after every one of its kids is on the new one; a refusal leaves it as it was.
export async function carryOver(
	calls: RenewCalls,
	from: PushDevice,
	to: PushDevice
): Promise<ApiFailure | null> {
	for (const kid of from.kids.filter((k) => !to.kids.includes(k))) {
		const r = await calls.optIn(kid, to.id);
		if (!r.ok) return r;
	}
	const r = await calls.removeDevice(from.id);
	return r.ok || r.reason === 'not_found' ? null : r;
}
