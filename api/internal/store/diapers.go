package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

const diaperColumns = entryColumns + `, at, kind`

func scanDiaper(row rowScanner) (model.Diaper, error) {
	var (
		d  model.Diaper
		at int64
	)
	e, err := scanEntry(row, &at, &d.Kind)
	if err != nil {
		return model.Diaper{}, err
	}
	d.Entry, d.At = e, unixTime(at)
	return d, nil
}

// CreateDiaper stores d for kidID as actorID, who needs an active grant (else ErrNotFound); the caller validates the rest.
func (s *Store) CreateDiaper(ctx context.Context, kidID, actorID string, d model.Diaper, now time.Time) (model.Diaper, error) {
	d.ID, d.KidID, d.CreatedBy, d.UpdatedBy = model.NewID(), kidID, actorID, actorID
	d.CreatedAt, d.UpdatedAt = unixTime(now.Unix()), unixTime(now.Unix())
	d.At = unixTime(d.At.Unix())
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO diapers(`+diaperColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			d.ID, kidID, d.Note, actorID, actorID, d.CreatedAt.Unix(), d.UpdatedAt.Unix(), d.At.Unix(), d.Kind)
		return err
	})
	if err != nil {
		return model.Diaper{}, err
	}
	return d, nil
}

// Diaper is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) Diaper(ctx context.Context, kidID, id string) (model.Diaper, error) {
	return scanDiaper(s.r.QueryRowContext(ctx, `SELECT `+diaperColumns+` FROM diapers WHERE kid_id = ? AND id = ?`, kidID, id))
}

// ListDiapers returns the kid's diapers within q, newest first by (at, id); never nil.
func (s *Store) ListDiapers(ctx context.Context, kidID string, q ListQuery) ([]model.Diaper, error) {
	return listEntries(ctx, s.r, `SELECT `+diaperColumns+` FROM diapers`, "at", kidID, q, scanDiaper)
}

// UpdateDiaper replaces every editable field of d.ID, which must belong to kidID; actorID needs an active grant.
func (s *Store) UpdateDiaper(ctx context.Context, kidID, actorID string, d model.Diaper, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx,
			`UPDATE diapers SET at = ?, kind = ?, note = ?, updated_by = ?, updated_at = ? WHERE kid_id = ? AND id = ?`,
			d.At.Unix(), d.Kind, d.Note, actorID, now.Unix(), kidID, d.ID)
		return oneRow(res, err)
	})
}

// DeleteDiaper is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) DeleteDiaper(ctx context.Context, kidID, actorID, id string) error {
	return s.deleteEntry(ctx, "diapers", kidID, actorID, id)
}
