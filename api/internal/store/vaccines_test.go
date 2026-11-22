package store

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

func newAdministration(on string, antigens ...string) model.Administration {
	return model.Administration{GivenOn: on, Antigens: antigens}
}

func mustAdministration(t *testing.T, s *Store, kidID, actorID string, a model.Administration) model.Administration {
	t.Helper()
	got, err := s.CreateAdministration(ctx, kidID, actorID, a, t0)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// doseIDs maps antigen to row id for one administration, read on the writer.
func doseIDs(t *testing.T, s *Store, administration string) map[string]string {
	t.Helper()
	rows, err := s.w.Query(`SELECT antigen, id FROM vaccine_doses WHERE administration = ?`, administration)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var antigen, id string
		if err := rows.Scan(&antigen, &id); err != nil {
			t.Fatal(err)
		}
		out[antigen] = id
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func countRows(t *testing.T, s *Store, table, kidID string) int {
	t.Helper()
	var n int
	if err := s.w.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE kid_id = ?`, kidID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestAdministrationCRUD(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)

	in := model.Administration{GivenOn: "2026-03-01", Antigens: []string{"polio", "dtap", "hib"},
		Lot: strp("AB12"), Product: strp("Hexyon"), Note: strp("left thigh")}
	created, err := s.CreateAdministration(ctx, k.ID, bo.ID, in, t0)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.KidID != k.ID || created.CreatedBy != bo.ID || created.UpdatedBy != bo.ID ||
		!created.CreatedAt.Equal(t0) || !created.UpdatedAt.Equal(t0) || created.GivenOn != "2026-03-01" ||
		!slices.Equal(created.Antigens, []string{"dtap", "hib", "polio"}) ||
		*created.Lot != "AB12" || *created.Product != "Hexyon" || *created.Note != "left thigh" {
		t.Fatalf("created = %+v", created)
	}
	ids := doseIDs(t, s, created.ID)
	if len(ids) != 3 || countRows(t, s, "vaccine_doses", k.ID) != 3 {
		t.Fatalf("rows = %v, want one per antigen", ids)
	}
	for _, id := range ids {
		if id == created.ID {
			t.Errorf("a row id equals the administration id %s", id)
		}
	}
	got, err := s.Administration(ctx, k.ID, created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("read = %+v, %v; want %+v", got, err, created)
	}

	later := t0.Add(time.Hour)
	upd := created
	upd.GivenOn, upd.Antigens, upd.Lot, upd.Product, upd.Note = "2026-03-02", []string{"hepb", "dtap", "polio"}, nil, strp("Infanrix"), nil
	if err := s.UpdateAdministration(ctx, k.ID, ana.ID, upd, later); err != nil {
		t.Fatal(err)
	}
	after := doseIDs(t, s, created.ID)
	if len(after) != 3 || after["dtap"] != ids["dtap"] || after["polio"] != ids["polio"] || after["hib"] != "" ||
		after["hepb"] == "" || after["hepb"] == ids["hib"] {
		t.Fatalf("rows after update = %v (before %v); want kept ids stable, hib gone, hepb new", after, ids)
	}
	got, err = s.Administration(ctx, k.ID, created.ID)
	want := model.Administration{ID: created.ID, KidID: k.ID, GivenOn: "2026-03-02", Antigens: []string{"dtap", "hepb", "polio"},
		Product: strp("Infanrix"), CreatedBy: bo.ID, UpdatedBy: ana.ID, CreatedAt: t0, UpdatedAt: later}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("after update = %+v, %v; want %+v", got, err, want)
	}
	doses, err := s.Doses(ctx, k.ID)
	if err != nil || len(doses) != 3 {
		t.Fatalf("doses = %+v, %v", doses, err)
	}
	for _, d := range doses {
		if d.Administration != created.ID || d.KidID != k.ID || d.GivenOn != "2026-03-02" || d.Lot != nil ||
			d.Product == nil || *d.Product != "Infanrix" || d.Note != nil || d.CreatedBy != bo.ID || d.UpdatedBy != ana.ID ||
			!d.CreatedAt.Equal(t0) || !d.UpdatedAt.Equal(later) || d.ID != after[d.Antigen] {
			t.Errorf("dose = %+v", d)
		}
	}

	if err := s.DeleteAdministration(ctx, k.ID, bo.ID, created.ID); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, s, "vaccine_doses", k.ID); n != 0 {
		t.Errorf("%d rows after delete", n)
	}
	if _, err := s.Administration(ctx, k.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("read after delete: %v", err)
	}
	if err := s.DeleteAdministration(ctx, k.ID, bo.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: %v", err)
	}
	if err := s.UpdateAdministration(ctx, k.ID, bo.ID, upd, later); !errors.Is(err, ErrNotFound) {
		t.Errorf("update after delete: %v", err)
	}
}

func TestAdministrationAntigenSet(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	a := mustAdministration(t, s, k.ID, ana.ID, newAdministration("2026-03-01", "mmr"))
	for name, antigens := range map[string][]string{"empty": {}, "nil": nil, "repeated": {"dtap", "hib", "dtap"}} {
		if _, err := s.CreateAdministration(ctx, k.ID, ana.ID, newAdministration("2026-03-01", antigens...), t0); !errors.Is(err, ErrInvalidAntigens) {
			t.Errorf("create %s: %v", name, err)
		}
		bad := a
		bad.Antigens = antigens
		if err := s.UpdateAdministration(ctx, k.ID, ana.ID, bad, t0); !errors.Is(err, ErrInvalidAntigens) {
			t.Errorf("update %s: %v", name, err)
		}
	}
	if n := countRows(t, s, "vaccine_doses", k.ID); n != 1 {
		t.Errorf("%d rows, want the one seeded", n)
	}
}

func TestListAdministrationsNewestFirst(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k, other := mustKid(t, s, ana.ID, "Lía"), mustKid(t, s, ana.ID, "Noa")
	var ids []string
	for _, on := range []string{"2026-02-01", "2026-02-01", "2026-03-01", "2026-01-15"} {
		ids = append(ids, mustAdministration(t, s, k.ID, ana.ID, newAdministration(on, "rv", "pcv")).ID)
	}
	mustAdministration(t, s, other.ID, ana.ID, newAdministration("2026-02-01", "mmr"))
	ties := slices.Clone(ids[:2])
	slices.Sort(ties)
	slices.Reverse(ties)
	want := append(append([]string{ids[2]}, ties...), ids[3])

	list, err := s.ListAdministrations(ctx, k.ID)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(list))
	for i, a := range list {
		got[i] = a.ID
		if !slices.Equal(a.Antigens, []string{"pcv", "rv"}) {
			t.Errorf("%s antigens = %v", a.ID, a.Antigens)
		}
	}
	if !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	if empty, err := s.ListAdministrations(ctx, mustKid(t, s, ana.ID, "Empty").ID); err != nil || empty == nil || len(empty) != 0 {
		t.Errorf("empty list = %#v, %v", empty, err)
	}
	if doses, err := s.Doses(ctx, k.ID); err != nil || len(doses) != 8 || doses[0].GivenOn != "2026-01-15" || doses[7].GivenOn != "2026-03-01" {
		t.Errorf("doses = %+v, %v; want 8, oldest first", doses, err)
	}
}

func TestAdministrationBelongsToItsKid(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	mine, other := mustKid(t, s, ana.ID, "Mine"), mustKid(t, s, ana.ID, "Other")
	a := mustAdministration(t, s, mine.ID, ana.ID, newAdministration("2026-02-01", "dtap", "hib"))
	before := doseIDs(t, s, a.ID)

	if _, err := s.Administration(ctx, other.ID, a.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("read through another kid: %v", err)
	}
	upd := a
	upd.Antigens = []string{"mmr"}
	if err := s.UpdateAdministration(ctx, other.ID, ana.ID, upd, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("update through another kid: %v", err)
	}
	if err := s.DeleteAdministration(ctx, other.ID, ana.ID, a.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete through another kid: %v", err)
	}
	if n := countRows(t, s, "vaccine_doses", other.ID); n != 0 {
		t.Errorf("other kid holds %d rows", n)
	}
	if got, err := s.Administration(ctx, mine.ID, a.ID); err != nil || !reflect.DeepEqual(got, a) {
		t.Errorf("administration changed: %+v, %v", got, err)
	}
	if after := doseIDs(t, s, a.ID); !maps.Equal(after, before) {
		t.Errorf("rows = %v, want %v", after, before)
	}
}

func TestVaccineWriteByDisabledActor(t *testing.T) {
	s := openMem(t)
	ana, bo, cy, di := mustParent(t, s, "ana"), mustParent(t, s, "bo"), mustParent(t, s, "cy"), mustParent(t, s, "di")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)
	mustGrant(t, s, k.ID, ana.ID, "di", model.RoleGuardian)
	a := mustAdministration(t, s, k.ID, ana.ID, newAdministration("2026-02-01", "dtap"))
	if err := s.SetDismissal(ctx, k.ID, ana.ID, "us", "hepb-1", "not_needed", t0); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{bo.ID, di.ID} {
		if err := s.SetParentDisabled(ctx, p, true, t0); err != nil {
			t.Fatal(err)
		}
	}

	for name, actor := range map[string]string{"disabled caregiver": bo.ID, "disabled guardian": di.ID, "no grant": cy.ID} {
		if _, err := s.CreateAdministration(ctx, k.ID, actor, newAdministration("2026-02-02", "mmr"), t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s create: %v", name, err)
		}
		upd := a
		upd.Antigens = []string{"mmr"}
		if err := s.UpdateAdministration(ctx, k.ID, actor, upd, t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s update: %v", name, err)
		}
		if err := s.DeleteAdministration(ctx, k.ID, actor, a.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s delete: %v", name, err)
		}
		if err := s.SetDismissal(ctx, k.ID, actor, "us", "hepb-1", "given_unknown", t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s set dismissal: %v", name, err)
		}
		if err := s.ClearDismissal(ctx, k.ID, actor, "us", "hepb-1"); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s clear dismissal: %v", name, err)
		}
		if err := s.DismissLines(ctx, k.ID, actor, "us", []string{"rv-1"}, "not_needed", t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s dismiss lines: %v", name, err)
		}
	}
	if list, err := s.ListAdministrations(ctx, k.ID); err != nil || len(list) != 1 || !reflect.DeepEqual(list[0], a) {
		t.Errorf("list = %+v, %v; want the one seeded administration unchanged", list, err)
	}
	if d, err := s.Dismissals(ctx, k.ID, "us"); err != nil || !maps.Equal(d, map[string]string{"hepb-1": "not_needed"}) {
		t.Errorf("dismissals = %v, %v", d, err)
	}
}

func TestAdministrationCap(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k, other := mustKid(t, s, ana.ID, "Lía"), mustKid(t, s, ana.ID, "Noa")
	tx, err := s.w.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for range constants.MaxVaccineAdministrationsPerKid - 1 {
		admin := model.NewID()
		for _, antigen := range []string{"dtap", "hib"} {
			if _, err := tx.Exec(insertDose, model.NewID(), k.ID, admin, "2026-02-01", antigen, ana.ID, ana.ID); err != nil {
				tx.Rollback()
				t.Fatal(err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	last := mustAdministration(t, s, k.ID, ana.ID, newAdministration("2026-02-02", "mmr"))
	if _, err := s.CreateAdministration(ctx, k.ID, ana.ID, newAdministration("2026-02-03", "mmr"), t0); !errors.Is(err, ErrConflict) {
		t.Fatalf("administration %d: %v, want ErrConflict", constants.MaxVaccineAdministrationsPerKid+1, err)
	}
	if list, err := s.ListAdministrations(ctx, k.ID); err != nil || len(list) != constants.MaxVaccineAdministrationsPerKid {
		t.Errorf("list = %d administrations, %v; want %d", len(list), err, constants.MaxVaccineAdministrationsPerKid)
	}
	upd := last
	upd.Antigens = []string{"mmr", "var"}
	if err := s.UpdateAdministration(ctx, k.ID, ana.ID, upd, t0); err != nil {
		t.Errorf("update at the cap: %v", err)
	}
	mustAdministration(t, s, other.ID, ana.ID, newAdministration("2026-02-02", "mmr")) // the cap is per kid
	if err := s.DeleteAdministration(ctx, k.ID, ana.ID, last.ID); err != nil {
		t.Fatal(err)
	}
	mustAdministration(t, s, k.ID, ana.ID, newAdministration("2026-02-04", "mmr"))
}

func TestDismissalsPerSchema(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k, other := mustKid(t, s, ana.ID, "Lía"), mustKid(t, s, ana.ID, "Noa")
	if err := s.SetDismissal(ctx, k.ID, ana.ID, "us", "hepb-1", "not_needed", t0); err != nil {
		t.Fatal(err)
	}
	if err := s.SetDismissal(ctx, k.ID, ana.ID, "us", "rv-1", "given_unknown", t0); err != nil {
		t.Fatal(err)
	}
	if err := s.SetDismissal(ctx, k.ID, ana.ID, "uk", "mmr-1", "not_needed", t0); err != nil {
		t.Fatal(err)
	}
	if err := s.SetDismissal(ctx, other.ID, ana.ID, "us", "dtap-1", "not_needed", t0); err != nil {
		t.Fatal(err)
	}
	if err := s.SetDismissal(ctx, k.ID, ana.ID, "us", "hepb-1", "given_unknown", t0.Add(time.Hour)); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	check := func(schema string, want map[string]string) {
		t.Helper()
		if got, err := s.Dismissals(ctx, k.ID, schema); err != nil || got == nil || !maps.Equal(got, want) {
			t.Errorf("%s dismissals = %#v, %v; want %v", schema, got, err, want)
		}
	}
	check("us", map[string]string{"hepb-1": "given_unknown", "rv-1": "given_unknown"})
	check("uk", map[string]string{"mmr-1": "not_needed"})
	check("mx", map[string]string{})

	if err := s.ClearDismissal(ctx, k.ID, ana.ID, "us", "rv-1"); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearDismissal(ctx, k.ID, ana.ID, "us", "rv-1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("clearing an absent dismissal: %v", err)
	}
	if err := s.ClearDismissal(ctx, k.ID, ana.ID, "us", "mmr-1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("clearing another schema's line: %v", err)
	}
	check("us", map[string]string{"hepb-1": "given_unknown"})
	check("uk", map[string]string{"mmr-1": "not_needed"})

	if err := s.DismissLines(ctx, k.ID, ana.ID, "us", []string{"hepb-1", "rv-1", "pcv-1"}, "not_needed", t0); err != nil {
		t.Fatal(err)
	}
	check("us", map[string]string{"hepb-1": "given_unknown", "rv-1": "not_needed", "pcv-1": "not_needed"})
	if err := s.SetDismissal(ctx, k.ID, ana.ID, "us", "x-1", "forgot", t0); err == nil {
		t.Error("an unknown reason was accepted")
	}
}

func TestDismissLinesIsAtomic(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	if _, err := s.w.Exec(`CREATE TEMP TRIGGER fail_line BEFORE INSERT ON vaccine_dismissals WHEN NEW.line = 'bad'
		BEGIN SELECT RAISE(ABORT, 'injected'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.DismissLines(ctx, k.ID, ana.ID, "us", []string{"hepb-1", "bad", "rv-1"}, "not_needed", t0); err == nil ||
		!strings.Contains(err.Error(), "injected") {
		t.Fatalf("a failing line: %v, want the injected failure", err)
	}
	if err := s.DismissLines(ctx, k.ID, ana.ID, "us", []string{"hepb-1", "rv-1"}, "maybe", t0); err == nil {
		t.Fatal("an unknown reason was accepted")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := s.DismissLines(cancelled, k.ID, ana.ID, "us", []string{"hepb-1", "rv-1"}, "not_needed", t0); err == nil {
		t.Fatal("a cancelled context wrote")
	}
	if n := countRows(t, s, "vaccine_dismissals", k.ID); n != 0 {
		t.Errorf("%d dismissals written by failed calls", n)
	}
}

func TestCaregiverCannotDismiss(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)
	if err := s.SetDismissal(ctx, k.ID, ana.ID, "us", "hepb-1", "not_needed", t0); err != nil {
		t.Fatal(err)
	}
	if err := s.SetDismissal(ctx, k.ID, bo.ID, "us", "rv-1", "not_needed", t0); !errors.Is(err, ErrForbidden) {
		t.Errorf("set: %v", err)
	}
	if err := s.ClearDismissal(ctx, k.ID, bo.ID, "us", "hepb-1"); !errors.Is(err, ErrForbidden) {
		t.Errorf("clear: %v", err)
	}
	if err := s.DismissLines(ctx, k.ID, bo.ID, "us", []string{"rv-1"}, "not_needed", t0); !errors.Is(err, ErrForbidden) {
		t.Errorf("dismiss lines: %v", err)
	}
	if d, err := s.Dismissals(ctx, k.ID, "us"); err != nil || !maps.Equal(d, map[string]string{"hepb-1": "not_needed"}) {
		t.Errorf("dismissals = %v, %v", d, err)
	}
	mustAdministration(t, s, k.ID, bo.ID, newAdministration("2026-02-01", "dtap")) // doses are a caregiver's to record
}

func TestVaccineSchemaRoundTrip(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	in := newKid("Lía")
	in.VaccineSchema, in.GrowthSource, in.GestationWeeks = strp("uk"), "cdc", intp(36)
	k, err := s.CreateKid(ctx, in, ana.ID, t0)
	if err != nil {
		t.Fatal(err)
	}
	check := func(want *string) {
		t.Helper()
		got, err := s.Kid(ctx, k.ID)
		if err != nil || !reflect.DeepEqual(got.VaccineSchema, want) || got.GrowthSource != "cdc" || got.TimeZone != "Europe/Madrid" ||
			got.GestationWeeks == nil || *got.GestationWeeks != 36 {
			t.Fatalf("kid = %+v, %v; want vaccine schema %v", got, err, want)
		}
		kids, err := s.KidsFor(ctx, ana.ID)
		if err != nil || len(kids) != 1 || !reflect.DeepEqual(kids[0].VaccineSchema, want) || kids[0].Role != model.RoleGuardian {
			t.Fatalf("KidsFor = %+v, %v", kids, err)
		}
	}
	check(strp("uk"))
	k.VaccineSchema = nil
	if err := s.UpdateKid(ctx, k, ana.ID, t0); err != nil {
		t.Fatal(err)
	}
	check(nil)
	k.VaccineSchema = strp("mx")
	if err := s.UpdateKid(ctx, k, ana.ID, t0); err != nil {
		t.Fatal(err)
	}
	check(strp("mx"))
	if plain, err := s.Kid(ctx, mustKid(t, s, ana.ID, "Noa").ID); err != nil || plain.VaccineSchema != nil {
		t.Errorf("a new kid = %+v, %v; want no vaccine schema", plain, err)
	}
}
