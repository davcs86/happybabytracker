const plural = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`;

// What deleting a child removes; a count the API could not give leaves its line uncounted.
export function deleteConsequences(c: { entries: number | null; people: number | null }): string[] {
	const who = 'including entries other people logged';
	return [
		c.entries === null
			? `Every logged entry, ${who}`
			: `${plural(c.entries, 'logged entry', 'logged entries')}: feeds, sleep, diapers, solids and pumping, ${who}`,
		'Growth measurements and vaccination records',
		'Notes, the trackers you chose, and the details on this page',
		c.people === null
			? 'Access for everyone who has it'
			: `Access for ${plural(c.people, 'person', 'people')}`
	];
}
