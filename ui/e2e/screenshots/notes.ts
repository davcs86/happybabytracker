// The README's hand-written parts; index.ts writes the rest from the shots themselves.
import type { Area } from './areas.ts';

export const FIXTURES = (now: string): string[] => [
	`All from \`ui/e2e/stub-api.ts\` (passwords are the stub's fake ones), as they stand on ${now.slice(0, 10)}:`,
	'',
	'| User | Role and kids |',
	'|---|---|',
	'| `admin` | the admin: only `/admin/parents` |',
	'| `ana` (Ana) | guardian of Mia and Ivy (and of Noa, Uma, Oli and a long-named kid the shots add), caregiver of Leo; of Sami in one milestone shot; the shell shots add four long-named children |',
	'| `cleo` (Cleo) | guardian of Leo, Nell, Otto, Pia, Remy and Sami |',
	'| `dan` (Dan) | no kids |',
	'| `bo` (Bo) | invited, no password yet (the invite link) |',
	'| `dee` (Dee) | disabled |',
	'',
	'| Kid | Born | Age on the frozen day | Notes |',
	'|---|---|---|---|',
	'| Mia | 2026-07-08 | 4 months | UTC |',
	'| Leo | 2024-08-20 | 2 years 3 months | born at 35 weeks 2 days, Europe/Madrid, growth rows from the stub |',
	'| Ivy | 2025-10-24 | 13 months | UTC |',
	'| Nell | 2026-02-24 | 9 months | milestones |',
	'| Otto | 2021-12-24 | 4 years 11 months | milestones on, one retired milestone row |',
	'| Pia | 2021-11-24 | 5th birthday | milestones on |',
	'| Remy | 2026-08-24 | 3 months | born at 32 weeks |',
	'| Sami | 2024-05-24 | 30 months | milestones off (the default past 24 months) |',
	'| Noa, Uma, Oli, NNN… | added by the shots | 10 days; 14 days at 30 weeks; 7 years; 40 days | created through the stub API |',
	'',
	'Entries, measurements, doses, courses and visits are seeded per test through the stub API (`seed.ts`);',
	'every test starts from `/__reset`.'
];

export const ABOUT_AREAS: Record<Area, string> = {
	auth: 'Sign-in, invite and password-reset pages, signed out.',
	sso: 'Single sign-on: the sign-in page by mode, and the pages a flow that cannot finish shows.',
	shell:
		'The app shell: tab bar on mobile, side navigation on desktop; its sheets (Switch child, More, Change theme) and the Choose a child pages. Every other shot shows the shell too (guardian and caregiver; the admin has none).',
	sheets:
		'The forms that open in a bottom sheet over the page they were opened from: record forms, child details, modules, access and Add a child. The record forms appear as sheets, state by state, in their own areas too.',
	home: 'Kid homes in every age band and role: quick log, running timers, Due, Today.',
	trackers:
		'Log and edit sheets of every tracker (Ana, Mia), opened from Home and the Timeline, with their errors, advanced modes and units; Log sleep, which no link opens, as a page.',
	timeline: 'The timeline: ranges, filters, paging, units and errors.',
	growth: 'Growth charts, the measurement list and its sheets, both sources and preterm ages.',
	vaccines: 'Vaccine schedules, dismissals and the vaccination sheets.',
	medication: 'Medication courses, doses and their sheets.',
	visits:
		'Medical visits: the list, a visit before and after it happens, cancelled, the delete confirm, its three sheets, the Find a US provider panel (stub scenarios `npi_on`, `npi_unavailable`, `npi_rate_limited`), and visits on Home, the Timeline and a Growth record.',
	milestones: 'Milestone checklists by age, and the sheets for marks and custom milestones.',
	export:
		'The PDF export page and its refusals. The PDF itself (POST /kids/[kid]/export/pdf) is a download, not a page: nothing is captured for it.',
	'kid-admin':
		'Guardian pages: modules, child profile, edit details, access and the add-a-child form, with the caregiver 404s.',
	account: 'My profile: details, units, password and signed-in devices.',
	notifications: 'Web Push devices and ntfy, under every server configuration the stub offers.',
	admin: 'The admin’s parents panel.',
	errors: 'The 404 and 503 pages.'
};

export const NOT_CAPTURED: string[] = [
	'**export-pdf**: the PDF is streamed as a download by `POST /kids/[kid]/export/pdf`; it is not a page.',
	'Endpoints that are not pages: `/healthz`, `/healthz/api`, `/healthz/reminders`, `/account/notifications/subscribe`, `/logout` (a POST that lands on the sign-in page, `auth/01`).',
	'The generic “Something went wrong” error page: the stub has no scenario that makes the UI answer a 5xx other than 503.',
	'The `down` and `slow` API scenarios: they render the same 503 page as `unavailable` (`errors/04`).',
	'The browser’s own notification permission prompt and a real push subscription: the headless browser cannot subscribe; “This device” is shown by registering a device through the stub and setting its device cookie.',
	'The “browser cannot receive notifications” message: it needs a browser without Web Push.',
	'**visits calendar**: `GET /kids/[kid]/visits/[visit]/calendar` is an .ics download, not a page; the form that asks for it is on `visits/03`.',
	'Install / PWA prompts: the app shows no install UI of its own.',
	'Caps that need hundreds of rows (the 21st medication course, the 501st vaccination): the same field-message pattern as the errors shown.',
	'Deleting a kid, an entry, a course, a dose or a visit: each lands on a page already shown; the confirm steps are captured.',
	'JavaScript off and dark mode: out of scope by decision, except `visits/24-provider-no-js`, the provider search results as a browser without JavaScript gets them.'
];
