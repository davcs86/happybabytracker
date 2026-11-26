import { expect, test, type Locator, type Page } from '@playwright/test';
import { formatAge, todayIn } from '../src/lib/format.js';
import {
	KID_COOKIE,
	expectAccessible,
	expectFieldError,
	expectLanding,
	expectNotFoundPage,
	signIn,
	submit,
	untilHydrated
} from './helpers.js';
import {
	ANA,
	CLEO,
	DAN,
	IVY,
	LEO,
	MIA,
	STUB_TOKEN,
	STUB_URL,
	UI_ORIGIN,
	openSession,
	receivedRequests,
	resetStub,
	storeVaccineSchema,
	stubAs
} from './stub-api.js';

test.beforeEach(async () => {
	await resetStub();
});

const ageOf = (kid: { birthDate: string; timeZone: string }) =>
	formatAge(kid.birthDate, todayIn(kid.timeZone));

// The header switcher opens a sheet; a click before hydration is the Choose a child page, so it retries.
async function openSwitcher(page: Page, current: string): Promise<Locator> {
	const sheet = page.getByRole('dialog', { name: 'Switch child' });
	await expect(async () => {
		await page.getByRole('link', { name: `Child: ${current}. Switch child` }).click();
		await expect(sheet).toBeVisible({ timeout: 1000 });
	}).toPass();
	return sheet;
}

// The More entries sit in the More sheet on a phone and in the side navigation from 768 px.
async function moreEntries(page: Page): Promise<Locator> {
	const more = page.getByRole('button', { name: 'More', exact: true });
	if (!(await more.isVisible())) return page.getByRole('navigation', { name: 'Primary' });
	const sheet = page.getByRole('dialog', { name: 'More' });
	await expect(async () => {
		await more.click();
		await expect(sheet).toBeVisible({ timeout: 1000 });
	}).toPass();
	return sheet;
}
const entry = (entries: Locator, name: string) =>
	entries.getByRole('link', { name: new RegExp(`^\\s*${name}`) });

type KidForm = {
	name?: string;
	birthDate?: string;
	sex?: 'Female' | 'Male';
	early?: { weeks: string; days: string };
	timeZone?: string;
};

const schedule = (page: Page) => page.getByLabel('Vaccination schedule', { exact: true });
const optionValues = (page: Page) =>
	schedule(page)
		.locator('option')
		.evaluateAll((o) => o.map((x) => (x as HTMLOptionElement).value));
const lastPatch = async (kid: string) => {
	const writes = (await receivedRequests()).filter(
		(r) => r.method === 'PATCH' && r.path === `/v1/kids/${kid}`
	);
	return JSON.parse(writes.at(-1)?.body ?? 'null') as Record<string, unknown>;
};
const storedSchedule = async (kid: string) =>
	(await (await stubAs(ANA))('GET', `/v1/kids/${kid}`)).body.vaccine_schema;
const overdue = (page: Page) => page.getByLabel('Show a missed dose as overdue for (minutes)');
const OVERDUE_HINT =
	'From 5 to 1440 (24 hours). A missed dose leaves the home list after this time, or when the next dose of that course is due, whichever comes first.';
const patches = async (kid: string) =>
	(await receivedRequests()).filter((r) => r.method === 'PATCH' && r.path === `/v1/kids/${kid}`);
const storedOverdue = async (kid: string) =>
	(await (await stubAs(ANA))('GET', `/v1/kids/${kid}`)).body.reminder_overdue_minutes;
const NO_SCHEDULE = 'No vaccination schedule chosen yet.';
const CAPTION = 'Schedule: Stub authority, Stub edition';

async function fillKid(page: Page, kid: KidForm) {
	if (kid.name !== undefined) await page.getByLabel('Name', { exact: true }).fill(kid.name);
	if (kid.birthDate !== undefined) await page.getByLabel('Birth date').fill(kid.birthDate);
	if (kid.sex) await page.getByRole('radio', { name: kid.sex, exact: true }).check();
	if (kid.early) {
		await page.getByRole('checkbox', { name: 'Born early?' }).check();
		await page.getByLabel('Weeks').fill(kid.early.weeks);
		await page.getByLabel('Days').fill(kid.early.days);
	}
	if (kid.timeZone) await page.getByLabel('Time zone').selectOption(kid.timeZone);
}

test('home with no kids says "Add your first child"', async ({ page }) => {
	await signIn(page, DAN);
	const add = page.getByRole('link', { name: 'Add your first child' });
	await expect(add).toHaveAttribute('href', '/kids/new');
	await expect(page.getByRole('navigation', { name: 'Children' })).toHaveCount(0);
	await expectAccessible(page);
});

test('the kid home shows name and age; only guardians get the profile and modules', async ({
	page
}) => {
	await signIn(page, ANA);
	await expectLanding(page, { kid: MIA });
	await expect(page.getByRole('main').getByText(ageOf(MIA))).toBeVisible();
	let entries = await moreEntries(page);
	for (const [name, path] of [
		['Modules', 'modules'],
		['Child profile', 'profile'],
		['Milestones', 'milestones'],
		['Export PDF', 'export']
	]) {
		await expect(entry(entries, name)).toHaveAttribute('href', `/kids/${MIA.id}/${path}`);
	}
	if (await entries.getByRole('button', { name: 'Close' }).isVisible()) {
		await page.keyboard.press('Escape');
	}
	await expectAccessible(page);
	const sheet = await openSwitcher(page, MIA.name);
	await sheet.getByRole('link', { name: new RegExp(`^${LEO.name}`) }).click();
	await expect(page.getByRole('heading', { name: LEO.name, level: 1 })).toBeVisible();
	await expect(page.getByRole('main').getByText(ageOf(LEO))).toBeVisible();
	entries = await moreEntries(page);
	await expect(
		entries.getByRole('link', { name: /^\s*(Modules|Child profile|Settings)/ })
	).toHaveCount(0);
	// Leo is past the milestone lists, so his Milestones module is off and the entry is hidden.
	await expect(entry(entries, 'Milestones')).toHaveCount(0);
	await expect(entry(entries, 'Export PDF')).toHaveAttribute('href', `/kids/${LEO.id}/export`);
	await expectAccessible(page);
});

test('/ and a kid page are sent with Cache-Control: no-store', async ({ page }) => {
	await signIn(page, ANA);
	for (const path of [
		'/',
		`/kids/${MIA.id}`,
		`/kids/${MIA.id}/settings`,
		`/kids/${MIA.id}/profile`
	]) {
		const response = await page.request.get(path);
		expect(response.status()).toBe(200);
		expect(response.headers()['cache-control']).toBe('no-store');
	}
});

test('the last visited kid is remembered in hbt_kid and preselected on /', async ({
	page,
	context
}) => {
	await signIn(page, ANA);
	const sheet = await openSwitcher(page, MIA.name);
	await expect(sheet.getByRole('link', { name: new RegExp(`^${MIA.name}`) })).toHaveAttribute(
		'aria-current',
		'true'
	);
	await sheet.getByRole('link', { name: new RegExp(`^${LEO.name}`) }).click();
	await expect(page).toHaveURL(new RegExp(`/kids/${LEO.id}$`));
	await expect(page.getByRole('heading', { name: LEO.name, level: 1 })).toBeVisible();
	await expectAccessible(page);
	const cookie = (await context.cookies()).find((c) => c.name === KID_COOKIE);
	expect(cookie).toMatchObject({
		value: LEO.id,
		httpOnly: true,
		secure: true,
		sameSite: 'Strict',
		path: '/'
	});
	const days = ((cookie?.expires ?? 0) - Date.now() / 1000) / 86400;
	expect(days).toBeGreaterThan(29.9);
	expect(days).toBeLessThanOrEqual(30);

	await page.goto('/');
	const again = await openSwitcher(page, LEO.name);
	await expect(again.getByRole('link', { name: new RegExp(`^${LEO.name}`) })).toHaveAttribute(
		'aria-current',
		'true'
	);
	await expect(again.getByRole('link', { name: new RegExp(`^${MIA.name}`) })).not.toHaveAttribute(
		'aria-current'
	);
});

test.describe('the time zone', () => {
	test.use({ timezoneId: 'Asia/Tokyo' });

	test('defaults to the device zone and stays editable', async ({ page }) => {
		await signIn(page, ANA);
		await page.goto('/kids/new');
		await expect(page.getByLabel('Time zone')).toHaveValue('Asia/Tokyo');
		await page.getByLabel('Time zone').selectOption('America/Mexico_City');
		await expect(page.getByLabel('Time zone')).toHaveValue('America/Mexico_City');
	});
});

test.describe('a device zone ICU lists only under an older name', () => {
	test.use({ timezoneId: 'Asia/Kolkata' });

	test('is added, selected and stored; the home page shows the age', async ({ page }) => {
		// Chromium reports ICU's Asia/Calcutta; Firefox and Safari report the current name.
		await page.addInitScript(() => {
			const resolved = Intl.DateTimeFormat.prototype.resolvedOptions;
			Intl.DateTimeFormat.prototype.resolvedOptions = function () {
				return { ...resolved.call(this), timeZone: 'Asia/Kolkata' };
			};
		});
		await signIn(page, ANA);
		await page.goto('/kids/new');
		await expect(page.getByLabel('Time zone')).toHaveValue('Asia/Kolkata');
		await expectAccessible(page);
		await fillKid(page, { name: 'Anu', birthDate: '2026-09-01', sex: 'Female' });
		await page.getByRole('button', { name: 'Add child' }).click();
		await expect(page.getByRole('heading', { name: 'Anu', level: 1 })).toBeVisible();
		await page.goto('/');
		await expect(
			page.getByRole('main').getByText(formatAge('2026-09-01', todayIn('Asia/Kolkata')))
		).toBeVisible();
		await expectAccessible(page);
	});
});

// Stored through another browser that reports the current IANA name.
async function storeKolkataKid(): Promise<string> {
	const session = await openSession(ANA.username, ANA.password);
	const res = await fetch(`${STUB_URL}/v1/kids`, {
		method: 'POST',
		headers: {
			authorization: `Bearer ${STUB_TOKEN}`,
			'x-session-token': session,
			'content-type': 'application/json'
		},
		body: JSON.stringify({
			name: 'Anu',
			birth_date: '2026-09-01',
			sex: 'female',
			time_zone: 'Asia/Kolkata'
		})
	});
	expect(res.status).toBe(201);
	return ((await res.json()) as { id: string }).id;
}

test('settings keeps a stored zone missing from the ICU list selected', async ({ page }) => {
	const id = await storeKolkataKid();
	await signIn(page, ANA);
	await page.goto(`/kids/${id}/settings`);
	await expect(page.getByLabel('Time zone')).toHaveValue('Asia/Kolkata');
	await expectAccessible(page);
	await page.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByRole('status')).toHaveText('Saved');
	await page.reload();
	await expect(page.getByLabel('Time zone')).toHaveValue('Asia/Kolkata');
	await page.goto('/');
	await expect(
		page.getByRole('main').getByText(formatAge('2026-09-01', todayIn('Asia/Kolkata')))
	).toBeVisible();
});

test('create a child without "born early"', async ({ page }) => {
	await signIn(page, ANA);
	await (await openSwitcher(page, MIA.name)).getByRole('link', { name: 'Add a child' }).click();
	await expect(page.getByRole('dialog', { name: 'Add a child' })).toBeVisible();
	await expect(page.getByLabel('Weeks')).toBeHidden();
	await expectAccessible(page);
	await fillKid(page, { name: '  Noa ', birthDate: '2026-10-01', sex: 'Female', timeZone: 'UTC' });
	await page.getByRole('button', { name: 'Add child' }).click();
	await expect(page.getByRole('heading', { name: 'Noa', level: 1 })).toBeVisible();
	await expect(
		page.getByRole('main').getByText(formatAge('2026-10-01', todayIn('UTC')))
	).toBeVisible();
	await expectAccessible(page);

	await entry(await moreEntries(page), 'Child profile').click();
	await page.getByRole('link', { name: 'Edit details' }).click();
	await expect(page.getByLabel('Name', { exact: true })).toHaveValue('Noa');
	await expect(page.getByRole('checkbox', { name: 'Born early?' })).not.toBeChecked();
	await expect(page.getByLabel('Weeks')).toBeHidden();
	await expect(page.getByLabel('Time zone')).toHaveValue('UTC');
	await expectAccessible(page);
});

test('create a child born early', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto('/kids/new');
	await page.getByRole('checkbox', { name: 'Born early?' }).check();
	await expect(page.getByLabel('Weeks')).toBeVisible();
	await expectAccessible(page);
	await fillKid(page, {
		name: 'Ivo',
		birthDate: '2026-09-15',
		sex: 'Male',
		early: { weeks: '34', days: '5' },
		timeZone: 'Europe/Madrid'
	});
	await page.getByRole('button', { name: 'Add child' }).click();
	await expect(page.getByRole('heading', { name: 'Ivo', level: 1 })).toBeVisible();
	await entry(await moreEntries(page), 'Child profile').click();
	await page.getByRole('link', { name: 'Edit details' }).click();
	await expect(page.getByRole('checkbox', { name: 'Born early?' })).toBeChecked();
	await expect(page.getByLabel('Weeks')).toHaveValue('34');
	await expect(page.getByLabel('Days')).toHaveValue('5');
	await expect(page.getByLabel('Time zone')).toHaveValue('Europe/Madrid');
	await expect(page.getByRole('radio', { name: 'Male', exact: true })).toBeChecked();
});

test('field errors from a 422 land on the right field', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto('/kids/new');
	const submit = () => page.getByRole('button', { name: 'Add child' }).click();

	await fillKid(page, { name: '   ', birthDate: '2026-10-01', sex: 'Female', timeZone: 'UTC' });
	await submit();
	await expectFieldError(
		page.getByLabel('Name', { exact: true }),
		'Enter a name of 1 to 64 characters'
	);
	await expectAccessible(page);

	await fillKid(page, { name: 'Noa', birthDate: '2999-01-01' });
	await submit();
	await expectFieldError(
		page.getByLabel('Birth date'),
		'Enter a birth date from 1900-01-01 to today'
	);
	await expect(page.getByLabel('Name', { exact: true })).not.toHaveAttribute('aria-invalid');
	await expect(page.getByLabel('Name', { exact: true })).toHaveValue('Noa');

	await page.goto('/kids/new');
	await fillKid(page, { name: 'Noa', birthDate: '2026-10-01', timeZone: 'UTC' });
	await submit();
	await expectFieldError(page.getByRole('radiogroup', { name: 'Sex' }), 'Choose female or male');

	await fillKid(page, { sex: 'Female', early: { weeks: '21', days: '0' } });
	await submit();
	await expectFieldError(page.getByLabel('Weeks'), 'Enter weeks from 22 to 44');

	await fillKid(page, { early: { weeks: '30', days: '7' } });
	await submit();
	await expectFieldError(page.getByLabel('Days'), 'Enter days from 0 to 6');
	await expectAccessible(page);

	await fillKid(page, { early: { weeks: '30', days: '1' } });
	// Stands in for a zone the API cannot load: the select itself only offers real ones.
	await page.getByLabel('Time zone').evaluate((select: HTMLSelectElement) => {
		select.removeAttribute('name');
		const zone = Object.assign(document.createElement('input'), {
			type: 'hidden',
			name: 'time_zone',
			value: 'Mars/Olympus'
		});
		select.form?.append(zone);
	});
	await submit();
	await expectFieldError(page.getByLabel('Time zone'), 'Choose a time zone from the list');
	await expectAccessible(page);
});

test('settings saves the full set; turning off "born early" clears the gestation', async ({
	page
}) => {
	await signIn(page, CLEO);
	await page.goto(`/kids/${LEO.id}/settings`);
	await untilHydrated(page.getByLabel('Name', { exact: true }));
	await expect(page.getByRole('heading', { name: 'Edit details', level: 1 })).toBeVisible();
	await expect(page.getByRole('checkbox', { name: 'Born early?' })).toBeChecked();
	await expect(page.getByLabel('Time zone')).toHaveValue(LEO.timeZone);
	await expectAccessible(page);

	await page.getByRole('checkbox', { name: 'Born early?' }).uncheck();
	await page.getByLabel('Name', { exact: true }).fill('Leonardo');
	await page.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByRole('status')).toHaveText('Saved');
	await expectAccessible(page);

	await page.reload();
	await untilHydrated(page.getByLabel('Name', { exact: true }));
	await expect(page.getByLabel('Name', { exact: true })).toHaveValue('Leonardo');
	await expect(page.getByRole('checkbox', { name: 'Born early?' })).not.toBeChecked();
	await expect(page.getByLabel('Time zone')).toHaveValue(LEO.timeZone);
	await page.getByRole('checkbox', { name: 'Born early?' }).check();
	await expect(page.getByLabel('Weeks')).toHaveValue('');
});

test('settings shows a 422 on its field', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${MIA.id}/settings`);
	await untilHydrated(page.getByLabel('Name', { exact: true }));
	await page.getByLabel('Birth date').fill('1899-12-31');
	await page.getByRole('button', { name: 'Save' }).click();
	await expectFieldError(
		page.getByLabel('Birth date'),
		'Enter a birth date from 1900-01-01 to today'
	);
	await expectAccessible(page);
});

const removeButton = (page: Page, name: string) =>
	page.getByRole('button', { name: `Delete all data for ${name}`, exact: true });
const openDelete = (page: Page, name: string) =>
	page.getByText(`Delete all data for ${name}…`, { exact: true }).click();
const tick = (page: Page) =>
	page.getByRole('checkbox', { name: 'I understand this is permanent and immediate' });

test('delete needs the tick and the exact name', async ({ page, context }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${MIA.id}`);
	await page.goto(`/kids/${MIA.id}/settings`);
	await untilHydrated(page.getByLabel('Name', { exact: true }));
	await expect(page.getByLabel(`Type ${MIA.name} to confirm`)).toBeHidden();
	await openDelete(page, MIA.name);
	const confirm = page.getByLabel(`Type ${MIA.name} to confirm`);
	await expect(removeButton(page, MIA.name)).toBeDisabled();
	await tick(page).check();
	await expect(removeButton(page, MIA.name)).toBeEnabled();
	await confirm.fill(MIA.name.toLowerCase());
	await removeButton(page, MIA.name).click();
	await expectFieldError(confirm, 'The name does not match');
	await expectAccessible(page);

	await confirm.fill(MIA.name);
	await removeButton(page, MIA.name).click();
	await expectLanding(page, { kid: LEO });
	const sheet = await openSwitcher(page, LEO.name);
	await expect(sheet.getByRole('link', { name: new RegExp(`^${MIA.name}`) })).toHaveCount(0);
	await expect(sheet.getByRole('link', { name: new RegExp(`^${IVY.name}`) })).toBeVisible();
	// The landing remembers the kid it showed, so this cannot tell whether the deleted kid's cookie was cleared.
	expect((await context.cookies()).find((c) => c.name === KID_COOKIE)?.value).toBe(LEO.id);
	await expectNotFoundPage(page, `/kids/${MIA.id}`);
});

test('a caregiver sees the kid without settings or access; direct visits are 404', async ({
	page
}) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${LEO.id}`);
	await expect(page.getByRole('heading', { name: LEO.name, level: 1 })).toBeVisible();
	await expect(page.getByRole('link', { name: /Settings|Access/ })).toHaveCount(0);
	await expect(page.getByRole('button', { name: /Delete|Save/ })).toHaveCount(0);
	await expectAccessible(page);

	const forbidden = await expectNotFoundPage(page, `/kids/${LEO.id}/settings`);
	const unknown = await expectNotFoundPage(page, '/kids/k-unknown/settings');
	expect(forbidden).toBe(unknown);
	await expectNotFoundPage(page, `/kids/${LEO.id}/access`);
	expect(await expectNotFoundPage(page, `/kids/${LEO.id}/profile`)).toBe(unknown);
	await expectNotFoundPage(page, '/kids/k-unknown');
	await expectNotFoundPage(page, '/kids/not.an.id');
});

test('settings offer None yet and exactly the shipped schedules; saving one shows on the Vaccines page', async ({
	page
}) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${IVY.id}/settings`);
	await untilHydrated(page.getByLabel('Name', { exact: true }));
	await expect(schedule(page)).toHaveValue('');
	expect(await optionValues(page)).toEqual(['', 'us', 'uk']);
	await expect(schedule(page).locator('option')).toHaveText([
		'None yet',
		'United States (stub) — Stub authority, Stub edition',
		'United Kingdom (stub) — Stub authority, Stub edition'
	]);
	await expectAccessible(page);

	await schedule(page).selectOption('uk');
	await page.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByRole('status')).toHaveText('Saved');
	expect(await lastPatch(IVY.id)).toMatchObject({ vaccine_schema: 'uk' });
	await page.goto(`/kids/${IVY.id}/vaccines`);
	await expect(page.getByText(CAPTION)).toBeVisible();
	await expect(page.getByText(NO_SCHEDULE)).toHaveCount(0);

	await page.goto(`/kids/${IVY.id}/settings`);
	await untilHydrated(page.getByLabel('Name', { exact: true }));
	await expect(schedule(page)).toHaveValue('uk');
	await schedule(page).selectOption('');
	await page.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByRole('status')).toHaveText('Saved');
	expect(await lastPatch(IVY.id)).toMatchObject({ vaccine_schema: null });
	await page.goto(`/kids/${IVY.id}/vaccines`);
	await expect(page.getByText(NO_SCHEDULE)).toBeVisible();
});

test('a new child has no schedule and the Vaccines page asks to choose one', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto('/kids/new');
	await expect(schedule(page)).toHaveValue('');
	expect(await optionValues(page)).toEqual(['', 'us', 'uk']);
	await expectAccessible(page);
	await fillKid(page, { name: 'Noa', birthDate: '2026-08-01', sex: 'Female' });
	await page.getByRole('button', { name: 'Add child' }).click();
	await expect(page.getByRole('heading', { name: 'Noa', level: 1 })).toBeVisible();
	const kids = (await (await stubAs(ANA))('GET', '/v1/kids')).body.kids as Record<
		string,
		unknown
	>[];
	const noa = kids.find((k) => k.name === 'Noa');
	expect(noa).toMatchObject({ vaccine_schema: null });
	await page.goto(`/kids/${noa?.id}/vaccines`);
	await expect(page.getByText(NO_SCHEDULE)).toBeVisible();
	await expect(page.getByRole('link', { name: /Choose a schedule/ })).toBeVisible();
});

test('a retired stored schedule stays selected and kept on save until another is chosen', async ({
	page
}) => {
	await storeVaccineSchema(IVY.id, 'retired-2020');
	await signIn(page, ANA);
	await page.goto(`/kids/${IVY.id}/settings`);
	await untilHydrated(page.getByLabel('Name', { exact: true }));
	await expect(schedule(page)).toHaveValue('retired-2020');
	expect(await optionValues(page)).toEqual(['retired-2020', '', 'us', 'uk']);
	await expect(schedule(page).locator('option').first()).toHaveText(
		'Unavailable schedule (retired-2020)'
	);
	await expectAccessible(page);

	await page.getByLabel('Name', { exact: true }).fill('Ivy Rose');
	await page.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByRole('status')).toHaveText('Saved');
	const body = await lastPatch(IVY.id);
	expect(body.name).toBe('Ivy Rose');
	expect(body.vaccine_schema).toBe('retired-2020');
	expect(await storedSchedule(IVY.id)).toBe('retired-2020');
	await expect(schedule(page)).toHaveValue('retired-2020');

	await schedule(page).selectOption('');
	await submit(page.getByRole('button', { name: 'Save' }));
	await expect(page.getByRole('status')).toHaveText('Saved');
	expect(await storedSchedule(IVY.id)).toBeNull();
	await page.reload();
	expect(await optionValues(page)).toEqual(['', 'us', 'uk']);
});

test('settings change the overdue minutes from 60 to 15 and read 15 back', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${MIA.id}/settings`);
	await untilHydrated(page.getByLabel('Name', { exact: true }));
	await expect(overdue(page)).toHaveValue('60');
	await expect(overdue(page)).toHaveAccessibleDescription(OVERDUE_HINT);
	await expectAccessible(page);

	await page.getByLabel('Name', { exact: true }).fill('Mia Rose');
	await submit(page.getByRole('button', { name: 'Save' }));
	await expect(page.getByRole('status')).toHaveText('Saved');
	expect(await lastPatch(MIA.id)).not.toHaveProperty('reminder_overdue_minutes');

	await overdue(page).fill('15');
	await submit(page.getByRole('button', { name: 'Save' }));
	await expect(page.getByRole('status')).toHaveText('Saved');
	expect(await lastPatch(MIA.id)).toMatchObject({ reminder_overdue_minutes: 15 });
	expect(await storedOverdue(MIA.id)).toBe(15);
	await expectAccessible(page);

	await page.reload();
	await untilHydrated(page.getByLabel('Name', { exact: true }));
	await expect(overdue(page)).toHaveValue('15');
	await page.getByLabel('Name', { exact: true }).fill('Mia');
	await submit(page.getByRole('button', { name: 'Save' }));
	await expect(page.getByRole('status')).toHaveText('Saved');
	expect(await lastPatch(MIA.id)).not.toHaveProperty('reminder_overdue_minutes');
	expect(await storedOverdue(MIA.id)).toBe(15);
});

test('settings refuse overdue minutes outside the range on the field and keep what was typed', async ({
	page
}) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${MIA.id}/settings`);
	await untilHydrated(overdue(page));
	const message = 'Enter minutes from 5 to 1440';
	for (const text of ['4', '1441', '60.5', '']) {
		await overdue(page).fill(text);
		await submit(page.getByRole('button', { name: 'Save' }));
		await expectFieldError(overdue(page), message);
		await expect(overdue(page)).toHaveValue(text);
		await expectAccessible(page);
	}
	expect(await patches(MIA.id)).toHaveLength(0);
	expect(await storedOverdue(MIA.id)).toBe(60);

	await overdue(page).fill('5');
	await submit(page.getByRole('button', { name: 'Save' }));
	await expect(page.getByRole('status')).toHaveText('Saved');
	expect(await storedOverdue(MIA.id)).toBe(5);
});

test('a new child form does not offer the overdue minutes', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto('/kids/new');
	await expect(overdue(page)).toHaveCount(0);
	await expect(page.getByText(OVERDUE_HINT)).toHaveCount(0);
});

test.describe('without JavaScript', () => {
	// axe runs in the page, so these pages are checked by the JavaScript tests above.
	test.use({ javaScriptEnabled: false, timezoneId: 'Asia/Tokyo' });

	test('creating a child works; the zone stays UTC and the early fields follow the box', async ({
		page
	}) => {
		await signIn(page, ANA);
		await page.goto('/kids/new');
		await expect(page.getByLabel('Time zone')).toHaveValue('UTC');
		await expect(page.getByLabel('Weeks')).toBeHidden();
		await fillKid(page, {
			name: 'Zoe',
			birthDate: '2026-08-01',
			sex: 'Female',
			early: { weeks: '36', days: '0' }
		});
		await page.getByRole('button', { name: 'Add child' }).click();
		await expect(page.getByRole('heading', { name: 'Zoe', level: 1 })).toBeVisible();
		await page.goto('/');
		await expect(
			page.getByRole('main').getByText(formatAge('2026-08-01', todayIn('UTC')))
		).toBeVisible();
	});

	test('settings renders a stored zone missing from the ICU list as selected', async ({ page }) => {
		const id = await storeKolkataKid();
		await signIn(page, ANA);
		await page.goto(`/kids/${id}/settings`);
		await expect(page.getByLabel('Time zone')).toHaveValue('Asia/Kolkata');
	});

	test('settings saves a schedule as a plain form post', async ({ page }) => {
		await signIn(page, ANA);
		await page.goto(`/kids/${IVY.id}/settings`);
		await schedule(page).selectOption('us');
		await page.getByRole('button', { name: 'Save' }).click();
		await expect(page.getByRole('status')).toHaveText('Saved');
		expect(await storedSchedule(IVY.id)).toBe('us');
		await page.goto(`/kids/${IVY.id}/vaccines`);
		await expect(page.getByText(CAPTION)).toBeVisible();
		expect(page.url()).toBe(`${UI_ORIGIN}/kids/${IVY.id}/vaccines`);
	});

	test('settings save the overdue minutes as a plain form post and refuse a bad one', async ({
		page
	}) => {
		await signIn(page, ANA);
		await page.goto(`/kids/${MIA.id}/settings`);
		await overdue(page).fill('4');
		await submit(page.getByRole('button', { name: 'Save' }));
		await expectFieldError(overdue(page), 'Enter minutes from 5 to 1440');
		await expect(overdue(page)).toHaveValue('4');
		expect(await storedOverdue(MIA.id)).toBe(60);

		await overdue(page).fill('15');
		await submit(page.getByRole('button', { name: 'Save' }));
		await expect(page.getByRole('status')).toHaveText('Saved');
		expect(await storedOverdue(MIA.id)).toBe(15);
		await expect(overdue(page)).toHaveValue('15');
	});

	test('delete is a plain form post that still checks the name', async ({ page }) => {
		await signIn(page, ANA);
		await page.goto(`/kids/${MIA.id}/settings`);
		await openDelete(page, MIA.name);
		const confirm = page.getByLabel(`Type ${MIA.name} to confirm`);
		await tick(page).check();
		await confirm.fill('Nope');
		await removeButton(page, MIA.name).click();
		await expectFieldError(confirm, 'The name does not match');
		await confirm.fill(MIA.name);
		await removeButton(page, MIA.name).click();
		await expectLanding(page, { kid: LEO });
		expect((await page.goto(`/kids/${MIA.id}`))?.status()).toBe(404);
	});

	test('a 422 lands on its field without JavaScript', async ({ page }) => {
		await signIn(page, ANA);
		await page.goto('/kids/new');
		await fillKid(page, { name: 'Zoe', birthDate: '2026-08-01' });
		await page.getByRole('button', { name: 'Add child' }).click();
		await expectFieldError(page.getByRole('radiogroup', { name: 'Sex' }), 'Choose female or male');
		await expect(page.getByLabel('Name', { exact: true })).toHaveValue('Zoe');
	});
});
