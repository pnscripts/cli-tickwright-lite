package cron

import (
	"errors"
	"sort"
	"time"
)

// Run is one computed run time.
type Run struct {
	Time time.Time
	// Note explains daylight-saving handling: "" for a normal run,
	// "shifted" when the wall-clock time did not exist (spring forward)
	// and the run moved forward by the gap, "repeated" for the second
	// occurrence of a wall-clock time during a fall-back hour.
	Note string
}

// ErrNeverRuns is returned when no run exists in the search window.
var ErrNeverRuns = errors.New("the expression never matches a real date (searched 8 years ahead)")

// searchYears bounds the search so that impossible dates such as
// 30 February terminate. Leap-day schedules need at most 8 years.
const searchYears = 8

// MatchesDay reports whether the calendar date matches the day fields,
// month field, using cron's rule: when both day-of-month and day-of-week
// are restricted, a day matches if EITHER matches.
func (s *Schedule) MatchesDay(y int, m time.Month, d int) bool {
	if !s.Fields[Month].Set[int(m)] {
		return false
	}
	wd := int(time.Date(y, m, d, 12, 0, 0, 0, time.UTC).Weekday())
	domOK := s.Fields[DayOfMonth].Set[d]
	dowOK := s.Fields[DayOfWeek].Set[wd]
	domR, dowR := s.DayRestricted()
	if domR && dowR {
		return domOK || dowOK
	}
	return domOK && dowOK
}

// hourUnrestricted reports whether the hour field covers all 24 hours.
// Such schedules follow real elapsed time across DST changes; schedules
// with specific hours follow the wall clock (run once per wall time).
func (s *Schedule) hourUnrestricted() bool { return s.Fields[Hour].Full() }

// Next returns the next n run times strictly after `after`, in loc.
//
// Daylight-saving rules (the same rules classic Vixie cron applies):
//   - spring forward: a wall-clock time that does not exist runs at the
//     instant it would have had without the change, i.e. moved forward by
//     the gap (03:30 becomes 04:30 in Europe/Sofia). Schedules whose hour
//     field is * simply have no runs in the missing hour.
//   - fall back: a repeated wall-clock time runs once, at its first
//     occurrence. Schedules whose hour field is * run in both copies of
//     the repeated hour, because real time keeps passing.
func (s *Schedule) Next(after time.Time, loc *time.Location, n int) ([]Run, error) {
	if loc == nil {
		loc = time.UTC
	}
	var out []Run
	local := after.In(loc)
	day := time.Date(local.Year(), local.Month(), local.Day(), 12, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	limit := day.AddDate(searchYears, 0, 0)
	seen := map[int64]bool{}
	secs := s.Fields[Second].Values()
	mins := s.Fields[Minute].Values()
	hours := s.Fields[Hour].Values()
	follow := s.hourUnrestricted()
	for ; day.Before(limit) && len(out) < n; day = day.AddDate(0, 0, 1) {
		y, m, d := day.Date()
		if !s.MatchesDay(y, m, d) {
			continue
		}
		var dayRuns []Run
		for _, h := range hours {
			for _, mi := range mins {
				for _, se := range secs {
					for _, r := range resolve(y, m, d, h, mi, se, loc, follow) {
						if !r.Time.After(after) {
							continue
						}
						k := r.Time.Unix()
						if seen[k] {
							continue
						}
						seen[k] = true
						dayRuns = append(dayRuns, r)
					}
				}
			}
		}
		sort.Slice(dayRuns, func(i, j int) bool { return dayRuns[i].Time.Before(dayRuns[j].Time) })
		for _, r := range dayRuns {
			if len(out) == n {
				break
			}
			out = append(out, r)
		}
	}
	if len(out) == 0 && n > 0 {
		return nil, ErrNeverRuns
	}
	return out, nil
}

// resolve turns one wall-clock time into zero, one or two instants.
func resolve(y int, m time.Month, d, h, mi, se int, loc *time.Location, follow bool) []Run {
	naive := time.Date(y, m, d, h, mi, se, 0, time.UTC)
	nu := naive.Unix()
	_, offBefore := naive.Add(-36 * time.Hour).Add(-14 * time.Hour).In(loc).Zone()
	_, offAfter := naive.Add(36 * time.Hour).Add(14 * time.Hour).In(loc).Zone()
	cands := []int{offBefore}
	if offAfter != offBefore {
		cands = append(cands, offAfter)
	}
	var valid []time.Time
	for _, off := range cands {
		t := time.Unix(nu-int64(off), 0).In(loc)
		if t.Year() == y && t.Month() == m && t.Day() == d && t.Hour() == h && t.Minute() == mi && t.Second() == se {
			valid = append(valid, t)
		}
	}
	if len(valid) == 0 {
		// Also check the offset in effect at the naive instant itself, in
		// case a zone changed twice within a few days.
		_, off := time.Unix(nu, 0).In(loc).Zone()
		t := time.Unix(nu-int64(off), 0).In(loc)
		if t.Hour() == h && t.Minute() == mi && t.Second() == se && t.Day() == d {
			return []Run{{Time: t}}
		}
		if follow {
			return nil
		}
		return []Run{{Time: time.Unix(nu-int64(offBefore), 0).In(loc), Note: "shifted"}}
	}
	sort.Slice(valid, func(i, j int) bool { return valid[i].Before(valid[j]) })
	if len(valid) == 1 {
		return []Run{{Time: valid[0]}}
	}
	if follow {
		return []Run{{Time: valid[0]}, {Time: valid[1], Note: "repeated"}}
	}
	return []Run{{Time: valid[0]}}
}
