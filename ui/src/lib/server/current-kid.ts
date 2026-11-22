import type { Cookies } from '@sveltejs/kit';
import type { Kid } from '../types';
import { KID_COOKIE } from './constants';

// The remembered kid if the parent still has it, else the first.
export function currentKidId(kids: Pick<Kid, 'id'>[], cookies: Pick<Cookies, 'get'>) {
	const remembered = cookies.get(KID_COOKIE);
	return kids.find((k) => k.id === remembered)?.id ?? kids[0]?.id;
}
