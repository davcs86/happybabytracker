import { expect, test, type BrowserContext, type Page, type Worker } from '@playwright/test';

// The worker's own scope has no DOM-lib types here: describe just what the callbacks touch.
// Notifications are observed at the registration, not displayed: headless Chromium refuses to
// show any, whatever the context grants, so a displayed notification cannot be the check.
type Pending = Promise<unknown>[];
type Dispatchable = Event & { waitUntil: (p: Promise<unknown>) => void };
type Shown = { title: string; options: object | undefined };
type WorkerGlobals = {
	PushEvent: new (type: string, init: { data?: string }) => Dispatchable;
	ExtendableEvent: new (type: string) => Dispatchable;
	registration: { showNotification(title: string, options?: object): Promise<void> };
	clients: {
		openWindow(url: string): Promise<unknown>;
		matchAll(options: object): Promise<{ url: string }[]>;
	};
	WindowClient: { prototype: { focus(): Promise<unknown>; url: string } };
	caches: { keys(): Promise<string[]> };
};

const TITLE = 'Medication due for Mia';
const ICON = '/icons/icon-192.png';
const PAYLOAD = JSON.stringify({
	v: 1,
	category: 'medication',
	kid: 'k1',
	title: TITLE,
	path: '/kids/k1/medication',
	tag: 'reminder-r1'
});

async function controlledWorker(context: BrowserContext, page: Page): Promise<Worker> {
	await page.goto('/login');
	await page.evaluate(async () => {
		await navigator.serviceWorker.ready;
		if (navigator.serviceWorker.controller) return;
		await new Promise((r) =>
			navigator.serviceWorker.addEventListener('controllerchange', r, { once: true })
		);
	});
	return context.serviceWorkers()[0] ?? (await context.waitForEvent('serviceworker'));
}

// A real push event cannot be fired from here: dispatch one and wait on what the worker passes to waitUntil.
function deliverPush(worker: Worker, data: string | undefined): Promise<Shown[]> {
	return worker.evaluate(async (data) => {
		const g = globalThis as unknown as WorkerGlobals;
		const shown: Shown[] = [];
		g.registration.showNotification = (title, options) => {
			shown.push({ title, options });
			return Promise.resolve();
		};
		const pending: Pending = [];
		const event = new g.PushEvent('push', data === undefined ? {} : { data });
		event.waitUntil = (p) => void pending.push(p);
		globalThis.dispatchEvent(event);
		await Promise.all(pending);
		return shown;
	}, data);
}

test('a push with a valid payload shows its title with the reminder tag and path', async ({
	context,
	page
}) => {
	const worker = await controlledWorker(context, page);
	expect(await deliverPush(worker, PAYLOAD)).toEqual([
		{
			title: TITLE,
			options: { body: '', tag: 'reminder-r1', icon: ICON, data: { path: '/kids/k1/medication' } }
		}
	]);
});

for (const [name, data] of [
	['no payload', undefined],
	['text that is not JSON', 'Medication due for Mia'],
	['a JSON array', '[]'],
	['a path to another origin', JSON.stringify({ title: TITLE, path: '//evil.example', tag: 't' })],
	['a missing tag', JSON.stringify({ title: TITLE, path: '/kids/k1/medication' })]
] as const) {
	test(`a push with ${name} shows the generic notification without a name`, async ({
		context,
		page
	}) => {
		const worker = await controlledWorker(context, page);
		// No tag key, and a click goes to the home page.
		expect(await deliverPush(worker, data)).toEqual([
			{ title: 'Reminder due', options: { body: '', icon: ICON, data: { path: '/' } } }
		]);
	});
}

// The window calls are recorded, not performed: Chrome allows them only for a real click.
// The event carries a stand-in notification: only close() and data are read.
function click(worker: Worker, datas: unknown[]) {
	return worker.evaluate(async (datas) => {
		const g = globalThis as unknown as WorkerGlobals;
		const calls: string[] = [];
		Object.defineProperty(g.clients, 'openWindow', {
			configurable: true,
			value: (url: string) => (calls.push(`open ${url}`), Promise.resolve(null))
		});
		g.WindowClient.prototype.focus = function () {
			calls.push(`focus ${new URL(this.url).pathname}`);
			return Promise.resolve(this);
		};
		let closed = 0;
		for (const data of datas) {
			const pending: Pending = [];
			const event = new g.ExtendableEvent('notificationclick');
			Object.defineProperty(event, 'notification', {
				value: { data, close: () => void closed++ }
			});
			event.waitUntil = (p) => void pending.push(p);
			globalThis.dispatchEvent(event);
			await Promise.all(pending);
		}
		return { calls, closed };
	}, datas);
}

test('a click on a notification whose path is not a same-origin path opens and focuses nothing', async ({
	context,
	page
}) => {
	const worker = await controlledWorker(context, page);
	const bad = [
		{ path: '//evil.example' },
		{ path: 'https://evil.example/x' },
		{ path: '/\\evil.example' },
		{ path: 'kids' },
		{ path: 5 },
		{},
		null,
		'/kids/k1/medication'
	];
	const result = await click(worker, bad);
	expect(result.calls).toEqual([]);
	expect(result.closed).toBe(bad.length);
});

test('a click opens the path when no window shows it, and focuses the window that does', async ({
	context,
	page
}) => {
	const worker = await controlledWorker(context, page);
	const opened = await click(worker, [{ path: '/kids/k1/medication' }]);
	expect(opened.calls).toEqual(['open /kids/k1/medication']);
	expect(opened.closed).toBe(1);

	const focused = await click(worker, [{ path: '/login' }]);
	expect(focused.calls).toEqual(['focus /login']);
});

test('a window on the same path with a fragment is focused, not duplicated', async ({
	context,
	page
}) => {
	const worker = await controlledWorker(context, page);
	await page.evaluate(() => (location.hash = 'top'));
	expect(page.url()).toMatch(/\/login#top$/);
	expect((await click(worker, [{ path: '/login' }])).calls).toEqual(['focus /login']);
});

test('installing the worker leaves the Cache API empty', async ({ context, page }) => {
	const worker = await controlledWorker(context, page);
	expect(
		await worker.evaluate(() => (globalThis as unknown as WorkerGlobals).caches.keys())
	).toEqual([]);
});

test('the worker answers no fetch: an offline navigation fails in the browser', async ({
	context,
	page,
	request
}) => {
	await controlledWorker(context, page);
	const source = await (await request.get('/service-worker.js')).text();
	expect(source).not.toMatch(/onfetch|["'`]fetch["'`]/);
	expect((await page.goto('/login'))?.fromServiceWorker()).toBe(false);
	await context.setOffline(true);
	await expect(page.goto('/login')).rejects.toThrow(/ERR_INTERNET_DISCONNECTED/);
	await context.setOffline(false);
});
