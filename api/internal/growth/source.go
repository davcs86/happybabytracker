package growth

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
)

// Info is a source's provenance, read from its first table file, plus the notices that ship with it.
// Licence is the short licence line; LicenceText is the source's LICENSE file, "" when it has none.
type Info struct {
	ID                                                        Source
	Name, Attribution, Licence, Version, OriginURL, Retrieved string
	Notice, LicenceText                                       string
}

var sourceNames = map[Source]string{
	SourceWHO: "WHO Child Growth Standards (2006) and Growth Reference (2007)",
	SourceCDC: "CDC 2000 Growth Charts",
}

// SourceInfo describes an offered source.
func SourceInfo(s Source) (Info, bool) {
	if Load() != nil {
		return Info{}, false
	}
	info, ok := infos[s]
	return info, ok
}

func newInfo(src Source, meta map[string]string) Info {
	return Info{
		ID:          src,
		Name:        sourceNames[src],
		Attribution: meta["attribution"],
		Licence:     meta["licence"],
		Version:     meta["version"],
		OriginURL:   meta["origin_url"],
		Retrieved:   meta["retrieved"],
		Notice:      readText(src, "NOTICE", true),
		LicenceText: readText(src, "LICENSE", false),
	}
}

func readText(src Source, name string, required bool) string {
	b, err := dataFS.ReadFile(path.Join("data", string(src), name))
	if err != nil && (required || !errors.Is(err, fs.ErrNotExist)) {
		panic(fmt.Sprintf("growth: %s/%s: %v", src, name, err))
	}
	return string(b)
}
