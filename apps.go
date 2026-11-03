package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jezek/xgb/xproto"
)

// app is a running application as seen through X11 (windows) and/or AT-SPI (accessibility tree).
type app struct {
	Name     string
	Pid      int
	Class    string // WM_CLASS class
	Instance string // WM_CLASS instance
	Comm     string // process name
	Windows  []xproto.Window // client windows, bottom-to-top stacking order
	A        *appInfo
}

func procComm(pid int) string {
	if pid <= 0 {
		return ""
	}
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func procExe(pid int) string {
	p, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return ""
	}
	return p
}

// normalWindow: a regular app window (not a dock, desktop, splash, tooltip...).
func (x *xconn) normalWindow(w xproto.Window) bool {
	for _, t := range x.windowTypes(w) {
		switch t {
		case "_NET_WM_WINDOW_TYPE_DESKTOP", "_NET_WM_WINDOW_TYPE_DOCK", "_NET_WM_WINDOW_TYPE_TOOLBAR",
			"_NET_WM_WINDOW_TYPE_MENU", "_NET_WM_WINDOW_TYPE_SPLASH", "_NET_WM_WINDOW_TYPE_DROPDOWN_MENU",
			"_NET_WM_WINDOW_TYPE_POPUP_MENU", "_NET_WM_WINDOW_TYPE_TOOLTIP", "_NET_WM_WINDOW_TYPE_NOTIFICATION",
			"_NET_WM_WINDOW_TYPE_COMBO", "_NET_WM_WINDOW_TYPE_DND":
			return false
		}
	}
	return true
}

// listApps merges X11 clients and AT-SPI applications by process id.
func listApps(withA11y bool) []*app {
	byPid := map[int]*app{}
	var order []*app
	if sessionType() == "x11" || sessionType() == "xwayland" {
		if x, err := X("apps"); err == nil {
			for _, w := range x.clientList() {
				if !x.normalWindow(w) {
					continue
				}
				pid := x.windowPid(w)
				inst, cls := x.wmClass(w)
				key := pid
				if pid == 0 {
					key = -int(w)
				}
				a := byPid[key]
				if a == nil {
					a = &app{Pid: pid, Class: cls, Instance: inst, Comm: procComm(pid)}
					byPid[key] = a
					order = append(order, a)
				}
				a.Windows = append(a.Windows, w)
			}
		}
	}
	if withA11y {
		if b, err := A11y("apps"); err == nil {
			for _, ai := range b.apps() {
				if ai.pid > 0 {
					if a := byPid[ai.pid]; a != nil {
						a.A = ai
						continue
					}
				}
				// accessibility-only (no X11 window: Wayland-native, or windowless)
				var kids int32
				if v, err := b.getProp(ai.bus, ai.root.path, ifAccessible, "ChildCount"); err == nil {
					kids, _ = v.Value().(int32)
				}
				if kids == 0 {
					continue
				}
				a := &app{Pid: ai.pid, A: ai, Comm: procComm(ai.pid)}
				key := ai.pid
				if key == 0 {
					key = -1000000 - len(order)
				}
				byPid[key] = a
				order = append(order, a)
			}
		}
	}
	for _, a := range order {
		switch {
		case a.Class != "":
			a.Name = a.Class
		case a.A != nil && a.A.name != "":
			a.Name = a.A.name
		case a.Comm != "":
			a.Name = a.Comm
		default:
			a.Name = "pid " + strconv.Itoa(a.Pid)
		}
	}
	return order
}

func (a *app) names() []string {
	n := []string{a.Name, a.Class, a.Instance, a.Comm}
	if a.A != nil {
		n = append(n, a.A.name)
	}
	if exe := procExe(a.Pid); exe != "" {
		n = append(n, filepath.Base(exe))
	}
	return n
}

// findApp resolves --app: a pid, an exact name (name, WM_CLASS, process, accessibility name), then a substring.
func findApp(query string, withA11y bool) (*app, *cmdError) {
	q := strings.TrimSpace(query)
	apps := listApps(withA11y)
	if pid, err := strconv.Atoi(q); err == nil {
		for _, a := range apps {
			if a.Pid == pid {
				return a, nil
			}
		}
	}
	lq := strings.ToLower(q)
	lq = strings.TrimSuffix(lq, ".desktop")
	for _, a := range apps {
		for _, n := range a.names() {
			if n != "" && strings.ToLower(n) == lq {
				return a, nil
			}
		}
	}
	// reverse-DNS ids (org.gnome.Calculator) match their last part
	if i := strings.LastIndex(lq, "."); i >= 0 && i < len(lq)-1 {
		last := lq[i+1:]
		for _, a := range apps {
			for _, n := range a.names() {
				if n != "" && strings.ToLower(n) == last {
					return a, nil
				}
			}
		}
	}
	for _, a := range apps {
		for _, n := range a.names() {
			if n != "" && strings.Contains(strings.ToLower(n), lq) {
				return a, nil
			}
		}
	}
	return nil, errf("App not running: %s", query)
}

// frontmostApp is the app owning the active window.
func frontmostApp(withA11y bool) (*app, *cmdError) {
	if sessionType() == "x11" || sessionType() == "xwayland" {
		if x, err := X("frontmost"); err == nil {
			aw := x.activeWindow()
			if aw != 0 {
				for _, a := range listApps(withA11y) {
					for _, w := range a.Windows {
						if w == aw {
							return a, nil
						}
					}
				}
			}
		}
	}
	if withA11y {
		// Wayland / no active X window: the app whose frame is active in the accessibility tree.
		if b, err := A11y("frontmost"); err == nil {
			for _, a := range listApps(true) {
				if a.A == nil {
					continue
				}
				for _, k := range b.childrenOf(a.A.root) {
					b.load(k)
					if k.has(stActive) {
						return a, nil
					}
				}
			}
		}
	}
	return nil, errf("No frontmost app: pass --app")
}

// targetApp resolves --app, or the frontmost app when empty.
func targetApp(query string, withA11y bool) (*app, *cmdError) {
	if strings.TrimSpace(query) == "" {
		return frontmostApp(withA11y)
	}
	return findApp(query, withA11y)
}

// frontWindow is the app's topmost mapped (not minimized) window, or its topmost window.
func (a *app) frontWindow(x *xconn) xproto.Window {
	if x == nil || len(a.Windows) == 0 {
		return 0
	}
	aw := x.activeWindow()
	for _, w := range a.Windows {
		if w == aw {
			return w
		}
	}
	for i := len(a.Windows) - 1; i >= 0; i-- {
		if x.mapped(a.Windows[i]) && !x.minimized(a.Windows[i]) {
			return a.Windows[i]
		}
	}
	return a.Windows[len(a.Windows)-1]
}

// needA11y returns the app's accessibility root or an error.
func (a *app) needA11y() (*a11yBus, *appInfo, *cmdError) {
	b, err := A11y("accessibility")
	if err != nil {
		return nil, nil, err
	}
	if a.A == nil {
		return nil, nil, errObj(obj{"error": fmt.Sprintf("%s does not expose an accessibility tree (no AT-SPI connection from pid %d). GTK and Qt apps do when accessibility is enabled; apps started before it was enabled must be restarted", a.Name, a.Pid), "app": a.Name})
	}
	prepareOrigins(b, a)
	return b, a.A, nil
}

// prepareOrigins records each accessible top-level window's screen origin, for toolkits whose
// extents are window-relative (GTK4).
func prepareOrigins(b *a11yBus, a *app) {
	if a.A == nil || !a.A.winCoords || len(a.A.origins) > 0 {
		return
	}
	x, err := X("origins")
	if err != nil {
		return
	}
	kids := b.childrenOf(a.A.root)
	b.loadAll(kids)
	for _, k := range kids {
		i := topIndex(k.ref)
		for _, w := range a.Windows {
			if x.windowTitle(w) == k.name {
				if r, ok := x.geometry(w); ok {
					// GTK4 client-side decorations include an invisible shadow margin.
					ext := x.propU32s(w, "_GTK_FRAME_EXTENTS")
					if len(ext) == 4 {
						r.X += float64(ext[0])
						r.Y += float64(ext[2])
					}
					a.A.origins[i] = r
				}
				break
			}
		}
	}
}

// windowOrigin is the window frame used for window-relative coordinates (--app x/y, windowCenter):
// the content rectangle of the app's front window.
func (a *app) windowOrigin() (rect, xproto.Window, bool) {
	x, err := X("window")
	if err != nil {
		return rect{}, 0, false
	}
	w := a.frontWindow(x)
	if w == 0 {
		return rect{}, 0, false
	}
	r, ok := x.geometry(w)
	return r, w, ok
}

func (a *app) summary() obj {
	o := obj{"name": a.Name, "pid": a.Pid}
	if a.Class != "" {
		o["class"] = a.Class
	}
	return o
}

func sortedAppNames(apps []*app) []string {
	var n []string
	for _, a := range apps {
		n = append(n, a.Name)
	}
	sort.Strings(n)
	return n
}
