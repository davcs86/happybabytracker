import { execSync, spawn, type ChildProcess } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { get } from 'node:https';
import { dirname } from 'node:path';
import type { FullConfig } from '@playwright/test';
import { slotPorts, STUB_OIDC_CLIENT, STUB_SEAL_KEY, STUB_TOKEN } from './stub-api.js';
import { ensureTls } from './tls.js';

const START_TIMEOUT_MS = 60_000;
const POLL_MS = 100;

const answers = (url: string, ca?: Buffer) =>
	ca
		? new Promise<boolean>((done) => {
				get(url, { ca }, (r) => {
					r.resume();
					done((r.statusCode ?? 0) >= 200 && (r.statusCode ?? 0) < 300);
				}).on('error', () => done(false));
			})
		: fetch(url).then(
				(r) => r.ok,
				() => false
			);

async function start(
	cwd: string,
	name: string,
	args: string[],
	env: NodeJS.ProcessEnv,
	readyUrl: string,
	ca?: Buffer
) {
	if (await answers(readyUrl, ca)) throw new Error(`${name}: ${readyUrl} is already in use`);
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
		while (!(await answers(readyUrl, ca))) {
			if (Date.now() > deadline) throw new Error(`${name}: ${readyUrl} did not answer in time`);
			await new Promise((r) => setTimeout(r, POLL_MS));
		}
	})();
	return { child, ready: Promise.race([ready, exited]) };
}

// One stub API, stub issuer and UI per worker slot: a worker reaches only its own, so no state is shared.
export default async function globalSetup(config: FullConfig) {
	const cwd = dirname(config.configFile ?? process.cwd());
	if (process.env.E2E_SKIP_BUILD !== '1') execSync('pnpm build', { cwd, stdio: 'inherit' });
	const tls = ensureTls();
	const ca = readFileSync(tls.cert);
	const children: ChildProcess[] = [];
	const stop = () => children.forEach((c) => c.kill());
	try {
		await Promise.all(
			Array.from({ length: config.workers }, async (_, i) => {
				const { stub, ui, issuer } = slotPorts(i);
				const stubUrl = `http://127.0.0.1:${stub}`;
				const uiOrigin = `http://127.0.0.1:${ui}`;
				const issuerUrl = `https://127.0.0.1:${issuer}`;
				const slot = { TEST_PARALLEL_INDEX: String(i) };
				const s = await start(cwd, `stub ${i}`, ['e2e/stub-api.ts'], slot, `${stubUrl}/__requests`);
				children.push(s.child);
				await s.ready;
				const o = await start(
					cwd,
					`issuer ${i}`,
					['e2e/stub-issuer.ts'],
					slot,
					`${issuerUrl}/.well-known/openid-configuration`,
					ca
				);
				children.push(o.child);
				await o.ready;
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
						PORT: String(ui),
						OIDC_ISSUER: issuerUrl,
						OIDC_CLIENT_ID: STUB_OIDC_CLIENT.id,
						OIDC_CLIENT_SECRET: STUB_OIDC_CLIENT.secret,
						SESSION_SEAL_KEY: STUB_SEAL_KEY,
						NODE_EXTRA_CA_CERTS: tls.cert
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
