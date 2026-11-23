package store

import (
	"context"
	"database/sql"
	"slices"
	"strings"
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

// orderSources returns sources in contents order; ok is false unless there is exactly one per name of contents.
// No sources is a simple bottle: nil.
func orderSources(contents []string, sources []model.BottleSource) (ordered []model.BottleSource, ok bool) {
	if len(sources) == 0 {
		return nil, true
	}
	for _, name := range contents {
		i := slices.IndexFunc(sources, func(s model.BottleSource) bool { return s.Source == name })
		if i < 0 {
			return nil, false
		}
		ordered = append(ordered, sources[i])
	}
	return ordered, len(ordered) == len(sources)
}

// replaceSources makes sources, in order, the whole set stored for bottleID.
func replaceSources(ctx context.Context, tx *sql.Tx, bottleID string, sources []model.BottleSource) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM bottle_feed_sources WHERE bottle_id = ?`, bottleID); err != nil {
		return err
	}
	for _, src := range sources {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO bottle_feed_sources(bottle_id, source, volume_ml, volume_floz_h, volume_typed) VALUES (?, ?, ?, ?, ?)`,
			bottleID, src.Source, src.Volume.ML, src.Volume.FlOzHundredths, string(src.Volume.Typed)); err != nil {
			return err
		}
	}
	return nil
}

// withSources sets Sources on each of bottles with one query for the whole page.
func (s *Store) withSources(ctx context.Context, bottles []model.BottleFeed) ([]model.BottleFeed, error) {
	if len(bottles) == 0 {
		return bottles, nil
	}
	at := make(map[string]int, len(bottles))
	args := make([]any, len(bottles))
	for i, b := range bottles {
		at[b.ID], args[i] = i, b.ID
	}
	rows, err := s.r.QueryContext(ctx,
		`SELECT bottle_id, source, volume_ml, volume_floz_h, volume_typed FROM bottle_feed_sources WHERE bottle_id IN (?`+
			strings.Repeat(`, ?`, len(args)-1)+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id, typed string
			src       model.BottleSource
		)
		if err := rows.Scan(&id, &src.Source, &src.Volume.ML, &src.Volume.FlOzHundredths, &typed); err != nil {
			return nil, err
		}
		src.Volume.Typed = units.Unit(typed)
		i := at[id]
		bottles[i].Sources = append(bottles[i].Sources, src)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range bottles {
		slices.SortFunc(bottles[i].Sources, func(a, b model.BottleSource) int {
			return slices.Index(model.ContentNames, a.Source) - slices.Index(model.ContentNames, b.Source)
		})
	}
	return bottles, nil
}

// CreateBottle stores b for kidID as actorID, who needs an active grant (else ErrNotFound); the caller validates the rest.
// b.Sources, when set, name exactly b.Contents.
func (s *Store) CreateBottle(ctx context.Context, kidID, actorID string, b model.BottleFeed, now time.Time) (model.BottleFeed, error) {
	set, ok := model.CanonicalContents(b.Contents)
	if !ok {
		return model.BottleFeed{}, ErrInvalidContents
	}
	sources, ok := orderSources(set, b.Sources)
	if !ok {
		return model.BottleFeed{}, ErrInvalidContents
	}
	has := contentFlags(set)
	b.Contents, b.Sources = set, sources
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
		if err != nil {
			return err
		}
		return replaceSources(ctx, tx, b.ID, b.Sources)
	})
	if err != nil {
		return model.BottleFeed{}, err
	}
	return b, nil
}

// Bottle is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) Bottle(ctx context.Context, kidID, id string) (model.BottleFeed, error) {
	b, err := scanBottle(s.r.QueryRowContext(ctx, `SELECT `+bottleColumns+` FROM bottle_feeds WHERE kid_id = ? AND id = ?`, kidID, id))
	if err != nil {
		return model.BottleFeed{}, err
	}
	one, err := s.withSources(ctx, []model.BottleFeed{b})
	if err != nil {
		return model.BottleFeed{}, err
	}
	return one[0], nil
}

// ListBottles returns the kid's bottles within q, newest first by (at, id); never nil.
func (s *Store) ListBottles(ctx context.Context, kidID string, q ListQuery) ([]model.BottleFeed, error) {
	rows, err := listEntries(ctx, s.r, `SELECT `+bottleColumns+` FROM bottle_feeds`, "at", kidID, q, scanBottle)
	if err != nil {
		return nil, err
	}
	return s.withSources(ctx, rows)
}

// UpdateBottle replaces every editable field of b.ID, which must belong to kidID, and its whole set of sources;
// actorID needs an active grant.
func (s *Store) UpdateBottle(ctx context.Context, kidID, actorID string, b model.BottleFeed, now time.Time) error {
	set, ok := model.CanonicalContents(b.Contents)
	if !ok {
		return ErrInvalidContents
	}
	sources, ok := orderSources(set, b.Sources)
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
		if err := oneRow(res, err); err != nil {
			return err
		}
		return replaceSources(ctx, tx, b.ID, sources)
	})
}

// DeleteBottle is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) DeleteBottle(ctx context.Context, kidID, actorID, id string) error {
	return s.deleteEntry(ctx, "bottle_feeds", kidID, actorID, id)
}
