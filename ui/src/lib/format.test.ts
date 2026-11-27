import { describe, expect, it } from 'vitest';
import {
	ageMonths,
	formatAge,
	formatAgeDays,
	formatAgo,
	formatDaysAgo,
	formatElapsed,
	formatDuration,
	formatInstant,
	formatMilk,
	formatVolume,
	todayIn
} from './format';

describe('formatAge', () => {
	const cases: [string, string, string, string][] = [
		['under 1 year shows months and days', '2026-07-08', '2026-11-20', '4 months 12 days'],
		['the same day is a lone "0 days"', '2026-11-20', '2026-11-20', '0 days'],
		['under 1 month shows days only, singular', '2026-11-19', '2026-11-20', '1 day'],
		['under 1 month shows days only, plural', '2026-10-21', '2026-11-20', '30 days'],
		['under 1 year omits days at 0, singular month', '2026-10-20', '2026-11-20', '1 month'],
		['under 1 year shows a singular day', '2026-09-19', '2026-11-20', '2 months 1 day'],
		['from 1 year shows years and months', '2024-08-20', '2026-11-20', '2 years 3 months'],
		['from 1 year omits months at 0, singular year', '2025-11-20', '2026-11-20', '1 year'],
		['from 1 year never shows days', '2025-11-01', '2026-11-20', '1 year'],
		['from 1 year shows a singular month', '2024-10-01', '2026-11-20', '2 years 1 month'],
		['31 Jan + 1 month clamps to 28 Feb', '2026-01-31', '2026-02-28', '1 month'],
		['31 Jan clamped to 28 Feb, then 1 day', '2026-01-31', '2026-03-01', '1 month 1 day'],
		['a birth date after today shows 0 days', '2026-11-21', '2026-11-20', '0 days']
	];
	for (const [rule, birth, today, want] of cases) {
		it(`${rule}: ${birth} on ${today} is "${want}"`, () => {
			expect(formatAge(birth, today)).toBe(want);
		});
	}

	describe('29 Feb birth dates on non-leap years (29 Feb + 12 months = 28 Feb)', () => {
		const leap: [string, string, string][] = [
			['28 Feb of the next year is exactly 1 year', '2025-02-28', '1 year'],
			['27 Feb is 11 months (to 29 Jan) and 29 days', '2025-02-27', '11 months 29 days'],
			['1 Mar is 1 year; from 1 year days are not shown', '2025-03-01', '1 year'],
			['28 Feb two years on is 2 years', '2026-02-28', '2 years'],
			['29 Mar is 1 year 1 month (29 Feb + 13 months = 29 Mar)', '2025-03-29', '1 year 1 month'],
			['28 Mar is still 1 year (29 Mar not reached)', '2025-03-28', '1 year'],
			['29 Feb of the next leap year is 4 years', '2028-02-29', '4 years'],
			['29 Mar of the birth year is 1 month', '2024-03-29', '1 month'],
			['28 Mar of the birth year is 28 days', '2024-03-28', '28 days']
		];
		for (const [rule, today, want] of leap) {
			it(`${rule}: 2024-02-29 on ${today} is "${want}"`, () => {
				expect(formatAge('2024-02-29', today)).toBe(want);
			});
		}
	});
});

describe('todayIn', () => {
	const now = new Date('2026-11-20T10:30:00Z');

	it('returns YYYY-MM-DD in the given zone', () => {
		expect(todayIn('UTC', now)).toBe('2026-11-20');
	});

	it('is the next day east of UTC+13:30 and the previous day west of UTC-10:30', () => {
		expect(todayIn('Pacific/Kiritimati', now)).toBe('2026-11-21');
		expect(todayIn('Pacific/Pago_Pago', now)).toBe('2026-11-19');
	});

	it('formats current IANA names that ICU only lists under older ones', () => {
		// 10:30Z is 16:00 in Kolkata (+05:30), 12:30 in Kyiv (+02:00), 17:30 in Ho Chi Minh (+07:00).
		const late = new Date('2026-11-20T20:00:00Z');
		expect(todayIn('Asia/Kolkata', now)).toBe('2026-11-20');
		expect(todayIn('Asia/Kolkata', late)).toBe('2026-11-21');
		expect(todayIn('Europe/Kyiv', new Date('2026-11-20T22:30:00Z'))).toBe('2026-11-21');
		expect(todayIn('Europe/Kyiv', new Date('2026-11-20T21:30:00Z'))).toBe('2026-11-20');
		expect(todayIn('Asia/Ho_Chi_Minh', new Date('2026-11-20T17:30:00Z'))).toBe('2026-11-21');
		expect(todayIn('Asia/Ho_Chi_Minh', new Date('2026-11-20T16:30:00Z'))).toBe('2026-11-20');
	});

	it('defaults to the current instant', () => {
		expect(todayIn('UTC')).toMatch(/^\d{4}-\d{2}-\d{2}$/);
	});
});

describe('formatInstant', () => {
	it('shows an RFC 3339 instant as a UTC date and minute', () => {
		expect(formatInstant('2026-11-21T09:05:59Z')).toBe('2026-11-21 09:05 UTC');
	});
});

describe('formatVolume', () => {
	const bottle = { ml: 120, floz_hundredths: 406 };

	it('shows whole millilitres', () => {
		expect(formatVolume(bottle, 'ml')).toBe('120 ml');
	});

	it('shows fluid ounces with up to two decimals and no trailing zeros', () => {
		expect(formatVolume(bottle, 'floz')).toBe('4.06 fl oz');
		expect(formatVolume({ ml: 133, floz_hundredths: 450 }, 'floz')).toBe('4.5 fl oz');
		expect(formatVolume({ ml: 118, floz_hundredths: 400 }, 'floz')).toBe('4 fl oz');
		expect(formatVolume({ ml: 1, floz_hundredths: 3 }, 'floz')).toBe('0.03 fl oz');
	});
});

describe('formatDuration', () => {
	it.each([
		[0, '0 s'],
		[59, '59 s'],
		[60, '1 min'],
		[1023, '17 min'],
		[3600, '1 h'],
		[3660, '1 h 1 min'],
		[26_400, '7 h 20 min'],
		[86_400, '24 h']
	])('%i seconds is %s', (seconds, want) => {
		expect(formatDuration(seconds)).toBe(want);
	});

	it('never shows a negative or fractional duration', () => {
		expect(formatDuration(-5)).toBe('0 s');
		expect(formatDuration(59.9)).toBe('59 s');
	});
});

describe('ageMonths', () => {
	// The cases of TestAgeMonths in api/internal/kidtime/kidtime_test.go, with now as the kid's local date.
	const cases: [string, string, string, number][] = [
		['day before month', '2026-08-31', '2026-09-30', 0],
		['month turns', '2026-08-31', '2026-10-01', 1],
		['birth day itself', '2026-08-31', '2026-08-31', 0],
		['leap day, day before first birthday', '2024-02-29', '2025-02-28', 11],
		['leap day, first birthday', '2024-02-29', '2025-03-01', 12],
		['leap day, next leap day', '2024-02-29', '2028-02-29', 48],
		['born tomorrow', '2026-06-02', '2026-06-01', 0],
		['born next year', '2027-01-15', '2026-06-01', 0],
		['one year', '2025-06-01', '2026-06-01', 12],
		['the kid zone already turned the day', '2026-06-02', '2026-06-02', 0],
		['the kid zone decides the month turn', '2026-05-02', '2026-06-02', 1]
	];
	for (const [name, birth, today, want] of cases) {
		it(`${name}: ${birth} on ${today} is ${want}`, () => {
			expect(ageMonths(birth, today)).toBe(want);
		});
	}
});

describe('formatAgeDays', () => {
	// Months of DAYS_PER_MONTH days, as the API's age_months axis counts them.
	it.each([
		[0, '0 days'],
		[1, '1 day'],
		[30, '30 days'],
		[31, '1 month'],
		[45, '1 month 14 days'],
		[61, '2 months'],
		[365, '11 months 30 days'],
		[366, '1 year'],
		[400, '1 year 1 month'],
		[1000, '2 years 8 months']
	])('%i days is %s', (days, want) => {
		expect(formatAgeDays(days)).toBe(want);
	});

	it('never shows a negative age', () => {
		expect(formatAgeDays(-3)).toBe('0 days');
	});
});

describe('formatMilk', () => {
	const milk = { gain: { grams: 62, ozHundredths: 219 }, volume: { ml: 60, flozHundredths: 204 } };
	const zero = { gain: { grams: 0, ozHundredths: 0 }, volume: { ml: 0, flozHundredths: 0 } };

	it('shows grams and millilitres for a kilogram and millilitre parent', () => {
		expect(formatMilk(milk, { weight: 'kg', volume: 'ml' })).toBe('weight gain 62 g, about 60 ml');
	});

	it('shows ounces and fluid ounces for a pound and fluid ounce parent', () => {
		expect(formatMilk(milk, { weight: 'lb', volume: 'floz' })).toBe(
			'weight gain 2.19 oz, about 2.04 fl oz'
		);
	});

	it('takes the two units separately', () => {
		expect(formatMilk(milk, { weight: 'kg', volume: 'floz' })).toBe(
			'weight gain 62 g, about 2.04 fl oz'
		);
		expect(formatMilk(milk, { weight: 'lb', volume: 'ml' })).toBe(
			'weight gain 2.19 oz, about 60 ml'
		);
	});

	it('formats a zero estimate rather than leaving it out', () => {
		expect(formatMilk(zero, { weight: 'kg', volume: 'ml' })).toBe('weight gain 0 g, about 0 ml');
		expect(formatMilk(zero, { weight: 'lb', volume: 'floz' })).toBe(
			'weight gain 0 oz, about 0 fl oz'
		);
	});

	it('formats the API columns as they are, converting nothing', () => {
		const odd = { gain: { grams: 100, ozHundredths: 1 }, volume: { ml: 7, flozHundredths: 90 } };
		expect(formatMilk(odd, { weight: 'lb', volume: 'floz' })).toBe(
			'weight gain 0.01 oz, about 0.9 fl oz'
		);
		expect(formatMilk(odd, { weight: 'kg', volume: 'ml' })).toBe('weight gain 100 g, about 7 ml');
	});
});

describe('formatAgo', () => {
	const now = new Date('2026-11-25T12:00:00Z');
	const ago = (minutes: number) =>
		formatAgo(new Date(now.getTime() - minutes * 60_000).toISOString(), now);
	it('says how long ago an instant was, in the units the Home tiles use', () => {
		expect(ago(0)).toBe('Just now');
		expect(ago(0.5)).toBe('Just now');
		expect(ago(1)).toBe('1 min ago');
		expect(ago(59)).toBe('59 min ago');
		expect(ago(60)).toBe('1 h ago');
		expect(ago(94)).toBe('1 h 34 min ago');
		expect(ago(24 * 60 - 1)).toBe('23 h 59 min ago');
		expect(ago(24 * 60)).toBe('1 d ago');
		expect(ago(3 * 24 * 60 + 600)).toBe('3 d ago');
	});
	it('never reads as the future', () => {
		expect(ago(-5)).toBe('Just now');
	});
});

describe('formatElapsed', () => {
	it('is H:MM:SS, with the hours growing past a day', () => {
		expect(formatElapsed(0)).toBe('0:00:00');
		expect(formatElapsed(65)).toBe('0:01:05');
		expect(formatElapsed(3600 + 34 * 60 + 7)).toBe('1:34:07');
		expect(formatElapsed(26 * 3600)).toBe('26:00:00');
		expect(formatElapsed(-3)).toBe('0:00:00');
	});
});

describe('formatDaysAgo', () => {
	const now = new Date('2026-11-25T12:00:00Z');
	const ago = (days: number, hours = 0) =>
		formatDaysAgo(new Date(now.getTime() - (days * 24 + hours) * 3_600_000).toISOString(), now);
	it('says today, yesterday, then whole days, then whole months', () => {
		expect(ago(0, 5)).toBe('today');
		expect(ago(1)).toBe('yesterday');
		expect(ago(2)).toBe('2 days ago');
		expect(ago(29)).toBe('29 days ago');
		expect(ago(30)).toBe('1 month ago');
		expect(ago(75)).toBe('2 months ago');
		expect(ago(400)).toBe('13 months ago');
	});
});
