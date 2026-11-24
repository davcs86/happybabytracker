import { describe, expect, it } from 'vitest';
import { PUSH_LABEL_MAX_LEN } from '../constants';
import { DEVICE_FIELD_MESSAGES, readDeviceForm } from './notification-form';

const form = (fields: [string, string][]) => {
	const data = new FormData();
	for (const [k, v] of fields) data.append(k, v);
	return data;
};

describe('readDeviceForm', () => {
	it('reads the label and the kids that are switched on, in order', () => {
		expect(
			readDeviceForm(
				form([
					['label', 'Ana phone'],
					['kid', 'k-1'],
					['kid', 'k-2']
				])
			)
		).toEqual({
			values: { label: 'Ana phone', kids: ['k-1', 'k-2'] },
			input: { label: 'Ana phone', kids: ['k-1', 'k-2'] },
			field: null
		});
	});

	it('reads no kids when every toggle is off', () => {
		expect(readDeviceForm(form([['label', 'Phone']])).input).toEqual({
			label: 'Phone',
			kids: []
		});
	});

	it('sends a repeated kid once', () => {
		expect(
			readDeviceForm(
				form([
					['label', 'Phone'],
					['kid', 'k-1'],
					['kid', 'k-1']
				])
			).input?.kids
		).toEqual(['k-1']);
	});

	it('trims the label and keeps the typed text for the form', () => {
		const r = readDeviceForm(form([['label', '  Kitchen tablet  ']]));
		expect(r.input?.label).toBe('Kitchen tablet');
		expect(r.values.label).toBe('  Kitchen tablet  ');
	});

	it('accepts a label of the maximum length counted in characters, not bytes', () => {
		const label = '\u{1F4F1}'.repeat(PUSH_LABEL_MAX_LEN);
		expect(readDeviceForm(form([['label', label]])).input?.label).toBe(label);
		expect(readDeviceForm(form([['label', 'a'.repeat(PUSH_LABEL_MAX_LEN)]])).field).toBeNull();
	});

	it.each([
		['one character over', 'a'.repeat(PUSH_LABEL_MAX_LEN + 1)],
		['one emoji over', '\u{1F4F1}'.repeat(PUSH_LABEL_MAX_LEN + 1)],
		['blank', ''],
		['spaces only', '   \t ']
	])('refuses a label that is %s on the label field', (_name, label) => {
		const r = readDeviceForm(
			form([
				['label', label],
				['kid', 'k-1']
			])
		);
		expect(r.input).toBeNull();
		expect(r.field).toBe('label');
		expect(r.values).toEqual({ label, kids: ['k-1'] });
	});

	it('refuses a form without a label field', () => {
		expect(readDeviceForm(form([['kid', 'k-1']])).field).toBe('label');
	});

	it.each([['../x'], ['a b'], ['k/1'], [''], ['k'.repeat(65)]])(
		'refuses the kid %j on the kid field',
		(kid) => {
			const r = readDeviceForm(
				form([
					['label', 'Phone'],
					['kid', kid]
				])
			);
			expect(r.input).toBeNull();
			expect(r.field).toBe('kid');
		}
	);

	it('refuses a kid that is a file', () => {
		const data = form([['label', 'Phone']]);
		data.append('kid', new File(['x'], 'k-1'));
		expect(readDeviceForm(data).field).toBe('kid');
	});

	it('reports the label before a bad kid', () => {
		expect(
			readDeviceForm(
				form([
					['label', ''],
					['kid', '../x']
				])
			).field
		).toBe('label');
	});

	it('words each message without a typed value', () => {
		expect(DEVICE_FIELD_MESSAGES).toEqual({
			label: `Enter a name of 1 to ${PUSH_LABEL_MAX_LEN} characters`,
			kid: 'Choose from your children'
		});
	});
});
