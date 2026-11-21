import { ADMIN_HOME, PARENT_HOME } from './constants';
import { safeNext } from './next';
import type { SessionKind } from '../types';

// The admin only ever lands inside /admin/; anything else would bounce through the hook.
export function landing(kind: SessionKind, next: string | null): string {
	const target = safeNext(next);
	if (kind === 'admin') return target?.startsWith('/admin/') ? target : ADMIN_HOME;
	return target ?? PARENT_HOME;
}
