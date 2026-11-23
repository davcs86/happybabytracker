package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

func mustVolume(t *testing.T, unit units.Unit, amount int64) units.Volume {
	t.Helper()
	v, err := units.NewVolume(unit, amount)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func strp(s string) *string { return &s }

func newBottle(t *testing.T, at time.Time, contents ...string) model.BottleFeed {
	t.Helper()
	return model.BottleFeed{At: at, Contents: contents, Volume: mustVolume(t, units.UnitML, 120)}
}

func mustBottle(t *testing.T, s *Store, kidID, actorID string, b model.BottleFeed) model.BottleFeed {
	t.Helper()
	got, err := s.CreateBottle(ctx, kidID, actorID, b, t0)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestBottleCRUD(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)

	in := newBottle(t, t0.Add(-time.Hour), model.ContentFormula, model.ContentBreastMilk)
	in.Note = strp("first")
	created, err := s.CreateBottle(ctx, k.ID, ana.ID, in, t0)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.KidID != k.ID || created.CreatedBy != ana.ID || created.UpdatedBy != ana.ID ||
		!created.CreatedAt.Equal(t0) || !created.UpdatedAt.Equal(t0) || !created.At.Equal(in.At) {
		t.Fatalf("created = %+v", created)
	}
	if want := []string{model.ContentBreastMilk, model.ContentFormula}; !slices.Equal(created.Contents, want) {
		t.Errorf("contents = %v, want %v", created.Contents, want)
	}
	got, err := s.Bottle(ctx, k.ID, created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("Bottle = %+v, %v; want %+v", got, err, created)
	}

	edit := got
	edit.At = t0.Add(-2 * time.Hour)
	edit.Contents = []string{model.ContentCowMilk}
	edit.Volume = mustVolume(t, units.UnitFlOz, 400)
	edit.Note = nil
	later := t0.Add(time.Hour)
	if err := s.UpdateBottle(ctx, k.ID, bo.ID, edit, later); err != nil {
		t.Fatal(err)
	}
	got, err = s.Bottle(ctx, k.ID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	edit.UpdatedBy, edit.UpdatedAt = bo.ID, later
	if !reflect.DeepEqual(got, edit) {
		t.Fatalf("after update = %+v, want %+v", got, edit)
	}
	if got.CreatedBy != ana.ID || !got.CreatedAt.Equal(t0) {
		t.Errorf("update touched the creation stamps: %+v", got.Entry)
	}

	list, err := s.ListBottles(ctx, k.ID, wideQuery())
	if err != nil || len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("ListBottles = %+v, %v", list, err)
	}
	if err := s.DeleteBottle(ctx, k.ID, bo.ID, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Bottle(ctx, k.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Bottle after delete: %v", err)
	}
	if err := s.DeleteBottle(ctx, k.ID, bo.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: %v", err)
	}
	if err := s.UpdateBottle(ctx, k.ID, bo.ID, edit, later); !errors.Is(err, ErrNotFound) {
		t.Errorf("update after delete: %v", err)
	}
}

func wideQuery() ListQuery {
	return ListQuery{From: time.Unix(0, 0), To: t0.Add(24 * time.Hour), Limit: 100}
}

func TestBottleBelongsToItsKid(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	mine, other := mustKid(t, s, ana.ID, "Mine"), mustKid(t, s, ana.ID, "Other")
	b := mustBottle(t, s, mine.ID, ana.ID, newBottle(t, t0, model.ContentFormula))

	if _, err := s.Bottle(ctx, other.ID, b.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("read through another kid: %v", err)
	}
	edit := b
	edit.Contents = []string{model.ContentOther}
	if err := s.UpdateBottle(ctx, other.ID, ana.ID, edit, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("update through another kid: %v", err)
	}
	if err := s.DeleteBottle(ctx, other.ID, ana.ID, b.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete through another kid: %v", err)
	}
	if list, err := s.ListBottles(ctx, other.ID, wideQuery()); err != nil || len(list) != 0 {
		t.Errorf("other kid's list = %+v, %v", list, err)
	}
	got, err := s.Bottle(ctx, mine.ID, b.ID)
	if err != nil || !reflect.DeepEqual(got, b) {
		t.Errorf("entry changed: %+v, %v", got, err)
	}
}

func TestBottleWriteByDisabledActor(t *testing.T) {
	s := openMem(t)
	ana, bo, cy := mustParent(t, s, "ana"), mustParent(t, s, "bo"), mustParent(t, s, "cy")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)
	b := mustBottle(t, s, k.ID, ana.ID, newBottle(t, t0, model.ContentFormula))
	if err := s.SetParentDisabled(ctx, bo.ID, true, t0); err != nil {
		t.Fatal(err)
	}

	for name, actor := range map[string]string{"disabled caregiver": bo.ID, "no grant": cy.ID} {
		if _, err := s.CreateBottle(ctx, k.ID, actor, newBottle(t, t0, model.ContentFormula), t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s create: %v", name, err)
		}
		if err := s.UpdateBottle(ctx, k.ID, actor, b, t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s update: %v", name, err)
		}
		if err := s.DeleteBottle(ctx, k.ID, actor, b.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s delete: %v", name, err)
		}
	}
	if list, err := s.ListBottles(ctx, k.ID, wideQuery()); err != nil || len(list) != 1 {
		t.Errorf("list = %d rows, %v; want the one seeded entry", len(list), err)
	}
}

func TestEditingNoteKeepsTypedVolume(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	in := newBottle(t, t0, model.ContentFormula)
	in.Volume = mustVolume(t, units.UnitFlOz, 400)
	b := mustBottle(t, s, k.ID, ana.ID, in)

	b.Note = strp("only the note changed")
	if err := s.UpdateBottle(ctx, k.ID, ana.ID, b, t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	got, err := s.Bottle(ctx, k.ID, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want := (units.Volume{ML: 118, FlOzHundredths: 400, Typed: units.UnitFlOz}); got.Volume != want {
		t.Errorf("volume = %+v, want %+v", got.Volume, want)
	}
}

func TestMixedBottleContents(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")

	for _, c := range []struct {
		name     string
		in, want []string
	}{
		{"breast milk and formula", []string{model.ContentFormula, model.ContentBreastMilk}, []string{model.ContentBreastMilk, model.ContentFormula}},
		{"cow milk alone", []string{model.ContentCowMilk}, []string{model.ContentCowMilk}},
		{"all four, scrambled", []string{model.ContentOther, model.ContentCowMilk, model.ContentFormula, model.ContentBreastMilk},
			[]string{model.ContentBreastMilk, model.ContentFormula, model.ContentCowMilk, model.ContentOther}},
	} {
		b, err := s.CreateBottle(ctx, k.ID, ana.ID, newBottle(t, t0, c.in...), t0)
		if err != nil || !slices.Equal(b.Contents, c.want) {
			t.Fatalf("%s: created %v, %v; want %v", c.name, b.Contents, err, c.want)
		}
		got, err := s.Bottle(ctx, k.ID, b.ID)
		if err != nil || !slices.Equal(got.Contents, c.want) {
			t.Errorf("%s: read back %v, %v; want %v", c.name, got.Contents, err, c.want)
		}
	}

	valid := mustBottle(t, s, k.ID, ana.ID, newBottle(t, t0, model.ContentFormula))
	for name, contents := range map[string][]string{
		"empty":     {},
		"nil":       nil,
		"duplicate": {model.ContentFormula, model.ContentFormula},
		"unknown":   {"goat_milk"},
	} {
		if _, err := s.CreateBottle(ctx, k.ID, ana.ID, newBottle(t, t0, contents...), t0); !errors.Is(err, ErrInvalidContents) {
			t.Errorf("create %s: %v, want ErrInvalidContents", name, err)
		}
		bad := valid
		bad.Contents = contents
		if err := s.UpdateBottle(ctx, k.ID, ana.ID, bad, t0); !errors.Is(err, ErrInvalidContents) {
			t.Errorf("update %s: %v, want ErrInvalidContents", name, err)
		}
	}
	got, err := s.Bottle(ctx, k.ID, valid.ID)
	if err != nil || !slices.Equal(got.Contents, []string{model.ContentFormula}) {
		t.Errorf("refused updates changed the row: %v, %v", got.Contents, err)
	}
}

func TestListNewestFirstWithEqualTimestamps(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	same := t0.Add(-time.Hour)
	var ids []string
	for _, at := range []time.Time{same, same.Add(-time.Minute), same, same.Add(time.Minute), same} {
		ids = append(ids, mustBottle(t, s, k.ID, ana.ID, newBottle(t, at, model.ContentFormula)).ID)
	}
	atOf := map[string]int64{ids[0]: same.Unix(), ids[1]: same.Unix() - 60, ids[2]: same.Unix(), ids[3]: same.Unix() + 60, ids[4]: same.Unix()}
	want := slices.Clone(ids)
	slices.SortFunc(want, func(a, b string) int {
		if atOf[a] != atOf[b] {
			return int(atOf[b] - atOf[a])
		}
		return -strings.Compare(a, b)
	})

	got, err := s.ListBottles(ctx, k.ID, wideQuery())
	if err != nil {
		t.Fatal(err)
	}
	if gotIDs := bottleIDs(got); !slices.Equal(gotIDs, want) {
		t.Fatalf("order = %v, want %v", gotIDs, want)
	}

	// Paging by cursor visits every row once, in the same order.
	var paged []string
	q := wideQuery()
	q.Limit = 2
	for range ids {
		page, err := s.ListBottles(ctx, k.ID, q)
		if err != nil {
			t.Fatal(err)
		}
		if len(page) == 0 {
			break
		}
		paged = append(paged, bottleIDs(page)...)
		last := page[len(page)-1]
		q.After = &Cursor{At: last.At.Unix(), ID: last.ID}
	}
	if !slices.Equal(paged, want) {
		t.Errorf("paged = %v, want %v", paged, want)
	}
}

func TestListBottlesRangeIsHalfOpenAndLimited(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	var ats []time.Time
	for i := range 4 {
		at := t0.Add(-time.Duration(i) * time.Hour)
		ats = append(ats, at)
		mustBottle(t, s, k.ID, ana.ID, newBottle(t, at, model.ContentFormula))
	}
	got, err := s.ListBottles(ctx, k.ID, ListQuery{From: ats[2], To: ats[0], Limit: 10})
	if err != nil || len(got) != 2 || !got[0].At.Equal(ats[1]) || !got[1].At.Equal(ats[2]) {
		t.Fatalf("[ats[2], ats[0]) = %+v, %v", got, err)
	}
	got, err = s.ListBottles(ctx, k.ID, ListQuery{From: ats[3], To: t0.Add(time.Second), Limit: 1})
	if err != nil || len(got) != 1 || !got[0].At.Equal(ats[0]) {
		t.Fatalf("limit 1 = %+v, %v", got, err)
	}
}

func bottleIDs(bs []model.BottleFeed) []string {
	out := make([]string, len(bs))
	for i, b := range bs {
		out[i] = b.ID
	}
	return out
}

// advancedBottle is a bottle whose sources are typed in unit and whose total is their sum.
func advancedBottle(t *testing.T, at time.Time, unit units.Unit, amounts map[string]int64) model.BottleFeed {
	t.Helper()
	b := model.BottleFeed{At: at}
	var sum int64
	for _, name := range model.ContentNames {
		if n, ok := amounts[name]; ok {
			b.Contents = append(b.Contents, name)
			b.Sources = append(b.Sources, model.BottleSource{Source: name, Volume: mustVolume(t, unit, n)})
			sum += n
		}
	}
	b.Volume = mustVolume(t, unit, sum)
	return b
}

func sourceRows(t *testing.T, s *Store, bottleID string) int {
	t.Helper()
	var n int
	if err := s.w.QueryRowContext(ctx, `SELECT COUNT(*) FROM bottle_feed_sources WHERE bottle_id = ?`, bottleID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestBottleSourcesRoundTrip(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")

	created := mustBottle(t, s, k.ID, ana.ID, advancedBottle(t, t0, units.UnitFlOz, map[string]int64{model.ContentFormula: 250, model.ContentBreastMilk: 150}))
	if want := []string{model.ContentBreastMilk, model.ContentFormula}; !slices.Equal(created.Contents, want) {
		t.Fatalf("contents = %v, want %v", created.Contents, want)
	}
	got, err := s.Bottle(ctx, k.ID, created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("Bottle = %+v, %v; want %+v", got, err, created)
	}
	if len(got.Sources) != 2 || got.Sources[0].Source != model.ContentBreastMilk || got.Sources[0].Volume.Typed != units.UnitFlOz ||
		got.Sources[0].Volume.FlOzHundredths != 150 || got.Sources[1].Volume.FlOzHundredths != 250 || got.Volume.FlOzHundredths != 400 {
		t.Errorf("sources = %+v total %+v", got.Sources, got.Volume)
	}
	list, err := s.ListBottles(ctx, k.ID, wideQuery())
	if err != nil || len(list) != 1 || !reflect.DeepEqual(list[0], created) {
		t.Fatalf("ListBottles = %+v, %v", list, err)
	}

	simple := mustBottle(t, s, k.ID, ana.ID, newBottle(t, t0.Add(-time.Hour), model.ContentFormula))
	if got, err := s.Bottle(ctx, k.ID, simple.ID); err != nil || len(got.Sources) != 0 {
		t.Errorf("simple bottle = %+v, %v; want no sources", got, err)
	}
}

func TestBottleSourcesReplacedOnUpdate(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k, other := mustKid(t, s, ana.ID, "Lía"), mustKid(t, s, ana.ID, "Other")
	b := mustBottle(t, s, k.ID, ana.ID, advancedBottle(t, t0, units.UnitML, map[string]int64{model.ContentBreastMilk: 60, model.ContentFormula: 90}))

	next := advancedBottle(t, t0, units.UnitML, map[string]int64{model.ContentFormula: 70, model.ContentCowMilk: 30})
	next.ID = b.ID
	if err := s.UpdateBottle(ctx, k.ID, ana.ID, next, t0); err != nil {
		t.Fatal(err)
	}
	got, err := s.Bottle(ctx, k.ID, b.ID)
	if err != nil || !reflect.DeepEqual(got.Sources, next.Sources) || !slices.Equal(got.Contents, next.Contents) || got.Volume != next.Volume {
		t.Fatalf("after replace = %+v, %v; want sources %+v", got, err, next.Sources)
	}
	if n := sourceRows(t, s, b.ID); n != 2 {
		t.Errorf("%d source rows, want 2", n)
	}

	if err := s.UpdateBottle(ctx, other.ID, ana.ID, advancedBottle(t, t0, units.UnitML, map[string]int64{model.ContentOther: 10}), t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("update through another kid: %v", err)
	}
	if n := sourceRows(t, s, b.ID); n != 2 {
		t.Errorf("%d source rows after a refused update, want 2", n)
	}

	simple := newBottle(t, t0, model.ContentFormula)
	simple.ID = b.ID
	if err := s.UpdateBottle(ctx, k.ID, ana.ID, simple, t0); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Bottle(ctx, k.ID, b.ID); err != nil || len(got.Sources) != 0 || sourceRows(t, s, b.ID) != 0 {
		t.Errorf("after switching to simple = %+v, %v; %d rows", got, err, sourceRows(t, s, b.ID))
	}
}

func TestSourcesMustMatchContents(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	good := mustBottle(t, s, k.ID, ana.ID, advancedBottle(t, t0, units.UnitML, map[string]int64{model.ContentBreastMilk: 60, model.ContentFormula: 90}))

	src := func(names ...string) []model.BottleSource {
		var out []model.BottleSource
		for _, n := range names {
			out = append(out, model.BottleSource{Source: n, Volume: mustVolume(t, units.UnitML, 10)})
		}
		return out
	}
	for name, c := range map[string]struct {
		contents []string
		sources  []model.BottleSource
	}{
		"a content without a source":  {[]string{model.ContentBreastMilk, model.ContentFormula}, src(model.ContentBreastMilk)},
		"a source without a content":  {[]string{model.ContentBreastMilk}, src(model.ContentBreastMilk, model.ContentFormula)},
		"another source":              {[]string{model.ContentBreastMilk}, src(model.ContentFormula)},
		"a repeated source":           {[]string{model.ContentBreastMilk, model.ContentFormula}, src(model.ContentBreastMilk, model.ContentBreastMilk)},
		"an unknown source":           {[]string{model.ContentBreastMilk}, src("goat_milk")},
		"a repeated source, one name": {[]string{model.ContentBreastMilk}, src(model.ContentBreastMilk, model.ContentBreastMilk)},
	} {
		bad := model.BottleFeed{At: t0, Contents: c.contents, Sources: c.sources, Volume: mustVolume(t, units.UnitML, 10)}
		if _, err := s.CreateBottle(ctx, k.ID, ana.ID, bad, t0); !errors.Is(err, ErrInvalidContents) {
			t.Errorf("create with %s: %v, want ErrInvalidContents", name, err)
		}
		bad.ID = good.ID
		if err := s.UpdateBottle(ctx, k.ID, ana.ID, bad, t0); !errors.Is(err, ErrInvalidContents) {
			t.Errorf("update with %s: %v, want ErrInvalidContents", name, err)
		}
	}
	if got, err := s.Bottle(ctx, k.ID, good.ID); err != nil || !reflect.DeepEqual(got, good) {
		t.Errorf("a refused update changed the bottle: %+v, %v", got, err)
	}
	if list, err := s.ListBottles(ctx, k.ID, wideQuery()); err != nil || len(list) != 1 {
		t.Errorf("a refused create left %d bottles, %v", len(list), err)
	}
}

// countingDriver counts the QueryContext calls that reach the wrapped driver.
type countingDriver struct {
	inner   driver.Driver
	queries *atomic.Int64
}

func (d countingDriver) Open(name string) (driver.Conn, error) {
	c, err := d.inner.Open(name)
	if err != nil {
		return nil, err
	}
	return &countingConn{Conn: c, queries: d.queries}, nil
}

type countingConn struct {
	driver.Conn
	queries *atomic.Int64
}

func (c *countingConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.queries.Add(1)
	return c.Conn.(driver.QueryerContext).QueryContext(ctx, query, args)
}

func (c *countingConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return c.Conn.(driver.ExecerContext).ExecContext(ctx, query, args)
}

func (c *countingConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	return c.Conn.(driver.ConnPrepareContext).PrepareContext(ctx, query)
}

func (c *countingConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return c.Conn.(driver.ConnBeginTx).BeginTx(ctx, opts)
}

type countingConnector struct {
	dsn string
	d   countingDriver
}

func (c countingConnector) Connect(context.Context) (driver.Conn, error) { return c.d.Open(c.dsn) }
func (c countingConnector) Driver() driver.Driver                        { return c.d }

// countReads makes s's read pool count its queries.
func countReads(t *testing.T, s *Store) *atomic.Int64 {
	t.Helper()
	n := new(atomic.Int64)
	rdsn := "file:" + filepath.Join(s.cleanup, dbFile) + "?mode=ro&_pragma=foreign_keys(1)"
	r := sql.OpenDB(countingConnector{rdsn, countingDriver{s.r.Driver(), n}})
	t.Cleanup(func() { r.Close() })
	s.r = r
	return n
}

func TestListLoadsSourcesWithOneExtraQuery(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	for i := range 100 {
		amounts := map[string]int64{model.ContentBreastMilk: 60, model.ContentFormula: 90}
		mustBottle(t, s, k.ID, ana.ID, advancedBottle(t, t0.Add(-time.Duration(i)*time.Minute), units.UnitML, amounts))
	}
	n := countReads(t, s)

	if list, err := s.ListBottles(ctx, k.ID, wideQuery()); err != nil || len(list) != 100 || len(list[99].Sources) != 2 {
		t.Fatalf("ListBottles = %d rows, %v", len(list), err)
	}
	if got := n.Load(); got != 2 {
		t.Errorf("a list of 100 bottles ran %d queries, want 2 (the page and one for every source)", got)
	}

	n.Store(0)
	if list, err := s.ListBottles(ctx, k.ID, ListQuery{From: t0.Add(time.Hour), To: t0.Add(2 * time.Hour), Limit: 10}); err != nil || len(list) != 0 {
		t.Fatalf("empty list = %v, %v", list, err)
	}
	if got := n.Load(); got != 1 {
		t.Errorf("an empty list ran %d queries, want 1", got)
	}
}

func TestTimelineBottleCarriesSources(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	adv := mustBottle(t, s, k.ID, ana.ID, advancedBottle(t, t0, units.UnitML, map[string]int64{model.ContentBreastMilk: 60, model.ContentFormula: 90}))
	mustBottle(t, s, k.ID, ana.ID, newBottle(t, t0.Add(-time.Hour), model.ContentFormula))

	items, err := s.Timeline(ctx, k.ID, []string{"bottle"}, wideQuery())
	if err != nil || len(items) != 2 {
		t.Fatalf("Timeline = %+v, %v", items, err)
	}
	if got := items[0].Entry.(model.BottleFeed); !reflect.DeepEqual(got, adv) {
		t.Errorf("advanced item = %+v, want %+v", got, adv)
	}
	if got := items[1].Entry.(model.BottleFeed); len(got.Sources) != 0 {
		t.Errorf("simple item has sources %+v", got.Sources)
	}
}
