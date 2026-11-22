# Growth reference tables

`convert.py` turns the downloaded WHO and CDC growth tables into the CSV files under
`api/internal/growth/data/` that the API embeds. It uses the Python standard library only.

```sh
python3 -I scripts/growth/convert.py <download-dir> <out-dir>
```

`<out-dir>` is normally `api/internal/growth/data`. The script writes `who/<table>-<sex>.csv` and
`cdc/<table>-<sex>.csv` (`<sex>` is `female` or `male`) and prints a table of output file, rows, source
file and source SHA-256 for the pull request. It never touches `LICENSE` or `NOTICE`. The downloads
are untrusted input: keep them in their own directory and run the script from another one with `-I`.

## Inputs

`<download-dir>` holds these files, saved under the base name of their URL (no query string):

| File | URL |
|---|---|
| `{wfa,lhfa,hcfa,bfa}-{girls,boys}-zscore-expanded-tables.xlsx` | linked from `https://www.who.int/tools/child-growth-standards/standards/{weight-for-age,length-height-for-age,head-circumference-for-age}` and `https://www.who.int/toolkits/child-growth-standards/standards/body-mass-index-for-age-bmi-for-age` |
| `hfa-{girls,boys}-z-who-2007-exp_<guid>.xlsx` (weight, 5–10 y: misnamed by WHO) | `https://www.who.int/tools/growth-reference-data-for-5to19-years/indicators/weight-for-age-5to10-years` |
| `hfa-{girls,boys}-z-who-2007-exp.xlsx`, `bmi-{girls,boys}-z-who-2007-exp.xlsx` | `https://www.who.int/tools/growth-reference-data-for-5to19-years/indicators/{height-for-age,bmi-for-age}` |
| `{wtageinf,lenageinf,hcageinf,wtage,statage,bmiagerev}.csv` | `https://www.cdc.gov/growthcharts/data/zscore/<name>.csv` |

The exact WHO URLs, with their `sfvrsn` revision tokens, are in `convert.py` and in each output file's
`origin_url`. WHO changes those tokens when it re-uploads a file; compare the SHA-256 with the
`source_sha256` already recorded before replacing a table.

## What it does

- WHO xlsx: reads the single worksheet and its shared strings with `zipfile` and `xml.etree`, ignores
  styles, and keeps the `Day` (2006, whole days) or `Month` (2007) column and `L`, `M`, `S`. Excel often
  stores a number as up to 17 digits of binary-float text (`0.38090000000000002`); the script writes the shortest
  decimal that reads back as the same double (`0.3809`) and stops if that is not also the 15-digit value
  Excel displays, so nothing is rounded.
- CDC CSV: strips the UTF-8 BOM (`lenageinf.csv`), splits on CRLF, skips the header row repeated before
  the sex-2 block (`lenageinf.csv`, `bmiagerev.csv`), maps Sex 1 to `male` and 2 to `female`, and copies
  `Agemos`, `L`, `M`, `S` character for character.
- Each output starts with `# key: value` lines (`source`, `table`, `sex`, `origin_url`,
  `source_sha256`, `version`, `retrieved`, `licence`, `attribution`, `adaptation`), then `age,l,m,s`.

The WHO tables are CC BY-NC-SA 3.0 IGO, not MIT: see `api/internal/growth/data/who/NOTICE` and the
"Third-party data" section of the root `README.md`.
