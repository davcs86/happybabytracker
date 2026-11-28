import { createApi } from '#lib/server/api.js';
import { clearOidcCookie, setOidcCookie, setSessionCookie } from '#lib/server/cookie.js';
import { OIDC_TXN_COOKIE } from '#lib/server/constants.js';
import { getEnv } from '#lib/server/env.js';
import { landing } from '#lib/server/landing.js';
import { logSsoRefusal, type SsoCode } from '#lib/server/oidc/log.js';
import { relyingParty } from '#lib/server/oidc/relying-party.js';
import { open, seal } from '#lib/server/oidc/seal.js';
import { INVITE_NO_LONGER_VALID, SSO_LABEL_FALLBACK, ssoBack } from '#lib/server/oidc/start.js';
import type { OidcCookie, OidcPending } from '#lib/server/oidc/types.js';
import type { PageServerLoad } from './$types';

type Outcome =
	| { outcome: 'continue'; to: string }
	| { outcome: 'confirm'; label: string; username: string }
	| { outcome: 'failed'; back: { href: string; label: string }; message: string | null };

// Answers 200 whatever happens, and reads only the sealed hbt_oidc: never the session cookie.
export const load: PageServerLoad = async ({
	cookies,
	url,
	getClientAddress
}): Promise<Outcome> => {
	const env = getEnv();
	const oidc = env.oidc;
	const nowS = Math.floor(Date.now() / 1000);
	const txn = oidc ? open(oidc.sealKey, cookies.get(OIDC_TXN_COOKIE), nowS) : null;
	clearOidcCookie(cookies);
	const failed = (code: SsoCode | null, from: OidcCookie | null): Outcome => {
		if (code) logSsoRefusal(code);
		return {
			outcome: 'failed',
			back: ssoBack(from?.purpose ?? null, from?.kind ?? null, from?.binding ?? null),
			message: null
		};
	};
	const rp = relyingParty(env);
	if (!oidc || !txn || txn.stage !== 'start' || !rp) return failed('state', txn);
	const r = await rp.finish(url, txn);
	if (!r.ok) return failed(r.code, txn);
	const api = createApi(env, fetch, undefined, getClientAddress());

	if (txn.purpose === 'invite') {
		if (!txn.binding) return failed('binding', txn);
		const peek = await api.peekToken(txn.binding);
		if (!peek.ok || peek.data.kind !== 'invite') {
			if (!peek.ok && peek.reason !== 'not_found') return failed(null, txn);
			logSsoRefusal('not_found');
			return {
				outcome: 'failed',
				back: ssoBack('login', null, null),
				message: INVITE_NO_LONGER_VALID
			};
		}
		const pending: OidcPending = {
			stage: 'pending',
			purpose: 'invite',
			binding: txn.binding,
			username: peek.data.username,
			kind: null,
			iss: r.iss,
			sub: r.sub,
			label: r.label,
			deviceLabel: txn.deviceLabel,
			exp: txn.exp
		};
		setOidcCookie(cookies, seal(oidc.sealKey, pending), Math.max(1, txn.exp - nowS));
		return {
			outcome: 'confirm',
			label: r.label ?? SSO_LABEL_FALLBACK,
			username: peek.data.username
		};
	}
	if (txn.purpose === 'link') {
		// Every field from the start cookie: the seal check does not tie them to the purpose.
		if (!txn.binding || !txn.username || !txn.kind) return failed('binding', txn);
		const pending: OidcPending = {
			stage: 'pending',
			purpose: 'link',
			binding: txn.binding,
			username: txn.username,
			kind: txn.kind,
			iss: r.iss,
			sub: r.sub,
			label: r.label,
			deviceLabel: null,
			exp: txn.exp
		};
		setOidcCookie(cookies, seal(oidc.sealKey, pending), Math.max(1, txn.exp - nowS));
		return {
			outcome: 'confirm',
			label: r.label ?? SSO_LABEL_FALLBACK,
			username: txn.username
		};
	}

	const s = await api.oidcSession(r.iss, r.sub, txn.deviceLabel ?? undefined);
	if (!s.ok) {
		const code = s.code === 'not_linked' || s.code === 'mode_refused' ? s.code : null;
		return failed(code, txn);
	}
	setSessionCookie(cookies, s.data.token, s.data.kind);
	return { outcome: 'continue', to: landing(s.data.kind, txn.next) };
};
