import { expect, test, type BrowserContext, type Page, type Worker } from '@playwright/test';

// The worker's own scope has no DOM-lib types here: describe just what the callbacks touch.
type Pending = Promise<unknown>[];
type Dispatchable = Event & { waitUntil: (p: Promise<unknown>) => void };
type WorkerGlobals = {
	PushEvent: new (type: string, init: { data?: string }) => Dispatchable;
	NotificationEvent: new (type: string, init: { notification: unknown }) => Dispatchable;
	registration: {
		showNotification(title: string, options?: object): Promise<void>;
		getNotifications(filter?: {
			tag?: string;
		}): Promise<{ title: string; body: string; tag: string; icon: string; data: unknown }[]>;
	};
	clients: { openWindow(url: string): Promise<unknown> };
	WindowClient: { prototype: { focus(): Promise<unknown>; url: string } };
	caches: { keys(): Promise<string[]> };
};

const TITLE = 'Medication due for Mia';
const PAYLOAD = JSON.stringify({
	v: 1,
	category: 'medication',
	kid: 'k1',
	title: TITLE,
	path: '/kids/k1/medication',
	tag: 'reminder-r1'
});

async function controlledWorker(context: BrowserContext, page: Page): Promise<Worker> {
	await context.grantPermissions(['notifications']);
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
function deliverPush(worker: Worker, data: string | undefined) {
	return worker.evaluate(async (data) => {
		const g = globalThis as unknown as WorkerGlobals;
		const pending: Pending = [];
		const event = new g.PushEvent('push', data === undefined ? {} : { data });
		event.waitUntil = (p) => void pending.push(p);
		globalThis.dispatchEvent(event);
		await Promise.all(pending);
	}, data);
}

function shown(worker: Worker) {
	return worker.evaluate(async () => {
		const list = await (globalThis as unknown as WorkerGlobals).registration.getNotifications();
		return list.map((n) => ({ title: n.title, body: n.body, tag: n.tag, data: n.data }));
	});
}

test('a push with a valid payload shows its title with the reminder tag and path', async ({
	context,
	page
}) => {
	const worker = await controlledWorker(context, page);
	await deliverPush(worker, PAYLOAD);
	expect(await shown(worker)).toEqual([
		{ title: TITLE, body: '', tag: 'reminder-r1', data: { path: '/kids/k1/medication' } }
	]);
	const icon = await worker.evaluate(
		async () =>
			(await (globalThis as unknown as WorkerGlobals).registration.getNotifications())[0].icon
	);
	expect(new URL(icon).pathname).toBe('/icons/icon-192.png');
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
		await deliverPush(worker, data);
		const list = await shown(worker);
		expect(list).toHaveLength(1);
		expect(list[0].title).toBe('Reminder due');
		expect(list[0].body).toBe('');
	});
}

// The window calls are recorded, not performed: Chrome allows them only for a real click.
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
		for (const [i, data] of datas.entries()) {
			const tag = `click-${i}`;
			await g.registration.showNotification('t', { tag, data });
			const [notification] = await g.registration.getNotifications({ tag });
			const pending: Pending = [];
			const event = new g.NotificationEvent('notificationclick', { notification });
			event.waitUntil = (p) => void pending.push(p);
			globalThis.dispatchEvent(event);
			await Promise.all(pending);
			if ((await g.registration.getNotifications({ tag })).length === 0) closed++;
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
	expect(source).not.toMatch(/["'`]fetch["'`]/);
	await context.setOffline(true);
	await expect(page.goto('/login')).rejects.toThrow(/ERR_INTERNET_DISCONNECTED/);
	await context.setOffline(false);
});
