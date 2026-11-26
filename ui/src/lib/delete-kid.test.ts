import { describe, expect, it } from 'vitest';
import { deleteConsequences } from './delete-kid';

describe('deleteConsequences', () => {
	it('counts what is known, in the design’s words', () => {
		expect(deleteConsequences({ entries: 42, people: 3 })).toEqual([
			'42 logged entries: feeds, sleep, diapers, solids and pumping, including entries other people logged',
			'Growth measurements and vaccination records',
			'Notes, the trackers you chose, and the details on this page',
			'Access for 3 people'
		]);
	});
	it('says one entry and one person in the singular', () => {
		const lines = deleteConsequences({ entries: 1, people: 1 });
		expect(lines[0]).toMatch(/^1 logged entry: /);
		expect(lines[3]).toBe('Access for 1 person');
	});
	it('falls back to the uncounted wording for what it could not read', () => {
		expect(deleteConsequences({ entries: null, people: null })).toEqual([
			'Every logged entry, including entries other people logged',
			'Growth measurements and vaccination records',
			'Notes, the trackers you chose, and the details on this page',
			'Access for everyone who has it'
		]);
	});
});
