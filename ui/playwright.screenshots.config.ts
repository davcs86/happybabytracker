import { defineConfig } from '@playwright/test';
import { SHOTS_NOW } from './e2e/screenshots/clock.js';
import { STUB_TOKEN, STUB_URL, UI_ORIGIN, UI_PORT } from './e2e/stub-api.js';

// Regenerates docs/screenshots; never part of the e2e suite (testMatch *.shot.ts only).
const frozen = { SHOTS_NOW };
const testEnv = {
	...frozen,
	INTERNAL_API_TOKEN: STUB_TOKEN,
	API_BASE_URL: STUB_URL,
	ORIGIN: UI_ORIGIN,
	PROTOCOL_HEADER: 'x-forwarded-proto',
	HOST_HEADER: 'x-forwarded-host',
	ADDRESS_HEADER: 'x-forwarded-for',
	XFF_DEPTH: '1',
	HOST: '127.0.0.1',
	PORT: String(UI_PORT)
};
const freeze = '--import ./e2e/screenshots/freeze-clock.ts';

export default defineConfig({
	testDir: 'e2e/screenshots',
	testMatch: '**/*.shot.ts',
	retries: 0,
	// The stub API's state is shared by every test.
	workers: 1,
	timeout: 120_000,
	use: {
		baseURL: UI_ORIGIN,
		actionTimeout: 15_000,
		navigationTimeout: 30_000,
		extraHTTPHeaders: { 'x-forwarded-proto': 'http', 'x-forwarded-for': '203.0.113.10' },
		colorScheme: 'light',
		deviceScaleFactor: 1,
		locale: 'en-US',
		timezoneId: 'UTC',
		launchOptions: process.env.PW_CHROMIUM_PATH
			? { executablePath: process.env.PW_CHROMIUM_PATH }
			: {}
	},
	projects: [
		{ name: 'mobile', use: { viewport: { width: 390, height: 844 } } },
		{ name: 'desktop', use: { viewport: { width: 1280, height: 800 } } }
	],
	webServer: [
		{
			command: `node ${freeze} e2e/stub-api.ts`,
			url: `${STUB_URL}/__requests`,
			env: frozen,
			reuseExistingServer: false
		},
		{
			command: `pnpm build && node ${freeze} build`,
			url: `${UI_ORIGIN}/healthz`,
			env: testEnv,
			timeout: 180_000,
			reuseExistingServer: false
		}
	]
});
