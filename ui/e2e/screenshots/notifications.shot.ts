import type { Page } from '@playwright/test';
import { submit } from '../helpers.js';
import { ANA, IVY, MIA, fakePushKeys, setScenario } from '../stub-api.js';
import { as, must } from './seed.js';
import { expect, openDetails, shot, signInAs, test } from './shots.js';

const PAGE = '/account/notifications';
const IPHONE_UA =
	'Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1';
// The stub draws a random topic: the shot shows this one instead.
const FROZEN_TOPIC = 'hbt-FrozenTopicForShots0';

async function open(page: Page) {
	await page.goto(PAGE);
	await page.waitForLoadState('networkidle');
}

async function freezeTopic(page: Page) {
	await page.locator('dd code').evaluate((el, topic) => (el.textContent = topic), FROZEN_TOPIC);
}

const device = (n: string, label: string) => ({
	endpoint: `https://push.example/send/${n}-endpoint`,
	keys: fakePushKeys(),
	label
});

test('web push and ntfy', async ({ page }) => {
	const ana = await as(ANA);
	await signInAs(page, ANA);
	await open(page);
	await shot(
		page,
		'notifications',
		'01-nothing-on',
		'Web Push and ntfy configured, nothing turned on yet'
	);
	await page.getByRole('button', { name: 'Turn on notifications on this device' }).click();
	await expect(page.getByRole('alert')).toBeVisible();
	await shot(
		page,
		'notifications',
		'02-permission-blocked',
		'The browser denies the permission: how to allow it'
	);
	await submit(page.getByRole('button', { name: 'Turn on ntfy' }));
	await page.waitForLoadState('networkidle');
	await freezeTopic(page);
	await shot(
		page,
		'notifications',
		'03-ntfy-on',
		'ntfy on: the server, the topic (frozen), the kids to remind about; the blocked-notifications message from the previous shot is still shown'
	);
	await openDetails(page, 'New topic');
	await freezeTopic(page);
	await shot(
		page,
		'notifications',
		'04-ntfy-new-topic-confirm',
		'New topic opened: the confirm step; the blocked-notifications message from the previous shot is still shown'
	);

	const phone = (await must(ana, 'PUT', '/v1/me/push-devices', device('phone', 'Ana’s phone')))
		.device;
	await must(ana, 'PUT', '/v1/me/push-devices', device('laptop', 'Work laptop'));
	await must(ana, 'PUT', `/v1/kids/${MIA.id}/push-devices/${phone.id}`);
	await must(ana, 'PUT', `/v1/kids/${IVY.id}/push-devices/${phone.id}`);
	await open(page);
	await freezeTopic(page);
	await shot(
		page,
		'notifications',
		'05-other-devices',
		'Two devices registered elsewhere, with their names and when they were added'
	);
	await page.context().addCookies([
		{
			name: 'hbt_push_device',
			value: phone.id,
			domain: '127.0.0.1',
			path: PAGE,
			httpOnly: true,
			secure: true,
			sameSite: 'Strict'
		}
	]);
	await open(page);
	await freezeTopic(page);
	await shot(
		page,
		'notifications',
		'06-this-device',
		'This device registered: its name and kids are editable'
	);

	await setScenario('push_off');
	await open(page);
	await freezeTopic(page);
	await shot(
		page,
		'notifications',
		'07-web-push-off',
		'Web Push not configured on the server: devices listed, ntfy still on'
	);
	await setScenario('ntfy_off');
	await open(page);
	await shot(
		page,
		'notifications',
		'08-ntfy-off-stale-topic',
		'ntfy not configured but a topic is stored: remove it'
	);
	await setScenario('channels_off');
	await must(ana, 'DELETE', '/v1/me/ntfy');
	await open(page);
	await shot(page, 'notifications', '09-both-off', 'Neither channel configured on the server');
});

test.describe('on an iPhone outside the Home Screen', () => {
	test.use({ userAgent: IPHONE_UA });

	test('ios', async ({ page }) => {
		await signInAs(page, ANA);
		await open(page);
		await shot(
			page,
			'notifications',
			'10-iphone-add-to-home-screen',
			'iPhone Safari: add the app to the Home Screen first'
		);
	});
});
