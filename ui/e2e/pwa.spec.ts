import { expect, test } from '@playwright/test';

// No session cookie: the Node adapter serves static files before the hooks run.
test('the manifest and its icon are public static files with the right types', async ({
	request
}) => {
	const manifest = await request.get('/manifest.webmanifest', { maxRedirects: 0 });
	expect(manifest.status()).toBe(200);
	expect(manifest.headers()['content-type']).toContain('application/manifest+json');
	expect((await manifest.json()).start_url).toBe('/');

	const icon = await request.get('/icons/icon-192.png', { maxRedirects: 0 });
	expect(icon.status()).toBe(200);
	expect(icon.headers()['content-type']).toBe('image/png');
});

test('the service worker is a public static JavaScript file served without a redirect', async ({
	request
}) => {
	const worker = await request.get('/service-worker.js', { maxRedirects: 0 });
	expect(worker.status()).toBe(200);
	expect(worker.headers()['content-type']).toMatch(/javascript/);
});

test('the public login page links the manifest and the touch icon', async ({ page }) => {
	await page.goto('/login');
	await expect(page.locator('head link[rel="manifest"]')).toHaveAttribute(
		'href',
		/manifest\.webmanifest$/
	);
	await expect(page.locator('head link[rel="apple-touch-icon"]')).toHaveAttribute(
		'href',
		/\/icons\/icon-192\.png$/
	);
});
