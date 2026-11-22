package store

import (
	"database/sql"
	"errors"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

// entryColumns are the columns every tracker table shares, in scanEntry's order.
const entryColumns = `id, kid_id, note, created_by, updated_by, created_at, updated_at`

func unixTime(n int64) time.Time { return time.Unix(n, 0).UTC() }

// scanEntry scans entryColumns, then the destinations in extra; ErrNotFound when the row is absent.
func scanEntry(row rowScanner, extra ...any) (model.Entry, error) {
	var (
		e                model.Entry
		note             sql.NullString
		created, updated int64
	)
	dest := append([]any{&e.ID, &e.KidID, &note, &e.CreatedBy, &e.UpdatedBy, &created, &updated}, extra...)
	if err := row.Scan(dest...); errors.Is(err, sql.ErrNoRows) {
		return model.Entry{}, ErrNotFound
	} else if err != nil {
		return model.Entry{}, err
	}
	if note.Valid {
		e.Note = &note.String
	}
	e.CreatedAt, e.UpdatedAt = unixTime(created), unixTime(updated)
	return e, nil
}

// pageTail is the WHERE ... LIMIT tail that lists a kid's rows by timeCol, newest first, within q.
// timeCol is a column name from code, never input.
func pageTail(timeCol, kidID string, q ListQuery) (string, []any) {
	tail := ` WHERE kid_id = ? AND ` + timeCol + ` >= ? AND ` + timeCol + ` < ?`
	args := []any{kidID, q.From.Unix(), q.To.Unix()}
	if q.After != nil {
		tail += ` AND (` + timeCol + ` < ? OR (` + timeCol + ` = ? AND id < ?))`
		args = append(args, q.After.At, q.After.At, q.After.ID)
	}
	return tail + ` ORDER BY ` + timeCol + ` DESC, id DESC LIMIT ?`, append(args, q.Limit)
}
