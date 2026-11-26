import { execSync, spawn, type ChildProcess } from 'node:child_process';
import { dirname } from 'node:path';
import type { FullConfig } from '@playwright/test';
import { slotPorts, STUB_TOKEN } from './stub-api.js';

const START_TIMEOUT_MS = 60_000;
const POLL_MS = 100;

const answers = (url: string) =>
	fetch(url).then(
		(r) => r.ok,
		() => false
	);

async function start(
	cwd: string,
	name: string,
	args: string[],
	env: NodeJS.ProcessEnv,
	readyUrl: string
) {
	if (await answers(readyUrl)) throw new Error(`${name}: ${readyUrl} is already in use`);
	// Like webServer: stdout dropped, stderr shown.
	const child = spawn('node', args, {
		cwd,
		env: { ...process.env, ...env },
		stdio: ['ignore', 'ignore', 'inherit']
	});
	const exited = new Promise<never>((_, reject) =>
		child.once('exit', (code) =>
			reject(new Error(`${name} exited (${code}) before ${readyUrl} answered`))
		)
	);
	const ready = (async () => {
		const deadline = Date.now() + START_TIMEOUT_MS;
		while (!(await answers(readyUrl))) {
			if (Date.now() > deadline) throw new Error(`${name}: ${readyUrl} did not answer in time`);
			await new Promise((r) => setTimeout(r, POLL_MS));
		}
	})();
	return { child, ready: Promise.race([ready, exited]) };
}

// One stub API and one UI per worker slot: a worker reaches only its own, so no state is shared.
export default async function globalSetup(config: FullConfig) {
	const cwd = dirname(config.configFile ?? process.cwd());
	if (process.env.E2E_SKIP_BUILD !== '1') execSync('pnpm build', { cwd, stdio: 'inherit' });
	const children: ChildProcess[] = [];
	const stop = () => children.forEach((c) => c.kill());
	try {
		await Promise.all(
			Array.from({ length: config.workers }, async (_, i) => {
				const { stub, ui } = slotPorts(i);
				const stubUrl = `http://127.0.0.1:${stub}`;
				const uiOrigin = `http://127.0.0.1:${ui}`;
				const slot = { TEST_PARALLEL_INDEX: String(i) };
				const s = await start(cwd, `stub ${i}`, ['e2e/stub-api.ts'], slot, `${stubUrl}/__requests`);
				children.push(s.child);
				await s.ready;
				const u = await start(
					cwd,
					`ui ${i}`,
					['build'],
					{
						INTERNAL_API_TOKEN: STUB_TOKEN,
						API_BASE_URL: stubUrl,
						ORIGIN: uiOrigin,
						PROTOCOL_HEADER: 'x-forwarded-proto',
						HOST_HEADER: 'x-forwarded-host',
						ADDRESS_HEADER: 'x-forwarded-for',
						XFF_DEPTH: '1',
						HOST: '127.0.0.1',
						PORT: String(ui)
					},
					`${uiOrigin}/healthz`
				);
				children.push(u.child);
				await u.ready;
			})
		);
	} catch (e) {
		stop();
		throw e;
	}
	return stop;
}
