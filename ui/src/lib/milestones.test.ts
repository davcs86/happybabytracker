import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import { MILESTONE_AGES, MILESTONE_DOMAINS } from './constants';
import { checklistTitle, domainLabel, reachedId, reachedLabel } from './milestones';
import type { Milestones } from './types';
import { EXPORT_MESSAGES } from './server/export-form';
import {
	MILESTONE_FIELD_MESSAGES,
	MILESTONE_LIMIT_MESSAGE,
	MILESTONE_REACHED_MESSAGE
} from './server/milestone-form';

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

	it('keeps the milestone and export messages, page titles and page words free of them', () => {
		const kid = resolve(import.meta.dirname, '../routes/(app)/kids/[kid]');
		const files = [
			resolve(kid, 'milestones/+page.svelte'),
			resolve(kid, 'milestones/new/+page.svelte'),
			resolve(kid, 'milestones/[entry]/+page.svelte'),
			resolve(kid, 'export/+page.svelte'),
			resolve(import.meta.dirname, 'components/milestones/Checklist.svelte'),
			resolve(import.meta.dirname, 'components/milestones/MilestoneFields.svelte')
		];
		const markup = files.map((f) =>
			readFileSync(f, 'utf8')
				.replace(/<script[\s\S]*?<\/script>/g, '')
				.replace(/<!--[\s\S]*?-->/g, '')
		);
		expect(markup.filter((m) => m.includes('<title') || m.includes('documentTitle='))).toHaveLength(
			4
		);
		const all = [
			...Object.values(MILESTONE_FIELD_MESSAGES),
			MILESTONE_REACHED_MESSAGE,
			MILESTONE_LIMIT_MESSAGE,
			...Object.values(EXPORT_MESSAGES),
			...markup
		];
		expect(all.flatMap(judged)).toEqual([]);
	});
});

describe('reachedId', () => {
	const reached = (id: string, extra = {}) => ({
		id,
		note: null,
		reachedOn: '2026-11-20',
		...extra
	});
	const m = {
		catalogShown: true,
		checklists: [
			{
				items: [
					{ id: 'smiles', reached: reached('r-smiles') },
					{ id: 'rolls', reached: null }
				]
			}
		],
		custom: [
			reached('c-old', { text: 'First wave' }),
			reached('c-new', { text: 'First wave', reachedOn: '2026-11-22' })
		],
		retired: [],
		reached: [reached('r-hidden', { item: 'claps' })]
	} as unknown as Milestones;
	it('finds the reached row of a catalog item, in a list or in the reached rows', () => {
		expect(reachedId(m, { item: 'smiles' })).toBe('r-smiles');
		expect(reachedId(m, { item: 'rolls' })).toBeNull();
		expect(reachedId(m, { item: 'claps' })).toBe('r-hidden');
		expect(reachedId(m, { item: 'unknown' })).toBeNull();
	});
	it('finds the newest custom milestone with that text', () => {
		expect(reachedId(m, { text: 'First wave' })).toBe('c-new');
		expect(reachedId(m, { text: 'Other' })).toBeNull();
	});
});
