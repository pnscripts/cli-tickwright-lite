package cron

import (
	"fmt"
	"strings"
)

// Lang selects the explanation language.
type Lang string

const (
	English   Lang = "en"
	Bulgarian Lang = "bg"
)

type words struct {
	and, or                             string
	everySecond, everyMinute, everyHour string
	everyN                              func(n int, unit FieldKind) string
	atSecond, atMinute                  func(vals string, plural bool) string
	pastEveryHour                       string
	fromThrough                         func(a, b string) string
	between                             func(a, b string) string
	inHours                             func(vals string) string
	at                                  func(times string) string
	everyDay                            string
	startingAt                          func(unit FieldKind, v string) string
	dom                                 func(desc string, plural bool) string
	domRange                            func(a, b int) string
	domEveryN                           func(n int, from int) string
	months                              func(desc string) string
	monthRange                          func(a, b string) string
	dowList                             func(names []string) string
	dowRange                            func(a, b string) string
	monthName                           [13]string
	dowName                             [7]string
	unitPlural                          func(unit FieldKind, n int) string
}

var en = words{
	and: "and", or: "or",
	everySecond: "every second", everyMinute: "every minute", everyHour: "every hour",
	everyN: func(n int, u FieldKind) string { return fmt.Sprintf("every %d %s", n, enUnit(u, n)) },
	atSecond: func(v string, pl bool) string {
		if pl {
			return "at seconds " + v
		}
		return "at second " + v
	},
	atMinute: func(v string, pl bool) string {
		if pl {
			return "at minutes " + v
		}
		return "at minute " + v
	},
	pastEveryHour: "past every hour",
	fromThrough:   func(a, b string) string { return "from " + a + " through " + b },
	between:       func(a, b string) string { return "between " + a + " and " + b },
	inHours:       func(v string) string { return "during the hours starting " + v },
	at:            func(t string) string { return "at " + t },
	everyDay:      "every day",
	startingAt: func(u FieldKind, v string) string {
		return "starting at " + enUnit(u, 1) + " " + v
	},
	dom: func(d string, pl bool) string {
		if pl {
			return "on days " + d + " of the month"
		}
		return "on day " + d + " of the month"
	},
	domRange: func(a, b int) string { return fmt.Sprintf("on days %d through %d of the month", a, b) },
	domEveryN: func(n, from int) string {
		if from == 1 {
			return fmt.Sprintf("every %d days of the month starting on day 1", n)
		}
		return fmt.Sprintf("every %d days of the month starting on day %d", n, from)
	},
	months:     func(d string) string { return "in " + d },
	monthRange: func(a, b string) string { return "from " + a + " through " + b },
	dowList:    func(n []string) string { return "on " + joinList(n, "and") },
	dowRange:   func(a, b string) string { return a + " through " + b },
	monthName: [13]string{"", "January", "February", "March", "April", "May", "June", "July",
		"August", "September", "October", "November", "December"},
	dowName: [7]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
}

func enUnit(u FieldKind, n int) string {
	s := map[FieldKind]string{Second: "second", Minute: "minute", Hour: "hour", DayOfMonth: "day", Month: "month", DayOfWeek: "day"}[u]
	if n != 1 {
		s += "s"
	}
	return s
}

func bgUnit(u FieldKind, n int) string {
	one := map[FieldKind]string{Second: "секунда", Minute: "минута", Hour: "час", DayOfMonth: "ден", Month: "месец", DayOfWeek: "ден"}
	many := map[FieldKind]string{Second: "секунди", Minute: "минути", Hour: "часа", DayOfMonth: "дни", Month: "месеца", DayOfWeek: "дни"}
	if n == 1 {
		return one[u]
	}
	return many[u]
}

var bg = words{
	and: "и", or: "или",
	everySecond: "всяка секунда", everyMinute: "всяка минута", everyHour: "всеки час",
	everyN: func(n int, u FieldKind) string {
		return fmt.Sprintf("на всеки %d %s", n, bgUnit(u, n))
	},
	atSecond: func(v string, pl bool) string {
		if pl {
			return "в секунди " + v
		}
		return "в секунда " + v
	},
	atMinute: func(v string, pl bool) string {
		if pl {
			return "в минути " + v
		}
		return "в минута " + v
	},
	pastEveryHour: "на всеки час",
	fromThrough:   func(a, b string) string { return "от " + a + " до " + b },
	between:       func(a, b string) string { return "между " + a + " и " + b },
	inHours:       func(v string) string { return "в часовете, започващи в " + v },
	at:            func(t string) string { return bgAt(t) },
	everyDay:      "всеки ден",
	startingAt: func(u FieldKind, v string) string {
		return "като се започне от " + bgUnit(u, 1) + " " + v
	},
	dom: func(d string, pl bool) string {
		if pl {
			return "на дни " + d + " от месеца"
		}
		return "на ден " + d + " от месеца"
	},
	domRange: func(a, b int) string { return fmt.Sprintf("от ден %d до ден %d от месеца", a, b) },
	domEveryN: func(n, from int) string {
		return fmt.Sprintf("на всеки %d дни от месеца, като се започне от ден %d", n, from)
	},
	months:     func(d string) string { return "през " + d },
	monthRange: func(a, b string) string { return "от " + a + " до " + b },
	dowList: func(n []string) string {
		return bgPrep(n[0]) + " " + joinListBG(n)
	},
	dowRange: func(a, b string) string { return "от " + a + " до " + b },
	monthName: [13]string{"", "януари", "февруари", "март", "април", "май", "юни", "юли",
		"август", "септември", "октомври", "ноември", "декември"},
	dowName: [7]string{"неделя", "понеделник", "вторник", "сряда", "четвъртък", "петък", "събота"},
}

// bgAt puts the preposition before a time; times are read as numbers
// ("в два и половина"), so it is always "в".
func bgAt(t string) string { return "в " + t }

// bgPrep returns the preposition before a weekday name.
func bgPrep(day string) string {
	if strings.HasPrefix(day, "в") || strings.HasPrefix(day, "ф") {
		return "във"
	}
	return "в"
}

func joinList(items []string, conj string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " " + conj + " " + items[len(items)-1]
}

func joinListBG(items []string) string { return joinList(items, "и") }

func (w *words) list(items []string) string { return joinList(items, w.and) }

func ints(vals []int, format func(int) string) []string {
	out := make([]string, len(vals))
	for i, v := range vals {
		out[i] = format(v)
	}
	return out
}

func plain(v int) string { return fmt.Sprint(v) }
func two(v int) string   { return fmt.Sprintf("%02d", v) }

// Explain describes the schedule in plain language.
func (s *Schedule) Explain(lang Lang) string {
	w := &en
	if lang == Bulgarian {
		w = &bg
	}
	var segs []string
	sec, min, hour := &s.Fields[Second], &s.Fields[Minute], &s.Fields[Hour]
	secV, secOne := sec.Single()
	minV, minOne := min.Single()
	hourVals := hour.Values()
	allSingle := minOne && secOne && onlyValues(hour) && len(hourVals) <= 8
	if allSingle {
		times := make([]string, len(hourVals))
		for i, h := range hourVals {
			times[i] = fmt.Sprintf("%02d:%02d", h, minV)
			if secV != 0 {
				times[i] += fmt.Sprintf(":%02d", secV)
			}
		}
		segs = append(segs, w.at(w.list(times)))
	} else {
		if !(secOne && secV == 0) {
			segs = append(segs, describeUnit(w, sec))
		}
		if !(min.Full() && len(segs) > 0) {
			segs = append(segs, describeUnit(w, min))
		}
		switch {
		case hour.Full():
			if minOne {
				segs = append(segs, w.pastEveryHour)
			}
		default:
			segs = append(segs, describeHours(w, hour))
		}
	}
	days := describeDays(w, s)
	if days == "" && allSingle {
		days = w.everyDay
	}
	if days != "" {
		segs = append(segs, days)
	}
	if m := describeMonths(w, &s.Fields[Month]); m != "" {
		segs = append(segs, m)
	}
	out := strings.Join(segs, ", ")
	// Join "at minute 30, past every hour" without the comma.
	out = strings.Replace(out, ", "+w.pastEveryHour, " "+w.pastEveryHour, 1)
	return capitalize(out)
}

func onlyValues(f *Field) bool {
	for _, p := range f.Parts {
		if p.Kind != PartValue {
			return false
		}
	}
	return true
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = []rune(strings.ToUpper(string(r[0])))[0]
	return string(r)
}

// describeUnit describes a seconds or minutes field.
func describeUnit(w *words, f *Field) string {
	if f.Full() {
		if f.Kind == Second {
			return w.everySecond
		}
		return w.everyMinute
	}
	if len(f.Parts) == 1 {
		p := f.Parts[0]
		switch p.Kind {
		case PartStepAll:
			if 60%p.Step == 0 {
				return w.everyN(p.Step, f.Kind)
			}
		case PartRange:
			every := w.everyMinute
			if f.Kind == Second {
				every = w.everySecond
			}
			return every + " " + w.fromThrough(plain(p.Start), plain(p.End))
		case PartStepRange:
			return w.everyN(p.Step, f.Kind) + " " + w.fromThrough(plain(p.Start), plain(p.End))
		case PartStepFrom:
			return w.everyN(p.Step, f.Kind) + " " + w.startingAt(f.Kind, plain(p.Start))
		}
	}
	vals := f.Values()
	if f.Kind == Second {
		return w.atSecond(w.list(ints(vals, plain)), len(vals) > 1)
	}
	return w.atMinute(w.list(ints(vals, plain)), len(vals) > 1)
}

func describeHours(w *words, f *Field) string {
	hh := func(h int) string { return fmt.Sprintf("%02d:00", h) }
	end := func(h int) string { return fmt.Sprintf("%02d:59", h) }
	if len(f.Parts) == 1 {
		p := f.Parts[0]
		switch p.Kind {
		case PartValue:
			return w.between(hh(p.Start), end(p.Start))
		case PartRange:
			return w.between(hh(p.Start), end(p.End))
		case PartStepAll:
			return w.everyN(p.Step, Hour)
		case PartStepRange:
			return w.everyN(p.Step, Hour) + " " + w.between(hh(p.Start), end(p.End))
		case PartStepFrom:
			return w.everyN(p.Step, Hour) + " " + w.startingAt(Hour, hh(p.Start))
		}
	}
	return w.inHours(w.list(ints(f.Values(), hh)))
}

func describeDays(w *words, s *Schedule) string {
	domR, dowR := s.DayRestricted()
	domF, dowF := &s.Fields[DayOfMonth], &s.Fields[DayOfWeek]
	var dom, dow string
	if !domF.Full() {
		dom = describeDOM(w, domF)
	}
	if !dowF.Full() {
		dow = describeDOW(w, dowF)
	}
	if domR && dowR {
		if dom == "" || dow == "" {
			return "" // one side matches every day, so every day matches
		}
		return dom + " " + w.or + " " + dow
	}
	if dom != "" && dow != "" {
		return dom + ", " + dow
	}
	if dom != "" {
		return dom
	}
	return dow
}

func describeDOM(w *words, f *Field) string {
	if len(f.Parts) == 1 {
		p := f.Parts[0]
		switch p.Kind {
		case PartRange:
			return w.domRange(p.Start, p.End)
		case PartStepAll, PartStepFrom:
			return w.domEveryN(p.Step, p.Start)
		}
	}
	vals := f.Values()
	return w.dom(w.list(ints(vals, plain)), len(vals) > 1)
}

func describeDOW(w *words, f *Field) string {
	if len(f.Parts) == 1 && f.Parts[0].Kind == PartRange {
		p := f.Parts[0]
		e := p.End
		if e == 7 {
			e = 0
		}
		return w.dowRange(w.dowName[p.Start], w.dowName[e])
	}
	vals := f.Values()
	// Show Sunday last when the list wraps the week like 1,3,0 => Mon, Wed, Sun.
	names := ints(vals, func(v int) string { return w.dowName[v] })
	if len(vals) > 1 && vals[0] == 0 {
		names = append(names[1:], names[0])
	}
	return w.dowList(names)
}

func describeMonths(w *words, f *Field) string {
	if f.Full() {
		return ""
	}
	if len(f.Parts) == 1 && f.Parts[0].Kind == PartRange {
		p := f.Parts[0]
		return w.monthRange(w.monthName[p.Start], w.monthName[p.End])
	}
	return w.months(w.list(ints(f.Values(), func(v int) string { return w.monthName[v] })))
}
