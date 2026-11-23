package store

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

var day = time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)

func clock(h, m int) time.Time {
	return day.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute)
}

func doseOf(medicationID string, givenAt time.Time) model.MedicationDose {
	return model.MedicationDose{MedicationID: medicationID, GivenAt: givenAt}
}

func mustDose(t *testing.T, s *Store, kidID, actorID string, v model.MedicationDose, now time.Time) model.MedicationDose {
	t.Helper()
	got, err := s.CreateDose(ctx, kidID, actorID, v, now)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// doseCourse is a UTC kid with a 08:00 and 14:00 course created at 10:00 (so 14:00 is materialised) and the 08:00 row
// of the day inserted by hand.
func doseCourse(t *testing.T) (s *Store, kid model.Kid, parent model.Parent, course model.Medication) {
	t.Helper()
	s = openMem(t)
	parent = mustParent(t, s, "ana")
	kid = mustKidIn(t, s, parent.ID, "Lía", "UTC")
	course = mustCourse(t, s, kid.ID, parent.ID, timesCourse(clock(0, 0), "08:00", "14:00"), clock(10, 0))
	execOK(t, s, `INSERT INTO reminders(id, kid_id, medication_id, due_at, state, created_at) VALUES (?, ?, ?, ?, 'pending', ?)`,
		model.NewID(), kid.ID, course.ID, clock(8, 0).Unix(), clock(8, 0).Unix())
	return
}

func states(t *testing.T, s *Store, medicationID string) map[time.Time]string {
	t.Helper()
	out := map[time.Time]string{}
	for _, r := range reminderRows(t, s, medicationID) {
		out[unixTime(r.DueAt)] = r.State
	}
	return out
}

func TestDoseCRUD(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)
	course := mustCourse(t, s, k.ID, ana.ID, timesCourse(clock(0, 0), "20:00"), clock(10, 0))

	in := doseOf(course.ID, clock(9, 0).Add(500*time.Millisecond))
	in.Amount, in.Note = strp("2.5 ml"), strp("after food")
	got := mustDose(t, s, k.ID, bo.ID, in, clock(10, 0))
	want := model.MedicationDose{Entry: model.Entry{ID: got.ID, KidID: k.ID, Note: strp("after food"), CreatedBy: bo.ID, UpdatedBy: bo.ID,
		CreatedAt: clock(10, 0), UpdatedAt: clock(10, 0)}, MedicationID: course.ID, GivenAt: clock(9, 0), Amount: strp("2.5 ml")}
	if got.ID == "" || !reflect.DeepEqual(got, want) {
		t.Fatalf("created = %+v, want %+v", got, want)
	}
	if read, err := s.Dose(ctx, k.ID, got.ID); err != nil || !reflect.DeepEqual(read, want) {
		t.Fatalf("read = %+v, %v", read, err)
	}

	edit := got
	edit.GivenAt, edit.Amount, edit.Note = clock(9, 30), nil, nil
	if err := s.UpdateDose(ctx, k.ID, ana.ID, edit, clock(11, 0)); err != nil {
		t.Fatal(err)
	}
	read, err := s.Dose(ctx, k.ID, got.ID)
	if err != nil || !read.GivenAt.Equal(clock(9, 30)) || read.Amount != nil || read.Note != nil || read.UpdatedBy != ana.ID ||
		!read.UpdatedAt.Equal(clock(11, 0)) || read.CreatedBy != bo.ID || read.MedicationID != course.ID {
		t.Errorf("after update = %+v, %v", read, err)
	}
	if err := s.DeleteDose(ctx, k.ID, bo.ID, got.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Dose(ctx, k.ID, got.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("read after delete: %v", err)
	}
	if err := s.DeleteDose(ctx, k.ID, bo.ID, got.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: %v", err)
	}
}

func TestDoseUpdateKeepsTheCourse(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	a := mustCourse(t, s, k.ID, ana.ID, timesCourse(clock(0, 0), "20:00"), clock(10, 0))
	b := mustCourse(t, s, k.ID, ana.ID, timesCourse(clock(0, 0), "21:00"), clock(10, 0))
	d := mustDose(t, s, k.ID, ana.ID, doseOf(a.ID, clock(9, 0)), clock(10, 0))
	edit := d
	edit.MedicationID = b.ID
	if err := s.UpdateDose(ctx, k.ID, ana.ID, edit, clock(10, 0)); err != nil {
		t.Fatal(err)
	}
	if read, err := s.Dose(ctx, k.ID, d.ID); err != nil || read.MedicationID != a.ID {
		t.Errorf("the dose moved to course %q (%v), want %q", read.MedicationID, err, a.ID)
	}
}

func TestDoseBelongsToItsKid(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	mine, other := mustKidIn(t, s, ana.ID, "Mine", "UTC"), mustKidIn(t, s, ana.ID, "Other", "UTC")
	mineCourse := mustCourse(t, s, mine.ID, ana.ID, timesCourse(clock(0, 0), "20:00"), clock(10, 0))
	otherCourse := mustCourse(t, s, other.ID, ana.ID, timesCourse(clock(0, 0), "20:00"), clock(10, 0))
	d := mustDose(t, s, mine.ID, ana.ID, doseOf(mineCourse.ID, clock(9, 0)), clock(10, 0))

	if _, err := s.Dose(ctx, other.ID, d.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("read through another kid: %v", err)
	}
	if err := s.UpdateDose(ctx, other.ID, ana.ID, d, clock(10, 0)); !errors.Is(err, ErrNotFound) {
		t.Errorf("update through another kid: %v", err)
	}
	if err := s.DeleteDose(ctx, other.ID, ana.ID, d.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete through another kid: %v", err)
	}
	if list, err := s.ListDoses(ctx, other.ID, "", dayQuery()); err != nil || len(list) != 0 {
		t.Errorf("other kid's list = %+v, %v", list, err)
	}
	for name, id := range map[string]string{"another kid's course": otherCourse.ID, "an unknown course": model.NewID(), "no course": "", "a malformed id": "x'; DROP"} {
		if _, err := s.CreateDose(ctx, mine.ID, ana.ID, doseOf(id, clock(9, 0)), clock(10, 0)); !errors.Is(err, ErrInvalidMedication) {
			t.Errorf("a dose for %s: %v, want ErrInvalidMedication", name, err)
		}
	}
	if n := countOf(t, s, "medication_doses"); n != 1 {
		t.Errorf("%d doses stored, want the one", n)
	}
}

func TestDoseWriteByDisabledActor(t *testing.T) {
	s := openMem(t)
	ana, bo, cy := mustParent(t, s, "ana"), mustParent(t, s, "bo"), mustParent(t, s, "cy")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)
	course := mustCourse(t, s, k.ID, ana.ID, timesCourse(clock(0, 0), "20:00"), clock(10, 0))
	d := mustDose(t, s, k.ID, ana.ID, doseOf(course.ID, clock(9, 0)), clock(10, 0))
	if err := s.SetParentDisabled(ctx, bo.ID, true, clock(10, 0)); err != nil {
		t.Fatal(err)
	}
	for name, actor := range map[string]string{"disabled caregiver": bo.ID, "no grant": cy.ID} {
		if _, err := s.CreateDose(ctx, k.ID, actor, doseOf(course.ID, clock(9, 0)), clock(10, 0)); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s create: %v", name, err)
		}
		if err := s.UpdateDose(ctx, k.ID, actor, d, clock(10, 0)); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s update: %v", name, err)
		}
		if err := s.DeleteDose(ctx, k.ID, actor, d.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s delete: %v", name, err)
		}
	}
	if n := countOf(t, s, "medication_doses"); n != 1 {
		t.Errorf("%d doses stored, want the one", n)
	}
}

func TestDoseCap(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k, other := mustKidIn(t, s, ana.ID, "Lía", "UTC"), mustKidIn(t, s, ana.ID, "Otro", "UTC")
	course := mustCourse(t, s, k.ID, ana.ID, timesCourse(clock(0, 0), "20:00"), clock(10, 0))
	otherCourse := mustCourse(t, s, other.ID, ana.ID, timesCourse(clock(0, 0), "20:00"), clock(10, 0))
	for i := range constants.MaxMedicationDosesPerKid {
		seedDose(t, s, k.ID, course.ID, ana.ID, clock(0, 0).Add(time.Duration(i)*time.Second), nil)
	}
	if _, err := s.CreateDose(ctx, k.ID, ana.ID, doseOf(course.ID, clock(9, 0)), clock(10, 0)); !errors.Is(err, ErrConflict) {
		t.Errorf("dose %d: %v, want ErrConflict", constants.MaxMedicationDosesPerKid+1, err)
	}
	if _, err := s.CreateDose(ctx, other.ID, ana.ID, doseOf(otherCourse.ID, clock(9, 0)), clock(10, 0)); err != nil {
		t.Errorf("another kid's first dose: %v", err)
	}
}

func dayQuery() ListQuery {
	return ListQuery{From: day.Add(-24 * time.Hour), To: day.Add(48 * time.Hour), Limit: 100}
}

func TestListDosesFilterAndCursor(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	a := mustCourse(t, s, k.ID, ana.ID, timesCourse(clock(0, 0), "20:00"), clock(10, 0))
	b := mustCourse(t, s, k.ID, ana.ID, timesCourse(clock(0, 0), "21:00"), clock(10, 0))
	var ids []string
	for i := range 5 {
		course := a.ID
		if i%2 == 1 {
			course = b.ID
		}
		ids = append(ids, mustDose(t, s, k.ID, ana.ID, doseOf(course, clock(1+i, 0)), clock(10, 0)).ID)
	}

	all, err := s.ListDoses(ctx, k.ID, "", dayQuery())
	if err != nil || len(all) != 5 {
		t.Fatalf("all = %d, %v", len(all), err)
	}
	for i, d := range all {
		if d.ID != ids[4-i] {
			t.Errorf("all[%d] = %s, want newest first (%s)", i, d.ID, ids[4-i])
		}
	}
	only, err := s.ListDoses(ctx, k.ID, b.ID, dayQuery())
	if err != nil || len(only) != 2 || only[0].ID != ids[3] || only[1].ID != ids[1] {
		t.Errorf("course b = %+v, %v; want doses 3 and 1", only, err)
	}
	if none, err := s.ListDoses(ctx, k.ID, model.NewID(), dayQuery()); err != nil || none == nil || len(none) != 0 {
		t.Errorf("unknown course = %#v, %v; want an empty, non-nil list", none, err)
	}
	q := dayQuery()
	q.Limit = 2
	q.After = &Cursor{At: all[1].GivenAt.Unix(), ID: all[1].ID}
	page, err := s.ListDoses(ctx, k.ID, "", q)
	if err != nil || len(page) != 2 || page[0].ID != ids[2] || page[1].ID != ids[1] {
		t.Errorf("after the second = %+v, %v", page, err)
	}
	q = dayQuery()
	q.From, q.To = clock(2, 0), clock(4, 0)
	if win, err := s.ListDoses(ctx, k.ID, "", q); err != nil || len(win) != 2 || win[0].ID != ids[2] || win[1].ID != ids[1] {
		t.Errorf("window [2:00, 4:00) = %+v, %v", win, err)
	}
}

func TestDoseMarksOneReminderDone(t *testing.T) {
	type tc struct {
		name  string
		setup func(t *testing.T, s *Store, kid model.Kid, course model.Medication)
		given time.Time
		now   time.Time
		want  map[time.Time]string // reminders after the dose, by due time
	}
	pend := func(h, m int) time.Time { return clock(h, m) }
	next := day.Add(24*time.Hour + 8*time.Hour)
	for _, c := range []tc{
		{name: "07:31 marks 08:00", given: clock(7, 31), now: clock(10, 0),
			want: map[time.Time]string{pend(8, 0): "done", pend(14, 0): "pending"}},
		{name: "08:01 marks 08:00", given: clock(8, 1), now: clock(10, 0),
			want: map[time.Time]string{pend(8, 0): "done", pend(14, 0): "pending"}},
		{name: "07:29 marks nothing", given: clock(7, 29), now: clock(10, 0),
			want: map[time.Time]string{pend(8, 0): "pending", pend(14, 0): "pending"}},
		{name: "a dose older than every row marks nothing", given: clock(3, 0), now: clock(10, 0),
			want: map[time.Time]string{pend(8, 0): "pending", pend(14, 0): "pending"}},
		{name: "13:40 marks only 14:00 and keeps a next row", given: clock(13, 40), now: clock(13, 45),
			want: map[time.Time]string{pend(8, 0): "pending", pend(14, 0): "done", next: "pending"}},
		{name: "an expired row becomes done", given: clock(8, 5), now: clock(10, 0),
			setup: func(t *testing.T, s *Store, _ model.Kid, course model.Medication) {
				execOK(t, s, `UPDATE reminders SET state = 'expired' WHERE medication_id = ? AND due_at = ?`, course.ID, clock(8, 0).Unix())
			},
			want: map[time.Time]string{pend(8, 0): "done", pend(14, 0): "pending"}},
		{name: "a cancelled row stays cancelled", given: clock(8, 5), now: clock(10, 0),
			setup: func(t *testing.T, s *Store, _ model.Kid, course model.Medication) {
				execOK(t, s, `UPDATE reminders SET state = 'cancelled' WHERE medication_id = ? AND due_at = ?`, course.ID, clock(8, 0).Unix())
			},
			want: map[time.Time]string{pend(8, 0): "cancelled", pend(14, 0): "pending"}},
		{name: "a dose between two occurrences marks only one", given: clock(11, 0), now: clock(11, 0),
			want: map[time.Time]string{pend(8, 0): "done", pend(14, 0): "pending"}},
		{name: "a dose for the previous occurrence", given: clock(8, 5), now: clock(13, 50),
			want: map[time.Time]string{pend(8, 0): "done", pend(14, 0): "pending"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, k, ana, course := doseCourse(t)
			if c.setup != nil {
				c.setup(t, s, k, course)
			}
			mustDose(t, s, k.ID, ana.ID, doseOf(course.ID, c.given), c.now)
			if got := states(t, s, course.ID); !reflect.DeepEqual(got, c.want) {
				t.Errorf("reminders = %v, want %v", got, c.want)
			}
		})
	}
}

func TestDoseDoneWindowBoundary(t *testing.T) {
	for name, c := range map[string]struct {
		before time.Duration
		done   bool
	}{
		"29 minutes before due":   {29 * time.Minute, true},
		"exactly the lead":        {constants.DoseDoneLead, true},
		"31 minutes before due":   {31 * time.Minute, false},
		"one second inside lead":  {constants.DoseDoneLead - time.Second, true},
		"one second outside lead": {constants.DoseDoneLead + time.Second, false},
	} {
		t.Run(name, func(t *testing.T) {
			s, k, ana, course := doseCourse(t)
			mustDose(t, s, k.ID, ana.ID, doseOf(course.ID, clock(8, 0).Add(-c.before)), clock(10, 0))
			want := "pending"
			if c.done {
				want = "done"
			}
			if got := states(t, s, course.ID)[clock(8, 0)]; got != want {
				t.Errorf("08:00 is %s, want %s", got, want)
			}
			if got := states(t, s, course.ID)[clock(14, 0)]; got != "pending" {
				t.Errorf("14:00 is %s, want pending", got)
			}
		})
	}
}

func TestDoseMarksTheNextRowForTheNextDue(t *testing.T) {
	s, k, ana, course := doseCourse(t)
	mustDose(t, s, k.ID, ana.ID, doseOf(course.ID, clock(13, 40)), clock(13, 45))
	got, err := s.Medication(ctx, k.ID, course.ID, clock(13, 45))
	want := day.Add(32 * time.Hour)
	if err != nil || got.NextDue == nil || !got.NextDue.Equal(want) {
		t.Errorf("next due = %v, %v; want %v", got.NextDue, err, want)
	}
	if got.LastDose == nil || !got.LastDose.GivenAt.Equal(clock(13, 40)) || got.DoseCount != 1 {
		t.Errorf("last dose %+v, count %d", got.LastDose, got.DoseCount)
	}
}

func TestDoseForAnotherCourseMarksNothing(t *testing.T) {
	s, k, ana, course := doseCourse(t)
	other := mustCourse(t, s, k.ID, ana.ID, timesCourse(clock(0, 0), "20:00"), clock(10, 0))
	mustDose(t, s, k.ID, ana.ID, doseOf(other.ID, clock(7, 50)), clock(10, 0))
	if got := states(t, s, course.ID); got[clock(8, 0)] != "pending" || got[clock(14, 0)] != "pending" {
		t.Errorf("another medication's dose changed %v", got)
	}
}

func TestTwoDosesMarkOneOccurrence(t *testing.T) {
	s, k, ana, course := doseCourse(t)
	mustDose(t, s, k.ID, ana.ID, doseOf(course.ID, clock(8, 0)), clock(10, 0))
	mustDose(t, s, k.ID, ana.ID, doseOf(course.ID, clock(8, 20)), clock(10, 0))
	mustDose(t, s, k.ID, ana.ID, doseOf(course.ID, clock(9, 0)), clock(10, 0))
	want := map[time.Time]string{clock(8, 0): "done", clock(14, 0): "pending"}
	if got := states(t, s, course.ID); !reflect.DeepEqual(got, want) {
		t.Errorf("reminders = %v, want %v", got, want)
	}
	if n := countOf(t, s, "medication_doses"); n != 3 {
		t.Errorf("%d doses, want 3", n)
	}
}

func TestDoseForAnUnscheduledCourseMarksNothing(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	v := timesCourse(clock(0, 0))
	v.Schedule.Kind, v.Schedule.Times = "none", nil
	course := mustCourse(t, s, k.ID, ana.ID, v, clock(10, 0))
	execOK(t, s, `INSERT INTO reminders(id, kid_id, medication_id, due_at, state, created_at) VALUES (?, ?, ?, ?, 'pending', ?)`,
		model.NewID(), k.ID, course.ID, clock(8, 0).Unix(), clock(8, 0).Unix())
	mustDose(t, s, k.ID, ana.ID, doseOf(course.ID, clock(8, 5)), clock(10, 0))
	if got := states(t, s, course.ID); !reflect.DeepEqual(got, map[time.Time]string{clock(8, 0): "pending"}) {
		t.Errorf("reminders = %v, want the stale row untouched", got)
	}
}

func TestInactiveCourseStillTakesDoses(t *testing.T) {
	s, k, ana, course := doseCourse(t)
	course.Active = false
	if _, err := s.UpdateMedication(ctx, k.ID, ana.ID, course, clock(10, 0)); err != nil {
		t.Fatal(err)
	}
	d := mustDose(t, s, k.ID, ana.ID, doseOf(course.ID, clock(9, 0)), clock(10, 0))
	if got, err := s.Dose(ctx, k.ID, d.ID); err != nil || got.MedicationID != course.ID {
		t.Errorf("dose = %+v, %v", got, err)
	}
	// A leftover pending row of an inactive course may be marked done: nothing sends a done row.
	if got, want := states(t, s, course.ID), map[time.Time]string{clock(8, 0): "done"}; !reflect.DeepEqual(got, want) {
		t.Errorf("reminders after a dose for an inactive course = %v, want %v", got, want)
	}
}

func TestEditingADoseMarksTheNewOccurrenceAndKeepsTheOld(t *testing.T) {
	s, k, ana, course := doseCourse(t)
	d := mustDose(t, s, k.ID, ana.ID, doseOf(course.ID, clock(8, 0)), clock(10, 0))
	// A note-only edit changes nothing else.
	edit := d
	edit.Note = strp("n")
	if err := s.UpdateDose(ctx, k.ID, ana.ID, edit, clock(10, 0)); err != nil {
		t.Fatal(err)
	}
	if got := states(t, s, course.ID); got[clock(8, 0)] != "done" || got[clock(14, 0)] != "pending" {
		t.Fatalf("after a note edit: %v", got)
	}
	// Moving it to 14:00 marks 14:00 and leaves 08:00 done: no history.
	edit.GivenAt = clock(13, 50)
	if err := s.UpdateDose(ctx, k.ID, ana.ID, edit, clock(14, 0)); err != nil {
		t.Fatal(err)
	}
	got := states(t, s, course.ID)
	if got[clock(8, 0)] != "done" || got[clock(14, 0)] != "done" || got[day.Add(32*time.Hour)] != "pending" {
		t.Errorf("after moving the dose: %v", got)
	}
}

func TestDeleteDoseKeepsDone(t *testing.T) {
	s, k, ana, course := doseCourse(t)
	d := mustDose(t, s, k.ID, ana.ID, doseOf(course.ID, clock(8, 0)), clock(10, 0))
	if err := s.DeleteDose(ctx, k.ID, ana.ID, d.ID); err != nil {
		t.Fatal(err)
	}
	if got := states(t, s, course.ID); got[clock(8, 0)] != "done" {
		t.Errorf("deleting the dose undid done: %v", got)
	}
	if got, err := s.Medication(ctx, k.ID, course.ID, clock(10, 0)); err != nil || got.DoseCount != 0 || got.LastDose != nil {
		t.Errorf("course after the delete = %+v, %v", got, err)
	}
}

func TestDoseDoesNotTouchOtherRowsOrDeliveries(t *testing.T) {
	s, k, ana, course := doseCourse(t)
	var rid string
	if err := s.w.QueryRow(`SELECT id FROM reminders WHERE medication_id = ? AND due_at = ?`, course.ID, clock(8, 0).Unix()).Scan(&rid); err != nil {
		t.Fatal(err)
	}
	execOK(t, s, `INSERT INTO reminder_deliveries(id, reminder_id, channel, target_id, parent_id, state, attempts, next_attempt_at) VALUES (?, ?, 'push', 'dev', ?, 'sent', 1, 0)`,
		model.NewID(), rid, ana.ID)
	mustDose(t, s, k.ID, ana.ID, doseOf(course.ID, clock(13, 40)), clock(13, 45))
	if n := countOf(t, s, "reminder_deliveries"); n != 1 {
		t.Errorf("%d deliveries, want the one kept", n)
	}
	if got := states(t, s, course.ID); got[clock(8, 0)] != "pending" || got[clock(14, 0)] != "done" {
		t.Errorf("reminders = %v", got)
	}
}

func TestDoseOnADSTDay(t *testing.T) {
	const zone = "America/New_York"
	ny, err := time.LoadLocation(zone)
	if err != nil {
		t.Skip("no tzdata")
	}
	local := func(y int, m time.Month, d, h, mi int) time.Time { return time.Date(y, m, d, h, mi, 0, 0, ny).UTC() }
	t.Run("spring forward gap time fires at the gap's end", func(t *testing.T) {
		s := openMem(t)
		ana := mustParent(t, s, "ana")
		k := mustKidIn(t, s, ana.ID, "Lía", zone)
		now := time.Date(2026, 3, 8, 3, 10, 0, 0, ny) // 07:10Z, EDT
		course := mustCourse(t, s, k.ID, ana.ID, timesCourse(time.Date(2026, 3, 7, 0, 0, 0, 0, ny), "02:30", "12:00"), now)
		gapEnd := local(2026, 3, 8, 3, 0)
		if got := states(t, s, course.ID); got[gapEnd] != "pending" {
			t.Fatalf("rows before the dose: %v", got)
		}
		mustDose(t, s, k.ID, ana.ID, doseOf(course.ID, gapEnd.Add(-constants.DoseDoneLead-time.Second)), now)
		if got := states(t, s, course.ID)[gapEnd]; got != "pending" {
			t.Errorf("a dose 30 minutes and a second before 03:00 EDT: %s", got)
		}
		mustDose(t, s, k.ID, ana.ID, doseOf(course.ID, gapEnd.Add(-constants.DoseDoneLead)), now)
		got := states(t, s, course.ID)
		if got[gapEnd] != "done" || got[local(2026, 3, 8, 12, 0)] != "pending" {
			t.Errorf("rows after the dose: %v", got)
		}
	})
	t.Run("the next row after done crosses the change", func(t *testing.T) {
		s := openMem(t)
		ana := mustParent(t, s, "ana")
		k := mustKidIn(t, s, ana.ID, "Lía", zone)
		now := time.Date(2026, 3, 7, 8, 10, 0, 0, ny) // EST
		course := mustCourse(t, s, k.ID, ana.ID, timesCourse(time.Date(2026, 3, 7, 0, 0, 0, 0, ny), "08:00"), now)
		mustDose(t, s, k.ID, ana.ID, doseOf(course.ID, now), now)
		got := states(t, s, course.ID)
		want := map[time.Time]string{local(2026, 3, 7, 8, 0): "done", local(2026, 3, 8, 8, 0): "pending"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("rows = %v, want %v", got, want)
		}
		if d := local(2026, 3, 8, 8, 0).Sub(local(2026, 3, 7, 8, 0)); d != 23*time.Hour {
			t.Fatalf("the test day is %v long, not the 23 hours of spring forward", d)
		}
	})
	t.Run("fall back day", func(t *testing.T) {
		s := openMem(t)
		ana := mustParent(t, s, "ana")
		k := mustKidIn(t, s, ana.ID, "Lía", zone)
		now := time.Date(2026, 11, 1, 8, 10, 0, 0, ny)
		course := mustCourse(t, s, k.ID, ana.ID, timesCourse(time.Date(2026, 10, 31, 0, 0, 0, 0, ny), "08:00", "20:00"), now)
		mustDose(t, s, k.ID, ana.ID, doseOf(course.ID, time.Date(2026, 11, 1, 7, 45, 0, 0, ny)), now)
		got := states(t, s, course.ID)
		if got[local(2026, 11, 1, 8, 0)] != "done" || got[local(2026, 11, 1, 20, 0)] != "pending" {
			t.Errorf("rows = %v", got)
		}
	})
}

func TestDoseRowsAreAtomicWithTheDose(t *testing.T) {
	// The reminder change and the dose commit together: a cap refusal must leave the reminders as they were.
	s, k, ana, course := doseCourse(t)
	for i := range constants.MaxMedicationDosesPerKid {
		seedDose(t, s, k.ID, course.ID, ana.ID, clock(0, 0).Add(-time.Duration(i+1)*time.Second), nil)
	}
	if _, err := s.CreateDose(ctx, k.ID, ana.ID, doseOf(course.ID, clock(8, 0)), clock(10, 0)); !errors.Is(err, ErrConflict) {
		t.Fatalf("over the cap: %v", err)
	}
	if got := states(t, s, course.ID); got[clock(8, 0)] != "pending" {
		t.Errorf("a refused dose marked 08:00: %v", got)
	}
}

func TestFollowingOccurrenceComesFromTheEngineNotTheRows(t *testing.T) {
	s, k, ana, course := doseCourse(t)
	execOK(t, s, `DELETE FROM reminders WHERE medication_id = ? AND due_at = ?`, course.ID, clock(14, 0).Unix())
	mustDose(t, s, k.ID, ana.ID, doseOf(course.ID, clock(15, 0)), clock(15, 0))
	if got := states(t, s, course.ID)[clock(8, 0)]; got != "pending" {
		t.Errorf("a dose after the unmaterialised 14:00 marked 08:00 %s", got)
	}
}
