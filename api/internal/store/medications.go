package store

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/schedule"
)

var errKidZone = errors.New("store: the kid's time zone is not a real zone")

const medicationColumns = entryColumns + `, name, dose, schedule_kind, schedule_times, every_hours, start_at, end_on, active`

func scanMedication(row rowScanner) (model.Medication, error) {
	var (
		v            model.Medication
		kind         string
		times, endOn sql.NullString
		hours        sql.NullInt64
		start        int64
		active       int
	)
	e, err := scanEntry(row, &v.Name, &v.Dose, &kind, &times, &hours, &start, &endOn, &active)
	if err != nil {
		return model.Medication{}, err
	}
	v.Entry, v.Kind, v.EveryHours = e, schedule.Kind(kind), int(hours.Int64)
	if times.Valid {
		v.Times = strings.Split(times.String, ",")
	}
	v.StartAt, v.EndOn, v.Active = unixTime(start), nullableString(endOn), active != 0
	return v, nil
}

// canonicalSchedule is sc with its times sorted; false when the engine refuses it.
func canonicalSchedule(sc schedule.Schedule) (schedule.Schedule, bool) {
	if schedule.Valid(sc) != nil {
		return sc, false
	}
	if sc.Kind != schedule.KindTimes {
		return schedule.Schedule{Kind: sc.Kind, EveryHours: sc.EveryHours}, true
	}
	return schedule.Schedule{Kind: sc.Kind, Times: slices.Sorted(slices.Values(sc.Times))}, true
}

// scheduleArgs are the three schedule columns of sc; the columns a kind does not use are NULL.
func scheduleArgs(sc schedule.Schedule) []any {
	var times, hours any
	switch sc.Kind {
	case schedule.KindTimes:
		times = strings.Join(sc.Times, ",")
	case schedule.KindEvery:
		hours = sc.EveryHours
	}
	return []any{string(sc.Kind), times, hours}
}

// courseTiming is what decides a course's reminder rows.
type courseTiming struct {
	schedule.Schedule
	StartAt time.Time
	EndOn   *string
	Active  bool
}

func timingOf(v model.Medication) courseTiming {
	return courseTiming{v.Schedule, v.StartAt, v.EndOn, v.Active}
}

func (a courseTiming) course() schedule.Course {
	c := schedule.Course{Schedule: a.Schedule, StartAt: a.StartAt, Active: a.Active}
	if a.EndOn != nil {
		c.EndOn = *a.EndOn
	}
	return c
}

func (a courseTiming) same(b courseTiming) bool {
	return a.Kind == b.Kind && a.EveryHours == b.EveryHours && slices.Equal(a.Times, b.Times) && a.StartAt.Equal(b.StartAt) &&
		a.Active == b.Active && (a.EndOn == nil) == (b.EndOn == nil) && (a.EndOn == nil || *a.EndOn == *b.EndOn)
}

// loadZone refuses names LoadLocation maps to something other than a real zone ("" is UTC, "Local" the host's);
// the schedule engine panics on a nil zone, so no caller may skip this.
func loadZone(name string) (*time.Location, error) {
	if name == "" || name == "Local" {
		return nil, errKidZone
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, errKidZone
	}
	return loc, nil
}

func kidZone(ctx context.Context, q querier, kidID string) (*time.Location, error) {
	var name string
	if err := q.QueryRowContext(ctx, `SELECT time_zone FROM kids WHERE id = ?`, kidID).Scan(&name); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return loadZone(name)
}

// materialiseCourse is the only code that inserts reminder rows: one per occurrence in [now - ReminderExpiry, now],
// then the first occurrence after now whose row is not done. Existing rows stay as they are.
func materialiseCourse(ctx context.Context, tx *sql.Tx, medicationID string, loc *time.Location, now time.Time) error {
	if loc == nil {
		return errKidZone
	}
	v, err := scanMedication(tx.QueryRowContext(ctx, `SELECT `+medicationColumns+` FROM medications WHERE id = ?`, medicationID))
	if err != nil {
		return err
	}
	if !v.Active || v.Kind == schedule.KindNone {
		return nil
	}
	course := timingOf(v).course()
	insert := func(due time.Time) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO reminders(id, kid_id, medication_id, due_at, state, created_at) VALUES (?, ?, ?, ?, 'pending', ?)
			 ON CONFLICT(medication_id, due_at) DO NOTHING`, model.NewID(), v.KidID, v.ID, due.Unix(), now.Unix())
		return err
	}
	for _, due := range schedule.Occurrences(course, loc, now.Add(-constants.ReminderExpiry), now) {
		if err := insert(due); err != nil {
			return err
		}
	}
	after := now
	for range constants.MaxScheduleOccurrences {
		due, ok := schedule.Next(course, loc, after)
		if !ok {
			return nil
		}
		var state string
		err := tx.QueryRowContext(ctx, `SELECT state FROM reminders WHERE medication_id = ? AND due_at = ?`, v.ID, due.Unix()).Scan(&state)
		if err == nil && state == "done" {
			after = due
			continue
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		return insert(due)
	}
	return nil
}

// rematerialise drops the course's undelivered pending rows that are due after now, or in the lookback window and no
// longer an occurrence, then materialises it again.
func rematerialise(ctx context.Context, tx *sql.Tx, kidID, medicationID string, loc *time.Location, now time.Time) error {
	if loc == nil {
		return errKidZone
	}
	v, err := scanMedication(tx.QueryRowContext(ctx, `SELECT `+medicationColumns+` FROM medications WHERE id = ?`, medicationID))
	if err != nil {
		return err
	}
	const undelivered = ` AND state = 'pending' AND NOT EXISTS (SELECT 1 FROM reminder_deliveries d WHERE d.reminder_id = reminders.id)`
	if _, err := tx.ExecContext(ctx, `DELETE FROM reminders WHERE kid_id = ? AND medication_id = ? AND due_at > ?`+undelivered,
		kidID, medicationID, now.Unix()); err != nil {
		return err
	}
	query, args := `DELETE FROM reminders WHERE kid_id = ? AND medication_id = ? AND due_at >= ? AND due_at <= ?`+undelivered,
		[]any{kidID, medicationID, now.Add(-constants.ReminderExpiry).Unix(), now.Unix()}
	if kept := schedule.Occurrences(timingOf(v).course(), loc, now.Add(-constants.ReminderExpiry), now); len(kept) > 0 {
		query += ` AND due_at NOT IN (?` + strings.Repeat(`, ?`, len(kept)-1) + `)`
		for _, due := range kept {
			args = append(args, due.Unix())
		}
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}
	return materialiseCourse(ctx, tx, medicationID, loc, now)
}

// rematerialiseKid is rematerialise for each of the kid's courses, in the zone named zone.
func rematerialiseKid(ctx context.Context, tx *sql.Tx, kidID, zone string, now time.Time) error {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM medications WHERE kid_id = ?`, kidID)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	loc, err := loadZone(zone)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := rematerialise(ctx, tx, kidID, id, loc, now); err != nil {
			return err
		}
	}
	return nil
}

// readMedications reads the kid's courses (or just course id when it is not empty), newest start first, with each
// one's last dose, next due and dose count, all through q so a snapshot q shows one moment.
func readMedications(ctx context.Context, q querier, kidID, id string, now time.Time) ([]model.Medication, error) {
	where, args := ` WHERE m.kid_id = ?`, []any{kidID}
	if id != "" {
		where, args = where+` AND m.id = ?`, append(args, id)
	}
	rows, err := q.QueryContext(ctx, `SELECT `+prefixed("m.", medicationColumns)+` FROM medications m`+where+` ORDER BY m.start_at DESC, m.id DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Medication{}
	for rows.Next() {
		v, err := scanMedication(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil || len(out) == 0 {
		return out, err
	}
	at := make(map[string]int, len(out))
	for i, v := range out {
		at[v.ID] = i
	}
	if err := eachRow(ctx, q, `SELECT m.id, d.id, d.given_at, d.amount FROM medications m
		JOIN medication_doses d ON d.id = (SELECT x.id FROM medication_doses x WHERE x.medication_id = m.id ORDER BY x.given_at DESC, x.id DESC LIMIT 1)`+where,
		args, func(row rowScanner) error {
			var (
				medication string
				last       model.LastDose
				given      int64
				amount     sql.NullString
			)
			if err := row.Scan(&medication, &last.ID, &given, &amount); err != nil {
				return err
			}
			last.GivenAt, last.Amount = unixTime(given), nullableString(amount)
			out[at[medication]].LastDose = &last
			return nil
		}); err != nil {
		return nil, err
	}
	if err := eachRow(ctx, q, `SELECT m.id, (SELECT MIN(r.due_at) FROM reminders r WHERE r.medication_id = m.id AND r.state <> 'done' AND r.due_at > ?)
		FROM medications m`+where, append([]any{now.Unix()}, args...), func(row rowScanner) error {
		var (
			medication string
			due        sql.NullInt64
		)
		if err := row.Scan(&medication, &due); err != nil {
			return err
		}
		out[at[medication]].NextDue = nullableTime(due)
		return nil
	}); err != nil {
		return nil, err
	}
	if err := eachRow(ctx, q, `SELECT m.id, (SELECT COUNT(*) FROM medication_doses d WHERE d.medication_id = m.id) FROM medications m`+where, args,
		func(row rowScanner) error {
			var medication string
			var n int
			if err := row.Scan(&medication, &n); err != nil {
				return err
			}
			out[at[medication]].DoseCount = n
			return nil
		}); err != nil {
		return nil, err
	}
	return out, nil
}

// prefixed puts prefix before each column of a comma-separated list.
func prefixed(prefix, columns string) string {
	parts := strings.Split(columns, ", ")
	for i, c := range parts {
		parts[i] = prefix + c
	}
	return strings.Join(parts, ", ")
}

func eachRow(ctx context.Context, q querier, query string, args []any, each func(rowScanner) error) error {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := each(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

func readMedication(ctx context.Context, q querier, kidID, id string, now time.Time) (model.Medication, error) {
	if id == "" {
		return model.Medication{}, ErrNotFound
	}
	list, err := readMedications(ctx, q, kidID, id, now)
	if err != nil {
		return model.Medication{}, err
	}
	if len(list) == 0 {
		return model.Medication{}, ErrNotFound
	}
	return list[0], nil
}

// CreateMedication stores v for kidID as actorID (an active grant, else ErrNotFound) and materialises its rows;
// ErrConflict at MaxMedicationsPerKid, ErrInvalidSchedule for a schedule the engine refuses. The caller validates the rest.
func (s *Store) CreateMedication(ctx context.Context, kidID, actorID string, v model.Medication, now time.Time) (model.Medication, error) {
	sc, ok := canonicalSchedule(v.Schedule)
	if !ok {
		return model.Medication{}, ErrInvalidSchedule
	}
	v.Schedule = sc
	v.ID, v.KidID, v.CreatedBy, v.UpdatedBy = model.NewID(), kidID, actorID, actorID
	v.CreatedAt, v.UpdatedAt = unixTime(now.Unix()), unixTime(now.Unix())
	v.StartAt = unixTime(v.StartAt.Unix())
	var out model.Medication
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM medications WHERE kid_id = ?`, kidID).Scan(&n); err != nil {
			return err
		}
		if n >= constants.MaxMedicationsPerKid {
			return ErrConflict
		}
		loc, err := kidZone(ctx, tx, kidID)
		if err != nil {
			return err
		}
		args := append(append([]any{v.ID, kidID, v.Note, actorID, actorID, v.CreatedAt.Unix(), v.UpdatedAt.Unix(), v.Name, v.Dose}, scheduleArgs(sc)...),
			v.StartAt.Unix(), v.EndOn, v.Active)
		if _, err := tx.ExecContext(ctx, `INSERT INTO medications(`+medicationColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, args...); err != nil {
			return err
		}
		if err := materialiseCourse(ctx, tx, v.ID, loc, now); err != nil {
			return err
		}
		out, err = readMedication(ctx, tx, kidID, v.ID, now)
		return err
	})
	if err != nil {
		return model.Medication{}, err
	}
	return out, nil
}

// Medication is ErrNotFound for an id that is absent or belongs to another kid; next due is the first row after now.
func (s *Store) Medication(ctx context.Context, kidID, id string, now time.Time) (model.Medication, error) {
	var out model.Medication
	err := s.inReadTx(ctx, func(q querier) error {
		var err error
		out, err = readMedication(ctx, q, kidID, id, now)
		return err
	})
	return out, err
}

// ListMedications returns the kid's courses, newest start first, from one snapshot; never nil.
func (s *Store) ListMedications(ctx context.Context, kidID string, now time.Time) ([]model.Medication, error) {
	var out []model.Medication
	err := s.inReadTx(ctx, func(q querier) error {
		var err error
		out, err = readMedications(ctx, q, kidID, "", now)
		return err
	})
	return out, err
}

// UpdateMedication replaces every editable field of v.ID, which must belong to kidID; actorID needs an active grant.
// It replaces the course's future pending rows only when a field that decides them changed. The result is as of now.
func (s *Store) UpdateMedication(ctx context.Context, kidID, actorID string, v model.Medication, now time.Time) (model.Medication, error) {
	sc, ok := canonicalSchedule(v.Schedule)
	if !ok {
		return model.Medication{}, ErrInvalidSchedule
	}
	v.Schedule = sc
	v.StartAt = unixTime(v.StartAt.Unix())
	var out model.Medication
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		stored, err := scanMedication(tx.QueryRowContext(ctx, `SELECT `+medicationColumns+` FROM medications WHERE kid_id = ? AND id = ?`, kidID, v.ID))
		if err != nil {
			return err
		}
		args := append(append([]any{v.Name, v.Dose}, scheduleArgs(sc)...), v.StartAt.Unix(), v.EndOn, v.Active, v.Note, actorID, now.Unix(), kidID, v.ID)
		res, err := tx.ExecContext(ctx,
			`UPDATE medications SET name = ?, dose = ?, schedule_kind = ?, schedule_times = ?, every_hours = ?, start_at = ?, end_on = ?, active = ?,
			   note = ?, updated_by = ?, updated_at = ? WHERE kid_id = ? AND id = ?`, args...)
		if err := oneRow(res, err); err != nil {
			return err
		}
		if !timingOf(stored).same(timingOf(v)) {
			loc, err := kidZone(ctx, tx, kidID)
			if err != nil {
				return err
			}
			if err := rematerialise(ctx, tx, kidID, v.ID, loc, now); err != nil {
				return err
			}
		}
		out, err = readMedication(ctx, tx, kidID, v.ID, now)
		return err
	})
	if err != nil {
		return model.Medication{}, err
	}
	return out, nil
}

// DeleteMedication removes the course with its doses, reminders and deliveries (ON DELETE CASCADE).
func (s *Store) DeleteMedication(ctx context.Context, kidID, actorID, id string) error {
	return s.deleteEntry(ctx, "medications", kidID, actorID, id)
}
