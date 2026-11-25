import { THEMES, type Theme } from './constants';

export const THEME_STORAGE_KEY = 'hbt_theme';

export const isTheme = (v: unknown): v is Theme => THEMES.some((t) => t === v);

// Applies at once and keeps the other tabs and the cookie in step.
export function applyTheme(theme: Theme): void {
	const d = document.documentElement;
	d.setAttribute('data-theme', theme);
	d.setAttribute(
		'data-resolved',
		theme === 'auto'
			? matchMedia('(prefers-color-scheme: dark)').matches
				? 'dark'
				: 'light'
			: theme
	);
	try {
		localStorage.setItem(THEME_STORAGE_KEY, theme);
	} catch {
		// Storage is a convenience: the cookie still carries the choice.
	}
	void fetch('/theme', {
		method: 'POST',
		body: new URLSearchParams({ theme, return: '/' }),
		redirect: 'manual'
	}).catch(() => {});
}
