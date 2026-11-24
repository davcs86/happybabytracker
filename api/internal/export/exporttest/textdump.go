// Package exporttest reads the text of an uncompressed export PDF; it is imported by _test files only.
package exporttest

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"
	"unicode/utf16"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

// Text is one string drawn on a page; X, Y are the baseline start in mm from the top-left corner.
type Text struct {
	S    string
	X, Y float64
}

// PtPerMM converts millimetres to PDF points.
const PtPerMM = 72 / 25.4

// PageTexts returns, per page in order, the `BT x y Td (…) Tj ET` strings of each uncompressed content stream.
func PageTexts(t testing.TB, pdf []byte) [][]Text {
	t.Helper()
	var pages [][]Text
	marker := []byte("<</Type /Page\n")
	rest := pdf
	for {
		i := bytes.Index(rest, marker)
		if i < 0 {
			break
		}
		rest = rest[i+len(marker):]
		content, after := nextStream(t, rest)
		pages = append(pages, contentTexts(t, content))
		rest = after
	}
	return pages
}

func nextStream(t testing.TB, b []byte) (content, rest []byte) {
	t.Helper()
	const lenKey, open = "<</Length ", ">>\nstream\n"
	i := bytes.Index(b, []byte(lenKey))
	if i < 0 {
		t.Fatal("no content stream after a page")
	}
	b = b[i+len(lenKey):]
	j := bytes.Index(b, []byte(open))
	if j < 0 {
		t.Fatal("malformed stream header")
	}
	n, err := strconv.Atoi(string(b[:j]))
	if err != nil {
		t.Fatalf("stream length: %v", err)
	}
	b = b[j+len(open):]
	if !bytes.HasPrefix(b[n:], []byte("\nendstream")) {
		t.Fatal("stream length does not end at endstream")
	}
	return b[:n], b[n:]
}

func contentTexts(t testing.TB, c []byte) []Text {
	t.Helper()
	var out []Text
	for {
		i := bytes.Index(c, []byte("BT "))
		if i < 0 {
			return out
		}
		c = c[i+3:]
		var x, y float64
		var tail string
		// Tf-only blocks (`BT /F1 9.00 Tf ET`) carry no Td and are skipped.
		if _, err := fmt.Sscanf(string(c[:min(len(c), 40)]), "%f %f %s", &x, &y, &tail); err != nil || tail != "Td" {
			continue
		}
		k := bytes.Index(c, []byte(" Td ("))
		c = c[k+len(" Td ("):]
		raw, n := unescape(t, c)
		c = c[n:]
		if !bytes.HasPrefix(c, []byte(") Tj ET")) {
			t.Fatalf("text not followed by Tj ET: %q", c[:min(len(c), 20)])
		}
		out = append(out, Text{S: decodeUTF16(t, raw), X: x / PtPerMM, Y: constants.ExportPageHeightMM - y/PtPerMM})
	}
}

// unescape reads a literal string body up to its closing parenthesis, returning the bytes and the length consumed.
func unescape(t testing.TB, c []byte) ([]byte, int) {
	t.Helper()
	var out []byte
	for i := 0; i < len(c); i++ {
		switch c[i] {
		case ')':
			return out, i
		case '\\':
			i++
			switch e := c[i]; e {
			case 'n':
				out = append(out, '\n')
			case 'r':
				out = append(out, '\r')
			case 't':
				out = append(out, '\t')
			case 'b':
				out = append(out, '\b')
			case 'f':
				out = append(out, '\f')
			case '0', '1', '2', '3', '4', '5', '6', '7':
				v, j := 0, 0
				for ; j < 3 && i+j < len(c) && c[i+j] >= '0' && c[i+j] <= '7'; j++ {
					v = v*8 + int(c[i+j]-'0')
				}
				out = append(out, byte(v))
				i += j - 1
			default:
				out = append(out, e)
			}
		default:
			out = append(out, c[i])
		}
	}
	t.Fatal("unterminated string")
	return nil, 0
}

func decodeUTF16(t testing.TB, b []byte) string {
	t.Helper()
	if len(b)%2 != 0 {
		t.Fatalf("odd UTF-16BE length %d", len(b))
	}
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = uint16(b[2*i])<<8 | uint16(b[2*i+1])
	}
	return string(utf16.Decode(u))
}

// Strings is the text of ts, in order.
func Strings(ts []Text) []string {
	out := make([]string, len(ts))
	for i, x := range ts {
		out[i] = x.S
	}
	return out
}
