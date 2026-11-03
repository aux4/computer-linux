package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseCombo(t *testing.T) {
	cases := map[string]string{
		"cmd+s":         "ctrl+s",
		"ctrl+shift+s":  "ctrl+shift+s",
		"shift+ctrl+S":  "ctrl+shift+s",
		"alt+f4":        "alt+f4",
		"enter":         "enter",
		"super+e":       "super+e",
		"option+tab":    "alt+tab",
		"fn+cmd+a":      "ctrl+a",
	}
	for in, want := range cases {
		c, err := parseCombo(in)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if got := c.canonical(); got != want {
			t.Errorf("%s: got %s want %s", in, got, want)
		}
	}
	for _, bad := range []string{"", "hyper+x", "ctrl+nosuchkey"} {
		if _, err := parseCombo(bad); err == nil {
			t.Errorf("%q should fail", bad)
		}
	}
}

func TestAccel(t *testing.T) {
	cases := map[string]string{
		"n;<Alt>f:n;<Control>n":   "ctrl+n",
		";;<Primary><Shift>s":     "ctrl+shift+s",
		"<Control>q":              "ctrl+q",
		"b;<Alt>v:b;<Control>b":   "ctrl+b",
		"s;;":                     "",
		";;<Control>Return":       "ctrl+enter",
	}
	for in, want := range cases {
		if got := accelOf(in); got != want {
			t.Errorf("%q: got %q want %q", in, got, want)
		}
	}
}

func TestNormalize(t *testing.T) {
	if normalizeLoose("1,792") != normalizeLoose("1 792") || normalizeLoose("1792") != "1792" {
		t.Error("loose normalization should drop commas and spaces")
	}
	if normalizeTitle("Don’t Save…") != normalizeTitle("don't save") {
		t.Error("titles: curly apostrophes and ellipsis")
	}
	if normalizeTitle("_Save") != "save" {
		t.Error("titles: GTK mnemonic underscore")
	}
}

func TestRoles(t *testing.T) {
	if !roleMatches("push button", "AXButton") || !roleMatches("push button", "push_button") || !roleMatches("push button", "ROLE_PUSH_BUTTON") {
		t.Error("button role aliases")
	}
	if roleMatches("label", "AXButton") {
		t.Error("label is not a button")
	}
	if !roleMatches("text", "AXTextArea") || !roleMatches("check box", "AXCheckBox") || !roleMatches("frame", "AXWindow") {
		t.Error("mac role mapping")
	}
}

func TestDuration(t *testing.T) {
	cases := map[string]time.Duration{"500ms": 500 * time.Millisecond, "10s": 10 * time.Second, "2m": 2 * time.Minute, "3": 3 * time.Second, "1.5s": 1500 * time.Millisecond}
	for in, want := range cases {
		got, err := parseDuration("timeout", in, 0)
		if err != nil || got != want {
			t.Errorf("%s: got %v %v", in, got, err)
		}
	}
	if _, err := parseDuration("timeout", "soon", 0); err == nil {
		t.Error("bad duration accepted")
	}
}

func TestTruncatePrivate(t *testing.T) {
	if got := truncatePrivate("a long and private sentence typed by an agent"); got != "a long and priva… (45 chars)" {
		t.Errorf("got %q", got)
	}
	if truncatePrivate("short") != "short" {
		t.Error("short text kept")
	}
}

func TestDecideMode(t *testing.T) {
	m, r, e := decideMode("click", "auto", false, "why")
	if e != nil || m != "foreground" || r != "why" {
		t.Errorf("auto fallback: %s %s %v", m, r, e)
	}
	if _, _, e := decideMode("click", "background", false, "why"); e == nil {
		t.Error("background must not fall back")
	}
	if m, _, _ := decideMode("click", "auto", true, ""); m != "background" {
		t.Error("auto prefers background")
	}
	os.Setenv("AUX4_COMPUTER_SIMULATE_BACKGROUND", "unavailable")
	defer os.Unsetenv("AUX4_COMPUTER_SIMULATE_BACKGROUND")
	if m, r, _ := decideMode("click", "auto", true, ""); m != "foreground" || r != "background simulated as unavailable" {
		t.Errorf("simulated: %s %s", m, r)
	}
}

func TestGroupLinesAndMatch(t *testing.T) {
	words := []ocrWord{
		{text: "Total:", conf: 0.9, r: rect{10, 10, 40, 12}},
		{text: "1,792", conf: 0.8, r: rect{55, 11, 30, 12}},
		{text: "AC", conf: 0.95, r: rect{10, 60, 15, 12}},
	}
	lines := groupLines(words)
	if len(lines) != 2 || lines[0].text != "Total: 1,792" || lines[1].text != "AC" {
		t.Fatalf("lines: %+v", lines)
	}
	r, ok := matchSpan(lines[0], "1792")
	if !ok || r.X != 55 {
		t.Errorf("match span: %+v %v", r, ok)
	}
	ms := ocrMatches(lines, rect{100, 200, 0, 0}, 1, &rect{100, 200, 300, 300}, "1792")
	if len(ms) != 1 {
		t.Fatalf("filtered matches: %d", len(ms))
	}
	m := ms[0].(obj)
	if wc := m["match"].(obj)["windowCenter"].(obj); wc["x"] != 70.0 {
		t.Errorf("windowCenter %v", wc)
	}
}

func TestExecArgs(t *testing.T) {
	got := execArgs("mousepad %U", "/tmp/a.txt")
	if len(got) != 2 || got[1] != "/tmp/a.txt" {
		t.Errorf("%v", got)
	}
	got = execArgs("galculator", "")
	if len(got) != 1 {
		t.Errorf("%v", got)
	}
}

func TestStateDirAndStop(t *testing.T) {
	d := t.TempDir()
	os.Setenv("XDG_STATE_HOME", d)
	defer os.Unsetenv("XDG_STATE_HOME")
	if stopFile() != filepath.Join(d, "aux4-computer", "STOP") || logFile() != filepath.Join(d, "aux4-computer", "actions.jsonl") {
		t.Errorf("state paths: %s %s", stopFile(), logFile())
	}
	if s, _ := stopState(); s {
		t.Error("not stopped yet")
	}
	if _, e := cmdStop(nil); e != nil {
		t.Fatal(e)
	}
	if s, why := stopState(); !s || why == "" {
		t.Error("stopped after stop")
	}
	logAction("click", map[string]string{"x": "1", "text": "a long and private sentence typed by an agent"}, "galculator", nil, errf("nope"), time.Now())
	st, err := os.Stat(logFile())
	if err != nil || st.Mode().Perm() != 0o600 {
		t.Errorf("log file mode: %v %v", st, err)
	}
	es := readLogEntries(5)
	if len(es) != 1 || es[0].(map[string]any)["provider"] != "linux" {
		t.Errorf("entries %v", es)
	}
	if _, e := cmdResume(nil); e != nil {
		t.Fatal(e)
	}
	if s, _ := stopState(); s {
		t.Error("resumed")
	}
}
