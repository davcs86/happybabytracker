import { expect, test, type BrowserContext, type Page } from '@playwright/test';
import { expectAccessible, expectNotFoundPage, signIn, submit, untilHydrated } from './helpers.js';
import {
	ANA,
	CLEO,
	DAN,
	IVY,
	LEO,
	MIA,
	UI_ORIGIN,
	fakePushKeys,
	receivedRequests,
	resetStub,
	setScenario,
	stubAs
} from './stub-api.js';

// The notifications page against the stub, with the browser's PushManager replaced by a fake.
const PAGE = '/account/notifications';
const SUBSCRIBE = `${PAGE}/subscribe`;
const DEVICES = '/v1/me/push-devices';
const DEVICE_COOKIE = 'hbt_push_device';
const OFF_SENTENCE = 'Web Push is not configured on this server.';
const OTHER_KEY_SENTENCE = 'Turn notifications off and on again on this device.';
const RESEND_FAILED =
	'This device could not be checked with the server. Reload the page to try again.';
const IOS_SENTENCE =
	'On iPhone and iPad, add this app to your Home Screen first, then open it from there to turn on notifications';
const NTFY_SENTENCE =
	"ntfy keeps each message for up to 72 hours on its server, so your child's name is stored there in plain text for that long";
const TOPIC = /^hbt-[0-9A-Za-z]{22}$/;
const SIGN_OUT_SENTENCE =
	'Signing out of this app does not stop notifications on this phone or browser: turn them off first with Turn off ntfy or Turn off on this device.';
const NEW_TOPIC_CONFIRM = `The current topic stops working. Subscribe to the new topic in the ntfy app again. ${SIGN_OUT_SENTENCE}`;
const STALE_TOPIC_SENTENCE = 'A topic from earlier is still stored for your account. Remove it.';
const IPHONE_UA =
	'Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1';
const MAC_UA =
	'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15';

type Sub = { endpoint: string; keys: { p256dh: string; auth: string } };

test.beforeEach(async () => {
	await resetStub();
});

type FakeState = { visible?: boolean; key?: string; endpoint?: string; noOptions?: boolean };

// A browser subscription that survives navigations (localStorage) and records the key it was made with.
async function fakePush(context: BrowserContext, n = 'browser'): Promise<Sub> {
	const sub = { endpoint: `https://push.example/send/${n}-endpoint-secret`, keys: fakePushKeys() };
	await context.grantPermissions(['notifications']);
	await context.addInitScript((sub: Sub) => {
		if (typeof PushManager === 'undefined') return;
		// The headless shell reports 'denied' whatever the context grants: report the grant made above.
		Object.defineProperty(Notification, 'permission', { get: () => 'granted' });
		Notification.requestPermission = async () => 'granted';
		const read = () => {
			const raw = localStorage.getItem('fake-push');
			return raw ? (JSON.parse(raw) as FakeState) : null;
		};
		const bump = (name: string) =>
			localStorage.setItem(name, String(Number(localStorage.getItem(name) ?? 0) + 1));
		const make = (s: FakeState) => {
			const endpoint = s.endpoint ?? sub.endpoint;
			const key = s.key ? Uint8Array.from(atob(s.key), (c) => c.charCodeAt(0)).buffer : null;
			return {
				endpoint,
				...(s.noOptions
					? {}
					: { options: { userVisibleOnly: s.visible ?? true, applicationServerKey: key } }),
				toJSON: () => ({ endpoint, expirationTime: null, keys: sub.keys }),
				unsubscribe: async () => {
					localStorage.removeItem('fake-push');
					bump('fake-push-unsubscribed');
					return true;
				}
			} as unknown as PushSubscription;
		};
		PushManager.prototype.subscribe = async function (options?: PushSubscriptionOptionsInit) {
			const key = btoa(
				String.fromCharCode(...new Uint8Array(options?.applicationServerKey as ArrayBuffer))
			);
			const existing = read();
			// As Chrome does: another key while a subscription exists is refused.
			if (existing?.key && existing.key !== key) {
				throw new DOMException('another key', 'InvalidStateError');
			}
			const state = { visible: options?.userVisibleOnly, key, endpoint: sub.endpoint };
			localStorage.setItem('fake-push', JSON.stringify(state));
			return make(state);
		};
		PushManager.prototype.getSubscription = async function () {
			bump('fake-push-gets');
			const s = read();
			return s ? make(s) : null;
		};
	}, sub);
	return sub;
}

const fakeState = async (page: Page) =>
	JSON.parse((await page.evaluate(() => localStorage.getItem('fake-push'))) ?? 'null') as FakeState;
const setFakeState = (page: Page, state: FakeState) =>
	page.evaluate((s) => localStorage.setItem('fake-push', JSON.stringify(s)), state);
const fakeCount = async (page: Page, name: string) =>
	Number((await page.evaluate((n) => localStorage.getItem(n), name)) ?? 0);
const b64 = (base64url: string) => Buffer.from(base64url, 'base64url').toString('base64');

const region = (page: Page, name: string) => page.getByRole('region', { name });
const enableButton = (page: Page) =>
	page.getByRole('button', { name: 'Turn on notifications on this device' });
const deviceCookie = async (context: BrowserContext) =>
	(await context.cookies()).find((c) => c.name === DEVICE_COOKIE);
const pushWrites = async (from = 0) =>
	(await receivedRequests())
		.slice(from)
		.filter((r) => r.method !== 'GET' && r.method !== 'DELETE' && r.path.includes('push-devices'));

async function open(page: Page) {
	await page.goto(PAGE);
	await expect(page.getByRole('heading', { name: 'Notifications', level: 1 })).toBeVisible();
}

// The enable button is enabled once the page has hydrated.
async function openHydrated(page: Page) {
	await open(page);
	await expect(enableButton(page)).toBeEnabled();
}

async function enable(page: Page) {
	const response = page.waitForResponse((r) => new URL(r.url()).pathname === SUBSCRIBE);
	await enableButton(page).click();
	expect((await response).status()).toBe(204);
	await expect(region(page, 'This device').getByLabel('Device name')).toBeVisible();
}

async function expectNoSecret(page: Page, sub: Sub) {
	const data = await page.request.get(`${PAGE}/__data.json`);
	expect(data.status()).toBe(200);
	for (const text of [await page.content(), await data.text()]) {
		for (const secret of [sub.endpoint, 'endpoint-secret', sub.keys.p256dh, sub.keys.auth]) {
			expect(text).not.toContain(secret);
		}
	}
}

test.describe('Web Push on', () => {
	test('enable subscribes with the server key, posts the subscription and lists this device', async ({
		page,
		context
	}) => {
		const sub = await fakePush(context);
		const ana = await stubAs(ANA);
		await signIn(page, ANA);
		await page.goto('/account');
		await page.getByRole('link', { name: 'Notifications' }).click();
		await expect(page).toHaveURL(new RegExp(`${PAGE}$`));
		await expect(enableButton(page)).toBeEnabled();
		await expect(page.getByText(IOS_SENTENCE)).toHaveCount(0);
		await expect(page.getByText(OFF_SENTENCE)).toHaveCount(0);
		await expectAccessible(page);

		const response = page.waitForResponse((r) => new URL(r.url()).pathname === SUBSCRIBE);
		await enableButton(page).click();
		expect((await response).status()).toBe(204);
		await expect(region(page, 'This device').getByLabel('Device name')).toHaveValue('This device');
		await expect(enableButton(page)).toHaveCount(0);

		const puts = (await receivedRequests()).filter((r) => r.method === 'PUT' && r.path === DEVICES);
		expect(puts).toHaveLength(1);
		// No label: the API names a new device "This device" and keeps a known one's name.
		expect(JSON.parse(puts[0].body ?? '')).toEqual(sub);
		const key = (await ana('GET', '/v1/push')).body.public_key as string;
		expect(await fakeState(page)).toEqual({ visible: true, key: b64(key), endpoint: sub.endpoint });

		const devices = (await ana('GET', DEVICES)).body.devices;
		expect(devices).toHaveLength(1);
		const cookie = await deviceCookie(context);
		expect(cookie).toMatchObject({
			value: devices[0].id,
			path: PAGE,
			httpOnly: true,
			sameSite: 'Strict',
			secure: true
		});
		expect(cookie?.expires).toBeGreaterThan(Date.now() / 1000 + 300 * 24 * 3600);
		await expectNoSecret(page, sub);
		await expectAccessible(page);

		// Each visit with permission granted sends the subscription again under the same name.
		await region(page, 'This device').getByLabel('Device name').fill('Kitchen tablet');
		await submit(region(page, 'This device').getByRole('button', { name: 'Save' }));
		await expect(page.getByRole('status')).toHaveText('Saved');
		const resent = page.waitForResponse((r) => new URL(r.url()).pathname === SUBSCRIBE);
		await page.reload();
		expect((await resent).status()).toBe(204);
		const after = (await receivedRequests()).filter(
			(r) => r.method === 'PUT' && r.path === DEVICES
		);
		expect(after).toHaveLength(2);
		expect(JSON.parse(after[1].body ?? '')).toEqual({ ...sub, label: 'Kitchen tablet' });
		expect((await ana('GET', DEVICES)).body.devices).toEqual([
			{ ...devices[0], label: 'Kitchen tablet' }
		]);
		await expect(region(page, 'This device').getByLabel('Device name')).toHaveValue(
			'Kitchen tablet'
		);
		await expectNoSecret(page, sub);
	});

	test('kid toggles opt this device in and out, and turning it off removes it', async ({
		page,
		context
	}) => {
		await fakePush(context);
		const ana = await stubAs(ANA);
		await signIn(page, ANA);
		await openHydrated(page);
		await enable(page);
		const here = region(page, 'This device');
		for (const kid of [MIA, IVY, LEO]) await expect(here.getByLabel(kid.name)).not.toBeChecked();

		await here.getByLabel(MIA.name).check();
		await here.getByLabel(LEO.name).check();
		await submit(here.getByRole('button', { name: 'Save' }));
		await expect(page.getByRole('status')).toHaveText('Saved');
		expect((await ana('GET', DEVICES)).body.devices[0].kids.sort()).toEqual(
			[LEO.id, MIA.id].sort()
		);
		await page.reload();
		await untilHydrated(here.getByLabel(MIA.name), 'checked');
		await expect(here.getByLabel(MIA.name)).toBeChecked();
		await expect(here.getByLabel(LEO.name)).toBeChecked();
		await expect(here.getByLabel(IVY.name)).not.toBeChecked();
		await expectAccessible(page);

		await expect(enableButton(page)).toHaveCount(0);
		await here.getByLabel(MIA.name).uncheck();
		await submit(here.getByRole('button', { name: 'Save' }));
		await expect(page.getByRole('status')).toHaveText('Saved');
		expect((await ana('GET', DEVICES)).body.devices[0].kids).toEqual([LEO.id]);

		await submit(here.getByRole('button', { name: 'Turn off on this device' }));
		await expect(enableButton(page)).toBeEnabled();
		expect((await ana('GET', DEVICES)).body).toEqual({ devices: [] });
		expect(await deviceCookie(context)).toBeUndefined();
		expect(await page.evaluate(() => localStorage.getItem('fake-push'))).toBeNull();
		// Turned off, a reload sends nothing.
		const before = (await receivedRequests()).length;
		await page.reload();
		await expect(enableButton(page)).toBeEnabled();
		expect(await pushWrites(before)).toEqual([]);
	});

	test("another parent's device and a kid without a grant are the 404 page", async ({
		page,
		context
	}) => {
		await fakePush(context);
		const dan = await stubAs(DAN);
		const dans = (
			await dan('PUT', DEVICES, {
				endpoint: 'https://push.example/send/dan',
				keys: fakePushKeys(),
				label: 'Dan'
			})
		).body.device;
		await signIn(page, ANA);
		const unknown = await expectNotFoundPage(page, '/kids/k-unknown');

		await openHydrated(page);
		await enable(page);
		const here = region(page, 'This device');
		await here
			.locator('input[name="device"]')
			.first()
			.evaluate((el, id) => {
				(el as HTMLInputElement).value = id;
			}, dans.id);
		await submit(here.getByRole('button', { name: 'Save' }));
		await expect(page.getByRole('heading', { name: 'Not found' })).toBeVisible();
		expect(await page.locator('body').innerText()).toBe(unknown);

		// This device is registered now: the re-send on load marks hydration.
		const resent = page.waitForResponse((r) => new URL(r.url()).pathname === SUBSCRIBE);
		await open(page);
		await resent;
		await here.getByLabel(MIA.name).evaluate((el) => {
			(el as HTMLInputElement).value = 'k-unknown';
		});
		await here.getByLabel(MIA.name).check();
		await submit(here.getByRole('button', { name: 'Save' }));
		await expect(page.getByRole('heading', { name: 'Not found' })).toBeVisible();
		expect(await page.locator('body').innerText()).toBe(unknown);
		expect((await dan('GET', DEVICES)).body.devices).toEqual([dans]);

		for (const id of [dans.id, 'not/an id']) {
			const resent = page.waitForResponse((r) => new URL(r.url()).pathname === SUBSCRIBE);
			await open(page);
			await resent;
			await here.locator('form[action="?/remove"] input[name="device"]').evaluate((el, id) => {
				(el as HTMLInputElement).value = id;
			}, id);
			await submit(here.getByRole('button', { name: 'Turn off on this device' }));
			await expect(page.getByRole('heading', { name: 'Not found' })).toBeVisible();
			expect(await page.locator('body').innerText()).toBe(unknown);
		}
		expect((await dan('GET', DEVICES)).body.devices).toEqual([dans]);
		expect((await (await stubAs(ANA))('GET', DEVICES)).body.devices).toHaveLength(1);
	});

	test('a registration by another parent on this browser moves the device and the cookie follows', async ({
		page,
		context
	}) => {
		await fakePush(context);
		const ana = await stubAs(ANA);
		const cleo = await stubAs(CLEO);
		await signIn(page, ANA);
		await openHydrated(page);
		await enable(page);
		const first = (await ana('GET', DEVICES)).body.devices[0].id;
		expect((await deviceCookie(context))?.value).toBe(first);

		await context.clearCookies({ name: 'hbt_session' });
		await signIn(page, CLEO);
		const before = (await receivedRequests()).length;
		await openHydrated(page);
		// Ana's device is not Cleo's: nothing is sent until Cleo turns it on.
		expect(await pushWrites(before)).toEqual([]);
		await enable(page);
		const moved = (await cleo('GET', DEVICES)).body.devices;
		expect(moved).toHaveLength(1);
		expect(moved[0].id).not.toBe(first);
		expect((await deviceCookie(context))?.value).toBe(moved[0].id);
		expect((await ana('GET', DEVICES)).body).toEqual({ devices: [] });
	});

	test('notifications refused by the browser send nothing', async ({ page }) => {
		await signIn(page, ANA);
		await page.addInitScript(() => {
			Notification.requestPermission = async () => 'denied';
		});
		await openHydrated(page);
		const before = (await receivedRequests()).length;
		await enableButton(page).click();
		await expect(page.getByRole('alert')).toHaveText(
			"Notifications are blocked for this site. Allow them in the browser's settings, then try again."
		);
		expect(await pushWrites(before)).toEqual([]);
		await expectAccessible(page);
	});

	test('a prompt dismissed without a choice says so and sends nothing', async ({ page }) => {
		await signIn(page, ANA);
		await page.addInitScript(() => {
			Notification.requestPermission = async () => 'default';
		});
		await openHydrated(page);
		const before = (await receivedRequests()).length;
		await enableButton(page).click();
		await expect(page.getByRole('alert')).toHaveText(
			'Notifications were not turned on. You can try again.'
		);
		expect(await pushWrites(before)).toEqual([]);
	});

	test('a browser subscription made under another key is replaced on enable', async ({
		page,
		context
	}) => {
		const sub = await fakePush(context);
		const ana = await stubAs(ANA);
		const key = (await ana('GET', '/v1/push')).body.public_key as string;
		await signIn(page, ANA);
		await setFakeState(page, { visible: true, key: b64(fakePushKeys().p256dh) });
		await openHydrated(page);
		await enable(page);
		expect(await fakeCount(page, 'fake-push-unsubscribed')).toBe(1);
		expect(await fakeState(page)).toEqual({ visible: true, key: b64(key), endpoint: sub.endpoint });
		const puts = (await receivedRequests()).filter((r) => r.method === 'PUT' && r.path === DEVICES);
		expect(puts.map((r) => JSON.parse(r.body ?? ''))).toEqual([sub]);
	});

	test('a subscription under another key is not sent again on load', async ({ page, context }) => {
		await fakePush(context);
		await signIn(page, ANA);
		await openHydrated(page);
		await enable(page);
		await setFakeState(page, { ...(await fakeState(page)), key: b64(fakePushKeys().p256dh) });
		const sent: string[] = [];
		page.on('request', (r) => {
			if (new URL(r.url()).pathname === SUBSCRIBE) sent.push(r.method());
		});
		const gets = await fakeCount(page, 'fake-push-gets');
		const before = (await receivedRequests()).length;
		await open(page);
		// The page has read the subscription; a send would start before this round trip returns.
		await expect.poll(() => fakeCount(page, 'fake-push-gets')).toBeGreaterThan(gets);
		await page.evaluate(() => navigator.serviceWorker.ready.then(() => undefined));
		expect(sent).toEqual([]);
		expect(await pushWrites(before)).toEqual([]);
		await expect(region(page, 'This device').getByRole('alert')).toHaveText(OTHER_KEY_SENTENCE);
	});

	test('a subscription without options is sent again on load', async ({ page, context }) => {
		await fakePush(context);
		await signIn(page, ANA);
		await openHydrated(page);
		await enable(page);
		await setFakeState(page, { ...(await fakeState(page)), noOptions: true });
		const resent = page.waitForResponse((r) => new URL(r.url()).pathname === SUBSCRIBE);
		await open(page);
		expect((await resent).status()).toBe(204);
		await expect(region(page, 'This device').getByRole('alert')).toHaveCount(0);
	});

	test('a re-send that fails on a visit has its own sentence', async ({ page, context }) => {
		await fakePush(context);
		await signIn(page, ANA);
		await openHydrated(page);
		await enable(page);
		await page.route(`**${SUBSCRIBE}`, (route) =>
			route.fulfill({ status: 502, contentType: 'application/json', body: '{}' })
		);
		await open(page);
		await expect(region(page, 'This device').getByRole('alert')).toHaveText(RESEND_FAILED);
	});

	test('a refused kid on the device form shows its message', async ({ page, context }) => {
		await fakePush(context);
		await signIn(page, ANA);
		await openHydrated(page);
		await enable(page);
		const here = region(page, 'This device');
		const mia = here.getByLabel(MIA.name);
		await mia.evaluate((el: HTMLInputElement) => (el.value = 'not an id'));
		await mia.check();
		await submit(here.getByRole('button', { name: 'Save' }));
		await expect(page.getByRole('alert')).toHaveText('Choose from your children');
	});

	test('turning on again after the cookie is gone keeps the chosen name', async ({
		page,
		context
	}) => {
		const sub = await fakePush(context);
		const ana = await stubAs(ANA);
		await signIn(page, ANA);
		await openHydrated(page);
		await enable(page);
		const here = region(page, 'This device');
		await here.getByLabel('Device name').fill('Kitchen tablet');
		await submit(here.getByRole('button', { name: 'Save' }));
		await expect(page.getByRole('status')).toHaveText('Saved');
		const before = (await ana('GET', DEVICES)).body.devices;

		await context.clearCookies({ name: DEVICE_COOKIE });
		const from = (await receivedRequests()).length;
		await openHydrated(page);
		await enable(page);
		await expect(here.getByLabel('Device name')).toHaveValue('Kitchen tablet');
		const puts = (await receivedRequests())
			.slice(from)
			.filter((r) => r.method === 'PUT' && r.path === DEVICES);
		expect(puts.map((r) => JSON.parse(r.body ?? ''))).toEqual([sub]);
		expect((await ana('GET', DEVICES)).body.devices).toEqual(before);
		expect((await deviceCookie(context))?.value).toBe(before[0].id);
	});

	test('a renewed subscription keeps the name and the kids, and the old device goes', async ({
		page,
		context
	}) => {
		const sub = await fakePush(context);
		const ana = await stubAs(ANA);
		await signIn(page, ANA);
		await openHydrated(page);
		await enable(page);
		const here = region(page, 'This device');
		await here.getByLabel('Device name').fill('Night phone');
		await here.getByLabel(MIA.name).check();
		await here.getByLabel(LEO.name).check();
		await submit(here.getByRole('button', { name: 'Save' }));
		await expect(page.getByRole('status')).toHaveText('Saved');
		const old = (await ana('GET', DEVICES)).body.devices[0];

		await setFakeState(page, { ...(await fakeState(page)), endpoint: `${sub.endpoint}-renewed` });
		const resent = page.waitForResponse((r) => new URL(r.url()).pathname === SUBSCRIBE);
		await open(page);
		expect((await resent).status()).toBe(204);
		const devices = (await ana('GET', DEVICES)).body.devices;
		expect(devices).toHaveLength(1);
		expect(devices[0].id).not.toBe(old.id);
		expect(devices[0]).toMatchObject({ label: 'Night phone' });
		expect([...devices[0].kids].sort()).toEqual([LEO.id, MIA.id].sort());
		expect((await deviceCookie(context))?.value).toBe(devices[0].id);
		await expect(here.getByLabel(MIA.name)).toBeChecked();
		await expect(here.getByLabel(LEO.name)).toBeChecked();
		await expect(here.getByLabel(IVY.name)).not.toBeChecked();
	});

	test('the subscribe route takes JSON only', async ({ page, context }) => {
		const sub = await fakePush(context);
		await signIn(page, ANA);
		await open(page);
		const before = (await receivedRequests()).length;
		const status = await page.evaluate(
			async ({ path, body }) =>
				(
					await fetch(path, {
						method: 'POST',
						headers: { 'content-type': 'text/plain' },
						body: JSON.stringify(body)
					})
				).status,
			{ path: SUBSCRIBE, body: sub }
		);
		expect(status).toBe(415);
		expect(await pushWrites(before)).toEqual([]);
	});

	test.describe('on an iPhone outside the Home Screen', () => {
		test.use({ userAgent: IPHONE_UA });

		test('the Home Screen sentence shows instead of the enable button', async ({ page }) => {
			await signIn(page, ANA);
			await open(page);
			await expect(page.getByText(IOS_SENTENCE)).toBeVisible();
			await expect(enableButton(page)).toHaveCount(0);
			await expectAccessible(page);
		});
	});

	test.describe('on an iPad that reports a Mac', () => {
		test.use({ userAgent: MAC_UA });

		test('a touch screen shows the sentence, opened from the Home Screen it does not', async ({
			page
		}) => {
			await page.addInitScript(() => {
				Object.defineProperty(Navigator.prototype, 'maxTouchPoints', { get: () => 5 });
			});
			await signIn(page, ANA);
			await open(page);
			await expect(page.getByText(IOS_SENTENCE)).toBeVisible();

			await page.addInitScript(() => {
				Object.defineProperty(Navigator.prototype, 'standalone', { get: () => true });
			});
			await openHydrated(page);
			await expect(page.getByText(IOS_SENTENCE)).toHaveCount(0);
		});

		test('without a touch screen it is a Mac', async ({ page }) => {
			await signIn(page, ANA);
			await openHydrated(page);
			await expect(page.getByText(IOS_SENTENCE)).toHaveCount(0);
		});
	});
});

test.describe('Web Push off', () => {
	test('the sentence shows, no enable button, an earlier device can be removed and no push write is sent', async ({
		page,
		context
	}) => {
		const sub = await fakePush(context);
		const ana = await stubAs(ANA);
		const earlier = (await ana('PUT', DEVICES, { ...sub, label: 'Old phone' })).body.device;
		await ana('PUT', `/v1/kids/${MIA.id}/push-devices/${earlier.id}`);
		await context.addCookies([
			{ name: DEVICE_COOKIE, value: earlier.id, url: `${UI_ORIGIN}${PAGE}` }
		]);
		await setScenario('push_off');
		await signIn(page, ANA);
		// A subscription and a matching cookie are there, yet nothing is sent while Web Push is off.
		await page.evaluate(() => localStorage.setItem('fake-push', '{}'));
		const before = (await receivedRequests()).length;
		await open(page);
		await expect(page.getByText(OFF_SENTENCE)).toBeVisible();
		await expect(enableButton(page)).toHaveCount(0);
		await expect(page.getByText(IOS_SENTENCE)).toHaveCount(0);
		await expect(page.getByLabel('Device name')).toHaveCount(0);
		const list = page.getByRole('list', { name: 'Your devices' });
		await expect(list.getByRole('listitem')).toHaveCount(1);
		await expect(list.getByRole('listitem')).toContainText('Old phone');
		await expectAccessible(page);
		await expectNoSecret(page, sub);

		const direct = await page.evaluate(
			async ({ path, sub }) => {
				const res = await fetch(path, {
					method: 'POST',
					headers: { 'content-type': 'application/json' },
					body: JSON.stringify(sub)
				});
				return { status: res.status, text: await res.text() };
			},
			{ path: SUBSCRIBE, sub }
		);
		expect(direct.status).toBe(409);
		expect(JSON.parse(direct.text)).toEqual({ message: OFF_SENTENCE });

		await submit(list.getByRole('button', { name: /Remove/ }));
		await expect(page.getByRole('list', { name: 'Your devices' })).toHaveCount(0);
		expect((await ana('GET', DEVICES)).body).toEqual({ devices: [] });
		expect(await pushWrites(before)).toEqual([]);
		await expectAccessible(page);
	});
});

test.describe('ntfy', () => {
	test('absent while ntfy is off', async ({ page }) => {
		await setScenario('ntfy_off');
		await signIn(page, ANA);
		await open(page);
		await expect(page.getByRole('heading', { name: 'ntfy' })).toHaveCount(0);
		await expect(page.getByText(NTFY_SENTENCE)).toHaveCount(0);
		await expectAccessible(page);
	});

	test('off with a topic stored earlier: one sentence, never the topic, and it can be removed', async ({
		page
	}) => {
		const ana = await stubAs(ANA);
		const topic = (await ana('PUT', '/v1/me/ntfy')).body.topic as string;
		await setScenario('ntfy_off');
		await signIn(page, ANA);
		await open(page);
		const ntfy = region(page, 'ntfy');
		await expect(ntfy.getByText(STALE_TOPIC_SENTENCE)).toBeVisible();
		await expect(page.getByText(NTFY_SENTENCE)).toHaveCount(0);
		await expect(ntfy.getByRole('button', { name: 'Turn on ntfy' })).toHaveCount(0);
		expect(await page.content()).not.toContain(topic);
		await expectAccessible(page);

		await submit(ntfy.getByRole('button', { name: 'Remove the stored topic' }));
		await expect(page.getByRole('heading', { name: 'ntfy' })).toHaveCount(0);
		expect((await ana('GET', '/v1/me/ntfy')).body).toEqual({ enabled: false, has_topic: false });
		await setScenario('normal');
		expect((await ana('GET', '/v1/me/ntfy')).body.topic).toBeNull();
	});

	test('on: the topic and the 72-hour sentence, kid toggles, a new topic, then off', async ({
		page,
		context
	}) => {
		await fakePush(context);
		const ana = await stubAs(ANA);
		await signIn(page, ANA);
		await openHydrated(page);
		const ntfy = region(page, 'ntfy');
		await expect(ntfy.getByText(NTFY_SENTENCE)).toBeVisible();
		await expect(ntfy.getByText(SIGN_OUT_SENTENCE, { exact: true })).toBeVisible();
		await submit(ntfy.getByRole('button', { name: 'Turn on ntfy' }));
		const state = (await ana('GET', '/v1/me/ntfy')).body;
		expect(state.topic).toMatch(TOPIC);
		await expect(ntfy.getByText(state.topic)).toBeVisible();
		await expect(ntfy.getByText(state.server)).toBeVisible();
		await expect(ntfy.getByText(NTFY_SENTENCE)).toBeVisible();
		await expectAccessible(page);

		await ntfy.getByLabel(IVY.name).check();
		await submit(ntfy.getByRole('button', { name: 'Save' }));
		await expect(page.getByRole('status')).toHaveText('Saved');
		expect((await ana('GET', '/v1/me/ntfy')).body.kids).toEqual([IVY.id]);
		await ntfy.getByLabel(IVY.name).uncheck();
		await ntfy.getByLabel(LEO.name).check();
		await submit(ntfy.getByRole('button', { name: 'Save' }));
		expect((await ana('GET', '/v1/me/ntfy')).body.kids).toEqual([LEO.id]);

		await expect(ntfy.getByText(SIGN_OUT_SENTENCE, { exact: true })).toBeVisible();
		// The confirm button exists only inside the opened details.
		const confirm = ntfy.getByRole('button', { name: 'Yes, create a new topic' });
		await expect(confirm).toBeHidden();
		await ntfy.locator('summary', { hasText: 'New topic' }).click();
		await expect(ntfy.getByText(NEW_TOPIC_CONFIRM)).toBeVisible();
		await expectAccessible(page);
		expect((await ana('GET', '/v1/me/ntfy')).body.topic).toBe(state.topic);
		await submit(confirm);
		const fresh = (await ana('GET', '/v1/me/ntfy')).body.topic;
		expect(fresh).toMatch(TOPIC);
		expect(fresh).not.toBe(state.topic);
		await expect(ntfy.getByText(fresh)).toBeVisible();
		await expect(ntfy.getByText(state.topic)).toHaveCount(0);
		// A second tap would replace the topic just subscribed to.
		await expect(confirm).toBeHidden();
		await expectAccessible(page);

		await submit(ntfy.getByRole('button', { name: 'Turn off ntfy' }));
		await expect(ntfy.getByText(fresh)).toHaveCount(0);
		await expect(ntfy.getByRole('button', { name: 'Turn on ntfy' })).toBeVisible();
		expect((await ana('GET', '/v1/me/ntfy')).body).toMatchObject({ topic: null, kids: [] });
	});
});

test.describe('without JavaScript', () => {
	test.use({ javaScriptEnabled: false });

	test('the page renders and the label, kid toggles, remove and ntfy work', async ({
		page,
		context
	}) => {
		const sub = {
			endpoint: 'https://push.example/send/nojs-endpoint-secret',
			keys: fakePushKeys()
		};
		const ana = await stubAs(ANA);
		const device = (await ana('PUT', DEVICES, { ...sub, label: 'Phone' })).body.device;
		const other = (
			await ana('PUT', DEVICES, {
				endpoint: 'https://push.example/send/other',
				keys: fakePushKeys(),
				label: 'Laptop'
			})
		).body.device;
		await context.addCookies([
			{ name: DEVICE_COOKIE, value: device.id, url: `${UI_ORIGIN}${PAGE}` }
		]);
		await signIn(page, ANA);
		await open(page);
		await expect(enableButton(page)).toHaveCount(0);
		await expectNoSecret(page, sub);

		const here = region(page, 'This device');
		await here.getByLabel('Device name').fill('Night phone');
		await here.getByLabel(MIA.name).check();
		await submit(here.getByRole('button', { name: 'Save' }));
		await expect(page.getByRole('status')).toHaveText('Saved');
		expect(
			(await ana('GET', DEVICES)).body.devices.find((d: { id: string }) => d.id === device.id)
		).toMatchObject({ label: 'Night phone', kids: [MIA.id] });

		await here.getByLabel('Device name').fill('');
		await submit(here.getByRole('button', { name: 'Save' }));
		await expect(here.getByLabel('Device name')).toHaveAttribute('aria-invalid', 'true');
		await expect(here.getByLabel('Device name')).toHaveAccessibleDescription(
			/Enter a name of 1 to 40 characters/
		);

		const others = page.getByRole('list', { name: 'Your other devices' });
		await submit(others.getByRole('button', { name: /Remove/ }));
		await expect(page.getByText('Laptop')).toHaveCount(0);
		expect((await ana('GET', DEVICES)).body.devices.map((d: { id: string }) => d.id)).not.toContain(
			other.id
		);

		await submit(here.getByRole('button', { name: 'Turn off on this device' }));
		expect((await ana('GET', DEVICES)).body).toEqual({ devices: [] });
		await expect(page.getByText('Turning on notifications needs JavaScript')).toBeVisible();
		await expect(enableButton(page)).toHaveCount(0);

		const ntfy = region(page, 'ntfy');
		await submit(ntfy.getByRole('button', { name: 'Turn on ntfy' }));
		const topic = (await ana('GET', '/v1/me/ntfy')).body.topic;
		await expect(ntfy.getByText(topic)).toBeVisible();
		await expect(ntfy.getByText(NTFY_SENTENCE)).toBeVisible();
		await ntfy.getByLabel(MIA.name).check();
		await submit(ntfy.getByRole('button', { name: 'Save' }));
		expect((await ana('GET', '/v1/me/ntfy')).body.kids).toEqual([MIA.id]);
		await expect(ntfy.getByRole('button', { name: 'Yes, create a new topic' })).toBeHidden();
		await ntfy.locator('summary', { hasText: 'New topic' }).click();
		await expect(ntfy.getByText(NEW_TOPIC_CONFIRM)).toBeVisible();
		await submit(ntfy.getByRole('button', { name: 'Yes, create a new topic' }));
		const fresh = (await ana('GET', '/v1/me/ntfy')).body.topic;
		expect(fresh).not.toBe(topic);
		await expect(ntfy.getByText(fresh)).toBeVisible();
		await submit(ntfy.getByRole('button', { name: 'Turn off ntfy' }));
		await expect(ntfy.getByText(fresh)).toHaveCount(0);
		expect((await ana('GET', '/v1/me/ntfy')).body.topic).toBeNull();
	});

	test('a topic stored earlier while ntfy is off can be removed', async ({ page }) => {
		const ana = await stubAs(ANA);
		await ana('PUT', '/v1/me/ntfy');
		await setScenario('ntfy_off');
		await signIn(page, ANA);
		await open(page);
		await expect(region(page, 'ntfy').getByText(STALE_TOPIC_SENTENCE)).toBeVisible();
		await submit(region(page, 'ntfy').getByRole('button', { name: 'Remove the stored topic' }));
		await expect(page.getByText(STALE_TOPIC_SENTENCE)).toHaveCount(0);
		expect((await ana('GET', '/v1/me/ntfy')).body).toEqual({ enabled: false, has_topic: false });
	});
});
