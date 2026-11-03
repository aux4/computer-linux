package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jezek/xgb/xproto"
)

type desktopEntry struct {
	ID, Name, Exec, Path string
}

func xdgDataDirs() []string {
	var dirs []string
	if h := os.Getenv("XDG_DATA_HOME"); h != "" {
		dirs = append(dirs, h)
	} else if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".local", "share"))
	}
	d := os.Getenv("XDG_DATA_DIRS")
	if d == "" {
		d = "/usr/local/share:/usr/share"
	}
	dirs = append(dirs, strings.Split(d, ":")...)
	dirs = append(dirs, "/var/lib/flatpak/exports/share", "/var/lib/snapd/desktop")
	return dirs
}

func parseDesktop(path string) *desktopEntry {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	e := &desktopEntry{ID: strings.TrimSuffix(filepath.Base(path), ".desktop"), Path: path}
	sc := bufio.NewScanner(f)
	inMain := false
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(l, "[") {
			inMain = l == "[Desktop Entry]"
			continue
		}
		if !inMain {
			continue
		}
		if v, ok := strings.CutPrefix(l, "Name="); ok && e.Name == "" {
			e.Name = v
		}
		if v, ok := strings.CutPrefix(l, "Exec="); ok && e.Exec == "" {
			e.Exec = v
		}
	}
	if e.Exec == "" {
		return nil
	}
	return e
}

// findDesktop resolves an app given by desktop id, Name or executable name.
func findDesktop(q string) *desktopEntry {
	lq := strings.ToLower(strings.TrimSuffix(q, ".desktop"))
	var all []*desktopEntry
	for _, d := range xdgDataDirs() {
		files, _ := filepath.Glob(filepath.Join(d, "applications", "*.desktop"))
		for _, f := range files {
			if e := parseDesktop(f); e != nil {
				all = append(all, e)
			}
		}
	}
	for _, e := range all {
		if strings.ToLower(e.ID) == lq {
			return e
		}
	}
	for _, e := range all {
		if strings.ToLower(e.Name) == lq {
			return e
		}
	}
	for _, e := range all {
		f := strings.Fields(e.Exec)
		if len(f) > 0 && strings.ToLower(filepath.Base(f[0])) == lq {
			return e
		}
	}
	for _, e := range all {
		id := strings.ToLower(e.ID)
		if i := strings.LastIndex(id, "."); i >= 0 && id[i+1:] == lq {
			return e
		}
	}
	return nil
}

// execArgs expands a desktop Exec line with an optional file/url argument.
func execArgs(execLine string, target string) []string {
	var out []string
	used := false
	for _, f := range strings.Fields(execLine) {
		switch f {
		case "%f", "%F", "%u", "%U":
			if target != "" {
				out = append(out, target)
				used = true
			}
			continue
		case "%i", "%c", "%k", "%d", "%D", "%n", "%N", "%v", "%m":
			continue
		}
		out = append(out, strings.Trim(f, `"`))
	}
	if target != "" && !used {
		out = append(out, target)
	}
	return out
}

func launch(argv []string) (int, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	devnull, _ := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = devnull, devnull, devnull
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	go cmd.Wait() // reap if it exits while we are still running
	_ = cmd.Process.Release()
	return pid, nil
}

func cmdOpen(a map[string]string) (obj, *cmdError) {
	mode, e := parseMode(a["mode"])
	if e != nil {
		return nil, e
	}
	hidden, e := parseBool("hidden", a["hidden"], false)
	if e != nil {
		return nil, e
	}
	if hidden && mode == "foreground" {
		return nil, errf("--hidden true can't be combined with --mode foreground")
	}
	appQ, file, url := strings.TrimSpace(a["app"]), a["file"], a["url"]
	if appQ == "" && file == "" && url == "" {
		return nil, errf("Provide an app, --file or --url")
	}
	if file != "" && url != "" {
		return nil, errf("Give --file or --url, not both")
	}
	var waitFor time.Duration
	if a["wait"] != "" {
		if waitFor, e = parseDuration("wait", a["wait"], 0); e != nil {
			return nil, e
		}
	}
	if file != "" {
		abs, err := filepath.Abs(file)
		if err == nil {
			file = abs
		}
		if _, err := os.Stat(file); err != nil {
			return nil, errf("File not found: %s", a["file"])
		}
	}
	if url != "" && !strings.Contains(url, ":") {
		return nil, errf("Invalid --url '%s': include the scheme, e.g. https://aux4.io", url)
	}
	if dryRun() {
		m, r, err := decideMode("open", mode, true, "")
		if err != nil {
			return nil, err
		}
		o := obj{"opened": firstNonEmpty(appQ, file, url)}
		if appQ != "" {
			o["app"] = appQ
		}
		return withDryRun(o, m, r), nil
	}
	if e := requireX11("open"); e != nil && sessionType() == "none" {
		return nil, e
	}
	var x *xconn
	if sessionType() == "x11" || sessionType() == "xwayland" {
		x, _ = X("open")
	}
	var prevActive xproto.Window
	before := map[xproto.Window]bool{}
	if x != nil {
		prevActive = x.activeWindow()
		for _, w := range x.clientList() {
			before[w] = true
		}
	}
	target := firstNonEmpty(file, url)
	o := obj{"mode": "background"}
	if mode == "foreground" {
		o["mode"] = "foreground"
	}
	if file != "" {
		o["file"] = file
	}
	if url != "" {
		o["url"] = url
	}
	var argv []string
	var running *app
	if appQ != "" {
		if r, err := findApp(appQ, false); err == nil {
			running = r
		}
		if running != nil && target == "" {
			o["app"], o["pid"], o["alreadyRunning"], o["opened"] = running.Name, running.Pid, true, appQ
			if mode == "foreground" && x != nil {
				if w := running.frontWindow(x); w != 0 {
					_ = x.activate(w)
				}
			}
			if waitFor > 0 {
				return waitOpenWindow(x, o, running.Pid, "", waitFor, nil)
			}
			return o, nil
		}
		if d := findDesktop(appQ); d != nil {
			argv = execArgs(d.Exec, target)
			o["desktopId"] = d.ID
		} else if p, err := exec.LookPath(appQ); err == nil {
			argv = []string{p}
			if target != "" {
				argv = append(argv, target)
			}
		} else {
			return nil, errf("App not found: %s (no .desktop entry or executable with that name)", appQ)
		}
	} else {
		opener := ""
		for _, c := range []string{"xdg-open", "gio", "exo-open"} {
			if p, err := exec.LookPath(c); err == nil {
				opener = p
				break
			}
		}
		if opener == "" {
			return nil, errf("No opener found for %s: install xdg-utils, or pass --app", target)
		}
		if filepath.Base(opener) == "gio" {
			argv = []string{opener, "open", target}
		} else {
			argv = []string{opener, target}
		}
	}
	pid, err := launch(argv)
	if err != nil {
		return nil, errf("Could not start %s: %v", argv[0], err)
	}
	o["opened"] = firstNonEmpty(appQ, target)
	o["pid"] = pid
	o["command"] = argv
	if appQ != "" {
		o["app"] = appQ
	}
	if x == nil {
		return o, nil
	}
	// Watch for the new window: to give it back focus (background), hide it, or report it (--wait).
	settle := waitFor
	if settle == 0 && (mode != "foreground" || hidden) {
		settle = 3 * time.Second
	}
	if settle == 0 {
		return o, nil
	}
	var title string
	if file != "" {
		title = filepath.Base(file)
	}
	res, w, werr := waitOpenWindowID(x, o, 0, title, settle, before)
	if werr != nil && waitFor == 0 {
		return o, nil // nothing appeared within the settle time: launched, not waited for
	}
	if werr != nil {
		return nil, werr
	}
	if hidden {
		iconify(x, w)
		res["hidden"] = true
	}
	if mode != "foreground" && prevActive != 0 && x.activeWindow() != prevActive {
		_ = x.activate(prevActive)
		res["restored"] = obj{"frontmost": x.waitActive(prevActive, time.Second)}
	}
	if mode == "foreground" {
		_ = x.activate(w)
	}
	if waitFor == 0 {
		delete(res, "waited")
		delete(res, "window")
	}
	return res, nil
}

// waitOpenWindow waits for a new window (not in before) — of pid when given, titled like title
// when given — and adds it to o.
func waitOpenWindow(x *xconn, o obj, pid int, title string, timeout time.Duration, before map[xproto.Window]bool) (obj, *cmdError) {
	r, _, e := waitOpenWindowID(x, o, pid, title, timeout, before)
	return r, e
}

func waitOpenWindowID(x *xconn, o obj, pid int, title string, timeout time.Duration, before map[xproto.Window]bool) (obj, xproto.Window, *cmdError) {
	if x == nil {
		return o, 0, nil
	}
	start := time.Now()
	for {
		for _, w := range x.clientList() {
			if before != nil && before[w] {
				continue
			}
			if pid != 0 && x.windowPid(w) != pid {
				continue
			}
			if !x.normalWindow(w) || !x.mapped(w) {
				continue
			}
			t := x.windowTitle(w)
			if title != "" && !containsFold(t, strings.TrimSuffix(title, filepath.Ext(title))) {
				continue
			}
			r, _ := x.geometry(w)
			o["waited"] = secs(time.Since(start))
			o["window"] = obj{"title": t, "frame": r.obj(), "id": uint32(w)}
			if wp := x.windowPid(w); wp != 0 {
				o["pid"] = wp
				if ap, err := findApp(strconv.Itoa(wp), false); err == nil {
					o["app"] = ap.Name
				}
			}
			return o, w, nil
		}
		if time.Since(start) > timeout {
			return nil, 0, errObj(obj{"error": "Timed out after " + timeout.String() + " waiting for the window", "waited": secs(time.Since(start)), "opened": o["opened"]})
		}
		time.Sleep(80 * time.Millisecond)
	}
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}
