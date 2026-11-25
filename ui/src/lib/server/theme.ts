import { THEMES, type Theme } from '../constants';
import { RETURN_PATH_PATTERN } from './constants';

export const parseTheme = (raw: unknown): Theme | null => THEMES.find((t) => t === raw) ?? null;

export const themeOrAuto = (raw: unknown): Theme => parseTheme(raw) ?? 'auto';

export const returnPath = (raw: unknown): string =>
	typeof raw === 'string' && RETURN_PATH_PATTERN.test(raw) ? raw : '/';
