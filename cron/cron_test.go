package cron

import (
	"strings"
	"testing"
	"time"
)

func mustLoc(t *testing.T, name string) *time.Location {
	t.Helper()
	l, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func mustParse(t *testing.T, e string) *Schedule {
	t.Helper()
	s, err := Parse(e)
	if err != nil {
		t.Fatalf("Parse(%q): %v", e, err)
	}
	return s
}

func TestParseValid(t *testing.T) {
	cases := []struct {
		expr    string
		seconds bool
		field   FieldKind
		want    []int
	}{
		{"* * * * *", false, Minute, seq(0, 59, 1)},
		{"*/15 * * * *", false, Minute, []int{0, 15, 30, 45}},
		{"5/20 * * * *", false, Minute, []int{5, 25, 45}},
		{"10-20/5 * * * *", false, Minute, []int{10, 15, 20}},
		{"0 9-17 * * *", false, Hour, seq(9, 17, 1)},
		{"0 0 1,15,31 * *", false, DayOfMonth, []int{1, 15, 31}},
		{"0 0 * JAN-MAR *", false, Month, []int{1, 2, 3}},
		{"0 0 * jan,Jul,DEC *", false, Month, []int{1, 7, 12}},
		{"0 0 * * MON-FRI", false, DayOfWeek, []int{1, 2, 3, 4, 5}},
		{"0 0 * * 7", false, DayOfWeek, []int{0}},
		{"0 0 * * 5-7", false, DayOfWeek, []int{0, 5, 6}},
		{"0 0 * * */2", false, DayOfWeek, []int{0, 2, 4, 6}},
		{"0 0 ? * 1", false, DayOfWeek, []int{1}},
		{"*/10 * * * * *", true, Second, []int{0, 10, 20, 30, 40, 50}},
		{"30 0 12 * * *", true, Hour, []int{12}},
		{"@hourly", false, Minute, []int{0}},
		{"@daily", false, Hour, []int{0}},
		{"@weekly", false, DayOfWeek, []int{0}},
		{"@monthly", false, DayOfMonth, []int{1}},
		{"@yearly", false, Month, []int{1}},
		{"@ANNUALLY", false, Month, []int{1}},
		{"  0   12  *  *  *  ", false, Hour, []int{12}},
	}
	for _, c := range cases {
		s := mustParse(t, c.expr)
		if s.HasSeconds != c.seconds {
			t.Errorf("%q: HasSeconds = %v", c.expr, s.HasSeconds)
		}
		got := s.Fields[c.field].Values()
		if !equalInts(got, c.want) {
			t.Errorf("%q %s = %v, want %v", c.expr, c.field, got, c.want)
		}
	}
}

func TestParseInvalid(t *testing.T) {
	cases := map[string]string{
		"":                "empty",
		"* * * *":         "expected 5 fields",
		"* * * * * * *":   "expected 5 fields",
		"60 * * * *":      "outside 0-59",
		"0 24 * * *":      "outside 0-23",
		"0 0 0 * *":       "outside 1-31",
		"0 0 32 * *":      "outside 1-31",
		"0 0 * 13 *":      "outside 1-12",
		"0 0 * * 8":       "outside 0-7",
		"*/0 * * * *":     "positive",
		"*/x * * * *":     "positive",
		"5-1 * * * *":     "cannot wrap",
		"0 0 * * FRI-MON": "5-6,0-1",
		"0 0 L * *":       "Quartz",
		"0 0 * * 5#3":     "Quartz",
		"0 0 15W * *":     "Quartz",
		"0 0 * FOO *":     "JAN-DEC",
		"0 0 * * FUN":     "SUN-SAT",
		"1,,2 * * * *":    "extra comma",
		"? * * * *":       "only allowed",
		"@reboot":         "start-up",
		"@sometimes":      "unknown macro",
	}
	for expr, want := range cases {
		_, err := Parse(expr)
		if err == nil {
			t.Errorf("Parse(%q) succeeded, want error containing %q", expr, want)
			continue
		}
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Parse(%q) error %q, want it to contain %q", expr, err, want)
		}
	}
}

func TestExplain(t *testing.T) {
	cases := []struct{ expr, en, bg string }{
		{"0 9 * * 1-5", "At 09:00, Monday through Friday", "В 09:00, от понеделник до петък"},
		{"*/5 * * * *", "Every 5 minutes", "На всеки 5 минути"},
		{"* * * * *", "Every minute", "Всяка минута"},
		{"30 3 * * *", "At 03:30, every day", "В 03:30, всеки ден"},
		{"0 30 2 * * *", "At 02:30, every day", "В 02:30, всеки ден"},
		{"15 10 * * 2", "At 10:15, on Tuesday", "В 10:15, във вторник"},
		{"0 7 * * 3,5", "At 07:00, on Wednesday and Friday", "В 07:00, в сряда и петък"},
		{"5 4 * * 0,3,6", "At 04:05, on Wednesday, Saturday and Sunday", "В 04:05, в сряда, събота и неделя"},
		{"0 8,12,18 * * *", "At 08:00, 12:00 and 18:00, every day", "В 08:00, 12:00 и 18:00, всеки ден"},
		{"*/15 9-17 * * MON-FRI", "Every 15 minutes, between 09:00 and 17:59, Monday through Friday",
			"На всеки 15 минути, между 09:00 и 17:59, от понеделник до петък"},
		{"0 */2 * * *", "At minute 0, every 2 hours", "В минута 0, на всеки 2 часа"},
		{"30 * * * *", "At minute 30 past every hour", "В минута 30 на всеки час"},
		{"0 0 1,15 * 1", "At 00:00, on days 1 and 15 of the month or on Monday",
			"В 00:00, на дни 1 и 15 от месеца или в понеделник"},
		{"0 0 1 1-3 *", "At 00:00, on day 1 of the month, from January through March",
			"В 00:00, на ден 1 от месеца, от януари до март"},
		{"0 6 * 6,7,8 *", "At 06:00, every day, in June, July and August", "В 06:00, всеки ден, през юни, юли и август"},
		{"*/10 * * * * *", "Every 10 seconds", "На всеки 10 секунди"},
		{"15,45 * * * * *", "At seconds 15 and 45", "В секунди 15 и 45"},
		{"0-29 * * * *", "Every minute from 0 through 29", "Всяка минута от 0 до 29"},
		{"@weekly", "At 00:00, on Sunday", "В 00:00, в неделя"},
		{"@hourly", "At minute 0 past every hour", "В минута 0 на всеки час"},
		{"0 0 */2 * *", "At 00:00, every 2 days of the month starting on day 1",
			"В 00:00, на всеки 2 дни от месеца, като се започне от ден 1"},
		{"0 22 * * 1-4", "At 22:00, Monday through Thursday", "В 22:00, от понеделник до четвъртък"},
	}
	for _, c := range cases {
		s := mustParse(t, c.expr)
		if got := s.Explain(English); got != c.en {
			t.Errorf("EN %q:\n got  %q\n want %q", c.expr, got, c.en)
		}
		if got := s.Explain(Bulgarian); got != c.bg {
			t.Errorf("BG %q:\n got  %q\n want %q", c.expr, got, c.bg)
		}
	}
}

type wantRun struct {
	local string // "2006-01-02 15:04:05 MST"
	note  string
}

func checkRuns(t *testing.T, expr, zone, from string, want []wantRun) {
	t.Helper()
	loc := mustLoc(t, zone)
	start, err := time.ParseInLocation("2006-01-02 15:04:05", from, loc)
	if err != nil {
		t.Fatal(err)
	}
	runs, err := mustParse(t, expr).Next(start, loc, len(want))
	if err != nil {
		t.Fatalf("%q: %v", expr, err)
	}
	if len(runs) != len(want) {
		t.Fatalf("%q: got %d runs, want %d", expr, len(runs), len(want))
	}
	for i, r := range runs {
		got := r.Time.Format("2006-01-02 15:04:05 MST")
		if got != want[i].local || r.Note != want[i].note {
			t.Errorf("%q in %s run %d: got %s %q, want %s %q", expr, zone, i+1, got, r.Note, want[i].local, want[i].note)
		}
	}
}

func TestNextUTC(t *testing.T) {
	checkRuns(t, "0 9 * * 1-5", "UTC", "2026-10-09 10:00:00", []wantRun{
		{"2026-10-12 09:00:00 UTC", ""}, {"2026-10-13 09:00:00 UTC", ""}, {"2026-10-14 09:00:00 UTC", ""},
	})
	checkRuns(t, "*/20 * * * * *", "UTC", "2026-01-01 00:00:00", []wantRun{
		{"2026-01-01 00:00:20 UTC", ""}, {"2026-01-01 00:00:40 UTC", ""}, {"2026-01-01 00:01:00 UTC", ""},
	})
	checkRuns(t, "@yearly", "UTC", "2026-06-01 00:00:00", []wantRun{
		{"2027-01-01 00:00:00 UTC", ""}, {"2028-01-01 00:00:00 UTC", ""},
	})
	// Strictly after: a start exactly on a run time is excluded.
	checkRuns(t, "0 12 * * *", "UTC", "2026-05-05 12:00:00", []wantRun{{"2026-05-06 12:00:00 UTC", ""}})
	// Leap day.
	checkRuns(t, "0 0 29 2 *", "UTC", "2026-01-01 00:00:00", []wantRun{
		{"2028-02-29 00:00:00 UTC", ""}, {"2032-02-29 00:00:00 UTC", ""},
	})
	// Day 31 skips short months.
	checkRuns(t, "0 0 31 * *", "UTC", "2026-01-31 12:00:00", []wantRun{
		{"2026-03-31 00:00:00 UTC", ""}, {"2026-05-31 00:00:00 UTC", ""}, {"2026-07-31 00:00:00 UTC", ""},
	})
	// Day-of-month OR day-of-week when both are set.
	checkRuns(t, "0 0 13 * 5", "UTC", "2026-02-01 00:00:00", []wantRun{
		{"2026-02-06 00:00:00 UTC", ""}, {"2026-02-13 00:00:00 UTC", ""}, {"2026-02-20 00:00:00 UTC", ""},
		{"2026-02-27 00:00:00 UTC", ""}, {"2026-03-06 00:00:00 UTC", ""}, {"2026-03-13 00:00:00 UTC", ""},
	})
	// Star-prefixed day-of-month step with a weekday means AND.
	checkRuns(t, "0 0 */10 * 1", "UTC", "2026-01-01 00:00:00", []wantRun{
		{"2026-05-11 00:00:00 UTC", ""}, {"2026-06-01 00:00:00 UTC", ""},
	})
}

func TestNeverRuns(t *testing.T) {
	for _, e := range []string{"0 0 30 2 *", "0 0 31 4 *", "0 0 31 2,4,6,9,11 *"} {
		_, err := mustParse(t, e).Next(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.UTC, 1)
		if err != ErrNeverRuns {
			t.Errorf("%q: err = %v, want ErrNeverRuns", e, err)
		}
	}
}

// Europe/Sofia: EET (UTC+2) / EEST (UTC+3). In 2026 the clocks go forward
// on 29 March at 03:00 (to 04:00) and back on 25 October at 04:00 (to 03:00).
func TestSofiaSpringForward(t *testing.T) {
	// A fixed time inside the missing hour moves forward by the gap.
	checkRuns(t, "30 3 * * *", "Europe/Sofia", "2026-03-27 12:00:00", []wantRun{
		{"2026-03-28 03:30:00 EET", ""},
		{"2026-03-29 04:30:00 EEST", "shifted"},
		{"2026-03-30 03:30:00 EEST", ""},
	})
	// Times before and after the gap are unaffected.
	checkRuns(t, "0 2,4 * * *", "Europe/Sofia", "2026-03-29 00:00:00", []wantRun{
		{"2026-03-29 02:00:00 EET", ""}, {"2026-03-29 04:00:00 EEST", ""}, {"2026-03-30 02:00:00 EEST", ""},
	})
	// 03:00 shifted to 04:00 collides with the real 04:00 run: one run only.
	checkRuns(t, "0 3,4 * * *", "Europe/Sofia", "2026-03-29 00:00:00", []wantRun{
		{"2026-03-29 04:00:00 EEST", "shifted"}, {"2026-03-30 03:00:00 EEST", ""},
	})
	// Hour * follows real time: nothing in the missing hour, no duplicates.
	checkRuns(t, "*/30 * * * *", "Europe/Sofia", "2026-03-29 02:00:00", []wantRun{
		{"2026-03-29 02:30:00 EET", ""}, {"2026-03-29 04:00:00 EEST", ""}, {"2026-03-29 04:30:00 EEST", ""},
	})
	// Seconds field through the gap.
	checkRuns(t, "59 59 2 * * *", "Europe/Sofia", "2026-03-29 00:00:00", []wantRun{
		{"2026-03-29 02:59:59 EET", ""}, {"2026-03-30 02:59:59 EEST", ""},
	})
	checkRuns(t, "15 30 3 29 3 *", "Europe/Sofia", "2026-01-01 00:00:00", []wantRun{
		{"2026-03-29 04:30:15 EEST", "shifted"}, {"2027-03-29 03:30:15 EEST", ""},
	})
	// 2027: last Sunday of March is the 28th.
	checkRuns(t, "45 3 28 3 *", "Europe/Sofia", "2027-01-01 00:00:00", []wantRun{
		{"2027-03-28 04:45:00 EEST", "shifted"},
	})
}

func TestSofiaFallBack(t *testing.T) {
	// A fixed time inside the repeated hour runs once, at the first pass.
	checkRuns(t, "30 3 * * *", "Europe/Sofia", "2026-10-24 12:00:00", []wantRun{
		{"2026-10-25 03:30:00 EEST", ""},
		{"2026-10-26 03:30:00 EET", ""},
	})
	// Hour * runs in both passes of the repeated hour.
	checkRuns(t, "*/30 * * * *", "Europe/Sofia", "2026-10-25 02:50:00", []wantRun{
		{"2026-10-25 03:00:00 EEST", ""},
		{"2026-10-25 03:30:00 EEST", ""},
		{"2026-10-25 03:00:00 EET", "repeated"},
		{"2026-10-25 03:30:00 EET", "repeated"},
		{"2026-10-25 04:00:00 EET", ""},
	})
	// Hourly job: 25 runs on the 25-hour day.
	loc := mustLoc(t, "Europe/Sofia")
	start := time.Date(2026, 10, 24, 23, 59, 0, 0, loc)
	runs, err := mustParse(t, "0 * * * *").Next(start, loc, 30)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, r := range runs {
		if r.Time.Day() == 25 {
			n++
		}
	}
	if n != 25 {
		t.Errorf("hourly runs on 2026-10-25 in Sofia = %d, want 25", n)
	}
	// Spring day has 23.
	start = time.Date(2026, 3, 28, 23, 59, 0, 0, loc)
	runs, _ = mustParse(t, "0 * * * *").Next(start, loc, 30)
	n = 0
	for _, r := range runs {
		if r.Time.Day() == 29 && r.Time.Month() == 3 {
			n++
		}
	}
	if n != 23 {
		t.Errorf("hourly runs on 2026-03-29 in Sofia = %d, want 23", n)
	}
	// Starting inside the second pass does not repeat the first.
	second := time.Date(2026, 10, 25, 1, 10, 0, 0, time.UTC) // 03:10 EET
	runs, _ = mustParse(t, "20 3 * * *").Next(second, loc, 1)
	if got := runs[0].Time.Format("2006-01-02 15:04 MST"); got != "2026-10-26 03:20 EET" {
		t.Errorf("after second pass: got %s", got)
	}
	runs, _ = mustParse(t, "*/20 * * * *").Next(second, loc, 2)
	if got := runs[0].Time.UTC().Format("15:04"); got != "01:20" {
		t.Errorf("wildcard hour in second pass: got %s UTC, want 01:20", got)
	}
}

func TestOtherZones(t *testing.T) {
	// New York: 2026-03-08 02:00 -> 03:00, 2026-11-01 02:00 -> 01:00.
	checkRuns(t, "30 2 * * *", "America/New_York", "2026-03-07 00:00:00", []wantRun{
		{"2026-03-07 02:30:00 EST", ""}, {"2026-03-08 03:30:00 EDT", "shifted"}, {"2026-03-09 02:30:00 EDT", ""},
	})
	checkRuns(t, "30 1 * * *", "America/New_York", "2026-10-31 12:00:00", []wantRun{
		{"2026-11-01 01:30:00 EDT", ""}, {"2026-11-02 01:30:00 EST", ""},
	})
	// Lord Howe Island has a 30-minute DST shift (02:00 -> 02:30 on 2026-10-04).
	checkRuns(t, "15 2 * * *", "Australia/Lord_Howe", "2026-10-03 12:00:00", []wantRun{
		{"2026-10-04 02:45:00 +11", "shifted"}, {"2026-10-05 02:15:00 +11", ""},
	})
	// Same instant schedule gives the same UTC result whatever zone the start is in.
	checkRuns(t, "0 0 * * *", "Asia/Tokyo", "2026-01-01 08:00:00", []wantRun{{"2026-01-02 00:00:00 JST", ""}})
}

func seq(a, b, step int) []int {
	var out []int
	for v := a; v <= b; v += step {
		out = append(out, v)
	}
	return out
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
