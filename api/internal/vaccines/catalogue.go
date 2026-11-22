package vaccines

import (
	"errors"
	"io"
	"slices"
	"strings"
)

// Antigen is one component a dose is recorded against.
type Antigen struct{ ID, Name string }

// Product is a generic (never branded) vaccine and the antigens one administration of it covers.
type Product struct {
	ID, Name string
	Antigens []string
}

var (
	ErrNoAntigens       = errors.New("vaccines: no antigen")
	ErrUnknownAntigen   = errors.New("vaccines: unknown antigen")
	ErrDuplicateAntigen = errors.New("vaccines: duplicate antigen")
)

var (
	antigenHeader = []string{"id", "name"}
	productHeader = []string{"id", "name", "antigens"}
)

// Antigens lists the catalogue in file order; nil when Load failed.
func Antigens() []Antigen {
	if Load() != nil {
		return nil
	}
	return slices.Clone(antigens)
}

// Products lists the generic products in file order; nil when Load failed.
func Products() []Product {
	if Load() != nil {
		return nil
	}
	return slices.Clone(products)
}

// ValidAntigens accepts a non-empty list of known antigen ids without duplicates.
func ValidAntigens(ids []string) error {
	if err := Load(); err != nil {
		return err
	}
	return checkAntigens(ids, antigenIdx)
}

func checkAntigens(ids []string, known map[string]bool) error {
	if len(ids) == 0 {
		return ErrNoAntigens
	}
	for i, id := range ids {
		if !known[id] {
			return ErrUnknownAntigen
		}
		if slices.Contains(ids[:i], id) {
			return ErrDuplicateAntigen
		}
	}
	return nil
}

func parseAntigens(name string, r io.Reader) ([]Antigen, error) {
	rows, err := catalogueRows(name, r, antigenHeader)
	if err != nil {
		return nil, err
	}
	var out []Antigen
	for _, row := range rows {
		a := Antigen{ID: row.fields[0], Name: row.fields[1]}
		if err := checkEntry(name, row, slices.ContainsFunc(out, func(o Antigen) bool { return o.ID == a.ID })); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func parseProducts(name string, r io.Reader, known map[string]bool) ([]Product, error) {
	rows, err := catalogueRows(name, r, productHeader)
	if err != nil {
		return nil, err
	}
	var out []Product
	for _, row := range rows {
		p := Product{ID: row.fields[0], Name: row.fields[1]}
		if err := checkEntry(name, row, slices.ContainsFunc(out, func(o Product) bool { return o.ID == p.ID })); err != nil {
			return nil, err
		}
		if row.fields[2] == "" {
			return nil, fail(name, row.line, "antigens", reasonEmpty)
		}
		p.Antigens = strings.Split(row.fields[2], "|")
		switch checkAntigens(p.Antigens, known) {
		case ErrUnknownAntigen:
			return nil, fail(name, row.line, "antigens", reasonUnknownAntigen)
		case ErrDuplicateAntigen:
			return nil, fail(name, row.line, "antigens", reasonDupAntigen)
		}
		out = append(out, p)
	}
	return out, nil
}

func checkEntry(name string, row csvRow, duplicate bool) error {
	switch {
	case row.fields[0] == "":
		return fail(name, row.line, "id", reasonEmpty)
	case duplicate:
		return fail(name, row.line, "id", reasonDuplicateID)
	case row.fields[1] == "":
		return fail(name, row.line, "name", reasonEmpty)
	}
	return nil
}

func catalogueRows(name string, r io.Reader, header []string) ([]csvRow, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	first, rest := cutLine(data)
	if err := checkHeader(name, 1, first, len(data) > 0, header); err != nil {
		return nil, err
	}
	rows, err := readRows(name, rest, 1, len(header))
	if err == nil && len(rows) == 0 {
		err = fail(name, 1, header[0], reasonNoLines)
	}
	return rows, err
}
