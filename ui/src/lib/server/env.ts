import * as privateEnv from '$app/env/private';
import { MAX_PORT, MIN_INTERNAL_API_TOKEN } from './constants';

export type Env = {
	internalApiToken: string;
	apiBaseUrl: string;
	origin: string;
	port: number;
};

function isHttpUrl(value: string): boolean {
	return URL.canParse(value) && /^https?:$/.test(new URL(value).protocol);
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

	if (bad.length > 0) throw new Error(`invalid environment: ${bad.join(', ')}`);
	return { ...env, origin: new URL(env.origin).origin, port: Number(env.port) };
}

let cached: Env | undefined;

export function getEnv(): Env {
	cached ??= loadEnv(privateEnv);
	return cached;
}
