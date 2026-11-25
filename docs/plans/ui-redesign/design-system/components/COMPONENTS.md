# Components: contract and mapping

Each entry gives: the shadcn part (shadcn-svelte / shadcn/ui), what to change from the generated default, the exact dimensions, states, accessibility, and the markup contract. Values are tokens from `../tokens/tokens.css`; the numbers in px are the design's, not suggestions. Install the parts with the shadcn CLI, then edit the generated files in `components/ui/` (they are yours); do not wrap them in a second layer.

```
# SvelteKit (shadcn-svelte)                              # React (shadcn/ui)
pnpm dlx shadcn-svelte@latest add button card badge      pnpm dlx shadcn@latest add button card badge
  alert alert-dialog checkbox collapsible input label      alert alert-dialog checkbox collapsible input label
  radio-group select separator sheet sonner switch         radio-group select separator sheet sonner switch
  tabs textarea toggle-group                               tabs textarea toggle-group
```

Rules that apply to every component:
1. Target ≥ 44 px (control 48). `box-sizing: border-box` so the border counts inside the size.
2. Colour comes from tokens only. No raw hex, no `oklch()` literals in components.
3. Accessible names are stable (they are the e2e selectors) and never rely on colour or an icon alone.
4. Native elements stay native (`input`, `select`, `textarea`, `details`, `a`, `form`), styled by the control layer, so the page works without JS.

---

## 1. Button

| Variant | Look | Use |
|---|---|---|
| `default` (primary) | `--primary` fill, `--primary-foreground` text | the one main action per screen or sheet |
| `outline` (secondary) | `--card` fill, 1 px `--border` | every other action |
| `destructive` (confirm) | `--card` fill, **1.5 px** `--destructive` border and text | only the second step of a destructive action |
| `ghost` | no fill, `--primary` text, underline on hover | inline text actions ("Older", "Show more") |
| `icon` | 48 × 48, `outline` look | repeated row actions (Edit pencil, Choose a date) |
| `toast` | text only, `--toast-link`, 44 px | action inside a toast |

| Size | Height | Radius | Text |
|---|---|---|---|
| `default` | 48 | 12 | 15 / 600 |
| `lg` (sheet Done/Save and Cancel) | 56 | 14 | 16 / 600 |
| `xl` (standalone pages: sign in, set password) | 52 | 12 | 15 / 600 |
| `icon` | 48 × 48 | 12 | – |

States: hover = 4 % darker fill (primary) or `--accent` fill (outline); active = same as hover; focus = global ring; disabled = `opacity: .5`, `aria-disabled`, still readable (never below 3:1 for the label); busy = label stays, `aria-busy="true"`, `disabled`, no spinner replacing the label (the label is the name). Icons are 16-20 px, `aria-hidden`.

```ts
// SvelteKit: src/lib/components/ui/button/button.svelte  (tailwind-variants)
export const buttonVariants = tv({
  base: "inline-flex items-center justify-center gap-2 rounded-xl border border-transparent text-[15px] font-semibold " +
        "transition-colors disabled:pointer-events-none disabled:opacity-50 [&_svg]:size-4 [&_svg]:shrink-0",
  variants: {
    variant: {
      default: "bg-primary text-primary-foreground hover:bg-primary/90",
      outline: "bg-card border-border text-foreground hover:bg-accent",
      destructive: "bg-card border-[1.5px] border-destructive text-destructive hover:bg-accent",
      ghost: "text-primary hover:underline",
      toast: "text-(--toast-link) min-h-11 px-3 hover:underline",
    },
    size: { default: "min-h-12 px-4", lg: "min-h-14 rounded-[14px] px-5 text-base", xl: "min-h-[52px] px-5", icon: "size-12" },
  },
  defaultVariants: { variant: "outline", size: "default" },
});
```
```ts
// React: components/ui/button.tsx  (class-variance-authority): same strings in cva({ variants: { ... } })
```

Do not add a `sm` size. A smaller button breaks the 44 px floor.

## 2. Input, Select, Textarea, Label, field message

| Part | Spec |
|---|---|
| Input / Select (native) | `min-height 48` (44 for the narrow inline weight fields, width 96), `padding 0 12px`, radius **8**, 1 px `--border`, `--background` fill, **16 px** text, line-height 1.4, shadow `0 1px 2px rgb(0 0 0 / .05)`; select hides the native arrow, `padding-right 36`, 16 px chevron at `right 12px` |
| Focus (form controls) | border `--primary` + `0 0 0 3px` ring of `--primary` at 32 %; no outline. Buttons, links, chips use the 2 px outline instead |
| Textarea | same box, `min-height 96`, `padding 12px`, vertical resize only |
| Label | 13.5 / 600, 6 px above the control, always visible (placeholder is never the label) |
| Hint | 12.5 / 400 `--muted-foreground`, below the control, linked with `aria-describedby` |
| Field message | 12.5 / **600**, `--destructive`, below the control, words that state the rule, linked with `aria-describedby`; no icon-only or colour-only signal |
| Invalid | `aria-invalid="true"` → border `--destructive` (rule in `tokens.css`); the message is the signal |

Forms use `novalidate`; the product's messages replace browser bubbles. Dates use `<input type="date">`, numbers `inputmode="decimal"` (keep `type="text"` so locales with commas work). Autocomplete tokens are set on auth fields (`username`, `new-password`, `current-password`).

```svelte
<div class="grid gap-1.5">
  <Label for="gdate">Date</Label>
  <Input id="gdate" type="date" name="date" aria-invalid={err ? 'true' : undefined} aria-describedby={err ? 'gdate-err' : undefined} />
  {#if err}<p id="gdate-err" class="text-[12.5px] font-semibold text-destructive">{err}</p>{/if}
</div>
```

## 3. Checkbox, RadioGroup, Switch

- **Checkbox**: 20 × 20 box, radius 4, 1 px border (`--foreground` mixed 55 % into `--background`), `--background` fill; checked = `--primary` fill and border with a 14 px white check. Sits inside a ≥ 44 px row; the label is part of the hit area. Native `input[type=checkbox]` with `appearance: none`; use the shadcn `Checkbox` only where a native input is impossible.
- **RadioGroup** (single choice in a sheet: theme, role, source): a `fieldset` + `legend` (13 px muted); rows are `label`s, min-height 60 (64 for the data-source rows), radius 12, padding `8 12`, gap 12, 1 px `--border` on `--card`; radio 20 px (border as the checkbox, checked = `--primary` border and a 9 px `--primary` dot); text 15/600 + one-line description 13 / 1.4 muted; optional 24-28 px decorative swatch at the end. Selected row = `--accent` fill + `--primary` border (1.5 px on source rows). The whole row is the target.
- **Switch**: used for one thing (module on/off). 44 px row, visible state words ("On"/"Off") next to it; never the only signal. Without JS it is a checkbox in a form.

## 4. ToggleGroup chip (filters and multi-select)

`ToggleGroup type="multiple"`: each item is a 44 px high chip, radius 8, 1 px `--border`, `--card` fill, 14/500 text, `padding 0 14px`, gap 6 inside, 8 between chips (wraps); optional 16 px leading tracker icon. Pressed: `--accent` fill, 1 px `--primary` border, 600 weight, plus a 14 px check **before** the label (the check makes it non-colour-only). `aria-pressed` on each item (Bits UI/Radix expose `data-state="on"`; add `aria-pressed` via the root `type="multiple"` items). Without JS: each chip is a `label` around a visible checkbox in a GET form.

Single-choice segmented controls (the measure Tabs on Growth, units) use `Tabs` (section 7), not ToggleGroup.

## 5. Card and Collapsible (visit card)

| Part | Spec |
|---|---|
| Card | `--card` fill, 1 px `--border`, no shadow, padding 12-14, content gap 10-12; radius by role: 12 item cards in dense lists (Milestones, Vaccines), 14 summary and row cards (Home, Timeline, Growth, Medication), 16 form, settings, section and empty-state cards; an emphasised card (running timer) uses a 1.5 px `--primary` border; an empty-state card uses a 1 px **dashed** `--border`, padding 22, and holds a chip, a 17/600 line, a 13.5 muted line and the add action |
| Row (inside a card or a list) | min-height 56 (60 with detail line), hairline between rows (1 px `--border`), tile + text left, time or action right |
| Collapsible (disclosure card) | native `details/summary`; the `details` is a card (1 px border, radius 14, padding `0 14`); summary ≥ 52 px, chevron 16 px at the end rotating 180° when open (transition only when `prefers-reduced-motion: no-preference`); the current group is `open` by default. A bare `details` with 13 px text is used for help and reference notes |
| Empty | one sentence + the add action inside the same card |

## 6. Badge

12 / 600, `padding 3px 8px`, radius 99, 1 px `--border`, `--muted-foreground` text on `--card`; height 24. It never carries a status colour: state is in the words ("Done", "Due", "Guardian"). Tracker identity uses the tracker chip, not the badge.

## 7. Tabs (measure tabs on Growth)

Segmented control: a track of `--muted` (the prototype's soft2), padding 4, gap 4, radius 12 (52 px high overall). Each tab: `flex: 1`, min-height 44, radius 9, 14/600, no border. Selected: `--card` fill, `--foreground` text, shadow `0 1px 3px rgb(0 0 0 / .15)`, `aria-current="page"`. Unselected: transparent fill, `--muted-foreground`. In the product each tab is a link (`?measure=weight`), so it works without JS; progressive enhancement may swap panels client-side. The unit selector (kg/lb) is a different control: two or three 44 px segments, radius 10, 1 px `--border`, selected = `--accent` + 1.5 px `--primary`.

## 8. Alert, status, Toast

| Kind | Markup | Spec |
|---|---|---|
| Alert (error) | `role="alert"` | 1.5 px solid `--foreground` border, `--background` fill, radius 12, 13.5 px text / 1.5, bold lead ("Couldn't save."), padding `10 12` |
| Status / notice | `role="status"` (or `alert` when it needs to be read at once) | 1 px `--border`, `--background` fill, radius 12, 13 px, padding `10 12`; for neutral outcomes and notices |
| Toast (Sonner) | `role="status"`; `role="alert"` for a failure | inverse surface (`--foreground` fill, `--background` text), radius 12, 14 px, padding `6 6 6 16`, gap 12, menu shadow; inset 12 px left and right, `bottom: 84px` in both layouts (clears the phone tab bar); **6 s**, or **9 s** for a failure that offers "Try again"; actions via the `toast` Button variant (44 px) |

```ts
// Sonner (svelte-sonner / sonner)
<Toaster position="bottom-center" duration={6000} offset={84}
  toastOptions={{ classNames: {
    toast: "!bg-foreground !text-background !rounded-xl !border-0 !shadow-[0_8px_24px_rgba(0,0,0,.25)] !text-[14px]",
    actionButton: "!bg-transparent !text-(--toast-link) !min-h-11 !px-3 !font-semibold" } }} />
```
`--toast-link` flips between themes on purpose; do not substitute `--primary`.

## 9. Sheet (bottom drawer)

shadcn `Sheet side="bottom"` (Dialog semantics: focus trap, Escape, overlay click, focus return).

| Part | Spec |
|---|---|
| Overlay | `rgba(20, 12, 8, .5)` |
| Panel | `--card`, max width 480 centred, `max-height 92%` of the frame (`92dvh` in production), scroll inside, radius `20 20 0 0`, shadow sheet, padding `10 16 20` + `env(safe-area-inset-bottom)` |
| Handle | 36 × 4, radius 2, `--border`, decorative (`aria-hidden`); dragging, if added, is an enhancement with a visible Close button as the equivalent |
| Title | `SheetTitle` 18 / 600, wraps |
| Close | 44+ px icon button, name "Close", top-right |
| Footer | Done/Save (56, primary) and Cancel (56, outline) side by side, equal width; stacked below 340 px |

Opening moves focus to the first field (or the title when no field). The trigger keeps its name. A failed save keeps the sheet open with an Alert at the top of the content.

### Drawer-first routing
Add/Edit links are real `<a href="/…/new" data-drawer="Title">`. SvelteKit: shallow routing (`preloadData(href)` then `pushState(href, { selected })`; render the loaded page component inside `Sheet` when `page.state.selected` is set; Back or Close calls `history.back()`), with the plain route serving direct visits and no-JS. React: Next.js intercepting routes (`(.)new` with a `@modal` slot) or React Router's `useNavigation` + `<Outlet>` in a dialog. The inner page is rendered in embed mode (no header, tabs, back link, title, age line).

## 10. Navigation components

| Component | Spec |
|---|---|
| Tab bar (phone) | `role="navigation"` "Primary", 4 equal items (Home, Timeline, Growth, More), min-height 52, 22 px icon over a 12/600 label, gap 2, `--card` fill, top 1 px border, padding `6 6 10` + safe area; current = `--primary` text + `aria-current="page"`, others `--muted-foreground` |
| Side nav (≥ 768) | outer width 225 (200 content + padding `20 12` + 1 px right border), `--card` fill, gap 4; brand line 16/700 on top; items min-height 44, radius 10, padding `0 10`, gap 10, icon 18 + 14/600 label; current item = `--background` fill + `--primary` text + `aria-current="page"`; a hairline (1 px `--border`) after Growth, then the former More entries, Sign out last |
| Header | 48 px row: child switcher (secondary button: initial circle 28, name 15/600 one line ellipsis, chevron), bell icon button 48, role badge |
| Child switcher sheet | rows 64 px: initial 36, name wraps, age muted, check + `aria-current="true"` on the current child |
| More sheet | list rows 56 px: icon 20, label, one-line note; entries from the single nav model |

## 11. Tracker chip and avatar

Tracker chip: 38 × 38 by default (36 in Home latest and due lists, 40 on Timeline running cards, 44 on Home running cards), radius 12, centred icon 18 (20 on running cards), fill/text from `.tracker-<name>` classes in `tokens.css` (`--h` hue). The chip is decorative: the row's text names the tracker. Avatar initial: 28-36 px circle, `--accent` fill, `--primary-ink` 600 text, decorative.

## 12. Charts (Growth)

SVG with `role="img"` and a text summary + a "Chart data" Collapsible holding a real table (the accessible equivalent). Marks use the tracker bar colour; reference bands use `--border` and `--muted-foreground`; ticks 11 px; no tooltips required to read a value. Empty and single-point charts show a sentence instead of an empty plot.

## 13. Print (PDF export)

A4, 12 mm margins, Noto Sans, greyscale, fixed light, no theme, 11 pt body. Page breaks avoid splitting a table row. See `patterns/PATTERNS.md` for the module list.
