package store

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

// ModuleState is one module for a kid: Default comes from the age band, Overridden is a stored row, whose value
// Enabled then carries.
type ModuleState struct {
	Module                       model.Module
	Enabled, Default, Overridden bool
}

// DefaultEnabled is the spec's 7.2 table: every module is on below 24 months; from 24 months only growth,
// vaccination, sickness and medication stay on.
func DefaultEnabled(m model.Module, b kidtime.Band) bool {
	if b != kidtime.Band24Plus {
		return true
	}
	return slices.Contains([]model.Module{"growth", "vaccination", "sickness", "medication"}, m)
}

// moduleTable is a module's entry table, its time column and whether it holds timers; code constants.
type moduleTable struct {
	table, timeCol string
	timer          bool
}

var moduleTables = map[model.Module]moduleTable{
	"bottles": {"bottle_feeds", "at", false},
	"nursing": {"nursing_sessions", "started_at", true},
	"pumping": {"pumping_sessions", "started_at", false},
	"diapers": {"diapers", "at", false},
	"sleep":   {"sleep_sessions", "started_at", true},
	"feeding": {"meals", "at", false},
}

// Modules re-evaluates on every read: an entry since bandStart-lookback (or a running timer) re-enables an
// off-by-default module once, as a stored row in this transaction; a stored row is final. ErrNotFound for no kid.
func (s *Store) Modules(ctx context.Context, kidID string, loc *time.Location, now time.Time) ([]ModuleState, error) {
	var out []ModuleState
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var birthDate string
		if err := tx.QueryRowContext(ctx, `SELECT birth_date FROM kids WHERE id = ?`, kidID).Scan(&birthDate); errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		months, err := kidtime.AgeMonths(birthDate, loc, now)
		if err != nil {
			return err
		}
		band := kidtime.BandFor(months)
		start, err := kidtime.BandStart(birthDate, loc, band)
		if err != nil {
			return err
		}
		rows, err := moduleRows(ctx, tx, kidID)
		if err != nil {
			return err
		}
		since := start.Add(-constants.BandOverrideLookback).Unix()
		for _, m := range model.Modules {
			t, tracked := moduleTables[m]
			if _, stored := rows[m]; stored || !tracked || DefaultEnabled(m, band) {
				continue
			}
			running := "0"
			if t.timer {
				running = "ended_at IS NULL"
			}
			var keep bool
			if err := tx.QueryRowContext(ctx,
				`SELECT EXISTS (SELECT 1 FROM `+t.table+` WHERE kid_id = ? AND ((`+t.timeCol+` >= ? AND `+t.timeCol+` <= ?) OR `+running+`))`,
				kidID, since, now.Unix()).Scan(&keep); err != nil {
				return err
			}
			if !keep {
				continue
			}
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO kid_modules(kid_id, module, enabled, updated_at) VALUES (?, ?, 1, ?) ON CONFLICT DO NOTHING`,
				kidID, m, now.Unix()); err != nil {
				return err
			}
			rows[m] = true
		}
		out = make([]ModuleState, len(model.Modules))
		for i, m := range model.Modules {
			enabled, overridden := rows[m]
			def := DefaultEnabled(m, band)
			if !overridden {
				enabled = def
			}
			out[i] = ModuleState{Module: m, Enabled: enabled, Default: def, Overridden: overridden}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func moduleRows(ctx context.Context, tx *sql.Tx, kidID string) (map[model.Module]bool, error) {
	rows, err := tx.QueryContext(ctx, `SELECT module, enabled FROM kid_modules WHERE kid_id = ?`, kidID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[model.Module]bool{}
	for rows.Next() {
		var (
			m       model.Module
			enabled bool
		)
		if err := rows.Scan(&m, &enabled); err != nil {
			return nil, err
		}
		out[m] = enabled
	}
	return out, rows.Err()
}

// SetModule stores the kid's override for m, a model.Modules id; actorID must be a guardian (ErrNotFound without a
// grant, ErrForbidden as a caregiver).
func (s *Store) SetModule(ctx context.Context, kidID, actorID string, m model.Module, enabled bool, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := actorIsGuardian(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO kid_modules(kid_id, module, enabled, updated_at) VALUES (?, ?, ?, ?)
			 ON CONFLICT(kid_id, module) DO UPDATE SET enabled = excluded.enabled, updated_at = excluded.updated_at`,
			kidID, m, enabled, now.Unix())
		return err
	})
}
