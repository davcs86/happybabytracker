package fonts

import (
	_ "embed"
	"errors"
	"fmt"
	"slices"
	"sync"
)

var (
	//go:embed NotoSans-Regular.ttf
	regular []byte
	//go:embed NotoSans-Bold.ttf
	bold []byte
)

// ErrFontsUnavailable: the embedded fonts failed to load; returned by Load on every call after.
var ErrFontsUnavailable = errors.New("fonts: unavailable")

var (
	once       sync.Once
	loadErr    error
	regularSet *set
	boldSet    *set
	loader     = load
)

// Regular returns a copy of the embedded Noto Sans Regular file.
func Regular() []byte { return slices.Clone(regular) }

// Bold returns a copy of the embedded Noto Sans Bold file.
func Bold() []byte { return slices.Clone(bold) }

// Load parses both fonts' cmaps once; a failure is returned on this and every later call.
func Load() error {
	once.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				loadErr = fmt.Errorf("%w: %v", ErrFontsUnavailable, r)
			}
		}()
		if err := loader(); err != nil {
			loadErr = fmt.Errorf("%w: %w", ErrFontsUnavailable, err)
		}
	})
	return loadErr
}

func load() error {
	r, err := parseCmap(regular)
	if err != nil {
		return fmt.Errorf("regular: %w", err)
	}
	b, err := parseCmap(bold)
	if err != nil {
		return fmt.Errorf("bold: %w", err)
	}
	regularSet, boldSet = r, b
	return nil
}
