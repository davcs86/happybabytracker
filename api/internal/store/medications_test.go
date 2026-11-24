package store

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/schedule"
)

var at10 = time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)

func mustKidIn(t *testing.T, s *Store, creator, name, zone string) model.Kid {
	t.Helper()
	in := newKid(name)
	in.TimeZone = zone
	k, err := s.CreateKid(ctx, in, creator, t0)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func timesCourse(start time.Time, times ...string) model.Medication {
	return model.Medication{Name: "Amoxicillin", Dose: "2.5 ml", Active: true, StartAt: start,
		Schedule: schedule.Schedule{Kind: schedule.KindTimes, Times: times}}
}

func everyCourse(start time.Time, hours int) model.Medication {
	return model.Medication{Name: "Ibuprofen", Dose: "5 ml", Active: true, StartAt: start,
		Schedule: schedule.Schedule{Kind: schedule.KindEvery, EveryHours: hours}}
}

func mustCourse(t *testing.T, s *Store, kidID, actorID string, v model.Medication, now time.Time) model.Medication {
	t.Helper()
	got, err := s.CreateMedication(ctx, kidID, actorID, v, now)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

type reminderRow struct {
	DueAt int64
	State string
}

func reminderRows(t *testing.T, s *Store, medicationID string) []reminderRow {
	t.Helper()
	rows, err := s.w.Query(`SELECT due_at, state FROM reminders WHERE medication_id = ? ORDER BY due_at`, medicationID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []reminderRow
	for rows.Next() {
		var r reminderRow
		if err := rows.Scan(&r.DueAt, &r.State); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func pending(at time.Time) reminderRow { return reminderRow{at.Unix(), "pending"} }

func seedDose(t *testing.T, s *Store, kidID, medicationID, by string, givenAt time.Time, amount *string) string {
	t.Helper()
	id := model.NewID()
	if _, err := s.w.Exec(`INSERT INTO medication_doses(id, kid_id, note, created_by, updated_by, created_at, updated_at, medication_id, given_at, amount)
		VALUES (?, ?, NULL, ?, ?, ?, ?, ?, ?, ?)`, id, kidID, by, by, t0.Unix(), t0.Unix(), medicationID, givenAt.Unix(), amount); err != nil {
		t.Fatal(err)
	}
	return id
}

func execOK(t *testing.T, s *Store, query string, args ...any) {
	t.Helper()
	if _, err := s.w.Exec(query, args...); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
}

func countOf(t *testing.T, s *Store, table string) int {
	t.Helper()
	var n int
	if err := s.w.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestMedicationCRUD(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)

	in := timesCourse(t0.Add(-time.Hour), "20:00", "08:00")
	in.EndOn, in.Note = strp("2026-11-30"), strp("with food")
	created, err := s.CreateMedication(ctx, k.ID, ana.ID, in, t0)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.KidID != k.ID || created.CreatedBy != ana.ID || !created.CreatedAt.Equal(t0) ||
		!reflect.DeepEqual(created.Times, []string{"08:00", "20:00"}) || created.DoseCount != 0 || created.LastDose != nil {
		t.Fatalf("created = %+v", created)
	}
	if want := time.Date(2026, 11, 20, 20, 0, 0, 0, time.UTC); created.NextDue == nil || !created.NextDue.Equal(want) {
		t.Fatalf("next due = %v, want %v", created.NextDue, want)
	}
	got, err := s.Medication(ctx, k.ID, created.ID, t0)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("Medication = %+v, %v; want %+v", got, err, created)
	}
	if list, err := s.ListMedications(ctx, k.ID, t0); err != nil || len(list) != 1 || !reflect.DeepEqual(list[0], created) {
		t.Fatalf("ListMedications = %+v, %v", list, err)
	}

	edit := got
	edit.Name, edit.EndOn, edit.Note = "Amoxicillin B", nil, nil
	edit.Schedule = schedule.Schedule{Kind: schedule.KindEvery, EveryHours: 6}
	later := t0.Add(time.Minute)
	updated, err := s.UpdateMedication(ctx, k.ID, bo.ID, edit, later)
	if err != nil {
		t.Fatal(err)
	}
	if updated.UpdatedBy != bo.ID || !updated.UpdatedAt.Equal(later) || updated.CreatedBy != ana.ID || updated.Name != "Amoxicillin B" ||
		updated.Kind != schedule.KindEvery || updated.EveryHours != 6 || updated.Times != nil || updated.EndOn != nil || updated.Note != nil {
		t.Fatalf("updated = %+v", updated)
	}
	if got, err = s.Medication(ctx, k.ID, created.ID, later); err != nil || !reflect.DeepEqual(got, updated) {
		t.Fatalf("after update = %+v, %v; want %+v", got, err, updated)
	}

	if err := s.DeleteMedication(ctx, k.ID, bo.ID, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Medication(ctx, k.ID, created.ID, later); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete: %v", err)
	}
	if err := s.DeleteMedication(ctx, k.ID, bo.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: %v", err)
	}
	if _, err := s.UpdateMedication(ctx, k.ID, bo.ID, edit, later); !errors.Is(err, ErrNotFound) {
		t.Errorf("update after delete: %v", err)
	}
}

func TestMedicationNoScheduleStoresNoColumns(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	v := mustCourse(t, s, k.ID, ana.ID, model.Medication{Name: "Vitamin D", Dose: "1 drop", Active: true, StartAt: t0,
		Schedule: schedule.Schedule{Kind: schedule.KindNone}}, t0)
	if v.NextDue != nil || len(reminderRows(t, s, v.ID)) != 0 {
		t.Errorf("an unscheduled course has next due %v and rows %v", v.NextDue, reminderRows(t, s, v.ID))
	}
	var times sql.NullString
	var hours sql.NullInt64
	if err := s.w.QueryRow(`SELECT schedule_times, every_hours FROM medications WHERE id = ?`, v.ID).Scan(&times, &hours); err != nil || times.Valid || hours.Valid {
		t.Errorf("columns = %v %v, %v; want NULLs", times, hours, err)
	}
}

func TestMedicationRefusesAnInvalidSchedule(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	for name, sc := range map[string]schedule.Schedule{
		"no kind":       {},
		"times empty":   {Kind: schedule.KindTimes},
		"times 7:05":    {Kind: schedule.KindTimes, Times: []string{"7:05"}},
		"every 0":       {Kind: schedule.KindEvery},
		"every 169":     {Kind: schedule.KindEvery, EveryHours: constants.MaxEveryHours + 1},
		"none and hour": {Kind: schedule.KindNone, EveryHours: 1},
	} {
		v := timesCourse(t0)
		v.Schedule = sc
		if _, err := s.CreateMedication(ctx, k.ID, ana.ID, v, t0); !errors.Is(err, ErrInvalidSchedule) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if list, err := s.ListMedications(ctx, k.ID, t0); err != nil || len(list) != 0 {
		t.Errorf("list = %+v, %v; want none", list, err)
	}
}

func TestMedicationBelongsToItsKid(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	mine, other := mustKidIn(t, s, ana.ID, "Mine", "UTC"), mustKidIn(t, s, ana.ID, "Other", "UTC")
	v := mustCourse(t, s, mine.ID, ana.ID, timesCourse(t0, "08:00"), t0)

	if _, err := s.Medication(ctx, other.ID, v.ID, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("read through another kid: %v", err)
	}
	edit := v
	edit.Name = "Changed"
	if _, err := s.UpdateMedication(ctx, other.ID, ana.ID, edit, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("update through another kid: %v", err)
	}
	if err := s.DeleteMedication(ctx, other.ID, ana.ID, v.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete through another kid: %v", err)
	}
	if list, err := s.ListMedications(ctx, other.ID, t0); err != nil || len(list) != 0 {
		t.Errorf("other kid's list = %+v, %v", list, err)
	}
	if got, err := s.Medication(ctx, mine.ID, v.ID, t0); err != nil || !reflect.DeepEqual(got, v) {
		t.Errorf("course changed: %+v, %v", got, err)
	}
}

func TestMedicationWriteByDisabledActor(t *testing.T) {
	s := openMem(t)
	ana, bo, cy := mustParent(t, s, "ana"), mustParent(t, s, "bo"), mustParent(t, s, "cy")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)
	v := mustCourse(t, s, k.ID, ana.ID, timesCourse(t0, "20:00"), t0)
	if err := s.SetParentDisabled(ctx, bo.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	for name, actor := range map[string]string{"disabled caregiver": bo.ID, "no grant": cy.ID} {
		if _, err := s.CreateMedication(ctx, k.ID, actor, timesCourse(t0, "20:00"), t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s create: %v", name, err)
		}
		if _, err := s.UpdateMedication(ctx, k.ID, actor, v, t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s update: %v", name, err)
		}
		if err := s.DeleteMedication(ctx, k.ID, actor, v.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s delete: %v", name, err)
		}
	}
	if list, err := s.ListMedications(ctx, k.ID, t0); err != nil || len(list) != 1 || !reflect.DeepEqual(list[0], v) {
		t.Errorf("list = %+v, %v; want the one seeded course unchanged", list, err)
	}
}

func TestMedicationCap(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k, other := mustKidIn(t, s, ana.ID, "Lía", "UTC"), mustKidIn(t, s, ana.ID, "Otro", "UTC")
	for i := range constants.MaxMedicationsPerKid {
		v := timesCourse(t0, "08:00")
		v.Name = fmt.Sprintf("M%d", i)
		mustCourse(t, s, k.ID, ana.ID, v, t0)
	}
	if _, err := s.CreateMedication(ctx, k.ID, ana.ID, timesCourse(t0, "08:00"), t0); !errors.Is(err, ErrConflict) {
		t.Errorf("course %d: %v, want ErrConflict", constants.MaxMedicationsPerKid+1, err)
	}
	if _, err := s.CreateMedication(ctx, other.ID, ana.ID, timesCourse(t0, "08:00"), t0); err != nil {
		t.Errorf("another kid's first course: %v", err)
	}
}

func TestCreateMaterialisesNext(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")

	v := mustCourse(t, s, k.ID, ana.ID, timesCourse(at10, "08:00", "20:00"), at10)
	evening := time.Date(2026, 11, 20, 20, 0, 0, 0, time.UTC)
	if v.NextDue == nil || !v.NextDue.Equal(evening) {
		t.Fatalf("next due = %v, want %v", v.NextDue, evening)
	}
	if got := reminderRows(t, s, v.ID); !reflect.DeepEqual(got, []reminderRow{pending(evening)}) {
		t.Errorf("rows = %v, want the 20:00 one only (08:00 is before the start)", got)
	}

	start := at10.Add(-20 * time.Minute)
	e := mustCourse(t, s, k.ID, ana.ID, everyCourse(start, 6), at10)
	want := []reminderRow{pending(start), pending(start.Add(6 * time.Hour))}
	if got := reminderRows(t, s, e.ID); !reflect.DeepEqual(got, want) {
		t.Errorf("every rows = %v, want %v", got, want)
	}
	if e.NextDue == nil || !e.NextDue.Equal(start.Add(6*time.Hour)) {
		t.Errorf("every next due = %v, want %v (a due-but-pending row is not next)", e.NextDue, start.Add(6*time.Hour))
	}
}

func TestCreateLateOccurrenceWindow(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	start := time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)
	for name, c := range map[string]struct {
		now  time.Time
		want int
	}{
		"29 minutes late": {time.Date(2026, 11, 20, 8, 29, 0, 0, time.UTC), 2},
		"30 minutes late": {time.Date(2026, 11, 20, 8, 30, 0, 0, time.UTC), 2},
		"31 minutes late": {time.Date(2026, 11, 20, 8, 31, 0, 0, time.UTC), 1},
	} {
		v := mustCourse(t, s, k.ID, ana.ID, timesCourse(start, "08:00"), c.now)
		if got := len(reminderRows(t, s, v.ID)); got != c.want {
			t.Errorf("%s: %d rows %v, want %d", name, got, reminderRows(t, s, v.ID), c.want)
		}
	}
}

func TestMaterialiseCourseIsIdempotentAndSkipsDone(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	v := mustCourse(t, s, k.ID, ana.ID, everyCourse(at10, 4), at10)
	run := func(now time.Time) {
		t.Helper()
		err := s.inTx(ctx, func(tx *sql.Tx) error { return materialiseCourse(ctx, tx, v.ID, time.UTC, now) })
		if err != nil {
			t.Fatal(err)
		}
	}
	run(at10)
	run(at10)
	if got := reminderRows(t, s, v.ID); len(got) != 2 || got[0] != pending(at10) || got[1] != pending(at10.Add(4*time.Hour)) {
		t.Fatalf("rows after two runs = %v", got)
	}

	execOK(t, s, `UPDATE reminders SET state = 'done' WHERE medication_id = ? AND due_at = ?`, v.ID, at10.Add(4*time.Hour).Unix())
	run(at10.Add(3*time.Hour + 45*time.Minute))
	want := []reminderRow{pending(at10), {at10.Add(4 * time.Hour).Unix(), "done"}, pending(at10.Add(8 * time.Hour))}
	if got := reminderRows(t, s, v.ID); !reflect.DeepEqual(got, want) {
		t.Errorf("rows with a done next = %v, want %v", got, want)
	}
}

func TestMaterialiseCourseRefusesANilZone(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	v := mustCourse(t, s, k.ID, ana.ID, timesCourse(at10, "20:00"), at10)
	err := s.inTx(ctx, func(tx *sql.Tx) error { return materialiseCourse(ctx, tx, v.ID, nil, at10) })
	if err == nil {
		t.Error("a nil zone was accepted")
	}
}

func TestEditReplacesPendingRows(t *testing.T) {
	today := func(h, m int) time.Time { return time.Date(2026, 11, 20, h, m, 0, 0, time.UTC) }
	for name, c := range map[string]struct {
		edit func(*model.Medication)
		want []reminderRow
	}{
		"times": {func(v *model.Medication) { v.Times = []string{"09:00", "21:00"} }, []reminderRow{pending(today(21, 0))}},
		"start_at": {func(v *model.Medication) { v.StartAt = today(20, 30) },
			[]reminderRow{pending(time.Date(2026, 11, 21, 8, 0, 0, 0, time.UTC))}},
		"end_on":   {func(v *model.Medication) { v.EndOn = strp("2026-11-19") }, nil},
		"inactive": {func(v *model.Medication) { v.Active = false }, nil},
		"kind":     {func(v *model.Medication) { v.Schedule = schedule.Schedule{Kind: schedule.KindNone} }, nil},
	} {
		t.Run(name, func(t *testing.T) {
			s := openMem(t)
			ana := mustParent(t, s, "ana")
			k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
			v := mustCourse(t, s, k.ID, ana.ID, timesCourse(at10, "08:00", "20:00"), at10)
			if len(reminderRows(t, s, v.ID)) != 1 {
				t.Fatalf("seed rows = %v", reminderRows(t, s, v.ID))
			}
			edit := v
			c.edit(&edit)
			got, err := s.UpdateMedication(ctx, k.ID, ana.ID, edit, at10)
			if err != nil {
				t.Fatal(err)
			}
			if rows := reminderRows(t, s, v.ID); !reflect.DeepEqual(rows, c.want) {
				t.Errorf("rows = %v, want %v", rows, c.want)
			}
			if (len(c.want) == 0) != (got.NextDue == nil) {
				t.Errorf("next due = %v with rows %v", got.NextDue, c.want)
			}
		})
	}
}

func TestEditKeepsDoneAndDeliveredRows(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	v := mustCourse(t, s, k.ID, ana.ID, everyCourse(at10, 2), at10)
	// Seeded rows: 10:00 and 12:00 pending; 12:00 gets a delivery, 14:00 is added done and 16:00 pending.
	two, four, six := at10.Add(2*time.Hour), at10.Add(4*time.Hour), at10.Add(6*time.Hour)
	var rid string
	if err := s.w.QueryRow(`SELECT id FROM reminders WHERE medication_id = ? AND due_at = ?`, v.ID, two.Unix()).Scan(&rid); err != nil {
		t.Fatal(err)
	}
	execOK(t, s, `INSERT INTO reminder_deliveries(id, reminder_id, channel, target_id, parent_id, state, attempts, next_attempt_at) VALUES (?, ?, 'push', 'dev', ?, 'sent', 1, 0)`,
		model.NewID(), rid, ana.ID)
	for due, state := range map[int64]string{four.Unix(): "done", six.Unix(): "pending"} {
		execOK(t, s, `INSERT INTO reminders(id, kid_id, medication_id, due_at, state, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
			model.NewID(), k.ID, v.ID, due, state, at10.Unix())
	}

	edit := v
	edit.Schedule = schedule.Schedule{Kind: schedule.KindEvery, EveryHours: 5} // grid 10:00, 15:00, ...
	if _, err := s.UpdateMedication(ctx, k.ID, ana.ID, edit, at10); err != nil {
		t.Fatal(err)
	}
	want := []reminderRow{pending(at10), pending(two), {four.Unix(), "done"}, pending(at10.Add(5 * time.Hour))}
	if got := reminderRows(t, s, v.ID); !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %v, want %v (the due one, the delivered and the done stay; the undelivered pending 16:00 went)", got, want)
	}
}

func TestNoteOnlyEditRewritesNothing(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	v := mustCourse(t, s, k.ID, ana.ID, timesCourse(at10, "20:00", "08:00"), at10)
	rowOf := func() string {
		var out string
		err := s.w.QueryRow(`SELECT name || '|' || dose || '|' || schedule_kind || '|' || IFNULL(schedule_times, '-') || '|' || IFNULL(every_hours, '-') ||
			'|' || start_at || '|' || IFNULL(end_on, '-') || '|' || active FROM medications WHERE id = ?`, v.ID).Scan(&out)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	idsOf := func() string {
		var out string
		if err := s.w.QueryRow(`SELECT group_concat(id) FROM reminders WHERE medication_id = ?`, v.ID).Scan(&out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	beforeRow, beforeIDs := rowOf(), idsOf()
	edit := v
	edit.Note = strp("only the note")
	got, err := s.UpdateMedication(ctx, k.ID, ana.ID, edit, at10.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if rowOf() != beforeRow || idsOf() != beforeIDs {
		t.Errorf("a note-only edit changed the course %q -> %q or its rows %q -> %q", beforeRow, rowOf(), beforeIDs, idsOf())
	}
	if got.Note == nil || *got.Note != "only the note" {
		t.Errorf("note = %v", got.Note)
	}
}

func TestInactiveHasNoNextDue(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	in := timesCourse(at10, "20:00")
	in.Active = false
	v := mustCourse(t, s, k.ID, ana.ID, in, at10)
	if v.NextDue != nil || len(reminderRows(t, s, v.ID)) != 0 {
		t.Errorf("an inactive course has next due %v and rows %v", v.NextDue, reminderRows(t, s, v.ID))
	}
	edit := v
	edit.Active = true
	on, err := s.UpdateMedication(ctx, k.ID, ana.ID, edit, at10)
	if err != nil || on.NextDue == nil || len(reminderRows(t, s, v.ID)) != 1 {
		t.Fatalf("activated = %+v, %v, rows %v", on, err, reminderRows(t, s, v.ID))
	}
	edit.Active = false
	off, err := s.UpdateMedication(ctx, k.ID, ana.ID, edit, at10)
	if err != nil || off.NextDue != nil || len(reminderRows(t, s, v.ID)) != 0 {
		t.Errorf("deactivated = %+v, %v, rows %v", off, err, reminderRows(t, s, v.ID))
	}
}

func TestKidZoneChangeRematerialises(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "Europe/London")
	v := mustCourse(t, s, k.ID, ana.ID, timesCourse(t0.Add(-10*24*time.Hour), "08:00"), t0)
	if want := time.Date(2026, 11, 21, 8, 0, 0, 0, time.UTC); v.NextDue == nil || !v.NextDue.Equal(want) {
		t.Fatalf("London next due = %v, want %v", v.NextDue, want)
	}

	k.TimeZone = "America/New_York"
	if err := s.UpdateKid(ctx, k, ana.ID, t0); err != nil {
		t.Fatal(err)
	}
	got, err := s.Medication(ctx, k.ID, v.ID, t0)
	if want := time.Date(2026, 11, 20, 13, 0, 0, 0, time.UTC); err != nil || got.NextDue == nil || !got.NextDue.Equal(want) {
		t.Fatalf("New York next due = %v, %v; want %v (08:00 EST)", got.NextDue, err, want)
	}
	if rows := reminderRows(t, s, v.ID); len(rows) != 1 {
		t.Errorf("rows = %v, want the one New York row", rows)
	}

	idOf := func() string {
		var id string
		if err := s.w.QueryRow(`SELECT id FROM reminders WHERE medication_id = ?`, v.ID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	before := idOf()
	k.Name = "Lía B"
	if err := s.UpdateKid(ctx, k, ana.ID, t0); err != nil {
		t.Fatal(err)
	}
	if after := idOf(); after != before {
		t.Errorf("a profile edit without a zone change rewrote the rows: %q -> %q", before, after)
	}
}

func TestKidZoneChangeWithoutCoursesNeedsNoZone(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	k.TimeZone = "Not/AZone"
	if err := s.UpdateKid(ctx, k, ana.ID, t0); err != nil {
		t.Errorf("a kid with no course: %v", err)
	}
}

func TestDeleteCascadesDoses(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	v := mustCourse(t, s, k.ID, ana.ID, everyCourse(at10, 2), at10)
	keep := mustCourse(t, s, k.ID, ana.ID, timesCourse(at10, "20:00"), at10)
	for i := range 3 {
		seedDose(t, s, k.ID, v.ID, ana.ID, at10.Add(-time.Duration(i)*time.Hour), nil)
	}
	seedDose(t, s, k.ID, keep.ID, ana.ID, at10, nil)
	var rid string
	if err := s.w.QueryRow(`SELECT id FROM reminders WHERE medication_id = ? LIMIT 1`, v.ID).Scan(&rid); err != nil {
		t.Fatal(err)
	}
	execOK(t, s, `INSERT INTO reminder_deliveries(id, reminder_id, channel, target_id, parent_id, state, attempts, next_attempt_at) VALUES (?, ?, 'push', 'dev', ?, 'sent', 1, 0)`,
		model.NewID(), rid, ana.ID)

	if got, err := s.Medication(ctx, k.ID, v.ID, at10); err != nil || got.DoseCount != 3 {
		t.Fatalf("dose count = %d, %v; want 3", got.DoseCount, err)
	}
	if err := s.DeleteMedication(ctx, k.ID, ana.ID, v.ID); err != nil {
		t.Fatal(err)
	}
	for table, want := range map[string]int{"medication_doses": 1, "reminders": 1, "reminder_deliveries": 0, "medications": 1} {
		if got := countOf(t, s, table); got != want {
			t.Errorf("%s holds %d rows, want %d", table, got, want)
		}
	}
	if got, err := s.Medication(ctx, k.ID, keep.ID, at10); err != nil || got.DoseCount != 1 {
		t.Errorf("the other course = %+v, %v", got, err)
	}
}

func TestMedicationLastDoseAndCount(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	a := mustCourse(t, s, k.ID, ana.ID, timesCourse(at10, "20:00"), at10)
	b := mustCourse(t, s, k.ID, ana.ID, timesCourse(at10, "21:00"), at10)
	seedDose(t, s, k.ID, a.ID, ana.ID, at10.Add(-3*time.Hour), strp("1 ml"))
	newest := seedDose(t, s, k.ID, a.ID, ana.ID, at10.Add(-time.Hour), strp("2 ml"))
	seedDose(t, s, k.ID, a.ID, ana.ID, at10.Add(-2*time.Hour), nil)

	got, err := s.Medication(ctx, k.ID, a.ID, at10)
	want := &model.LastDose{ID: newest, GivenAt: at10.Add(-time.Hour), Amount: strp("2 ml")}
	if err != nil || got.DoseCount != 3 || !reflect.DeepEqual(got.LastDose, want) {
		t.Errorf("course a = %+v (last %+v), %v; want 3 doses, last %+v", got, got.LastDose, err, want)
	}
	list, err := s.ListMedications(ctx, k.ID, at10)
	if err != nil || len(list) != 2 {
		t.Fatalf("list = %+v, %v", list, err)
	}
	for _, m := range list {
		switch m.ID {
		case a.ID:
			if m.DoseCount != 3 || !reflect.DeepEqual(m.LastDose, want) {
				t.Errorf("listed a = %+v", m)
			}
		case b.ID:
			if m.DoseCount != 0 || m.LastDose != nil {
				t.Errorf("listed b = %+v", m)
			}
		}
	}
}

func TestListMedicationsOrderAndQueries(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	var ids []string
	for i := range constants.MaxMedicationsPerKid {
		v := timesCourse(at10.Add(time.Duration(i)*time.Minute), "20:00")
		v.Name = fmt.Sprintf("M%d", i)
		ids = append(ids, mustCourse(t, s, k.ID, ana.ID, v, at10).ID)
	}
	n := countReads(t, s, nil)
	list, err := s.ListMedications(ctx, k.ID, at10)
	if err != nil || len(list) != constants.MaxMedicationsPerKid {
		t.Fatalf("list = %d rows, %v", len(list), err)
	}
	if got := n.Load(); got != 4 {
		t.Errorf("a list of %d courses ran %d queries, want 4 (courses, last doses, next dues, dose counts)", len(list), got)
	}
	if list[0].ID != ids[len(ids)-1] || list[len(list)-1].ID != ids[0] {
		t.Errorf("order: first %s last %s; want newest start first", list[0].Name, list[len(list)-1].Name)
	}
	empty := mustKidIn(t, s, ana.ID, "Vacío", "UTC")
	if none, err := s.ListMedications(ctx, empty.ID, at10); err != nil || none == nil || len(none) != 0 {
		t.Errorf("empty list = %v, %v; want a non-nil empty slice", none, err)
	}
}

// A write that lands between the course read and its children must not show as a mixed course.
func TestCourseReadIsOneSnapshot(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	reads := map[string]func(id string) (model.Medication, error){
		"Medication": func(id string) (model.Medication, error) { return s.Medication(ctx, k.ID, id, at10) },
		"ListMedications": func(string) (model.Medication, error) {
			list, err := s.ListMedications(ctx, k.ID, at10)
			if err != nil || len(list) != 1 {
				t.Fatalf("ListMedications = %v, %v", list, err)
			}
			return list[0], nil
		},
	}
	for name, read := range reads {
		for step, write := range map[string]func(v model.Medication){
			"a dose": func(v model.Medication) { seedDose(t, s, k.ID, v.ID, ana.ID, at10.Add(-time.Minute), strp("late")) },
			"switched off": func(v model.Medication) {
				v.Active = false
				if _, err := s.UpdateMedication(ctx, k.ID, ana.ID, v, at10); err != nil {
					t.Error(err)
				}
			},
		} {
			v := mustCourse(t, s, k.ID, ana.ID, timesCourse(at10, "20:00"), at10)
			seedDose(t, s, k.ID, v.ID, ana.ID, at10.Add(-time.Hour), strp("first"))
			var wrote atomic.Bool
			countReads(t, s, func(n int64) {
				if n == 2 && wrote.CompareAndSwap(false, true) {
					write(v)
				}
			})
			got, err := read(v.ID)
			if err != nil || !wrote.Load() {
				t.Fatalf("%s/%s: %v, wrote between the reads: %v", name, step, err, wrote.Load())
			}
			if got.Active != (got.NextDue != nil) {
				t.Errorf("%s/%s: active %v with next due %v, a mixed read", name, step, got.Active, got.NextDue)
			}
			if got.DoseCount != 1 || got.LastDose == nil || got.LastDose.Amount == nil || *got.LastDose.Amount != "first" {
				t.Errorf("%s/%s: %d doses, last %+v, a mixed read", name, step, got.DoseCount, got.LastDose)
			}
			if err := s.DeleteMedication(ctx, k.ID, ana.ID, v.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestEditDropsAnUndeliveredOldRowInTheLookback(t *testing.T) {
	at0810 := time.Date(2026, 11, 20, 8, 10, 0, 0, time.UTC)
	start := time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)
	eight := time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)
	hm := func(h, m int) time.Time { return time.Date(2026, 11, 20, h, m, 0, 0, time.UTC) }
	for name, c := range map[string]struct {
		prepare func(t *testing.T, s *Store, medicationID, parentID string)
		times   string
		want    []reminderRow
	}{
		"undelivered": {nil, "09:00", []reminderRow{pending(hm(9, 0))}},
		"delivered": {func(t *testing.T, s *Store, medicationID, parentID string) {
			var rid string
			if err := s.w.QueryRow(`SELECT id FROM reminders WHERE medication_id = ? AND due_at = ?`, medicationID, eight.Unix()).Scan(&rid); err != nil {
				t.Fatal(err)
			}
			execOK(t, s, `INSERT INTO reminder_deliveries(id, reminder_id, channel, target_id, parent_id, state, attempts, next_attempt_at) VALUES (?, ?, 'push', 'dev', ?, 'sent', 1, 0)`,
				model.NewID(), rid, parentID)
		}, "09:00", []reminderRow{pending(eight), pending(hm(9, 0))}},
		"done": {func(t *testing.T, s *Store, medicationID, _ string) {
			execOK(t, s, `UPDATE reminders SET state = 'done' WHERE medication_id = ? AND due_at = ?`, medicationID, eight.Unix())
		}, "09:00", []reminderRow{{eight.Unix(), "done"}, pending(hm(9, 0))}},
		"the new timing's own late row is created": {nil, "08:05", []reminderRow{pending(hm(8, 5)), pending(hm(8, 5).Add(24 * time.Hour))}},
		"an occurrence of the new timing stays":    {nil, "08:00", []reminderRow{pending(eight), pending(eight.Add(24 * time.Hour))}},
	} {
		t.Run(name, func(t *testing.T) {
			s := openMem(t)
			ana := mustParent(t, s, "ana")
			k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
			v := mustCourse(t, s, k.ID, ana.ID, timesCourse(start, "08:00"), at0810)
			if len(reminderRows(t, s, v.ID)) != 2 {
				t.Fatalf("seed rows = %v", reminderRows(t, s, v.ID))
			}
			if c.prepare != nil {
				c.prepare(t, s, v.ID, ana.ID)
			}
			ids := func() string {
				var out sql.NullString
				if err := s.w.QueryRow(`SELECT group_concat(id) FROM reminders WHERE medication_id = ? AND due_at <= ?`, v.ID, at0810.Unix()).Scan(&out); err != nil {
					t.Fatal(err)
				}
				return out.String
			}
			before := ids()
			edit := v
			edit.Times = []string{c.times}
			if _, err := s.UpdateMedication(ctx, k.ID, ana.ID, edit, at0810); err != nil {
				t.Fatal(err)
			}
			if got := reminderRows(t, s, v.ID); !reflect.DeepEqual(got, c.want) {
				t.Errorf("rows = %v, want %v", got, c.want)
			}
			if name == "an occurrence of the new timing stays" || name == "delivered" || name == "done" {
				if after := ids(); after != before {
					t.Errorf("the kept row was rewritten: %q -> %q", before, after)
				}
			}
		})
	}
}

func TestZoneChangeDropsAnUndeliveredOldRowInTheLookback(t *testing.T) {
	at0810 := time.Date(2026, 11, 20, 8, 10, 0, 0, time.UTC)
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	v := mustCourse(t, s, k.ID, ana.ID, timesCourse(time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC), "08:00"), at0810)
	e := mustCourse(t, s, k.ID, ana.ID, everyCourse(time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC), 4), at0810)
	idOf := func(medicationID string, due time.Time) string {
		var id string
		if err := s.w.QueryRow(`SELECT id FROM reminders WHERE medication_id = ? AND due_at = ?`, medicationID, due.Unix()).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	everyRow := idOf(e.ID, time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC))

	k.TimeZone = "America/New_York"
	if err := s.UpdateKid(ctx, k, ana.ID, at0810); err != nil {
		t.Fatal(err)
	}
	if want := []reminderRow{pending(time.Date(2026, 11, 20, 13, 0, 0, 0, time.UTC))}; !reflect.DeepEqual(reminderRows(t, s, v.ID), want) {
		t.Errorf("times rows = %v, want %v (the 08:00Z one was never sent and is no longer an occurrence)", reminderRows(t, s, v.ID), want)
	}
	if got := idOf(e.ID, time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)); got != everyRow {
		t.Errorf("an every course's due row was rewritten by a zone change: %q -> %q", everyRow, got)
	}
}

func TestNoopEditRewritesNoRowInTheLookback(t *testing.T) {
	at0810 := time.Date(2026, 11, 20, 8, 10, 0, 0, time.UTC)
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "UTC")
	v := mustCourse(t, s, k.ID, ana.ID, timesCourse(time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC), "08:00"), at0810)
	rows := func() string {
		var out string
		if err := s.w.QueryRow(`SELECT group_concat(id || ':' || due_at || ':' || state) FROM reminders WHERE medication_id = ?`, v.ID).Scan(&out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	before := rows()
	edit := v
	edit.Name, edit.Note = "Renamed", strp("n")
	if _, err := s.UpdateMedication(ctx, k.ID, ana.ID, edit, at0810); err != nil {
		t.Fatal(err)
	}
	if after := rows(); after != before {
		t.Errorf("a no-op edit rewrote rows: %q -> %q", before, after)
	}
}
