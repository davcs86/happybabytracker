package vaccines

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/kidtime"
)

func age(n int, u Unit) Age    { return Age{N: n, Unit: u} }
func until(n int, u Unit) *Age { a := age(n, u); return &a }

func line(id, group, antigen string, series int, from Age, u *Age, kind Kind) Line {
	return Line{ID: id, Group: group, Antigen: antigen, Series: series, From: from, Until: u, Kind: kind}
}

// usShaped is a small synthetic schedule in the shape of the US one.
func usShaped() Schema {
	return Schema{ID: "us", Lines: []Line{
		line("hepatitis_b-1", "HepB", "hepatitis_b", 1, age(0, UnitDay), until(2, UnitMonth), KindRoutine),
		line("diphtheria-1", "DTaP", "diphtheria", 1, age(2, UnitMonth), until(3, UnitMonth), KindRoutine),
		line("tetanus-1", "DTaP", "tetanus", 1, age(2, UnitMonth), until(3, UnitMonth), KindRoutine),
		line("rotavirus-1", "RV", "rotavirus", 1, age(2, UnitMonth), nil, KindRoutine),
		line("rotavirus-2", "RV", "rotavirus", 2, age(4, UnitMonth), until(5, UnitMonth), KindMayNotBeNeeded),
		line("covid19-1", "COVID-19", "covid19", 1, age(6, UnitMonth), nil, KindSupplementary),
		line("influenza-1", "Influenza", "influenza", 1, age(6, UnitMonth), nil, KindRecurring),
	}}
}

const usBorn = "2025-01-10"

func mustEval(t *testing.T, s Schema, doses []Dose, dismissals map[string]string, born, today string) Evaluation {
	t.Helper()
	e, err := Evaluate(s, doses, dismissals, born, today, time.UTC)
	if err != nil {
		t.Fatalf("Evaluate(%s, %s): %v", born, today, err)
	}
	return e
}

func find(e Evaluation, id string) (LineState, bool) {
	for _, l := range append(append([]LineState{}, e.Lines...), e.Also...) {
		if l.Line.ID == id {
			return l, true
		}
	}
	return LineState{}, false
}

func statusOf(t *testing.T, e Evaluation, id string) Status {
	t.Helper()
	l, ok := find(e, id)
	if !ok {
		t.Fatalf("line %s missing", id)
	}
	return l.Status
}

func TestEvaluateStatuses(t *testing.T) {
	s := usShaped()
	cases := []struct {
		name       string
		doses      []Dose
		dismissals map[string]string
		today, id  string
		want       Status
	}{
		{"before from", nil, nil, "2025-03-09", "diphtheria-1", StatusUpcoming},
		{"on from", nil, nil, "2025-03-10", "diphtheria-1", StatusDue},
		{"day before until", nil, nil, "2025-03-09", "hepatitis_b-1", StatusDue},
		{"on until", nil, nil, "2025-03-10", "hepatitis_b-1", StatusOverdue},
		{"default window still due", nil, nil, "2025-04-09", "rotavirus-1", StatusDue},
		{"default window ends one month after from", nil, nil, "2025-04-10", "rotavirus-1", StatusOverdue},
		{"given beats overdue", []Dose{{ID: "d1", Administration: "a1", Antigen: "hepatitis_b", GivenOn: "2025-01-11"}}, nil, "2025-06-01", "hepatitis_b-1", StatusGiven},
		{"dismissed beats overdue", nil, map[string]string{"hepatitis_b-1": "not_needed"}, "2025-06-01", "hepatitis_b-1", StatusDismissed},
		{"dismissed beats due", nil, map[string]string{"diphtheria-1": "not_needed"}, "2025-03-15", "diphtheria-1", StatusDismissed},
		{"given beats dismissed", []Dose{{ID: "d1", Administration: "a1", Antigen: "hepatitis_b", GivenOn: "2025-01-11"}}, map[string]string{"hepatitis_b-1": "given_unknown"}, "2025-03-15", "hepatitis_b-1", StatusGiven},
		{"may_not_be_needed due", nil, nil, "2025-05-10", "rotavirus-2", StatusDue},
		{"may_not_be_needed day before until", nil, nil, "2025-06-09", "rotavirus-2", StatusDue},
		{"may_not_be_needed on until is available", nil, nil, "2025-06-10", "rotavirus-2", StatusAvailable},
		{"may_not_be_needed long after is available", nil, nil, "2027-01-01", "rotavirus-2", StatusAvailable},
		{"may_not_be_needed given beats available", []Dose{
			{ID: "r1", Administration: "a1", Antigen: "rotavirus", GivenOn: "2025-03-10"},
			{ID: "r2", Administration: "a2", Antigen: "rotavirus", GivenOn: "2025-05-10"},
		}, nil, "2027-01-01", "rotavirus-2", StatusGiven},
		{"may_not_be_needed dismissed beats available", nil, map[string]string{"rotavirus-2": "not_needed"}, "2027-01-01", "rotavirus-2", StatusDismissed},
		{"may_not_be_needed upcoming", nil, nil, "2025-05-09", "rotavirus-2", StatusUpcoming},
		{"supplementary available", nil, nil, "2027-01-01", "covid19-1", StatusAvailable},
		{"supplementary given", []Dose{{ID: "c1", Administration: "a2", Antigen: "covid19", GivenOn: "2025-08-01"}}, nil, "2027-01-01", "covid19-1", StatusGiven},
		{"supplementary dismissed", nil, map[string]string{"covid19-1": "not_needed"}, "2027-01-01", "covid19-1", StatusDismissed},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := mustEval(t, s, c.doses, c.dismissals, usBorn, c.today)
			if got := statusOf(t, e, c.id); got != c.want {
				t.Errorf("%s = %q; want %q", c.id, got, c.want)
			}
		})
	}

	e := mustEval(t, s, []Dose{{ID: "d1", Administration: "a1", Antigen: "hepatitis_b", GivenOn: "2025-01-11"}},
		map[string]string{"diphtheria-1": "not_needed"}, usBorn, "2025-03-15")
	hb, _ := find(e, "hepatitis_b-1")
	if hb.GivenOn != "2025-01-11" || hb.DoseID != "d1" || hb.Administration != "a1" || hb.FromDate != usBorn || hb.UntilDate != "2025-03-10" {
		t.Errorf("given line = %+v", hb)
	}
	dt, _ := find(e, "diphtheria-1")
	if dt.DismissReason != "not_needed" || dt.FromDate != "2025-03-10" || dt.UntilDate != "2025-04-10" {
		t.Errorf("dismissed line = %+v", dt)
	}
	for _, l := range e.Lines {
		if l.Line.Kind != KindRoutine && l.Line.Kind != KindMayNotBeNeeded {
			t.Errorf("Lines holds %s of kind %s", l.Line.ID, l.Line.Kind)
		}
	}
	if len(e.Also) != 1 || e.Also[0].Line.ID != "covid19-1" {
		t.Errorf("Also = %+v; want covid19-1 only", e.Also)
	}

	flu := []Dose{
		{ID: "f2", Administration: "a4", Antigen: "influenza", GivenOn: "2026-10-01"},
		{ID: "f1", Administration: "a3", Antigen: "influenza", GivenOn: "2025-10-01"},
	}
	e = mustEval(t, s, flu, nil, usBorn, "2027-01-01")
	if _, ok := find(e, "influenza-1"); ok {
		t.Error("recurring line is in Lines or Also")
	}
	if len(e.Recurring) != 1 || e.Recurring[0].Line.ID != "influenza-1" {
		t.Fatalf("Recurring = %+v", e.Recurring)
	}
	if got := e.Recurring[0].Doses; len(got) != 2 || got[0].ID != "f1" || got[1].ID != "f2" {
		t.Errorf("recurring doses = %+v; want f1 then f2", got)
	}
	if len(e.Other) != 0 {
		t.Errorf("Other = %+v", e.Other)
	}
	e = mustEval(t, s, nil, nil, usBorn, "2027-01-01")
	if len(e.Recurring) != 1 || len(e.Recurring[0].Doses) != 0 {
		t.Errorf("recurring without doses = %+v", e.Recurring)
	}
}

func TestEvaluateOrder(t *testing.T) {
	e := mustEval(t, usShaped(), nil, nil, usBorn, "2025-03-10")
	var ids []string
	for _, l := range e.Lines {
		ids = append(ids, l.Line.ID)
	}
	want := []string{"hepatitis_b-1", "diphtheria-1", "tetanus-1", "rotavirus-1", "rotavirus-2"}
	if !reflect.DeepEqual(ids, want) {
		t.Errorf("order = %v; want %v", ids, want)
	}
}

// dtSeries is DTaP-shaped: five diphtheria and five tetanus lines.
func dtSeries() Schema {
	var s Schema
	ages := []Age{age(2, UnitMonth), age(4, UnitMonth), age(6, UnitMonth), age(15, UnitMonth), age(4, UnitYear)}
	for _, a := range []string{"diphtheria", "tetanus"} {
		for i, from := range ages {
			id := a + "-" + string(rune('1'+i))
			s.Lines = append(s.Lines, line(id, "DTaP", a, i+1, from, nil, KindRoutine))
		}
	}
	return s
}

func givenOn(e Evaluation, id string) string {
	l, _ := find(e, id)
	return l.DoseID
}

func TestCountMatching(t *testing.T) {
	s := dtSeries()
	s.Lines = append(s.Lines,
		line("hepatitis_b-1", "HepB", "hepatitis_b", 1, age(0, UnitDay), nil, KindRoutine),
		line("hepatitis_b-2", "HepB", "hepatitis_b", 2, age(1, UnitMonth), nil, KindRoutine),
		line("hepatitis_b-3", "HepB", "hepatitis_b", 3, age(6, UnitMonth), nil, KindRoutine))
	history := []Dose{
		{ID: "d1", Administration: "a1", Antigen: "diphtheria", GivenOn: "2025-03-10"},
		{ID: "d2", Administration: "a2", Antigen: "diphtheria", GivenOn: "2025-05-10"},
		{ID: "d3", Administration: "a3", Antigen: "diphtheria", GivenOn: "2025-07-10"},
		{ID: "h1", Administration: "a4", Antigen: "hepatitis_b", GivenOn: "2025-01-10"},
		{ID: "h2", Administration: "a5", Antigen: "hepatitis_b", GivenOn: "2025-02-10"},
		{ID: "x-d", Administration: "x", Antigen: "diphtheria", GivenOn: "2026-04-10"},
		{ID: "x-h", Administration: "x", Antigen: "hepatitis_b", GivenOn: "2026-04-10"},
	}
	e := mustEval(t, s, history, nil, usBorn, "2026-04-20")
	d4, _ := find(e, "diphtheria-4")
	h3, _ := find(e, "hepatitis_b-3")
	if d4.DoseID != "x-d" || h3.DoseID != "x-h" || d4.Administration != "x" || h3.Administration != "x" {
		t.Errorf("one administration did not fill diphtheria-4 and hepatitis_b-3: %+v, %+v", d4, h3)
	}
	s = dtSeries()

	var six []Dose
	for i, d := range []string{"2025-03-10", "2025-05-10", "2025-07-10", "2026-04-10", "2029-01-10", "2029-06-01"} {
		six = append(six, Dose{ID: "d" + string(rune('1'+i)), Administration: "a", Antigen: "diphtheria", GivenOn: d})
	}
	e = mustEval(t, s, six, nil, usBorn, "2030-01-01")
	for i := 1; i <= 5; i++ {
		id := "diphtheria-" + string(rune('0'+i))
		if statusOf(t, e, id) != StatusGiven {
			t.Errorf("%s not given", id)
		}
	}
	if len(e.Other) != 1 || e.Other[0].ID != "d6" {
		t.Errorf("Other = %+v; want d6", e.Other)
	}

	three := six[:3]
	e = mustEval(t, s, []Dose{three[0], three[2]}, nil, usBorn, "2025-08-01")
	if givenOn(e, "diphtheria-2") != "d3" || statusOf(t, e, "diphtheria-3") == StatusGiven {
		t.Errorf("deleting the middle dose did not shift: %+v", e.Lines)
	}

	late := []Dose{
		{ID: "a", Administration: "a", Antigen: "diphtheria", GivenOn: "2025-05-10"},
		{ID: "b", Administration: "b", Antigen: "diphtheria", GivenOn: "2025-03-10"},
		{ID: "c2", Administration: "c", Antigen: "diphtheria", GivenOn: "2025-07-10"},
		{ID: "c1", Administration: "c", Antigen: "diphtheria", GivenOn: "2025-07-10"},
	}
	e = mustEval(t, s, late, nil, usBorn, "2025-08-01")
	if givenOn(e, "diphtheria-1") != "b" || givenOn(e, "diphtheria-2") != "a" || givenOn(e, "diphtheria-3") != "c1" || givenOn(e, "diphtheria-4") != "c2" {
		t.Errorf("doses not matched by date then id: %+v", e.Lines)
	}

	e = mustEval(t, s, []Dose{{ID: "t1", Administration: "t", Antigen: "typhoid", GivenOn: "2025-03-10"}}, nil, usBorn, "2025-08-01")
	if len(e.Other) != 1 || e.Other[0].ID != "t1" {
		t.Errorf("Other = %+v; want t1", e.Other)
	}
}

func TestOrphanDismissals(t *testing.T) {
	e := mustEval(t, usShaped(), nil, map[string]string{"gone-1": "not_needed", "diphtheria-9": "given_unknown"}, usBorn, "2025-03-15")
	for _, l := range append(e.Lines, e.Also...) {
		if l.Status == StatusDismissed || l.DismissReason != "" {
			t.Errorf("%s dismissed by an orphan dismissal", l.Line.ID)
		}
	}
}

func TestCohorts(t *testing.T) {
	old := line("diphtheria-4", "DTaP", "diphtheria", 4, age(15, UnitMonth), nil, KindRoutine)
	old.BornTo = "2024-06-30"
	young := line("diphtheria-4-b2024", "6-in-1", "diphtheria", 4, age(18, UnitMonth), nil, KindRoutine)
	young.BornFrom = "2024-07-01"
	mmr := line("measles-1", "MMR", "measles", 1, age(12, UnitMonth), nil, KindRoutine)
	mmr.BornFrom = "2024-07-01"
	s := Schema{Lines: []Line{old, young, mmr}}

	e := mustEval(t, s, []Dose{{ID: "m1", Administration: "m", Antigen: "measles", GivenOn: "2025-07-01"}}, nil, "2024-06-30", "2026-01-01")
	if _, ok := find(e, "diphtheria-4"); !ok {
		t.Error("born 2024-06-30 lacks diphtheria-4")
	}
	if _, ok := find(e, "diphtheria-4-b2024"); ok {
		t.Error("born 2024-06-30 has diphtheria-4-b2024")
	}
	if _, ok := find(e, "measles-1"); ok {
		t.Error("born 2024-06-30 has a measles line")
	}
	if len(e.Other) != 1 || e.Other[0].ID != "m1" {
		t.Errorf("Other = %+v; want m1 (no cohort of measles)", e.Other)
	}

	e = mustEval(t, s, nil, nil, "2024-07-01", "2026-01-01")
	if _, ok := find(e, "diphtheria-4"); ok {
		t.Error("born 2024-07-01 has diphtheria-4")
	}
	l, ok := find(e, "diphtheria-4-b2024")
	if !ok || l.FromDate != "2026-01-01" {
		t.Errorf("born 2024-07-01 diphtheria-4-b2024 = %+v, %v", l, ok)
	}
}

func TestWindowsInTheKidsZone(t *testing.T) {
	auckland, err := time.LoadLocation("Pacific/Auckland")
	if err != nil {
		t.Fatal(err)
	}
	s := Schema{Lines: []Line{line("rotavirus-1", "RV", "rotavirus", 1, age(8, UnitWeek), until(12, UnitWeek), KindRoutine)}}
	const born = "2025-02-05" // from 2025-04-02, until 2025-04-30
	cases := []struct {
		instant time.Time
		loc     *time.Location
		want    Status
	}{
		{time.Date(2025, 4, 1, 11, 30, 0, 0, time.UTC), auckland, StatusDue},
		{time.Date(2025, 4, 1, 11, 30, 0, 0, time.UTC), time.UTC, StatusUpcoming},
		{time.Date(2025, 4, 29, 12, 30, 0, 0, time.UTC), auckland, StatusOverdue},
		{time.Date(2025, 4, 29, 12, 30, 0, 0, time.UTC), time.UTC, StatusDue},
	}
	for _, c := range cases {
		start, _ := kidtime.DayBounds(c.loc, c.instant)
		today := start.Format(time.DateOnly)
		e, err := Evaluate(s, nil, nil, born, today, c.loc)
		if err != nil {
			t.Fatal(err)
		}
		if got := e.Lines[0].Status; got != c.want {
			t.Errorf("%s in %s (today %s) = %q; want %q", c.instant, c.loc, today, got, c.want)
		}
	}
}

func TestDueGroups(t *testing.T) {
	s := Schema{Lines: []Line{
		line("diphtheria-1", "DTaP", "diphtheria", 1, age(2, UnitMonth), until(7, UnitMonth), KindRoutine),
		line("tetanus-1", "DTaP", "tetanus", 1, age(2, UnitMonth), until(3, UnitMonth), KindRoutine),
		line("rotavirus-1", "RV", "rotavirus", 1, age(8, UnitWeek), until(9, UnitMonth), KindRoutine),
		line("covid19-1", "COVID-19", "covid19", 1, age(2, UnitMonth), nil, KindSupplementary),
		line("influenza-1", "Influenza", "influenza", 1, age(2, UnitMonth), nil, KindRecurring),
		line("diphtheria-2", "DTaP", "diphtheria", 2, age(4, UnitMonth), until(8, UnitMonth), KindRoutine),
		line("rotavirus-2", "RV", "rotavirus", 2, age(4, UnitMonth), until(5, UnitMonth), KindMayNotBeNeeded),
		line("hib-1", "Hib", "hib", 1, age(1, UnitMonth), until(2, UnitMonth), KindMayNotBeNeeded),
		line("pertussis-1", "DTaP", "pertussis", 1, age(2, UnitMonth), until(3, UnitMonth), KindMayNotBeNeeded),
		line("diphtheria-3", "DTaP", "diphtheria", 3, age(6, UnitMonth), nil, KindRoutine),
	}}
	e := mustEval(t, s, nil, nil, usBorn, "2025-06-01")
	want := []DueGroup{
		{From: age(8, UnitWeek), FromDate: "2025-03-07", Status: StatusDue, Count: 1},
		{From: age(2, UnitMonth), FromDate: "2025-03-10", Status: StatusOverdue, Count: 2},
		{From: age(4, UnitMonth), FromDate: "2025-05-10", Status: StatusDue, Count: 2},
	}
	if got := e.Due(); !reflect.DeepEqual(got, want) {
		t.Errorf("Due() = %+v; want %+v", got, want)
	}
	for _, id := range []string{"hib-1", "pertussis-1"} {
		if got := statusOf(t, e, id); got != StatusAvailable {
			t.Errorf("%s past its window = %q; want available (and not counted)", id, got)
		}
	}

	doses := []Dose{
		{ID: "1", Administration: "a", Antigen: "diphtheria", GivenOn: "2025-03-10"},
		{ID: "2", Administration: "a", Antigen: "tetanus", GivenOn: "2025-03-10"},
		{ID: "3", Administration: "b", Antigen: "rotavirus", GivenOn: "2025-03-10"},
		{ID: "4", Administration: "c", Antigen: "diphtheria", GivenOn: "2025-05-10"},
	}
	e = mustEval(t, s, doses, map[string]string{"rotavirus-2": "not_needed"}, usBorn, "2025-06-01")
	if got := e.Due(); len(got) != 0 {
		t.Errorf("Due() with everything given or dismissed = %+v; want none", got)
	}
}

func TestEmptyWindowIsAnError(t *testing.T) {
	s := Schema{Lines: []Line{line("diphtheria-1", "DTaP", "diphtheria", 1, age(4, UnitWeek), until(1, UnitMonth), KindRoutine)}}
	if _, err := Evaluate(s, nil, nil, "2025-01-10", "2025-03-01", time.UTC); err != nil {
		t.Fatalf("a January birth has a window: %v", err)
	}
	_, err := Evaluate(s, nil, nil, "2025-02-01", "2025-03-01", time.UTC)
	if !errors.Is(err, ErrEmptyWindow) {
		t.Fatalf("4w..1m from 2025-02-01 is empty: err = %v; want ErrEmptyWindow", err)
	}
	if strings.Contains(err.Error(), "2025") {
		t.Errorf("error carries a date: %v", err)
	}
}

func TestEvaluateBadDates(t *testing.T) {
	for _, c := range [][2]string{{"2025-02-30", "2025-03-01"}, {usBorn, "tomorrow"}} {
		_, err := Evaluate(usShaped(), nil, nil, c[0], c[1], time.UTC)
		if !errors.Is(err, kidtime.ErrBadDate) {
			t.Errorf("Evaluate(%q, %q) err = %v; want ErrBadDate", c[0], c[1], err)
		}
	}
}

func TestDefaultWindowRunsFromTheFromDate(t *testing.T) {
	cases := []struct {
		born     string
		from     Age
		fromDate string
		untilDay string
		lastDue  string
	}{
		{"2025-01-31", age(1, UnitMonth), "2025-03-03", "2025-04-03", "2025-04-02"},
		// birth + 13m would be 2025-03-29: a window counted from birth fails here.
		{"2024-02-29", age(12, UnitMonth), "2025-03-01", "2025-04-01", "2025-03-31"},
	}
	for _, c := range cases {
		s := Schema{Lines: []Line{line("rotavirus-1", "RV", "rotavirus", 1, c.from, nil, KindRoutine)}}
		for today, want := range map[string]Status{c.lastDue: StatusDue, c.untilDay: StatusOverdue} {
			e := mustEval(t, s, nil, nil, c.born, today)
			l := e.Lines[0]
			if l.FromDate != c.fromDate || l.UntilDate != c.untilDay || l.Status != want {
				t.Errorf("born %s on %s = %s..%s %q; want %s..%s %q", c.born, today, l.FromDate, l.UntilDate, l.Status, c.fromDate, c.untilDay, want)
			}
		}
	}
}

func TestEmptySchemaSendsEverythingToOther(t *testing.T) {
	doses := []Dose{
		{ID: "b", Administration: "b", Antigen: "measles", GivenOn: "2025-05-01"},
		{ID: "a", Administration: "a", Antigen: "diphtheria", GivenOn: "2025-03-01"},
	}
	e := mustEval(t, Schema{}, doses, map[string]string{"diphtheria-1": "not_needed"}, usBorn, "2025-06-01")
	if len(e.Lines)+len(e.Also)+len(e.Recurring) != 0 || len(e.Other) != 2 || e.Other[0].ID != "a" || e.Other[1].ID != "b" {
		t.Errorf("empty schema = %+v", e)
	}
	if len(e.Due()) != 0 {
		t.Errorf("Due() = %+v", e.Due())
	}
}

func TestStatusAvailableIsNotTheZeroValue(t *testing.T) {
	var zero Status
	if StatusAvailable == zero {
		t.Error("StatusAvailable is the zero value")
	}
}
