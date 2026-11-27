import type { Page } from '@playwright/test';
import { untilHydrated } from '../helpers.js';
import { ANA, CLEO, DAN, IVY, LEO, setScenario } from '../stub-api.js';
import { as, at, kidPath, must } from './seed.js';
import { shot, signInAs, test } from './shots.js';

const TODAY = at('00:00').slice(0, 10);
const daysAgo = (n: number) => at('00:00', n).slice(0, 10);
const url = (kid: string, query = '') => `/kids/${kid}/export${query}`;
const modules = (page: Page) => page.getByRole('group', { name: 'Modules to include' });

async function open(page: Page, path: string) {
	await page.goto(path);
	await page.waitForLoadState('networkidle');
}

async function download(page: Page, error: string) {
	await page.getByRole('button', { name: 'Download PDF' }).click();
	await page.waitForURL((u) => u.searchParams.get('error') === error);
}

test('export', async ({ page }) => {
	const ana = await as(ANA);
	await signInAs(page, ANA);
	await open(page, url(IVY.id));
	await shot(page, 'export', '01-default', 'Export PDF: the last 30 days, every module ticked');
	await must(ana, 'PUT', kidPath(IVY.id, 'modules/pumping'), { enabled: false });
	await open(page, url(IVY.id));
	await shot(
		page,
		'export',
		'02-module-switched-off',
		'A switched-off module sits unticked under its own sub-heading'
	);
	await untilHydrated(modules(page).getByRole('checkbox').first(), 'checked');
	for (const box of await modules(page).getByRole('checkbox').all()) await box.uncheck();
	await download(page, 'modules');
	await shot(
		page,
		'export',
		'03-no-module-error',
		'Every module unticked: the message, the typed dates kept'
	);
	await open(page, url(IVY.id));
	await untilHydrated(page.getByLabel('From', { exact: true }));
	await page.getByLabel('From', { exact: true }).fill(daysAgo(94));
	await download(page, 'to');
	await shot(page, 'export', '04-range-too-long', 'A 95-day range: the message on To');
	await open(page, url(IVY.id, `?error=from&from=${TODAY}&to=${TODAY}`));
	await shot(
		page,
		'export',
		'05-from-error',
		'A refused From date, as the redirect after a bad POST shows it'
	);
	await setScenario('export_busy');
	await open(page, url(IVY.id));
	await download(page, 'busy');
	await shot(
		page,
		'export',
		'06-busy',
		'The API is busy: try again in a moment, the same choices kept'
	);
});

test('roles', async ({ page }) => {
	await must(await as(CLEO), 'PUT', kidPath(LEO.id, 'modules/diapers'), { enabled: true });
	await signInAs(page, ANA);
	await open(page, url(LEO.id));
	await shot(
		page,
		'export',
		'07-caregiver',
		'A caregiver can export (2-year-old in Europe/Madrid)'
	);
	await signInAs(page, DAN);
	await page.goto(url(IVY.id));
	await shot(page, 'export', '08-no-grant-404', 'A parent without a grant gets the 404 page');
});
