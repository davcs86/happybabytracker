import { defineConfig } from '@playwright/test';
import main from './playwright.config.ts';
// Its servers, and the workers once imported, run frozen at the screenshots' instant, so the
// baselines made from the build (visits) keep their dates and ages.
import shots from './playwright.screenshots.config.ts';

const base = {
	...main.use,
	deviceScaleFactor: 1,
	locale: 'en-US',
	timezoneId: 'America/Los_Angeles'
};
const variant = (name: string, width: number, colorScheme: 'light' | 'dark') => ({
	name,
	use: { ...base, viewport: { width, height: width === 390 ? 800 : 780 }, colorScheme }
});

export default defineConfig({
	testDir: 'tests/visual',
	testMatch: '**/*.visual.spec.ts',
	// Baselines: tests/visual/__baselines__/<variant>/<page>/NN-<state>.png, kept in git LFS.
	snapshotPathTemplate: '{testDir}/__baselines__/{projectName}/{arg}{ext}',
	workers: 1,
	retries: 0,
	webServer: shots.webServer,
	projects: [
		variant('phone-light', 390, 'light'),
		variant('phone-dark', 390, 'dark'),
		variant('desktop-light', 980, 'light'),
		variant('desktop-dark', 980, 'dark'),
		variant('wide-light', 1280, 'light'),
		variant('wide-dark', 1280, 'dark')
	]
});
