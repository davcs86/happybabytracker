package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

const visitColumns = entryColumns + `, starts_at, duration_min, kind, provider, location, outcome_notes, cancelled_at, revision`

// linkedCounts are the three per-visit counts of linked rows, as columns over visits.
const linkedCounts = `(SELECT COUNT(*) FROM growth_measurements g WHERE g.visit_id = visits.id),
	(SELECT COUNT(DISTINCT d.administration) FROM vaccine_doses d WHERE d.visit_id = visits.id),
	(SELECT COUNT(*) FROM medications m WHERE m.visit_id = visits.id)`

// LinkedCounts is how many rows of each module a visit recorded; a vaccination is one administration.
type LinkedCounts struct{ Growth, Vaccinations, Medications int }

// VisitRow is a visit with its linked counts, as the list shows it.
type VisitRow struct {
	model.Visit
	Linked LinkedCounts
}

type LinkedGrowth struct{ ID, MeasuredOn string }

type LinkedVaccination struct {
	ID, GivenOn string
	Antigens    []string
}

type LinkedMedication struct {
	ID, Name  string
	DoseCount int
}

// VisitLinks are the rows a visit recorded, oldest first.
type VisitLinks struct {
	Growth       []LinkedGrowth
	Vaccinations []LinkedVaccination
	Medications  []LinkedMedication
}

// VisitDetail is one visit with its linked rows, read from one snapshot.
type VisitDetail struct {
	model.Visit
	Links VisitLinks
}

func scanVisit(row rowScanner, extra ...any) (model.Visit, error) {
	var (
		v                           model.Visit
		start, cancelled            sql.NullInt64
		kind                        string
		provider, location, outcome sql.NullString
	)
	e, err := scanEntry(row, append([]any{&start, &v.DurationMin, &kind, &provider, &location, &outcome, &cancelled, &v.Revision}, extra...)...)
	if err != nil {
		return model.Visit{}, err
	}
	v.Entry, v.StartsAt, v.Kind = e, unixTime(start.Int64), model.VisitKind(kind)
	v.Provider, v.Location, v.OutcomeNotes = nullableString(provider), nullableString(location), nullableString(outcome)
	v.CancelledAt = nullableTime(cancelled)
	return v, nil
}

func readVisit(ctx context.Context, q querier, kidID, id string) (model.Visit, error) {
	return scanVisit(q.QueryRowContext(ctx, `SELECT `+visitColumns+` FROM visits WHERE kid_id = ? AND id = ?`, kidID, id))
}

// hasLinks is whether any module row is linked to the visit.
func hasLinks(ctx context.Context, q querier, visitID string) (bool, error) {
	var n int
	err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM growth_measurements WHERE visit_id = ?)
		OR EXISTS (SELECT 1 FROM vaccine_doses WHERE visit_id = ?) OR EXISTS (SELECT 1 FROM medications WHERE visit_id = ?)`,
		visitID, visitID, visitID).Scan(&n)
	return n == 1, err
}

func isPast(v model.Visit, now time.Time) bool { return v.CancelledAt == nil && !v.StartsAt.After(now) }

// CreateVisit stores v for kidID as actorID (an active grant, else ErrNotFound) at revision 0; ErrConflict at
// MaxVisitsPerKid. The caller validates the rest.
func (s *Store) CreateVisit(ctx context.Context, kidID, actorID string, v model.Visit, now time.Time) (model.Visit, error) {
	v.ID, v.KidID, v.CreatedBy, v.UpdatedBy, v.Revision = model.NewID(), kidID, actorID, actorID, 0
	v.CreatedAt, v.UpdatedAt, v.StartsAt = unixTime(now.Unix()), unixTime(now.Unix()), unixTime(v.StartsAt.Unix())
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM visits WHERE kid_id = ?`, kidID).Scan(&n); err != nil {
			return err
		}
		if n >= constants.MaxVisitsPerKid {
			return ErrConflict
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO visits(`+visitColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			v.ID, kidID, v.Note, actorID, actorID, v.CreatedAt.Unix(), v.UpdatedAt.Unix(), v.StartsAt.Unix(), v.DurationMin,
			string(v.Kind), v.Provider, v.Location, v.OutcomeNotes, nullableUnix(v.CancelledAt), 0)
		return err
	})
	if err != nil {
		return model.Visit{}, err
	}
	return v, nil
}

// Visit is the visit with its linked rows; ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) Visit(ctx context.Context, kidID, id string) (VisitDetail, error) {
	var out VisitDetail
	err := s.inReadTx(ctx, func(q querier) error {
		var err error
		out, err = readVisitDetail(ctx, q, kidID, id)
		return err
	})
	return out, err
}

func readVisitDetail(ctx context.Context, q querier, kidID, id string) (VisitDetail, error) {
	v, err := readVisit(ctx, q, kidID, id)
	if err != nil {
		return VisitDetail{}, err
	}
	links := VisitLinks{Growth: []LinkedGrowth{}, Vaccinations: []LinkedVaccination{}, Medications: []LinkedMedication{}}
	if err := eachRow(ctx, q, `SELECT id, measured_on FROM growth_measurements WHERE kid_id = ? AND visit_id = ? ORDER BY measured_on, id`,
		[]any{kidID, id}, func(row rowScanner) error {
			var g LinkedGrowth
			if err := row.Scan(&g.ID, &g.MeasuredOn); err != nil {
				return err
			}
			links.Growth = append(links.Growth, g)
			return nil
		}); err != nil {
		return VisitDetail{}, err
	}
	doses, err := queryDoses(ctx, q, `WHERE kid_id = ? AND visit_id = ? ORDER BY given_on, administration, antigen`, kidID, id)
	if err != nil {
		return VisitDetail{}, err
	}
	for _, a := range groupDoses(doses) {
		links.Vaccinations = append(links.Vaccinations, LinkedVaccination{ID: a.ID, GivenOn: a.GivenOn, Antigens: a.Antigens})
	}
	if err := eachRow(ctx, q, `SELECT m.id, m.name, (SELECT COUNT(*) FROM medication_doses d WHERE d.medication_id = m.id)
		FROM medications m WHERE m.kid_id = ? AND m.visit_id = ? ORDER BY m.start_at, m.id`, []any{kidID, id}, func(row rowScanner) error {
		var m LinkedMedication
		if err := row.Scan(&m.ID, &m.Name, &m.DoseCount); err != nil {
			return err
		}
		links.Medications = append(links.Medications, m)
		return nil
	}); err != nil {
		return VisitDetail{}, err
	}
	return VisitDetail{Visit: v, Links: links}, nil
}

// ascPageTail is pageTail soonest first: the cursor's next page is strictly later.
func ascPageTail(timeCol, kidID string, q ListQuery) (string, []any) {
	tail := ` WHERE kid_id = ? AND ` + timeCol + ` >= ? AND ` + timeCol + ` < ?`
	args := []any{kidID, q.From.Unix(), q.To.Unix()}
	if q.After != nil {
		tail += ` AND (` + timeCol + ` > ? OR (` + timeCol + ` = ? AND id > ?))`
		args = append(args, q.After.At, q.After.At, q.After.ID)
	}
	return tail + ` ORDER BY ` + timeCol + `, id LIMIT ?`, append(args, q.Limit)
}

// ListVisits pages the kid's visits by starts_at in [From, To), soonest first when asc, else newest first; never nil.
func (s *Store) ListVisits(ctx context.Context, kidID string, q ListQuery, asc bool) ([]VisitRow, error) {
	tail, args := pageTail("starts_at", kidID, q)
	if asc {
		tail, args = ascPageTail("starts_at", kidID, q)
	}
	out := []VisitRow{}
	err := eachRow(ctx, s.r, `SELECT `+visitColumns+`, `+linkedCounts+` FROM visits`+tail, args, func(row rowScanner) error {
		var r VisitRow
		v, err := scanVisit(row, &r.Linked.Growth, &r.Linked.Vaccinations, &r.Linked.Medications)
		if err != nil {
			return err
		}
		r.Visit = v
		out = append(out, r)
		return nil
	})
	return out, err
}

// UpdateVisit replaces every editable field of v.ID, which must belong to kidID, and adds 1 to its revision; actorID
// needs an active grant. ErrHasOutcome cancels or moves into the future a visit with an outcome; ErrNotPast sets
// notes on a visit that is not past.
func (s *Store) UpdateVisit(ctx context.Context, kidID, actorID string, v model.Visit, now time.Time) (VisitDetail, error) {
	var out VisitDetail
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		stored, err := readVisit(ctx, tx, kidID, v.ID)
		if err != nil {
			return err
		}
		linked, err := hasLinks(ctx, tx, v.ID)
		if err != nil {
			return err
		}
		if (stored.OutcomeNotes != nil || linked) && (v.CancelledAt != nil || v.StartsAt.After(now)) {
			return ErrHasOutcome
		}
		if v.OutcomeNotes != nil && !isPast(v, now) {
			return ErrNotPast
		}
		res, err := tx.ExecContext(ctx,
			`UPDATE visits SET starts_at = ?, duration_min = ?, kind = ?, provider = ?, location = ?, note = ?, outcome_notes = ?,
			   cancelled_at = ?, revision = revision + 1, updated_by = ?, updated_at = ? WHERE kid_id = ? AND id = ?`,
			v.StartsAt.Unix(), v.DurationMin, string(v.Kind), v.Provider, v.Location, v.Note, v.OutcomeNotes,
			nullableUnix(v.CancelledAt), actorID, now.Unix(), kidID, v.ID)
		if err := oneRow(res, err); err != nil {
			return err
		}
		out, err = readVisitDetail(ctx, tx, kidID, v.ID)
		return err
	})
	if err != nil {
		return VisitDetail{}, err
	}
	return out, nil
}

// DeleteVisit removes the visit; with deleteLinked its rows go first, each as its own module deletes it (a whole
// administration, a course with its doses and reminders). Without, the links clear. actorID needs an active grant.
func (s *Store) DeleteVisit(ctx context.Context, kidID, actorID, id string, deleteLinked bool) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		if _, err := readVisit(ctx, tx, kidID, id); err != nil {
			return err
		}
		if deleteLinked {
			for _, q := range []string{
				`DELETE FROM growth_measurements WHERE kid_id = ? AND visit_id = ?`,
				`DELETE FROM vaccine_doses WHERE kid_id = ? AND administration IN (SELECT administration FROM vaccine_doses WHERE visit_id = ?)`,
				`DELETE FROM medications WHERE kid_id = ? AND visit_id = ?`,
			} {
				if _, err := tx.ExecContext(ctx, q, kidID, id); err != nil {
					return err
				}
			}
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM visits WHERE kid_id = ? AND id = ?`, kidID, id)
		return oneRow(res, err)
	})
}
