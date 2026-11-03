package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func cmdSelect(a map[string]string) (obj, *cmdError) {
	mode, e := parseMode(a["mode"])
	if e != nil {
		return nil, e
	}
	all, e := parseBool("all", a["all"], false)
	if e != nil {
		return nil, e
	}
	forms := 0
	if all {
		forms++
	}
	if a["text"] != "" {
		forms++
	}
	if a["start"] != "" || a["length"] != "" {
		forms++
	}
	if forms != 1 {
		return nil, errf("Give exactly one of --all true, --text <text>, or --start N --length N")
	}
	occurrence, e := parsePositive("occurrence", a["occurrence"], 1)
	if e != nil {
		return nil, e
	}
	start, length := 0, 0
	if a["start"] != "" || a["length"] != "" {
		if a["start"] == "" || a["length"] == "" {
			return nil, errf("--start and --length go together")
		}
		if start, e = parseIntArg("start", a["start"], 0); e != nil || start < 0 {
			return nil, errf("Invalid --start '%s': use 0 or more", a["start"])
		}
		if length, e = parseIntArg("length", a["length"], 0); e != nil || length < 0 {
			return nil, errf("Invalid --length '%s': use 0 or more", a["length"])
		}
	}
	if dryRun() {
		m, r, err := decideMode("select", mode, true, "")
		if err != nil {
			return nil, err
		}
		o := obj{}
		if a["app"] != "" {
			o["app"] = a["app"]
		}
		return withDryRun(o, m, r), nil
	}
	ap, e := targetApp(a["app"], true)
	if e != nil {
		return nil, e
	}
	b, _, e := ap.needA11y()
	if e != nil {
		return nil, e
	}
	n, e := editTarget(b, ap, a["element"])
	if e != nil {
		return nil, e
	}
	if !n.ifaces[ifText] {
		return nil, errObj(obj{"error": fmt.Sprintf("Element %s (%s) doesn't take an accessibility text selection", n.ref, n.role), "app": ap.Name})
	}
	full, count := b.textOf(n, 0)
	runes := []rune(full)
	switch {
	case all:
		start, length = 0, count
	case a["text"] != "":
		lower := []rune(strings.ToLower(full))
		needle := []rune(strings.ToLower(a["text"]))
		var hits []int
		for i := 0; i+len(needle) <= len(lower); i++ {
			if string(lower[i:i+len(needle)]) == string(needle) {
				hits = append(hits, i)
			}
		}
		if len(hits) < occurrence {
			return nil, errObj(obj{"error": fmt.Sprintf("Text '%s' not found (occurrence %d; %d occurrences in the element)", a["text"], occurrence, len(hits)), "occurrences": len(hits), "app": ap.Name})
		}
		start, length = hits[occurrence-1], len(needle)
	default:
		if start+length > len(runes) {
			return nil, errObj(obj{"error": fmt.Sprintf("Range %d+%d is past the end of the text (%d characters)", start, length, len(runes)), "app": ap.Name})
		}
	}
	m, _, err := decideMode("select", mode, true, "")
	if err != nil {
		return nil, err
	}
	_ = m
	if !b.setSelection(n, start, start+length) {
		if mode == "background" {
			return nil, errObj(obj{"error": "Background mode not possible for select: the element refused the selection", "mode": "background", "app": ap.Name})
		}
		return nil, errObj(obj{"error": "The element refused the accessibility selection", "app": ap.Name})
	}
	sel := string(runes[start:min(start+length, len(runes))])
	o := obj{"app": ap.Name, "element": shortElem(n), "method": "accessibility", "mode": "background", "range": obj{"start": start, "length": length}}
	if len([]rune(sel)) > 4000 {
		o["selected"] = cut(sel, 4000)
		o["selectedTruncated"] = true
	} else {
		o["selected"] = sel
	}
	return o, nil
}

func cmdSet(a map[string]string) (obj, *cmdError) {
	mode, e := parseMode(a["mode"])
	if e != nil {
		return nil, e
	}
	if a["element"] == "" {
		return nil, errf("Provide --element <ref> from find, state or read")
	}
	n := 0
	for _, k := range []string{"value", "checked", "option"} {
		if a[k] != "" {
			n++
		}
	}
	if n != 1 {
		return nil, errf("Give exactly one of --value, --checked true|false, or --option <title>")
	}
	var want bool
	if a["checked"] != "" {
		if want, e = parseBool("checked", a["checked"], false); e != nil {
			return nil, e
		}
	}
	if dryRun() {
		m, r, err := decideMode("set", mode, true, "")
		if err != nil {
			return nil, err
		}
		o := obj{"element": a["element"]}
		if a["app"] != "" {
			o["app"] = a["app"]
		}
		return withDryRun(o, m, r), nil
	}
	if _, _, err := decideMode("set", mode, true, ""); err != nil {
		return nil, err
	}
	ap, e := targetApp(a["app"], true)
	if e != nil {
		return nil, e
	}
	b, ai, e := ap.needA11y()
	if e != nil {
		return nil, e
	}
	el, e := b.resolve(ai, a["element"])
	if e != nil {
		return nil, e
	}
	o := obj{"app": ap.Name, "element": shortElem(el), "mode": "background"}
	fail := func(msg string) (obj, *cmdError) {
		return nil, errObj(obj{"error": msg, "app": ap.Name, "element": shortElem(el)})
	}
	switch {
	case a["value"] != "":
		v := a["value"]
		if el.ifaces[ifValue] && !el.ifaces[ifEditableText] {
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return fail(fmt.Sprintf("Element %s (%s) takes a number: '%s' isn't one", el.ref, el.role, v))
			}
			before := b.valueOf(el)
			if err := b.setProp(el.bus, el.path, ifValue, "CurrentValue", f); err != nil {
				return fail("The app refused the value: " + err.Error())
			}
			time.Sleep(50 * time.Millisecond)
			o["before"], o["after"], o["method"] = before, b.valueOf(el), "Value"
			return o, nil
		}
		if !el.ifaces[ifEditableText] {
			return fail(fmt.Sprintf("Element %s (%s) doesn't take a value: use --checked or --option, or click it", el.ref, el.role))
		}
		before, _ := b.textOf(el, 0)
		if !b.setTextContents(el, v) {
			return fail("The app refused the new text")
		}
		time.Sleep(50 * time.Millisecond)
		after, _ := b.textOf(el, 0)
		o["before"], o["after"], o["method"] = before, after, "EditableText"
		return o, nil
	case a["checked"] != "":
		b.refreshStates(el)
		before := el.has(stChecked) || el.has(stPressed)
		o["before"] = before
		if before == want {
			o["after"], o["method"] = before, "unchanged"
			return o, nil
		}
		for i := 0; i < 2; i++ {
			if _, err := b.press(el); err != nil {
				return fail("Pressing the element failed: " + err.Error())
			}
			time.Sleep(80 * time.Millisecond)
			b.refreshStates(el)
			now := el.has(stChecked) || el.has(stPressed)
			if now == want {
				break
			}
			if now == before { // a press that changes nothing is not repeated
				break
			}
			before = now
		}
		after := el.has(stChecked) || el.has(stPressed)
		o["after"], o["method"] = after, "press"
		if after != want {
			return fail(fmt.Sprintf("The element stayed %v after pressing it", after))
		}
		return o, nil
	default:
		return setOption(b, ap, el, a["option"], o)
	}
}

func setOption(b *a11yBus, ap *app, el *node, option string, o obj) (obj, *cmdError) {
	want := normalizeTitle(option)
	before := el.label()
	if v := b.valueOf(el); v != "" {
		before = v
	}
	// candidate options: descendants that are menu items, list items, radio buttons or tabs
	var opts []*node
	nodes, _ := snapshot(b, el, 4, 500, true)
	for _, n := range nodes {
		switch n.role {
		case "menu item", "check menu item", "radio menu item", "list item", "radio button", "page tab", "table cell":
			if n.label() != "" {
				opts = append(opts, n)
			}
		}
	}
	var titles []string
	for _, n := range opts {
		titles = append(titles, n.label())
		if normalizeTitle(n.label()) != want {
			continue
		}
		method := ""
		if el.ifaces[ifSelection] && n.parent == el {
			if idx := indexOf(n.ref); idx >= 0 {
				var ok bool
				if b.call(el.bus, el.path, ifSelection+".SelectChild", int32(idx)).Store(&ok) == nil && ok {
					method = "Selection"
				}
			}
		}
		if method == "" {
			if _, err := b.press(n); err != nil {
				return nil, errObj(obj{"error": "Picking the option failed: " + err.Error(), "app": ap.Name})
			}
			method = "press"
		}
		time.Sleep(120 * time.Millisecond)
		b.refreshName(el)
		after := el.name
		if v := b.valueOf(el); v != "" {
			after = v
		}
		if after == "" || after == before && normalizeTitle(before) != want {
			after = n.label()
		}
		o["before"], o["after"], o["method"] = before, after, method
		return o, nil
	}
	return nil, errObj(obj{"error": fmt.Sprintf("No option '%s'. Available: %s", option, strings.Join(titles, ", ")), "app": ap.Name, "options": titles})
}

func indexOf(ref string) int {
	p := strings.Split(ref, ".")
	i, err := strconv.Atoi(p[len(p)-1])
	if err != nil {
		return -1
	}
	return i
}

// ---------------------------------------------------------------------------
// Menus

func menuBars(b *a11yBus, ap *app) []*node {
	nodes, _ := snapshot(b, ap.A.root, 12, 4000, true)
	var bars []*node
	for _, n := range nodes {
		if n.role == "menu bar" && n.has(stShowing) {
			bars = append(bars, n)
		}
	}
	if len(bars) == 0 {
		for _, n := range nodes {
			if n.role == "menu bar" {
				bars = append(bars, n)
			}
		}
	}
	return bars
}

func menuItems(b *a11yBus, n *node) []*node {
	var out []*node
	for _, k := range b.childrenOf(n) {
		b.load(k)
		if k.role == "separator" || k.role == "tearoff menu item" {
			continue
		}
		if k.role == "menu" && k.label() == "" { // a submenu container: flatten
			out = append(out, menuItems(b, k)...)
			continue
		}
		if menuFamily[k.role] {
			out = append(out, k)
		}
	}
	return out
}

func menuEntry(b *a11yBus, n *node, depth, level int, count *int) obj {
	*count++
	o := obj{"title": strings.TrimSpace(n.label()), "ref": n.ref}
	if n.has(stChecked) {
		o["checked"] = true
	}
	if !(n.has(stSensitive) || n.has(stEnabled)) {
		o["enabled"] = false
	}
	for _, act := range b.actions(n) {
		if s := accelOf(act.Key); s != "" {
			o["shortcut"] = s
			break
		}
	}
	kids := menuItems(b, n)
	if len(kids) > 0 {
		if level >= depth {
			o["submenu"] = true
		} else {
			var ch []any
			for _, k := range kids {
				if *count >= 600 {
					break
				}
				ch = append(ch, menuEntry(b, k, depth, level+1, count))
			}
			o["children"] = ch
		}
	}
	return o
}

func cmdMenu(a map[string]string) (obj, *cmdError) {
	mode, e := parseMode(a["mode"])
	if e != nil {
		return nil, e
	}
	depth, e := parsePositive("depth", a["depth"], 2)
	if e != nil {
		return nil, e
	}
	if a["path"] != "" && dryRun() {
		m, r, err := decideMode("menu", mode, true, "")
		if err != nil {
			return nil, err
		}
		o := obj{"path": splitPath(a["path"])}
		if a["app"] != "" {
			o["app"] = a["app"]
		}
		return withDryRun(o, m, r), nil
	}
	ap, e := targetApp(a["app"], true)
	if e != nil {
		return nil, e
	}
	b, _, e := ap.needA11y()
	if e != nil {
		return nil, e
	}
	bars := menuBars(b, ap)
	if len(bars) == 0 {
		return nil, errObj(obj{"error": ap.Name + " has no menu bar in its accessibility tree (apps with a header bar or hamburger menu use buttons instead: find --role AXButton)", "app": ap.Name})
	}
	var top []*node
	for _, bar := range bars {
		top = append(top, menuItems(b, bar)...)
	}
	if a["path"] == "" {
		count := 0
		var menus []any
		for _, m := range top {
			if count >= 600 {
				break
			}
			menus = append(menus, menuEntry(b, m, depth, 1, &count))
		}
		o := obj{"app": ap.Name, "depth": depth, "menus": menus}
		if count >= 600 {
			o["truncated"] = true
		}
		return o, nil
	}
	path := splitPath(a["path"])
	level := top
	var cur *node
	var resolved []string
	for i, title := range path {
		var hit *node
		for _, n := range level {
			if normalizeTitle(n.label()) == normalizeTitle(title) {
				hit = n
				break
			}
		}
		if hit == nil {
			var avail []string
			for _, n := range level {
				avail = append(avail, strings.TrimSpace(n.label()))
			}
			where := "the menu bar"
			if i > 0 {
				where = "'" + resolved[len(resolved)-1] + "'"
			}
			return nil, errObj(obj{"error": fmt.Sprintf("No menu item '%s' under %s. Available: %s", title, where, strings.Join(avail, ", ")), "app": ap.Name})
		}
		cur = hit
		resolved = append(resolved, strings.TrimSpace(hit.label()))
		if i < len(path)-1 {
			level = menuItems(b, hit)
		}
	}
	if kids := menuItems(b, cur); len(kids) > 0 && cur.role == "menu" {
		count := 0
		return obj{"app": ap.Name, "path": resolved, "ref": cur.ref, "items": menuEntry(b, cur, 2, 1, &count)["children"]}, nil
	}
	if !(cur.has(stSensitive) || cur.has(stEnabled)) {
		return nil, errObj(obj{"error": fmt.Sprintf("Menu item '%s' is disabled", strings.Join(resolved, " > ")), "app": ap.Name})
	}
	if _, _, err := decideMode("menu", mode, true, ""); err != nil {
		return nil, err
	}
	act, err := b.press(cur)
	if err != nil {
		return nil, errObj(obj{"error": "Pressing the menu item failed: " + err.Error(), "app": ap.Name})
	}
	o := obj{"app": ap.Name, "path": resolved, "pressed": strings.TrimSpace(cur.label()), "ref": cur.ref, "method": act, "mode": "background"}
	if strings.Contains(cur.role, "check") || strings.Contains(cur.role, "radio") {
		time.Sleep(100 * time.Millisecond)
		b.refreshStates(cur)
		o["checked"] = cur.has(stChecked)
	}
	return o, nil
}

func splitPath(p string) []string {
	var out []string
	for _, s := range strings.Split(p, ">") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Dialogs

func findDialog(b *a11yBus, ap *app, nodes []*node) *node {
	if ap.A == nil {
		return nil
	}
	for _, k := range b.childrenOf(ap.A.root) {
		b.load(k)
		if (dialogRoles[k.role] || k.has(stModal)) && k.has(stShowing) && !k.has(stIconified) {
			return k
		}
	}
	return nil
}

func describeDialog(b *a11yBus, ap *app, d *node, _ []*node, origin *rect) obj {
	nodes, _ := snapshot(b, d, 25, 2000, false)
	kind := "dialog"
	switch d.role {
	case "alert":
		kind = "alert"
	case "file chooser":
		kind = "open-panel"
	}
	var texts []string
	buttons := []any{}
	fields := []any{}
	hasSave := false
	for _, n := range nodes {
		switch {
		case n.role == "push button" || n.role == "toggle button":
			if len(buttons) >= 25 || n.label() == "" {
				continue
			}
			e := elem(b, n, origin, false)
			e["title"] = n.label()
			e["default"] = n.has(stIsDefault)
			t := normalizeTitle(n.label())
			e["cancel"] = t == "cancel"
			buttons = append(buttons, e)
			if t == "save" || t == "save as" {
				hasSave = true
			}
		case isTextInputRole(n.role) || n.role == "check box" || n.role == "combo box" || n.role == "radio button":
			if len(fields) < 15 {
				fields = append(fields, elem(b, n, origin, true))
			}
		case n.role == "label" || n.role == "static" || n.role == "paragraph":
			if s := strings.TrimSpace(n.label()); s != "" {
				texts = append(texts, s)
			}
		}
		if n.role == "file chooser" {
			kind = "open-panel"
		}
	}
	if kind == "open-panel" && hasSave {
		kind = "save-panel"
	}
	if texts == nil {
		texts = []string{}
	}
	o := obj{"app": ap.Name, "kind": kind, "title": d.label(), "text": texts, "buttons": buttons, "fields": fields, "ref": d.ref, "role": d.role}
	if d.hasFrame {
		o["frame"] = d.frame.obj()
	}
	return o
}

func cmdDialog(a map[string]string) (obj, *cmdError) {
	mode, e := parseMode(a["mode"])
	if e != nil {
		return nil, e
	}
	if a["press"] != "" && dryRun() {
		m, r, err := decideMode("dialog", mode, true, "")
		if err != nil {
			return nil, err
		}
		o := obj{"press": a["press"]}
		if a["app"] != "" {
			o["app"] = a["app"]
		}
		return withDryRun(o, m, r), nil
	}
	var candidates []*app
	if a["app"] != "" {
		ap, e := findApp(a["app"], true)
		if e != nil {
			return nil, e
		}
		candidates = []*app{ap}
	} else {
		if f, err := frontmostApp(true); err == nil {
			candidates = append(candidates, f)
		}
		candidates = append(candidates, listApps(true)...)
	}
	b, e := A11y("dialog")
	if e != nil {
		return nil, e
	}
	for _, ap := range candidates {
		if ap.A == nil {
			continue
		}
		prepareOrigins(b, ap)
		d := findDialog(b, ap, nil)
		if d == nil {
			continue
		}
		origin := appOrigin(ap, true)
		desc := describeDialog(b, ap, d, nil, origin)
		if a["press"] == "" {
			return obj{"app": ap.Name, "dialog": desc}, nil
		}
		if _, _, err := decideMode("dialog", mode, true, ""); err != nil {
			return nil, err
		}
		dnodes, _ := snapshot(b, d, 25, 2000, false)
		want := normalizeTitle(a["press"])
		var titles []string
		for _, n := range dnodes {
			if n.role != "push button" && n.role != "toggle button" {
				continue
			}
			titles = append(titles, n.label())
			if normalizeTitle(n.label()) != want {
				continue
			}
			act, err := b.press(n)
			if err != nil {
				return nil, errObj(obj{"error": "Pressing the button failed: " + err.Error(), "app": ap.Name})
			}
			closed := false
			var next obj
			for i := 0; i < 20; i++ {
				time.Sleep(100 * time.Millisecond)
				ap.A.root.children = nil
				nd := findDialog(b, ap, nil)
				if nd == nil {
					closed = true
					break
				}
				if nd.path != d.path {
					closed = true
					next = describeDialog(b, ap, nd, nil, origin)
					break
				}
			}
			o := obj{"app": ap.Name, "closed": closed, "dialog": obj{"kind": desc["kind"], "title": desc["title"]}, "kind": desc["kind"], "method": act, "mode": "background", "pressed": n.label()}
			if next != nil {
				o["next"] = next
			}
			return o, nil
		}
		return nil, errObj(obj{"error": fmt.Sprintf("No button '%s' in the dialog. Buttons: %s", a["press"], strings.Join(titles, ", ")), "app": ap.Name, "buttons": titles})
	}
	if a["press"] != "" {
		return nil, errf("No dialog is open%s", map[bool]string{true: " in " + a["app"], false: ""}[a["app"] != ""])
	}
	o := obj{"dialog": nil}
	if a["app"] != "" && len(candidates) > 0 {
		o["app"] = candidates[0].Name
	}
	return o, nil
}
