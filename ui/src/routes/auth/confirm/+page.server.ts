import { redirect } from '@sveltejs/kit';
import { createApi } from '#lib/server/api.js';
import { clearOidcCookie, setSessionCookie } from '#lib/server/cookie.js';
import { OIDC_TXN_COOKIE, PARENT_HOME, SESSION_COOKIE } from '#lib/server/constants.js';
import { getEnv } from '#lib/server/env.js';
import { bindingMatches } from '#lib/server/oidc/link.js';
import { logSsoRefusal } from '#lib/server/oidc/log.js';
import { open } from '#lib/server/oidc/seal.js';
import { INVITE_NO_LONGER_VALID, ssoBack } from '#lib/server/oidc/start.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = () => ({});

const ALREADY_LINKED = 'That single sign-on account is already linked to an account';
const SSO_OFF = 'Single sign-on is turned off';

const failed = (back: { href: string; label: string }, message: string | null = null) => ({
	failed: true as const,
	back,
	message
});

export const actions: Actions = {
	// Always clears hbt_oidc, so a second post meets no pending link.
	default: async ({ request, cookies, locals, getClientAddress }) => {
		const form = await request.formData();
		const choice = form.get('choice');
		const env = getEnv();
		const txn = env.oidc
			? open(env.oidc.sealKey, cookies.get(OIDC_TXN_COOKIE), Math.floor(Date.now() / 1000))
			: null;
		clearOidcCookie(cookies);
		const toSignIn = ssoBack('login', null, null);
		if (!txn || txn.stage !== 'pending' || (txn.purpose === 'link' && !txn.kind)) {
			logSsoRefusal('state');
			return failed(toSignIn);
		}
		const back = ssoBack(txn.purpose, txn.kind, txn.binding);
		if (choice === 'cancel') redirect(303, back.href);
		if (choice !== 'confirm') {
			logSsoRefusal('state');
			return failed(back);
		}

		if (txn.purpose === 'link') {
			// The live session must be the one that started the link.
			const session = locals.session;
			if (!session || !bindingMatches(cookies.get(SESSION_COOKIE), txn.binding)) {
				logSsoRefusal('binding');
				return failed(back);
			}
			const r =
				session.kind === 'admin'
					? await locals.api.adminLinkIdentity(txn.iss, txn.sub)
					: await locals.api.linkIdentity(txn.iss, txn.sub);
			if (r.ok) redirect(303, back.href);
			if (r.code === 'reauth_required') {
				logSsoRefusal('reauth_required');
				return failed(back, 'Confirm your password again, then link');
			}
			if (r.code === 'already_linked') {
				logSsoRefusal('already_linked');
				return failed(back, ALREADY_LINKED);
			}
			if (r.code === 'mode_refused') {
				logSsoRefusal('mode_refused');
				return failed(back, SSO_OFF);
			}
			return failed(back);
		}

		const api = createApi(env, fetch, undefined, getClientAddress());
		const r = await api.redeemTokenOidc(
			txn.binding,
			txn.iss,
			txn.sub,
			txn.deviceLabel ?? undefined
		);
		if (r.ok) {
			setSessionCookie(cookies, r.data.token, r.data.kind);
			redirect(303, PARENT_HOME);
		}
		if (r.status === 404) {
			logSsoRefusal('not_found');
			return failed(toSignIn, INVITE_NO_LONGER_VALID);
		}
		if (r.code === 'already_linked') {
			logSsoRefusal('already_linked');
			return failed(back, ALREADY_LINKED);
		}
		if (r.code === 'mode_refused') {
			logSsoRefusal('mode_refused');
			return failed(back, SSO_OFF);
		}
		return failed(back);
	}
};
