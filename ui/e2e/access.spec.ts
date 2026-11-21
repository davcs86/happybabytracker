import { expect, test, type Page } from '@playwright/test';
import {
	KID_COOKIE,
	expectAccessible,
	expectFieldError,
	expectNotFoundPage,
	signIn
} from './helpers.js';
import {
	ADMIN,
	ANA,
	CLEO,
	LEO,
	MIA,
	STUB_TOKEN,
	STUB_URL,
	UI_ORIGIN,
	openSession,
	receivedRequests,
	resetStub
} from './stub-api.js';

const PARENT_NOT_FOUND = 'No active account has that username.';
const LAST_GUARDIAN = 'A child needs at least one guardian.';

test.beforeEach(async () => {
	await resetStub();
});

const grant = (page: Page, displayName: string) =>
	page
		.getByRole('list', { name: 'People with access' })
		.getByRole('listitem')
		.filter({ hasText: displayName });

async function add(page: Page, username: string, role: 'Caregiver' | 'Guardian') {
	await page.getByLabel('Username').fill(username);
	await page.getByLabel('Role', { exact: true }).selectOption({ label: role });
	await page.getByRole('button', { name: 'Add', exact: true }).click();
}

test('a guardian sees the grants with their roles', async ({ page }) => {
	await signIn(page, CLEO);
	await page.goto(`/kids/${LEO.id}/access`);
	await expect(page.getByRole('heading', { name: `Access to ${LEO.name}` })).toBeVisible();
	await expect(grant(page, 'Cleo')).toContainText('Guardian');
	await expect(grant(page, 'Cleo')).toContainText('(you)');
	await expect(grant(page, 'Ana')).toContainText('Caregiver');
	await expect(grant(page, 'Cleo').getByRole('button', { name: 'Leave' })).toBeVisible();
	await expect(grant(page, 'Ana').getByRole('button', { name: 'Leave' })).toHaveCount(0);
	await expectAccessible(page);
});

test('grant, change role and remove', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${MIA.id}/access`);
	await add(page, '  DAN ', 'Caregiver');
	await expect(grant(page, 'Dan')).toContainText('Caregiver');
	await expectAccessible(page);

	await grant(page, 'Dan').getByLabel('Role for Dan').selectOption({ label: 'Guardian' });
	await grant(page, 'Dan').getByRole('button', { name: 'Change role for Dan' }).click();
	await expect(grant(page, 'Dan')).toContainText('Guardian');
	await expect(grant(page, 'Dan').getByLabel('Role for Dan')).toHaveValue('guardian');

	await grant(page, 'Dan').getByRole('button', { name: 'Remove Dan' }).click();
	await expect(grant(page, 'Dan')).toHaveCount(0);
	await expect(grant(page, 'Ana')).toBeVisible();
	await expectAccessible(page);
});

test(`an unknown or disabled username reads "${PARENT_NOT_FOUND}"`, async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${MIA.id}/access`);
	for (const username of ['nobody', 'dee', '   ']) {
		await add(page, username, 'Caregiver');
		await expectFieldError(page.getByLabel('Username'), PARENT_NOT_FOUND);
	}
	await expectAccessible(page);
});

test(`the only guardian cannot leave: "${LAST_GUARDIAN}"`, async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${MIA.id}/access`);
	await grant(page, 'Ana').getByRole('button', { name: 'Leave' }).click();
	await expect(page.getByRole('alert')).toHaveText(LAST_GUARDIAN);
	await expect(grant(page, 'Ana')).toContainText('Guardian');
	await expectAccessible(page);
});

test(`once the other guardian left, the remaining one cannot leave: "${LAST_GUARDIAN}"`, async ({
	page
}) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${MIA.id}/access`);
	await add(page, 'cleo', 'Guardian');
	await grant(page, 'Ana').getByRole('button', { name: 'Leave' }).click();
	await expect(page).toHaveURL(/127\.0\.0\.1:\d+\/$/);

	await page.getByRole('button', { name: 'Sign out', exact: true }).click();
	await signIn(page, CLEO);
	await page.goto(`/kids/${MIA.id}/access`);
	await expect(grant(page, 'Ana')).toHaveCount(0);
	await grant(page, 'Cleo').getByRole('button', { name: 'Leave' }).click();
	await expect(page.getByRole('alert')).toHaveText(LAST_GUARDIAN);
});

test('leaving as one of two guardians goes home without the kid', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${MIA.id}/access`);
	await add(page, 'cleo', 'Guardian');
	await expect(grant(page, 'Cleo')).toContainText('Guardian');
	await grant(page, 'Ana').getByRole('button', { name: 'Leave' }).click();
	await expect(page).toHaveURL(/127\.0\.0\.1:\d+\/$/);
	await expect(page.getByRole('heading', { name: MIA.name, exact: true })).toHaveCount(0);
	await expectAccessible(page);
	await expectNotFoundPage(page, `/kids/${MIA.id}`);
});

test('the caregiver view has no admin controls', async ({ page }) => {
	await signIn(page, ANA);
	await page.goto(`/kids/${LEO.id}`);
	await expect(page.getByRole('heading', { name: LEO.name, level: 1 })).toBeVisible();
	await expect(page.getByRole('link', { name: /Access|Settings|Parents/ })).toHaveCount(0);
	await expect(
		page.getByRole('button', { name: /Remove|Change role|Add|Delete|Leave/ })
	).toHaveCount(0);
	await expectAccessible(page);
	await page.goto('/');
	await expect(page.getByRole('link', { name: `Access for ${LEO.name}` })).toHaveCount(0);
	await expect(page.getByRole('link', { name: `Settings for ${LEO.name}` })).toHaveCount(0);
	await expect(page.getByRole('link', { name: 'Parents' })).toHaveCount(0);
	await expectNotFoundPage(page, `/kids/${LEO.id}/access`);
});

test.describe('a caregiver leaves a kid', () => {
	const LEAVE_TEXT = `You will lose access to ${LEO.name}. A guardian has to add you again.`;
	const leaveControl = (page: Page) => page.getByText('Leave this child', { exact: true });
	const confirm = (page: Page) => page.getByRole('button', { name: `Yes, leave ${LEO.name}` });
	const deletes = async () => (await receivedRequests()).filter((r) => r.method === 'DELETE');

	async function openLeave(page: Page) {
		await signIn(page, ANA);
		await page.goto(`/kids/${LEO.id}`);
		await leaveControl(page).click();
		await expect(page.getByText(LEAVE_TEXT)).toBeVisible();
	}

	async function expectLeft(page: Page) {
		await expect(page).toHaveURL(/127\.0\.0\.1:\d+\/$/);
		await expect(page.getByRole('heading', { name: LEO.name, exact: true })).toHaveCount(0);
		const unknown = await expectNotFoundPage(page, '/kids/k-unknown');
		for (const path of [`/kids/${LEO.id}`, `/kids/${LEO.id}/settings`, `/kids/${LEO.id}/access`]) {
			expect(await expectNotFoundPage(page, path)).toBe(unknown);
		}
	}

	test('only a caregiver gets the control, and the first click does not submit', async ({
		page
	}) => {
		await signIn(page, ANA);
		await page.goto(`/kids/${LEO.id}`);
		await expect(leaveControl(page)).toBeVisible();
		await expect(confirm(page)).toBeHidden();
		await expectAccessible(page);
		await leaveControl(page).click();
		await expect(confirm(page)).toBeVisible();
		await expectAccessible(page);
		expect(await deletes()).toEqual([]);

		await page.goto(`/kids/${MIA.id}`);
		await expect(page.getByRole('heading', { name: MIA.name, level: 1 })).toBeVisible();
		await expect(leaveControl(page)).toHaveCount(0);
		await expectAccessible(page);
	});

	test('the admin never reaches a kid page', async ({ page }) => {
		await signIn(page, ADMIN);
		await page.goto(`/kids/${LEO.id}`);
		await expect(page).toHaveURL(/\/admin\/parents$/);
		await expect(leaveControl(page)).toHaveCount(0);
		await expectAccessible(page);
	});

	test('confirming leaves: home without the kid, then the 404 page everywhere', async ({
		page,
		context
	}) => {
		await openLeave(page);
		await confirm(page).click();
		await expectLeft(page);
		expect((await context.cookies()).find((c) => c.name === KID_COOKIE)).toBeUndefined();
		await page.goto('/');
		await expectAccessible(page);
		expect(await deletes()).toMatchObject([{ path: `/v1/kids/${LEO.id}/access/p-ana` }]);
	});

	test('closing the confirmation again sends no request', async ({ page }) => {
		await openLeave(page);
		await leaveControl(page).click();
		await expect(confirm(page)).toBeHidden();
		await page.goto(`/kids/${LEO.id}`);
		await expect(page.getByRole('heading', { name: LEO.name, level: 1 })).toBeVisible();
		expect(await deletes()).toEqual([]);
	});

	test('hbt_kid pointing at another kid is left alone', async ({ page, context }) => {
		await openLeave(page);
		await context.addCookies([
			{
				name: KID_COOKIE,
				value: MIA.id,
				url: UI_ORIGIN,
				httpOnly: true,
				secure: true,
				sameSite: 'Strict'
			}
		]);
		await confirm(page).click();
		await expect(page).toHaveURL(/127\.0\.0\.1:\d+\/$/);
		expect((await context.cookies()).find((c) => c.name === KID_COOKIE)?.value).toBe(MIA.id);
	});

	test('posted parent, parent_id and kid fields are ignored', async ({ page }) => {
		await openLeave(page);
		await confirm(page).evaluate((button: HTMLButtonElement) => {
			for (const [name, value] of [
				['parent_id', 'p-cleo'],
				['parent', 'p-cleo'],
				['kid', 'k-mia']
			]) {
				button.form?.append(
					Object.assign(document.createElement('input'), { type: 'hidden', name, value })
				);
			}
		});
		await confirm(page).click();
		await expect(page).toHaveURL(/127\.0\.0\.1:\d+\/$/);
		expect(await deletes()).toMatchObject([{ path: `/v1/kids/${LEO.id}/access/p-ana` }]);
		await expect(page.getByRole('heading', { name: MIA.name, exact: true })).toBeVisible();

		await page.getByRole('button', { name: 'Sign out', exact: true }).click();
		await signIn(page, CLEO);
		await page.goto(`/kids/${LEO.id}/access`);
		await expect(grant(page, 'Cleo')).toContainText('Guardian');
		await expect(grant(page, 'Ana')).toHaveCount(0);
	});

	test('a grant revoked meanwhile still ends at home', async ({ page }) => {
		await openLeave(page);
		const cleo = await openSession(CLEO.username, CLEO.password);
		const revoke = await fetch(`${STUB_URL}/v1/kids/${LEO.id}/access/p-ana`, {
			method: 'DELETE',
			headers: { authorization: `Bearer ${STUB_TOKEN}`, 'x-session-token': cleo }
		});
		expect(revoke.status).toBe(204);
		await confirm(page).click();
		await expect(page).toHaveURL(/127\.0\.0\.1:\d+\/$/);
		// Cleo's revoke, then Ana's leave, which the stub answers 404.
		expect((await deletes()).map((r) => r.session === cleo)).toEqual([true, false]);
	});

	test.describe('without JavaScript', () => {
		test.use({ javaScriptEnabled: false });

		// axe runs in the page, so the 404 pages are checked by the JavaScript tests above.
		test('the whole flow works', async ({ page }) => {
			await openLeave(page);
			await confirm(page).click();
			await expect(page).toHaveURL(/127\.0\.0\.1:\d+\/$/);
			await expect(page.getByRole('heading', { name: LEO.name, exact: true })).toHaveCount(0);
			for (const path of [
				`/kids/${LEO.id}`,
				`/kids/${LEO.id}/settings`,
				`/kids/${LEO.id}/access`
			]) {
				expect((await page.goto(path))?.status()).toBe(404);
				await expect(page.getByRole('heading', { name: 'Not found' })).toBeVisible();
			}
			expect(await deletes()).toMatchObject([{ path: `/v1/kids/${LEO.id}/access/p-ana` }]);
		});
	});
});
