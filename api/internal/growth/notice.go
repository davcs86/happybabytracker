package growth

import (
	"slices"
	"strings"
)

const (
	attributionHead = "Attribution"
	disclaimerHead  = "Adaptation disclaimer"
)

// AdaptationDisclaimers is the NOTICE's "Adaptation disclaimer" paragraph of each work whose table group in the
// "Attribution" list names one of tables, in NOTICE order; the rest of the NOTICE is never returned.
func AdaptationDisclaimers(src Source, tables []string) []string {
	info, ok := SourceInfo(src)
	if !ok {
		return nil
	}
	groups, paragraphs := noticeSections(info.Notice)
	var out []string
	for i, g := range groups {
		if i < len(paragraphs) && slices.ContainsFunc(g, func(t string) bool { return slices.Contains(tables, t) }) {
			out = append(out, paragraphs[i])
		}
	}
	return out
}

// noticeSections reads the table names of each "- a, b:" item under "Attribution" and the indented
// paragraphs under "Adaptation disclaimer"; each section ends at the next unindented line.
func noticeSections(notice string) (groups [][]string, paragraphs []string) {
	section, para := "", ""
	flush := func() {
		if para != "" {
			paragraphs = append(paragraphs, para)
		}
		para = ""
	}
	for _, line := range strings.Split(notice, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			flush()
		case !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "- "):
			flush()
			section = ""
			if strings.HasPrefix(line, attributionHead) {
				section = attributionHead
			} else if strings.HasPrefix(line, disclaimerHead) {
				section = disclaimerHead
			}
		case section == attributionHead && strings.HasPrefix(line, "- ") && strings.HasSuffix(trimmed, ":"):
			names := strings.Split(strings.TrimSuffix(strings.TrimPrefix(trimmed, "- "), ":"), ",")
			for i := range names {
				names[i] = strings.TrimSpace(names[i])
			}
			groups = append(groups, names)
		case section == disclaimerHead:
			para = strings.TrimSpace(para + " " + trimmed)
		}
	}
	flush()
	return groups, paragraphs
}
