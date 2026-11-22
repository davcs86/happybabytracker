import { expect, test, type Page } from '@playwright/test';
import { formatAge, todayIn } from '../src/lib/format.js';
import {
	KID_COOKIE,
	expectAccessible,
	expectFieldError,
	expectLanding,
	expectNotFoundPage,
	signIn
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
	openSession,
	resetStub
} from './stub-api.js';

test.beforeEach(async () => {
	await resetStub();
});

const ageOf = (kid: { birthDate: string; timeZone: string }) =>
	formatAge(kid.birthDate, todayIn(kid.timeZone));
const switcherOf = (page: Page) => page.getByRole('navigation', { name: 'Children' });

type KidForm = {
	name?: string;
	birthDate?: string;
	sex?: 'Female' | 'Male';
	early?: { weeks: string; days: string };
	timeZone?: string;
};

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

test('the kid home shows name and age; only guardians get settings, access and modules', async ({
	page
}) => {
	await signIn(page, ANA);
	await expectLanding(page, { kid: MIA });
	await expect(page.getByText(ageOf(MIA))).toBeVisible();
	for (const [name, path] of [
		['Settings', 'settings'],
		['Access', 'access'],
		['Modules', 'modules']
	]) {
		await expect(page.getByRole('link', { name: `${name} for ${MIA.name}` })).toHaveAttribute(
			'href',
			`/kids/${MIA.id}/${path}`
		);
	}
	await expect(switcherOf(page).getByRole('link')).toHaveText([MIA.name, LEO.name, IVY.name]);
	await expectAccessible(page);

	await switcherOf(page).getByRole('link', { name: LEO.name }).click();
	await expect(page.getByRole('heading', { name: LEO.name, level: 1 })).toBeVisible();
	await expect(page.getByText(ageOf(LEO))).toBeVisible();
	await expect(page.getByRole('link', { name: /Settings|Access|Modules/ })).toHaveCount(0);
	await expectAccessible(page);
});

test('/ and a kid page are sent with Cache-Control: no-store', async ({ page }) => {
	await signIn(page, ANA);
	for (const path of ['/', `/kids/${MIA.id}`, `/kids/${MIA.id}/settings`]) {
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
	const switcher = switcherOf(page);
	await expect(switcher.getByRole('link', { name: MIA.name })).toHaveAttribute(
		'aria-current',
		'true'
	);
	await switcher.getByRole('link', { name: LEO.name }).click();
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
	await expect(switcher.getByRole('link', { name: LEO.name })).toHaveAttribute(
		'aria-current',
		'true'
	);
	await expect(switcher.getByRole('link', { name: MIA.name })).not.toHaveAttribute('aria-current');
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
		await expect(page.getByText(formatAge('2026-09-01', todayIn('Asia/Kolkata')))).toBeVisible();
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
	await expect(page.getByText(formatAge('2026-09-01', todayIn('Asia/Kolkata')))).toBeVisible();
});

test('create a child without "born early"', async ({ page }) => {
	await signIn(page, ANA);
	await page.getByRole('link', { name: 'Add a child' }).click();
	await expect(page.getByRole('heading', { name: 'Add a child', level: 1 })).toBeVisible();
	await expect(page.getByLabel('Weeks')).toBeHidden();
	await expectAccessible(page);
	await fillKid(page, { name: '  Noa ', birthDate: '2026-10-01', sex: 'Female', timeZone: 'UTC' });
	await page.getByRole('button', { name: 'Add child' }).click();
	await expect(page.getByRole('heading', { name: 'Noa', level: 1 })).toBeVisible();
	await expect(page.getByText(formatAge('2026-10-01', todayIn('UTC')))).toBeVisible();
	await expectAccessible(page);

	await page.getByRole('link', { name: 'Settings for Noa' }).click();
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
	await page.getByRole('link', { name: 'Settings for Ivo' }).click();
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
	await expect(page.getByRole('heading', { name: `Settings for ${LEO.name}` })).toBeVisible();
	await expect(page.getByRole('checkbox', { name: 'Born early?' })).toBeChecked();
	await expect(page.getByLabel('Time zone')).toHaveValue(LEO.timeZone);
	await expectAccessible(page);

	await page.getByRole('checkbox', { name: 'Born early?' }).uncheck();
	await page.getByLabel('Name', { exact: true }).fill('Leonardo');
	await page.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByRole('status')).toHaveText('Saved');
	await expectAccessible(page);

	await page.reload();
	await expect(page.getByLabel('Name', { exact: true })).toHaveValue('Leonardo');
	await expect(page.getByRole('checkbox', { name: 'Born early?' })).not.toBeChecked();
	await expect(page.getByLabel('Time zone')).toHaveValue(LEO.timeZone);
	await page.getByRole('checkbox', { name: 'Born early?' }).check();
	await expect(page.getByLabel('Weeks')).toHaveValue('');
});

test('settings shows a 422 on its field', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${MIA.id}/settings`);
	await page.getByLabel('Birth date').fill('1899-12-31');
	await page.getByRole('button', { name: 'Save' }).click();
	await expectFieldError(
		page.getByLabel('Birth date'),
		'Enter a birth date from 1900-01-01 to today'
	);
	await expectAccessible(page);
});

test('delete needs the exact name', async ({ page, context }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${MIA.id}`);
	await page.goto(`/kids/${MIA.id}/settings`);
	const confirm = page.getByLabel(`Type ${MIA.name} to confirm`);
	await confirm.fill(MIA.name.toLowerCase());
	await page.getByRole('button', { name: `Delete ${MIA.name}` }).click();
	await expectFieldError(confirm, 'The name does not match');
	await expectAccessible(page);

	await confirm.fill(MIA.name);
	await page.getByRole('button', { name: `Delete ${MIA.name}` }).click();
	await expectLanding(page, { kid: LEO });
	await expect(switcherOf(page).getByRole('link')).toHaveText([LEO.name, IVY.name]);
	// The cookie of the deleted kid is cleared; the landing then remembers the kid it showed.
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
	await expectNotFoundPage(page, '/kids/k-unknown');
	await expectNotFoundPage(page, '/kids/not.an.id');
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
		await expect(page.getByText(formatAge('2026-08-01', todayIn('UTC')))).toBeVisible();
	});

	test('settings renders a stored zone missing from the ICU list as selected', async ({ page }) => {
		const id = await storeKolkataKid();
		await signIn(page, ANA);
		await page.goto(`/kids/${id}/settings`);
		await expect(page.getByLabel('Time zone')).toHaveValue('Asia/Kolkata');
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
