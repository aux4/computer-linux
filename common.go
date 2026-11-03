package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ---------------------------------------------------------------------------
// Output

type obj = map[string]any

func encode(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return strings.TrimRight(buf.String(), "\n")
}

// cmdError is a failure carried back to main: a JSON object printed on stderr, exit 1.
type cmdError struct {
	fields obj
}

func (e *cmdError) Error() string {
	if s, ok := e.fields["error"].(string); ok {
		return s
	}
	return "error"
}

func errf(format string, args ...any) *cmdError {
	return &cmdError{fields: obj{"error": fmt.Sprintf(format, args...)}}
}

func errObj(fields obj) *cmdError {
	return &cmdError{fields: fields}
}

// unsupported is the standard "this platform/session can't do it" error.
func unsupported(command, reason, hint string) *cmdError {
	o := obj{
		"error":       fmt.Sprintf("%s is not supported here: %s", command, reason),
		"unsupported": true,
		"command":     command,
		"reason":      reason,
		"session":     sessionType(),
	}
	if hint != "" {
		o["hint"] = hint
	}
	return &cmdError{fields: o}
}

// ---------------------------------------------------------------------------
// Environment seams (contract test seams)

func envOn(names ...string) bool {
	for _, n := range names {
		v := strings.ToLower(strings.TrimSpace(os.Getenv(n)))
		if v != "" && v != "0" && v != "false" && v != "no" {
			return true
		}
	}
	return false
}

func dryRun() bool {
	return envOn("AUX4_COMPUTER_DRY_RUN", "AUX4_COMPUTER_LINUX_DRY_RUN")
}

func denyPermissions() bool {
	return envOn("AUX4_COMPUTER_DENY_PERMISSIONS", "AUX4_COMPUTER_LINUX_DENY_PERMISSIONS")
}

func simulateBackgroundUnavailable() bool {
	v := os.Getenv("AUX4_COMPUTER_SIMULATE_BACKGROUND")
	if v == "" {
		v = os.Getenv("AUX4_COMPUTER_LINUX_SIMULATE_BACKGROUND")
	}
	return strings.EqualFold(strings.TrimSpace(v), "unavailable")
}

// ---------------------------------------------------------------------------
// Shared state dir: stop switch + action log

func stateDir() string {
	if d := os.Getenv("AUX4_COMPUTER_STATE_DIR"); d != "" {
		return d
	}
	if d := os.Getenv("XDG_STATE_HOME"); d != "" {
		return filepath.Join(d, "aux4-computer")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "aux4-computer")
}

func stopFile() string {
	if f := os.Getenv("AUX4_COMPUTER_STOP_FILE"); f != "" {
		return f
	}
	return filepath.Join(stateDir(), "STOP")
}

func logFile() string {
	if f := os.Getenv("AUX4_COMPUTER_LOG_FILE"); f != "" {
		return f
	}
	return filepath.Join(stateDir(), "actions.jsonl")
}

func logMaxBytes() int64 {
	if v, err := strconv.ParseInt(os.Getenv("AUX4_COMPUTER_LOG_MAX_BYTES"), 10, 64); err == nil && v > 0 {
		return v
	}
	return 10 * 1024 * 1024
}

// stopState reports whether the stop switch is on and why.
func stopState() (bool, string) {
	if envOn("AUX4_COMPUTER_STOP") {
		return true, "AUX4_COMPUTER_STOP=1 is set"
	}
	if _, err := os.Stat(stopFile()); err == nil {
		return true, stopFile() + " exists"
	}
	return false, ""
}

func stoppedError(command string) *cmdError {
	_, why := stopState()
	return errObj(obj{
		"command": command,
		"stopped": true,
		"error":   fmt.Sprintf("Stopped: aux4 computer linux is not sending input or changing state (%s). Run 'aux4 computer resume' (or 'aux4 computer linux resume') to allow it again.", why),
	})
}

func truncatePrivate(s string) string {
	r := []rune(s)
	if len(r) <= 16 {
		return s
	}
	return fmt.Sprintf("%s… (%d chars)", string(r[:16]), len(r))
}

var privateArgs = map[string]bool{"text": true, "value": true}

func logAction(command string, args map[string]string, app string, result obj, err *cmdError, started time.Time) {
	entry := obj{
		"ts":       time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		"provider": "linux",
		"command":  command,
		"ms":       time.Since(started).Milliseconds(),
	}
	a := obj{}
	for k, v := range args {
		if v == "" {
			continue
		}
		if privateArgs[k] {
			v = truncatePrivate(v)
		}
		a[k] = v
	}
	entry["args"] = a
	if app != "" {
		entry["app"] = app
	}
	if err != nil {
		entry["result"] = "error"
		entry["error"] = err.Error()
		if m, ok := err.fields["mode"]; ok {
			entry["mode"] = m
		}
	} else {
		entry["result"] = "ok"
		for _, k := range []string{"mode", "method", "fallbackReason"} {
			if v, ok := result[k]; ok {
				entry[k] = v
			}
		}
	}
	path := logFile()
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	if st, e := os.Stat(path); e == nil && st.Size() >= logMaxBytes() {
		_ = os.Rename(path, path+".1")
	}
	f, e := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if e != nil {
		return
	}
	defer f.Close()
	_ = f.Chmod(0o600)
	_, _ = f.WriteString(encode(entry) + "\n")
}

func readLogEntries(limit int) []any {
	read := func(p string) []any {
		f, err := os.Open(p)
		if err != nil {
			return nil
		}
		defer f.Close()
		var out []any
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
		for sc.Scan() {
			var m map[string]any
			if json.Unmarshal(sc.Bytes(), &m) == nil {
				out = append(out, m)
			}
		}
		return out
	}
	entries := read(logFile())
	if len(entries) < limit {
		entries = append(read(logFile()+".1"), entries...)
	}
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	if entries == nil {
		entries = []any{}
	}
	return entries
}

// ---------------------------------------------------------------------------
// Modes

func parseMode(m string) (string, *cmdError) {
	if m == "" {
		return "auto", nil
	}
	switch m {
	case "auto", "background", "foreground":
		return m, nil
	}
	return "", errf("Invalid --mode '%s': use auto, background or foreground", m)
}

// decideMode turns a requested mode + whether background is possible into the mode used.
// It returns the mode, the fallback reason (when auto fell back) or an error.
func decideMode(command, requested string, bgPossible bool, reason string) (string, string, *cmdError) {
	if simulateBackgroundUnavailable() && bgPossible {
		bgPossible = false
		reason = "background simulated as unavailable"
	}
	switch requested {
	case "foreground":
		return "foreground", "", nil
	case "background":
		if !bgPossible {
			return "", "", errObj(obj{"error": fmt.Sprintf("Background mode not possible for %s: %s", command, reason), "mode": "background"})
		}
		return "background", "", nil
	default:
		if bgPossible {
			return "background", "", nil
		}
		return "foreground", reason, nil
	}
}

// ---------------------------------------------------------------------------
// Parsing helpers

func parseFloatArg(name, v string) (float64, *cmdError) {
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return 0, errf("Invalid --%s '%s': use a number", name, v)
	}
	return f, nil
}

func parseIntArg(name, v string, def int) (int, *cmdError) {
	if strings.TrimSpace(v) == "" {
		return def, nil
	}
	i, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0, errf("Invalid --%s '%s': use a whole number", name, v)
	}
	return i, nil
}

func parsePositive(name, v string, def int) (int, *cmdError) {
	i, e := parseIntArg(name, v, def)
	if e != nil || i <= 0 {
		return 0, errf("Invalid --%s '%s': use a positive number", name, v)
	}
	return i, nil
}

func parseBool(name, v string, def bool) (bool, *cmdError) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "":
		return def, nil
	case "true", "yes", "1":
		return true, nil
	case "false", "no", "0":
		return false, nil
	}
	return false, errf("Invalid --%s '%s': use true or false", name, v)
}

var durRe = regexp.MustCompile(`^([0-9]*\.?[0-9]+)\s*(ms|s|m|h)?$`)

// parseDuration accepts 500ms, 10s, 2m, 1h or a plain number of seconds.
func parseDuration(name, v string, def time.Duration) (time.Duration, *cmdError) {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return def, nil
	}
	m := durRe.FindStringSubmatch(v)
	if m == nil {
		return 0, errf("Invalid --%s '%s': use a duration like 500ms, 10s or 2m", name, v)
	}
	n, _ := strconv.ParseFloat(m[1], 64)
	unit := time.Second
	switch m[2] {
	case "ms":
		unit = time.Millisecond
	case "m":
		unit = time.Minute
	case "h":
		unit = time.Hour
	}
	return time.Duration(n * float64(unit)), nil
}

func secs(d time.Duration) float64 {
	return round2(d.Seconds())
}

func round2(f float64) float64 {
	return float64(int64(f*100+sign(f)*0.5)) / 100
}

func round1(f float64) float64 {
	return float64(int64(f*10+sign(f)*0.5)) / 10
}

func sign(f float64) float64 {
	if f < 0 {
		return -1
	}
	return 1
}

// normalizeLoose lowercases and drops whitespace and commas: the wait / ocr --text match.
func normalizeLoose(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsSpace(r) || r == ',' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// normalizeTitle is used for menu / dialog button titles: case-insensitive, trailing ellipsis
// optional, curly apostrophes match straight ones, GTK mnemonics underscores ignored.
func normalizeTitle(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "’", "'")
	s = strings.ReplaceAll(s, "‘", "'")
	s = strings.TrimSuffix(s, "…")
	s = strings.TrimSuffix(s, "...")
	s = strings.ReplaceAll(s, "_", "")
	return strings.TrimSpace(s)
}

func containsFold(hay, needle string) bool {
	return strings.Contains(strings.ToLower(hay), strings.ToLower(needle))
}

func cut(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func point(x, y float64) obj {
	return obj{"x": round1(x), "y": round1(y)}
}

type rect struct {
	X, Y, W, H float64
}

func (r rect) obj() obj {
	return obj{"x": round1(r.X), "y": round1(r.Y), "width": round1(r.W), "height": round1(r.H)}
}

func (r rect) center() (float64, float64) {
	return r.X + r.W/2, r.Y + r.H/2
}

func (r rect) contains(x, y float64) bool {
	return x >= r.X && y >= r.Y && x < r.X+r.W && y < r.Y+r.H
}

func (r rect) empty() bool {
	return r.W <= 0 || r.H <= 0
}
