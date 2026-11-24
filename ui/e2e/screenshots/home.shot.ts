import type { Page } from '@playwright/test';
import { submit } from '../helpers.js';
import {
	ANA,
	CLEO,
	DAN,
	IVY,
	LEO,
	MIA,
	NELL,
	OTTO,
	PIA,
	REMY,
	SAMI,
	moveStubClock,
	setScenario
} from '../stub-api.js';
import { as, at, kidPath, must, seedDay, type Api } from './seed.js';
import { expect, minutesAgo, openDetails, shot, signInAs, test } from './shots.js';

const kidHome = (id: string) => `/kids/${id}`;

async function open(page: Page, path: string) {
	await page.goto(path);
	await page.waitForLoadState('networkidle');
}

// Courses made 3.5 hours before the frozen instant, so their morning doses are already due or late.
export async function seedDueMedication(api: Api, kid: string) {
	await moveStubClock(-3.5 * 3600);
	const course = (body: Record<string, unknown>) =>
		must(api, 'POST', kidPath(kid, 'medications'), body);
	await course({
		name: 'Iron drops',
		dose: '0.5 ml',
		schedule: { kind: 'times', times: ['08:00', '20:00'] }
	});
	await course({
		name: 'Vitamin D',
		dose: '1 drop',
		schedule: { kind: 'times', times: ['10:00'] }
	});
	await moveStubClock(0);
}

test('shell: no kids and many kids', async ({ page }) => {
	await signInAs(page, DAN);
	await expect(page.getByRole('heading', { name: 'Welcome, Dan' })).toBeVisible();
	await shot(
		page,
		'shell',
		'01-no-kids-welcome',
		'A parent with no child yet: the welcome and the main navigation'
	);
	await signInAs(page, CLEO);
	await expect(page.getByRole('heading', { name: LEO.name, level: 1 })).toBeVisible();
	await shot(
		page,
		'shell',
		'02-six-kids-switcher',
		'A guardian of six children: the kid switcher wraps (24+ band kid home)'
	);
});

test('young kid home', async ({ page }) => {
	const ana = await as(ANA);
	await signInAs(page, ANA);
	await expect(page.getByRole('heading', { name: MIA.name, level: 1 })).toBeVisible();
	await shot(
		page,
		'home',
		'01-mia-nothing-logged',
		'A 4-month-old with nothing logged: quick-log tiles only'
	);

	await seedDay(ana, MIA.id);
	await must(ana, 'POST', kidPath(MIA.id, 'nursing'), { started_at: minutesAgo(12), side: 'left' });
	await must(ana, 'POST', kidPath(MIA.id, 'sleep'), { started_at: minutesAgo(3) });
	await must(ana, 'PATCH', kidPath(MIA.id), { vaccine_schema: 'us' });
	await seedDueMedication(ana, MIA.id);
	await open(page, kidHome(MIA.id));
	await shot(
		page,
		'home',
		'02-mia-busy-day',
		'Running timers, the Due list (medication then vaccines) and Today totals'
	);
	await must(ana, 'PATCH', '/v1/me', {
		units: { weight: 'lb', length: 'in', volume: 'floz', temp: 'f' }
	});
	await open(page, kidHome(MIA.id));
	await shot(
		page,
		'home',
		'03-mia-busy-day-imperial',
		'The same day for a parent with imperial units'
	);
});

test('timer overrun and newborn', async ({ page }) => {
	const ana = await as(ANA);
	await must(ana, 'POST', kidPath(MIA.id, 'sleep'), { started_at: minutesAgo(26 * 60) });
	await signInAs(page, ANA);
	await submit(page.getByRole('button', { name: 'End sleep' }));
	await expect(page.getByRole('alert')).toContainText('ran over');
	await shot(
		page,
		'home',
		'04-sleep-timer-overrun',
		'Ending a sleep timer that ran over 24 hours links to its end time'
	);

	const noa = await must(ana, 'POST', '/v1/kids', {
		name: 'Noa',
		birth_date: at('00:00', 10).slice(0, 10),
		sex: 'female',
		time_zone: 'UTC'
	});
	await open(page, kidHome(noa.id));
	await shot(page, 'home', '05-newborn', 'A 10-day-old: age in days, under-12-month quick log');
});

test('older bands and due lists', async ({ page }) => {
	const ana = await as(ANA);
	await must(ana, 'PATCH', kidPath(IVY.id), { vaccine_schema: 'us' });
	await must(ana, 'POST', kidPath(IVY.id, 'meals'), { at: at('08:00'), what: 'Porridge' });
	await must(ana, 'POST', kidPath(IVY.id, 'diapers'), { at: at('08:30'), kind: 'wet' });
	await signInAs(page, ANA);
	await open(page, kidHome(IVY.id));
	await shot(
		page,
		'home',
		'06-ivy-13-months',
		'A 13-month-old: Feeding leads the quick log; vaccine visits due'
	);
	await setScenario('home_due_many');
	await open(page, kidHome(IVY.id));
	await shot(
		page,
		'home',
		'07-due-many',
		'More due than fit: three lines and a link per hidden kind'
	);
});

test('caregiver home and leaving', async ({ page }) => {
	await signInAs(page, ANA);
	await open(page, kidHome(LEO.id));
	await shot(
		page,
		'home',
		'08-leo-caregiver',
		'A caregiver on a 2-year-old: no settings, access or modules links; Leave'
	);
	await openDetails(page, 'Leave this child');
	await shot(
		page,
		'home',
		'09-leo-caregiver-leave-confirm',
		'Leave this child opened: the confirm step'
	);
	await setScenario('leave_unavailable');
	await submit(page.getByRole('button', { name: `Yes, leave ${LEO.name}` }));
	await expect(page.getByRole('alert')).toBeVisible();
	await shot(
		page,
		'home',
		'10-leo-leave-refused',
		'Leaving fails (the API is unreachable): the message in the opened step'
	);
});

test('milestone-age kids', async ({ page }) => {
	await signInAs(page, CLEO);
	const kids: [{ id: string }, string, string][] = [
		[
			LEO,
			'11-leo-guardian-24-plus',
			'A guardian on a 2-year-old born at 35 weeks: older-band tiles, every kid link'
		],
		[NELL, '12-nell-9-months', 'A 9-month-old'],
		[
			REMY,
			'13-remy-preterm-corrected-age',
			'Born at 32 weeks, 3 months old: the corrected age beside the age'
		],
		[OTTO, '14-otto-4-years-11-months', 'A kid at 4 years 11 months'],
		[PIA, '15-pia-5th-birthday', 'A kid on the 5th birthday'],
		[
			SAMI,
			'16-sami-30-months-milestones-off',
			'A 30-month-old with the milestones module off: no Milestones tile'
		]
	];
	for (const [kid, name, about] of kids) {
		await open(page, kidHome(kid.id));
		await shot(page, 'home', name, about);
	}
});
