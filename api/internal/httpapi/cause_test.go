package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/store"
)

// rawDB opens a second connection to the kit's database file, for faults the store API cannot cause.
func rawDB(t *testing.T, dir string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "happybabytracker.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func fileKit(t *testing.T) (*kit, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return newKitWith(t, s), rawDB(t, dir)
}

// onlyLine parses the single log line a request must have written.
func onlyLine(t *testing.T, k *kit) map[string]any {
	t.Helper()
	logs := strings.TrimSpace(k.Log.String())
	if strings.Contains(logs, "\n") {
		t.Fatalf("more than one log line:\n%s", logs)
	}
	var line map[string]any
	if err := json.Unmarshal([]byte(logs), &line); err != nil {
		t.Fatalf("log %q: %v", logs, err)
	}
	return line
}

func TestCauseOfSQLiteErrors(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE t (v TEXT UNIQUE)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO t VALUES ('planted-secret-value')`); err != nil {
		t.Fatal(err)
	}
	_, unique := db.Exec(`INSERT INTO t VALUES ('planted-secret-value')`)
	_, other := db.Exec(`SELECT * FROM missing_table`)

	if got := storeCause(unique); got != "SQLITE_CONSTRAINT(2067)" {
		t.Errorf("unique violation: %q", got)
	}
	if got := storeCause(fmt.Errorf("wrapped: %w", unique)); got != "SQLITE_CONSTRAINT(2067)" {
		t.Errorf("wrapped unique violation: %q", got)
	}
	if got := storeCause(other); got != "sqlite(1)" {
		t.Errorf("code outside the map: %q, want sqlite(1)", got)
	}
	for _, leak := range []string{"planted-secret-value", "missing_table", "UNIQUE", "t.v"} {
		if got := storeCause(unique) + storeCause(other); strings.Contains(got, leak) {
			t.Errorf("cause leaks %q: %q", leak, got)
		}
	}
}

func TestCauseNamesTheFixedSQLiteCodes(t *testing.T) {
	cases := map[int]string{
		5: "SQLITE_BUSY(5)", 6: "SQLITE_LOCKED(6)", 8: "SQLITE_READONLY(8)", 10: "SQLITE_IOERR(10)",
		13: "SQLITE_FULL(13)", 14: "SQLITE_CANTOPEN(14)", 11: "SQLITE_CORRUPT(11)", 19: "SQLITE_CONSTRAINT(19)",
		// An extended code keeps its number and takes its primary code's name.
		776: "SQLITE_READONLY(776)", 1811: "SQLITE_CONSTRAINT(1811)",
		1: "sqlite(1)", 9999: "sqlite(9999)",
	}
	for code, want := range cases {
		if got := sqliteCause(code); got != want {
			t.Errorf("sqliteCause(%d) = %q, want %q", code, got, want)
		}
	}
}

func TestCauseOfOtherErrorsIsTheTypeOnly(t *testing.T) {
	err := errors.New("password=hunter2-planted-secret at /data/x.db")
	got := storeCause(err)
	if got != "other:*errors.errorString" {
		t.Errorf("cause = %q", got)
	}
	if strings.Contains(got, "hunter2") || strings.Contains(got, "/data") {
		t.Errorf("cause leaks the error text: %q", got)
	}
	if got := storeCause(fmt.Errorf("ctx: %w", err)); got != "other:*fmt.wrapError" {
		t.Errorf("wrapped: %q", got)
	}
	if got := storeCause(nil); got != "" {
		t.Errorf("nil error: %q, want empty", got)
	}
}

func TestStoreFailureLogsACauseOnTheSingleLine(t *testing.T) {
	k := newKit(t)
	const user, pass = "zelda-unique-731", "wrong-password-unique-5521"
	k.addParent(user)
	token := k.asParent(user)
	if err := k.Store.Close(); err != nil {
		t.Fatal(err)
	}

	for name, rec := range map[string]func() int{
		"session lookup": func() int { return k.do(http.MethodGet, "/v1/session", token, nil).Code },
		"login":          func() int { return k.login("203.0.113.77", user, pass).Code },
	} {
		k.Log.Reset()
		if code := rec(); code != http.StatusInternalServerError {
			t.Fatalf("%s: %d, want 500", name, code)
		}
		line := onlyLine(t, k)
		if line["status"] != float64(500) || line["cause"] != "other:*errors.errorString" {
			t.Errorf("%s: log line = %v", name, line)
		}
		for _, secret := range []string{user, pass, token, "203.0.113.77", testInternalToken, "closed", "sql:"} {
			if strings.Contains(k.Log.String(), secret) {
				t.Errorf("%s: log contains %q: %s", name, secret, k.Log.String())
			}
		}
	}
	wantErr(t, k.do(http.MethodGet, "/v1/session", token, nil), http.StatusInternalServerError, "internal")
}

func TestSQLiteFailureLogsItsCode(t *testing.T) {
	k, db := fileKit(t)
	k.addParent("ana")
	if _, err := db.Exec(`CREATE TRIGGER planted BEFORE INSERT ON sessions BEGIN SELECT RAISE(ABORT, 'planted-trigger-text'); END`); err != nil {
		t.Fatal(err)
	}
	rec := k.login("203.0.113.5", "ana", testParentPass)
	wantErr(t, rec, http.StatusInternalServerError, "internal")
	line := onlyLine(t, k)
	if line["cause"] != "SQLITE_CONSTRAINT(1811)" {
		t.Errorf("log line = %v", line)
	}
	if strings.Contains(k.Log.String(), "planted-trigger-text") || strings.Contains(rec.Body.String(), "planted-trigger-text") {
		t.Errorf("the SQLite message leaked:\nlog %s\nbody %s", k.Log.String(), rec.Body.String())
	}
}

func TestFailedSlideIsLoggedAndTheRequestServed(t *testing.T) {
	k, db := fileKit(t)
	token := k.asParent("ana")
	if _, err := db.Exec(`CREATE TRIGGER planted BEFORE UPDATE ON sessions BEGIN SELECT RAISE(ABORT, 'planted-trigger-text'); END`); err != nil {
		t.Fatal(err)
	}
	k.now = t0.Add(10 * 24 * time.Hour) // ten days in: a slide is due

	rec := k.do(http.MethodGet, "/v1/session", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("a failed slide must not fail the request: %d %q", rec.Code, rec.Body.String())
	}
	if got := bodyMap(t, rec)["expires_at"]; got != stamp(t0.Add(30*24*time.Hour)) {
		t.Errorf("expires_at = %v, want the original expiry", got)
	}
	line := onlyLine(t, k)
	if line["status"] != float64(200) || line["slide"] != "failed" || line["cause"] != "SQLITE_CONSTRAINT(1811)" {
		t.Errorf("log line = %v", line)
	}
	if strings.Contains(k.Log.String(), "planted-trigger-text") || strings.Contains(k.Log.String(), token) {
		t.Errorf("log leaks: %s", k.Log.String())
	}

	// The session still expires on its original schedule.
	k.Log.Reset()
	k.now = t0.Add(30 * 24 * time.Hour)
	wantErr(t, k.do(http.MethodGet, "/v1/session", token, nil), http.StatusUnauthorized, "no_session")
	if _, has := onlyLine(t, k)["cause"]; has {
		t.Error("a refused session logged a cause")
	}
}

func TestHealthyRequestsLogNoCause(t *testing.T) {
	k := newKit(t)
	k.do(http.MethodGet, "/v1/session", k.asParent("ana"), nil)
	line := onlyLine(t, k)
	if _, has := line["cause"]; has {
		t.Errorf("log line = %v", line)
	}
	if _, has := line["slide"]; has {
		t.Errorf("log line = %v", line)
	}
}
