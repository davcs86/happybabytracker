import { describe, expect, it } from 'vitest';
import {
	ageHeader,
	correctedAgeLabel,
	coverageHint,
	formatLength,
	formatPercentile,
	formatWeight,
	formatZ
} from './growth';

describe('formatWeight', () => {
	it.each([
		[3200, 'kg', '3.2 kg'],
		[3000, 'kg', '3 kg'],
		[3215, 'kg', '3.215 kg'],
		[3205, 'kg', '3.205 kg'],
		[250000, 'kg', '250 kg']
	] as const)('%i g in %s is %s', (grams, unit, want) => {
		expect(formatWeight({ grams, ozHundredths: 0 }, unit)).toBe(want);
	});

	it.each([
		[11300, '7 lb 1 oz'],
		[13150, '8 lb 3.5 oz'],
		[12800, '8 lb 0 oz'],
		[800, '0 lb 8 oz'],
		// Ounces to one decimal, a tie rounding up.
		[13155, '8 lb 3.6 oz'],
		[13154, '8 lb 3.5 oz'],
		// 7 lb 15.96 oz rounds to the next pound, never "7 lb 16 oz".
		[12796, '8 lb 0 oz']
	])('%i hundredths of an ounce is %s', (ozHundredths, want) => {
		expect(formatWeight({ grams: 0, ozHundredths }, 'lb')).toBe(want);
	});
});

describe('formatLength', () => {
	it.each([
		[495, 1949, 'cm', '49.5 cm'],
		[500, 1969, 'cm', '50 cm'],
		[495, 1950, 'in', '19.5 in'],
		[508, 2000, 'in', '20 in'],
		[496, 1953, 'in', '19.53 in']
	] as const)('%i mm / %i hundredths in %s is %s', (mm, inHundredths, unit, want) => {
		expect(formatLength({ mm, inHundredths }, unit)).toBe(want);
	});
});

describe('formatPercentile', () => {
	it.each([
		[0, '<0.1'],
		[0.05, '<0.1'],
		[0.1, '0.1'],
		[50, '50.0'],
		[57.9, '57.9'],
		[50.25, '50.3'],
		[99.9, '99.9'],
		[99.95, '>99.9'],
		[100, '>99.9']
	])('%d is %s', (p, want) => {
		expect(formatPercentile(p)).toBe(want);
	});
});

describe('formatZ', () => {
	it.each([
		[0.207, '0.21'],
		[0.2, '0.20'],
		[-1.5, '-1.50'],
		[-0.001, '0.00'],
		[0, '0.00'],
		[3.456, '3.46']
	])('%d is %s', (z, want) => {
		expect(formatZ(z)).toBe(want);
	});
});

describe('correctedAgeLabel', () => {
	it('shows only the age lived when no correction applies', () => {
		expect(correctedAgeLabel(45, null)).toBe('1 month 14 days');
	});

	it('shows both ages when a correction applies', () => {
		expect(correctedAgeLabel(45, 12)).toBe('1 month 14 days (corrected 12 days)');
	});

	it('says before expected term for a corrected age below zero, never a negative age', () => {
		expect(correctedAgeLabel(10, -23)).toBe('10 days (before expected term)');
	});

	it('counts a corrected age of zero as a real age', () => {
		expect(correctedAgeLabel(33, 0)).toBe('1 month 2 days (corrected 0 days)');
	});
});

describe('coverageHint', () => {
	it("lists each measure in the API's order, whole months inside the API's range", () => {
		expect(
			coverageHint([
				{ measure: 'weight', fromMonths: 0, toMonths: 120 },
				{ measure: 'length', fromMonths: 0, toMonths: 228 },
				{ measure: 'head', fromMonths: 0, toMonths: 60.98 },
				{ measure: 'bmi', fromMonths: 23.5, toMonths: 240.5 }
			])
		).toBe('Months covered: weight 0–120, length 0–228, head circumference 0–60, BMI 24–240');
	});
});

describe('ageHeader', () => {
	it('shows the age alone when no correction applies', () => {
		expect(ageHeader('3 months', null, '2026-11-22')).toBe('3 months');
	});

	it('says before expected term while the due date is still ahead, never a zero age', () => {
		expect(ageHeader('20 days', '2026-12-31', '2026-11-22')).toBe('20 days (before expected term)');
	});

	it('counts the due date itself as a corrected age of zero', () => {
		expect(ageHeader('70 days', '2026-11-22', '2026-11-22')).toBe('70 days (corrected 0 days)');
	});

	it('shows the corrected age after the due date', () => {
		expect(ageHeader('4 months', '2026-10-01', '2026-11-22')).toBe(
			'4 months (corrected 1 month 21 days)'
		);
	});
});
