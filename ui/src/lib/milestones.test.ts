import { describe, expect, it } from 'vitest';
import { MILESTONE_AGES, MILESTONE_DOMAINS } from './constants';
import { checklistTitle, domainLabel, reachedLabel } from './milestones';

// The plan's Global Constraints: none of these is written by the app.
const DENYLIST = [
	'late',
	'behind',
	'delayed',
	'delay',
	'normal',
	'abnormal',
	'typical',
	'atypical',
	'on track',
	'ahead',
	'missed',
	'missing',
	'concern',
	'worry',
	'should',
	'red flag',
	'warning'
];
const judged = (s: string) => DENYLIST.filter((w) => new RegExp(`\\b${w}\\b`, 'i').test(s));

describe('checklistTitle', () => {
	it.each([
		[2, 'By 2 months'],
		[4, 'By 4 months'],
		[6, 'By 6 months'],
		[9, 'By 9 months'],
		[12, 'By 1 year'],
		[15, 'By 15 months'],
		[18, 'By 18 months'],
		[24, 'By 2 years'],
		[30, 'By 30 months'],
		[36, 'By 3 years'],
		[48, 'By 4 years'],
		[60, 'By 5 years']
	])('words %i months as CDC does: %s', (age, title) => {
		expect(checklistTitle(age)).toBe(title);
	});

	it('has a title for every catalog age', () => {
		expect(MILESTONE_AGES.map(checklistTitle).every((t) => t.startsWith('By '))).toBe(true);
	});
});

describe('domainLabel', () => {
	it('is the CDC heading without "Milestones"', () => {
		expect(MILESTONE_DOMAINS.map(domainLabel)).toEqual([
			'Social/Emotional',
			'Language/Communication',
			'Cognitive (learning, thinking, problem-solving)',
			'Movement/Physical Development'
		]);
	});
});

describe('reachedLabel', () => {
	it.each([
		['2026-03-04', 'Reached on 4 March 2026'],
		['2026-01-01', 'Reached on 1 January 2026'],
		['2025-12-31', 'Reached on 31 December 2025'],
		['2024-02-29', 'Reached on 29 February 2024'],
		['2026-11-10', 'Reached on 10 November 2026']
	])('words %s as a civil date', (date, label) => {
		expect(reachedLabel(date)).toBe(label);
	});

	it('names every month', () => {
		const months = Array.from({ length: 12 }, (_, i) =>
			reachedLabel(`2026-${String(i + 1).padStart(2, '0')}-15`)
		);
		expect(months).toEqual(
			[
				'January',
				'February',
				'March',
				'April',
				'May',
				'June',
				'July',
				'August',
				'September',
				'October',
				'November',
				'December'
			].map((m) => `Reached on 15 ${m} 2026`)
		);
	});
});

describe('wording', () => {
	it('avoids every judgement word', () => {
		const all = [
			...MILESTONE_AGES.map(checklistTitle),
			...MILESTONE_DOMAINS.map(domainLabel),
			reachedLabel('2026-03-04')
		];
		expect(all.flatMap(judged)).toEqual([]);
	});
});
