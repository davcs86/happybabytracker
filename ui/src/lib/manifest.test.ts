import { readFileSync, existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const root = resolve(import.meta.dirname, '../..');
const staticFile = (p: string) => resolve(root, 'static', p.replace(/^\//, ''));
const manifest = JSON.parse(readFileSync(staticFile('manifest.webmanifest'), 'utf8'));
const appHtml = readFileSync(resolve(root, 'src/app.html'), 'utf8');
const HEX = /^#[0-9a-f]{6}$/i;
const PNG_SIGNATURE = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);

type Icon = { src: string; sizes: string; type: string; purpose?: string };

function pngSize(path: string) {
	const b = readFileSync(path);
	expect(b.subarray(0, 8).equals(PNG_SIGNATURE)).toBe(true);
	expect(b.subarray(12, 16).toString('ascii')).toBe('IHDR');
	return `${b.readUInt32BE(16)}x${b.readUInt32BE(20)}`;
}

describe('web app manifest', () => {
	it('declares the install fields', () => {
		expect(manifest).toMatchObject({
			name: 'Happy Baby Tracker',
			short_name: 'HappyBaby',
			start_url: '/',
			scope: '/',
			display: 'standalone',
			lang: 'en'
		});
		expect(manifest.theme_color).toMatch(HEX);
		expect(manifest.background_color).toMatch(HEX);
	});

	it('lists a 192, a 512 and a maskable 512 icon, each a PNG of the declared size', () => {
		const icons: Icon[] = manifest.icons;
		const find = (sizes: string, maskable: boolean) =>
			icons.find((i) => i.sizes === sizes && (i.purpose === 'maskable') === maskable);
		for (const icon of [find('192x192', false), find('512x512', false), find('512x512', true)]) {
			expect(icon).toBeDefined();
			expect(icon?.type).toBe('image/png');
			const file = staticFile(icon!.src);
			expect(existsSync(file)).toBe(true);
			expect(pngSize(file)).toBe(icon!.sizes);
		}
	});

	it('keeps the SVG source of the icons', () => {
		expect(existsSync(staticFile('icons/icon.svg'))).toBe(true);
	});
});

describe('app.html install metadata', () => {
	it('links the manifest', () => {
		expect(appHtml).toMatch(/<link rel="manifest" href="[^"]*manifest\.webmanifest"/);
	});

	it('sets theme-color in light and dark', () => {
		expect(appHtml).toMatch(/<meta name="theme-color" content="#[0-9a-f]{6}"[^>]*light/i);
		expect(appHtml).toMatch(/<meta name="theme-color" content="#[0-9a-f]{6}"[^>]*dark/i);
	});

	it('links an apple touch icon and the standalone meta', () => {
		expect(appHtml).toContain('<link rel="apple-touch-icon" href="/icons/icon-192.png"');
		expect(appHtml).toContain('<meta name="apple-mobile-web-app-capable" content="yes"');
		expect(appHtml).toContain('name="apple-mobile-web-app-title"');
	});
});
