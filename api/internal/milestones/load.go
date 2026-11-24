package milestones

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"sync"
)

//go:embed data/cdc.csv
var dataFS embed.FS

const catalogFile = "data/cdc.csv"

// ErrCatalogUnavailable: the embedded catalog failed to load; returned by Load on every call after.
var ErrCatalogUnavailable = errors.New("milestones: catalog unavailable")

var (
	once    sync.Once
	loadErr error
	catalog Catalog
	byID    map[string]Item
	loader  = load
)

// Load reads the embedded catalog once; a failure is returned on this and every later call.
func Load() error {
	once.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				loadErr = fmt.Errorf("%w: %v", ErrCatalogUnavailable, r)
			}
		}()
		if err := loader(); err != nil {
			loadErr = fmt.Errorf("%w: %w", ErrCatalogUnavailable, err)
		}
	})
	return loadErr
}

func load() error { return loadFrom(dataFS) }

func loadFrom(fsys fs.FS) error {
	f, err := fsys.Open(catalogFile)
	if err != nil {
		return err
	}
	defer f.Close()
	c, err := parse(catalogFile, f)
	if err != nil {
		return err
	}
	idx := make(map[string]Item, len(c.Items))
	for _, it := range c.Items {
		idx[it.ID] = it
	}
	catalog, byID = c, idx
	return nil
}

// Get returns the shipped catalog; false when Load failed.
func Get() (Catalog, bool) {
	if Load() != nil {
		return Catalog{}, false
	}
	c := catalog
	c.Omitted = slices.Clone(c.Omitted)
	c.Items = slices.Clone(c.Items)
	return c, true
}

// Lookup finds an item by id, retired items included; false when Load failed.
func Lookup(id string) (Item, bool) {
	if Load() != nil {
		return Item{}, false
	}
	it, ok := byID[id]
	return it, ok
}

// Ages lists the distinct age_months of active items, ascending; nil when Load failed.
func Ages() []int {
	if Load() != nil {
		return nil
	}
	var out []int
	for _, it := range catalog.Items {
		if it.Retired == "" && !slices.Contains(out, it.AgeMonths) {
			out = append(out, it.AgeMonths)
		}
	}
	slices.Sort(out)
	return out
}
