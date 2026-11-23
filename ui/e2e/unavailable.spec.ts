import { expect, test } from '@playwright/test';
import { expectAccessible, expectLanding, login, sessionCookie } from './helpers.js';
import { ANA, MIA, resetStub, setScenario, type Scenario } from './stub-api.js';

test.beforeEach(async () => {
	await resetStub();
});

const cases: [string, Scenario][] = [
	['answers 503', 'unavailable'],
	['closes the connection', 'down'],
	['is slower than the timeout', 'slow']
];

for (const [what, scenario] of cases) {
	test(`when the API ${what} a signed-in user sees the unavailable page and keeps the cookie`, async ({
		page,
		context
	}) => {
		if (scenario === 'slow') test.setTimeout(30_000);
		await page.goto('/login');
		await login(page, ANA.username, ANA.password);
		await expectLanding(page, { kid: MIA });
		const before = await sessionCookie(context);

		await setScenario(scenario);
		const response = await page.goto('/');
		expect(response?.status()).toBe(503);
		await expect(page).toHaveURL(/127\.0\.0\.1:\d+\/$/);
		await expect(
			page.getByRole('heading', { name: 'The app cannot reach its server right now' })
		).toBeVisible();
		await expectAccessible(page);
		expect((await sessionCookie(context))?.value).toBe(before?.value);

		await setScenario('normal');
		await page.reload();
		await expectLanding(page, { kid: MIA });
	});
}

test('/healthz/api passes the API status through, and /healthz never calls it', async ({
	request
}) => {
	expect((await request.get('/healthz/api')).status()).toBe(200);
	await setScenario('unavailable');
	expect((await request.get('/healthz/api')).status()).toBe(503);
	await setScenario('down');
	expect((await request.get('/healthz/api')).status()).toBe(503);
	const ui = await request.get('/healthz');
	expect(ui.status()).toBe(200);
	expect(await ui.text()).toBe('ok');
});

test('/healthz/reminders passes the API status through, without a session and without a body', async ({
	request
}) => {
	const ok = await request.get('/healthz/reminders');
	expect(ok.status()).toBe(200);
	expect(await ok.text()).toBe('ok');
	await setScenario('unavailable');
	const unavailable = await request.get('/healthz/reminders');
	expect(unavailable.status()).toBe(503);
	expect(await unavailable.text()).toBe('unavailable');
	await setScenario('down');
	expect((await request.get('/healthz/reminders')).status()).toBe(503);
});

test('a login while the API is down says so and sets no cookie', async ({ page, context }) => {
	await page.goto('/login');
	await setScenario('down');
	const response = await login(page, ANA.username, ANA.password);
	expect(response.status()).toBe(503);
	await expect(page.getByRole('alert')).toHaveText(
		'The app cannot reach its server, try again soon'
	);
	expect(await sessionCookie(context)).toBeUndefined();
});
