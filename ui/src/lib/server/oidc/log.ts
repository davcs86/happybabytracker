export type SsoCode =
	| 'discovery'
	| 'idp_error'
	| 'state'
	| 'exchange'
	| 'id_token'
	| 'auth_time'
	| 'not_linked'
	| 'mode_refused'
	| 'already_linked'
	| 'reauth_required'
	| 'binding'
	| 'not_found'
	| 'oidc_not_configured';

// One fixed shape: an enumerated code, never a URL, claim, token or value.
export function logSsoRefusal(code: SsoCode): void {
	console.warn(JSON.stringify({ level: 'warn', msg: 'sso refused', code }));
}
