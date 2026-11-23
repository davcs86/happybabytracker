package store

import (
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

const (
	testP256dh = "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4"
	testAuth   = "BTBZMqHH6r4Tts7J_aSIgg"
)

func mustDevice(t *testing.T, s *Store, parentID, endpoint string) model.PushDevice {
	t.Helper()
	d, _, err := s.UpsertDevice(ctx, parentID, endpoint, testP256dh, testAuth, "Phone", t0)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func scalar(t *testing.T, s *Store, query string, args ...any) int {
	t.Helper()
	var n int
	if err := s.w.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func targetIDs(t *testing.T, s *Store, kidID string) []string {
	t.Helper()
	var out []PushTarget
	if err := s.inReadTx(ctx, func(q querier) error {
		var err error
		out, err = PushTargets(ctx, q, kidID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(out))
	for i, tg := range out {
		ids[i] = tg.ID
	}
	return ids
}

func TestUpsertDeviceCreatesThenUpdates(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	d, created, err := s.UpsertDevice(ctx, ana.ID, "https://push.example.com/a", testP256dh, testAuth, "Phone", t0)
	if err != nil || !created {
		t.Fatalf("first upsert: created=%v err=%v", created, err)
	}
	if d.Label != "Phone" || !d.CreatedAt.Equal(t0) || d.Kids == nil || len(d.Kids) != 0 {
		t.Fatalf("device = %+v", d)
	}
	again, created, err := s.UpsertDevice(ctx, ana.ID, "https://push.example.com/a", "BOTHER", "AUTHOTHER", "Tablet", t0.Add(time.Hour))
	if err != nil || created || again.ID != d.ID || again.Label != "Tablet" {
		t.Fatalf("second upsert: %+v created=%v err=%v", again, created, err)
	}
	var p256dh, auth string
	if err := s.w.QueryRow(`SELECT p256dh, auth FROM push_subscriptions WHERE id = ?`, d.ID).Scan(&p256dh, &auth); err != nil {
		t.Fatal(err)
	}
	if p256dh != "BOTHER" || auth != "AUTHOTHER" {
		t.Fatalf("keys not updated: %q %q", p256dh, auth)
	}
}

func TestUpsertDeviceMovesToTheNewParent(t *testing.T) {
	s := openMem(t)
	ana, ben := mustParent(t, s, "ana"), mustParent(t, s, "ben")
	kid := mustKid(t, s, ana.ID, "Mia")
	mustGrant(t, s, kid.ID, ana.ID, "ben", model.RoleCaregiver)
	d := mustDevice(t, s, ana.ID, "https://push.example.com/shared")
	if err := s.SetPushOptin(ctx, kid.ID, ana.ID, d.ID); err != nil {
		t.Fatal(err)
	}
	moved, created, err := s.UpsertDevice(ctx, ben.ID, "https://push.example.com/shared", testP256dh, testAuth, "Ben's", t0.Add(time.Hour))
	if err != nil || !created || moved.ID == d.ID || len(moved.Kids) != 0 || !moved.CreatedAt.Equal(t0.Add(time.Hour)) {
		t.Fatalf("move: %+v created=%v err=%v", moved, created, err)
	}
	if got, _ := s.Devices(ctx, ana.ID); len(got) != 0 {
		t.Fatalf("old owner still sees %+v", got)
	}
	if got, _ := s.Devices(ctx, ben.ID); len(got) != 1 || got[0].ID != moved.ID {
		t.Fatalf("new owner sees %+v", got)
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM push_kid_optins WHERE subscription_id = ?`, d.ID); n != 0 {
		t.Fatalf("%d opt-ins survived the move", n)
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM push_subscriptions WHERE id = ?`, d.ID); n != 0 {
		t.Fatalf("the old row survived the move")
	}
	if err := s.SetPushOptin(ctx, kid.ID, ben.ID, d.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("opt-in with the old id: %v", err)
	}
	if err := s.DeleteDevice(ctx, ana.ID, d.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("old owner delete: %v", err)
	}
}

func TestUpsertDeviceCap(t *testing.T) {
	s := openMem(t)
	ana, ben := mustParent(t, s, "ana"), mustParent(t, s, "ben")
	for i := range constants.PushDevicesPerParent {
		mustDevice(t, s, ana.ID, fmt.Sprintf("https://push.example.com/%d", i))
	}
	if _, _, err := s.UpsertDevice(ctx, ana.ID, "https://push.example.com/extra", testP256dh, testAuth, "x", t0); !errors.Is(err, ErrConflict) {
		t.Fatalf("eleventh device: %v", err)
	}
	if _, created, err := s.UpsertDevice(ctx, ana.ID, "https://push.example.com/0", testP256dh, testAuth, "y", t0); err != nil || created {
		t.Fatalf("re-send at the cap: created=%v err=%v", created, err)
	}
	mustDevice(t, s, ben.ID, "https://push.example.com/ben")
	if _, _, err := s.UpsertDevice(ctx, ana.ID, "https://push.example.com/ben", testP256dh, testAuth, "z", t0); !errors.Is(err, ErrConflict) {
		t.Fatalf("move into a full parent: %v", err)
	}
}

func TestUpsertDeviceRefusesADisabledParent(t *testing.T) {
	s := openMem(t)
	ana, ben := mustParent(t, s, "ana"), mustParent(t, s, "ben")
	mustKid(t, s, ben.ID, "Mia")
	if err := s.SetParentDisabled(ctx, ana.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.UpsertDevice(ctx, ana.ID, "https://push.example.com/a", testP256dh, testAuth, "x", t0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("disabled parent: %v", err)
	}
}

func TestRenameAndDeleteDeviceAreOwnerOnly(t *testing.T) {
	s := openMem(t)
	ana, ben := mustParent(t, s, "ana"), mustParent(t, s, "ben")
	d := mustDevice(t, s, ana.ID, "https://push.example.com/a")
	if _, err := s.RenameDevice(ctx, ben.ID, d.ID, "mine", t0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("rename by another parent: %v", err)
	}
	if err := s.DeleteDevice(ctx, ben.ID, d.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete by another parent: %v", err)
	}
	if err := s.DeleteDevice(ctx, ana.ID, model.NewID()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete a missing device: %v", err)
	}
	got, err := s.RenameDevice(ctx, ana.ID, d.ID, "Laptop", t0)
	if err != nil || got.Label != "Laptop" {
		t.Fatalf("rename: %+v %v", got, err)
	}
	if err := s.DeleteDevice(ctx, ana.ID, d.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.Devices(ctx, ana.ID); len(list) != 0 {
		t.Fatalf("after delete: %+v", list)
	}
}

func TestPushOptins(t *testing.T) {
	s := openMem(t)
	ana, ben := mustParent(t, s, "ana"), mustParent(t, s, "ben")
	kid := mustKid(t, s, ana.ID, "Mia")
	other := mustKid(t, s, ben.ID, "Leo")
	d := mustDevice(t, s, ana.ID, "https://push.example.com/a")
	bd := mustDevice(t, s, ben.ID, "https://push.example.com/b")
	if err := s.SetPushOptin(ctx, other.ID, ana.ID, d.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("no grant: %v", err)
	}
	if err := s.SetPushOptin(ctx, kid.ID, ana.ID, bd.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another parent's device: %v", err)
	}
	if err := s.ClearPushOptin(ctx, kid.ID, ana.ID, bd.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("clear another parent's device: %v", err)
	}
	for range 2 {
		if err := s.SetPushOptin(ctx, kid.ID, ana.ID, d.ID); err != nil {
			t.Fatal(err)
		}
	}
	list, err := s.Devices(ctx, ana.ID)
	if err != nil || len(list) != 1 || !slices.Equal(list[0].Kids, []string{kid.ID}) {
		t.Fatalf("devices: %+v %v", list, err)
	}
	if got := targetIDs(t, s, kid.ID); !slices.Equal(got, []string{d.ID}) {
		t.Fatalf("targets = %v", got)
	}
	for range 2 {
		if err := s.ClearPushOptin(ctx, kid.ID, ana.ID, d.ID); err != nil {
			t.Fatal(err)
		}
	}
	if got := targetIDs(t, s, kid.ID); len(got) != 0 {
		t.Fatalf("targets after clear = %v", got)
	}
}

func TestPushTargetsExcludeRevokedDisabledAndNotOptedIn(t *testing.T) {
	s := openMem(t)
	ana, ben, cid, dan := mustParent(t, s, "ana"), mustParent(t, s, "ben"), mustParent(t, s, "cid"), mustParent(t, s, "dan")
	kid := mustKid(t, s, ana.ID, "Mia")
	for _, u := range []string{"ben", "cid", "dan"} {
		mustGrant(t, s, kid.ID, ana.ID, u, model.RoleCaregiver)
	}
	devs := map[string]model.PushDevice{}
	for _, p := range []model.Parent{ana, ben, cid, dan} {
		devs[p.Username] = mustDevice(t, s, p.ID, "https://push.example.com/"+p.Username)
		if p.Username != "dan" {
			if err := s.SetPushOptin(ctx, kid.ID, p.ID, devs[p.Username].ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := s.RemoveAccess(ctx, kid.ID, ana.ID, ben.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.w.Exec(`UPDATE parents SET disabled = 1 WHERE id = ?`, cid.ID); err != nil {
		t.Fatal(err)
	}
	got := targetIDs(t, s, kid.ID)
	if !slices.Equal(got, []string{devs["ana"].ID}) {
		t.Fatalf("targets = %v, want only ana's %s", got, devs["ana"].ID)
	}
	all, err := func() ([]PushTarget, error) {
		var out []PushTarget
		err := s.inReadTx(ctx, func(q querier) error {
			var err error
			out, err = PushTargets(ctx, q, kid.ID)
			return err
		})
		return out, err
	}()
	if err != nil || all[0].ParentID != ana.ID || all[0].Endpoint != "https://push.example.com/ana" || all[0].P256dh != testP256dh || all[0].Auth != testAuth {
		t.Fatalf("target = %+v %v", all, err)
	}
}

func TestRevokeRemovesOptins(t *testing.T) {
	s := openMem(t)
	ana, ben := mustParent(t, s, "ana"), mustParent(t, s, "ben")
	kid, other := mustKid(t, s, ana.ID, "Mia"), mustKid(t, s, ana.ID, "Leo")
	mustGrant(t, s, kid.ID, ana.ID, "ben", model.RoleCaregiver)
	mustGrant(t, s, other.ID, ana.ID, "ben", model.RoleCaregiver)
	bd, ad := mustDevice(t, s, ben.ID, "https://push.example.com/b"), mustDevice(t, s, ana.ID, "https://push.example.com/a")
	for _, o := range []struct{ kid, parent, dev string }{{kid.ID, ben.ID, bd.ID}, {other.ID, ben.ID, bd.ID}, {kid.ID, ana.ID, ad.ID}} {
		if err := s.SetPushOptin(ctx, o.kid, o.parent, o.dev); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RemoveAccess(ctx, kid.ID, ana.ID, ben.ID); err != nil {
		t.Fatal(err)
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM push_kid_optins WHERE subscription_id = ? AND kid_id = ?`, bd.ID, kid.ID); n != 0 {
		t.Fatalf("ben's opt-in for the revoked kid survived")
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM push_kid_optins WHERE subscription_id = ? AND kid_id = ?`, bd.ID, other.ID); n != 1 {
		t.Fatalf("ben's opt-in for the other kid was removed")
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM push_kid_optins WHERE subscription_id = ?`, ad.ID); n != 1 {
		t.Fatalf("ana's opt-in was removed")
	}
}

func TestLeaveRemovesOptins(t *testing.T) {
	s := openMem(t)
	ana, ben := mustParent(t, s, "ana"), mustParent(t, s, "ben")
	kid := mustKid(t, s, ana.ID, "Mia")
	mustGrant(t, s, kid.ID, ana.ID, "ben", model.RoleCaregiver)
	bd := mustDevice(t, s, ben.ID, "https://push.example.com/b")
	if err := s.SetPushOptin(ctx, kid.ID, ben.ID, bd.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveAccess(ctx, kid.ID, ben.ID, ben.ID); err != nil {
		t.Fatal(err)
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM push_kid_optins WHERE subscription_id = ?`, bd.ID); n != 0 {
		t.Fatalf("opt-in survived leaving")
	}
	if list, _ := s.Devices(ctx, ben.ID); len(list) != 1 {
		t.Fatalf("leaving removed the device: %+v", list)
	}
}

func TestDisableRemovesDevices(t *testing.T) {
	s := openMem(t)
	ana, ben := mustParent(t, s, "ana"), mustParent(t, s, "ben")
	kid := mustKid(t, s, ana.ID, "Mia")
	mustGrant(t, s, kid.ID, ana.ID, "ben", model.RoleCaregiver)
	bd := mustDevice(t, s, ben.ID, "https://push.example.com/b")
	mustDevice(t, s, ana.ID, "https://push.example.com/a")
	if err := s.SetPushOptin(ctx, kid.ID, ben.ID, bd.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.w.Exec(`INSERT INTO ntfy_topics(parent_id, topic, created_at) VALUES (?, 'hbt-bentopic', 0)`, ben.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.w.Exec(`INSERT INTO ntfy_kid_optins(parent_id, kid_id) VALUES (?, ?)`, ben.ID, kid.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.SetParentDisabled(ctx, ben.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	for q, want := range map[string]int{
		`SELECT COUNT(*) FROM push_subscriptions WHERE parent_id = ?`:                                                        0,
		`SELECT COUNT(*) FROM push_kid_optins o JOIN push_subscriptions s ON s.id = o.subscription_id WHERE s.parent_id = ?`: 0,
		`SELECT COUNT(*) FROM ntfy_topics WHERE parent_id = ?`:                                                               0,
		`SELECT COUNT(*) FROM ntfy_kid_optins WHERE parent_id = ?`:                                                           0,
	} {
		if n := scalar(t, s, q, ben.ID); n != want {
			t.Errorf("%s: %d", q, n)
		}
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM push_kid_optins`); n != 0 {
		t.Errorf("opt-ins left: %d", n)
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM push_subscriptions WHERE parent_id = ?`, ana.ID); n != 1 {
		t.Errorf("ana's device removed")
	}
}

func TestDisableRefusedKeepsDevices(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	mustKid(t, s, ana.ID, "Mia")
	mustDevice(t, s, ana.ID, "https://push.example.com/a")
	if err := s.SetParentDisabled(ctx, ana.ID, true, t0); !errors.Is(err, ErrLastGuardian) {
		t.Fatalf("disable: %v", err)
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM push_subscriptions`); n != 1 {
		t.Fatalf("refused disable deleted the device")
	}
}

func TestDeleteKidCascadesOptins(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	kid := mustKid(t, s, ana.ID, "Mia")
	d := mustDevice(t, s, ana.ID, "https://push.example.com/a")
	if err := s.SetPushOptin(ctx, kid.ID, ana.ID, d.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteKid(ctx, kid.ID, ana.ID); err != nil {
		t.Fatal(err)
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM push_kid_optins`); n != 0 {
		t.Fatalf("opt-in survived the kid")
	}
}
