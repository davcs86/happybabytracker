package store

import (
	"errors"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/schedule"
)

func newVisit(at time.Time) model.Visit {
	return model.Visit{StartsAt: at, DurationMin: constants.DefaultVisitDurationMin, Kind: "well_child"}
}

// set is an UpdateVisit edit that replaces the stored visit with v.
func set(v model.Visit) func(model.Visit) (model.Visit, error) {
	return func(model.Visit) (model.Visit, error) { return v, nil }
}

func mustVisit(t *testing.T, s *Store, kidID, actorID string, v model.Visit) model.Visit {
	t.Helper()
	got, err := s.CreateVisit(ctx, kidID, actorID, v, t0)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// linkRow stamps visitID on a row of table created through its module, as the outcome will.
func linkRow(t *testing.T, s *Store, table, column, id, visitID string) {
	t.Helper()
	if _, err := s.w.Exec(`UPDATE `+table+` SET visit_id = ? WHERE `+column+` = ?`, visitID, id); err != nil {
		t.Fatal(err)
	}
}

func TestVisitCRUD(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)

	in := newVisit(t0.Add(48 * time.Hour))
	in.Provider, in.Location, in.Note = strp("Dr. Lee"), strp("12 Main St"), strp("4-month check")
	v, err := s.CreateVisit(ctx, k.ID, bo.ID, in, t0)
	if err != nil {
		t.Fatal(err)
	}
	if v.ID == "" || v.KidID != k.ID || v.CreatedBy != bo.ID || v.Revision != 0 || !v.CreatedAt.Equal(t0) {
		t.Fatalf("created %+v", v)
	}
	got, err := s.Visit(ctx, k.ID, v.ID)
	if err != nil || !got.StartsAt.Equal(in.StartsAt) || *got.Provider != "Dr. Lee" || *got.Note != "4-month check" ||
		got.Kind != "well_child" || got.DurationMin != 30 || len(got.Links.Growth)+len(got.Links.Vaccinations)+len(got.Links.Medications) != 0 {
		t.Fatalf("read back %+v, %v", got, err)
	}

	edit := got.Visit
	edit.DurationMin, edit.Location = 45, nil
	later := t0.Add(time.Hour)
	updated, err := s.UpdateVisit(ctx, k.ID, ana.ID, edit.ID, later, set(edit))
	if err != nil || updated.Revision != 1 || updated.UpdatedBy != ana.ID || !updated.UpdatedAt.Equal(later) ||
		updated.DurationMin != 45 || updated.Location != nil || updated.CreatedBy != bo.ID {
		t.Fatalf("updated %+v, %v", updated, err)
	}
	if _, err := s.UpdateVisit(ctx, k.ID, ana.ID, updated.Visit.ID, later, set(updated.Visit)); err != nil {
		t.Fatal(err)
	}
	if again, _ := s.Visit(ctx, k.ID, v.ID); again.Revision != 2 {
		t.Errorf("revision after two writes = %d, want 2", again.Revision)
	}
	if err := s.DeleteVisit(ctx, k.ID, bo.ID, v.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Visit(ctx, k.ID, v.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete: %v, want ErrNotFound", err)
	}
}

func TestListVisitsBothOrders(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	var ids []string
	for i := range 5 {
		ids = append(ids, mustVisit(t, s, k.ID, ana.ID, newVisit(t0.Add(time.Duration(i-2)*24*time.Hour))).ID)
	}
	all := ListQuery{From: time.Unix(0, 0), To: t0.Add(constants.VisitHorizon), Limit: 2}
	idsOf := func(rows []VisitRow) (out []string) {
		for _, r := range rows {
			out = append(out, r.ID)
		}
		return out
	}
	page, err := s.ListVisits(ctx, k.ID, all, true)
	if err != nil || len(page) != 2 || page[0].ID != ids[0] || page[1].ID != ids[1] {
		t.Fatalf("asc page 1 %v, %v", idsOf(page), err)
	}
	next := all
	next.After = &Cursor{At: page[1].StartsAt.Unix(), ID: page[1].ID}
	if page, _ = s.ListVisits(ctx, k.ID, next, true); len(page) != 2 || page[0].ID != ids[2] || page[1].ID != ids[3] {
		t.Errorf("asc page 2 %v", idsOf(page))
	}
	if page, _ = s.ListVisits(ctx, k.ID, all, false); len(page) != 2 || page[0].ID != ids[4] || page[1].ID != ids[3] {
		t.Errorf("desc page 1 %v", idsOf(page))
	}
	upcoming := ListQuery{From: t0, To: t0.Add(constants.VisitHorizon), Limit: 10}
	if page, _ = s.ListVisits(ctx, k.ID, upcoming, true); len(page) != 3 || page[0].ID != ids[2] {
		t.Errorf("upcoming %v, want the three from now, soonest first", idsOf(page))
	}
}

func TestListVisitsCountsLinkedRows(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	v := mustVisit(t, s, k.ID, ana.ID, newVisit(t0.Add(-time.Hour)))
	g := mustGrowth(t, s, k.ID, ana.ID, newGrowth(t, "2026-03-01"))
	a, err := s.CreateAdministration(ctx, k.ID, ana.ID, model.Administration{GivenOn: "2026-03-01", Antigens: []string{"dtap", "ipv"}}, t0)
	if err != nil {
		t.Fatal(err)
	}
	m := mustCourse(t, s, k.ID, ana.ID, timesCourse(t0.Add(-time.Hour), "08:00"), t0)
	linkRow(t, s, "growth_measurements", "id", g.ID, v.ID)
	linkRow(t, s, "vaccine_doses", "administration", a.ID, v.ID)
	linkRow(t, s, "medications", "id", m.ID, v.ID)
	rows, err := s.ListVisits(ctx, k.ID, ListQuery{From: time.Unix(0, 0), To: t0.Add(time.Hour), Limit: 5}, false)
	if err != nil || len(rows) != 1 || rows[0].Linked != (LinkedCounts{Growth: 1, Vaccinations: 1, Medications: 1}) {
		t.Fatalf("rows %+v, %v", rows, err)
	}
	d, err := s.Visit(ctx, k.ID, v.ID)
	if err != nil || len(d.Links.Growth) != 1 || d.Links.Growth[0].MeasuredOn != "2026-03-01" ||
		len(d.Links.Vaccinations) != 1 || len(d.Links.Vaccinations[0].Antigens) != 2 ||
		len(d.Links.Medications) != 1 || d.Links.Medications[0].Name != "Amoxicillin" {
		t.Fatalf("links %+v, %v", d.Links, err)
	}
}

func TestOutcomeRulesOnUpdate(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	past := mustVisit(t, s, k.ID, ana.ID, newVisit(t0.Add(-time.Hour)))

	upcoming := mustVisit(t, s, k.ID, ana.ID, newVisit(t0.Add(time.Hour)))
	upcoming.OutcomeNotes = strp("Healthy")
	if _, err := s.UpdateVisit(ctx, k.ID, ana.ID, upcoming.ID, t0, set(upcoming)); !errors.Is(err, ErrNotPast) {
		t.Errorf("notes on an upcoming visit: %v, want ErrNotPast", err)
	}

	withNotes := past
	withNotes.OutcomeNotes = strp("Healthy")
	stored, err := s.UpdateVisit(ctx, k.ID, ana.ID, withNotes.ID, t0, set(withNotes))
	if err != nil {
		t.Fatal(err)
	}
	cancel := stored.Visit
	cancelled := t0
	cancel.CancelledAt = &cancelled
	if _, err := s.UpdateVisit(ctx, k.ID, ana.ID, cancel.ID, t0, set(cancel)); !errors.Is(err, ErrHasOutcome) {
		t.Errorf("cancel with notes: %v, want ErrHasOutcome", err)
	}
	move := stored.Visit
	move.StartsAt = t0.Add(24 * time.Hour)
	if _, err := s.UpdateVisit(ctx, k.ID, ana.ID, move.ID, t0, set(move)); !errors.Is(err, ErrHasOutcome) {
		t.Errorf("move to the future with notes: %v, want ErrHasOutcome", err)
	}
	if after, _ := s.Visit(ctx, k.ID, past.ID); after.Revision != stored.Revision || after.CancelledAt != nil {
		t.Errorf("a refused write changed the row: %+v", after.Visit)
	}

	linked := mustVisit(t, s, k.ID, ana.ID, newVisit(t0.Add(-2*time.Hour)))
	g := mustGrowth(t, s, k.ID, ana.ID, newGrowth(t, "2026-03-01"))
	linkRow(t, s, "growth_measurements", "id", g.ID, linked.ID)
	linked.CancelledAt = &cancelled
	if _, err := s.UpdateVisit(ctx, k.ID, ana.ID, linked.ID, t0, set(linked)); !errors.Is(err, ErrHasOutcome) {
		t.Errorf("cancel with a linked row: %v, want ErrHasOutcome", err)
	}

	plain := mustVisit(t, s, k.ID, ana.ID, newVisit(t0.Add(-3*time.Hour)))
	plain.CancelledAt = &cancelled
	if _, err := s.UpdateVisit(ctx, k.ID, ana.ID, plain.ID, t0, set(plain)); err != nil {
		t.Errorf("cancel a past visit without an outcome: %v", err)
	}
}

func TestDeleteVisitLinked(t *testing.T) {
	for _, deleteLinked := range []bool{true, false} {
		s := openMem(t)
		ana := mustParent(t, s, "ana")
		k := mustKid(t, s, ana.ID, "Lía")
		v := mustVisit(t, s, k.ID, ana.ID, newVisit(t0.Add(-time.Hour)))
		g := mustGrowth(t, s, k.ID, ana.ID, newGrowth(t, "2026-03-01"))
		a, err := s.CreateAdministration(ctx, k.ID, ana.ID, model.Administration{GivenOn: "2026-03-01", Antigens: []string{"dtap", "ipv"}}, t0)
		if err != nil {
			t.Fatal(err)
		}
		m := mustCourse(t, s, k.ID, ana.ID, timesCourse(t0.Add(-time.Hour), "08:00", "20:00"), t0)
		if _, err := s.CreateDose(ctx, k.ID, ana.ID, model.MedicationDose{MedicationID: m.ID, GivenAt: t0.Add(-time.Minute)}, t0); err != nil {
			t.Fatal(err)
		}
		linkRow(t, s, "growth_measurements", "id", g.ID, v.ID)
		linkRow(t, s, "vaccine_doses", "administration", a.ID, v.ID)
		linkRow(t, s, "medications", "id", m.ID, v.ID)
		if err := s.DeleteVisit(ctx, k.ID, ana.ID, v.ID, deleteLinked); err != nil {
			t.Fatal(err)
		}
		want, linkedLeft := 0, 0
		if !deleteLinked {
			want = 1
		}
		for _, table := range []string{"growth_measurements", "medications"} {
			var n, l int
			if err := s.r.QueryRow(`SELECT COUNT(*), COUNT(visit_id) FROM `+table+` WHERE kid_id = ?`, k.ID).Scan(&n, &l); err != nil || n != want || l != linkedLeft {
				t.Errorf("linked=%v: %s rows %d (linked %d), want %d (linked 0); err %v", deleteLinked, table, n, l, want, err)
			}
		}
		var doses, meds, reminders int
		_ = s.r.QueryRow(`SELECT COUNT(*) FROM vaccine_doses WHERE kid_id = ?`, k.ID).Scan(&doses)
		_ = s.r.QueryRow(`SELECT COUNT(*) FROM medication_doses WHERE kid_id = ?`, k.ID).Scan(&meds)
		_ = s.r.QueryRow(`SELECT COUNT(*) FROM reminders WHERE kid_id = ?`, k.ID).Scan(&reminders)
		if deleteLinked && (doses != 0 || meds != 0 || reminders != 0) {
			t.Errorf("linked=delete left %d vaccine rows, %d doses, %d reminders", doses, meds, reminders)
		}
		if !deleteLinked && (doses != 2 || meds != 1 || reminders == 0) {
			t.Errorf("linked=keep left %d vaccine rows, %d doses, %d reminders", doses, meds, reminders)
		}
	}
}

func TestVisitAccess(t *testing.T) {
	s := openMem(t)
	ana, cy := mustParent(t, s, "ana"), mustParent(t, s, "cy")
	k, other := mustKid(t, s, ana.ID, "Lía"), mustKid(t, s, cy.ID, "Teo")
	v := mustVisit(t, s, k.ID, ana.ID, newVisit(t0))
	if _, err := s.CreateVisit(ctx, k.ID, cy.ID, newVisit(t0), t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("create without a grant: %v", err)
	}
	if _, err := s.UpdateVisit(ctx, k.ID, cy.ID, v.ID, t0, set(v)); !errors.Is(err, ErrNotFound) {
		t.Errorf("update without a grant: %v", err)
	}
	if err := s.DeleteVisit(ctx, k.ID, cy.ID, v.ID, true); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete without a grant: %v", err)
	}
	if _, err := s.Visit(ctx, other.ID, v.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("another kid's visit: %v", err)
	}
	if err := s.DeleteVisit(ctx, other.ID, cy.ID, v.ID, true); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete another kid's visit: %v", err)
	}
	bo := mustParent(t, s, "bo")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)
	if err := s.SetParentDisabled(ctx, bo.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateVisit(ctx, k.ID, bo.ID, newVisit(t0), t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("create as a disabled caregiver: %v", err)
	}
}

func TestVisitCapAndCascade(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	for i := range constants.MaxVisitsPerKid {
		if _, err := s.w.Exec(`INSERT INTO visits(id, kid_id, starts_at, kind, created_by, updated_by, created_at, updated_at)
			VALUES (?, ?, ?, 'other', ?, ?, 100, 100)`, model.NewID(), k.ID, 1000+i, ana.ID, ana.ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.CreateVisit(ctx, k.ID, ana.ID, newVisit(t0), t0); !errors.Is(err, ErrConflict) {
		t.Errorf("visit over the cap: %v, want ErrConflict", err)
	}
	if err := s.DeleteKid(ctx, k.ID, ana.ID); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := s.r.QueryRow(`SELECT COUNT(*) FROM visits`).Scan(&n); err != nil || n != 0 {
		t.Errorf("visits after the kid's delete: %d, err %v", n, err)
	}
}

func TestUpdateAdministrationKeepsVisitID(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	v := mustVisit(t, s, k.ID, ana.ID, newVisit(t0.Add(-time.Hour)))
	a, err := s.CreateAdministration(ctx, k.ID, ana.ID, model.Administration{GivenOn: "2026-03-01", Antigens: []string{"dtap"}}, t0)
	if err != nil {
		t.Fatal(err)
	}
	linkRow(t, s, "vaccine_doses", "administration", a.ID, v.ID)
	a.Antigens = []string{"dtap", "ipv"}
	if err := s.UpdateAdministration(ctx, k.ID, ana.ID, a, t0); err != nil {
		t.Fatal(err)
	}
	var rows, linked int
	if err := s.r.QueryRow(`SELECT COUNT(*), COUNT(visit_id) FROM vaccine_doses WHERE administration = ?`, a.ID).Scan(&rows, &linked); err != nil ||
		rows != 2 || linked != 2 {
		t.Errorf("after adding an antigen: %d rows, %d linked, err %v; want 2 and 2", rows, linked, err)
	}
	got, err := s.Administration(ctx, k.ID, a.ID)
	if err != nil || got.VisitID == nil || *got.VisitID != v.ID {
		t.Errorf("administration read back with visit %v, err %v", got.VisitID, err)
	}
}

func TestOutcomeCreatesLinkedRows(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	v := mustVisit(t, s, k.ID, ana.ID, newVisit(t0.Add(-time.Hour)))
	g := newGrowth(t, "2026-11-20")
	o := VisitOutcome{NotesSet: true, Notes: strp("Healthy"), Growth: &g,
		Vaccinations: []model.Administration{{GivenOn: "2026-11-20", Antigens: []string{"dtap", "ipv"}}, {GivenOn: "2026-11-20", Antigens: []string{"hepb"}}},
		Medications:  []model.Medication{timesCourse(v.StartsAt, "08:00", "20:00")}}
	d, err := s.RecordOutcome(ctx, k.ID, ana.ID, v.ID, 0, o, t0)
	if err != nil {
		t.Fatal(err)
	}
	if d.Revision != 1 || d.OutcomeNotes == nil || *d.OutcomeNotes != "Healthy" || len(d.Links.Growth) != 1 ||
		len(d.Links.Vaccinations) != 2 || len(d.Links.Medications) != 1 {
		t.Fatalf("detail %+v", d)
	}
	var reminders int
	if err := s.r.QueryRow(`SELECT COUNT(*) FROM reminders WHERE medication_id = ?`, d.Links.Medications[0].ID).Scan(&reminders); err != nil || reminders == 0 {
		t.Errorf("the course's reminders: %d, err %v; want materialised", reminders, err)
	}
}

func TestOutcomeSecondSubmitWritesNothing(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	v := mustVisit(t, s, k.ID, ana.ID, newVisit(t0.Add(-time.Hour)))
	o := VisitOutcome{Vaccinations: []model.Administration{{GivenOn: "2026-11-20", Antigens: []string{"dtap"}}}}
	if _, err := s.RecordOutcome(ctx, k.ID, ana.ID, v.ID, 0, o, t0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RecordOutcome(ctx, k.ID, ana.ID, v.ID, 0, o, t0); !errors.Is(err, ErrStale) {
		t.Fatalf("second submit: %v, want ErrStale", err)
	}
	var n int
	if err := s.r.QueryRow(`SELECT COUNT(*) FROM vaccine_doses WHERE kid_id = ?`, k.ID).Scan(&n); err != nil || n != 1 {
		t.Errorf("vaccine rows after a double submit: %d, err %v; want 1", n, err)
	}
}

func TestOutcomeRollsBackOnLimit(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	v := mustVisit(t, s, k.ID, ana.ID, newVisit(t0.Add(-time.Hour)))
	for range constants.MaxMedicationsPerKid {
		mustCourse(t, s, k.ID, ana.ID, model.Medication{Name: "Saline", Dose: "1 drop", StartAt: t0, Schedule: schedule.Schedule{Kind: schedule.KindNone}}, t0)
	}
	g := newGrowth(t, "2026-11-20")
	o := VisitOutcome{NotesSet: true, Notes: strp("Healthy"), Growth: &g, Medications: []model.Medication{timesCourse(t0, "08:00")}}
	_, err := s.RecordOutcome(ctx, k.ID, ana.ID, v.ID, 0, o, t0)
	var row *OutcomeRowError
	if !errors.As(err, &row) || row.Section != "medications" || row.Index != 0 || !errors.Is(err, ErrConflict) {
		t.Fatalf("over the course cap: %v, want OutcomeRowError{medications, 0, ErrConflict}", err)
	}
	d, _ := s.Visit(ctx, k.ID, v.ID)
	if d.OutcomeNotes != nil || d.Revision != 0 || len(d.Links.Growth) != 0 {
		t.Errorf("a refused outcome left %+v", d)
	}
}

func TestOutcomeRefusals(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	upcoming := mustVisit(t, s, k.ID, ana.ID, newVisit(t0.Add(time.Hour)))
	notes := VisitOutcome{NotesSet: true, Notes: strp("x")}
	if _, err := s.RecordOutcome(ctx, k.ID, ana.ID, upcoming.ID, 0, notes, t0); !errors.Is(err, ErrNotPast) {
		t.Errorf("upcoming: %v, want ErrNotPast", err)
	}
	past := mustVisit(t, s, k.ID, ana.ID, newVisit(t0.Add(-time.Hour)))
	bad := VisitOutcome{Vaccinations: []model.Administration{{GivenOn: "2026-11-20", Antigens: []string{"dtap"}}, {GivenOn: "2026-11-20", Antigens: []string{"ipv", "ipv"}}}}
	_, err := s.RecordOutcome(ctx, k.ID, ana.ID, past.ID, 0, bad, t0)
	var row *OutcomeRowError
	if !errors.As(err, &row) || row.Section != "vaccinations" || row.Index != 1 || !errors.Is(err, ErrInvalidAntigens) {
		t.Errorf("a repeated antigen in row 1: %v", err)
	}
	if _, err := s.RecordOutcome(ctx, k.ID, ana.ID, "vi-missing", 0, notes, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown visit: %v", err)
	}
	cy := mustParent(t, s, "cy")
	if _, err := s.RecordOutcome(ctx, k.ID, cy.ID, past.ID, 0, notes, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("no grant: %v", err)
	}
}

func TestUpdateVisitMergesOnTheRowReadInTheWrite(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	v := mustVisit(t, s, k.ID, ana.ID, newVisit(t0.Add(-time.Hour)))
	if _, err := s.RecordOutcome(ctx, k.ID, ana.ID, v.ID, 0, VisitOutcome{NotesSet: true, Notes: strp("Healthy")}, t0); err != nil {
		t.Fatal(err)
	}
	d, err := s.UpdateVisit(ctx, k.ID, ana.ID, v.ID, t0, func(stored model.Visit) (model.Visit, error) {
		stored.DurationMin = 60
		return stored, nil
	})
	if err != nil || d.OutcomeNotes == nil || *d.OutcomeNotes != "Healthy" || d.DurationMin != 60 || d.Revision != 2 {
		t.Errorf("an edit of the length after notes were saved: %+v, %v", d.Visit, err)
	}
}
