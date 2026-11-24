package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

const milestoneColumns = entryColumns + `, catalog_item, custom_text, reached_on`

func scanMilestone(row rowScanner) (model.Milestone, error) {
	var (
		m          model.Milestone
		item, text sql.NullString
	)
	e, err := scanEntry(row, &item, &text, &m.ReachedOn)
	if err != nil {
		return model.Milestone{}, err
	}
	m.Entry, m.Item, m.Text = e, nullableString(item), nullableString(text)
	return m, nil
}

// CreateMilestone stores m for kidID as actorID, who needs an active grant (else ErrNotFound); ErrConflict when the kid
// already holds MaxMilestonesPerKid rows, ErrDuplicate when m.Item is already reached. The caller validates the rest.
func (s *Store) CreateMilestone(ctx context.Context, kidID, actorID string, m model.Milestone, now time.Time) (model.Milestone, error) {
	m.ID, m.KidID, m.CreatedBy, m.UpdatedBy = model.NewID(), kidID, actorID, actorID
	m.CreatedAt, m.UpdatedAt = unixTime(now.Unix()), unixTime(now.Unix())
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM milestones_reached WHERE kid_id = ?`, kidID).Scan(&n); err != nil {
			return err
		}
		if n >= constants.MaxMilestonesPerKid {
			return ErrConflict
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO milestones_reached(`+milestoneColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			m.ID, kidID, m.Note, actorID, actorID, m.CreatedAt.Unix(), m.UpdatedAt.Unix(), m.Item, m.Text, m.ReachedOn)
		// Ids are random, so only the (kid, catalog item) index can be the violated one.
		var se *sqlite.Error
		if errors.As(err, &se) && se.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
			return ErrDuplicate
		}
		return err
	})
	if err != nil {
		return model.Milestone{}, err
	}
	return m, nil
}

// Milestone is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) Milestone(ctx context.Context, kidID, id string) (model.Milestone, error) {
	return scanMilestone(s.r.QueryRowContext(ctx, `SELECT `+milestoneColumns+` FROM milestones_reached WHERE kid_id = ? AND id = ?`, kidID, id))
}

// ListMilestones returns all of the kid's reached milestones, newest first by (reached_on, id); never nil.
func (s *Store) ListMilestones(ctx context.Context, kidID string) ([]model.Milestone, error) {
	return listMilestones(ctx, s.r, kidID)
}

// KidMilestones reads the kid and ListMilestones' rows from one snapshot; ErrNotFound for an unknown kid.
func (s *Store) KidMilestones(ctx context.Context, kidID string) (model.Kid, []model.Milestone, error) {
	var (
		kid  model.Kid
		rows []model.Milestone
	)
	err := s.inReadTx(ctx, func(q querier) error {
		var err error
		if kid, err = scanKid(q.QueryRowContext(ctx, `SELECT `+kidColumns+` FROM kids k WHERE k.id = ?`, kidID)); err != nil {
			return err
		}
		rows, err = listMilestones(ctx, q, kidID)
		return err
	})
	if err != nil {
		return model.Kid{}, nil, err
	}
	return kid, rows, nil
}

func listMilestones(ctx context.Context, q querier, kidID string) ([]model.Milestone, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT `+milestoneColumns+` FROM milestones_reached WHERE kid_id = ? ORDER BY reached_on DESC, id DESC LIMIT ?`,
		kidID, constants.MaxMilestonesPerKid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Milestone{}
	for rows.Next() {
		m, err := scanMilestone(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// UpdateMilestone replaces ReachedOn, Text and Note of m.ID, which must belong to kidID; Item never changes.
// actorID needs an active grant. The caller validates the rest.
func (s *Store) UpdateMilestone(ctx context.Context, kidID, actorID string, m model.Milestone, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx,
			`UPDATE milestones_reached SET reached_on = ?, custom_text = ?, note = ?, updated_by = ?, updated_at = ?
			 WHERE kid_id = ? AND id = ?`, m.ReachedOn, m.Text, m.Note, actorID, now.Unix(), kidID, m.ID)
		return oneRow(res, err)
	})
}

// DeleteMilestone is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) DeleteMilestone(ctx context.Context, kidID, actorID, id string) error {
	return s.deleteEntry(ctx, "milestones_reached", kidID, actorID, id)
}
