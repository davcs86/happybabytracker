import { redirect } from '@sveltejs/kit';
import { createApi } from '#lib/server/api.js';
import { clearOidcCookie, setSessionCookie } from '#lib/server/cookie.js';
import { OIDC_TXN_COOKIE, PARENT_HOME } from '#lib/server/constants.js';
import { getEnv } from '#lib/server/env.js';
import { logSsoRefusal } from '#lib/server/oidc/log.js';
import { open } from '#lib/server/oidc/seal.js';
import { INVITE_NO_LONGER_VALID, ssoBack } from '#lib/server/oidc/start.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = () => ({});

const failed = (back: { href: string; label: string }, message: string | null = null) => ({
	failed: true as const,
	back,
	message
});

export const actions: Actions = {
	// Always clears hbt_oidc, so a second post meets no pending link.
	default: async ({ request, cookies, getClientAddress }) => {
		const form = await request.formData();
		const choice = form.get('choice');
		const env = getEnv();
		const txn = env.oidc
			? open(env.oidc.sealKey, cookies.get(OIDC_TXN_COOKIE), Math.floor(Date.now() / 1000))
			: null;
		clearOidcCookie(cookies);
		const toSignIn = ssoBack('login', null, null);
		if (!txn || txn.stage !== 'pending' || txn.purpose !== 'invite') {
			logSsoRefusal('state');
			return failed(toSignIn);
		}
		const back = ssoBack(txn.purpose, txn.kind, txn.binding);
		if (choice === 'cancel') redirect(303, back.href);
		if (choice !== 'confirm') {
			logSsoRefusal('state');
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
			return failed(back, 'That single sign-on account is already linked to an account');
		}
		if (r.code === 'mode_refused') {
			logSsoRefusal('mode_refused');
			return failed(back, 'Single sign-on is turned off');
		}
		return failed(back);
	}
};
