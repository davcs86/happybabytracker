const SECTION = /^\/kids\/[^/]+\/(timeline|growth|vaccines|medication|milestones|export)(?:\/|$)/;
const SECTIONS = ['timeline', 'growth', 'vaccines', 'medication', 'milestones', 'export'];

// The page a child switch returns to: the same module when it has one, else Home.
export const switchTarget = (pathname: string): string => pathname.match(SECTION)?.[1] ?? '';

// A return target from a query string: only the module names above, else Home.
export const sectionOrHome = (raw: string | null): string => SECTIONS.find((s) => s === raw) ?? '';
