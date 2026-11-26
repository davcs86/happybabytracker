const BROWSERS: [RegExp, string][] = [
	[/Edg(e|A|iOS)?\//, 'Edge'],
	[/OPR\/|Opera/, 'Opera'],
	[/Firefox\/|FxiOS\//, 'Firefox'],
	[/Chrome\/|CriOS\//, 'Chrome'],
	[/Version\/.*Safari\//, 'Safari']
];
const SYSTEMS: [RegExp, string][] = [
	[/iPhone/, 'iPhone'],
	[/iPad/, 'iPad'],
	[/Android/, 'Android'],
	[/CrOS/, 'ChromeOS'],
	[/Macintosh|Mac OS X/, 'Mac'],
	[/Windows/, 'Windows'],
	[/Linux|X11/, 'Linux']
];

// A coarse name for the sign-in list, from a closed set of words: never the user agent's own text.
export function deviceLabel(userAgent: string): string {
	const ua = userAgent.slice(0, 512);
	const browser = BROWSERS.find(([re]) => re.test(ua))?.[1] ?? 'Browser';
	const system = SYSTEMS.find(([re]) => re.test(ua))?.[1];
	return system ? `${browser} on ${system}` : browser;
}
