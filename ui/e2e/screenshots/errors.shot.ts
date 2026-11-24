import { ANA, setScenario } from '../stub-api.js';
import { expect, shot, signInAs, test } from './shots.js';

test('error pages', async ({ page }) => {
	await signInAs(page, ANA);
	await page.goto('/kids/k-unknown');
	await expect(page.getByRole('heading', { name: 'Not found' })).toBeVisible();
	await shot(
		page,
		'errors',
		'01-404-unknown-kid',
		'An unknown kid (or one without a grant): the 404 page in the app shell'
	);
	await page.goto('/no-such-page');
	await expect(page.getByRole('heading', { name: 'Not found' })).toBeVisible();
	await shot(page, 'errors', '02-404-unknown-path', 'A path that does not exist');
	await page.goto('/admin/parents');
	await expect(page.getByRole('heading', { name: 'Not found' })).toBeVisible();
	await shot(page, 'errors', '03-404-parent-on-admin', 'A parent on /admin/*');
	await setScenario('unavailable');
	await page.goto('/');
	await expect(
		page.getByRole('heading', { name: 'The app cannot reach its server right now' })
	).toBeVisible();
	await shot(
		page,
		'errors',
		'04-503-api-unavailable',
		'The API is down, unavailable or slow: the 503 page, the session kept'
	);
});
