import { formatAgo, formatInstant } from './format';
import type { AdminLink, AdminParent } from './types';

export function lastActiveLine(p: AdminParent, now: Date): string {
	if (p.lastActiveAt) return `Last active ${formatAgo(p.lastActiveAt, now)}`;
	return p.hasPassword ? 'No recent sign-in' : 'Hasn’t signed in yet';
}

// The parent's unused one-time link, or '' when there is none.
export function linkLine(link: AdminLink | null, now: Date): string {
	if (!link) return '';
	const kind = link.kind === 'reset' ? 'Reset' : 'Invite';
	const state =
		Date.parse(link.expiresAt) > now.getTime()
			? `Valid until ${formatInstant(link.expiresAt)}.`
			: 'Expired.';
	return `${kind} link issued ${formatAgo(link.issuedAt, now)}. ${state}`;
}
