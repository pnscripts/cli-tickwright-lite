package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func fixedNow() time.Time { return time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC) }

func exec(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(args, &out, &errb, fixedNow)
	return code, out.String(), errb.String()
}

func TestExitCodes(t *testing.T) {
	cases := []struct {
		args []string
		code int
	}{
		{[]string{}, exitUsage},
		{[]string{"help"}, exitOK},
		{[]string{"version"}, exitOK},
		{[]string{"validate", "0 9 * * 1-5"}, exitOK},
		{[]string{"validate", "0 25 * * *"}, exitInvalid},
		{[]string{"validate"}, exitUsage},
		{[]string{"explain", "@daily"}, exitOK},
		{[]string{"explain", "@daily", "--lang", "fr"}, exitUsage},
		{[]string{"next", "0 9 * * *", "--tz", "Mars/Olympus"}, exitUsage},
		{[]string{"next", "0 0 30 2 *"}, exitInvalid},
		{[]string{"next", "0 9 * * *", "-n", "0"}, exitUsage},
		{[]string{"frobnicate", "x"}, exitUsage},
		{[]string{"next", "@daily", "--tz", "europe/sofia"}, exitOK},
		{[]string{"next", "@daily", "--tz", "america/new_york"}, exitOK},
		{[]string{"next", "@daily", "--tz", "utc"}, exitOK},
		{[]string{"validate", "@every 5m"}, exitInvalid},
	}
	for _, c := range cases {
		code, out, errs := exec(c.args...)
		if code != c.code {
			t.Errorf("%v: exit %d, want %d\nstdout: %s\nstderr: %s", c.args, code, c.code, out, errs)
		}
	}
}

func TestUnquotedFieldsAndFlagsAnywhere(t *testing.T) {
	code, out, _ := exec("explain", "--lang", "en", "0", "9", "*", "*", "1-5", "--color", "never")
	if code != 0 || !strings.Contains(out, "At 09:00, Monday through Friday") {
		t.Fatalf("got %d %q", code, out)
	}
}

func TestNextJSONSofia(t *testing.T) {
	code, out, _ := exec("next", "30 3 * * *", "--tz", "Europe/Sofia", "-n", "3", "--json")
	if code != 0 {
		t.Fatal(out)
	}
	var v struct {
		Timezone string
		Runs     []struct{ Local, UTC, Note string }
	}
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatal(err)
	}
	want := []string{"2026-03-28T03:30:00+02:00", "2026-03-29T04:30:00+03:00", "2026-03-30T03:30:00+03:00"}
	for i, w := range want {
		if v.Runs[i].Local != w {
			t.Errorf("run %d = %s, want %s", i, v.Runs[i].Local, w)
		}
	}
	if v.Runs[1].Note != "shifted" || v.Timezone != "Europe/Sofia" {
		t.Errorf("unexpected %+v", v)
	}
}

func TestFromFlag(t *testing.T) {
	_, out, _ := exec("next", "@hourly", "--tz", "Europe/Sofia", "--from", "2026-10-25 02:30", "-n", "3", "--json")
	if !strings.Contains(out, `"local": "2026-10-25T03:00:00+03:00"`) ||
		!strings.Contains(out, `"local": "2026-10-25T03:00:00+02:00"`) {
		t.Errorf("expected both passes of 03:00, got %s", out)
	}
}

func TestExplainBothLanguages(t *testing.T) {
	_, out, _ := exec("explain", "15 10 * * 2", "--color", "never")
	if !strings.Contains(out, "EN  At 10:15, on Tuesday") || !strings.Contains(out, "BG  В 10:15, във вторник") {
		t.Errorf("got %q", out)
	}
}
