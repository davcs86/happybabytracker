import { createApi } from '#lib/server/api.js';
import { clearOidcCookie, setSessionCookie } from '#lib/server/cookie.js';
import { OIDC_TXN_COOKIE } from '#lib/server/constants.js';
import { getEnv } from '#lib/server/env.js';
import { landing } from '#lib/server/landing.js';
import { logSsoRefusal, type SsoCode } from '#lib/server/oidc/log.js';
import { relyingParty } from '#lib/server/oidc/relying-party.js';
import { open } from '#lib/server/oidc/seal.js';
import { ssoBack } from '#lib/server/oidc/start.js';
import type { OidcCookie } from '#lib/server/oidc/types.js';
import type { PageServerLoad } from './$types';

type Outcome =
	| { outcome: 'continue'; to: string }
	| { outcome: 'confirm'; label: string; username: string }
	| { outcome: 'failed'; back: { href: string; label: string } };

// Answers 200 whatever happens, and reads only the sealed hbt_oidc: never the session cookie.
export const load: PageServerLoad = async ({
	cookies,
	url,
	getClientAddress
}): Promise<Outcome> => {
	const env = getEnv();
	const txn = env.oidc
		? open(env.oidc.sealKey, cookies.get(OIDC_TXN_COOKIE), Math.floor(Date.now() / 1000))
		: null;
	clearOidcCookie(cookies);
	const failed = (code: SsoCode | null, from: OidcCookie | null): Outcome => {
		if (code) logSsoRefusal(code);
		return {
			outcome: 'failed',
			back: ssoBack(from?.purpose ?? null, from?.kind ?? null, from?.binding ?? null)
		};
	};
	const rp = relyingParty(env);
	if (!txn || txn.stage !== 'start' || !rp) return failed('state', txn);
	const r = await rp.finish(url, txn);
	if (!r.ok) return failed(r.code, txn);
	if (txn.purpose !== 'login') return failed(null, txn);

	const api = createApi(env, fetch, undefined, getClientAddress());
	const s = await api.oidcSession(r.iss, r.sub, txn.deviceLabel ?? undefined);
	if (!s.ok) {
		const code = s.code === 'not_linked' || s.code === 'mode_refused' ? s.code : null;
		return failed(code, txn);
	}
	setSessionCookie(cookies, s.data.token, s.data.kind);
	return { outcome: 'continue', to: landing(s.data.kind, txn.next) };
};
