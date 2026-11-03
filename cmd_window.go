package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/jezek/xgb/xproto"
)

// a11yWindowRef matches an X window to the accessible top-level window (by title) of the app.
func a11yWindowRef(b *a11yBus, ap *app, title string) string {
	if b == nil || ap.A == nil {
		return ""
	}
	kids := b.childrenOf(ap.A.root)
	b.loadAll(kids)
	for _, k := range kids {
		if k.name == title {
			return k.ref
		}
	}
	if len(kids) == 1 {
		return kids[0].ref
	}
	return ""
}

func windowState(x *xconn, ap *app, b *a11yBus, w xproto.Window, index int) obj {
	r, _ := x.geometry(w)
	title := x.windowTitle(w)
	o := obj{
		"index":      index,
		"id":         uint32(w),
		"title":      title,
		"frame":      r.obj(),
		"minimized":  x.minimized(w),
		"fullscreen": x.hasState(w, "_NET_WM_STATE_FULLSCREEN"),
		"ref":        a11yWindowRef(b, ap, title),
	}
	if x.activeWindow() == w {
		o["focused"] = true
		o["main"] = true
	}
	for _, t := range x.windowTypes(w) {
		if t == "_NET_WM_WINDOW_TYPE_DIALOG" {
			o["subrole"] = "dialog"
		}
	}
	if !x.mapped(w) && !x.minimized(w) {
		o["hidden"] = true
	}
	return o
}

func windowEntries(x *xconn, ap *app, b *a11yBus) []any {
	out := []any{}
	if x == nil {
		// no X11: the accessible top-level windows
		if b != nil && ap.A != nil {
			kids := b.childrenOf(ap.A.root)
			b.loadAll(kids)
			for i, k := range kids {
				e := obj{"index": i, "ref": k.ref, "title": k.name, "minimized": k.has(stIconified)}
				if k.hasFrame {
					e["frame"] = k.frame.obj()
				}
				if k.has(stActive) {
					e["focused"] = true
				}
				out = append(out, e)
			}
		}
		return out
	}
	for i, w := range ap.Windows {
		if i >= 50 {
			break
		}
		out = append(out, windowState(x, ap, b, w, i))
	}
	return out
}

func cmdWindowList(a map[string]string) (obj, *cmdError) {
	ap, e := targetApp(a["app"], true)
	if e != nil {
		return nil, e
	}
	var x *xconn
	if sessionType() == "x11" || sessionType() == "xwayland" {
		x, _ = X("window list")
	}
	b, _ := A11y("window list")
	if b != nil && ap.A == nil {
		b = nil
	}
	return obj{"app": ap.Name, "pid": ap.Pid, "windows": windowEntries(x, ap, b)}, nil
}

type winTarget struct {
	x     *xconn
	ap    *app
	b     *a11yBus
	w     xproto.Window
	index int
}

func pickWindow(command string, a map[string]string) (*winTarget, *cmdError) {
	x, e := X(command)
	if e != nil {
		return nil, e
	}
	ap, e := targetApp(a["app"], false)
	if e != nil {
		return nil, e
	}
	if len(ap.Windows) == 0 {
		return nil, errObj(obj{"error": ap.Name + " has no windows", "app": ap.Name})
	}
	var b *a11yBus
	if bb, err := A11y(command); err == nil {
		b = bb
		for _, ai := range bb.apps() {
			if ai.pid == ap.Pid {
				ap.A = ai
			}
		}
		if ap.A == nil {
			b = nil
		}
	}
	t := &winTarget{x: x, ap: ap, b: b, index: -1}
	switch {
	case a["index"] != "":
		i, e := parseIntArg("index", a["index"], 0)
		if e != nil || i < 0 || i >= len(ap.Windows) {
			return nil, errObj(obj{"error": fmt.Sprintf("Invalid --index '%s': %s has %d windows (see window list)", a["index"], ap.Name, len(ap.Windows)), "app": ap.Name})
		}
		t.index = i
	case a["title"] != "":
		var titles []string
		for i, w := range ap.Windows {
			tt := x.windowTitle(w)
			titles = append(titles, tt)
			if strings.EqualFold(tt, a["title"]) {
				t.index = i
				break
			}
		}
		if t.index < 0 {
			for i, tt := range titles {
				if containsFold(tt, a["title"]) {
					t.index = i
					break
				}
			}
		}
		if t.index < 0 {
			return nil, errObj(obj{"error": fmt.Sprintf("No window titled '%s' in %s. Windows: %s", a["title"], ap.Name, strings.Join(titles, ", ")), "app": ap.Name})
		}
	default:
		fw := ap.frontWindow(x)
		for i, w := range ap.Windows {
			if w == fw {
				t.index = i
			}
		}
	}
	t.w = ap.Windows[t.index]
	return t, nil
}

func (t *winTarget) state() obj {
	if !t.exists() {
		return nil
	}
	return windowState(t.x, t.ap, t.b, t.w, t.index)
}

func (t *winTarget) exists() bool {
	for _, w := range t.x.clientList() {
		if w == t.w {
			return true
		}
	}
	return false
}

func (t *winTarget) waitFor(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(40 * time.Millisecond)
	}
	return cond()
}

// keepFront puts the previously active window back if acting on the window activated it.
func (t *winTarget) keepFront(prev xproto.Window, o obj) {
	if prev != 0 && t.x.activeWindow() != prev && t.x.activeWindow() == t.w {
		_ = t.x.activate(prev)
		o["restored"] = obj{"frontmost": t.x.waitActive(prev, time.Second)}
	}
}

func windowDry(command string, a map[string]string, extra obj) (obj, *cmdError) {
	mode, e := parseMode(a["mode"])
	if e != nil {
		return nil, e
	}
	bg := command != "raise"
	if command == "raise" && mode == "background" {
		return nil, errObj(obj{"error": "Background mode not possible for window raise: raising a window brings its app to the front", "mode": "background"})
	}
	m, r, err := decideMode("window "+command, mode, bg, "raising a window brings its app to the front")
	if err != nil {
		return nil, err
	}
	if command == "raise" {
		m, r = "foreground", ""
	}
	o := obj{"action": command}
	for k, v := range extra {
		o[k] = v
	}
	if a["app"] != "" {
		o["app"] = a["app"]
	}
	return withDryRun(o, m, r), nil
}

func windowMode(command string, a map[string]string) (string, *cmdError) {
	mode, e := parseMode(a["mode"])
	if e != nil {
		return "", e
	}
	m, _, err := decideMode("window "+command, mode, true, "")
	return m, err
}

func cmdWindowMove(a map[string]string) (obj, *cmdError) {
	if a["x"] == "" || a["y"] == "" {
		return nil, errf("Provide --x and --y (the window's top-left corner in screen pixels)")
	}
	px, e := parseFloatArg("x", a["x"])
	if e != nil {
		return nil, e
	}
	py, e := parseFloatArg("y", a["y"])
	if e != nil {
		return nil, e
	}
	if dryRun() {
		return windowDry("move", a, obj{"requested": obj{"x": px, "y": py}})
	}
	if _, e := windowMode("move", a); e != nil {
		return nil, e
	}
	t, e := pickWindow("window move", a)
	if e != nil {
		return nil, e
	}
	if t.x.hasState(t.w, "_NET_WM_STATE_FULLSCREEN") {
		return nil, errObj(obj{"error": "The window is full screen: leave full screen first (window fullscreen false)", "app": t.ap.Name})
	}
	before := t.state()
	prev := t.x.activeWindow()
	moveResize(t.x, t.w, &px, &py, nil, nil)
	t.waitFor(2*time.Second, func() bool {
		r, _ := t.x.geometry(t.w)
		return int(r.X) == int(px) && int(r.Y) == int(py)
	})
	// window managers place the frame, not the content: correct by the difference once
	if r, _ := t.x.geometry(t.w); int(r.X) != int(px) || int(r.Y) != int(py) {
		nx, ny := px+(px-r.X), py+(py-r.Y)
		moveResize(t.x, t.w, &nx, &ny, nil, nil)
		t.waitFor(2*time.Second, func() bool {
			r, _ := t.x.geometry(t.w)
			return int(r.X) == int(px) && int(r.Y) == int(py)
		})
	}
	o := obj{"action": "move", "app": t.ap.Name, "before": before, "after": t.state(), "method": "_NET_MOVERESIZE_WINDOW", "mode": "background", "requested": obj{"x": px, "y": py}}
	t.keepFront(prev, o)
	if r, _ := t.x.geometry(t.w); int(r.X) != int(px) || int(r.Y) != int(py) {
		o["note"] = fmt.Sprintf("the window manager placed the window at %d,%d", int(r.X), int(r.Y))
	}
	return o, nil
}

// moveResize sends _NET_MOVERESIZE_WINDOW (StaticGravity: x/y are the content's position).
func moveResize(x *xconn, w xproto.Window, px, py, pw, ph *float64) {
	flags := uint32(10) | 2<<12 // static gravity, source = pager
	vals := make([]uint32, 4)
	if px != nil {
		flags |= 1 << 8
		vals[0] = uint32(int32(*px))
	}
	if py != nil {
		flags |= 1 << 9
		vals[1] = uint32(int32(*py))
	}
	if pw != nil {
		flags |= 1 << 10
		vals[2] = uint32(*pw)
	}
	if ph != nil {
		flags |= 1 << 11
		vals[3] = uint32(*ph)
	}
	if err := x.sendClientMessage(w, "_NET_MOVERESIZE_WINDOW", flags, vals[0], vals[1], vals[2], vals[3]); err != nil {
		// no EWMH window manager: configure the window directly
		var mask uint16
		var list []uint32
		if px != nil {
			mask |= xproto.ConfigWindowX
			list = append(list, uint32(int32(*px)))
		}
		if py != nil {
			mask |= xproto.ConfigWindowY
			list = append(list, uint32(int32(*py)))
		}
		if pw != nil {
			mask |= xproto.ConfigWindowWidth
			list = append(list, uint32(*pw))
		}
		if ph != nil {
			mask |= xproto.ConfigWindowHeight
			list = append(list, uint32(*ph))
		}
		xproto.ConfigureWindow(x.c, w, mask, list)
	}
	x.sync()
}

func cmdWindowResize(a map[string]string) (obj, *cmdError) {
	if a["width"] == "" && a["height"] == "" {
		return nil, errf("Provide --width and/or --height")
	}
	var pw, ph *float64
	req := obj{}
	if a["width"] != "" {
		v, e := parseFloatArg("width", a["width"])
		if e != nil || v <= 0 {
			return nil, errf("Invalid --width '%s': use a positive number", a["width"])
		}
		pw = &v
		req["width"] = v
	}
	if a["height"] != "" {
		v, e := parseFloatArg("height", a["height"])
		if e != nil || v <= 0 {
			return nil, errf("Invalid --height '%s': use a positive number", a["height"])
		}
		ph = &v
		req["height"] = v
	}
	if dryRun() {
		return windowDry("resize", a, obj{"requested": req})
	}
	if _, e := windowMode("resize", a); e != nil {
		return nil, e
	}
	t, e := pickWindow("window resize", a)
	if e != nil {
		return nil, e
	}
	before := t.state()
	prev := t.x.activeWindow()
	r0, _ := t.x.geometry(t.w)
	if pw == nil {
		pw = &r0.W
	}
	if ph == nil {
		ph = &r0.H
	}
	moveResize(t.x, t.w, nil, nil, pw, ph)
	t.waitFor(2*time.Second, func() bool {
		r, _ := t.x.geometry(t.w)
		return int(r.W) == int(*pw) && int(r.H) == int(*ph)
	})
	o := obj{"action": "resize", "app": t.ap.Name, "before": before, "after": t.state(), "method": "_NET_MOVERESIZE_WINDOW", "mode": "background", "requested": req}
	if r, _ := t.x.geometry(t.w); int(r.W) != int(*pw) || int(r.H) != int(*ph) {
		o["note"] = fmt.Sprintf("the app kept the window at %dx%d (its minimum/maximum size)", int(r.W), int(r.H))
	}
	t.keepFront(prev, o)
	return o, nil
}

func cmdWindowMinimize(a map[string]string) (obj, *cmdError) {
	if dryRun() {
		return windowDry("minimize", a, nil)
	}
	if _, e := windowMode("minimize", a); e != nil {
		return nil, e
	}
	t, e := pickWindow("window minimize", a)
	if e != nil {
		return nil, e
	}
	before := t.state()
	iconify(t.x, t.w)
	ok := t.waitFor(3*time.Second, func() bool { return t.x.minimized(t.w) })
	o := obj{"action": "minimize", "app": t.ap.Name, "before": before, "after": t.state(), "method": "WM_CHANGE_STATE", "mode": "background"}
	if !ok {
		return nil, errObj(obj{"error": "The window manager did not minimize the window", "app": t.ap.Name, "before": before})
	}
	return o, nil
}

func iconify(x *xconn, w xproto.Window) {
	_ = x.sendClientMessage(w, "WM_CHANGE_STATE", 3)
	x.sync()
}

func deiconify(x *xconn, w xproto.Window) bool {
	prev := x.activeWindow()
	xproto.MapWindow(x.c, w)
	x.sync()
	deadline := time.Now().Add(1500 * time.Millisecond)
	for time.Now().Before(deadline) && x.minimized(w) {
		time.Sleep(40 * time.Millisecond)
	}
	if x.minimized(w) {
		_ = x.activate(w)
		deadline = time.Now().Add(1500 * time.Millisecond)
		for time.Now().Before(deadline) && x.minimized(w) {
			time.Sleep(40 * time.Millisecond)
		}
	}
	if prev != 0 && x.activeWindow() != prev {
		_ = x.activate(prev)
		x.waitActive(prev, time.Second)
		return true
	}
	return false
}

func cmdWindowRestore(a map[string]string) (obj, *cmdError) {
	if dryRun() {
		return windowDry("restore", a, nil)
	}
	if _, e := windowMode("restore", a); e != nil {
		return nil, e
	}
	t, e := pickWindow("window restore", a)
	if e != nil {
		return nil, e
	}
	before := t.state()
	restoredFront := deiconify(t.x, t.w)
	o := obj{"action": "restore", "app": t.ap.Name, "before": before, "after": t.state(), "method": "map", "mode": "background"}
	if restoredFront {
		o["restored"] = obj{"frontmost": true}
	}
	if t.x.minimized(t.w) {
		return nil, errObj(obj{"error": "The window manager did not restore the window", "app": t.ap.Name, "before": before})
	}
	return o, nil
}

func cmdWindowClose(a map[string]string) (obj, *cmdError) {
	if dryRun() {
		return windowDry("close", a, nil)
	}
	if _, e := windowMode("close", a); e != nil {
		return nil, e
	}
	t, e := pickWindow("window close", a)
	if e != nil {
		return nil, e
	}
	before := t.state()
	_ = t.x.sendClientMessage(t.w, "_NET_CLOSE_WINDOW", uint32(xproto.TimeCurrentTime), 2)
	t.x.sync()
	closed := t.waitFor(2*time.Second, func() bool { return !t.exists() })
	o := obj{"action": "close", "app": t.ap.Name, "before": before, "method": "_NET_CLOSE_WINDOW", "mode": "background", "closed": closed}
	if closed {
		o["after"] = nil
		return o, nil
	}
	o["after"] = t.state()
	if b, err := A11y("window close"); err == nil {
		if ap, err := findApp(fmt.Sprint(t.ap.Pid), true); err == nil && ap.A != nil {
			prepareOrigins(b, ap)
			if d := findDialog(b, ap, nil); d != nil {
				o["dialog"] = describeDialog(b, ap, d, nil, appOrigin(ap, true))
				o["note"] = "the app asks before closing: answer with dialog --app " + ap.Name + " --press <button>"
			}
		}
	}
	return o, nil
}

func cmdWindowFullscreen(a map[string]string) (obj, *cmdError) {
	want, e := parseBool("enabled", a["enabled"], true)
	if e != nil {
		return nil, e
	}
	if dryRun() {
		return windowDry("fullscreen", a, obj{"requested": obj{"fullscreen": want}})
	}
	if _, e := windowMode("fullscreen", a); e != nil {
		return nil, e
	}
	t, e := pickWindow("window fullscreen", a)
	if e != nil {
		return nil, e
	}
	before := t.state()
	prev := t.x.activeWindow()
	action := uint32(0)
	if want {
		action = 1
	}
	_ = t.x.sendClientMessage(t.w, "_NET_WM_STATE", action, uint32(t.x.atom("_NET_WM_STATE_FULLSCREEN")), 0, 2)
	t.x.sync()
	ok := t.waitFor(5*time.Second, func() bool { return t.x.hasState(t.w, "_NET_WM_STATE_FULLSCREEN") == want })
	o := obj{"action": "fullscreen", "app": t.ap.Name, "before": before, "after": t.state(), "method": "_NET_WM_STATE_FULLSCREEN", "mode": "background", "requested": obj{"fullscreen": want}}
	t.keepFront(prev, o)
	if !ok {
		return nil, errObj(obj{"error": "The window manager did not change full screen", "app": t.ap.Name, "before": before})
	}
	return o, nil
}

func cmdWindowRaise(a map[string]string) (obj, *cmdError) {
	if dryRun() {
		return windowDry("raise", a, nil)
	}
	mode, e := parseMode(a["mode"])
	if e != nil {
		return nil, e
	}
	if mode == "background" {
		return nil, errObj(obj{"error": "Background mode not possible for window raise: raising a window brings its app to the front", "mode": "background"})
	}
	t, e := pickWindow("window raise", a)
	if e != nil {
		return nil, e
	}
	before := t.state()
	if t.x.minimized(t.w) {
		xproto.MapWindow(t.x.c, t.w)
	}
	_ = t.x.activate(t.w)
	front := t.x.waitActive(t.w, 2*time.Second)
	return obj{"action": "raise", "app": t.ap.Name, "before": before, "after": t.state(), "frontmost": front, "method": "_NET_ACTIVE_WINDOW", "mode": "foreground"}, nil
}

// ---------------------------------------------------------------------------
// App-level: hide / unhide / quit

func appHidden(x *xconn, ap *app) bool {
	for _, w := range ap.Windows {
		if x.mapped(w) && !x.minimized(w) {
			return false
		}
	}
	return true
}

func cmdHide(a map[string]string) (obj, *cmdError) {
	return hideUnhide(a, true)
}

func cmdUnhide(a map[string]string) (obj, *cmdError) {
	return hideUnhide(a, false)
}

func hideUnhide(a map[string]string, hide bool) (obj, *cmdError) {
	if a["app"] == "" {
		return nil, errf("Provide the app")
	}
	name := "unhide"
	if hide {
		name = "hide"
	}
	if dryRun() {
		return obj{"dryRun": true, "app": a["app"], "mode": "background", "hidden": hide}, nil
	}
	x, e := X(name)
	if e != nil {
		return nil, e
	}
	ap, e := findApp(a["app"], false)
	if e != nil {
		return nil, e
	}
	before := appHidden(x, ap)
	o := obj{"app": ap.Name, "pid": ap.Pid, "before": obj{"hidden": before}, "mode": "background"}
	if before == hide {
		o["changed"], o["hidden"], o["method"] = false, before, "unchanged"
		return o, nil
	}
	restored := false
	for _, w := range ap.Windows {
		if hide {
			iconify(x, w)
		} else if deiconify(x, w) {
			restored = true
		}
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && appHidden(x, ap) != hide {
		time.Sleep(50 * time.Millisecond)
	}
	now := appHidden(x, ap)
	o["changed"], o["hidden"] = now != before, now
	if hide {
		o["method"] = "iconify"
	} else {
		o["method"] = "map"
	}
	if restored {
		o["restored"] = obj{"frontmost": true}
	}
	if now != hide {
		return nil, errObj(obj{"error": "The window manager did not " + name + " " + ap.Name, "app": ap.Name})
	}
	return o, nil
}

var sessionProcesses = map[string]bool{
	"xorg": true, "xvfb": true, "xwayland": true, "gnome-shell": true, "plasmashell": true, "kwin_x11": true, "kwin_wayland": true,
	"xfwm4": true, "xfce4-panel": true, "xfdesktop": true, "xfce4-session": true, "openbox": true, "fluxbox": true, "i3": true,
	"mutter": true, "marco": true, "metacity": true, "mate-panel": true, "cinnamon": true, "lxpanel": true, "lxsession": true,
	"pcmanfm-desktop": true, "nautilus-desktop": true, "dbus-daemon": true, "at-spi-bus-launcher": true, "at-spi2-registryd": true,
	"gnome-session-binary": true, "systemd": true, "ksmserver": true, "awesome": true, "bspwm": true, "sway": true, "weston": true,
}

func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); err != nil {
		return false
	}
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err == nil {
		f := strings.Fields(string(b))
		if len(f) > 2 && f[2] == "Z" {
			return false
		}
	}
	return true
}

func cmdQuit(a map[string]string) (obj, *cmdError) {
	if a["app"] == "" {
		return nil, errf("Provide the app to quit")
	}
	force, e := parseBool("force", a["force"], false)
	if e != nil {
		return nil, e
	}
	timeout, e := parseDuration("timeout", a["timeout"], 5*time.Second)
	if e != nil {
		return nil, e
	}
	if dryRun() {
		return obj{"dryRun": true, "app": a["app"], "force": force, "mode": "background"}, nil
	}
	ap, e := findApp(a["app"], true)
	if e != nil {
		return nil, e
	}
	if sessionProcesses[strings.ToLower(ap.Comm)] || sessionProcesses[strings.ToLower(ap.Name)] {
		return nil, errObj(obj{"error": fmt.Sprintf("Refusing to quit %s: it is part of the desktop session", ap.Name), "app": ap.Name})
	}
	if ap.Pid <= 0 {
		return nil, errObj(obj{"error": ap.Name + " has no known process id (_NET_WM_PID), so it can't be quit safely", "app": ap.Name})
	}
	start := time.Now()
	o := obj{"app": ap.Name, "pid": ap.Pid, "force": force, "mode": "background"}
	if force {
		_ = syscall.Kill(ap.Pid, syscall.SIGTERM)
		o["method"] = "SIGTERM"
		half := time.Now().Add(timeout / 2)
		for time.Now().Before(half) && pidAlive(ap.Pid) {
			time.Sleep(50 * time.Millisecond)
		}
		if pidAlive(ap.Pid) {
			_ = syscall.Kill(ap.Pid, syscall.SIGKILL)
			o["method"] = "SIGKILL"
		}
		deadline := start.Add(timeout)
		for time.Now().Before(deadline) && pidAlive(ap.Pid) {
			time.Sleep(50 * time.Millisecond)
		}
		o["terminated"] = !pidAlive(ap.Pid)
		o["waited"] = secs(time.Since(start))
		if pidAlive(ap.Pid) {
			return nil, errObj(o)
		}
		return o, nil
	}
	x, xe := X("quit")
	b, _ := A11y("quit")
	checkDialog := func() obj {
		if b == nil {
			return nil
		}
		fresh, err := findApp(fmt.Sprint(ap.Pid), true)
		if err != nil || fresh.A == nil {
			return nil
		}
		prepareOrigins(b, fresh)
		if d := findDialog(b, fresh, nil); d != nil {
			return describeDialog(b, fresh, d, nil, appOrigin(fresh, true))
		}
		return nil
	}
	if d := checkDialog(); d != nil {
		o["terminated"], o["dialog"], o["method"] = false, d, "close-windows"
		o["note"] = "a dialog is open and keeps the app from quitting: answer it with dialog --app " + ap.Name + " --press <button>, then quit again"
		return o, nil
	}
	o["method"] = "close-windows"
	if xe == nil {
		for _, w := range ap.Windows {
			_ = x.sendClientMessage(w, "_NET_CLOSE_WINDOW", uint32(xproto.TimeCurrentTime), 2)
		}
		x.sync()
	} else {
		_ = syscall.Kill(ap.Pid, syscall.SIGTERM)
		o["method"] = "SIGTERM"
	}
	deadline := start.Add(timeout)
	lastDialogCheck := time.Now()
	for time.Now().Before(deadline) {
		if !pidAlive(ap.Pid) {
			o["terminated"], o["waited"] = true, secs(time.Since(start))
			return o, nil
		}
		if time.Since(lastDialogCheck) > 300*time.Millisecond {
			lastDialogCheck = time.Now()
			if d := checkDialog(); d != nil {
				o["terminated"], o["dialog"], o["waited"] = false, d, secs(time.Since(start))
				o["note"] = "the app asks before quitting: answer with dialog --app " + ap.Name + " --press <button> (it quits after \"Don't Save\"/\"Close without Saving\"), or press Cancel to keep it running"
				return o, nil
			}
		}
		time.Sleep(60 * time.Millisecond)
	}
	o["terminated"], o["waited"] = false, secs(time.Since(start))
	o["error"] = fmt.Sprintf("%s did not quit within %s", ap.Name, timeout)
	return nil, errObj(o)
}
