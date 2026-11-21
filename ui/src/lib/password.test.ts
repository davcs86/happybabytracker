import { describe, expect, it } from 'vitest';
import { passwordLength, passwordProblem } from './password';

describe('passwordLength', () => {
	it('counts code points, not UTF-16 units', () => {
		expect(passwordLength('\u{1F600}'.repeat(12))).toBe(12);
		expect(passwordLength('abc')).toBe(3);
	});
});

describe('passwordProblem', () => {
	const ok = 'a'.repeat(12);

	it('accepts 12 and 256 characters', () => {
		expect(passwordProblem(ok, ok)).toBeNull();
		expect(passwordProblem('b'.repeat(256), 'b'.repeat(256))).toBeNull();
		expect(passwordProblem('\u{1F600}'.repeat(12), '\u{1F600}'.repeat(12))).toBeNull();
	});

	it('refuses 11 characters, including 11 multibyte ones', () => {
		expect(passwordProblem('a'.repeat(11), 'a'.repeat(11))?.field).toBe('password');
		// 22 UTF-16 units but 11 code points.
		const emoji = '\u{1F600}'.repeat(11);
		expect(passwordProblem(emoji, emoji)?.field).toBe('password');
	});

	it('refuses 257 characters', () => {
		expect(passwordProblem('c'.repeat(257), 'c'.repeat(257))?.field).toBe('password');
	});

	it('refuses a mismatched confirmation', () => {
		expect(passwordProblem(ok, ok + 'x')).toEqual({
			field: 'confirm',
			message: 'The passwords do not match'
		});
	});
});
