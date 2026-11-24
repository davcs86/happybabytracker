package fonts

import (
	"encoding/binary"
	"testing"
)

func TestCoverage(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	var want []rune
	for _, rg := range [][2]rune{{0x20, 0x7e}, {0xa0, 0xff}, {0x100, 0x17f}} {
		for r := rg[0]; r <= rg[1]; r++ {
			want = append(want, r)
		}
	}
	want = append(want, []rune("Ελένη Ольга Łucja Ștefan € ‘’“” \uFFFD\u200B\uFEFF\u00AD")...)
	for _, r := range want {
		if !Covers(r) {
			t.Errorf("Covers(%U) = false, want true", r)
		}
	}
	refused := []rune{'李', 0x1F600, 0x7f, 0x2028, 0x2029, 0xD800, 0xDFFF, -1, 0x110000}
	for r := rune(0); r <= 0x1f; r++ {
		refused = append(refused, r)
	}
	for _, r := range refused {
		if Covers(r) {
			t.Errorf("Covers(%U) = true, want false", r)
		}
	}
	if !regularSet.has(0x2028) || !regularSet.has(0x2029) {
		t.Error("U+2028/U+2029 are expected in the cmap and refused by category only")
	}
	if *regularSet != *boldSet {
		t.Error("Regular and Bold cover different sets")
	}
	n := 0
	for r := rune(0); r <= 0xffff; r++ {
		if regularSet.has(r) {
			n++
		}
	}
	if n < 2500 {
		t.Errorf("Regular covers %d BMP code points, want at least 2500", n)
	}
}

func TestEncode(t *testing.T) {
	for _, c := range []struct {
		in, out string
		sub     bool
	}{
		{"Mia 😀", "Mia ?", true},
		{"李小明", "???", true},
		{"Zoë Müller-Señor", "Zoë Müller-Señor", false},
		{"a\xffb", "a?b", true},
		{"a\xe2\x82b", "a??b", true},
		{"line\u2028x\ty", "line?x?y", true},
		{"\uFFFD", "\uFFFD", false},
		{"", "", false},
	} {
		out, sub := Encode(c.in)
		if out != c.out || sub != c.sub {
			t.Errorf("Encode(%q) = %q, %v; want %q, %v", c.in, out, sub, c.out, c.sub)
		}
	}
}

// sfnt wraps one cmap subtable (platform 3, the given encoding) in a minimal font.
func sfnt(encoding uint16, sub []byte) []byte {
	cmap := binary.BigEndian.AppendUint16(nil, 0)
	cmap = binary.BigEndian.AppendUint16(cmap, 1)
	cmap = binary.BigEndian.AppendUint16(cmap, 3)
	cmap = binary.BigEndian.AppendUint16(cmap, encoding)
	cmap = binary.BigEndian.AppendUint32(cmap, 12)
	cmap = append(cmap, sub...)
	f := binary.BigEndian.AppendUint32(nil, 0x00010000)
	f = binary.BigEndian.AppendUint16(f, 1)
	f = append(f, make([]byte, 6)...)
	f = append(f, "cmap"...)
	f = binary.BigEndian.AppendUint32(f, 0)
	f = binary.BigEndian.AppendUint32(f, 28)
	f = binary.BigEndian.AppendUint32(f, uint32(len(cmap)))
	return append(f, cmap...)
}

type seg struct{ start, end, delta, rangeOffset uint16 }

// format4 builds a format 4 subtable; the 0xFFFF terminator is the caller's last segment.
func format4(segs []seg, glyphs []uint16) []byte {
	n := len(segs)
	length := 16 + 8*n + 2*len(glyphs)
	b := binary.BigEndian.AppendUint16(nil, 4)
	b = binary.BigEndian.AppendUint16(b, uint16(length))
	b = binary.BigEndian.AppendUint16(b, 0)
	b = binary.BigEndian.AppendUint16(b, uint16(2*n))
	b = append(b, make([]byte, 6)...)
	for _, s := range segs {
		b = binary.BigEndian.AppendUint16(b, s.end)
	}
	b = binary.BigEndian.AppendUint16(b, 0)
	for _, s := range segs {
		b = binary.BigEndian.AppendUint16(b, s.start)
	}
	for _, s := range segs {
		b = binary.BigEndian.AppendUint16(b, s.delta)
	}
	for _, s := range segs {
		b = binary.BigEndian.AppendUint16(b, s.rangeOffset)
	}
	for _, g := range glyphs {
		b = binary.BigEndian.AppendUint16(b, g)
	}
	return b
}

type group struct{ start, end, glyph uint32 }

func format12(groups []group) []byte {
	b := binary.BigEndian.AppendUint16(nil, 12)
	b = binary.BigEndian.AppendUint16(b, 0)
	b = binary.BigEndian.AppendUint32(b, uint32(16+12*len(groups)))
	b = binary.BigEndian.AppendUint32(b, 0)
	b = binary.BigEndian.AppendUint32(b, uint32(len(groups)))
	for _, g := range groups {
		b = binary.BigEndian.AppendUint32(b, g.start)
		b = binary.BigEndian.AppendUint32(b, g.end)
		b = binary.BigEndian.AppendUint32(b, g.glyph)
	}
	return b
}

var end4 = seg{0xffff, 0xffff, 1, 0}

func TestParseCmapValid(t *testing.T) {
	// 'A'..'C' by delta; 'a','b' through glyphIdArray, where 'b' maps to glyph 0 (not covered).
	f4 := sfnt(1, format4([]seg{{0x41, 0x43, 1, 0}, {0x61, 0x62, 0, 4}, end4}, []uint16{7, 0}))
	s, err := parseCmap(f4)
	if err != nil {
		t.Fatal(err)
	}
	for r, want := range map[rune]bool{'A': true, 'C': true, 'D': false, 'a': true, 'b': false, 0xffff: false} {
		if s.has(r) != want {
			t.Errorf("format 4: has(%U) = %v", r, !want)
		}
	}
	f12 := sfnt(10, format12([]group{{0x41, 0x42, 5}, {0x100, 0x101, 0}, {0x1F600, 0x1F600, 9}}))
	if s, err = parseCmap(f12); err != nil {
		t.Fatal(err)
	}
	for r, want := range map[rune]bool{'A': true, 'B': true, 'C': false, 0x100: false, 0x101: true} {
		if s.has(r) != want {
			t.Errorf("format 12: has(%U) = %v", r, !want)
		}
	}
}

func TestParseCmapRefusesMalformed(t *testing.T) {
	good4 := format4([]seg{{0x41, 0x43, 1, 0}, end4}, nil)
	with := func(b []byte, off int, v uint16) []byte {
		c := append([]byte(nil), b...)
		binary.BigEndian.PutUint16(c[off:], v)
		return c
	}
	with32 := func(b []byte, off int, v uint32) []byte {
		c := append([]byte(nil), b...)
		binary.BigEndian.PutUint32(c[off:], v)
		return c
	}
	valid := sfnt(1, good4)
	for name, font := range map[string][]byte{
		"empty":                  nil,
		"short header":           valid[:10],
		"table records cut":      valid[:20],
		"no cmap":                append(append(append([]byte(nil), valid[:12]...), "glyf"...), valid[16:]...),
		"cmap beyond file":       with32(valid, 20, 1<<20),
		"cmap length overflow":   with32(valid, 24, 0xffffffff),
		"cmap header cut":        valid[:30],
		"encoding records cut":   with(valid, 30, 50),
		"subtable offset beyond": with32(valid, 36, 1<<16),
		"no unicode subtable":    with(valid, 32, 1),
		"unknown format":         sfnt(1, with(good4, 0, 6)),
		"f4 length beyond":       sfnt(1, with(good4, 2, 400)),
		"f4 length too short":    sfnt(1, with(good4, 2, 20)),
		"f4 odd segCountX2":      sfnt(1, with(good4, 6, 3)),
		"f4 zero segments":       sfnt(1, with(good4, 6, 0)),
		"f4 segments beyond":     sfnt(1, with(good4, 6, 200)),
		"f4 start after end":     sfnt(1, format4([]seg{{0x43, 0x41, 1, 0}, end4}, nil)),
		"f4 rangeOffset beyond":  sfnt(1, format4([]seg{{0x41, 0x43, 0, 4}, end4}, []uint16{1})),
		"f4 rangeOffset odd":     sfnt(1, format4([]seg{{0x41, 0x41, 0, 3}, end4}, []uint16{1, 1})),
		"f4 overlapping":         sfnt(1, format4([]seg{{0x41, 0x43, 1, 0}, {0x42, 0x44, 1, 0}, end4}, nil)),
		"f4 subtable cut":        sfnt(1, good4[:10]),
		"f12 groups beyond":      sfnt(10, with32(format12([]group{{0x41, 0x42, 5}}), 12, 1<<28)),
		"f12 length short":       sfnt(10, with32(format12([]group{{0x41, 0x42, 5}}), 4, 20)),
		"f12 start after end":    sfnt(10, format12([]group{{0x42, 0x41, 5}})),
		"f12 above U+10FFFF":     sfnt(10, format12([]group{{0x41, 0x110000, 5}})),
		"f12 overlapping":        sfnt(10, format12([]group{{0x41, 0x42, 5}, {0x41, 0x43, 5}})),
		"f12 header cut":         sfnt(10, format12(nil)[:10]),
	} {
		if _, err := parseCmap(font); err == nil {
			t.Errorf("%s: parseCmap accepted a malformed font", name)
		}
	}
}

func TestParseCmapTruncatedRealFontNeverPanics(t *testing.T) {
	reg := Regular()
	for n := 0; n < len(reg); n += 97 {
		_, _ = parseCmap(reg[:n])
	}
	if _, err := parseCmap(reg[:40]); err == nil {
		t.Error("a 40-byte prefix of Regular was accepted")
	}
}
