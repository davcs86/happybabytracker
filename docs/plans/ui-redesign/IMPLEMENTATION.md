# Implementation guide: bring `ui/` to the new design

Audience: the engineer (or agent) implementing the new design in `davcs86/happybabytracker/ui` (SvelteKit, Svelte 5, Tailwind 4, shadcn-svelte, `pnpm`). A React + shadcn/ui target is covered by the same design system (`design-system/components/COMPONENTS.md` shows both); the commands below are for the SvelteKit repo.

"Legacy" = what the repo implements today. "Prototype" = the Claude Design project in `prototype/`. Where they differ, **the prototype wins**, except where the page spec says a legacy behaviour is kept.

## 0. Assumptions and decisions to confirm before coding (repo rule: list assumptions)
| # | Assumption | Basis | If wrong |
|---|---|---|---|
| A1 | Server code (API client, form actions, load functions, `lib/*.ts` domain logic and their unit tests) is **not** changed; only `.svelte`, CSS, `app.html`, theme plumbing and e2e selectors | the request was to update the *design* | list the server change per page in the PR |
| A2 | Theme preference is stored in a cookie `hbt_theme` (`light`/`dark`/`auto`), server-read in `hooks.server.ts` | only the cookie works before first paint with no JS | pick another store, keep `data-theme`/`data-resolved` |
| A3 | Drawer-first forms use SvelteKit shallow routing over the existing `new` / `[entry]` routes | no new routes needed; the full pages stay as the no-JS fallback | add `@modal`-style routes (React/Next only) |
| A4 | Accessible names change on purpose in a few places (listed in section 6); e2e selectors are updated in the same PR | the prototype renames the nav landmark and moves Sign out | keep legacy names and drop the rename |
| A5 | The legacy shadcn style `nova` defaults (32 px buttons, `rounded-lg`, `ring-3`) are replaced, not wrapped | targets must be 44-56 px | n/a |
| A6 | The prototype's "More" sheet and desktop inline entries replace the legacy header "Sign out" button; Sign out stays a `POST /logout` | owner decision in the prototype | restore the header button |
| A7 | Pages with no prototype page (the Modules page, health endpoints, `/logout`) are restyled with the same components but are not pixel-specified. Add a child, Child settings and Access now have prototype pages | they have no prototype state matrix | ask for a prototype page before polishing them |
| A8 | The spec is verified in headless Chromium only | tooling available | add WebKit/Firefox projects |

## 1. Order of work (each phase ends with a gate; do not start the next on a red gate)
1. **Tokens, fonts, theme** (section 2). Gate: `pnpm check && pnpm test`, theme toggles without a flash, contrast table unchanged.
2. **UI parts** (section 3). Gate: a throwaway `/dev/components` page (not committed) renders every component state; axe clean in both themes.
3. **Shell**: tab bar, side nav, header (child switcher + bell + role badge), More sheet, theme sheet, toast host. Gate: `e2e/shell.spec.ts` updated and green; 390 and 980 baselines match for Kid Home default.
4. **Pages**, in this order (each is one PR): Sign in / Set password / Error → Kid Home → Timeline → Growth → Vaccines → Milestones (+ form) → Medication (+ Course, Dose) → Symptom → Export (+ PDF) → Child profile → My profile → Notifications → Choose child → Admin. Gate per page: its spec in `pages/`, its baselines, axe (all four variants), its e2e file.
5. **Drawer routes** (section 5) once at least three forms exist as pages.
6. **Baselines and CI** (section 7).

## 2. Tokens, fonts, theme
```bash
cd ui
pnpm add @fontsource-variable/geist-mono          # Geist is already a dependency
```
1. **`src/routes/layout.css`**: keep the four `@import` lines, add `@import '@fontsource-variable/geist-mono';`, then **replace everything below the imports** (the `@custom-variant dark` line, both `:root` blocks, the dark media block, `@theme inline`, `@layer base`) with the contents of `design-system/tokens/tokens.css`. The legacy `--chart-*` and `--sidebar-*` variables are not referenced by any component outside `layout.css` today, so they go with the old blocks (re-add them only if a generated part needs one).
2. **`src/app.html`**: `<html lang="en" data-theme="%hbt.theme%">`; add, as the first element in `<head>` (before any stylesheet):
```html
<script>
  (function () {
    var d = document.documentElement, t = d.getAttribute('data-theme') || 'auto', m = matchMedia('(prefers-color-scheme: dark)');
    function r() { d.setAttribute('data-resolved', t === 'auto' ? (m.matches ? 'dark' : 'light') : t); }
    r(); if (t === 'auto') m.addEventListener('change', r);
  })();
</script>
```
   If the app ships a CSP, add this script's hash through `kit.csp`. Replace the two `theme-color` metas with `#fdf8f3` (light) and `#19120e` (dark).
3. **`src/hooks.server.ts`**: read `event.cookies.get('hbt_theme')`, validate to `light|dark|auto` (default `auto`), and in `resolve(event, { transformPageChunk: ({ html }) => html.replace('%hbt.theme%', theme) })`.
4. **Theme action** (works without JS): `src/routes/theme/+server.ts` handles `POST` with `theme=light|dark|auto` and `return=/path` (whitelist same-origin paths), sets the cookie (`Path=/; Max-Age=31536000; SameSite=Lax; HttpOnly=false` so the sheet can apply it live), responds `303`. The Change theme sheet posts to it; with JS it also sets `data-theme`/`data-resolved` immediately and `fetch`es the action.
5. Delete uses of `dark:` that duplicate a token (the variant now follows `data-resolved`); prefer tokens.
6. Remove the `class="dark"`/media assumptions in any component CSS.

## 3. shadcn parts
```bash
pnpm dlx shadcn-svelte@latest add badge checkbox collapsible radio-group separator sheet sonner switch tabs textarea toggle-group alert-dialog
```
Then edit **every** generated part in `src/lib/components/ui/*` to the dimensions in `design-system/components/COMPONENTS.md` (the existing `button`, `card`, `input`, `label`, `alert` too). The `nova` style ships 32 px buttons; the design needs 48/52/56 px, radius 12/14, and the focus rules (outline for buttons and links; border + 3 px ring for form controls). Keep the generated file structure and the `tv()` variants API so future `shadcn-svelte add` diffs stay readable. Icons: use `@lucide/svelte` with the 32 names in `design-system/icons/README.md`; the SVGs in `design-system/icons/` are for non-Svelte consumers (PDF, README, e2e fixtures).

New app-level components (put in `src/lib/components/`): `PageHeader.svelte` (child switcher + bell + role badge), `ChildSwitcherSheet.svelte` (replaces `KidSwitcher.svelte`), `MoreSheet.svelte`, `ThemeSheet.svelte`, `TrackerChip.svelte` (`class="tracker-chip tracker-{name}"`), `EmptyState.svelte`, `FormSheet.svelte` (Sheet wrapper with Done/Cancel), `DisclosureCard.svelte`. One function builds the navigation lists for the tab bar, the More sheet and the side nav (see `design-system/DESIGN.md` 3.1) so they cannot drift; `AppShell.svelte` consumes it.

## 4. Pixel-perfect rules
- Implement from the **page specs** (`pages/<page>.md`: element boxes, computed styles, shadcn part, accessible name) and compare with the **baselines** (`screenshots/`). When a number in a spec conflicts with prose, the number (measured from the rendered prototype) wins.
- Phone width 390; side-nav layout 980 (≥ 768); wide layout 1280 (≥ 1280, `xl:`). Between 768 and 1279 content is one column: max 720 px (list and detail pages) or 448 px (full-page forms), left-aligned; `main` padding `16 16 28`, section gap 18; side nav 225 px outer (legacy `md:w-56` is 224 px: set `md:w-[225px]`). At ≥ 1280 the content is a fixed 920 px two-column block (448 + 24 + 448), centred; see `design-system/DESIGN.md` §2.1 and the column map there. <!-- wide:v1 -->
- **Wide grid recipe (Tailwind 4).** Wrapper: `class="xl:mx-auto xl:grid xl:w-[920px] xl:grid-cols-[448px_448px] xl:items-start xl:gap-x-6 xl:gap-y-[18px]"`. Column wrappers: `class="contents xl:block"` (a `display: contents` wrapper below 1280 keeps one flat column; at ≥ 1280 it is a normal block holding that column's children). Children without a column: `xl:col-span-2`. Forms: `<form class="xl:grid xl:grid-cols-[448px_448px] xl:grid-rows-[auto_1fr] xl:gap-x-6 xl:gap-y-[18px]">` with the primary group first, the optional group `xl:row-span-2`, then the Save/Cancel row; keep DOM order primary, optional, actions so tab order matches reading order. Do not use `order-*`. The breakpoint is the Tailwind 4 default (`--breakpoint-xl: 80rem`); do not override it.
- Use Tailwind arbitrary values only for numbers that are not on the token scale (`min-h-12` for 48, `min-h-14` for 56, `text-[15px]`); never raw colours.
- Geist must be loaded before any screenshot or measurement.
- Animations/transitions: off in tests (`animations: 'disabled'`); in product only the sheet open/close and the chevron turn, both gated by `prefers-reduced-motion`.
- Do not copy anything listed under "Prototype-only" (DESIGN.md section 13): the simulation panel, the iframe drawer, sample data, the 34 px segments.

## 5. Drawer-first forms (SvelteKit shallow routing)
Pattern, per list page:
```svelte
<script lang="ts">
  import { goto, preloadData, pushState } from '$app/navigation';
  import { page } from '$app/state';
  import NewPage from './new/+page.svelte';
  async function open(e: MouseEvent, href: string) {
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;       // modified click: let the browser open the page
    e.preventDefault();
    const r = await preloadData(href);
    if (r.type === 'loaded' && r.status === 200) pushState(href, { selected: r.data });
    else goto(href);                                                          // fallback: full page
  }
</script>
<a href="/kids/{kid}/growth/new" data-drawer="Add measurement" onclick={(e) => open(e, `/kids/${kid}/growth/new`)}>Add measurement</a>
{#if page.state.selected}<FormSheet title="Add measurement" onclose={() => history.back()}><NewPage data={page.state.selected} form={page.form} embed /></FormSheet>{/if}
```
`embed` hides the header, tabs, back link, title and age line. The form keeps `method="POST"` and `use:enhance`; on success `await invalidateAll(); history.back();` then show the toast. Add `PageState { selected?: Record<string, unknown> }` to `src/app.d.ts`. Validation refusals and server failures render inside the sheet (design: `PATTERNS.md` P1).

## 6. Accessible names changed on purpose (update e2e in the same PR)
| Legacy | New | Why |
|---|---|---|
| `nav` "Main" | `nav` "Primary" | one primary landmark, either tab bar or side nav in the tree |
| header button "Sign out" | last entry "Sign out" in the More sheet / side nav (note "This device") | owner decision (fewer header controls) |
| page `h1` "Timeline for {name}", "Growth for {name}", "Medication for {name}", "Milestones for {name}", "Vaccines for {name}", "Export PDF for {name}", "Settings for {name}" | "Timeline", "Growth", "Medication", "Milestones", "Vaccines", "Export PDF", "Child profile" | the child is in the header switcher (owner decision) |
| kid-switcher control | button "Child: {name}. Switch child" | see DESIGN 3.2 |
Everything else keeps its name; the per-page specs list the names to assert. Diff the accessibility snapshot against `specs/*.layout.json` (`name` fields).

## 7. Tests and baselines
- Keep `e2e/*.spec.ts` and `expectAccessible`; add dark and 980 px projects in `playwright.config.ts` (see `design-system/VERIFICATION.md`).
- Visual spec: `tests/visual/pages.visual.spec.ts` (generated, in this export) loops over `tests/visual/index.json` and compares against the baselines installed by `tools/install-baselines.sh`. Baselines go to `e2e/__screenshots__/` (git LFS recommended: ~1,000 images).
- The repo's `pnpm screenshots` flow documents the legacy design at 390×844 and 1280×800; re-generate it after the redesign merges so `docs/screenshots` matches, or retire it in favour of the baselines here (decision for the owner).
- Run order in CI: `pnpm check`, `pnpm test`, `pnpm test:e2e` (with dark + wide projects), then the visual project.

## 8. Definition of done (per page)
Spec read; every state in the page spec reachable and matching its baseline in the four variants; axe clean on each state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; no-JS path works (disable JS in a Playwright project and run the form flow); copy matches the fixed sentences; denylist test passes (Milestones); names diffed.
