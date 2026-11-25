# Design system: core rules + brand preset

Two layers, so that several products stay consistent while each keeps its own look.

```
design-system/
├─ DESIGN.md                      CORE rules: layout, navigation, forms, feedback, copy, accessibility, verification. Product-neutral.
├─ components/COMPONENTS.md       CORE contract: every component, dimensions, states, a11y, shadcn-svelte and shadcn/ui mapping.
├─ patterns/PATTERNS.md           CORE flows and state catalog (record form, destructive action, filters, auth, ...).
├─ VERIFICATION.md                How to prove an implementation matches (tokens, axe + layout per state, pixel baselines).
├─ preset-happybaby/BRAND.md      PRESET: palette, contrast table, type roles, radius use, tracker hues, shadows, icons.
├─ tokens/tokens.css              GENERATED from the preset: Tailwind 4 `@theme inline` + shadcn variable names + themes.
├─ tokens/tokens.json             GENERATED: same values for tools (with contrast results).
├─ icons/                         32 lucide SVGs actually used + inventory.
└─ (fonts/ at the export root)    Geist and Geist Mono, OFL.
```

## Using it for a new product
1. Copy the folder. Keep `DESIGN.md`, `components/`, `patterns/`, `VERIFICATION.md` unchanged (improve them upstream, not per product).
2. Replace `preset-<name>/BRAND.md` values, regenerate `tokens/` (`gen_tokens.py` in the export), re-check every contrast row.
3. Install the shadcn parts listed at the top of `COMPONENTS.md` and apply the variant/size changes there.
4. Wire the verification matrix in CI before building screens, not after.

## Decisions recorded (the "why" lives next to each rule in DESIGN.md)
- 48 px controls, 44 px floor, 56 px sheet actions: large, forgiving touch targets.
- Drawer first, full page as the no-JS route: the server is the source of truth; JS only enhances.
- ToggleGroup chips (check + fill + border) for every filter/multi-select; RadioGroup rows for single choices; text labels for destructive actions.
- One accent, tracker hues as the only multi-colour; colour never the only signal.
- Light, Dark, Auto with `data-theme` / `data-resolved` set before first paint.
- Neutral, non-judging wording for developmental and health content (denylist test).
- Stable accessible names as the e2e contract.

## Versioning
Treat `DESIGN.md` changes as semver: a changed number or removed rule is a major; a new component or pattern is a minor; a clarification is a patch. Record renames of accessible names in release notes.
