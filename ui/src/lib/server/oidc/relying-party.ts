import { AsyncLocalStorage } from 'node:async_hooks';
import * as client from 'openid-client';
import type { Env, OidcEnv } from '../env';
import {
	OIDC_AUTH_TIME_SKEW_S,
	OIDC_CALLBACK_PATH,
	OIDC_LABEL_MAX_LEN,
	OIDC_TIMEOUT_MS
} from '../constants';
import type { SsoCode } from './log';
import type { OidcPurpose, OidcStart } from './types';

export type RelyingParty = {
	start(
		purpose: OidcPurpose
	): Promise<
		{ ok: true; url: string; state: string; nonce: string; verifier: string } | { ok: false }
	>;
	finish(
		callback: URL,
		txn: OidcStart
	): Promise<
		{ ok: true; iss: string; sub: string; label: string | null } | { ok: false; code: SsoCode }
	>;
};

type Deps = { fetch?: typeof fetch; jwksCache?: { jwks: { keys: object[] }; uat: number } };

// Cc and Cf are removed before the cut, so the label renders as plain text of bounded length.
function cleanLabel(value: unknown): string | null {
	if (typeof value !== 'string') return null;
	const cut = [...value.replace(/[\p{Cc}\p{Cf}]/gu, '').trim()].slice(0, OIDC_LABEL_MAX_LEN);
	return cut.join('').trim() || null;
}

export function createRelyingParty(oidc: OidcEnv, origin: string, deps: Deps = {}): RelyingParty {
	const doFetch = deps.fetch ?? fetch;
	const redirectUri = new URL(OIDC_CALLBACK_PATH, origin).href;
	// Set once the token endpoint answered 2xx inside a finish: later failures are the ID token's.
	const exchanged = new AsyncLocalStorage<{ done: boolean }>();
	let tokenEndpoint: string | undefined;
	let loading: Promise<client.Configuration> | undefined;

	const customFetch: client.CustomFetch = async (url, init) => {
		const res = await doFetch(url, init as RequestInit);
		const flag = exchanged.getStore();
		if (flag && res.ok && url === tokenEndpoint) flag.done = true;
		return res;
	};

	async function discover(): Promise<client.Configuration> {
		const config = await client.discovery(
			new URL(oidc.issuer),
			oidc.clientId,
			{ id_token_signed_response_alg: 'RS256' },
			client.ClientSecretBasic(oidc.clientSecret),
			{ [client.customFetch]: customFetch, timeout: OIDC_TIMEOUT_MS / 1000 }
		);
		const jwksUri = config.serverMetadata().jwks_uri;
		if (!jwksUri || !URL.canParse(jwksUri)) throw new Error('jwks_uri refused');
		if (new URL(jwksUri).origin !== new URL(oidc.issuer).origin) {
			throw new Error('jwks_uri refused');
		}
		client.enableNonRepudiationChecks(config);
		if (deps.jwksCache) {
			client.setJwksCache(config, deps.jwksCache as Parameters<typeof client.setJwksCache>[1]);
		}
		// The canonical form, as customFetch receives it.
		const advertised = config.serverMetadata().token_endpoint;
		tokenEndpoint = advertised ? new URL(advertised).href : undefined;
		return config;
	}

	// Cached once it resolves; a rejection is dropped so the next attempt retries.
	function configuration(): Promise<client.Configuration> {
		loading ??= discover().catch((e: unknown) => {
			loading = undefined;
			throw e;
		});
		return loading;
	}

	async function label(
		config: client.Configuration,
		claims: client.IDToken,
		accessToken: string
	): Promise<string | null> {
		if (typeof claims.preferred_username === 'string') return cleanLabel(claims.preferred_username);
		try {
			const info = await client.fetchUserInfo(config, accessToken, claims.sub);
			return cleanLabel(info.preferred_username);
		} catch {
			return null;
		}
	}

	return {
		async start(purpose) {
			let config: client.Configuration;
			try {
				config = await configuration();
			} catch {
				return { ok: false };
			}
			const state = client.randomState();
			const nonce = client.randomNonce();
			const verifier = client.randomPKCECodeVerifier();
			const params: Record<string, string> = {
				redirect_uri: redirectUri,
				scope: oidc.scopes,
				state,
				nonce,
				code_challenge: await client.calculatePKCECodeChallenge(verifier),
				code_challenge_method: 'S256',
				response_mode: 'query'
			};
			if (purpose !== 'login') {
				params.prompt = 'login';
				params.max_age = '1';
			}
			let url: string;
			try {
				url = client.buildAuthorizationUrl(config, params).href;
			} catch {
				return { ok: false };
			}
			return { ok: true, url, state, nonce, verifier };
		},

		async finish(callback, txn) {
			if (callback.searchParams.has('error')) return { ok: false, code: 'idp_error' };
			if (callback.searchParams.get('state') !== txn.state) return { ok: false, code: 'state' };
			let config: client.Configuration;
			try {
				config = await configuration();
			} catch {
				return { ok: false, code: 'discovery' };
			}
			const flag = { done: false };
			let tokens: Awaited<ReturnType<typeof client.authorizationCodeGrant>>;
			try {
				tokens = await exchanged.run(flag, () =>
					client.authorizationCodeGrant(config, new URL(redirectUri + callback.search), {
						pkceCodeVerifier: txn.verifier,
						expectedState: txn.state,
						expectedNonce: txn.nonce,
						idTokenExpected: true
					})
				);
			} catch {
				return { ok: false, code: flag.done ? 'id_token' : 'exchange' };
			}
			const claims = tokens.claims();
			if (!claims) return { ok: false, code: 'id_token' };
			if (
				txn.purpose !== 'login' &&
				(typeof claims.auth_time !== 'number' ||
					claims.auth_time < txn.startedAt - OIDC_AUTH_TIME_SKEW_S)
			) {
				return { ok: false, code: 'auth_time' };
			}
			return {
				ok: true,
				iss: claims.iss,
				sub: claims.sub,
				label: await label(config, claims, tokens.access_token)
			};
		}
	};
}

let instance: RelyingParty | undefined;

export function relyingParty(env: Env): RelyingParty | null {
	if (!env.oidc) return null;
	instance ??= createRelyingParty(env.oidc, env.origin);
	return instance;
}
