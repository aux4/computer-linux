package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jezek/xgb/xproto"
)

func withDryRun(o obj, mode, reason string) obj {
	o["dryRun"] = true
	o["mode"] = mode
	if reason != "" {
		o["fallbackReason"] = reason
	}
	return o
}

func setFallback(o obj, mode, reason string, restored obj) {
	o["mode"] = mode
	if reason != "" {
		o["fallbackReason"] = reason
		if restored != nil {
			o["restored"] = restored
		}
	}
}

// screenPoint converts --x/--y (window-relative with --app) into screen coordinates.
func screenPoint(ap *app, appGiven bool, x, y float64) (float64, float64, *cmdError) {
	if !appGiven {
		return x, y, nil
	}
	r, _, ok := ap.windowOrigin()
	if !ok {
		return 0, 0, errObj(obj{"error": fmt.Sprintf("%s has no window on screen to take --x/--y relative to", ap.Name), "app": ap.Name})
	}
	return r.X + x, r.Y + y, nil
}

func cmdClick(a map[string]string) (obj, *cmdError) {
	mode, e := parseMode(a["mode"])
	if e != nil {
		return nil, e
	}
	button := a["button"]
	if button == "" {
		button = "left"
	}
	if button != "left" && button != "right" {
		return nil, errf("Invalid --button '%s': use left or right", button)
	}
	count, e := parsePositive("count", a["count"], 1)
	if e != nil {
		return nil, e
	}
	ref := a["element"]
	if ref == "" && (a["x"] == "" || a["y"] == "") {
		return nil, errf("Provide --x and --y, or --element <ref> from find/read/state")
	}
	var px, py float64
	if ref == "" {
		if px, e = parseFloatArg("x", a["x"]); e != nil {
			return nil, e
		}
		if py, e = parseFloatArg("y", a["y"]); e != nil {
			return nil, e
		}
	}
	bgPossible, reason := true, ""
	if count > 1 {
		bgPossible, reason = false, "multi-click has no accessibility equivalent"
	} else if button == "right" {
		bgPossible, reason = false, "right-click has no accessibility equivalent"
	}
	if dryRun() {
		m, r, err := decideMode("click", mode, bgPossible, reason)
		if err != nil {
			return nil, err
		}
		if ref != "" {
			o := obj{"clicked": ref}
			if a["app"] != "" {
				o["app"] = a["app"]
			}
			return withDryRun(o, m, r), nil
		}
		o := obj{"button": button, "count": count, "x": px, "y": py}
		if a["app"] != "" {
			o["app"] = a["app"]
		}
		return withDryRun(o, m, r), nil
	}
	ap, e := targetApp(a["app"], true)
	if e != nil {
		return nil, e
	}
	appGiven := a["app"] != ""

	if ref != "" {
		b, ai, e := ap.needA11y()
		if e != nil {
			return nil, e
		}
		n, e := b.resolve(ai, ref)
		if e != nil {
			return nil, e
		}
		m, r, err := decideMode("click", mode, bgPossible, reason)
		if err != nil {
			return nil, err
		}
		o := obj{"app": ap.Name, "clicked": ref, "element": shortElem(n)}
		if m == "background" {
			act, perr := b.press(n)
			if perr == nil {
				o["action"] = act
				o["mode"] = "background"
				if isTextInputRole(n.role) {
					b.grabFocus(n)
				}
				return o, nil
			}
			if isTextInputRole(n.role) && b.grabFocus(n) {
				o["action"] = "focus"
				o["mode"] = "background"
				return o, nil
			}
			if mode == "background" {
				return nil, errObj(obj{"error": "Background mode not possible for click: " + perr.Error(), "mode": "background", "app": ap.Name})
			}
			m, r = "foreground", perr.Error()
		}
		if !n.hasFrame {
			return nil, errObj(obj{"error": "Element " + ref + " has no on-screen position to click", "app": ap.Name})
		}
		cx, cy := n.frame.center()
		return foregroundClick(o, ap, cx, cy, button, count, mode, r)
	}

	sx, sy, e := screenPoint(ap, appGiven, px, py)
	if e != nil {
		return nil, e
	}
	o := obj{"app": ap.Name, "button": button, "count": count, "x": px, "y": py, "screen": point(sx, sy)}
	m, r, err := decideMode("click", mode, bgPossible, reason)
	if err != nil {
		return nil, err
	}
	if m == "background" {
		b, ai, ae := ap.needA11y()
		var why string
		if ae != nil {
			why = ae.Error()
		} else {
			target := pressableAt(b, ai, sx, sy)
			if target == nil {
				why = "no accessible element with an action at that point"
			} else {
				act, perr := b.press(target)
				if perr == nil {
					o["action"] = act
					o["element"] = obj{"label": target.label(), "role": target.role, "ref": target.ref}
					o["mode"] = "background"
					if isTextInputRole(target.role) {
						b.grabFocus(target)
					}
					return o, nil
				}
				if isTextInputRole(target.role) && b.grabFocus(target) {
					o["action"] = "focus"
					o["element"] = obj{"label": target.label(), "role": target.role, "ref": target.ref}
					o["mode"] = "background"
					return o, nil
				}
				why = perr.Error()
			}
		}
		if mode == "background" {
			return nil, errObj(obj{"error": "Background mode not possible for click: " + why, "mode": "background", "app": ap.Name})
		}
		r = why
	}
	return foregroundClick(o, ap, sx, sy, button, count, mode, r)
}

// pressableAt finds the element at a screen point in the app's windows and walks up to the
// nearest one with an action (or a text field, which takes focus).
func pressableAt(b *a11yBus, ai *appInfo, sx, sy float64) *node {
	kids := b.childrenOf(ai.root)
	b.loadAll(kids)
	// active window first, then the others
	var order []*node
	for _, k := range kids {
		if k.has(stActive) {
			order = append([]*node{k}, order...)
		} else {
			order = append(order, k)
		}
	}
	for _, w := range order {
		if !w.hasFrame || !w.frame.contains(sx, sy) || !w.has(stShowing) {
			continue
		}
		hit := b.accessibleAt(w, sx, sy)
		for n := hit; n != nil && n != w; n = n.parent {
			b.load(n)
			if len(b.actions(n)) > 0 || isTextInputRole(n.role) {
				return n
			}
		}
	}
	return nil
}

func foregroundClick(o obj, ap *app, sx, sy float64, button string, count int, requested, reason string) (obj, *cmdError) {
	x, e := X("click")
	if e != nil {
		return nil, e
	}
	if e := x.requireXTest("click"); e != nil {
		return nil, e
	}
	var target xproto.Window
	if ap != nil {
		target = ap.frontWindow(x)
	}
	fs := x.beginForeground(target, requested == "auto")
	btn := byte(1)
	if button == "right" {
		btn = 3
	}
	x.motion(int(math.Round(sx)), int(math.Round(sy)))
	x.sync()
	time.Sleep(40 * time.Millisecond)
	for i := 0; i < count; i++ {
		x.button(btn, true)
		x.button(btn, false)
		x.sync()
		time.Sleep(60 * time.Millisecond)
	}
	restored := fs.end()
	setFallback(o, "foreground", reason, restored)
	if o["screen"] == nil {
		o["screen"] = point(sx, sy)
	}
	return o, nil
}

func cmdMove(a map[string]string) (obj, *cmdError) {
	px, e := parseFloatArg("x", a["x"])
	if e != nil {
		return nil, e
	}
	py, e := parseFloatArg("y", a["y"])
	if e != nil {
		return nil, e
	}
	if dryRun() {
		return obj{"dryRun": true, "mode": "foreground", "x": px, "y": py}, nil
	}
	x, e := X("move")
	if e != nil {
		return nil, e
	}
	if e := x.requireXTest("move"); e != nil {
		return nil, e
	}
	x.motion(int(px), int(py))
	x.sync()
	return obj{"mode": "foreground", "x": px, "y": py}, nil
}

func cmdDrag(a map[string]string) (obj, *cmdError) {
	var v [4]float64
	for i, k := range []string{"x", "y", "toX", "toY"} {
		f, e := parseFloatArg(k, a[k])
		if e != nil {
			return nil, e
		}
		v[i] = f
	}
	o := obj{"from": point(v[0], v[1]), "to": point(v[2], v[3]), "mode": "foreground"}
	if dryRun() {
		o["dryRun"] = true
		return o, nil
	}
	x, e := X("drag")
	if e != nil {
		return nil, e
	}
	if e := x.requireXTest("drag"); e != nil {
		return nil, e
	}
	x.motion(int(v[0]), int(v[1]))
	x.sync()
	time.Sleep(50 * time.Millisecond)
	x.button(1, true)
	x.sync()
	steps := 12
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x.motion(int(v[0]+(v[2]-v[0])*t), int(v[1]+(v[3]-v[1])*t))
		x.sync()
		time.Sleep(15 * time.Millisecond)
	}
	x.button(1, false)
	x.sync()
	return o, nil
}

func delay(a map[string]string) *cmdError {
	d, e := parseIntArg("delay", a["delay"], 0)
	if e != nil || d < 0 {
		return errf("Invalid --delay '%s': use milliseconds", a["delay"])
	}
	if d > 0 && !dryRun() {
		time.Sleep(time.Duration(d) * time.Millisecond)
	}
	return nil
}

func cmdType(a map[string]string) (obj, *cmdError) {
	mode, e := parseMode(a["mode"])
	if e != nil {
		return nil, e
	}
	text := a["text"]
	if text == "" {
		return nil, errf("Provide the text to type")
	}
	if e := delay(a); e != nil {
		return nil, e
	}
	if dryRun() {
		m, r, err := decideMode("type", mode, true, "")
		if err != nil {
			return nil, err
		}
		o := obj{"typed": len([]rune(text))}
		if a["app"] != "" {
			o["app"] = a["app"]
		}
		return withDryRun(o, m, r), nil
	}
	ap, e := targetApp(a["app"], true)
	if e != nil {
		return nil, e
	}
	o := obj{"app": ap.Name, "typed": len([]rune(text))}
	m, reason, err := decideMode("type", mode, true, "")
	if err != nil {
		return nil, err
	}
	if m == "background" {
		why := ""
		b, _, ae := ap.needA11y()
		if ae != nil {
			why = ae.Error()
		} else if n, te := editTarget(b, ap, ""); te != nil {
			why = te.Error()
		} else if !n.ifaces[ifEditableText] {
			why = "the focused element doesn't accept text through accessibility"
		} else {
			pos := b.caret(n)
			if s, en, ok := b.selection(n); ok {
				b.deleteText(n, s, en)
				pos = s
			}
			if pos < 0 {
				_, cnt := b.textOf(n, 1)
				pos = cnt
			}
			if b.insertText(n, pos, text) {
				b.setCaret(n, pos+len([]rune(text)))
				o["method"] = "accessibility"
				o["mode"] = "background"
				o["element"] = shortElem(n)
				return o, nil
			}
			why = "the app refused the text insertion"
		}
		if mode == "background" {
			return nil, errObj(obj{"error": "Background mode not possible for type: " + why, "mode": "background", "app": ap.Name})
		}
		reason = why
	}
	x, xe := X("type")
	if xe != nil {
		return nil, xe
	}
	if e := x.requireXTest("type"); e != nil {
		return nil, e
	}
	var target xproto.Window
	if a["app"] != "" {
		target = ap.frontWindow(x)
	}
	fs := x.beginForeground(target, mode == "auto")
	if err := x.typeText(text); err != nil {
		fs.end()
		return nil, errf("Typing failed: %v", err)
	}
	o["method"] = "xtest"
	setFallback(o, "foreground", reason, fs.end())
	return o, nil
}

// ---------------------------------------------------------------------------
// Shortcuts

type combo struct {
	mods []string // ctrl, shift, alt, super
	key  string
}

var keyNames = map[string]xproto.Keysym{
	"enter": 0xff0d, "return": 0xff0d, "tab": 0xff09, "space": 0x20, "escape": 0xff1b, "esc": 0xff1b,
	"backspace": 0xff08, "delete": 0xff08, "forwarddelete": 0xffff, "del": 0xffff, "insert": 0xff63,
	"up": 0xff52, "down": 0xff54, "left": 0xff51, "right": 0xff53, "home": 0xff50, "end": 0xff57,
	"pageup": 0xff55, "pagedown": 0xff56, "menu": 0xff67, "capslock": 0xffe5, "printscreen": 0xff61, "print": 0xff61,
	"plus": '+', "minus": '-', "comma": ',', "period": '.', "slash": '/', "semicolon": ';', "equal": '=', "equals": '=',
}

func parseCombo(keys string) (combo, *cmdError) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(keys)), "+")
	if len(parts) == 0 || keys == "" {
		return combo{}, errf("Provide the keys, e.g. ctrl+s or enter")
	}
	// "ctrl++" means ctrl and the plus key
	if strings.HasSuffix(keys, "++") {
		parts = append(parts[:len(parts)-2], "+")
	}
	c := combo{key: strings.TrimSpace(parts[len(parts)-1])}
	seen := map[string]bool{}
	for _, m := range parts[:len(parts)-1] {
		var n string
		switch strings.TrimSpace(m) {
		case "cmd", "command", "ctrl", "control", "primary":
			n = "ctrl"
		case "shift":
			n = "shift"
		case "alt", "option", "opt":
			n = "alt"
		case "super", "win", "meta", "windows", "logo":
			n = "super"
		case "fn":
			continue
		default:
			return combo{}, errf("Invalid modifier '%s' in '%s': use ctrl (or cmd), shift, alt (or option), super", m, keys)
		}
		if !seen[n] {
			c.mods = append(c.mods, n)
			seen[n] = true
		}
	}
	if c.key == "" {
		return combo{}, errf("Invalid keys '%s': the last part must be a key", keys)
	}
	if _, ok := keySym(c.key); !ok {
		return combo{}, errf("Unknown key '%s' in '%s'", c.key, keys)
	}
	return c, nil
}

func keySym(k string) (xproto.Keysym, bool) {
	if ks, ok := keyNames[k]; ok {
		return ks, true
	}
	if len(k) >= 2 && k[0] == 'f' {
		var n int
		if _, err := fmt.Sscanf(k, "f%d", &n); err == nil && n >= 1 && n <= 24 {
			return xproto.Keysym(0xffbe + n - 1), true
		}
	}
	r := []rune(k)
	if len(r) == 1 {
		return runeKeysym(r[0]), true
	}
	return 0, false
}

func (c combo) modKeysyms() []xproto.Keysym {
	var out []xproto.Keysym
	for _, m := range c.mods {
		switch m {
		case "ctrl":
			out = append(out, ksControlL)
		case "shift":
			out = append(out, ksShiftL)
		case "alt":
			out = append(out, ksAltL)
		case "super":
			out = append(out, ksSuperL)
		}
	}
	return out
}

func (c combo) canonical() string {
	order := map[string]int{"ctrl": 0, "alt": 1, "shift": 2, "super": 3}
	ms := append([]string{}, c.mods...)
	for i := range ms {
		for j := i + 1; j < len(ms); j++ {
			if order[ms[j]] < order[ms[i]] {
				ms[i], ms[j] = ms[j], ms[i]
			}
		}
	}
	return strings.Join(append(ms, c.key), "+")
}

// parseAccel turns a GTK accelerator ("<Primary><Shift>s", "<Control>q") into a combo string.
func parseAccel(acc string) string {
	acc = strings.TrimSpace(acc)
	if acc == "" {
		return ""
	}
	var mods []string
	for strings.HasPrefix(acc, "<") {
		end := strings.Index(acc, ">")
		if end < 0 {
			break
		}
		switch strings.ToLower(acc[1:end]) {
		case "control", "ctrl", "primary", "ctl":
			mods = append(mods, "ctrl")
		case "shift", "shft":
			mods = append(mods, "shift")
		case "alt", "mod1":
			mods = append(mods, "alt")
		case "super", "meta", "mod4", "hyper":
			mods = append(mods, "super")
		}
		acc = acc[end+1:]
	}
	key := strings.ToLower(acc)
	switch key {
	case "return", "kp_enter":
		key = "enter"
	case "kp_add":
		key = "plus"
	case "kp_subtract":
		key = "minus"
	}
	if key == "" {
		return ""
	}
	c := combo{mods: mods, key: key}
	return c.canonical()
}

// accelOf extracts the accelerator from an AT-SPI key binding ("n;<Alt>f:n;<Control>n").
func accelOf(binding string) string {
	parts := strings.Split(binding, ";")
	if len(parts) >= 3 {
		return parseAccel(parts[2])
	}
	if len(parts) == 1 && strings.Contains(binding, "<") {
		return parseAccel(binding)
	}
	return ""
}

func cmdShortcut(a map[string]string) (obj, *cmdError) {
	mode, e := parseMode(a["mode"])
	if e != nil {
		return nil, e
	}
	c, e := parseCombo(a["keys"])
	if e != nil {
		return nil, e
	}
	if e := delay(a); e != nil {
		return nil, e
	}
	appGiven := a["app"] != ""
	if !appGiven {
		if mode == "background" {
			return nil, errObj(obj{"error": "Background mode not possible for shortcut: a system-wide shortcut needs real input (pass --app to run it in an app)", "mode": "background"})
		}
		if dryRun() {
			return obj{"dryRun": true, "keys": a["keys"], "mode": "foreground"}, nil
		}
		x, e := X("shortcut")
		if e != nil {
			return nil, e
		}
		if e := x.requireXTest("shortcut"); e != nil {
			return nil, e
		}
		ks, _ := keySym(c.key)
		if err := x.pressKeysym(ks, c.modKeysyms()); err != nil {
			return nil, errf("Shortcut failed: %v", err)
		}
		return obj{"keys": a["keys"], "mode": "foreground", "method": "xtest"}, nil
	}
	if dryRun() {
		m, r, err := decideMode("shortcut", mode, true, "")
		if err != nil {
			return nil, err
		}
		return withDryRun(obj{"app": a["app"], "keys": a["keys"]}, m, r), nil
	}
	ap, e := findApp(a["app"], true)
	if e != nil {
		return nil, e
	}
	o := obj{"app": ap.Name, "keys": a["keys"]}
	m, reason, err := decideMode("shortcut", mode, true, "")
	if err != nil {
		return nil, err
	}
	if m == "background" {
		why := "no enabled menu item for " + c.canonical()
		if b, _, ae := ap.needA11y(); ae == nil {
			want := c.canonical()
			if want == "ctrl+a" {
				if n, te := editTarget(b, ap, ""); te == nil && n.ifaces[ifText] {
					_, cnt := b.textOf(n, 1)
					if b.setSelection(n, 0, cnt) {
						o["method"] = "accessibility"
						o["mode"] = "background"
						o["selected"] = cnt
						return o, nil
					}
				}
			}
			nodes, _ := snapshot(b, ap.A.root, 30, 6000, false)
			for _, n := range nodes {
				if !strings.Contains(n.role, "menu item") {
					continue
				}
				for i, act := range b.actions(n) {
					if accelOf(act.Key) == want {
						if !(n.has(stSensitive) || n.has(stEnabled)) {
							why = "the menu item '" + n.label() + "' for " + want + " is disabled"
							continue
						}
						var ok bool
						if b.call(n.bus, n.path, ifAction+".DoAction", int32(i)).Store(&ok) == nil && ok {
							o["method"] = "menu"
							o["menuItem"] = n.label()
							o["mode"] = "background"
							return o, nil
						}
					}
				}
			}
		} else {
			why = ae.Error()
		}
		if mode == "background" {
			return nil, errObj(obj{"error": "Background mode not possible for shortcut: " + why, "mode": "background", "app": ap.Name})
		}
		reason = why
	}
	x, xe := X("shortcut")
	if xe != nil {
		return nil, xe
	}
	if e := x.requireXTest("shortcut"); e != nil {
		return nil, e
	}
	fs := x.beginForeground(ap.frontWindow(x), mode == "auto")
	ks, _ := keySym(c.key)
	perr := x.pressKeysym(ks, c.modKeysyms())
	time.Sleep(60 * time.Millisecond)
	restored := fs.end()
	if perr != nil {
		return nil, errf("Shortcut failed: %v", perr)
	}
	o["method"] = "xtest"
	setFallback(o, "foreground", reason, restored)
	return o, nil
}

// ---------------------------------------------------------------------------
// Scroll

func cmdScroll(a map[string]string) (obj, *cmdError) {
	mode, e := parseMode(a["mode"])
	if e != nil {
		return nil, e
	}
	dx, dy := 0.0, 0.0
	if a["dx"] != "" {
		if dx, e = parseFloatArg("dx", a["dx"]); e != nil {
			return nil, e
		}
	}
	if a["dy"] != "" {
		if dy, e = parseFloatArg("dy", a["dy"]); e != nil {
			return nil, e
		}
	}
	hasPoint := a["x"] != "" && a["y"] != ""
	var px, py float64
	if hasPoint {
		if px, e = parseFloatArg("x", a["x"]); e != nil {
			return nil, e
		}
		if py, e = parseFloatArg("y", a["y"]); e != nil {
			return nil, e
		}
	}
	if dryRun() {
		m, r, err := decideMode("scroll", mode, true, "")
		if err != nil {
			return nil, err
		}
		o := obj{"dx": dx, "dy": dy}
		if hasPoint {
			o["x"], o["y"] = px, py
		}
		if a["app"] != "" {
			o["app"] = a["app"]
		}
		return withDryRun(o, m, r), nil
	}
	ap, e := targetApp(a["app"], true)
	if e != nil {
		return nil, e
	}
	o := obj{"app": ap.Name, "dx": dx, "dy": dy}
	var sx, sy float64
	if hasPoint {
		if sx, sy, e = screenPoint(ap, a["app"] != "", px, py); e != nil {
			return nil, e
		}
		o["screen"] = point(sx, sy)
	}
	m, reason, err := decideMode("scroll", mode, true, "")
	if err != nil {
		return nil, err
	}
	if m == "background" {
		why := "no scroll area found"
		if b, ai, ae := ap.needA11y(); ae == nil {
			var start *node
			if hasPoint {
				start = pressableAtAny(b, ai, sx, sy)
			} else if n, te := editTarget(b, ap, ""); te == nil {
				start = n
			}
			if pane := scrollPaneOf(b, ai, start); pane != nil {
				res, ok := scrollPane(b, pane, dx, dy)
				if ok {
					for k, v := range res {
						o[k] = v
					}
					o["method"] = "accessibility"
					o["mode"] = "background"
					if !hasPoint && pane.hasFrame {
						cx, cy := pane.frame.center()
						o["screen"] = point(cx, cy)
					}
					return o, nil
				}
				why = "the scroll area's scroll bars don't accept a value"
			}
		} else {
			why = ae.Error()
		}
		if mode == "background" {
			return nil, errObj(obj{"error": "Background mode not possible for scroll: " + why, "mode": "background", "app": ap.Name})
		}
		reason = why
	}
	x, xe := X("scroll")
	if xe != nil {
		return nil, xe
	}
	if e := x.requireXTest("scroll"); e != nil {
		return nil, e
	}
	fs := x.beginForeground(ap.frontWindow(x), mode == "auto")
	if !hasPoint {
		if r, _, ok := ap.windowOrigin(); ok {
			sx, sy = r.center()
		}
	}
	x.motion(int(sx), int(sy))
	x.sync()
	wheel := func(n int, up, down byte) {
		btn := down
		if n < 0 {
			btn, n = up, -n
		}
		for i := 0; i < n; i++ {
			x.button(btn, true)
			x.button(btn, false)
		}
	}
	// one wheel click scrolls about 50 pixels; negative dy scrolls down
	wheel(int(math.Round(-dy/50)), 5, 4)
	wheel(int(math.Round(dx/50)), 6, 7)
	x.sync()
	time.Sleep(60 * time.Millisecond)
	o["method"] = "xtest"
	setFallback(o, "foreground", reason, fs.end())
	return o, nil
}

// pressableAtAny returns the deepest element at a screen point in the app's windows.
func pressableAtAny(b *a11yBus, ai *appInfo, sx, sy float64) *node {
	kids := b.childrenOf(ai.root)
	b.loadAll(kids)
	for _, w := range kids {
		if w.hasFrame && w.frame.contains(sx, sy) && w.has(stShowing) {
			return b.accessibleAt(w, sx, sy)
		}
	}
	return nil
}

func scrollPaneOf(b *a11yBus, ai *appInfo, start *node) *node {
	for n := start; n != nil; n = n.parent {
		b.load(n)
		if n.role == "scroll pane" {
			return n
		}
	}
	// otherwise the first visible scroll pane of the active window
	nodes, _ := snapshot(b, ai.root, 30, 4000, false)
	for _, n := range nodes {
		if n.role == "scroll pane" {
			return n
		}
	}
	return nil
}

func scrollPane(b *a11yBus, pane *node, dx, dy float64) (obj, bool) {
	res := obj{}
	ok := false
	for _, k := range b.childrenOf(pane) {
		b.load(k)
		if k.role != "scroll bar" || !k.ifaces[ifValue] {
			continue
		}
		vertical := k.has(29)
		delta := -dy
		if !vertical {
			delta = dx
		}
		if delta == 0 {
			continue
		}
		get := func(p string) float64 {
			v, err := b.getProp(k.bus, k.path, ifValue, p)
			if err != nil {
				return 0
			}
			f, _ := v.Value().(float64)
			return f
		}
		cur, min, max := get("CurrentValue"), get("MinimumValue"), get("MaximumValue")
		nv := math.Max(min, math.Min(max, cur+delta))
		if b.setProp(k.bus, k.path, ifValue, "CurrentValue", nv) != nil {
			continue
		}
		ok = true
		frac := 0.0
		if max > min {
			frac = (get("CurrentValue") - min) / (max - min)
		}
		if vertical {
			res["vertical"] = math.Round(frac*1000) / 1000
		} else {
			res["horizontal"] = math.Round(frac*1000) / 1000
		}
	}
	return res, ok
}

func cmdFocus(a map[string]string) (obj, *cmdError) {
	if a["app"] == "" {
		return nil, errf("Provide the app to focus")
	}
	if dryRun() {
		return obj{"dryRun": true, "focused": a["app"], "mode": "foreground"}, nil
	}
	ap, e := findApp(a["app"], false)
	if e != nil {
		return nil, e
	}
	x, e := X("focus")
	if e != nil {
		return nil, e
	}
	w := ap.frontWindow(x)
	if w == 0 {
		return nil, errObj(obj{"error": ap.Name + " has no window to focus", "app": ap.Name})
	}
	if x.minimized(w) {
		xproto.MapWindow(x.c, w)
	}
	_ = x.activate(w)
	ok := x.waitActive(w, 2*time.Second)
	return obj{"focused": ap.Name, "app": ap.Name, "mode": "foreground", "success": ok}, nil
}
