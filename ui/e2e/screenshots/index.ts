// Writes docs/screenshots/README.md from the last full run's manifest, or with --check compares it
// with the PNGs on disk. Run with node from ui/ after `playwright test -c playwright.screenshots.config.ts`.
import { execFileSync } from 'node:child_process';
import { existsSync, readdirSync, readFileSync, writeFileSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';
import { AREAS, PROJECTS } from './areas.ts';
import { SHOTS_NOW } from './clock.ts';
import { ABOUT_AREAS, FIXTURES, NOT_CAPTURED } from './notes.ts';

type Entry = { area: string; name: string; about: string; project: string; path: string };

const ROOT = resolve(import.meta.dirname, '../../..');
const DIR = join(ROOT, 'docs/screenshots');
const README = join(DIR, 'README.md');
const MANIFEST = join(ROOT, 'ui/test-results/screenshots-manifest.jsonl');
const ROUTES = join(ROOT, 'ui/src/routes');

function pngsOnDisk(): string[] {
	if (!existsSync(DIR)) return [];
	return readdirSync(DIR, { withFileTypes: true })
		.filter((d) => d.isDirectory())
		.flatMap((d) =>
			readdirSync(join(DIR, d.name))
				.filter((f) => f.endsWith('.png'))
				.map((f) => `${d.name}/${f}`)
		)
		.sort();
}

function fail(problems: string[]): never {
	for (const p of problems) console.error(p);
	process.exit(1);
}

function compare(listed: string[], label: string) {
	const disk = new Set(pngsOnDisk());
	const want = new Set(listed);
	const problems = [
		...[...want].filter((f) => !disk.has(f)).map((f) => `${label} lists a missing file: ${f}`),
		...[...disk].filter((f) => !want.has(f)).map((f) => `on disk but not in ${label}: ${f}`)
	];
	if (problems.length > 0) fail(problems);
	console.log(`${label}: ${want.size} files, matching docs/screenshots exactly`);
}

if (process.argv.includes('--check')) {
	const text = readFileSync(README, 'utf8');
	const links = [...text.matchAll(/\]\(([a-z-]+\/[a-z0-9-]+\.png)\)/g)].map((m) => m[1]);
	compare(links, 'README.md');
	process.exit(0);
}

const entries: Entry[] = readFileSync(MANIFEST, 'utf8')
	.split('\n')
	.filter(Boolean)
	.map((l) => JSON.parse(l) as Entry);
const file = (e: Pick<Entry, 'area' | 'name'>, project: string) =>
	`${e.area}/${e.name}-${project}.png`;

// One row per area and name, which both projects must have taken.
const rows = new Map<string, Entry & { paths: Set<string>; projects: Set<string> }>();
for (const e of entries) {
	const key = `${e.area}/${e.name}`;
	const row = rows.get(key) ?? { ...e, paths: new Set(), projects: new Set() };
	if (row.projects.has(e.project)) fail([`duplicate shot ${key} in ${e.project}`]);
	if (row.about !== e.about) fail([`${key}: the projects describe it differently`]);
	row.projects.add(e.project);
	row.paths.add(e.path);
	rows.set(key, row);
}
const incomplete = [...rows.entries()]
	.filter(([, r]) => PROJECTS.some((p) => !r.projects.has(p)))
	.map(([k]) => `${k} is missing a project: run the whole set`);
if (incomplete.length > 0) fail(incomplete);
const sorted = [...rows.values()].sort(
	(a, b) =>
		AREAS.indexOf(a.area as (typeof AREAS)[number]) -
			AREAS.indexOf(b.area as (typeof AREAS)[number]) || a.name.localeCompare(b.name)
);
const files = sorted.flatMap((r) => PROJECTS.map((p) => file(r, p)));
compare(files, 'manifest');

// Lossless only; the PNGs stay byte-identical between runs either way.
try {
	execFileSync('oxipng', ['-q', '-o', '4', '--strip', 'safe', ...files.map((f) => join(DIR, f))]);
	console.log('oxipng: optimised losslessly');
} catch {
	console.log('oxipng not found or failed: PNGs left as Playwright wrote them');
}

// Every page route, matched against the paths the shots were taken on.
function pageRoutes(dir: string): string[] {
	return readdirSync(dir, { withFileTypes: true }).flatMap((d) => {
		const p = join(dir, d.name);
		if (d.isDirectory()) return pageRoutes(p);
		return d.name === '+page.svelte' || d.name === '+server.ts' ? [p] : [];
	});
}
const routes = pageRoutes(ROUTES)
	.map((p) => {
		const rel = relative(ROUTES, p).split('/');
		const kind = rel.pop() === '+page.svelte' ? 'page' : 'endpoint';
		const route = `/${rel.filter((s) => !/^\(.*\)$/.test(s)).join('/')}`;
		return { route, kind };
	})
	.sort((a, b) => a.route.localeCompare(b.route));
const pattern = (route: string) =>
	new RegExp(
		`^${route.replace(/\[([a-z]+)(=[a-z]+)?\]/g, '[^/]+').replace(/\/$/, '')}/?$`.replace(
			'^/?$',
			'^/$'
		)
	);
// A literal segment beats a parameter: /kids/new is not /kids/[kid].
const routeOf = (path: string) =>
	routes
		.filter((r) => r.kind === 'page' && pattern(r.route).test(path))
		.sort((a, b) => (b.route.match(/\[/g)?.length ?? 0) - (a.route.match(/\[/g)?.length ?? 0))
		.pop()?.route;

const byRoute = new Map<string, string[]>();
for (const r of sorted) {
	for (const p of r.paths) {
		const route = routeOf(p);
		if (!route) continue;
		const names = byRoute.get(route) ?? [];
		if (!names.includes(`${r.area}/${r.name}`)) names.push(`${r.area}/${r.name}`);
		byRoute.set(route, names);
	}
}

const out: string[] = [
	'# happybabytracker UI screenshots',
	'',
	'<!-- Generated by ui/e2e/screenshots/index.ts: edit the shots or notes.ts, then regenerate. -->',
	'',
	`${sorted.length} screens and states, each at two viewports: **mobile** 390×844 and **desktop** 1280×800,`,
	'light colour scheme, JavaScript on, device scale factor 1, full-page captures.',
	'Every value shown is fake stub data from `ui/e2e/stub-api.ts`; no real person or child appears.',
	'',
	'## Regenerate',
	'',
	'```sh',
	'cd ui',
	'PW_CHROMIUM_PATH=/path/to/chrome-headless-shell pnpm screenshots',
	'node e2e/screenshots/index.ts --check   # the README lists exactly the PNGs on disk',
	'```',
	'',
	'`pnpm screenshots` builds the UI, starts it and the stub API on ports 4173 and 4181 (one Playwright',
	'run at a time), runs `ui/e2e/screenshots/*.shot.ts` (never part of `pnpm test:e2e`), then runs',
	'`index.ts`, which checks that every shot exists in both viewports and nothing else is on disk,',
	'optimises the PNGs with `oxipng` when it is installed (lossless), and rewrites this README.',
	'A shot that is renamed or removed leaves its old PNGs behind: delete them, or the check fails.',
	'',
	'## Determinism',
	'',
	`The stub API and the UI server run with \`ui/e2e/screenshots/freeze-clock.ts\` preloaded, and the`,
	`browser uses Playwright's \`page.clock.setFixedTime\`: all three read **${SHOTS_NOW}**`,
	'(browser time zone UTC, locale en-US). Ages, dates, "today" and every relative time are therefore',
	'the same on every run. Animations are disabled, reduced motion is emulated, the caret is hidden,',
	'focus is cleared and the pointer is parked before each capture, after the network is idle and the',
	'fonts are loaded. Two consecutive runs produce byte-identical PNGs.',
	'',
	'## Fixtures',
	'',
	...FIXTURES(SHOTS_NOW),
	'',
	'## Coverage by route',
	'',
	'| Route | Kind | Shots |',
	'|---|---|---|',
	...routes.map(({ route, kind }) => {
		const names = byRoute.get(route);
		const cell =
			kind === 'endpoint'
				? 'not a page'
				: names
					? names.map((n) => `\`${n}\``).join(', ')
					: '**none**';
		return `| \`${route}\` | ${kind} | ${cell} |`;
	}),
	'',
	'## Index',
	''
];
for (const area of AREAS) {
	const inArea = sorted.filter((r) => r.area === area);
	if (inArea.length === 0) continue;
	out.push(`### ${area}`, '', ABOUT_AREAS[area], '', '| Shot | What it shows | Mobile | Desktop |');
	out.push('|---|---|---|---|');
	for (const r of inArea) {
		const links = PROJECTS.map((p) => `[${p}](${file(r, p)})`);
		out.push(`| \`${r.name}\` | ${r.about.replace(/\|/g, '\\|')} | ${links.join(' | ')} |`);
	}
	out.push('');
}
out.push('## Not captured', '', ...NOT_CAPTURED.map((n) => `- ${n}`), '');
writeFileSync(README, out.join('\n'));
console.log(`README.md: ${sorted.length} shots, ${files.length} files`);
