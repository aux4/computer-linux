package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const contractVersion = 1

func packageVersion() string {
	exe, err := os.Executable()
	if err != nil {
		return "unknown"
	}
	dirs := []string{filepath.Dir(exe)}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		dirs = append(dirs, filepath.Dir(real))
	}
	for _, d := range dirs {
		for i := 0; i < 5; i++ {
			b, err := os.ReadFile(filepath.Join(d, ".aux4"))
			if err == nil {
				var m struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				}
				if json.Unmarshal(b, &m) == nil && m.Name == "computer-linux" && m.Version != "" {
					return m.Version
				}
			}
			d = filepath.Dir(d)
		}
	}
	return "unknown"
}

func tesseractPath() string {
	p, err := exec.LookPath("tesseract")
	if err != nil {
		return ""
	}
	return p
}

type permStatus struct {
	accessibility, screen, input bool
	a11yGuidance, screenGuidance, inputGuidance string
	a11yEnabled                                  *bool
}

func checkPermissions() permStatus {
	p := permStatus{}
	if denyPermissions() {
		p.a11yGuidance = "Accessibility permission not granted: " + a11yGuidance
		p.screenGuidance = "Screen capture not available (denied by AUX4_COMPUTER_DENY_PERMISSIONS)."
		p.inputGuidance = "Input not available (denied by AUX4_COMPUTER_DENY_PERMISSIONS)."
		return p
	}
	if b, err := A11y("permissions"); err == nil {
		p.accessibility = true
		p.a11yGuidance = "Accessibility (AT-SPI) is available."
		// org.a11y.Status IsEnabled: toolkits only expose their trees when it's on.
		if sess, e := sessionBusIsEnabled(); e == nil {
			p.a11yEnabled = &sess
			if !sess {
				p.a11yGuidance = "The accessibility bus is running but toolkit accessibility is off, so GTK/Qt apps may not expose their UI. Enable it: gsettings set org.gnome.desktop.interface toolkit-accessibility true (and restart the apps)."
			}
		}
		_ = b
	} else {
		p.a11yGuidance = err.Error()
	}
	switch sessionType() {
	case "wayland":
		p.screenGuidance = "Wayland session without X11: screen capture is not available to clients. " + waylandHint
		p.inputGuidance = "Wayland session without X11: synthetic input is not available to clients. " + waylandHint
	case "none":
		p.screenGuidance = "No graphical session: DISPLAY is not set."
		p.inputGuidance = "No graphical session: DISPLAY is not set."
	default:
		if x, err := X("permissions"); err == nil {
			p.screen = true
			p.screenGuidance = "Screen capture is available (X11)."
			if sessionType() == "xwayland" {
				p.screenGuidance = "XWayland: only X11 apps' windows can be captured; the Wayland desktop and Wayland-native apps are not visible to X11 clients."
			}
			p.input = x.hasXTest
			if x.hasXTest {
				p.inputGuidance = "Input through XTEST is available."
			} else {
				p.inputGuidance = "The X server has no XTEST extension, so input can't be sent."
			}
		} else {
			p.screenGuidance = err.Error()
			p.inputGuidance = err.Error()
		}
	}
	return p
}

func sessionBusIsEnabled() (bool, error) {
	conn, err := sessionBus()
	if err != nil {
		return false, err
	}
	v, err := conn.Object("org.a11y.Bus", "/org/a11y/bus").GetProperty("org.a11y.Status.IsEnabled")
	if err != nil {
		return false, err
	}
	b, _ := v.Value().(bool)
	return b, nil
}

func (p permStatus) obj() obj {
	o := obj{
		"accessibility":   obj{"granted": p.accessibility, "guidance": p.a11yGuidance},
		"screenRecording": obj{"granted": p.screen, "guidance": p.screenGuidance},
		"input":           obj{"granted": p.input, "guidance": p.inputGuidance},
	}
	if p.a11yEnabled != nil {
		o["accessibility"].(obj)["toolkitAccessibility"] = *p.a11yEnabled
	}
	return o
}

func cmdPermissions(a map[string]string) (obj, *cmdError) {
	return checkPermissions().obj(), nil
}

func cmdInfo(a map[string]string) (obj, *cmdError) {
	p := checkPermissions()
	x11 := p.screen
	ocr := tesseractPath() != ""
	caps := obj{
		"background":    p.accessibility,
		"accessibility": p.accessibility,
		"screenshot":    x11,
		"annotate":      x11 && p.accessibility,
		"ocr":           ocr,
		"ocrScreen":     ocr && x11,
		"input":         p.input,
		"windows":       x11,
		"clipboard":     x11,
		"menus":         p.accessibility,
		"dialogs":       p.accessibility,
	}
	o := obj{
		"provider":     "linux",
		"contract":     contractVersion,
		"platform":     runtime.GOOS + "/" + runtime.GOARCH,
		"version":      packageVersion(),
		"session":      sessionType(),
		"capabilities": caps,
		"permissions":  p.obj(),
		"stopped":      false,
	}
	if s, why := stopState(); s {
		o["stopped"] = true
		o["stopReason"] = why
	}
	if t := tesseractPath(); t != "" {
		o["ocrEngine"] = "tesseract"
	}
	if sessionType() == "wayland" {
		o["unsupported"] = []string{"screenshot", "ocr (except --image)", "click --x/--y fallback", "move", "drag", "shortcut (real keys)", "scroll (real wheel)", "windows", "window", "clipboard", "focus"}
	}
	return o, nil
}

func cmdStop(a map[string]string) (obj, *cmdError) {
	f := stopFile()
	if err := os.MkdirAll(filepath.Dir(f), 0o700); err != nil {
		return nil, errf("Cannot create %s: %v", filepath.Dir(f), err)
	}
	if err := os.WriteFile(f, []byte("stopped by aux4 computer linux stop\n"), 0o600); err != nil {
		return nil, errf("Cannot create %s: %v", f, err)
	}
	return obj{"stopped": true, "file": f}, nil
}

func cmdResume(a map[string]string) (obj, *cmdError) {
	f := stopFile()
	existed := false
	if _, err := os.Stat(f); err == nil {
		existed = true
		if err := os.Remove(f); err != nil {
			return nil, errf("Cannot remove %s: %v", f, err)
		}
	}
	o := obj{"stopped": false, "file": f, "removed": existed}
	if envOn("AUX4_COMPUTER_STOP") {
		o["stopped"] = true
		o["note"] = "AUX4_COMPUTER_STOP=1 is still set in this environment"
	}
	return o, nil
}

func cmdLog(a map[string]string) (obj, *cmdError) {
	limit, e := parsePositive("limit", a["limit"], 20)
	if e != nil {
		return nil, e
	}
	stopped, why := stopState()
	o := obj{"entries": readLogEntries(limit), "path": logFile(), "stopped": stopped}
	if stopped {
		o["stopReason"] = why
	}
	return o, nil
}

func cmdCursor(a map[string]string) (obj, *cmdError) {
	x, e := X("cursor")
	if e != nil {
		return nil, e
	}
	px, py := x.pointer()
	return obj{"x": px, "y": py}, nil
}

func cmdScreen(a map[string]string) (obj, *cmdError) {
	x, e := X("screen")
	if e != nil {
		return nil, e
	}
	sc := x.scale()
	var ds []any
	for _, d := range x.displays() {
		o := obj{"index": d.Index, "id": d.ID, "main": d.Main, "x": d.R.X, "y": d.R.Y, "width": d.R.W, "height": d.R.H, "scale": sc}
		if d.Name != "" {
			o["name"] = d.Name
		}
		ds = append(ds, o)
	}
	return obj{"displays": ds}, nil
}

func cmdApps(a map[string]string) (obj, *cmdError) {
	var x *xconn
	if sessionType() == "x11" || sessionType() == "xwayland" {
		x, _ = X("apps")
	}
	_, a11yErr := A11y("apps")
	front := 0
	if f, err := frontmostApp(false); err == nil {
		front = f.Pid
	}
	var list []any
	for _, ap := range listApps(a11yErr == nil) {
		o := obj{"name": ap.Name, "pid": ap.Pid, "active": ap.Pid != 0 && ap.Pid == front, "accessibility": ap.A != nil}
		if ap.Class != "" {
			o["class"] = ap.Class
		}
		if ap.Comm != "" {
			o["process"] = ap.Comm
		}
		if x != nil && len(ap.Windows) > 0 {
			hidden := true
			for _, w := range ap.Windows {
				if !x.minimized(w) && x.mapped(w) {
					hidden = false
				}
			}
			o["hidden"] = hidden
			o["windows"] = len(ap.Windows)
		}
		list = append(list, o)
	}
	if list == nil {
		list = []any{}
	}
	return obj{"apps": list}, nil
}


func cmdWindows(a map[string]string) (obj, *cmdError) {
	x, e := X("windows")
	if e != nil {
		return nil, e
	}
	var list []any
	for _, ap := range listApps(false) {
		for _, w := range ap.Windows {
			if !x.mapped(w) || x.minimized(w) {
				continue
			}
			r, ok := x.geometry(w)
			if !ok {
				continue
			}
			list = append(list, obj{"id": uint32(w), "app": ap.Name, "pid": ap.Pid, "title": x.windowTitle(w), "frame": r.obj()})
		}
	}
	if list == nil {
		list = []any{}
	}
	return obj{"windows": list}, nil
}

func trimLines(s string) string { return strings.TrimSpace(s) }
