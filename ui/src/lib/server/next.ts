const BASE = 'http://next.invalid';

const unsafe = (c: string) => c === '\\' || c.charCodeAt(0) < 0x20 || c.charCodeAt(0) === 0x7f;

// Only a path on this origin: never `//host`, a backslash, a control character or another scheme.
export function safeNext(raw: string | null): string | null {
	if (!raw || !raw.startsWith('/') || raw.startsWith('//') || [...raw].some(unsafe)) {
		return null;
	}
	const url = new URL(raw, BASE);
	const path = url.pathname + url.search + url.hash;
	return url.origin === BASE && !path.startsWith('//') ? path : null;
}
