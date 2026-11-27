import { createCipheriv, createDecipheriv, hkdfSync, randomBytes } from 'node:crypto';
import type { OidcCookie } from './types';

const INFO = 'hbt_oidc v1';
const AAD = Buffer.from('hbt_oidc');
const IV_BYTES = 12;
const TAG_BYTES = 16;

function key(sealKey: string): Buffer {
	return Buffer.from(hkdfSync('sha256', sealKey, '', INFO, 32));
}

export function seal(sealKey: string, payload: OidcCookie): string {
	const iv = randomBytes(IV_BYTES);
	const cipher = createCipheriv('aes-256-gcm', key(sealKey), iv);
	cipher.setAAD(AAD);
	const ct = Buffer.concat([cipher.update(JSON.stringify(payload), 'utf8'), cipher.final()]);
	return Buffer.concat([iv, ct, cipher.getAuthTag()]).toString('base64url');
}

function isCookie(v: unknown): v is OidcCookie {
	if (typeof v !== 'object' || v === null) return false;
	const c = v as Record<string, unknown>;
	if (typeof c.exp !== 'number') return false;
	if (c.stage === 'start') return typeof c.startedAt === 'number';
	return c.stage === 'pending';
}

// Null on any failure: a forged, stale, foreign or expired value is never trusted.
export function open(sealKey: string, value: string | undefined, nowS: number): OidcCookie | null {
	if (!value || !/^[A-Za-z0-9_-]+$/.test(value)) return null;
	const raw = Buffer.from(value, 'base64url');
	if (raw.length <= IV_BYTES + TAG_BYTES) return null;
	try {
		const decipher = createDecipheriv('aes-256-gcm', key(sealKey), raw.subarray(0, IV_BYTES));
		decipher.setAAD(AAD);
		decipher.setAuthTag(raw.subarray(raw.length - TAG_BYTES));
		const plain = Buffer.concat([
			decipher.update(raw.subarray(IV_BYTES, raw.length - TAG_BYTES)),
			decipher.final()
		]);
		const payload: unknown = JSON.parse(plain.toString('utf8'));
		if (!isCookie(payload) || payload.exp <= nowS) return null;
		return payload;
	} catch {
		return null;
	}
}
