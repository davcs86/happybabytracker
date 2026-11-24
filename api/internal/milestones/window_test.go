package milestones

import (
	"errors"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
)

func zone(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func ptr(n int) *int { return &n }

func TestAgeBasis(t *testing.T) {
	utc := time.UTC
	akl := zone(t, "Pacific/Auckland")
	noon := func(date string) time.Time {
		d, err := time.Parse(time.DateOnly, date)
		if err != nil {
			t.Fatal(err)
		}
		return d.Add(12 * time.Hour)
	}
	tests := []struct {
		name      string
		birth     string
		gw, gd    *int
		loc       *time.Location
		now       time.Time
		chrono    int
		corrected *int
		basis     int
	}{
		{"term kid", "2026-01-15", nil, nil, utc, noon("2026-11-24"), 10, nil, 10},
		{"term with gestation given", "2026-01-15", ptr(40), ptr(0), utc, noon("2026-11-24"), 10, nil, 10},
		{"32 weeks at 3 months", "2026-01-01", ptr(32), ptr(0), utc, noon("2026-04-01"), 3, ptr(1), 1},
		{"32 weeks the day before the second birthday", "2024-01-01", ptr(32), ptr(0), utc, noon("2025-12-31"), 23, ptr(22), 22},
		{"32 weeks on the second birthday", "2024-01-01", ptr(32), ptr(0), utc, noon("2026-01-01"), 24, nil, 24},
		{"36w6d is corrected", "2026-01-01", ptr(36), ptr(6), utc, noon("2026-06-01"), 5, ptr(4), 4},
		{"37w0d is not", "2026-01-01", ptr(37), ptr(0), utc, noon("2026-06-01"), 5, nil, 5},
		{"before the expected term", "2026-01-01", ptr(32), ptr(0), utc, noon("2026-02-01"), 1, ptr(0), 0},
		{"born on the 31st, day short", "2026-01-31", nil, nil, utc, noon("2026-02-28"), 0, nil, 0},
		{"born on the 31st, in March", "2026-01-31", nil, nil, utc, noon("2026-03-31"), 2, nil, 2},
		{"born on the 30th", "2026-01-30", nil, nil, utc, noon("2026-03-29"), 1, nil, 1},
		{"born on 29 February, the day before the year", "2024-02-29", nil, nil, utc, noon("2025-02-28"), 11, nil, 11},
		{"born on 29 February, 1 March", "2024-02-29", nil, nil, utc, noon("2025-03-01"), 12, nil, 12},
		{"second birthday reached in Auckland, instant in UTC still before it: Auckland", "2024-01-01", ptr(32), ptr(0), akl, time.Date(2025, 12, 31, 12, 0, 0, 0, time.UTC), 24, nil, 24},
		{"second birthday reached in Auckland, instant in UTC still before it: UTC", "2024-01-01", ptr(32), ptr(0), utc, time.Date(2025, 12, 31, 12, 0, 0, 0, time.UTC), 23, ptr(22), 22},
		{"Auckland, a preterm kid", "2026-01-01", ptr(32), ptr(0), akl, noon("2026-04-01"), 3, ptr(1), 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AgeBasisFor(tt.birth, tt.gw, tt.gd, tt.loc, tt.now)
			if err != nil {
				t.Fatal(err)
			}
			if got.ChronologicalMonths != tt.chrono || got.BasisMonths != tt.basis {
				t.Errorf("chrono %d basis %d, want %d and %d", got.ChronologicalMonths, got.BasisMonths, tt.chrono, tt.basis)
			}
			switch {
			case tt.corrected == nil && got.CorrectedMonths != nil:
				t.Errorf("corrected %d, want nil", *got.CorrectedMonths)
			case tt.corrected != nil && (got.CorrectedMonths == nil || *got.CorrectedMonths != *tt.corrected):
				t.Errorf("corrected %v, want %d", got.CorrectedMonths, *tt.corrected)
			}
		})
	}
	t.Run("Auckland and UTC disagree on the same instant", func(t *testing.T) {
		now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC) // 2026-04-01 01:00 in Auckland
		a, err := AgeBasisFor("2026-01-01", nil, nil, akl, now)
		if err != nil {
			t.Fatal(err)
		}
		u, err := AgeBasisFor("2026-01-01", nil, nil, utc, now)
		if err != nil {
			t.Fatal(err)
		}
		if a.BasisMonths != 3 || u.BasisMonths != 2 {
			t.Errorf("Auckland %d, UTC %d, want 3 and 2", a.BasisMonths, u.BasisMonths)
		}
	})
	t.Run("malformed birth date", func(t *testing.T) {
		if _, err := AgeBasisFor("2026-13-01", nil, nil, utc, time.Now()); !errors.Is(err, kidtime.ErrBadDate) {
			t.Errorf("err %v, want ErrBadDate", err)
		}
	})
}

func TestPositions(t *testing.T) {
	ages := constants.MilestoneCatalogAges
	tests := []struct {
		basis, current, next int // 0 means none
	}{
		{0, 0, 2},
		{1, 0, 2},
		{2, 2, 4},
		{3, 2, 4},
		{14, 12, 15},
		{59, 48, 60},
		{60, 60, 0},
		{70, 60, 0},
	}
	for _, tt := range tests {
		got := Positions(ages, tt.basis)
		if len(got) != len(ages) {
			t.Errorf("basis %d: %d positions for %d ages", tt.basis, len(got), len(ages))
		}
		for _, a := range ages {
			want := PositionLater
			switch {
			case a == tt.current:
				want = PositionCurrent
			case a == tt.next:
				want = PositionNext
			case tt.current != 0 && a < tt.current:
				want = PositionEarlier
			}
			if got[a] != want {
				t.Errorf("basis %d age %d: %q, want %q", tt.basis, a, got[a], want)
			}
		}
	}
	t.Run("only the ages passed appear", func(t *testing.T) {
		got := Positions([]int{6, 12}, 7)
		want := map[int]Position{6: PositionCurrent, 12: PositionNext}
		if len(got) != 2 || got[6] != want[6] || got[12] != want[12] {
			t.Errorf("got %v, want %v", got, want)
		}
		got = Positions([]int{12, 24}, 3)
		if len(got) != 2 || got[12] != PositionNext || got[24] != PositionLater {
			t.Errorf("got %v", got)
		}
		if got := Positions(nil, 3); len(got) != 0 {
			t.Errorf("got %v for no ages", got)
		}
	})
}

func TestCatalogShown(t *testing.T) {
	tests := []struct {
		name, birth, today string
		want               bool
	}{
		{"4 years 11 months", "2021-06-15", "2026-05-15", true},
		{"the day before the 5th birthday", "2021-06-15", "2026-06-14", true},
		{"the 5th birthday", "2021-06-15", "2026-06-15", false},
		{"the day after", "2021-06-15", "2026-06-16", false},
		{"29 February birth, 28 February", "2020-02-29", "2025-02-28", true},
		{"29 February birth, 1 March", "2020-02-29", "2025-03-01", false},
		{"newborn", "2026-11-24", "2026-11-24", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CatalogShown(tt.birth, tt.today)
			if err != nil || got != tt.want {
				t.Errorf("got %v, %v, want %v", got, err, tt.want)
			}
		})
	}
	t.Run("a preterm kid's 5th birthday is chronological", func(t *testing.T) {
		// Correction plays no part: the signature takes no gestation.
		got, err := CatalogShown("2021-06-15", "2026-06-15")
		if err != nil || got {
			t.Errorf("got %v, %v, want false", got, err)
		}
	})
	for _, bad := range [][2]string{{"2021-6-15", "2026-06-15"}, {"2021-06-15", "tomorrow"}, {"", "2026-06-15"}, {"2021-06-15", ""}} {
		if _, err := CatalogShown(bad[0], bad[1]); !errors.Is(err, kidtime.ErrBadDate) {
			t.Errorf("%q, %q: err %v, want ErrBadDate", bad[0], bad[1], err)
		}
	}
}
