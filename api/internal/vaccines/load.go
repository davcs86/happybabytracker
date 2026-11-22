package vaccines

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"sync"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

//go:embed data/*.csv
var dataFS embed.FS

const (
	antigensFile = "data/antigens.csv"
	productsFile = "data/products.csv"
)

// ErrSchedulesUnavailable: the embedded files failed to load; returned by Load on every call after.
var ErrSchedulesUnavailable = errors.New("vaccines: schedules unavailable")

var (
	once       sync.Once
	loadErr    error
	antigens   []Antigen
	antigenIdx map[string]bool
	products   []Product
	schemas    []Schema
	loader     = load
)

// Load reads the embedded catalogue and schedules once; a failure is returned on this and every later call.
func Load() error {
	once.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				loadErr = fmt.Errorf("%w: %v", ErrSchedulesUnavailable, r)
			}
		}()
		if err := loader(); err != nil {
			loadErr = fmt.Errorf("%w: %w", ErrSchedulesUnavailable, err)
		}
	})
	return loadErr
}

func load() error { return loadFrom(dataFS) }

// loadFrom sets antigenIdx before parsing the schedules, which check their antigens against it.
func loadFrom(fsys fs.FS) error {
	b, err := fs.ReadFile(fsys, antigensFile)
	if err != nil {
		return err
	}
	ants, err := parseAntigens(antigensFile, bytes.NewReader(b))
	if err != nil {
		return err
	}
	idx := map[string]bool{}
	for _, a := range ants {
		idx[a.ID] = true
	}
	antigens, antigenIdx = ants, idx
	if b, err = fs.ReadFile(fsys, productsFile); err != nil {
		return err
	}
	prods, err := parseProducts(productsFile, bytes.NewReader(b), idx)
	if err != nil {
		return err
	}
	paths, err := fs.Glob(fsys, "data/*.csv")
	if err != nil {
		return err
	}
	byID := map[string]Schema{}
	for _, p := range paths {
		if p == antigensFile || p == productsFile {
			continue
		}
		if b, err = fs.ReadFile(fsys, p); err != nil {
			return err
		}
		s, err := parse(p, bytes.NewReader(b))
		if err != nil {
			return err
		}
		byID[s.ID] = s
	}
	var list []Schema
	for _, id := range constants.VaccineSchedules {
		if s, ok := byID[id]; ok {
			list = append(list, s)
		}
	}
	products, schemas = prods, list
	return nil
}

// Schemas lists the shipped schedules in constants.VaccineSchedules order; nil when Load failed.
func Schemas() []Schema {
	if Load() != nil {
		return nil
	}
	return slices.Clone(schemas)
}

// GetSchema finds a shipped schedule by id.
func GetSchema(id string) (Schema, bool) {
	if Load() != nil {
		return Schema{}, false
	}
	for _, s := range schemas {
		if s.ID == id {
			return s, true
		}
	}
	return Schema{}, false
}
