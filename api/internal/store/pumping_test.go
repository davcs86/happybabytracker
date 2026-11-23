package store

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

func newPumping(t *testing.T, startedAt time.Time) model.PumpingSession {
	t.Helper()
	return model.PumpingSession{StartedAt: startedAt, DurationSeconds: 900, Volume: mustVolume(t, units.UnitML, 90), Side: "both"}
}

func mustPumping(t *testing.T, s *Store, kidID, actorID string, p model.PumpingSession) model.PumpingSession {
	t.Helper()
	got, err := s.CreatePumping(ctx, kidID, actorID, p, t0)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestPumpingCRUD(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)

	in := newPumping(t, t0.Add(-time.Hour))
	in.Note = strp("first")
	created, err := s.CreatePumping(ctx, k.ID, ana.ID, in, t0)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.KidID != k.ID || created.CreatedBy != ana.ID || created.UpdatedBy != ana.ID ||
		!created.CreatedAt.Equal(t0) || !created.UpdatedAt.Equal(t0) || !created.StartedAt.Equal(in.StartedAt) {
		t.Fatalf("created = %+v", created)
	}
	got, err := s.Pumping(ctx, k.ID, created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("Pumping = %+v, %v; want %+v", got, err, created)
	}

	edit := got
	edit.StartedAt, edit.DurationSeconds, edit.Side = t0.Add(-2*time.Hour), 1, "left"
	edit.Volume = mustVolume(t, units.UnitFlOz, 400)
	edit.Note = nil
	later := t0.Add(time.Hour)
	if err := s.UpdatePumping(ctx, k.ID, bo.ID, edit, later); err != nil {
		t.Fatal(err)
	}
	edit.UpdatedBy, edit.UpdatedAt = bo.ID, later
	if got, err = s.Pumping(ctx, k.ID, created.ID); err != nil || !reflect.DeepEqual(got, edit) {
		t.Fatalf("after update = %+v, %v; want %+v", got, err, edit)
	}
	if list, err := s.ListPumping(ctx, k.ID, wideQuery()); err != nil || len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("ListPumping = %+v, %v", list, err)
	}
	if err := s.DeletePumping(ctx, k.ID, bo.ID, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Pumping(ctx, k.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete: %v", err)
	}
	if err := s.DeletePumping(ctx, k.ID, bo.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: %v", err)
	}
	if err := s.UpdatePumping(ctx, k.ID, bo.ID, edit, later); !errors.Is(err, ErrNotFound) {
		t.Errorf("update after delete: %v", err)
	}
}

// A session is recorded on the kid it is for (spec 6.3), whoever pumped.
func TestPumpingBelongsToItsKidAndItsWriter(t *testing.T) {
	s := openMem(t)
	ana, bo, cy := mustParent(t, s, "ana"), mustParent(t, s, "bo"), mustParent(t, s, "cy")
	mine, other := mustKid(t, s, ana.ID, "Mine"), mustKid(t, s, ana.ID, "Other")
	mustGrant(t, s, mine.ID, ana.ID, "bo", model.RoleCaregiver)
	p := mustPumping(t, s, mine.ID, ana.ID, newPumping(t, t0))

	if _, err := s.Pumping(ctx, other.ID, p.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("read through another kid: %v", err)
	}
	if err := s.UpdatePumping(ctx, other.ID, ana.ID, p, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("update through another kid: %v", err)
	}
	if err := s.DeletePumping(ctx, other.ID, ana.ID, p.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete through another kid: %v", err)
	}
	if list, err := s.ListPumping(ctx, other.ID, wideQuery()); err != nil || len(list) != 0 {
		t.Errorf("other kid's list = %+v, %v", list, err)
	}

	if err := s.SetParentDisabled(ctx, bo.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	for name, actor := range map[string]string{"no grant": cy.ID, "disabled caregiver": bo.ID} {
		if _, err := s.CreatePumping(ctx, mine.ID, actor, newPumping(t, t0), t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s create: %v", name, err)
		}
		if err := s.UpdatePumping(ctx, mine.ID, actor, p, t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s update: %v", name, err)
		}
		if err := s.DeletePumping(ctx, mine.ID, actor, p.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s delete: %v", name, err)
		}
	}
	if got, err := s.Pumping(ctx, mine.ID, p.ID); err != nil || !reflect.DeepEqual(got, p) {
		t.Errorf("entry changed: %+v, %v", got, err)
	}
}

func TestPumpingListNewestFirstByStartWithEqualTimestamps(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	same := t0.Add(-time.Hour)
	var ids []string
	starts := []time.Time{same, same.Add(-time.Minute), same, same.Add(time.Minute), same}
	for _, at := range starts {
		ids = append(ids, mustPumping(t, s, k.ID, ana.ID, newPumping(t, at)).ID)
	}
	startOf := map[string]int64{}
	for i, id := range ids {
		startOf[id] = starts[i].Unix()
	}
	want := slices.Clone(ids)
	slices.SortFunc(want, func(a, b string) int {
		if startOf[a] != startOf[b] {
			return int(startOf[b] - startOf[a])
		}
		return -strings.Compare(a, b)
	})

	got, err := s.ListPumping(ctx, k.ID, wideQuery())
	if err != nil {
		t.Fatal(err)
	}
	gotIDs := make([]string, len(got))
	for i, p := range got {
		gotIDs[i] = p.ID
	}
	if !slices.Equal(gotIDs, want) {
		t.Fatalf("order = %v, want %v", gotIDs, want)
	}

	var paged []string
	q := wideQuery()
	q.Limit = 2
	for range ids {
		page, err := s.ListPumping(ctx, k.ID, q)
		if err != nil {
			t.Fatal(err)
		}
		if len(page) == 0 {
			break
		}
		for _, p := range page {
			paged = append(paged, p.ID)
		}
		last := page[len(page)-1]
		q.After = &Cursor{At: last.StartedAt.Unix(), ID: last.ID}
	}
	if !slices.Equal(paged, want) {
		t.Errorf("paged = %v, want %v", paged, want)
	}

	// The range is half-open on started_at.
	r, err := s.ListPumping(ctx, k.ID, ListQuery{From: same.Add(-time.Minute), To: same.Add(time.Minute), Limit: 10})
	if err != nil || len(r) != 4 {
		t.Errorf("[-1m, +1m) = %d rows, %v; want the 4 that start before +1m", len(r), err)
	}
}

// advancedPumping is a session whose sides are typed in unit and whose total is their sum; its side is the derived one.
func advancedPumping(t *testing.T, startedAt time.Time, unit units.Unit, amounts map[string]int64) model.PumpingSession {
	t.Helper()
	p := model.PumpingSession{StartedAt: startedAt, DurationSeconds: 900}
	var sum int64
	for _, name := range model.PumpingSideNames {
		if n, ok := amounts[name]; ok {
			p.Sides = append(p.Sides, model.PumpingSide{Side: name, Volume: mustVolume(t, unit, n)})
			sum += n
		}
	}
	p.Volume = mustVolume(t, unit, sum)
	p.Side, _ = model.DerivedPumpingSide(p.Sides)
	return p
}

func sideRows(t *testing.T, s *Store, sessionID string) int {
	t.Helper()
	var n int
	if err := s.w.QueryRowContext(ctx, `SELECT COUNT(*) FROM pumping_session_sides WHERE session_id = ?`, sessionID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestPumpingSidesRoundTrip(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")

	in := advancedPumping(t, t0, units.UnitFlOz, map[string]int64{"right": 250, "left": 150})
	if in.Side != "both" || in.Sides[0].Side != "left" {
		t.Fatalf("test input = %+v", in)
	}
	in.Sides[0], in.Sides[1] = in.Sides[1], in.Sides[0] // given right first; stored left, right
	created := mustPumping(t, s, k.ID, ana.ID, in)
	if created.Sides[0].Side != "left" || created.Sides[1].Side != "right" {
		t.Fatalf("created sides = %+v, want left, right", created.Sides)
	}
	got, err := s.Pumping(ctx, k.ID, created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("Pumping = %+v, %v; want %+v", got, err, created)
	}
	if got.Sides[0].Volume.Typed != units.UnitFlOz || got.Sides[0].Volume.FlOzHundredths != 150 || got.Sides[1].Volume.FlOzHundredths != 250 || got.Volume.FlOzHundredths != 400 {
		t.Errorf("sides = %+v total %+v", got.Sides, got.Volume)
	}
	list, err := s.ListPumping(ctx, k.ID, wideQuery())
	if err != nil || len(list) != 1 || !reflect.DeepEqual(list[0], created) {
		t.Fatalf("ListPumping = %+v, %v", list, err)
	}

	one := mustPumping(t, s, k.ID, ana.ID, advancedPumping(t, t0.Add(-time.Hour), units.UnitML, map[string]int64{"right": 70}))
	if one.Side != "right" || len(one.Sides) != 1 {
		t.Errorf("right only = %+v", one)
	}
	simple := mustPumping(t, s, k.ID, ana.ID, newPumping(t, t0.Add(-2*time.Hour)))
	if got, err := s.Pumping(ctx, k.ID, simple.ID); err != nil || len(got.Sides) != 0 {
		t.Errorf("simple session = %+v, %v; want no sides", got, err)
	}
}

func TestPumpingSidesReplacedOnUpdate(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k, other := mustKid(t, s, ana.ID, "Lía"), mustKid(t, s, ana.ID, "Other")
	p := mustPumping(t, s, k.ID, ana.ID, advancedPumping(t, t0, units.UnitML, map[string]int64{"left": 60, "right": 50}))

	next := advancedPumping(t, t0, units.UnitML, map[string]int64{"right": 80})
	next.ID = p.ID
	if err := s.UpdatePumping(ctx, k.ID, ana.ID, next, t0); err != nil {
		t.Fatal(err)
	}
	got, err := s.Pumping(ctx, k.ID, p.ID)
	if err != nil || !reflect.DeepEqual(got.Sides, next.Sides) || got.Side != "right" || got.Volume != next.Volume {
		t.Fatalf("after replace = %+v, %v; want sides %+v", got, err, next.Sides)
	}
	if n := sideRows(t, s, p.ID); n != 1 {
		t.Errorf("%d side rows, want 1", n)
	}

	foreign := advancedPumping(t, t0, units.UnitML, map[string]int64{"left": 10})
	foreign.ID = p.ID
	if err := s.UpdatePumping(ctx, other.ID, ana.ID, foreign, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("update through another kid: %v", err)
	}
	if n := sideRows(t, s, p.ID); n != 1 {
		t.Errorf("%d side rows after a refused update, want 1", n)
	}

	simple := newPumping(t, t0)
	simple.ID = p.ID
	if err := s.UpdatePumping(ctx, k.ID, ana.ID, simple, t0); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Pumping(ctx, k.ID, p.ID); err != nil || len(got.Sides) != 0 || sideRows(t, s, p.ID) != 0 {
		t.Errorf("after switching to simple = %+v, %v; %d rows", got, err, sideRows(t, s, p.ID))
	}
}

func TestSidesMustMatchSide(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	good := mustPumping(t, s, k.ID, ana.ID, advancedPumping(t, t0, units.UnitML, map[string]int64{"left": 60, "right": 50}))

	side := func(names ...string) []model.PumpingSide {
		var out []model.PumpingSide
		for _, n := range names {
			out = append(out, model.PumpingSide{Side: n, Volume: mustVolume(t, units.UnitML, 10)})
		}
		return out
	}
	for name, c := range map[string]struct {
		side  string
		sides []model.PumpingSide
	}{
		"both beside one side":     {"both", side("left")},
		"left beside the right":    {"left", side("right")},
		"right beside two sides":   {"right", side("left", "right")},
		"left beside two sides":    {"left", side("right", "left")},
		"an unknown side":          {"both", side("middle")},
		"the combined side":        {"both", side("both")},
		"a repeated side":          {"left", side("left", "left")},
		"two sides, one repeated":  {"both", side("left", "right", "left")},
		"an empty side beside one": {"", side("left")},
	} {
		bad := model.PumpingSession{StartedAt: t0, DurationSeconds: 900, Side: c.side, Sides: c.sides, Volume: mustVolume(t, units.UnitML, 10)}
		if _, err := s.CreatePumping(ctx, k.ID, ana.ID, bad, t0); !errors.Is(err, ErrInvalidSides) {
			t.Errorf("create with %s: %v, want ErrInvalidSides", name, err)
		}
		bad.ID = good.ID
		if err := s.UpdatePumping(ctx, k.ID, ana.ID, bad, t0); !errors.Is(err, ErrInvalidSides) {
			t.Errorf("update with %s: %v, want ErrInvalidSides", name, err)
		}
	}
	if got, err := s.Pumping(ctx, k.ID, good.ID); err != nil || !reflect.DeepEqual(got, good) {
		t.Errorf("a refused update changed the session: %+v, %v", got, err)
	}
	if list, err := s.ListPumping(ctx, k.ID, wideQuery()); err != nil || len(list) != 1 {
		t.Errorf("a refused create left %d sessions, %v", len(list), err)
	}
}

func TestListLoadsSidesWithOneExtraQuery(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	for i := range 100 {
		mustPumping(t, s, k.ID, ana.ID, advancedPumping(t, t0.Add(-time.Duration(i)*time.Minute), units.UnitML, map[string]int64{"left": 60, "right": 50}))
	}
	n := countReads(t, s, nil)

	if list, err := s.ListPumping(ctx, k.ID, wideQuery()); err != nil || len(list) != 100 || len(list[99].Sides) != 2 {
		t.Fatalf("ListPumping = %d rows, %v", len(list), err)
	}
	if got := n.Load(); got != 2 {
		t.Errorf("a list of 100 sessions ran %d queries, want 2 (the page and one for every side)", got)
	}

	n.Store(0)
	if items, err := s.Timeline(ctx, k.ID, []string{"pumping"}, wideQuery()); err != nil || len(items) != 100 || len(items[99].Entry.(model.PumpingSession).Sides) != 2 {
		t.Fatalf("Timeline = %d items, %v", len(items), err)
	}
	if got := n.Load(); got != 2 {
		t.Errorf("a pumping timeline of 100 sessions ran %d queries, want 2", got)
	}

	n.Store(0)
	if list, err := s.ListPumping(ctx, k.ID, ListQuery{From: t0.Add(time.Hour), To: t0.Add(2 * time.Hour), Limit: 10}); err != nil || len(list) != 0 {
		t.Fatalf("empty list = %v, %v", list, err)
	}
	if got := n.Load(); got != 1 {
		t.Errorf("an empty list ran %d queries, want 1", got)
	}
}

func TestTimelinePumpingCarriesSides(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	adv := mustPumping(t, s, k.ID, ana.ID, advancedPumping(t, t0, units.UnitML, map[string]int64{"left": 60, "right": 50}))
	mustPumping(t, s, k.ID, ana.ID, newPumping(t, t0.Add(-time.Hour)))

	items, err := s.Timeline(ctx, k.ID, []string{"pumping"}, wideQuery())
	if err != nil || len(items) != 2 {
		t.Fatalf("Timeline = %+v, %v", items, err)
	}
	if got := items[0].Entry.(model.PumpingSession); !reflect.DeepEqual(got, adv) {
		t.Errorf("advanced item = %+v, want %+v", got, adv)
	}
	if got := items[1].Entry.(model.PumpingSession); len(got.Sides) != 0 {
		t.Errorf("simple item has sides %+v", got.Sides)
	}
}

// A write that lands between the session read and the sides read must not show as a mixed pair.
func TestPumpingReadIsOneSnapshot(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	old := mustPumping(t, s, k.ID, ana.ID, advancedPumping(t, t0, units.UnitML, map[string]int64{"left": 60, "right": 50}))
	next := advancedPumping(t, t0, units.UnitML, map[string]int64{"left": 10, "right": 20})
	next.ID = old.ID

	for name, read := range map[string]func() (model.PumpingSession, error){
		"Pumping": func() (model.PumpingSession, error) { return s.Pumping(ctx, k.ID, old.ID) },
		"ListPumping": func() (model.PumpingSession, error) {
			list, err := s.ListPumping(ctx, k.ID, wideQuery())
			if err != nil || len(list) != 1 {
				t.Fatalf("ListPumping = %v, %v", list, err)
			}
			return list[0], nil
		},
	} {
		if err := s.UpdatePumping(ctx, k.ID, ana.ID, old, t0); err != nil {
			t.Fatal(err)
		}
		var wrote atomic.Bool
		countReads(t, s, func(n int64) {
			if n == 2 && wrote.CompareAndSwap(false, true) {
				if err := s.UpdatePumping(ctx, k.ID, ana.ID, next, t0); err != nil {
					t.Error(err)
				}
			}
		})
		got, err := read()
		if err != nil || !wrote.Load() {
			t.Fatalf("%s: %v, wrote between the reads: %v", name, err, wrote.Load())
		}
		var sum int64
		for _, side := range got.Sides {
			sum += side.Volume.ML
		}
		if sum != got.Volume.ML || (got.Volume.ML != old.Volume.ML && got.Volume.ML != next.Volume.ML) {
			t.Errorf("%s: total %d ml with sides %+v, a mixed read", name, got.Volume.ML, got.Sides)
		}
	}
}
