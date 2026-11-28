import type { RequestEvent } from '@sveltejs/kit';
import { OIDC_TXN_MAX_AGE_S } from '../constants';
import { INVITE_VALIDITY } from '../../constants';
import { setOidcCookie } from '../cookie';
import { deviceLabel } from '../device-label';
import { getEnv, type OidcEnv } from '../env';
import type { AuthMode, SessionKind } from '../../types';
import { logSsoRefusal } from './log';
import { relyingParty } from './relying-party';
import { seal } from './seal';
import type { OidcPurpose, OidcStart } from './types';

export const SSO_UNAVAILABLE = 'Single sign-on is unavailable';
export const SSO_LABEL_FALLBACK = 'the single sign-on account you just signed in with';
export const INVITE_NO_LONGER_VALID = `This link is no longer valid. Ask the person who sent it for a new one. This kind of link lasts ${INVITE_VALIDITY}.`;

let warnedNotConfigured = false;

// Logged once per process: a mode that offers SSO without relying-party settings.
function notConfigured(): void {
	if (warnedNotConfigured) return;
	warnedNotConfigured = true;
	logSsoRefusal('oidc_not_configured');
}

export function ssoOffered(mode: AuthMode, oidc: OidcEnv | null): boolean {
	if (mode === 'local') return false;
	if (!oidc) notConfigured();
	return oidc !== null;
}

export function ssoBack(
	purpose: OidcPurpose | null,
	kind: SessionKind | null,
	binding: string | null
): { href: string; label: string } {
	if (purpose === 'invite' && binding) {
		return { href: `/invite/${encodeURIComponent(binding)}`, label: 'Go to your invite link' };
	}
	if (purpose === 'link') {
		return kind === 'admin'
			? { href: '/admin/account', label: 'Go to Account' }
			: { href: '/account', label: 'Go to My profile' };
	}
	return { href: '/login', label: 'Go to sign in' };
}

export async function startSso(
	event: RequestEvent,
	purpose: OidcPurpose,
	extra: { next?: string | null; binding?: string; username?: string; kind?: SessionKind }
): Promise<{ ok: true; location: string } | { ok: false; message: string }> {
	const env = getEnv();
	const rp = relyingParty(env);
	if (!env.oidc || !rp) {
		notConfigured();
		return { ok: false, message: SSO_UNAVAILABLE };
	}
	const s = await rp.start(purpose);
	if (!s.ok) {
		logSsoRefusal('discovery');
		return { ok: false, message: SSO_UNAVAILABLE };
	}
	const now = Math.floor(Date.now() / 1000);
	const txn: OidcStart = {
		stage: 'start',
		purpose,
		state: s.state,
		nonce: s.nonce,
		verifier: s.verifier,
		startedAt: now,
		next: extra.next ?? null,
		binding: extra.binding ?? null,
		username: extra.username ?? null,
		kind: extra.kind ?? null,
		deviceLabel:
			purpose === 'link' ? null : deviceLabel(event.request.headers.get('user-agent') ?? ''),
		exp: now + OIDC_TXN_MAX_AGE_S
	};
	setOidcCookie(event.cookies, seal(env.oidc.sealKey, txn), OIDC_TXN_MAX_AGE_S);
	return { ok: true, location: s.url };
}
