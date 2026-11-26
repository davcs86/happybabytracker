import { defineConfig } from '@playwright/test';
import { UI_ORIGIN } from './e2e/stub-api.js';

const phone = { width: 390, height: 844 };
const stubSpecs = /stub-[^/]*\.spec\.ts$/;

export default defineConfig({
	testDir: 'e2e',
	retries: 0,
	// Starts one stub API and one UI per worker; E2E_SKIP_BUILD=1 reuses ./build, stale or not.
	globalSetup: './e2e/stacks.ts',
	use: {
		// Read in each worker: its own UI.
		baseURL: UI_ORIGIN,
		// The design's motion is user-triggered and honours this; tests assert the settled page.
		reducedMotion: 'reduce',
		extraHTTPHeaders: { 'x-forwarded-proto': 'http', 'x-forwarded-for': '203.0.113.10' },
		launchOptions: process.env.PW_CHROMIUM_PATH
			? { executablePath: process.env.PW_CHROMIUM_PATH }
			: {}
	},
	projects: [
		// The stub's own contract does not depend on the viewport: one run.
		{ name: 'stub', testMatch: stubSpecs, use: { viewport: phone } },
		{ name: 'phone', testIgnore: stubSpecs, use: { viewport: phone } },
		{
			name: 'desktop',
			testIgnore: stubSpecs,
			use: { viewport: { width: 1280, height: 800 } }
		},
		{
			name: 'side-nav',
			testMatch: /(shell|theme)\.spec\.ts/,
			use: { viewport: { width: 980, height: 800 } }
		}
	]
});
