# Milestone catalog documents

| File | What it is |
|---|---|
| `cdc.md` | Generated review document of the shipped catalog, `api/internal/milestones/data/cdc.csv`: its provenance, one table per checklist age with every item's id, domain, text and source page, the retired items and where CDC's PDF words an item differently. A test fails when it is stale. Do not edit it. |

Regenerate it after changing the CSV, from `api/`:

```sh
UPDATE_DOCS=1 go test ./internal/milestones/ -race
```

How to change the catalog (fetching the pages, ids, retiring items, the checks and the pull request checklist):
[`../runbooks/updating-milestone-catalog.md`](../runbooks/updating-milestone-catalog.md). What the app shows
of it and under which conditions: the README's "Milestones" and "Third-party data" sections.
