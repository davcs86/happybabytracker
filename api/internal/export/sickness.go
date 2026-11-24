package export

import (
	"strings"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

// symptomLabel and tempMethodLabel word the stored names as SYMPTOM_LABEL and TEMP_METHOD_LABEL do.
var (
	symptomLabel = map[string]string{
		"congestion": "Congestion", "cough": "Cough", "fever": "Fever", "vomiting": "Vomiting", "diarrhea": "Diarrhea",
		"constipation": "Constipation", "rash": "Rash", "ear_pain": "Ear pain", "sore_throat": "Sore throat",
		"poor_appetite": "Poor appetite", "unusually_sleepy": "Unusually sleepy", "other": "Other",
	}
	tempMethodLabel = map[string]string{
		"oral": "Oral", "rectal": "Rectal", "axillary": "Armpit", "ear": "Ear", "forehead": "Forehead", "other": "Other",
	}
)

// symptomLine is timelineLine's symptoms case without its clock.
func symptomLine(v model.Symptom, u model.Units) string {
	out := symptomLabel[v.Symptom]
	if v.Symptom == "other" && v.OtherText != nil && *v.OtherText != "" {
		out += ": " + Text(*v.OtherText)
	}
	if v.Temperature != nil {
		out += " " + FormatTemperature(*v.Temperature, u)
		if v.Method != nil {
			out += " (" + strings.ToLower(tempMethodLabel[*v.Method]) + ")"
		}
	}
	if v.Severity != nil {
		out += ", " + *v.Severity
	}
	return out
}

// sicknessItems prints the shown symptoms oldest first; the snapshot holds the newest, so overflow drops the oldest.
func sicknessItems(snap store.ExportSnapshot, req Request, b *budget) ([]item, error) {
	out := []item{{heading: moduleTitle["sickness"]}}
	entries, count := snap.Entries["symptom"], snap.Counts["symptom"]
	if count == 0 {
		return append(out, item{paragraph: noEntries}), nil
	}
	s := newSection(b)
	n := min(s.take(count), len(entries))
	rows := make([][]string, n)
	for i, it := range entries[:n] {
		rows[n-1-i] = []string{FormatInstant(it.At, req.Loc), symptomLine(it.Entry.(model.Symptom), req.Units), note(it.Entry)}
	}
	return append(out, s.shown(entryCols, rows, count)...), nil
}
