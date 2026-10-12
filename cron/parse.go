// Package cron parses, explains and schedules cron expressions.
//
// Supported syntax: 5 fields (minute hour day-of-month month day-of-week),
// 6 fields (a leading seconds field) and the macros @yearly, @annually,
// @monthly, @weekly, @daily, @midnight and @hourly.
package cron

import (
	"fmt"
	"strconv"
	"strings"
)

// FieldKind identifies one of the six cron fields.
type FieldKind int

const (
	Second FieldKind = iota
	Minute
	Hour
	DayOfMonth
	Month
	DayOfWeek
)

var fieldNames = [...]string{"second", "minute", "hour", "day-of-month", "month", "day-of-week"}

func (k FieldKind) String() string { return fieldNames[k] }

type bounds struct{ min, max int }

var fieldBounds = [...]bounds{{0, 59}, {0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 7}}

var monthNames = map[string]int{"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4, "MAY": 5, "JUN": 6,
	"JUL": 7, "AUG": 8, "SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12}
var dowNames = map[string]int{"SUN": 0, "MON": 1, "TUE": 2, "WED": 3, "THU": 4, "FRI": 5, "SAT": 6}

// PartKind describes the shape of one comma-separated element of a field.
type PartKind int

const (
	PartAll       PartKind = iota // * or ?
	PartValue                     // 5
	PartRange                     // 1-5
	PartStepAll                   // */15
	PartStepRange                 // 1-30/5
	PartStepFrom                  // 5/15 (from 5 to the field maximum)
)

// Part is one comma-separated element of a field, kept for explanations.
type Part struct {
	Kind             PartKind
	Start, End, Step int
}

// Field is a parsed field: its source text, its parts and the expanded set.
type Field struct {
	Kind   FieldKind
	Source string
	Parts  []Part
	Set    [64]bool
	// Star is true when the field text starts with * or ?. Cron uses this
	// to decide how day-of-month and day-of-week combine.
	Star bool
}

// Values returns the matching values in ascending order.
func (f *Field) Values() []int {
	var out []int
	b := fieldBounds[f.Kind]
	for v := b.min; v <= b.max; v++ {
		if f.Set[v] {
			out = append(out, v)
		}
	}
	return out
}

// Full reports whether every value of the field's range matches.
func (f *Field) Full() bool {
	b := fieldBounds[f.Kind]
	if f.Kind == DayOfWeek {
		b.max = 6
	}
	for v := b.min; v <= b.max; v++ {
		if !f.Set[v] {
			return false
		}
	}
	return true
}

// Single returns the only value of the field if it has exactly one.
func (f *Field) Single() (int, bool) {
	v := f.Values()
	if len(v) == 1 {
		return v[0], true
	}
	return 0, false
}

// Schedule is a parsed cron expression.
type Schedule struct {
	Source     string
	Macro      string // the macro used, e.g. "@daily", or ""
	HasSeconds bool   // 6-field form
	Fields     [6]Field
}

// ParseError describes why an expression is invalid.
type ParseError struct {
	Field FieldKind
	Text  string
	Msg   string
	Hint  string
}

func (e *ParseError) Error() string {
	s := fmt.Sprintf("%s field %q: %s", e.Field, e.Text, e.Msg)
	if e.Hint != "" {
		s += " (" + e.Hint + ")"
	}
	return s
}

var macros = map[string]string{
	"@yearly":   "0 0 1 1 *",
	"@annually": "0 0 1 1 *",
	"@monthly":  "0 0 1 * *",
	"@weekly":   "0 0 * * 0",
	"@daily":    "0 0 * * *",
	"@midnight": "0 0 * * *",
	"@hourly":   "0 * * * *",
}

// MacroNames lists the supported macros in a stable order.
var MacroNames = []string{"@yearly", "@annually", "@monthly", "@weekly", "@daily", "@midnight", "@hourly"}

// Parse parses a 5-field, 6-field or macro cron expression.
func Parse(expr string) (*Schedule, error) {
	src := strings.TrimSpace(expr)
	if src == "" {
		return nil, fmt.Errorf("empty expression")
	}
	s := &Schedule{Source: src}
	text := src
	if strings.HasPrefix(src, "@") {
		m := strings.ToLower(src)
		exp, ok := macros[m]
		if !ok {
			if m == "@reboot" {
				return nil, fmt.Errorf("@reboot runs once at start-up and has no schedule to compute")
			}
			if strings.HasPrefix(m, "@every") {
				return nil, fmt.Errorf("%q is a Go/robfig extension, not cron syntax (for every 5 minutes write */5 * * * *)", src)
			}
			return nil, fmt.Errorf("unknown macro %q (supported: %s)", src, strings.Join(MacroNames, ", "))
		}
		s.Macro = m
		text = exp
	}
	parts := strings.Fields(text)
	switch len(parts) {
	case 5:
		parts = append([]string{"0"}, parts...)
	case 6:
		s.HasSeconds = true
	default:
		return nil, fmt.Errorf("expected 5 fields (minute hour day-of-month month day-of-week) or 6 fields (with seconds first), got %d", len(parts))
	}
	for i, p := range parts {
		f, err := parseField(FieldKind(i), p)
		if err != nil {
			return nil, err
		}
		s.Fields[i] = *f
	}
	// Day-of-week 7 is Sunday.
	if s.Fields[DayOfWeek].Set[7] {
		s.Fields[DayOfWeek].Set[0] = true
		s.Fields[DayOfWeek].Set[7] = false
	}
	return s, nil
}

func parseField(kind FieldKind, text string) (*Field, error) {
	f := &Field{Kind: kind, Source: text}
	f.Star = strings.HasPrefix(text, "*") || strings.HasPrefix(text, "?")
	b := fieldBounds[kind]
	for _, raw := range strings.Split(text, ",") {
		if raw == "" {
			return nil, &ParseError{kind, text, "empty list element", "remove the extra comma"}
		}
		p, err := parsePart(kind, raw, b)
		if err != nil {
			return nil, err
		}
		f.Parts = append(f.Parts, p)
		step := p.Step
		if step == 0 {
			step = 1
		}
		for v := p.Start; v <= p.End; v += step {
			f.Set[v] = true
		}
	}
	return f, nil
}

func parsePart(kind FieldKind, raw string, b bounds) (Part, error) {
	perr := func(msg, hint string) (Part, error) {
		return Part{}, &ParseError{kind, raw, msg, hint}
	}
	if strings.ContainsAny(raw, "LW#") && (kind == DayOfMonth || kind == DayOfWeek) {
		up := strings.ToUpper(raw)
		if !isName(up, kind) {
			hint := "standard cron has no last-day or nth-weekday syntax"
			if strings.Contains(raw, "#") {
				hint = "for the 2nd Monday, run on days 8-14 (day-of-week *) and check the weekday inside the job"
			} else if strings.Contains(up, "L") {
				hint = "for the last day of the month, run on days 28-31 and check that tomorrow is the 1st inside the job"
			}
			return perr("the Quartz extensions L, W and # are not supported", hint)
		}
	}
	rng, stepText, hasStep := strings.Cut(raw, "/")
	step := 0
	if hasStep {
		n, err := strconv.Atoi(stepText)
		if err != nil || n <= 0 {
			return perr("step must be a positive whole number", "")
		}
		step = n
	}
	if rng == "*" || rng == "?" {
		if rng == "?" && kind != DayOfMonth && kind != DayOfWeek {
			return perr("? is only allowed in the day-of-month and day-of-week fields", "use *")
		}
		if hasStep {
			end := b.max
			if kind == DayOfWeek {
				end = 6
			}
			return Part{PartStepAll, b.min, end, step}, nil
		}
		end := b.max
		if kind == DayOfWeek {
			end = 6
		}
		return Part{PartAll, b.min, end, 0}, nil
	}
	lo, hi, isRange := strings.Cut(rng, "-")
	start, err := parseValue(kind, lo, b)
	if err != nil {
		return Part{}, err
	}
	if isRange {
		end, err := parseValue(kind, hi, b)
		if err != nil {
			return Part{}, err
		}
		if start > end {
			hint := fmt.Sprintf("ranges cannot wrap around; write %d-%d,%d-%d", start, b.max, b.min, end)
			if kind == DayOfWeek {
				hint = fmt.Sprintf("ranges cannot wrap around; write %d-6,0-%d", start, end)
			}
			return perr(fmt.Sprintf("range start %d is greater than end %d", start, end), hint)
		}
		if hasStep {
			return Part{PartStepRange, start, end, step}, nil
		}
		return Part{PartRange, start, end, 0}, nil
	}
	if hasStep {
		end := b.max
		if kind == DayOfWeek {
			end = 6
		}
		return Part{PartStepFrom, start, end, step}, nil
	}
	return Part{PartValue, start, start, 0}, nil
}

func isName(up string, kind FieldKind) bool {
	for _, piece := range strings.FieldsFunc(up, func(r rune) bool { return r == '-' || r == '/' }) {
		if _, ok := dowNames[piece]; ok && kind == DayOfWeek {
			continue
		}
		if _, err := strconv.Atoi(piece); err == nil {
			continue
		}
		return false
	}
	return true
}

func parseValue(kind FieldKind, s string, b bounds) (int, error) {
	up := strings.ToUpper(s)
	if kind == Month {
		if v, ok := monthNames[up]; ok {
			return v, nil
		}
	}
	if kind == DayOfWeek {
		if v, ok := dowNames[up]; ok {
			return v, nil
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		hint := ""
		switch kind {
		case Month:
			hint = "use 1-12 or JAN-DEC"
		case DayOfWeek:
			hint = "use 0-7 (0 and 7 are Sunday) or SUN-SAT"
		}
		return 0, &ParseError{kind, s, "not a number or known name", hint}
	}
	if n < b.min || n > b.max {
		return 0, &ParseError{kind, s, fmt.Sprintf("value %d is outside %d-%d", n, b.min, b.max), ""}
	}
	return n, nil
}

// Normalized returns the expression in 6-field form.
func (s *Schedule) Normalized() string {
	parts := make([]string, 6)
	for i := range s.Fields {
		parts[i] = s.Fields[i].Source
	}
	return strings.Join(parts, " ")
}

// DayRestricted reports whether day-of-month and day-of-week are restricted.
func (s *Schedule) DayRestricted() (dom, dow bool) {
	return !s.Fields[DayOfMonth].Star, !s.Fields[DayOfWeek].Star
}
