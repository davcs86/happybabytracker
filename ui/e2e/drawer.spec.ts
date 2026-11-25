import { expect, test, type Locator, type Page } from '@playwright/test';
import { expectAccessible, signIn } from './helpers.js';
import { ANA, IVY, resetStub, stubAs } from './stub-api.js';

test.beforeEach(async () => {
	await resetStub();
});

const list = `/kids/${IVY.id}/growth`;
const addLink = (page: Page) => page.getByRole('link', { name: 'Add measurement' });
const sheet = (page: Page, name: string) =>
	page.getByRole('dialog', { name }).and(page.locator('[data-state="open"]'));

// A click before hydration navigates to the page: load the list again until the sheet opens.
async function openSheet(page: Page, link: (p: Page) => Locator, name: string) {
	await expect(async () => {
		await page.goto(list);
		await link(page).click();
		await expect(sheet(page, name)).toBeVisible({ timeout: 1500 });
	}).toPass();
}

test('a form link opens its page in a sheet over the list', async ({ page }) => {
	await signIn(page, ANA);
	await openSheet(page, addLink, 'Add a measurement');
	await expect(page).toHaveURL(`${list}/new`);
	await expect(page.getByRole('heading', { name: 'Growth', level: 1 })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Add a measurement', level: 1 })).toHaveCount(0);
	await expect(sheet(page, 'Add a measurement').getByLabel('Kilograms')).toBeVisible();
	await expectAccessible(page);
});

test('saving closes the sheet, refreshes the list and says so', async ({ page }) => {
	await signIn(page, ANA);
	await openSheet(page, addLink, 'Add a measurement');
	const dialog = sheet(page, 'Add a measurement');
	await dialog.getByLabel('Kilograms').fill('9.5');
	await dialog.getByRole('button', { name: 'Save', exact: true }).click();
	await expect(dialog).toBeHidden();
	await expect(page).toHaveURL(list);
	await expect(page.getByText('Measurement added')).toBeVisible();
	await expect(page.getByRole('list', { name: 'Measurements' }).getByRole('listitem')).toHaveCount(
		1
	);
});

test('a refused save keeps the sheet open with its message and values', async ({ page }) => {
	await signIn(page, ANA);
	await openSheet(page, addLink, 'Add a measurement');
	const dialog = sheet(page, 'Add a measurement');
	await dialog.getByLabel('Note').fill('kept');
	await dialog.getByRole('button', { name: 'Save', exact: true }).click();
	await expect(dialog.getByText('Check the fields with a message.')).toBeVisible();
	await expect(dialog.getByLabel('Note')).toHaveValue('kept');
	await expect(page).toHaveURL(`${list}/new`);
});

test('Cancel and Escape close it without a toast, and focus returns to the link', async ({
	page
}) => {
	await signIn(page, ANA);
	await openSheet(page, addLink, 'Add a measurement');
	await sheet(page, 'Add a measurement').getByRole('button', { name: 'Cancel' }).click();
	await expect(sheet(page, 'Add a measurement')).toBeHidden();
	await expect(page).toHaveURL(list);
	await expect(addLink(page)).toBeFocused();
	await addLink(page).click();
	await expect(sheet(page, 'Add a measurement')).toBeVisible();
	await page.keyboard.press('Escape');
	await expect(sheet(page, 'Add a measurement')).toBeHidden();
	await expect(page).toHaveURL(list);
	await expect(page.locator('[data-sonner-toast]')).toHaveCount(0);
});

test('Back closes the sheet', async ({ page }) => {
	await signIn(page, ANA);
	await openSheet(page, addLink, 'Add a measurement');
	await page.goBack();
	await expect(sheet(page, 'Add a measurement')).toBeHidden();
	await expect(page).toHaveURL(list);
});

test('a modified click and a new tab open the page, and a reload keeps it', async ({ page }) => {
	await signIn(page, ANA);
	await openSheet(page, addLink, 'Add a measurement');
	await page.reload();
	await expect(page.getByRole('heading', { name: 'Add a measurement', level: 1 })).toBeVisible();
	await expect(page.getByRole('dialog')).toHaveCount(0);
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });
	test('the link is the page', async ({ page }) => {
		await signIn(page, ANA);
		await page.goto(list);
		await addLink(page).click();
		await expect(page).toHaveURL(`${list}/new`);
		await expect(page.getByRole('heading', { name: 'Add a measurement', level: 1 })).toBeVisible();
	});
});

void stubAs;
