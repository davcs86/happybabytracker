import * as privateEnv from '$app/env/private';
import {
	MAX_PORT,
	MIN_INTERNAL_API_TOKEN,
	MIN_SESSION_SEAL_KEY,
	OIDC_DEFAULT_SCOPES
} from './constants';

export type OidcEnv = {
	issuer: string;
	clientId: string;
	clientSecret: string;
	scopes: string;
	sealKey: string;
};

export type Env = {
	internalApiToken: string;
	apiBaseUrl: string;
	origin: string;
	port: number;
	oidc: OidcEnv | null;
};

const RELYING_PARTY = ['OIDC_ISSUER', 'OIDC_CLIENT_ID', 'OIDC_CLIENT_SECRET'] as const;

function isHttpUrl(value: string): boolean {
	return URL.canParse(value) && /^https?:$/.test(new URL(value).protocol);
}

function isHttpsUrl(value: string): boolean {
	return URL.canParse(value) && new URL(value).protocol === 'https:';
}

// Present means !== undefined: an empty value counts as set.
function loadOidc(source: Record<string, string | undefined>, bad: string[]): OidcEnv | null {
	const refuse = (key: string) => {
		if (!bad.includes(key)) bad.push(key);
	};
	const present = (key: string) => source[key] !== undefined;
	const rp = RELYING_PARTY.filter(present);
	const sealKey = source.SESSION_SEAL_KEY;
	const scopes = source.OIDC_SCOPES;

	if (rp.length > 0 && rp.length < RELYING_PARTY.length) {
		for (const key of RELYING_PARTY) if (!present(key)) refuse(key);
	}
	if (rp.length === 0) {
		if (sealKey !== undefined) refuse('SESSION_SEAL_KEY');
		if (scopes !== undefined) refuse('OIDC_SCOPES');
	}
	if (present('OIDC_ISSUER') && [...(sealKey ?? '')].length < MIN_SESSION_SEAL_KEY) {
		refuse('SESSION_SEAL_KEY');
	}
	if (rp.length === RELYING_PARTY.length && !isHttpsUrl(source.OIDC_ISSUER ?? '')) {
		refuse('OIDC_ISSUER');
	}
	if (scopes === '') refuse('OIDC_SCOPES');
	if (present('OIDC_AUTO_PROVISION')) refuse('OIDC_AUTO_PROVISION');

	if (rp.length < RELYING_PARTY.length) return null;
	return {
		issuer: source.OIDC_ISSUER ?? '',
		clientId: source.OIDC_CLIENT_ID ?? '',
		clientSecret: source.OIDC_CLIENT_SECRET ?? '',
		scopes: scopes ?? OIDC_DEFAULT_SCOPES,
		sealKey: sealKey ?? ''
	};
}

function isPort(value: string): boolean {
	return /^\d{1,5}$/.test(value) && Number(value) >= 1 && Number(value) <= MAX_PORT;
}

// Error messages name the variable only, never its value.
export function loadEnv(source: Record<string, string | undefined>): Env {
	const bad: string[] = [];
	const read = (key: string, ok: (value: string) => boolean): string => {
		const value = source[key] ?? '';
		if (!ok(value)) bad.push(key);
		return value;
	};

	const env = {
		internalApiToken: read('INTERNAL_API_TOKEN', (v) => [...v].length >= MIN_INTERNAL_API_TOKEN),
		apiBaseUrl: read('API_BASE_URL', isHttpUrl),
		origin: read('ORIGIN', isHttpUrl),
		port: read('PORT', isPort)
	};
	const oidc = loadOidc(source, bad);

	if (bad.length > 0) throw new Error(`invalid environment: ${bad.join(', ')}`);
	return { ...env, origin: new URL(env.origin).origin, port: Number(env.port), oidc };
}

let cached: Env | undefined;

export function getEnv(): Env {
	cached ??= loadEnv(privateEnv);
	return cached;
}
