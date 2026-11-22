package httpapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/store"
)

// After a later birth-date correction an older entry stays editable: only a time the PATCH changes is checked.
func TestPatchOfAnEntryThatPredatesTheBirthDate(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid() // born 2026-03-04 in Madrid
	old := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	tooOld := ts(old.Add(time.Hour)) // still before the corrected birth date
	base := "/v1/kids/" + kid + "/"
	end := ts(old.Add(10 * time.Minute))
	ids := map[string]string{
		"bottles":  k.createBottle(ana, kid, map[string]any{"at": ts(old)}),
		"diapers":  k.createDiaper(ana, kid, map[string]any{"at": ts(old)}),
		"meals":    k.createMeal(ana, kid, map[string]any{"at": ts(old)}),
		"pumping":  k.createPumping(ana, kid, map[string]any{"started_at": ts(old)}),
		"sleep":    k.createSleep(ana, kid, map[string]any{"started_at": ts(old), "ended_at": end}),
		"nursing":  k.createNursing(ana, kid, finishedNursingBody(map[string]any{"started_at": ts(old), "ended_at": end})),
		"sleep2":   k.createSleep(ana, kid, map[string]any{"started_at": ts(old.Add(-time.Hour)), "ended_at": end}),
		"nursing2": k.createNursing(ana, kid, finishedNursingBody(map[string]any{"started_at": ts(old.Add(-time.Hour)), "ended_at": end})),
	}
	if rec := k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"birth_date": "2026-05-01"}); rec.Code != http.StatusOK {
		t.Fatalf("correct the birth date: %d %q", rec.Code, rec.Body.String())
	}
	timeField := map[string]string{"bottles": "at", "diapers": "at", "meals": "at", "pumping": "started_at", "sleep": "started_at", "nursing": "started_at"}
	for tracker, field := range timeField {
		path := base + tracker + "/" + ids[tracker]
		if rec := k.do(http.MethodPatch, path, ana, map[string]any{"note": "later"}); rec.Code != http.StatusOK {
			t.Errorf("%s: a note-only patch = %d %q, want 200", tracker, rec.Code, rec.Body.String())
		}
		wantInvalid2(t, tracker+" moved before the birth date", k.do(http.MethodPatch, path, ana, map[string]any{field: tooOld}), field)
	}
	for _, tracker := range []string{"sleep", "nursing"} {
		path := base + tracker + "/" + ids[tracker+"2"]
		wantInvalid2(t, tracker+" end moved before the birth date", k.do(http.MethodPatch, path, ana, map[string]any{"ended_at": ts(old.Add(5 * time.Minute))}), "ended_at")
		if rec := k.do(http.MethodPatch, path, ana, map[string]any{"ended_at": end, "note": "same end"}); rec.Code != http.StatusOK {
			t.Errorf("%s: re-sending the stored end = %d %q, want 200", tracker, rec.Code, rec.Body.String())
		}
	}
}

func TestEntryOKAnswersInvalidStoreSentinels(t *testing.T) {
	for err, field := range map[error]string{
		fmt.Errorf("wrapped: %w", store.ErrInvalidSession):  "at",
		fmt.Errorf("wrapped: %w", store.ErrInvalidContents): "contents",
	} {
		rec := httptest.NewRecorder()
		if entryOK(rec, err) {
			t.Errorf("%v: entryOK = true", err)
		}
		wantInvalid2(t, err.Error(), rec, field)
	}
}
