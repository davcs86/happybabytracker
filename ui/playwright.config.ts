import { defineConfig } from '@playwright/test';
import { STUB_TOKEN, STUB_URL, UI_ORIGIN, UI_PORT } from './e2e/stub-api.js';

const testEnv = {
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

export default defineConfig({
	testDir: 'e2e',
	retries: 0,
	// The stub API's state is shared by every test.
	workers: 1,
	use: {
		baseURL: UI_ORIGIN,
		// The design's motion is user-triggered and honours this; tests assert the settled page.
		reducedMotion: 'reduce',
		extraHTTPHeaders: { 'x-forwarded-proto': 'http', 'x-forwarded-for': '203.0.113.10' },
		launchOptions: process.env.PW_CHROMIUM_PATH
			? { executablePath: process.env.PW_CHROMIUM_PATH }
			: {}
	},
	projects: [
		{ name: 'phone', use: { viewport: { width: 390, height: 844 } } },
		{ name: 'desktop', use: { viewport: { width: 1280, height: 800 } } },
		{
			name: 'side-nav',
			testMatch: /(shell|theme)\.spec\.ts/,
			use: { viewport: { width: 980, height: 800 } }
		}
	],
	webServer: [
		{
			command: 'node e2e/stub-api.ts',
			url: `${STUB_URL}/__requests`,
			reuseExistingServer: false
		},
		{
			command: 'pnpm build && node build',
			url: `${UI_ORIGIN}/healthz`,
			env: testEnv,
			timeout: 180_000,
			reuseExistingServer: false
		}
	]
});
