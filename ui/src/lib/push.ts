// Shared by the page and the service worker: no server-only or DOM-only imports.

// Structural, so the worker and the page need no DOM type; the endpoint and keys are secrets.
type SubscriptionLike = {
	toJSON(): { endpoint?: unknown; keys?: Record<string, unknown> };
};

// Throws on text outside the base64url alphabet or of an impossible length.
export function base64UrlToBytes(text: string): Uint8Array<ArrayBuffer> {
	if (!/^[A-Za-z0-9_-]*={0,2}$/.test(text)) throw new Error('not base64url');
	const raw = text.replace(/=+$/, '');
	if (raw.length % 4 === 1) throw new Error('not base64url');
	const padded = raw.replace(/-/g, '+').replace(/_/g, '/') + '='.repeat((4 - (raw.length % 4)) % 4);
	const binary = atob(padded);
	return Uint8Array.from(binary, (c) => c.charCodeAt(0));
}

const nonEmpty = (v: unknown): v is string => typeof v === 'string' && v !== '';

// The body of an upsert without its label; null when the browser's subscription lacks a part.
export function subscriptionBody(
	sub: SubscriptionLike
): { endpoint: string; keys: { p256dh: string; auth: string } } | null {
	const { endpoint, keys } = sub.toJSON();
	if (!nonEmpty(endpoint) || !keys || !nonEmpty(keys.p256dh) || !nonEmpty(keys.auth)) return null;
	return { endpoint, keys: { p256dh: keys.p256dh, auth: keys.auth } };
}

// One leading slash and no backslash or control character: a URL parser reads the rest as another origin.
const isSameOriginPath = (p: string) =>
	/^\/(?![/\\])/.test(p) && ![...p].some((c) => c === '\\' || c <= '\u001f' || c === '\u007f');

export function parsePayload(data: unknown): { title: string; path: string; tag: string } | null {
	if (typeof data !== 'object' || data === null || Array.isArray(data)) return null;
	const { title, path, tag } = data as Record<string, unknown>;
	if (!nonEmpty(title) || !nonEmpty(tag) || typeof path !== 'string' || !isSameOriginPath(path)) {
		return null;
	}
	return { title, path, tag };
}

// iPadOS answers with a Mac user agent, so a Mac with a touch screen counts as an iPad.
export function isIosOutsideStandalone(
	ua: string,
	standalone: boolean,
	maxTouchPoints = 0
): boolean {
	if (standalone) return false;
	return /iPhone|iPad|iPod/.test(ua) || (/Macintosh/.test(ua) && maxTouchPoints > 1);
}
