import type { Page } from '@playwright/test';
import { ANA, CLEO, LEO, MIA } from '../stub-api.js';
import { as, at, kidPath, must, seedDay } from './seed.js';
import { shot, signInAs, test } from './shots.js';

const YESTERDAY = at('00:00', 1).slice(0, 10);
const TODAY = at('00:00').slice(0, 10);
const url = (kid: string, query = '') => `/kids/${kid}/timeline${query}`;
const both = `?from=${YESTERDAY}&to=${TODAY}`;

async function open(page: Page, path: string) {
	await page.goto(path);
	await page.waitForLoadState('networkidle');
}

test('timeline', async ({ page }) => {
	const ana = await as(ANA);
	await signInAs(page, ANA);
	await open(page, url(MIA.id));
	await shot(
		page,
		'timeline',
		'01-empty-today',
		'Nothing logged in the default range (the kid’s today)'
	);
	await seedDay(ana, MIA.id);
	await open(page, url(MIA.id));
	await shot(
		page,
		'timeline',
		'02-today',
		'Today: every tracker ticked, the day totals, newest first'
	);
	await open(page, url(MIA.id, both));
	await shot(
		page,
		'timeline',
		'03-two-days',
		'Two days under their headings; the overnight sleep marks its end'
	);
	await open(page, url(MIA.id, `${both}&types=symptom`));
	await shot(
		page,
		'timeline',
		'04-symptoms-only',
		'Narrowed to symptoms: fever with temperature, cough, other'
	);
	await open(page, url(MIA.id, `${both}&types=bottle&types=nursing&types=pumping`));
	await shot(page, 'timeline', '05-feeding-only', 'Narrowed to bottles, nursing and pumping');
	await open(page, url(MIA.id, `?from=${TODAY}&to=${YESTERDAY}`));
	await shot(page, 'timeline', '06-reversed-range', 'A reversed range: the message on the field');
	await open(page, url(MIA.id, '?cursor=not-a-cursor'));
	await shot(
		page,
		'timeline',
		'07-invalid-cursor',
		'A broken Older link: a message and a way back'
	);
	await must(ana, 'PATCH', '/v1/me', {
		units: { weight: 'lb', length: 'in', volume: 'floz', temp: 'f' }
	});
	await open(page, url(MIA.id, both));
	await shot(page, 'timeline', '08-two-days-imperial', 'The same two days in imperial units');
});

test('paging and long text', async ({ page }) => {
	const ana = await as(ANA);
	for (let i = 0; i < 55; i++) {
		const minute = String(i).padStart(2, '0');
		await must(ana, 'POST', kidPath(MIA.id, 'diapers'), { at: at(`09:${minute}`), kind: 'wet' });
	}
	await must(ana, 'POST', kidPath(MIA.id, 'meals'), {
		at: at('10:05'),
		what: 'Sweetpotato'.repeat(18).slice(0, 200),
		note: 'Mashed'.repeat(50).slice(0, 200)
	});
	await signInAs(page, ANA);
	await open(page, url(MIA.id));
	await shot(
		page,
		'timeline',
		'09-first-page-older',
		'A full first page (long), with Older at the end; a 200-character meal wraps'
	);
	await page.getByRole('link', { name: 'Older' }).click();
	await page.waitForURL(/cursor=/);
	await shot(page, 'timeline', '10-second-page', 'The second page: Back to newest');
});

test('caregiver and module off', async ({ page }) => {
	const cleo = await as(CLEO);
	await must(cleo, 'POST', kidPath(LEO.id, 'symptoms'), {
		at: at('07:00'),
		symptom: 'rash',
		severity: 'mild',
		note: 'On both arms'
	});
	await must(cleo, 'POST', kidPath(LEO.id, 'meals'), { at: at('08:00'), what: 'Toast and banana' });
	await must(cleo, 'POST', kidPath(LEO.id, 'diapers'), { at: at('09:00'), kind: 'dirty' });
	await signInAs(page, ANA);
	await open(page, url(LEO.id));
	await shot(
		page,
		'timeline',
		'11-caregiver',
		'A caregiver reads a 2-year-old’s timeline (kid zone Europe/Madrid)'
	);
	await must(cleo, 'PUT', kidPath(LEO.id, 'modules/diapers'), { enabled: false });
	await open(page, url(LEO.id));
	await shot(
		page,
		'timeline',
		'12-module-off-unticked',
		'Diapers switched off: unticked, its entries hidden'
	);
});
