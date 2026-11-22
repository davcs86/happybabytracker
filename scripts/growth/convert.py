#!/usr/bin/env python3
# Standard library only; run with `python3 -I` from outside the download directory (untrusted input).
"""Convert the downloaded WHO xlsx and CDC CSV growth tables to api/internal/growth/data/*/*.csv."""

import hashlib
import math
import os
import re
import sys
import xml.etree.ElementTree as ET
import zipfile

RETRIEVED = "2026-11-22"
WHO_CDN = "https://cdn.who.int/media/docs/default-source/child-growth"
WHO_VERSION = "published 2006/2007; files dated 2020-11-10, retrieved 2026-11-22"
WHO_LICENCE = (
    "CC BY-NC-SA 3.0 IGO (https://creativecommons.org/licenses/by-nc-sa/3.0/igo/legalcode); "
    "not MIT: non-commercial use only, share-alike; see LICENSE and NOTICE in this directory"
)
WHO_2006 = (
    "WHO Child Growth Standards: length/height-for-age, weight-for-age, weight-for-length, "
    "weight-for-height and body mass index-for-age: methods and development. Geneva: World Health "
    "Organization; 2006. Licence: CC BY-NC-SA 3.0 IGO."
)
WHO_2007_HC = (
    "WHO Child Growth Standards: head circumference-for-age, arm circumference-for-age, triceps "
    "skinfold-for-age and subscapular skinfold-for-age: methods and development. Geneva: World Health "
    "Organization; 2007. Licence: CC BY-NC-SA 3.0 IGO."
)
WHO_2007_REF = (
    "WHO Growth Reference 2007 for 5-19 years (https://www.who.int/tools/growth-reference-data-for-5to19-years). "
    "Geneva: World Health Organization; 2007. Licence: CC BY-NC-SA 3.0 IGO."
)
WHO_ADAPTATION = (
    "converted from xlsx to CSV, values unchanged; only the age, L, M and S columns kept; "
    "an adaptation not created by WHO, see NOTICE"
)
CDC_BASE = "https://www.cdc.gov/growthcharts/data/zscore/"
CDC_SOURCE = "CDC 2000 growth charts, LMS data files (CDC, National Center for Health Statistics)"
CDC_VERSION = "CDC 2000 growth charts, page last reviewed 2024-09-02"
CDC_LICENCE = "public domain (U.S. Government work, https://www.cdc.gov/other/agencymaterials.html); see NOTICE"
CDC_ATTRIBUTION = (
    "Source: CDC, National Center for Health Statistics, 2000 CDC Growth Charts; "
    "not endorsed by CDC; free at https://www.cdc.gov/growthcharts/cdc-data-files.htm"
)
CDC_ADAPTATION = (
    "converted from CSV to CSV, values unchanged; only the Agemos, L, M and S columns kept, one file per sex"
)

WHO_2006_SRC = "WHO Child Growth Standards 2006 (0-5 years), expanded z-score tables"
WHO_2007_SRC = "WHO Growth Reference 2007 (5-19 years), expanded z-score tables"

# (table, sex, source file, url path under WHO_CDN, age column, source, attribution)
WHO_FILES = []
for _table, _ind, _dir in (
    ("wfa-day", "wfa", "weight-for-age/expanded-tables"),
    ("lhfa-day", "lhfa", "length-height-for-age/expandable-tables"),
    ("hcfa-day", "hcfa", "head-circumference-for-age/expanded-tables"),
    ("bfa-day", "bfa", "body-mass-index-for-age/expanded-tables"),
):
    for _sex, _who_sex in (("female", "girls"), ("male", "boys")):
        WHO_FILES.append((_table, _sex, f"{_ind}-{_who_sex}-zscore-expanded-tables.xlsx",
                          f"child-growth-standards/indicators/{_dir}", "Day", WHO_2006_SRC,
                          WHO_2007_HC if _ind == "hcfa" else WHO_2006))
for _table, _sex, _name, _dir in (
    ("wfa-month", "female", "hfa-girls-z-who-2007-exp_7ea58763-36a2-436d-bef0-7fcfbadd2820.xlsx",
     "weight-for-age-(5-10-years)"),
    ("wfa-month", "male", "hfa-boys-z-who-2007-exp_0ff9c43c-8cc0-4c23-9fc6-81290675e08b.xlsx",
     "weight-for-age-(5-10-years)"),
    ("hfa-month", "female", "hfa-girls-z-who-2007-exp.xlsx", "height-for-age-(5-19-years)"),
    ("hfa-month", "male", "hfa-boys-z-who-2007-exp.xlsx", "height-for-age-(5-19-years)"),
    ("bfa-month", "female", "bmi-girls-z-who-2007-exp.xlsx", "bmi-for-age-(5-19-years)"),
    ("bfa-month", "male", "bmi-boys-z-who-2007-exp.xlsx", "bmi-for-age-(5-19-years)"),
):
    WHO_FILES.append((_table, _sex, _name, f"growth-reference-5-19-years/{_dir}", "Month",
                      WHO_2007_SRC, WHO_2007_REF))

# sfvrsn must name the revision whose SHA-256 is recorded; WHO changes it on re-upload.
WHO_SFVRSN = {
    "wfa-girls-zscore-expanded-tables.xlsx": "f01bc813_10",
    "wfa-boys-zscore-expanded-tables.xlsx": "65cce121_10",
    "lhfa-girls-zscore-expanded-tables.xlsx": "27f1e2cb_10",
    "lhfa-boys-zscore-expanded-tables.xlsx": "7b4a3428_12",
    "hcfa-girls-zscore-expanded-tables.xlsx": "3a34b8b0_8",
    "hcfa-boys-zscore-expanded-tables.xlsx": "2ab1bec8_8",
    "bfa-girls-zscore-expanded-tables.xlsx": "ae4cb8d1_12",
    "bfa-boys-zscore-expanded-tables.xlsx": "f8e1fbe2_10",
    "hfa-girls-z-who-2007-exp_7ea58763-36a2-436d-bef0-7fcfbadd2820.xlsx": "6ede55a4_4",
    "hfa-boys-z-who-2007-exp_0ff9c43c-8cc0-4c23-9fc6-81290675e08b.xlsx": "b3ca0d6f_4",
    "hfa-girls-z-who-2007-exp.xlsx": "79d310ee_2",
    "hfa-boys-z-who-2007-exp.xlsx": "7fa263d_2",
    "bmi-girls-z-who-2007-exp.xlsx": "79222875_2",
    "bmi-boys-z-who-2007-exp.xlsx": "a84bca93_2",
}

CDC_TABLES = ("wtageinf", "lenageinf", "hcageinf", "wtage", "statage", "bmiagerev")
CDC_SEX = {"1": "male", "2": "female"}

NS = {"m": "http://schemas.openxmlformats.org/spreadsheetml/2006/main"}
MAX_XML_BYTES = 32 * 1024 * 1024


def die(msg):
    sys.exit(f"convert.py: {msg}")


def sha256(path):
    with open(path, "rb") as f:
        return hashlib.sha256(f.read()).hexdigest()


def shortest(text, where):
    """The shortest decimal that parses to the same double as `text`, checked against Excel's 15 digits."""
    x = float(text)
    if not math.isfinite(x):
        die(f"{where}: not finite: {text!r}")
    r = repr(x)
    if r.endswith(".0"):
        r = r[:-2]
    if "e" in r or "E" in r or r == "-0":
        die(f"{where}: unexpected form {r!r} for {text!r}")
    if float(r) != x or ("%.15g" % x) != r:
        die(f"{where}: {text!r} has more than 15 significant digits ({r!r})")
    return r


def col_of(ref):
    m = re.match(r"([A-Z]+)\d+$", ref or "")
    if not m:
        die(f"bad cell reference {ref!r}")
    return m.group(1)


def read_member(z, name):
    info = z.getinfo(name)
    if info.file_size > MAX_XML_BYTES:
        die(f"{name}: {info.file_size} bytes is too large")
    return z.read(name)


def xlsx_rows(path):
    with zipfile.ZipFile(path) as z:
        names = z.namelist()
        sheets = [n for n in names if re.fullmatch(r"xl/worksheets/sheet\d+\.xml", n)]
        if len(sheets) != 1:
            die(f"{path}: {len(sheets)} worksheets, want 1")
        shared = []
        if "xl/sharedStrings.xml" in names:
            sst = ET.fromstring(read_member(z, "xl/sharedStrings.xml"))
            for si in sst.findall("m:si", NS):
                shared.append("".join(t.text or "" for t in si.iter(f"{{{NS['m']}}}t")))
        sheet = ET.fromstring(read_member(z, sheets[0]))
    rows = []
    for row in sheet.find("m:sheetData", NS).findall("m:row", NS):
        cells = {}
        for c in row.findall("m:c", NS):
            v = c.find("m:v", NS)
            if v is None:
                continue
            kind = c.get("t", "n")
            if kind == "s":
                cells[col_of(c.get("r"))] = ("s", shared[int(v.text)])
            elif kind == "n":
                cells[col_of(c.get("r"))] = ("n", v.text)
            else:
                die(f"{path}: cell {c.get('r')} has unsupported type {kind!r}")
        rows.append(cells)
    return rows


def convert_who(path, age_name, where):
    rows = xlsx_rows(path)
    header = {v: col for col, (kind, v) in rows[0].items() if kind == "s"}
    for name in (age_name, "L", "M", "S"):
        if name not in header:
            die(f"{where}: no {name!r} column in {sorted(header)}")
    out = []
    for i, cells in enumerate(rows[1:], start=2):
        vals = []
        for name in (age_name, "L", "M", "S"):
            kind, text = cells.get(header[name], (None, None))
            if kind != "n":
                die(f"{where}: row {i} column {name} is not a number")
            vals.append(shortest(text, f"{where} row {i} {name}"))
        if not re.fullmatch(r"\d+", vals[0]):
            die(f"{where}: row {i} age {vals[0]!r} is not a whole number")
        out.append(vals)
    return out


def cdc_rows(path):
    with open(path, "rb") as f:
        data = f.read()
    if data.startswith(b"\xef\xbb\xbf"):
        data = data[3:]
    text = data.decode("ascii")
    if "\n" in text.replace("\r\n", ""):
        die(f"{path}: mixed line endings")
    lines = [ln for ln in text.split("\r\n") if ln != ""]
    header = lines[0].split(",")
    if header[:5] != ["Sex", "Agemos", "L", "M", "S"]:
        die(f"{path}: header {header[:5]}")
    by_sex = {"male": [], "female": []}
    order = []
    for n, ln in enumerate(lines[1:], start=2):
        if ln == lines[0]:
            continue  # the header row is repeated before the sex-2 block
        cols = ln.split(",")
        if len(cols) != len(header):
            die(f"{path}:{n}: {len(cols)} columns, want {len(header)}")
        sex = CDC_SEX.get(cols[0])
        if sex is None:
            die(f"{path}:{n}: sex {cols[0]!r}")
        if not order or order[-1] != sex:
            order.append(sex)
        vals = cols[1:5]
        for v in vals:
            if not re.fullmatch(r"-?\d+(\.\d+)?", v) or not math.isfinite(float(v)):
                die(f"{path}:{n}: value {v!r}")
        by_sex[sex].append(vals)
    if order != ["male", "female"]:
        die(f"{path}: sex blocks {order}, want male then female")
    return by_sex


def write_table(out_dir, src, meta, rows):
    os.makedirs(os.path.join(out_dir, src), exist_ok=True)
    name = f"{meta['table']}-{meta['sex']}.csv"
    lines = [f"# {k}: {v}" for k, v in meta.items()]
    lines.append("age,l,m,s")
    lines += [",".join(r) for r in rows]
    with open(os.path.join(out_dir, src, name), "w", encoding="utf-8", newline="\n") as f:
        f.write("\n".join(lines) + "\n")
    return f"{src}/{name}"


def main(argv):
    if len(argv) != 3:
        die("usage: python3 -I scripts/growth/convert.py <download-dir> <out-dir>")
    dl, out = argv[1], argv[2]
    report = []
    for table, sex, fname, url_dir, age_name, source, attribution in WHO_FILES:
        path = os.path.join(dl, fname)
        digest = sha256(path)
        meta = {
            "source": source,
            "table": table,
            "sex": sex,
            "origin_url": f"{WHO_CDN}/{url_dir}/{fname}?sfvrsn={WHO_SFVRSN[fname]}",
            "source_sha256": digest,
            "version": WHO_VERSION,
            "retrieved": RETRIEVED,
            "licence": WHO_LICENCE,
            "attribution": attribution,
            "adaptation": WHO_ADAPTATION,
        }
        rows = convert_who(path, age_name, fname)
        report.append((write_table(out, "who", meta, rows), len(rows), fname, digest))
    for table in CDC_TABLES:
        fname = f"{table}.csv"
        path = os.path.join(dl, fname)
        digest = sha256(path)
        for sex, rows in sorted(cdc_rows(path).items()):
            meta = {
                "source": CDC_SOURCE,
                "table": table,
                "sex": sex,
                "origin_url": CDC_BASE + fname,
                "source_sha256": digest,
                "version": CDC_VERSION,
                "retrieved": RETRIEVED,
                "licence": CDC_LICENCE,
                "attribution": CDC_ATTRIBUTION,
                "adaptation": CDC_ADAPTATION,
            }
            report.append((write_table(out, "cdc", meta, rows), len(rows), fname, digest))
    print("| file | rows | source | source sha256 |")
    print("|---|---|---|---|")
    for name, n, fname, digest in report:
        print(f"| {name} | {n} | {fname} | {digest} |")


if __name__ == "__main__":
    main(sys.argv)
