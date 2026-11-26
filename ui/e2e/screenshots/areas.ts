// Index order of the README; a shot outside these areas is refused.
export const AREAS = [
	'auth',
	'shell',
	'sheets',
	'home',
	'trackers',
	'timeline',
	'growth',
	'vaccines',
	'medication',
	'visits',
	'milestones',
	'export',
	'kid-admin',
	'account',
	'notifications',
	'admin',
	'errors'
] as const;
export type Area = (typeof AREAS)[number];
export const PROJECTS = ['mobile', 'desktop'] as const;
