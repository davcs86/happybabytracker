import { readFile } from 'node:fs/promises';
import { expect, test, type Download, type Page } from '@playwright/test';
import { EXPORT_MODULES, MODULE_LABEL } from '../src/lib/constants.js';
import { todayIn } from '../src/lib/format.js';
import { EXPORT_DEFAULT_DAYS } from '../src/lib/server/constants.js';
import { expectAccessible, expectNotFoundPage, navLink, signIn, untilHydrated } from './helpers.js';
import {
	ADMIN,
	ANA,
	DAN,
	IVY,
	LEO,
	UI_ORIGIN,
	receivedRequests,
	resetStub,
	setScenario,
	stubAs
} from './stub-api.js';

// The export page posts a plain form to /kids/<kid>/export/pdf, which streams the API's PDF. The stub's PDF is
// a marked stand-in whose one text line lists the modules it received, so a test reads what the UI sent.

test.beforeEach(async () => {
	await resetStub();
});

const MS_PER_DAY = 86_400_000;
const addDays = (date: string, n: number) =>
	new Date(Date.parse(`${date}T00:00:00Z`) + n * MS_PER_DAY).toISOString().slice(0, 10);
const pageUrl = (kid: string) => `/kids/${kid}/export`;
const pdfUrl = (kid: string) => `${pageUrl(kid)}/pdf`;
const MODULES_MESSAGE = 'Choose at least one module';
const BUSY_MESSAGE = 'The PDF cannot be made right now, try again in a moment';
const NOTE = 'The file is made when you download it and is not kept.';

const modulesGroup = (page: Page) => page.getByRole('group', { name: 'Modules to include' });
const offGroup = (page: Page) => page.getByRole('group', { name: 'Switched off for this child' });
const box = (page: Page, id: (typeof EXPORT_MODULES)[number]) =>
	modulesGroup(page).getByRole('checkbox', { name: MODULE_LABEL[id], exact: true });
const from = (page: Page) => page.getByLabel('From', { exact: true });
const to = (page: Page) => page.getByLabel('To', { exact: true });
const overflows = (page: Page) =>
	page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth);
const sentModules = (pdf: string) => /Modules: ([a-z,]*)\)/.exec(pdf)?.[1].split(',') ?? null;

async function download(page: Page): Promise<{ download: Download; pdf: string }> {
	const event = page.waitForEvent('download');
	await page.getByRole('button', { name: 'Download PDF' }).click();
	const d = await event;
	const file = await d.path();
	return { download: d, pdf: await readFile(file, 'latin1') };
}
async function switchOff(module: string, kid = IVY.id) {
	const ana = await stubAs(ANA);
	expect((await ana('PUT', `/v1/kids/${kid}/modules/${module}`, { enabled: false })).status).toBe(
		204
	);
}
const exportBodies = async (kid: string) =>
	(await receivedRequests())
		.filter((r) => r.method === 'POST' && r.path === `/v1/kids/${kid}/export`)
		.map((r) => JSON.parse(r.body ?? '{}'));

test('every module is listed; a switched-off one is unticked under its sub-heading; the last 30 days', async ({
	page
}) => {
	await switchOff('pumping');
	await signIn(page, ANA);
	await page.goto(pageUrl(IVY.id));
	await expect(
		page.getByRole('heading', { name: 'Export PDF', exact: true, level: 1 })
	).toBeVisible();
	await expect(page.getByText(NOTE)).toBeVisible();
	await expect(modulesGroup(page).getByRole('checkbox')).toHaveCount(EXPORT_MODULES.length);
	for (const id of EXPORT_MODULES) {
		if (id === 'pumping') await expect(box(page, id)).not.toBeChecked();
		else await expect(box(page, id)).toBeChecked();
	}
	await expect(page.getByRole('heading', { name: 'Switched off for this child' })).toBeVisible();
	await expect(offGroup(page).getByRole('checkbox')).toHaveCount(1);
	await expect(offGroup(page).getByRole('checkbox', { name: 'Pumping' })).not.toBeChecked();
	const today = todayIn(IVY.timeZone);
	await expect(from(page)).toHaveValue(addDays(today, 1 - EXPORT_DEFAULT_DAYS));
	await expect(to(page)).toHaveValue(today);
	await expect(to(page)).toHaveAttribute('max', today);
	await expect(from(page)).toHaveAttribute('min', IVY.birthDate);
	await expectAccessible(page);
});

test('the kid links lead to the page', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${IVY.id}`);
	await (await navLink(page, 'Export PDF')).click();
	await expect(
		page.getByRole('heading', { name: 'Export PDF', exact: true, level: 1 })
	).toBeVisible();
});

test('submitting downloads the file named by its dates, with exactly the ticked modules', async ({
	page
}) => {
	await signIn(page, ANA);
	await page.goto(pageUrl(IVY.id));
	const today = todayIn(IVY.timeZone);
	const { download: d, pdf } = await download(page);
	expect(d.suggestedFilename()).toBe(
		`happybabytracker-${addDays(today, 1 - EXPORT_DEFAULT_DAYS)}-${today}.pdf`
	);
	expect(pdf.startsWith('%PDF-1.')).toBe(true);
	expect(sentModules(pdf)).toEqual([...EXPORT_MODULES]);
	await expect(page).toHaveURL(new RegExp(`${pageUrl(IVY.id)}$`));
});

test('a direct POST streams the PDF with its headers, modules in a fixed order', async ({
	page
}) => {
	await signIn(page, ANA);
	const today = todayIn(IVY.timeZone);
	const form = new FormData();
	form.append('from', addDays(today, -6));
	form.append('to', today);
	form.append('modules', 'growth');
	form.append('modules', 'bottles');
	const response = await page.request.post(pdfUrl(IVY.id), {
		form,
		headers: { origin: UI_ORIGIN }
	});
	expect(response.status()).toBe(200);
	const headers = response.headers();
	expect(headers['content-type']).toBe('application/pdf');
	expect(headers['content-disposition']).toBe(
		`attachment; filename="happybabytracker-${addDays(today, -6)}-${today}.pdf"`
	);
	expect(headers['cache-control']).toBe('no-store');
	expect(headers['x-content-type-options']).toBe('nosniff');
	const body = (await response.body()).toString('latin1');
	expect(body.startsWith('%PDF-1.')).toBe(true);
	expect(sentModules(body)).toEqual(['bottles', 'growth']);
});

test('unticking every module shows the modules message and keeps the typed dates', async ({
	page
}) => {
	await signIn(page, ANA);
	await page.goto(pageUrl(IVY.id));
	await untilHydrated(box(page, 'bottles'), 'checked');
	await untilHydrated(from(page));
	const typed = addDays(todayIn(IVY.timeZone), -10);
	await from(page).fill(typed);
	for (const id of EXPORT_MODULES) await box(page, id).uncheck();
	await page.getByRole('button', { name: 'Download PDF' }).click();
	await page.waitForURL((u) => u.searchParams.get('error') === 'modules');
	await expect(page.getByRole('alert')).toHaveText(MODULES_MESSAGE);
	await expect(box(page, 'bottles')).toHaveAccessibleDescription(MODULES_MESSAGE);
	await expect(from(page)).toHaveValue(typed);
	for (const id of EXPORT_MODULES) await expect(box(page, id)).not.toBeChecked();
	expect(await exportBodies(IVY.id)).toEqual([]);
	await expectAccessible(page);
});

test('ticking the switched-off module sends it', async ({ page }) => {
	await switchOff('pumping');
	await signIn(page, ANA);
	await page.goto(pageUrl(IVY.id));
	await untilHydrated(box(page, 'bottles'), 'checked');
	await box(page, 'pumping').check();
	const { pdf } = await download(page);
	expect(sentModules(pdf)).toEqual([...EXPORT_MODULES]);
	const [sent] = await exportBodies(IVY.id);
	expect(sent.modules).toContain('pumping');
});

test('a direct POST with an unknown module lands back on the page with the message', async ({
	page
}) => {
	await signIn(page, ANA);
	const today = todayIn(IVY.timeZone);
	const response = await page.request.post(pdfUrl(IVY.id), {
		form: { from: today, to: today, modules: 'bogus' },
		headers: { origin: UI_ORIGIN }
	});
	expect(response.status()).toBe(200);
	const url = new URL(response.url());
	expect(url.pathname).toBe(pageUrl(IVY.id));
	expect(url.search).toBe(`?error=modules&from=${today}&to=${today}&modules=`);
	expect(await response.text()).toContain(MODULES_MESSAGE);
	expect(await exportBodies(IVY.id)).toEqual([]);
});

test('a busy API lands back on the page with the message and the same choices', async ({
	page
}) => {
	await switchOff('pumping');
	await setScenario('export_busy');
	await signIn(page, ANA);
	await page.goto(pageUrl(IVY.id));
	await page.getByRole('button', { name: 'Download PDF' }).click();
	await page.waitForURL((u) => u.searchParams.get('error') === 'busy');
	await expect(page.getByRole('alert')).toHaveText(BUSY_MESSAGE);
	await expect(box(page, 'bottles')).toBeChecked();
	await expect(box(page, 'pumping')).not.toBeChecked();
	await expect(to(page)).toHaveValue(todayIn(IVY.timeZone));
	await expectAccessible(page);
});

test('a caregiver can export', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(pageUrl(LEO.id));
	const today = todayIn(LEO.timeZone);
	const { download: d, pdf } = await download(page);
	expect(d.suggestedFilename()).toBe(
		`happybabytracker-${addDays(today, 1 - EXPORT_DEFAULT_DAYS)}-${today}.pdf`
	);
	expect(sentModules(pdf)?.length).toBeGreaterThan(0);
});

test('no grant gets the 404 page, on the page and on a direct POST; the admin is redirected', async ({
	page
}) => {
	await signIn(page, DAN);
	await expectNotFoundPage(page, pageUrl(IVY.id));
	const today = todayIn(IVY.timeZone);
	const post = () =>
		page.request.post(pdfUrl(IVY.id), {
			form: { from: today, to: today, modules: 'sleep' },
			headers: { origin: UI_ORIGIN }
		});
	const response = await post();
	expect(response.status()).toBe(404);
	expect(await response.text()).toContain('Not found');
	expect(await exportBodies(IVY.id)).toEqual([]);

	await page.context().clearCookies();
	await signIn(page, ADMIN);
	await page.goto(pageUrl(IVY.id));
	await expect(page).toHaveURL(/\/admin\/parents$/);
	expect(new URL((await post()).url()).pathname).toBe('/admin/parents');
});

test('no horizontal overflow at 390 px', async ({ page }) => {
	await switchOff('pumping');
	await page.setViewportSize({ width: 390, height: 844 });
	await signIn(page, ANA);
	await page.goto(`${pageUrl(IVY.id)}?error=modules&modules=`);
	await expect(page.getByRole('alert')).toBeVisible();
	expect(await overflows(page)).toBe(false);
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('downloads, and a refused form comes back with the message', async ({ page }) => {
		await signIn(page, ANA);
		await page.goto(pageUrl(IVY.id));
		const { pdf } = await download(page);
		expect(sentModules(pdf)).toEqual([...EXPORT_MODULES]);

		for (const id of EXPORT_MODULES) await box(page, id).uncheck();
		await page.getByRole('button', { name: 'Download PDF' }).click();
		await page.waitForURL((u) => u.searchParams.get('error') === 'modules');
		await expect(page.getByRole('alert')).toHaveText(MODULES_MESSAGE);
	});
});
