# Icons

One SVG per icon, copied from the prototype's own path data (the source of truth for the pixel result). Every icon is a lucide icon: in the repo import it from `@lucide/svelte/icons/<name>` (or `lucide-react`) with `size={24}`, `strokeWidth={2}`, and the stroke is `currentColor`.

If a lucide release draws an icon slightly differently from these files (older and newer lucide versions differ by sub-pixel path changes), keep the file here as the reference and compare at 24 px before accepting.

| lucide name | used for (prototype key) | file |
|---|---|---|
| `arrow-up` | pumping | `arrow-up.svg` |
| `baby` | child | `baby.svg` |
| `bell` | bell | `bell.svg` |
| `calendar` | cal | `calendar.svg` |
| `chart-line` | Growth | `chart-line.svg` |
| `check` | check | `check.svg` |
| `chevron-down` | chevron-down | `chevron-down.svg` |
| `circle` | dot | `circle.svg` |
| `clock` | clock | `clock.svg` |
| `droplet` | diaper | `droplet.svg` |
| `ellipsis` | More | `ellipsis.svg` |
| `file-down` | export | `file-down.svg` |
| `flag` | milestones | `flag.svg` |
| `heart` | nursing | `heart.svg` |
| `house` | Home | `house.svg` |
| `info` | info | `info.svg` |
| `layout-grid` | modules | `layout-grid.svg` |
| `list` | Timeline | `list.svg` |
| `log-out` | logout | `log-out.svg` |
| `milk` | bottle | `milk.svg` |
| `minus` | minus | `minus.svg` |
| `moon` | sleep | `moon.svg` |
| `notebook-pen` | medication, sickness, symptom | `notebook-pen.svg` |
| `pencil` | pencil | `pencil.svg` |
| `plus` | plus | `plus.svg` |
| `settings` | settings | `settings.svg` |
| `shield-check` | shield | `shield-check.svg` |
| `sun-moon` | theme | `sun-moon.svg` |
| `syringe` | vaccination, vaccines | `syringe.svg` |
| `user` | me | `user.svg` |
| `utensils` | solids | `utensils.svg` |
| `x` | x | `x.svg` |

Sizes in use: 22 px tab bar and side nav, 20 px list/row, 16 px chevrons and inline, 18-24 px in 38/40 px tracker chips. Tracker icons: milk (bottle), heart (nursing), droplet (diaper), moon (sleep), arrow-up (pumping), utensils (solids), notebook-pen (symptom).
