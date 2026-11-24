// Package milestones loads the bundled developmental milestone checklist catalog.
// Item ids are <age_months>m-<domain code>-<slug> and are never renumbered or reused.
package milestones

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

// Domain is one of the four checklist areas.
type Domain string

const (
	DomainSocialEmotional Domain = "social_emotional"
	DomainLanguage        Domain = "language_communication"
	DomainCognitive       Domain = "cognitive"
	DomainMovement        Domain = "movement_physical"
)

// Item is one checklist line. Retired is empty, or the date of the update that dropped the item.
type Item struct {
	ID        string
	AgeMonths int
	Domain    Domain
	Text      string
	SourceURL string
	Retired   string
}

// Catalog is one file: its provenance and its items in file order. Omitted is nil when the file has no omitted key.
type Catalog struct {
	ID, Name, Authority, Edition, Retrieved, OriginURL, Licence, Attribution, ScreeningNote string
	Omitted                                                                                 []string
	Items                                                                                   []Item
}

// Every loader error has this shape and never carries a cell's value.
const errFormat = "milestones: %s:%d: column %s: %s"

const (
	reasonEmpty        = "is empty"
	reasonHeader       = "header is not "
	reasonNoItems      = "no items"
	reasonProvLine     = "not a provenance line"
	reasonUnknownKey   = "unknown provenance key"
	reasonDuplicateKey = "duplicate provenance key"
	reasonAbsentKey    = "provenance key not present"
	reasonDate         = "not a YYYY-MM-DD date"
	reasonURL          = "not an https URL"
	reasonDuplicateID  = "duplicate id"
	reasonIDFormat     = "not <n>m-<se|lc|cg|mp>-<slug of a-z, 0-9 and hyphens>"
	reasonIDAge        = "does not start with its age_months"
	reasonAge          = "not one of the catalog ages"
	reasonDomain       = "unknown domain"
	reasonControl      = "contains a control character"
	reasonUTF8         = "not valid UTF-8"
	reasonIDDomain     = "domain code does not match the domain column"
	reasonSourceURL    = "not under " + sourcePrefix
	reasonRetired      = "not empty or a YYYY-MM-DD date"
)

const (
	dateLayout   = "2006-01-02"
	sourcePrefix = "https://www.cdc.gov/"
	// omittedKey is the one optional provenance key: semicolon-separated entries, none empty.
	omittedKey = "omitted"
)

var (
	provenanceKeys = []string{"catalog", "name", "authority", "edition", "retrieved", "origin_url", "licence", "attribution", "screening_note"}
	itemHeader     = []string{"id", "age_months", "domain", "text", "source_url", "retired"}
	domains        = []Domain{DomainSocialEmotional, DomainLanguage, DomainCognitive, DomainMovement}
	idPattern      = regexp.MustCompile(`^[0-9]{1,2}m-(se|lc|cg|mp)-[a-z0-9-]{1,48}$`)
	domainCodes    = map[Domain]string{DomainSocialEmotional: "se", DomainLanguage: "lc", DomainCognitive: "cg", DomainMovement: "mp"}
	reasonTextLong = fmt.Sprintf("longer than %d characters", constants.MilestoneCatalogTextMaxLen)
)

func fail(file string, line int, column, reason string) error {
	return fmt.Errorf(errFormat, file, line, column, reason)
}

// parse reads one catalog file.
func parse(name string, r io.Reader) (Catalog, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return Catalog{}, err
	}
	meta := map[string]string{}
	n, header, found := 0, "", false
	for len(data) > 0 {
		var l string
		l, data = cutLine(data)
		n++
		if !strings.HasPrefix(l, "#") {
			header, found = l, true
			break
		}
		if err := provenance(name, n, l, meta); err != nil {
			return Catalog{}, err
		}
	}
	if !found {
		n++
	}
	if err := checkHeader(name, n, header, found); err != nil {
		return Catalog{}, err
	}
	for _, k := range provenanceKeys {
		if _, ok := meta[k]; !ok {
			return Catalog{}, fail(name, n, k, reasonAbsentKey)
		}
	}
	rows, err := readRows(name, data, n)
	if err != nil {
		return Catalog{}, err
	}
	if len(rows) == 0 {
		return Catalog{}, fail(name, n, itemHeader[0], reasonNoItems)
	}
	c := Catalog{
		ID: meta["catalog"], Name: meta["name"], Authority: meta["authority"], Edition: meta["edition"],
		Retrieved: meta["retrieved"], OriginURL: meta["origin_url"], Licence: meta["licence"],
		Attribution: meta["attribution"], ScreeningNote: meta["screening_note"],
	}
	if v, ok := meta[omittedKey]; ok {
		for _, o := range strings.Split(v, ";") {
			c.Omitted = append(c.Omitted, strings.TrimSpace(o))
		}
	}
	seen := map[string]bool{}
	for _, row := range rows {
		it, err := parseItem(name, row, seen)
		if err != nil {
			return Catalog{}, err
		}
		seen[it.ID] = true
		c.Items = append(c.Items, it)
	}
	return c, nil
}

func provenance(name string, n int, l string, meta map[string]string) error {
	k, v, ok := strings.Cut(strings.TrimPrefix(l, "# "), ": ")
	if !strings.HasPrefix(l, "# ") || !ok || k == "" {
		return fail(name, n, "key", reasonProvLine)
	}
	if !slices.Contains(provenanceKeys, k) && k != omittedKey {
		return fail(name, n, "key", reasonUnknownKey)
	}
	if _, dup := meta[k]; dup {
		return fail(name, n, k, reasonDuplicateKey)
	}
	v = strings.TrimSpace(v)
	switch {
	case v == "":
		return fail(name, n, k, reasonEmpty)
	case k == "retrieved" && !isDate(v):
		return fail(name, n, k, reasonDate)
	case k == "origin_url" && !strings.HasPrefix(v, "https://"):
		return fail(name, n, k, reasonURL)
	case k == omittedKey && slices.ContainsFunc(strings.Split(v, ";"), func(o string) bool { return strings.TrimSpace(o) == "" }):
		return fail(name, n, k, reasonEmpty)
	}
	meta[k] = v
	return nil
}

func parseItem(name string, row csvRow, seen map[string]bool) (Item, error) {
	f := row.fields
	col := func(i int, reason string) error { return fail(name, row.line, itemHeader[i], reason) }
	it := Item{ID: f[0], Domain: Domain(f[2]), Text: f[3], SourceURL: f[4], Retired: f[5]}
	switch {
	case it.ID == "":
		return Item{}, col(0, reasonEmpty)
	case !idPattern.MatchString(it.ID):
		return Item{}, col(0, reasonIDFormat)
	case seen[it.ID]:
		return Item{}, col(0, reasonDuplicateID)
	}
	age, err := strconv.Atoi(f[1])
	if err != nil || strconv.Itoa(age) != f[1] || !slices.Contains(constants.MilestoneCatalogAges, age) {
		return Item{}, col(1, reasonAge)
	}
	it.AgeMonths = age
	switch {
	case !strings.HasPrefix(it.ID, f[1]+"m-"):
		return Item{}, col(0, reasonIDAge)
	case !slices.Contains(domains, it.Domain):
		return Item{}, col(2, reasonDomain)
	case !strings.HasPrefix(it.ID, f[1]+"m-"+domainCodes[it.Domain]+"-"):
		return Item{}, col(0, reasonIDDomain)
	case it.Text == "":
		return Item{}, col(3, reasonEmpty)
	case !utf8.ValidString(it.Text):
		return Item{}, col(3, reasonUTF8)
	case utf8.RuneCountInString(it.Text) > constants.MilestoneCatalogTextMaxLen:
		return Item{}, col(3, reasonTextLong)
	case strings.ContainsFunc(it.Text, unicode.IsControl):
		return Item{}, col(3, reasonControl)
	case !strings.HasPrefix(it.SourceURL, sourcePrefix):
		return Item{}, col(4, reasonSourceURL)
	case it.Retired != "" && !isDate(it.Retired):
		return Item{}, col(5, reasonRetired)
	}
	return it, nil
}

func isDate(s string) bool {
	_, err := time.Parse(dateLayout, s)
	return len(s) == len(dateLayout) && err == nil
}

func cutLine(data []byte) (string, []byte) {
	l, rest, _ := bytes.Cut(data, []byte("\n"))
	return string(l), rest
}

func checkHeader(name string, n int, got string, found bool) error {
	fields := strings.Split(got, ",")
	for i, w := range itemHeader {
		if !found || i >= len(fields) || fields[i] != w {
			return fail(name, n, w, reasonHeader+strings.Join(itemHeader, ","))
		}
	}
	if len(fields) != len(itemHeader) {
		return fail(name, n, itemHeader[len(itemHeader)-1], reasonHeader+strings.Join(itemHeader, ","))
	}
	return nil
}

type csvRow struct {
	line   int
	fields []string
}

// readRows reads the CSV records after line offset of the file, numbering lines from the file's start.
func readRows(name string, data []byte, offset int) ([]csvRow, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = len(itemHeader)
	var rows []csvRow
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			return rows, nil
		}
		var pe *csv.ParseError
		if errors.As(err, &pe) {
			column := "byte " + strconv.Itoa(pe.Column)
			if errors.Is(pe.Err, csv.ErrFieldCount) {
				column = "record"
			}
			return nil, fail(name, pe.Line+offset, column, pe.Err.Error())
		}
		if err != nil {
			return nil, err
		}
		line, _ := r.FieldPos(0)
		rows = append(rows, csvRow{line + offset, rec})
	}
}
