import { MAX_PASSWORD_LEN, MIN_PASSWORD_LEN } from './constants';

export type PasswordProblem = { field: 'password' | 'confirm'; message: string };

// Code points, as the API counts them; String.length would count an emoji twice.
export function passwordLength(password: string): number {
	return [...password].length;
}

export function passwordProblem(password: string, confirm: string): PasswordProblem | null {
	const n = passwordLength(password);
	if (n < MIN_PASSWORD_LEN) {
		return { field: 'password', message: `Use at least ${MIN_PASSWORD_LEN} characters` };
	}
	if (n > MAX_PASSWORD_LEN) {
		return { field: 'password', message: `Use at most ${MAX_PASSWORD_LEN} characters` };
	}
	if (password !== confirm) return { field: 'confirm', message: 'The passwords do not match' };
	return null;
}
