import { expect, test, type Page } from '@playwright/test';
import { login, openSheet, submit } from '../../e2e/helpers.js';
import { as, seedVisits } from '../../e2e/screenshots/seed.js';
import { NOW_MS } from '../../e2e/screenshots/shots.js';
import { ANA, MIA, resetStub, setScenario } from '../../e2e/stub-api.js';

// The visits pages have no design baseline (page spec, Acceptance): these were made from the build.
const TOLERANCE = 0.01;
const list = `/kids/${MIA.id}/visits`;
const visit = (id: string) => `${list}/${id}`;

async function ready(page: Page, seed: boolean) {
	await resetStub();
	await page.clock.setFixedTime(new Date(NOW_MS));
	const v = seed ? await seedVisits(await as(ANA), MIA.id) : null;
	await page.goto('/login');
	await login(page, ANA.username, ANA.password);
	await page.waitForURL((u) => u.pathname !== '/login');
	return v;
}

async function compare(page: Page, name: string, fullPage: boolean) {
	await page.waitForLoadState('networkidle');
	await page.evaluate(() => document.fonts.ready);
	await page.mouse.move(0, 0);
	await expect(page).toHaveScreenshot(['visits', `${name}.png`], {
		fullPage,
		maxDiffPixelRatio: TOLERANCE,
		animations: 'disabled',
		caret: 'hide'
	});
}

type Seeded = Awaited<ReturnType<typeof seedVisits>>;

const pages: [string, (v: Seeded) => string][] = [
	['01-list-default', () => list],
	['03-visit-upcoming', (v) => visit(v.checkup.id)],
	['04-visit-past-outcome', (v) => visit(v.sick.id)],
	['05-visit-cancelled', (v) => visit(v.cancelled.id)]
];

for (const [name, path] of pages) {
	test(`visits · ${name}`, async ({ page }) => {
		const v = (await ready(page, true))!;
		await page.goto(path(v));
		await compare(page, name, true);
	});
}

test('visits · 02-list-empty', async ({ page }) => {
	await ready(page, false);
	await page.goto(list);
	await compare(page, '02-list-empty', true);
});

// Name, the sheet's title, the link that opens it, and the page it opens over.
const sheets: [string, string, string, (v: Seeded) => string][] = [
	['06-sheet-add', 'Add a visit', 'Add a visit', () => list],
	['07-sheet-edit', 'Edit visit', 'Edit visit', (v) => visit(v.checkup.id)],
	['08-sheet-after', 'After the visit', 'Record the visit', (v) => visit(v.plain.id)]
];

for (const [name, title, link, from] of sheets) {
	test(`visits · ${name}`, async ({ page }) => {
		const v = (await ready(page, true))!;
		await openSheet(page, from(v), (p) => p.getByRole('link', { name: link }), title);
		await compare(page, name, false);
	});
}

// The "Find a US provider" panel (page spec, plan 10 Acceptance): results, after a pick, unavailable.
const panel: [string, 'npi_on' | 'npi_unavailable', boolean][] = [
	['09-sheet-provider-results', 'npi_on', false],
	['10-sheet-provider-picked', 'npi_on', true],
	['11-sheet-provider-unavailable', 'npi_unavailable', false]
];

for (const [name, scenario, pick] of panel) {
	test(`visits · ${name}`, async ({ page }) => {
		await ready(page, false);
		await setScenario(scenario);
		const sheet = await openSheet(
			page,
			list,
			(p) => p.getByRole('link', { name: 'Add a visit' }),
			'Add a visit'
		);
		const button = (name: string) => sheet.getByRole('button', { name, exact: true });
		await sheet.getByText('Find a US provider', { exact: true }).click();
		await sheet.getByLabel('Last name', { exact: true }).fill('Quill');
		await sheet.getByLabel('State', { exact: true }).selectOption('WA');
		await submit(button('Search'));
		const shown =
			scenario === 'npi_on'
				? sheet.getByRole('radiogroup', { name: 'Choose a provider' })
				: sheet.getByRole('alert');
		await expect(shown).toBeVisible();
		if (pick) {
			await sheet.getByRole('radio', { name: 'AVERY QUILL, MD', exact: true }).check();
			await submit(button('Use this provider'));
			await expect(sheet.getByLabel('Last name', { exact: true })).toBeHidden();
			await sheet.getByLabel('Place (optional)', { exact: true }).scrollIntoViewIfNeeded();
		} else {
			await shown.scrollIntoViewIfNeeded();
		}
		await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur?.());
		await compare(page, name, false);
	});
}
