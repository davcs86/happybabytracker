// Package testlog is the log-leak check that tests of several packages share; no production code imports it.
package testlog

import (
	"encoding/json"
	"slices"
	"strings"
)

// Leaks names each field of a JSON log line outside allowed, and each string field carrying a banned value;
// request_id, time and duration_ms are random or measured, so they are never matched.
func Leaks(line string, allowed []string, banned ...string) []string {
	var fields map[string]any
	if err := json.Unmarshal([]byte(line), &fields); err != nil {
		return []string{"not JSON: " + err.Error()}
	}
	var out []string
	for key, v := range fields {
		if !slices.Contains(allowed, key) {
			out = append(out, "unexpected field "+key)
		}
		text, isText := v.(string)
		if !isText || key == "request_id" || key == "time" || key == "duration_ms" {
			continue
		}
		for _, b := range banned {
			if strings.Contains(text, b) {
				out = append(out, key+" carries "+b)
			}
		}
	}
	return out
}
