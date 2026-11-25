# AGENTS.md — UI design rules

Binding for every change under `ui/` (markup, CSS, components, copy, e2e selectors), for people and agents,
together with `CLAUDE.md`. Where the two meet, the stricter rule wins. The design system is a contract, not
a reference: it is followed, not approximated.

## Sources, in order of authority

1. `docs/plans/ui-redesign/design-system/DESIGN.md` (rules and **owner decisions**),
   `components/COMPONENTS.md` (sizes), `patterns/PATTERNS.md` (flows and wording).
2. The page spec `docs/plans/ui-redesign/pages/<page>.md`, its prototype
   `prototype/<Page>.dc.html` and the measured `specs/*.layout.json`.
3. `docs/plans/ui-redesign/IMPLEMENTATION.md` (how it maps onto SvelteKit).

A number in a spec beats prose; the prototype beats the legacy UI; an owner decision beats all three.
Read the sources that cover your change **before** you write markup, and compare the result with the
baseline pictures in `docs/plans/ui-redesign/screenshots/`.

## Rules

1. **Forms live in floating bottom sheets.** Every Add or Edit of a record, and every sheet the design
   names (Edit details, Access, Modules, Add a child), is a link with a real `href` and a `data-drawer`
   title that opens the same page, in embed mode, in a `FormSheet` (SvelteKit shallow routing, DESIGN 4.1,
   IMPLEMENTATION 5). The page is the no-JS and new-tab route. Never add an inline form or a page-only
   form for such a job. Sheet anatomy, failure handling and toasts follow DESIGN 4.2, 4.3 and PATTERNS P1.
   Pages that are not records (Notifications, My profile, sign-in) stay pages.
2. **One component, one definition.** Use `ui/src/lib/components/ui/*` and the app components
   (`FormSheet`, `AppShell`, the `navModel` function). No second copy of a nav list, a sheet, a chip, an
   age line or a field set. Native `input/select/textarea` keep the styling layer of DESIGN 4.3.
3. **Tokens and sizes only.** Colours through tokens, never a raw hex/oklch or a `dark:` utility. Sizes,
   radii and type from DESIGN 5 (targets at least 44 px, text at least 12 px, buttons 48, sheet Done
   56). No new transition except the sheet and the chevron turn, and none under `prefers-reduced-motion`.
4. **Words are part of the design.** Fixed sentences, titles and accessible names are copied from the
   prototype and the page spec, not reworded. A renamed accessible name updates its e2e specs in the same
   change.
5. **No silent deviation.** If the design is silent, contradicts itself or cannot be built as drawn, stop
   and ask (CLAUDE.md rule 1); do not pick a look. If an element needs data the app lacks, omit it and
   list it in `docs/wiki/Not-built-yet.md`; never fake it.
6. **The design is not edited to fit the code.** `docs/plans/ui-redesign/**` changes only when the owner
   decides a change, and the commit says which decision.

## Done means

A UI change is done only with output from this session of: `pnpm check`, `pnpm lint`, `pnpm test`,
`pnpm test:e2e` (axe clean in light and dark, at 390 and 1280 px), and `scripts/smoke.sh` when the
markup the smoke reads changed. A visible change also regenerates `pnpm screenshots`, the visual
baselines (`pnpm test:visual`) and the affected `docs/wiki` pages in the same PR.

The PR description has a **Deviations from the design** section: "none", or each deviation with the
owner decision that allows it.
