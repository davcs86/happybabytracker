package fonts

import (
	"encoding/binary"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

var errMalformed = errors.New("malformed cmap")

// set holds the BMP code points a font maps to a glyph other than .notdef.
type set [0x10000 / 64]uint64

func (s *set) add(c uint32)       { s[c/64] |= 1 << (c % 64) }
func (s *set) has(r rune) bool    { return r >= 0 && r <= 0xffff && s[r/64]&(1<<(r%64)) != 0 }
func u16(b []byte, off int) int   { return int(binary.BigEndian.Uint16(b[off:])) }
func u32(b []byte, off int) int64 { return int64(binary.BigEndian.Uint32(b[off:])) }

// Covers reports whether both fonts print r; false when Load failed.
func Covers(r rune) bool {
	if Load() != nil || r < 0 || r > 0xffff || !utf8.ValidRune(r) {
		return false
	}
	if unicode.IsControl(r) || unicode.In(r, unicode.Zl, unicode.Zp) {
		return false
	}
	return regularSet.has(r) && boldSet.has(r)
}

// Encode replaces each rune the fonts cannot print, and each invalid byte, with ExportFallbackRune.
func Encode(s string) (out string, substituted bool) {
	var b strings.Builder
	b.Grow(len(s))
	for s != "" {
		r, n := utf8.DecodeRuneInString(s)
		if (r == utf8.RuneError && n == 1) || !Covers(r) {
			r, substituted = constants.ExportFallbackRune, true
		}
		b.WriteRune(r)
		s = s[n:]
	}
	return b.String(), substituted
}

// parseCmap reads the best Unicode cmap subtable (format 12, else 4); every offset is bounds-checked.
func parseCmap(f []byte) (*set, error) {
	if len(f) < 12 {
		return nil, errMalformed
	}
	numTables := u16(f, 4)
	if len(f) < 12+16*numTables {
		return nil, errMalformed
	}
	var cmap []byte
	for i := range numTables {
		rec := 12 + 16*i
		if string(f[rec:rec+4]) != "cmap" {
			continue
		}
		off, length := u32(f, rec+8), u32(f, rec+12)
		if off+length > int64(len(f)) {
			return nil, errMalformed
		}
		cmap = f[off : off+length]
		break
	}
	if len(cmap) < 4 {
		return nil, errMalformed
	}
	n := u16(cmap, 2)
	if len(cmap) < 4+8*n {
		return nil, errMalformed
	}
	var best []byte
	for i := range n {
		rec := 4 + 8*i
		platform, encoding := u16(cmap, rec), u16(cmap, rec+2)
		if platform != 0 && !(platform == 3 && (encoding == 1 || encoding == 10)) {
			continue
		}
		off := u32(cmap, rec+4)
		if off+2 > int64(len(cmap)) {
			return nil, errMalformed
		}
		sub := cmap[off:]
		switch u16(sub, 0) {
		case 12:
			best = sub
		case 4:
			if best == nil {
				best = sub
			}
		}
		if best != nil && u16(best, 0) == 12 {
			break
		}
	}
	if best == nil {
		return nil, errMalformed
	}
	if u16(best, 0) == 12 {
		return parseFormat12(best)
	}
	return parseFormat4(best)
}

func parseFormat4(b []byte) (*set, error) {
	if len(b) < 14 {
		return nil, errMalformed
	}
	length, segX2 := u16(b, 2), u16(b, 6)
	if length > len(b) || segX2 == 0 || segX2%2 != 0 || length < 16+4*segX2 {
		return nil, errMalformed
	}
	b = b[:length]
	n := segX2 / 2
	ends, starts, deltas, ranges := 14, 16+2*n, 16+4*n, 16+6*n
	s := new(set)
	prevEnd := -1
	for i := range n {
		start, end := u16(b, starts+2*i), u16(b, ends+2*i)
		delta, ro := u16(b, deltas+2*i), u16(b, ranges+2*i)
		if start > end || start <= prevEnd || ro%2 != 0 {
			return nil, errMalformed
		}
		prevEnd = end
		if ro != 0 && ranges+2*i+ro+2*(end-start)+2 > length {
			return nil, errMalformed
		}
		for c := start; c <= end; c++ {
			g := c
			if ro != 0 {
				if g = u16(b, ranges+2*i+ro+2*(c-start)); g == 0 {
					continue
				}
			}
			if (g+delta)&0xffff != 0 {
				s.add(uint32(c))
			}
		}
	}
	return s, nil
}

func parseFormat12(b []byte) (*set, error) {
	if len(b) < 16 {
		return nil, errMalformed
	}
	length, groups := u32(b, 4), u32(b, 12)
	if length > int64(len(b)) || 16+12*groups > length {
		return nil, errMalformed
	}
	s := new(set)
	prevEnd := int64(-1)
	for i := range int(groups) {
		g := 16 + 12*i
		start, end, glyph := u32(b, g), u32(b, g+4), u32(b, g+8)
		if start > end || end > unicode.MaxRune || start <= prevEnd {
			return nil, errMalformed
		}
		prevEnd = end
		for c := start; c <= min(end, 0xffff); c++ {
			if glyph+c-start != 0 {
				s.add(uint32(c))
			}
		}
	}
	return s, nil
}
