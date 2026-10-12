// Command tickwright-lite validates and explains cron expressions and lists
// their next run times. Tickwright Lite by PN Scripts, MIT licence.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	_ "time/tzdata" // embed the zone database so binaries work everywhere

	"github.com/pnscripts/tickwright-lite/cron"
)

var version = "1.0.0"

const (
	exitOK      = 0
	exitInvalid = 1
	exitUsage   = 2
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, time.Now)) }

const usage = `tickwright-lite %s: cron expressions, checked and explained

Usage:
  tickwright validate <expr>             exit 0 if valid, 1 if not
  tickwright explain  <expr> [--lang en|bg|both]
  tickwright next     <expr> [-n 10] [--tz Zone] [--from TIME]
  tickwright version

Expressions: 5 fields (min hour dom month dow), 6 fields (sec first),
or @yearly @annually @monthly @weekly @daily @midnight @hourly.
Quote expressions so the shell does not expand *.

Common flags:
  --tz Zone        IANA time zone (default: local zone, e.g. Europe/Sofia)
  --from TIME      start after TIME: RFC 3339 or "2006-01-02 15:04" in --tz
  --json           machine-readable output
  --color MODE     auto, always or never (NO_COLOR is respected)

Exit codes: 0 ok, 1 invalid expression, 2 usage error.
`

type opts struct {
	n      int
	tz     string
	from   string
	lang   string
	json   bool
	color  string
	strict bool
}

type ui struct {
	out   io.Writer
	color bool
}

func (u ui) c(code, s string) string {
	if !u.color {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}
func (u ui) bold(s string) string   { return u.c("1", s) }
func (u ui) dim(s string) string    { return u.c("2", s) }
func (u ui) green(s string) string  { return u.c("32", s) }
func (u ui) yellow(s string) string { return u.c("33", s) }
func (u ui) red(s string) string    { return u.c("31", s) }
func (u ui) cyan(s string) string   { return u.c("36", s) }

// splitArgs separates flags from positionals so flags can come anywhere.
func splitArgs(args []string, valued map[string]bool) (flags, pos []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		if strings.HasPrefix(a, "-") && len(a) > 1 && !isNumberish(a) {
			flags = append(flags, a)
			name := strings.TrimLeft(a, "-")
			if !strings.Contains(name, "=") && valued[name] && i+1 < len(args) {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}
		pos = append(pos, a)
	}
	return
}

func isNumberish(a string) bool { return len(a) > 1 && a[1] >= '0' && a[1] <= '9' }

func run(args []string, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) == 0 {
		fmt.Fprintf(stderr, usage, version)
		return exitUsage
	}
	cmd := args[0]
	switch cmd {
	case "-h", "--help", "help":
		fmt.Fprintf(stdout, usage, version)
		return exitOK
	case "version", "--version":
		fmt.Fprintln(stdout, "tickwright-lite", version)
		return exitOK
	}
	o := opts{}
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.IntVar(&o.n, "n", 10, "number of run times")
	fs.StringVar(&o.tz, "tz", "", "IANA time zone")
	fs.StringVar(&o.from, "from", "", "start time")
	fs.StringVar(&o.lang, "lang", "both", "en, bg or both")
	fs.BoolVar(&o.json, "json", false, "JSON output")
	fs.StringVar(&o.color, "color", "auto", "auto, always or never")
	flags, pos := splitArgs(args[1:], map[string]bool{"n": true, "tz": true, "from": true, "lang": true, "color": true})
	if err := fs.Parse(flags); err != nil {
		return exitUsage
	}
	u := ui{out: stdout, color: wantColor(o.color, stdout)}
	if o.n < 1 || o.n > 1000 {
		fmt.Fprintln(stderr, "error: -n must be between 1 and 1000")
		return exitUsage
	}
	loc := localZone()
	if o.tz != "" {
		l, err := loadZone(o.tz)
		if err != nil {
			fmt.Fprintf(stderr, "error: unknown time zone %q (use an IANA name such as Europe/Sofia, America/New_York or UTC)\n", o.tz)
			return exitUsage
		}
		loc = l
	}
	from := now()
	if o.from != "" {
		t, err := parseFrom(o.from, loc)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return exitUsage
		}
		from = t
	}
	switch cmd {
	case "validate":
		return cmdValidate(u, stderr, joinExpr(pos), o)
	case "explain":
		return cmdExplain(u, stderr, joinExpr(pos), o)
	case "next":
		return cmdNext(u, stderr, joinExpr(pos), o, loc, from)
	}
	fmt.Fprintf(stderr, "error: unknown command %q\n\n", cmd)
	fmt.Fprintf(stderr, usage, version)
	return exitUsage
}

// localZone returns the system zone with its IANA name when it can be
// found (TZ or /etc/localtime), so output says "Europe/Sofia", not "Local".
func localZone() *time.Location {
	name := os.Getenv("TZ")
	if name == "" {
		if target, err := os.Readlink("/etc/localtime"); err == nil {
			if i := strings.Index(target, "zoneinfo/"); i >= 0 {
				name = target[i+len("zoneinfo/"):]
			}
		}
	}
	if name != "" {
		if l, err := time.LoadLocation(strings.TrimPrefix(name, ":")); err == nil {
			return l
		}
	}
	return time.Local
}

// loadZone accepts IANA names in any letter case: "europe/sofia" and
// "america/new_york" resolve to Europe/Sofia and America/New_York.
func loadZone(name string) (*time.Location, error) {
	if l, err := time.LoadLocation(name); err == nil {
		return l, nil
	}
	if l, err := time.LoadLocation(strings.ToUpper(name)); err == nil {
		return l, nil
	}
	b := []byte(strings.ToLower(name))
	up := true
	for i, c := range b {
		if up && c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
		up = c == '/' || c == '_' || c == '-'
	}
	return time.LoadLocation(string(b))
}

func joinExpr(pos []string) string { return strings.Join(pos, " ") }

func wantColor(mode string, w io.Writer) bool {
	switch mode {
	case "always":
		return true
	case "never":
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func parseFrom(s string, loc *time.Location) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02T15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot read --from %q (use RFC 3339 or \"2006-01-02 15:04\")", s)
}

func parse(u ui, stderr io.Writer, expr string, o opts) (*cron.Schedule, int) {
	if expr == "" {
		fmt.Fprintln(stderr, "error: missing cron expression")
		return nil, exitUsage
	}
	s, err := cron.Parse(expr)
	if err != nil {
		if o.json {
			writeJSON(u.out, map[string]any{"expression": expr, "valid": false, "error": err.Error()})
		} else {
			fmt.Fprintf(u.out, "%s %s\n  %s\n", u.red("invalid:"), expr, err)
		}
		return nil, exitInvalid
	}
	return s, exitOK
}

func writeJSON(w io.Writer, v any) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func form(s *cron.Schedule) string {
	switch {
	case s.Macro != "":
		return "macro " + s.Macro + " = " + strings.SplitN(s.Normalized(), " ", 2)[1]
	case s.HasSeconds:
		return "6 fields, with seconds"
	}
	return "5 fields"
}

func cmdValidate(u ui, stderr io.Writer, expr string, o opts) int {
	s, code := parse(u, stderr, expr, o)
	if s == nil {
		return code
	}
	if o.json {
		writeJSON(u.out, map[string]any{"expression": expr, "valid": true, "normalized": s.Normalized(), "form": form(s)})
		return exitOK
	}
	fmt.Fprintf(u.out, "%s %s  %s\n", u.green("valid:"), expr, u.dim("("+form(s)+")"))
	return exitOK
}

func cmdExplain(u ui, stderr io.Writer, expr string, o opts) int {
	s, code := parse(u, stderr, expr, o)
	if s == nil {
		return code
	}
	langs := []cron.Lang{cron.English, cron.Bulgarian}
	switch o.lang {
	case "en":
		langs = langs[:1]
	case "bg":
		langs = langs[1:]
	case "both":
	default:
		fmt.Fprintln(stderr, "error: --lang must be en, bg or both")
		return exitUsage
	}
	if o.json {
		m := map[string]any{"expression": expr, "normalized": s.Normalized(), "form": form(s)}
		for _, l := range langs {
			m[string(l)] = s.Explain(l)
		}
		writeJSON(u.out, m)
		return exitOK
	}
	fmt.Fprintf(u.out, "%s  %s\n", u.bold(expr), u.dim("("+form(s)+")"))
	for _, l := range langs {
		fmt.Fprintf(u.out, "  %s  %s\n", u.cyan(strings.ToUpper(string(l))), s.Explain(l))
	}
	return exitOK
}

func noteText(n string) string {
	switch n {
	case "shifted":
		return "moved forward: wall time skipped by DST"
	case "repeated":
		return "second pass of the repeated DST hour"
	}
	return ""
}

func cmdNext(u ui, stderr io.Writer, expr string, o opts, loc *time.Location, from time.Time) int {
	s, code := parse(u, stderr, expr, o)
	if s == nil {
		return code
	}
	runs, err := s.Next(from, loc, o.n)
	if err != nil {
		fmt.Fprintf(u.out, "%s %s\n", u.yellow("never runs:"), err)
		return exitInvalid
	}
	if o.json {
		type jr struct {
			Local string `json:"local"`
			UTC   string `json:"utc"`
			Note  string `json:"note,omitempty"`
		}
		var list []jr
		for _, r := range runs {
			list = append(list, jr{r.Time.Format(time.RFC3339), r.Time.UTC().Format(time.RFC3339), r.Note})
		}
		writeJSON(u.out, map[string]any{"expression": expr, "timezone": loc.String(), "from": from.In(loc).Format(time.RFC3339), "runs": list})
		return exitOK
	}
	fmt.Fprintf(u.out, "%s  %s\n", u.bold(expr), u.dim(s.Explain(cron.English)))
	fmt.Fprintf(u.out, "%s  %s, after %s\n\n", u.dim("zone"), loc, from.In(loc).Format("2006-01-02 15:04:05 MST"))
	for i, r := range runs {
		lt := r.Time.Format("Mon 2006-01-02 15:04:05")
		zone := r.Time.Format("MST")
		line := fmt.Sprintf("%3d  %s %-5s %s", i+1, lt, zone, u.dim(r.Time.UTC().Format("2006-01-02 15:04Z")))
		if r.Note != "" {
			line += "  " + u.yellow(noteText(r.Note))
		}
		fmt.Fprintln(u.out, line)
	}
	return exitOK
}

// wrap breaks text into lines of at most width visible characters
// (colour codes are not counted); following lines start with indent.
func wrap(text string, width int, indent string) string {
	words := strings.Split(text, " ")
	var b strings.Builder
	col := 0
	for i, w := range words {
		vis := visibleLen(w)
		if i > 0 && col+1+vis > width {
			b.WriteString("\n" + indent)
			col = 0
		} else if i > 0 {
			b.WriteByte(' ')
			col++
		}
		b.WriteString(w)
		col += vis
	}
	return b.String()
}

func visibleLen(s string) int {
	n, esc := 0, false
	for _, r := range s {
		switch {
		case r == '\x1b':
			esc = true
		case esc && r == 'm':
			esc = false
		case !esc:
			n++
		}
	}
	return n
}
