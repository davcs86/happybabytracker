package store

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

const bottleColumns = entryColumns + `, at, has_breast_milk, has_formula, has_cow_milk, has_other, volume_ml, volume_floz_h, volume_typed`

func scanBottle(row rowScanner) (model.BottleFeed, error) {
	var (
		b         model.BottleFeed
		at        int64
		has       [4]bool
		typedUnit string
	)
	e, err := scanEntry(row, &at, &has[0], &has[1], &has[2], &has[3], &b.Volume.ML, &b.Volume.FlOzHundredths, &typedUnit)
	if err != nil {
		return model.BottleFeed{}, err
	}
	b.Entry, b.At, b.Volume.Typed = e, unixTime(at), units.Unit(typedUnit)
	b.Contents = []string{}
	for i, name := range model.ContentNames {
		if has[i] {
			b.Contents = append(b.Contents, name)
		}
	}
	return b, nil
}

// contentFlags maps a canonical contents set to the four columns, in model.ContentNames order.
func contentFlags(set []string) [4]bool {
	var has [4]bool
	for i, name := range model.ContentNames {
		has[i] = slices.Contains(set, name)
	}
	return has
}

// CreateBottle stores b for kidID as actorID, who needs an active grant (else ErrNotFound); the caller validates the rest.
func (s *Store) CreateBottle(ctx context.Context, kidID, actorID string, b model.BottleFeed, now time.Time) (model.BottleFeed, error) {
	set, ok := model.CanonicalContents(b.Contents)
	if !ok {
		return model.BottleFeed{}, ErrInvalidContents
	}
	has := contentFlags(set)
	b.Contents = set
	b.ID, b.KidID, b.CreatedBy, b.UpdatedBy = model.NewID(), kidID, actorID, actorID
	b.CreatedAt, b.UpdatedAt = unixTime(now.Unix()), unixTime(now.Unix())
	b.At = unixTime(b.At.Unix())
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO bottle_feeds(`+bottleColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			b.ID, kidID, b.Note, actorID, actorID, b.CreatedAt.Unix(), b.UpdatedAt.Unix(),
			b.At.Unix(), has[0], has[1], has[2], has[3], b.Volume.ML, b.Volume.FlOzHundredths, string(b.Volume.Typed))
		return err
	})
	if err != nil {
		return model.BottleFeed{}, err
	}
	return b, nil
}

// Bottle is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) Bottle(ctx context.Context, kidID, id string) (model.BottleFeed, error) {
	return scanBottle(s.r.QueryRowContext(ctx, `SELECT `+bottleColumns+` FROM bottle_feeds WHERE kid_id = ? AND id = ?`, kidID, id))
}

// ListBottles returns the kid's bottles within q, newest first by (at, id); never nil.
func (s *Store) ListBottles(ctx context.Context, kidID string, q ListQuery) ([]model.BottleFeed, error) {
	return listEntries(ctx, s.r, `SELECT `+bottleColumns+` FROM bottle_feeds`, "at", kidID, q, scanBottle)
}

// UpdateBottle replaces every editable field of b.ID, which must belong to kidID; actorID needs an active grant.
func (s *Store) UpdateBottle(ctx context.Context, kidID, actorID string, b model.BottleFeed, now time.Time) error {
	set, ok := model.CanonicalContents(b.Contents)
	if !ok {
		return ErrInvalidContents
	}
	has := contentFlags(set)
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx,
			`UPDATE bottle_feeds SET at = ?, has_breast_milk = ?, has_formula = ?, has_cow_milk = ?, has_other = ?,
			   volume_ml = ?, volume_floz_h = ?, volume_typed = ?, note = ?, updated_by = ?, updated_at = ?
			 WHERE kid_id = ? AND id = ?`,
			b.At.Unix(), has[0], has[1], has[2], has[3], b.Volume.ML, b.Volume.FlOzHundredths, string(b.Volume.Typed),
			b.Note, actorID, now.Unix(), kidID, b.ID)
		return oneRow(res, err)
	})
}

// DeleteBottle is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) DeleteBottle(ctx context.Context, kidID, actorID, id string) error {
	return s.deleteEntry(ctx, "bottle_feeds", kidID, actorID, id)
}
