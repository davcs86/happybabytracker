import type { Page } from '@playwright/test';
import { submit, untilHydrated } from '../helpers.js';
import { ANA, IVY, LEO, MIA, REMY, CLEO } from '../stub-api.js';
import { as, at, kidPath, must, type Api } from './seed.js';
import { expect, openDetails, shot, signInAs, test } from './shots.js';

const TODAY = at('00:00').slice(0, 10);
const daysAgo = (n: number) => at('00:00', n).slice(0, 10);
const page_ = (kid: string, measure = '') =>
	`/kids/${kid}/growth${measure ? `?measure=${measure}` : ''}`;
const field = (page: Page, label: string) => page.getByLabel(label, { exact: true });
const save = (page: Page) => submit(page.getByRole('button', { name: 'Save', exact: true }));
const kg = (amount: number) => ({ unit: 'kg', amount });
const cm = (amount: number) => ({ unit: 'cm', amount });

async function open(page: Page, path: string) {
	await page.goto(path);
	await page.waitForLoadState('networkidle');
}

// Ivy (13 months, WHO): at birth, at 150 days with a head circumference, and today.
async function seedIvy(api: Api) {
	const post = (body: Record<string, unknown>) =>
		must(api, 'POST', kidPath(IVY.id, 'growth'), body);
	const mid = new Date(Date.parse(`${IVY.birthDate}T00:00:00Z`) + 150 * 86_400_000)
		.toISOString()
		.slice(0, 10);
	await post({ measured_on: IVY.birthDate, weight: kg(3300), length: cm(500), head: cm(345) });
	await post({ measured_on: mid, weight: kg(6500), length: cm(640), head: cm(420) });
	return post({ measured_on: TODAY, weight: kg(9500), length: cm(760), note: 'After the bath' });
}

test('a term kid: measures, sources, units', async ({ page }) => {
	const ana = await as(ANA);
	const latest = await seedIvy(ana);
	await signInAs(page, ANA);
	await open(page, page_(IVY.id));
	await shot(
		page,
		'growth',
		'01-ivy-weight-who',
		'Three measurements over the WHO weight lines, scored newest first'
	);
	for (const [n, measure, about] of [
		['02', 'length', 'Length or height chart'],
		['03', 'head', 'Head circumference chart'],
		['04', 'bmi', 'BMI chart']
	] as const) {
		await open(page, page_(IVY.id, measure));
		await shot(page, 'growth', `${n}-ivy-${measure}-who`, about);
	}
	await open(page, page_(IVY.id));
	await openDetails(page, 'Chart data');
	await shot(
		page,
		'growth',
		'20-ivy-chart-data-table',
		'Chart data opened: the percentile lines as a table'
	);
	await must(ana, 'PATCH', kidPath(IVY.id), { growth_source: 'cdc' });
	await open(page, page_(IVY.id));
	await shot(page, 'growth', '05-ivy-weight-cdc', 'The CDC source: its lines and attribution');
	await open(page, page_(IVY.id, 'bmi'));
	await shot(
		page,
		'growth',
		'06-ivy-bmi-cdc-no-table',
		'CDC has no BMI table under 24 months: points only, rows unscored'
	);
	await must(ana, 'PATCH', kidPath(IVY.id), { growth_source: 'who' });
	await must(ana, 'PATCH', '/v1/me', { units: { weight: 'lb', length: 'in' } });
	await open(page, page_(IVY.id));
	await shot(
		page,
		'growth',
		'07-ivy-imperial',
		'Pounds and ounces and inches for an imperial parent'
	);
	await open(page, `/kids/${IVY.id}/growth/${latest.id}`);
	await shot(
		page,
		'growth',
		'08-edit-imperial',
		'Edit a measurement for an imperial parent: it keeps the unit it was typed in (kg and cm); the empty head box shows inches'
	);
	await must(ana, 'PATCH', '/v1/me', { units: { weight: 'kg', length: 'cm' } });
	await open(page, `/kids/${IVY.id}/growth/${latest.id}`);
	await shot(page, 'growth', '09-edit', 'Edit measurement: 9.5 kg, 76 cm, a note');
	await openDetails(page, 'Delete this measurement');
	await shot(page, 'growth', '10-edit-delete-confirm', 'Delete opened: the confirm step');
});

test('empty, new and errors', async ({ page }) => {
	await signInAs(page, ANA);
	await open(page, page_(MIA.id));
	await shot(page, 'growth', '11-empty', 'No measurement yet: the WHO lines alone');
	await open(page, `/kids/${MIA.id}/growth/new`);
	await shot(page, 'growth', '12-new-empty', 'Add a measurement: date prefilled to today');
	await untilHydrated(field(page, 'Date'));
	await save(page);
	await expect(page.getByRole('alert').first()).toBeVisible();
	await shot(page, 'growth', '13-new-nothing-measured', 'Refused: no measurement typed');
	await field(page, 'Date').fill(daysAgo(-1));
	await field(page, 'Kilograms').fill('7.1');
	await field(page, 'Note').fill('Kept after the error');
	await save(page);
	await expect(field(page, 'Date')).toHaveAttribute('aria-invalid', 'true');
	await shot(page, 'growth', '14-new-future-date', 'Refused: a future date, the values kept');
	await field(page, 'Date').fill(TODAY);
	await field(page, 'Length or height').fill('64');
	await field(page, 'Head circumference').fill('41.5');
	await shot(
		page,
		'growth',
		'15-new-filled',
		'Weight, length and head typed and the date corrected; the earlier refusal stays on screen until the next save'
	);
});

test('preterm and unscored', async ({ page }) => {
	const ana = await as(ANA);
	const early = await must(ana, 'POST', '/v1/kids', {
		name: 'Uma',
		birth_date: daysAgo(14),
		sex: 'female',
		gestation_weeks: 30,
		gestation_days: 0,
		time_zone: 'UTC'
	});
	await must(ana, 'POST', kidPath(early.id, 'growth'), {
		measured_on: daysAgo(14),
		weight: kg(1400)
	});
	await must(ana, 'POST', kidPath(early.id, 'growth'), { measured_on: TODAY, weight: kg(1650) });
	const oli = await must(ana, 'POST', '/v1/kids', {
		name: 'Oli',
		birth_date: daysAgo(7 * 365 + 2),
		sex: 'male',
		time_zone: 'UTC'
	});
	await must(ana, 'POST', kidPath(oli.id, 'growth'), {
		measured_on: daysAgo(4 * 365),
		head: cm(490)
	});
	await must(ana, 'POST', kidPath(oli.id, 'growth'), {
		measured_on: TODAY,
		head: cm(510),
		note: 'Measured at school'.repeat(11).replace(/ /g, '').slice(0, 200)
	});
	const cleo = await as(CLEO);
	await must(cleo, 'POST', kidPath(REMY.id, 'growth'), {
		measured_on: TODAY,
		weight: kg(5200),
		length: cm(590)
	});
	await signInAs(page, ANA);
	await open(page, page_(LEO.id));
	await shot(
		page,
		'growth',
		'16-leo-preterm-caregiver',
		'Born at 35 weeks: both ages, plotted at corrected age, seen by a caregiver'
	);
	await open(page, page_(early.id));
	await shot(
		page,
		'growth',
		'17-before-expected-term',
		'Born at 30 weeks two weeks ago: before expected term'
	);
	await open(page, page_(oli.id, 'head'));
	await shot(
		page,
		'growth',
		'18-who-head-after-5-unscored',
		'WHO has no head table after five years: the latest row is unscored; a long note wraps'
	);
	await signInAs(page, CLEO);
	await open(page, page_(REMY.id));
	await shot(
		page,
		'growth',
		'19-remy-preterm-guardian',
		'Born at 32 weeks, 3 months: corrected age for a guardian'
	);
});
