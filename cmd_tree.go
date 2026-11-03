package main

import (
	"strings"

	"github.com/godbus/dbus/v5"
)

func sessionBus() (*dbus.Conn, error) {
	return dbus.SessionBus()
}

var menuFamily = map[string]bool{"menu": true, "menu item": true, "check menu item": true, "radio menu item": true, "tearoff menu item": true, "menu bar": true, "separator": true, "popup menu": true}

// snapshot loads the subtree of root to maxDepth and returns the visible nodes in document order.
// Hidden widgets (not SHOWING) and their subtrees are left out, except menus, which are only
// shown while open but can be pressed through accessibility at any time.
func snapshot(b *a11yBus, root *node, maxDepth, maxNodes int, keepHidden bool) ([]*node, bool) {
	var out []*node
	truncated := b.walk(root, maxDepth, maxNodes, func(n *node, depth int) bool {
		if !keepHidden && depth > 1 && !n.has(stShowing) && !menuFamily[n.role] && !(n.parent != nil && menuFamily[n.parent.role]) {
			return false
		}
		if n.has(6) { // defunct
			return false
		}
		out = append(out, n)
		return true
	})
	sortNodes(out)
	return out, truncated
}

// elem shapes a node like a find match.
func elem(b *a11yBus, n *node, origin *rect, withValue bool) obj {
	o := obj{
		"ref":     n.ref,
		"role":    n.role,
		"label":   n.label(),
		"enabled": n.has(stEnabled) || n.has(stSensitive),
		"focused": n.has(stFocused),
	}
	if withValue {
		if isPasswordRole(n.role) {
			o["value"] = ""
		} else {
			o["value"] = cut(b.valueOf(n), 4000)
		}
	}
	if n.hasFrame {
		o["frame"] = n.frame.obj()
		cx, cy := n.frame.center()
		o["center"] = point(cx, cy)
		if origin != nil {
			o["windowCenter"] = point(cx-origin.X, cy-origin.Y)
		}
	} else {
		o["frame"] = obj{}
	}
	return o
}

func shortElem(n *node) obj {
	return obj{"ref": n.ref, "role": n.role, "label": n.label()}
}

func appOrigin(a *app, given bool) *rect {
	if !given {
		return nil
	}
	if sessionType() != "x11" && sessionType() != "xwayland" {
		return nil
	}
	r, _, ok := a.windowOrigin()
	if !ok {
		return nil
	}
	return &r
}

func cmdRead(a map[string]string) (obj, *cmdError) {
	depth, e := parsePositive("depth", a["depth"], 10)
	if e != nil {
		return nil, e
	}
	ap, e := targetApp(a["app"], true)
	if e != nil {
		return nil, e
	}
	b, ai, e := ap.needA11y()
	if e != nil {
		return nil, e
	}
	nodes, truncated := snapshot(b, ai.root, depth, 3000, false)
	origin := appOrigin(ap, a["app"] != "")
	els := make([]any, 0, len(nodes))
	for _, n := range nodes {
		els = append(els, elem(b, n, origin, true))
	}
	o := obj{"app": ap.Name, "elements": els}
	if truncated {
		o["truncated"] = true
	}
	return o, nil
}

// findMatches searches the app's tree: label or value containing text (case-insensitive) and/or role.
func findMatches(b *a11yBus, ap *app, text, role string, origin *rect, loose bool) []any {
	nodes, _ := snapshot(b, ap.A.root, 30, 6000, false)
	matches := []any{}
	nt := normalizeLoose(text)
	for _, n := range nodes {
		if !roleMatches(n.role, role) {
			continue
		}
		if text != "" {
			hit := false
			if loose {
				hit = strings.Contains(normalizeLoose(n.label()), nt)
			} else {
				hit = containsFold(n.label(), text)
			}
			if !hit && (n.ifaces[ifText] || n.ifaces[ifValue]) {
				v := b.valueOf(n)
				if loose {
					hit = nt != "" && strings.Contains(normalizeLoose(v), nt)
				} else {
					hit = containsFold(v, text)
				}
			}
			if !hit {
				continue
			}
		}
		if menuFamily[n.role] && role == "" && !n.has(stShowing) {
			// closed menus would flood text matches; find them by role or through menu
			continue
		}
		matches = append(matches, elem(b, n, origin, true))
	}
	return matches
}

func cmdFind(a map[string]string) (obj, *cmdError) {
	if a["text"] == "" && a["role"] == "" {
		// an empty text with a role is fine; neither is not
	}
	ap, e := targetApp(a["app"], true)
	if e != nil {
		return nil, e
	}
	b, _, e := ap.needA11y()
	if e != nil {
		return nil, e
	}
	origin := appOrigin(ap, a["app"] != "")
	return obj{"app": ap.Name, "matches": findMatches(b, ap, a["text"], a["role"], origin, false)}, nil
}

// focusedNode returns the focused element, or nil.
func focusedNode(nodes []*node) *node {
	for _, n := range nodes {
		if n.has(stFocused) && !windowRoles[n.role] {
			return n
		}
	}
	return nil
}

// editTarget picks the element that text goes to: the focused editable element, else the first
// visible editable text element of the front window (multi-line first).
func editTarget(b *a11yBus, ap *app, ref string) (*node, *cmdError) {
	if ref != "" {
		n, e := b.resolve(ap.A, ref)
		if e != nil {
			return nil, e
		}
		return n, nil
	}
	nodes, _ := snapshot(b, ap.A.root, 30, 6000, false)
	if f := focusedNode(nodes); f != nil && (f.ifaces[ifEditableText] || f.ifaces[ifText]) {
		return f, nil
	}
	var single *node
	for _, n := range nodes {
		if n.ifaces[ifEditableText] && n.has(stEditable) && n.has(stShowing) {
			if n.has(stMultiLine) {
				return n, nil
			}
			if single == nil {
				single = n
			}
		}
	}
	if single != nil {
		return single, nil
	}
	return nil, errObj(obj{"error": "No focused text element in " + ap.Name + ": pass --element with a ref from find or state", "app": ap.Name})
}

func cmdState(a map[string]string) (obj, *cmdError) {
	depth, e := parseIntArg("depth", a["depth"], 3)
	if e != nil || depth < 0 {
		return nil, errf("Invalid --depth '%s': use 0 or a positive number", a["depth"])
	}
	limit, e := parsePositive("limit", a["limit"], 120)
	if e != nil {
		return nil, e
	}
	ap, e := targetApp(a["app"], true)
	if e != nil {
		return nil, e
	}
	o := obj{"depth": depth}
	var x *xconn
	if sessionType() == "x11" || sessionType() == "xwayland" {
		x, _ = X("state")
	}
	front, _ := frontmostApp(false)
	if front != nil {
		o["frontmost"] = front.summary()
	} else {
		o["frontmost"] = nil
	}
	appO := ap.summary()
	appO["frontmost"] = front != nil && front.Pid == ap.Pid
	if x != nil && len(ap.Windows) > 0 {
		hidden := true
		for _, w := range ap.Windows {
			if x.mapped(w) && !x.minimized(w) {
				hidden = false
			}
		}
		appO["hidden"] = hidden
	}
	o["app"] = appO
	b, ai, ae := ap.needA11y()
	wins := windowEntries(x, ap, b)
	if len(wins) > 20 {
		wins = wins[:20]
	}
	o["windows"] = wins
	if ae != nil {
		o["focused"] = nil
		o["dialog"] = nil
		o["elements"] = []any{}
		o["note"] = ae.Error()
		return o, nil
	}
	origin := appOrigin(ap, true)
	nodes, truncated := snapshot(b, ai.root, 40, 4000, false)
	if f := focusedNode(nodes); f != nil {
		o["focused"] = elem(b, f, origin, true)
	} else {
		o["focused"] = nil
	}
	dlg := findDialog(b, ap, nodes)
	if dlg != nil {
		o["dialog"] = describeDialog(b, ap, dlg, nodes, origin)
	} else {
		o["dialog"] = nil
	}
	// tree root: the dialog, else the focused/active window, else the first window
	var root *node
	if dlg != nil {
		root = dlg
	} else {
		for _, n := range nodes {
			if strings.Count(n.ref, ".") == 0 && windowRoles[n.role] && n.has(stActive) {
				root = n
				break
			}
		}
		if root == nil {
			for _, n := range nodes {
				if strings.Count(n.ref, ".") == 0 && windowRoles[n.role] {
					root = n
					break
				}
			}
		}
	}
	els := []any{}
	if root != nil && depth > 0 {
		o["root"] = root.ref
		byParent := map[*node][]*node{}
		for _, n := range nodes {
			if n.parent != nil {
				byParent[n.parent] = append(byParent[n.parent], n)
			}
		}
		var visit func(n *node, d int)
		visit = func(n *node, d int) {
			for _, c := range byParent[n] {
				if len(els) >= limit {
					truncated = true
					return
				}
				meaningful := actionableRoles[c.role] || textRoles[c.role] || (c.role == "label")
				nd := d
				if meaningful {
					nd = d + 1
					if nd > depth {
						continue
					}
					e := obj{"ref": c.ref, "role": c.role, "label": cut(c.label(), 80)}
					if v := b.valueOf(c); v != "" && !isPasswordRole(c.role) && v != c.label() {
						e["value"] = cut(v, 80)
					}
					if c.hasFrame {
						e["frame"] = obj{"x": int(c.frame.X), "y": int(c.frame.Y), "width": int(c.frame.W), "height": int(c.frame.H)}
					}
					if !(c.has(stEnabled) || c.has(stSensitive)) {
						e["enabled"] = false
					}
					if c.has(stFocused) {
						e["focused"] = true
					}
					els = append(els, e)
				}
				if menuFamily[c.role] && c.role != "menu bar" {
					continue // menus are listed by the menu command
				}
				visit(c, nd)
			}
		}
		visit(root, 0)
	}
	o["elements"] = els
	if truncated {
		o["truncated"] = true
		o["note"] = "the element list was cut at --limit " + a["limit"] + "; use find or a larger --limit/--depth for more"
		if a["limit"] == "" {
			o["note"] = "the element list was cut at --limit 120; use find or a larger --limit/--depth for more"
		}
	}
	return o, nil
}
