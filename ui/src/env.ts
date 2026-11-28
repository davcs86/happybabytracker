import { defineEnvVars } from '@sveltejs/kit/env';

// Lenient on purpose: loadEnv() in lib/server/env.ts owns validation and its error messages.
const optional = { schema: (value: string | undefined) => value ?? '' };
// Keeps undefined: a variable that is present counts as set, even when empty.
const presence = { schema: (value: string | undefined) => value };

export const variables = defineEnvVars({
	INTERNAL_API_TOKEN: optional,
	API_BASE_URL: optional,
	ORIGIN: optional,
	PORT: optional,
	OIDC_ISSUER: presence,
	OIDC_CLIENT_ID: presence,
	OIDC_CLIENT_SECRET: presence,
	OIDC_SCOPES: presence,
	SESSION_SEAL_KEY: presence,
	OIDC_AUTO_PROVISION: presence
});
