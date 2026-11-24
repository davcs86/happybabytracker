import type { Page } from '@playwright/test';
import { submit } from '../helpers.js';
import { ANA, CLEO, LEO, MIA, moveStubClock } from '../stub-api.js';
import { as, at, kidPath, must, type Api } from './seed.js';
import { expect, openDetails, shot, signInAs, test } from './shots.js';

const url = (kid: string, rest = '') => `/kids/${kid}/medication${rest}`;
const field = (page: Page, label: string) => page.getByLabel(label, { exact: true });
const save = (page: Page) => submit(page.getByRole('button', { name: 'Save', exact: true }));

async function open(page: Page, path: string) {
	await page.goto(path);
	await page.waitForLoadState('networkidle');
}

// Courses made two days before the frozen instant, so they have past doses and a next due.
async function seedCourses(api: Api, kid: string) {
	const course = (body: Record<string, unknown>) =>
		must(api, 'POST', kidPath(kid, 'medications'), { dose: '2.5 ml', ...body });
	const dose = (medication: string, body: Record<string, unknown>) =>
		must(api, 'POST', kidPath(kid, 'medication-doses'), { medication, ...body });
	await moveStubClock(-2 * 86_400);
	const calpol = await course({
		name: 'Calpol',
		schedule: { kind: 'times', times: ['08:00', '20:00'] },
		note: 'With food'
	});
	const amoxil = await course({
		name: 'Amoxil',
		dose: '5 ml',
		schedule: { kind: 'every', hours: 8 },
		start_at: at('06:00', 2),
		end_on: at('00:00', -5).slice(0, 10)
	});
	const vitamin = await course({ name: 'Vitamin D', dose: '1 drop', schedule: { kind: 'none' } });
	await course({
		name: 'Paused cream',
		dose: 'Thin layer',
		schedule: { kind: 'none' },
		active: false
	});
	await course({
		name: 'Old antibiotic',
		dose: '4 ml',
		schedule: { kind: 'every', hours: 12 },
		start_at: at('07:00', 2),
		end_on: at('00:00', 1).slice(0, 10)
	});
	await moveStubClock(0);
	await dose(calpol.id, { given_at: at('20:05', 1), amount: '2.5 ml' });
	const morning = await dose(calpol.id, { given_at: at('08:10'), note: 'Spat a little out' });
	await dose(amoxil.id, { given_at: at('06:00') });
	await dose(vitamin.id, { given_at: at('09:00') });
	return { calpol, amoxil, vitamin, morning };
}

test('course list and doses', async ({ page }) => {
	const ana = await as(ANA);
	await signInAs(page, ANA);
	await open(page, url(MIA.id));
	await shot(page, 'medication', '01-no-courses', 'No course yet');
	const rows = await seedCourses(ana, MIA.id);
	await open(page, url(MIA.id));
	await shot(
		page,
		'medication',
		'02-courses',
		'Fixed times, every N hours and as needed; paused and ended courses; recent doses'
	);
	await openDetails(page, /ended|Ended/);
	await shot(page, 'medication', '03-ended-courses-open', 'The ended course behind its details');
	await open(page, url(MIA.id, `?medication=${rows.calpol.id}`));
	await shot(
		page,
		'medication',
		'04-doses-filtered-by-course',
		'Recent doses filtered to one course'
	);
	await open(page, url(MIA.id));
	const card = page
		.getByRole('listitem')
		.filter({ has: page.getByRole('heading', { name: 'Vitamin D', exact: true }) });
	await card.getByLabel('Amount (optional)').fill('2 drops');
	await submit(card.getByRole('button', { name: 'Log dose now' }));
	await shot(
		page,
		'medication',
		'05-dose-logged-now',
		'Log dose now: the course shows the dose just logged'
	);
	await open(page, url(MIA.id, `/doses/${rows.morning.id}`));
	await shot(page, 'medication', '06-dose-edit', 'Edit a dose: time, amount and note');
	await field(page, 'Amount').fill('a'.repeat(81));
	await save(page);
	await expect(page.getByRole('alert').first()).toBeVisible();
	await shot(
		page,
		'medication',
		'07-dose-edit-error',
		'Refused: an 81-character amount, the typed values kept'
	);
	await open(page, url(MIA.id, `/doses/${rows.morning.id}`));
	await openDetails(page, 'Delete this dose');
	await shot(page, 'medication', '08-dose-delete-confirm', 'Delete dose opened: the confirm step');
	await open(page, url(MIA.id, `/${rows.calpol.id}`));
	await shot(page, 'medication', '09-course-edit', 'Edit course: fixed times 08:00 and 20:00');
	await openDetails(page, 'Delete this course');
	await shot(
		page,
		'medication',
		'10-course-delete-confirm',
		'Delete course opened: says how many doses go with it'
	);
	await open(page, url(MIA.id, `/${rows.amoxil.id}`));
	await shot(
		page,
		'medication',
		'11-course-edit-every',
		'Edit an every-8-hours course with a last day'
	);
});

test('new course form', async ({ page }) => {
	await signInAs(page, ANA);
	await open(page, url(MIA.id, '/new'));
	await shot(page, 'medication', '12-new-course-empty', 'New course: as needed by default');
	await save(page);
	await expect(page.getByRole('alert').first()).toBeVisible();
	await shot(page, 'medication', '13-new-course-errors', 'Refused: name and dose required');
	await field(page, 'Name').fill('Ibuprofen');
	await field(page, 'Dose').fill('2.5 ml');
	await page.getByRole('radio', { name: 'At fixed times' }).check();
	await field(page, 'Time 1').fill('08:00');
	await field(page, 'Time 2').fill('14:00');
	await field(page, 'Time 3').fill('20:00');
	await shot(
		page,
		'medication',
		'14-new-course-fixed-times',
		'At fixed times: the time slots appear'
	);
	await page.getByRole('radio', { name: 'Every few hours' }).check();
	await field(page, 'Hours between doses').fill('6');
	await shot(
		page,
		'medication',
		'15-new-course-every-hours',
		'Every few hours: the interval and the start'
	);
});

test('module off and caregiver', async ({ page }) => {
	const ana = await as(ANA);
	await seedCourses(ana, MIA.id);
	await must(ana, 'PUT', kidPath(MIA.id, 'modules/medication'), { enabled: false });
	await signInAs(page, ANA);
	await open(page, url(MIA.id));
	await shot(
		page,
		'medication',
		'16-module-off-guardian',
		'Medication switched off: a note with a link to Modules; courses still listed'
	);
	const cleo = await as(CLEO);
	await must(cleo, 'POST', kidPath(LEO.id, 'medications'), {
		name: 'Antihistamine'.repeat(6).slice(0, 80),
		dose: '5 ml',
		schedule: { kind: 'none' }
	});
	await must(cleo, 'PUT', kidPath(LEO.id, 'modules/medication'), { enabled: false });
	await open(page, url(LEO.id));
	await shot(
		page,
		'medication',
		'17-module-off-caregiver-long-name',
		'A caregiver with the module off: a note without a link; an 80-character name wraps'
	);
});
