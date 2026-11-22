import { expect, test, type Page } from '@playwright/test';
import { todayIn } from '../src/lib/format.js';
import { formatDayHeading } from '../src/lib/timeline.js';
import { expectAccessible, expectNotFoundPage, signIn } from './helpers.js';
import { ANA, DAN, IVY, LEO, resetStub, stubAs } from './stub-api.js';

// The stub's scores are fake: a score is checked by its shape, never its number.

test.beforeEach(async () => {
	await resetStub();
});

type Api = Awaited<ReturnType<typeof stubAs>>;

const WHO = 'WHO Child Growth Standards (2006) and Growth Reference (2007)';
const CDC = 'CDC 2000 Growth Charts';
const MS_PER_DAY = 86_400_000;
const addDays = (date: string, n: number) =>
	new Date(Date.parse(`${date}T00:00:00Z`) + n * MS_PER_DAY).toISOString().slice(0, 10);

const growthUrl = (kid: string, measure?: string) =>
	`/kids/${kid}/growth${measure ? `?measure=${measure}` : ''}`;
const chart = (page: Page) => page.locator('svg[role="img"]');
const curveLines = (page: Page) => chart(page).locator('[data-line]');
const kidPoints = (page: Page) => chart(page).locator('circle[data-point]');
const rows = (page: Page) => page.getByRole('list', { name: 'Measurements' }).getByRole('listitem');
const measureLink = (page: Page, name: string) =>
	page.getByRole('navigation', { name: 'Measures' }).getByRole('link', { name, exact: true });
const overflows = (page: Page) =>
	page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth);

const kg = (amount: number) => ({ unit: 'kg', amount });
const cm = (amount: number) => ({ unit: 'cm', amount });
const MID = addDays(IVY.birthDate, 150);
const TODAY = todayIn(IVY.timeZone);

// Ivy (term, Ana guards her): at birth, at 150 days with a head circumference, and today.
async function seedIvy(api: Api, note: string | null = 'after the bath') {
	const post = async (body: Record<string, unknown>) => {
		const r = await api('POST', `/v1/kids/${IVY.id}/growth`, body);
		expect(r.status).toBe(201);
		return r.body as { id: string };
	};
	await post({ measured_on: IVY.birthDate, weight: kg(3300), length: cm(500) });
	await post({ measured_on: MID, weight: kg(6500), length: cm(640), head: cm(420) });
	return post({ measured_on: TODAY, weight: kg(9500), length: cm(760), note });
}

test('three measurements are listed newest first with scores over the source’s lines', async ({
	page
}) => {
	const latest = await seedIvy(await stubAs(ANA));
	await signIn(page, ANA);
	await page.goto(growthUrl(IVY.id));
	await expect(page.getByRole('heading', { name: 'Growth for Ivy', level: 1 })).toBeVisible();
	await expect(rows(page)).toHaveCount(3);
	await expect(rows(page)).toHaveText([
		new RegExp(formatDayHeading(TODAY)),
		new RegExp(formatDayHeading(MID)),
		new RegExp(formatDayHeading(IVY.birthDate))
	]);
	await expect(rows(page).first()).toContainText(/Weight 9\.5 kg: z -?\d+\.\d{2}, percentile/);
	await expect(rows(page).first()).toContainText(/Length 76 cm: z -?\d+\.\d{2}, percentile/);
	await expect(rows(page).first()).toContainText(/BMI [\d.]+: z -?\d+\.\d{2}, percentile/);
	await expect(rows(page).first()).toContainText('after the bath');
	await expect(rows(page).first().getByRole('link', { name: /Edit/ })).toHaveAttribute(
		'href',
		`/kids/${IVY.id}/growth/${latest.id}`
	);
	await expect(page.getByRole('link', { name: 'Add measurement' })).toHaveAttribute(
		'href',
		`/kids/${IVY.id}/growth/new`
	);
	await expect(measureLink(page, 'Weight')).toHaveAttribute('aria-current', 'page');
	await expect(curveLines(page)).toHaveCount(5);
	await expect(kidPoints(page)).toHaveCount(3);
	await expect(chart(page)).toHaveAccessibleName(/Weight/);
	await expect(chart(page).getByText('Age (months)', { exact: true })).toHaveCount(1);
	await expect(page.getByRole('link', { name: 'Growth', exact: true })).toHaveAttribute(
		'aria-current',
		'page'
	);
	await expectAccessible(page);
});

test('the measure links switch the chart and keep the list', async ({ page }) => {
	await seedIvy(await stubAs(ANA));
	await signIn(page, ANA);
	await page.goto(growthUrl(IVY.id));
	await measureLink(page, 'Length').click();
	await expect(page).toHaveURL(growthUrl(IVY.id, 'length'));
	await expect(measureLink(page, 'Length')).toHaveAttribute('aria-current', 'page');
	await expect(measureLink(page, 'Weight')).not.toHaveAttribute('aria-current', 'page');
	await expect(chart(page)).toHaveAccessibleName(/Length/);
	await expect(chart(page).getByText('Length (cm)', { exact: true })).toHaveCount(1);
	await expect(kidPoints(page)).toHaveCount(3);
	await expect(rows(page)).toHaveCount(3);

	await measureLink(page, 'Head circumference').click();
	await expect(page).toHaveURL(growthUrl(IVY.id, 'head'));
	await expect(kidPoints(page)).toHaveCount(1);
	await expect(rows(page)).toHaveCount(3);
	await expectAccessible(page);

	await measureLink(page, 'BMI').click();
	await expect(page).toHaveURL(growthUrl(IVY.id, 'bmi'));
	await expect(chart(page).getByText('BMI (kg/m²)', { exact: true })).toHaveCount(1);
	await expect(kidPoints(page)).toHaveCount(3);
});

test('switching the kid’s source changes the shown source and its curves', async ({ page }) => {
	const api = await stubAs(ANA);
	await seedIvy(api);
	await signIn(page, ANA);
	await page.goto(growthUrl(IVY.id));
	await expect(page.getByTestId('growth-source')).toContainText(WHO);
	await expect(curveLines(page)).toHaveCount(5);

	expect((await api('PATCH', `/v1/kids/${IVY.id}`, { growth_source: 'cdc' })).status).toBe(200);
	await page.goto(growthUrl(IVY.id));
	await expect(page.getByTestId('growth-source')).toContainText(CDC);
	await expect(curveLines(page)).toHaveCount(9);
	await expect(chart(page).getByText('97th', { exact: true })).toHaveCount(1);
	await expect(chart(page).getByText('5th', { exact: true })).toHaveCount(1);
	await expectAccessible(page);
});

test('a preterm kid shows both ages and is plotted at corrected age', async ({ page }) => {
	const api = await stubAs(ANA);
	await signIn(page, ANA);
	// Leo is past his second birthday: the axis is chronological, his earlier rows still corrected.
	await page.goto(growthUrl(LEO.id));
	await expect(rows(page)).toHaveCount(3);
	await expect(rows(page).nth(1)).toContainText(/\(corrected [^)]+\)/);
	await expect(rows(page).nth(2)).toContainText('(before expected term)');
	await expect(rows(page).nth(2)).toContainText('Weight 2.5 kg: before expected term');
	await expect(kidPoints(page)).toHaveCount(2);
	await expect(chart(page).getByText('Age (months)', { exact: true })).toHaveCount(1);

	const born = addDays(todayIn('UTC'), -60);
	const noa = await api('POST', '/v1/kids', {
		name: 'Noa',
		birth_date: born,
		sex: 'female',
		gestation_weeks: 30,
		gestation_days: 0,
		time_zone: 'UTC'
	});
	expect(noa.status).toBe(201);
	const r = await api('POST', `/v1/kids/${noa.body.id}/growth`, {
		measured_on: born,
		weight: kg(1400)
	});
	expect(r.status).toBe(201);
	await page.goto(growthUrl(noa.body.id));
	await expect(chart(page).getByText('Corrected age (months)', { exact: true })).toHaveCount(1);
	await expect(rows(page)).toHaveCount(1);
	await expect(rows(page).first()).toContainText('(before expected term)');
	await expect(kidPoints(page)).toHaveCount(0);
	await expectAccessible(page);
});

test('CDC has no BMI table under 24 months: the points without curves, and it says so', async ({
	page
}) => {
	const api = await stubAs(ANA);
	await seedIvy(api);
	await api('PATCH', `/v1/kids/${IVY.id}`, { growth_source: 'cdc' });
	await signIn(page, ANA);
	await page.goto(growthUrl(IVY.id, 'bmi'));
	await expect(page.getByText(`${CDC} has no table for this measure at this age`)).toBeVisible();
	await expect(curveLines(page)).toHaveCount(0);
	await expect(chart(page).getByText('50th', { exact: true })).toHaveCount(0);
	await expect(kidPoints(page)).toHaveCount(3);
	await expect(rows(page).first()).toContainText(
		/BMI [\d.]+: this source has no table for this age/
	);
	await expectAccessible(page);
});

test('WHO has no head circumference table after five years: early curves drawn, and it says so', async ({
	page
}) => {
	const api = await stubAs(ANA);
	const today = todayIn('UTC');
	const born = addDays(today, -7 * 365 - 2);
	const kid = await api('POST', '/v1/kids', {
		name: 'Oli',
		birth_date: born,
		sex: 'male',
		time_zone: 'UTC'
	});
	expect(kid.status).toBe(201);
	for (const [on, mm] of [
		[addDays(born, 3 * 365), 490],
		[today, 510]
	] as const) {
		const r = await api('POST', `/v1/kids/${kid.body.id}/growth`, {
			measured_on: on,
			head: cm(mm)
		});
		expect(r.status).toBe(201);
	}
	await signIn(page, ANA);
	await page.goto(growthUrl(kid.body.id, 'head'));
	await expect(page.getByText(`${WHO} has no table for this measure at this age`)).toBeVisible();
	await expect(curveLines(page)).toHaveCount(5);
	await expect(kidPoints(page)).toHaveCount(2);
	await expect(rows(page).first()).toContainText(
		/Head circumference 51 cm: this source has no table for this age/
	);
	await expect(rows(page).nth(1)).toContainText(/Head circumference 49 cm: z -?\d+\.\d{2}/);
	await expectAccessible(page);

	await page.goto(growthUrl(kid.body.id, 'weight'));
	await expect(page.getByText(/has no table for this measure/)).toHaveCount(0);
});

test('an imperial parent sees pounds and ounces and inches', async ({ page }) => {
	const api = await stubAs(ANA);
	await seedIvy(api);
	expect((await api('PATCH', '/v1/me', { units: { weight: 'lb', length: 'in' } })).status).toBe(
		200
	);
	await signIn(page, ANA);
	await page.goto(growthUrl(IVY.id));
	await expect(rows(page).first()).toContainText(/Weight 20 lb 15\.1 oz/);
	await expect(rows(page).first()).toContainText(/Length 29\.92 in/);
	await expect(chart(page).getByText('Weight (lb)', { exact: true })).toHaveCount(1);
	await page.goto(growthUrl(IVY.id, 'length'));
	await expect(chart(page).getByText('Length (in)', { exact: true })).toHaveCount(1);
});

test('the attribution block names the source and its licence; guardians get a settings link', async ({
	page
}) => {
	await signIn(page, ANA);
	await page.goto(growthUrl(IVY.id));
	const about = page.getByRole('region', { name: 'About the reference data' });
	await expect(about).toContainText('Stub attribution: World Health Organization');
	await expect(about).toContainText('Licence: CC BY-NC-SA 3.0 IGO');
	const settings = page.getByRole('link', { name: /Change the growth source/ });
	await expect
		.poll(() => settings.evaluate((a) => new URL((a as HTMLAnchorElement).href).pathname))
		.toBe(`/kids/${IVY.id}/settings`);
	await expect(rows(page)).toHaveCount(0);
	await expect(page.getByText('No measurements yet')).toBeVisible();
	await expectAccessible(page);

	// Ana is Leo's caregiver: the source is named, the settings link is not offered.
	await page.goto(growthUrl(LEO.id));
	await expect(page.getByTestId('growth-source')).toContainText(WHO);
	await expect(page.getByRole('link', { name: /Change the growth source/ })).toHaveCount(0);
});

test('the page stays with the growth module switched off', async ({ page }) => {
	const api = await stubAs(ANA);
	expect((await api('PUT', `/v1/kids/${IVY.id}/modules/growth`, { enabled: false })).status).toBe(
		204
	);
	await signIn(page, ANA);
	const response = await page.goto(growthUrl(IVY.id));
	expect(response?.status()).toBe(200);
	expect(response?.headers()['cache-control']).toBe('no-store');
	await expect(page.getByRole('heading', { name: 'Growth for Ivy', level: 1 })).toBeVisible();
});

test('an unknown measure falls back to weight; an unknown kid is the 404 page', async ({
	page
}) => {
	await signIn(page, ANA);
	await page.goto(growthUrl(IVY.id, 'shoe'));
	await expect(measureLink(page, 'Weight')).toHaveAttribute('aria-current', 'page');
	await expectNotFoundPage(page, growthUrl('k-nobody'));
	await page.context().clearCookies();
	await signIn(page, DAN);
	await expectNotFoundPage(page, growthUrl(IVY.id));
});

test('a long unbroken note wraps on a phone', async ({ page }, testInfo) => {
	test.skip(testInfo.project.name !== 'phone', 'the overflow is a phone-width check');
	await seedIvy(await stubAs(ANA), 'w'.repeat(200));
	await signIn(page, ANA);
	await page.goto(growthUrl(IVY.id));
	await expect(rows(page)).toHaveCount(3);
	await page.getByText('Chart data', { exact: true }).click();
	expect(await overflows(page)).toBe(false);
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('the chart, its data table and the measure links work as plain HTML', async ({ page }) => {
		await seedIvy(await stubAs(ANA));
		await signIn(page, ANA);
		await page.goto(growthUrl(IVY.id));
		await expect(curveLines(page)).toHaveCount(5);
		await expect(kidPoints(page)).toHaveCount(3);
		await page.getByText('Chart data', { exact: true }).click();
		const data = page.getByRole('table', { name: 'Percentile values' });
		await expect(data.getByRole('columnheader', { name: '97th' })).toBeVisible();
		await expect(
			page.getByRole('table', { name: 'Measurements plotted' }).getByRole('row')
		).toHaveCount(4);
		await measureLink(page, 'Length').click();
		await expect(page).toHaveURL(growthUrl(IVY.id, 'length'));
		await expect(measureLink(page, 'Length')).toHaveAttribute('aria-current', 'page');
	});
});
