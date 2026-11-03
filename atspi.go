package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	ifAccessible   = "org.a11y.atspi.Accessible"
	ifComponent    = "org.a11y.atspi.Component"
	ifAction       = "org.a11y.atspi.Action"
	ifText         = "org.a11y.atspi.Text"
	ifEditableText = "org.a11y.atspi.EditableText"
	ifValue        = "org.a11y.atspi.Value"
	ifSelection    = "org.a11y.atspi.Selection"
	ifApplication  = "org.a11y.atspi.Application"
	ifProps        = "org.freedesktop.DBus.Properties"
)

// AT-SPI state bits
const (
	stActive      = 1
	stChecked     = 4
	stEditable    = 7
	stEnabled     = 8
	stExpanded    = 10
	stFocusable   = 11
	stFocused     = 12
	stIconified   = 15
	stModal       = 16
	stMultiLine   = 17
	stPressed     = 20
	stSelected    = 23
	stSensitive   = 24
	stShowing     = 25
	stSingleLine  = 26
	stVisible     = 30
	stIndeterm    = 32
	stIsDefault   = 39
	stCheckable   = 41
	stReadOnly    = 43
)

const callTimeout = 3 * time.Second

type a11yBus struct {
	conn *dbus.Conn
}

var a11y *a11yBus
var a11yErr error
var a11yOnce sync.Once

func a11yAddress() (string, error) {
	if a := os.Getenv("AT_SPI_BUS_ADDRESS"); a != "" {
		return a, nil
	}
	sess, err := dbus.SessionBus()
	if err == nil {
		var addr string
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()
		if e := sess.Object("org.a11y.Bus", "/org/a11y/bus").CallWithContext(ctx, "org.a11y.Bus.GetAddress", 0).Store(&addr); e == nil && addr != "" {
			return addr, nil
		} else if e != nil {
			err = e
		}
	}
	// X11 sessions also publish the address on the root window.
	if sessionType() == "x11" || sessionType() == "xwayland" {
		if x, e := X("accessibility"); e == nil {
			if s := x.propString(x.root, "AT_SPI_BUS"); s != "" {
				return s, nil
			}
		}
	}
	if err == nil {
		err = fmt.Errorf("the accessibility bus (org.a11y.Bus) is not running")
	}
	return "", err
}

const a11yGuidance = "Install at-spi2-core and make sure the session runs it (it starts with the desktop; in a bare X session run under dbus-run-session), then enable toolkit accessibility: gsettings set org.gnome.desktop.interface toolkit-accessibility true. Apps started before accessibility was enabled must be restarted. Do not set NO_AT_BRIDGE=1."

func A11y(command string) (*a11yBus, *cmdError) {
	if denyPermissions() {
		return nil, errObj(obj{"error": "Accessibility permission not granted: " + a11yGuidance, "permission": "accessibility"})
	}
	a11yOnce.Do(func() {
		addr, err := a11yAddress()
		if err != nil {
			a11yErr = err
			return
		}
		conn, err := dbus.Dial(addr)
		if err != nil {
			a11yErr = err
			return
		}
		if err = conn.Auth(nil); err != nil {
			conn.Close()
			a11yErr = err
			return
		}
		if err = conn.Hello(); err != nil {
			conn.Close()
			a11yErr = err
			return
		}
		a11y = &a11yBus{conn: conn}
	})
	if a11yErr != nil {
		return nil, errObj(obj{"error": "Accessibility permission not granted: AT-SPI is not available (" + a11yErr.Error() + "). " + a11yGuidance, "permission": "accessibility"})
	}
	return a11y, nil
}

// node is one accessible object.
type node struct {
	bus  string
	path dbus.ObjectPath

	loaded   bool
	role     string
	name     string
	desc     string
	nchild   int
	states   [2]uint32
	ifaces   map[string]bool
	app      *appInfo
	ref      string
	parent   *node
	children []*node
	frame    rect
	hasFrame bool
}

type appInfo struct {
	bus       string
	name      string
	pid       int
	toolkit   string
	version   string
	winCoords bool // GTK4: extents are window-relative
	root      *node
	origins   map[int]rect // top-level child index -> window content rect (screen)
}

func (b *a11yBus) obj(bus string, path dbus.ObjectPath) dbus.BusObject {
	return b.conn.Object(bus, path)
}

func (b *a11yBus) call(bus string, path dbus.ObjectPath, method string, args ...any) *dbus.Call {
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	return b.obj(bus, path).CallWithContext(ctx, method, 0, args...)
}

func (b *a11yBus) getProp(bus string, path dbus.ObjectPath, iface, prop string) (dbus.Variant, error) {
	var v dbus.Variant
	err := b.call(bus, path, ifProps+".Get", iface, prop).Store(&v)
	return v, err
}

func (b *a11yBus) setProp(bus string, path dbus.ObjectPath, iface, prop string, value any) error {
	return b.call(bus, path, ifProps+".Set", iface, prop, dbus.MakeVariant(value)).Err
}

type objRef struct {
	Bus  string
	Path dbus.ObjectPath
}

func (b *a11yBus) pidOf(bus string) int {
	var pid uint32
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	if err := b.conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.GetConnectionUnixProcessID", 0, bus).Store(&pid); err != nil {
		return 0
	}
	return int(pid)
}

// apps lists the applications registered on the accessibility bus.
func (b *a11yBus) apps() []*appInfo {
	var kids []objRef
	if err := b.call("org.a11y.atspi.Registry", "/org/a11y/atspi/accessible/root", ifAccessible+".GetChildren").Store(&kids); err != nil {
		return nil
	}
	out := make([]*appInfo, len(kids))
	var wg sync.WaitGroup
	for i, k := range kids {
		wg.Add(1)
		go func(i int, k objRef) {
			defer wg.Done()
			a := &appInfo{bus: k.Bus, origins: map[int]rect{}}
			root := &node{bus: k.Bus, path: k.Path, app: a, ref: ""}
			a.root = root
			if v, err := b.getProp(k.Bus, k.Path, ifAccessible, "Name"); err == nil {
				a.name, _ = v.Value().(string)
			}
			a.pid = b.pidOf(k.Bus)
			if v, err := b.getProp(k.Bus, k.Path, ifApplication, "ToolkitName"); err == nil {
				a.toolkit, _ = v.Value().(string)
			}
			if v, err := b.getProp(k.Bus, k.Path, ifApplication, "Version"); err == nil {
				a.version, _ = v.Value().(string)
			}
			a.winCoords = strings.EqualFold(a.toolkit, "GTK") && strings.HasPrefix(a.version, "4")
			out[i] = a
		}(i, k)
	}
	wg.Wait()
	res := out[:0]
	self := os.Getpid()
	for _, a := range out {
		if a != nil && a.pid != self {
			res = append(res, a)
		}
	}
	return res
}

func (b *a11yBus) load(n *node) {
	if n.loaded {
		return
	}
	n.loaded = true
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		var props map[string]dbus.Variant
		if err := b.call(n.bus, n.path, ifProps+".GetAll", ifAccessible).Store(&props); err == nil {
			if v, ok := props["Name"]; ok {
				n.name, _ = v.Value().(string)
			}
			if v, ok := props["Description"]; ok {
				n.desc, _ = v.Value().(string)
			}
			if v, ok := props["ChildCount"]; ok {
				if c, ok := v.Value().(int32); ok {
					n.nchild = int(c)
				}
			}
		}
	}()
	go func() {
		defer wg.Done()
		_ = b.call(n.bus, n.path, ifAccessible+".GetRoleName").Store(&n.role)
	}()
	go func() {
		defer wg.Done()
		var st []uint32
		if b.call(n.bus, n.path, ifAccessible+".GetState").Store(&st) == nil {
			for i := 0; i < 2 && i < len(st); i++ {
				n.states[i] = st[i]
			}
		}
	}()
	go func() {
		defer wg.Done()
		var ifs []string
		n.ifaces = map[string]bool{}
		if b.call(n.bus, n.path, ifAccessible+".GetInterfaces").Store(&ifs) == nil {
			for _, i := range ifs {
				n.ifaces[i] = true
			}
		}
	}()
	wg.Wait()
	if n.ifaces[ifComponent] {
		b.loadFrame(n)
	}
}

func (b *a11yBus) loadFrame(n *node) {
	coord := uint32(0) // screen
	if n.app != nil && n.app.winCoords {
		coord = 1 // window
	}
	var ext struct{ X, Y, W, H int32 }
	if b.call(n.bus, n.path, ifComponent+".GetExtents", coord).Store(&ext) != nil {
		return
	}
	r := rect{float64(ext.X), float64(ext.Y), float64(ext.W), float64(ext.H)}
	if n.app != nil && n.app.winCoords {
		if o, ok := n.app.origins[topIndex(n.ref)]; ok {
			r.X += o.X
			r.Y += o.Y
		}
	}
	if r.W > 0 && r.H > 0 && !(r.X == -2147483648 || r.Y == -2147483648) {
		n.frame = r
		n.hasFrame = true
	}
}

func topIndex(ref string) int {
	p := strings.SplitN(ref, ".", 2)
	i, err := strconv.Atoi(p[0])
	if err != nil {
		return -1
	}
	return i
}

func (n *node) has(bit int) bool {
	return n.states[bit/32]&(1<<(uint(bit)%32)) != 0
}

func (n *node) label() string {
	if n.name != "" {
		return n.name
	}
	return n.desc
}

func childRef(parent string, i int) string {
	if parent == "" {
		return strconv.Itoa(i)
	}
	return parent + "." + strconv.Itoa(i)
}

func (b *a11yBus) childrenOf(n *node) []*node {
	if n.children != nil {
		return n.children
	}
	var kids []objRef
	if err := b.call(n.bus, n.path, ifAccessible+".GetChildren").Store(&kids); err != nil {
		// Some objects only answer GetChildAtIndex.
		b.load(n)
		for i := 0; i < n.nchild && i < 2000; i++ {
			var k objRef
			if b.call(n.bus, n.path, ifAccessible+".GetChildAtIndex", int32(i)).Store(&k) == nil {
				kids = append(kids, k)
			}
		}
	}
	n.children = make([]*node, 0, len(kids))
	for i, k := range kids {
		if k.Path == "/org/a11y/atspi/null" {
			continue
		}
		n.children = append(n.children, &node{bus: k.Bus, path: k.Path, app: n.app, ref: childRef(n.ref, i), parent: n})
	}
	return n.children
}

// walk visits nodes breadth-first to maxDepth (root's children are depth 1), loading each level
// in parallel. visit returns false to skip a node's subtree. maxNodes bounds the walk.
func (b *a11yBus) walk(root *node, maxDepth int, maxNodes int, visit func(n *node, depth int) bool) bool {
	level := []*node{root}
	count := 0
	truncated := false
	for depth := 0; depth <= maxDepth && len(level) > 0; depth++ {
		b.loadAll(level)
		var next []*node
		for _, n := range level {
			if depth > 0 {
				count++
				if count > maxNodes {
					truncated = true
					break
				}
				if !visit(n, depth) {
					continue
				}
			}
			if depth < maxDepth && (n.nchild > 0 || depth == 0) && !n.has(31) /* manages descendants */ {
				next = append(next, n)
			}
		}
		if truncated {
			break
		}
		// fetch the next level's children lists in parallel
		var wg sync.WaitGroup
		sem := make(chan struct{}, 24)
		for _, n := range next {
			wg.Add(1)
			sem <- struct{}{}
			go func(n *node) {
				defer wg.Done()
				b.childrenOf(n)
				<-sem
			}(n)
		}
		wg.Wait()
		level = level[:0]
		for _, n := range next {
			// ordering: depth-first-like ordering is restored by callers sorting on ref
			level = append(level, n.children...)
		}
	}
	return truncated
}

func (b *a11yBus) loadAll(nodes []*node) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 32)
	for _, n := range nodes {
		if n.loaded {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(n *node) {
			defer wg.Done()
			b.load(n)
			<-sem
		}(n)
	}
	wg.Wait()
}

// refLess orders refs as a depth-first (document order) traversal.
func refLess(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		x, _ := strconv.Atoi(pa[i])
		y, _ := strconv.Atoi(pb[i])
		if x != y {
			return x < y
		}
	}
	return len(pa) < len(pb)
}

func sortNodes(ns []*node) {
	sort.SliceStable(ns, func(i, j int) bool { return refLess(ns[i].ref, ns[j].ref) })
}

// resolve finds a node by ref (child-index path from the app root).
func (b *a11yBus) resolve(a *appInfo, ref string) (*node, *cmdError) {
	ref = strings.TrimSpace(ref)
	cur := a.root
	if ref == "" {
		b.load(cur)
		return cur, nil
	}
	for _, p := range strings.Split(ref, ".") {
		i, err := strconv.Atoi(p)
		if err != nil || i < 0 {
			return nil, errf("Invalid element ref '%s': use a ref from find, read or state", ref)
		}
		kids := b.childrenOf(cur)
		var found *node
		for _, k := range kids {
			if k.ref == childRef(cur.ref, i) {
				found = k
				break
			}
		}
		if found == nil {
			return nil, errf("Element '%s' not found in %s (the UI may have changed: run find or state again)", ref, a.displayName())
		}
		cur = found
	}
	b.load(cur)
	return cur, nil
}

func (a *appInfo) displayName() string {
	return a.name
}

// textOf returns the element's text through the Text interface.
func (b *a11yBus) textOf(n *node, max int) (string, int) {
	if !n.ifaces[ifText] {
		return "", 0
	}
	var count int32
	if v, err := b.getProp(n.bus, n.path, ifText, "CharacterCount"); err == nil {
		count, _ = v.Value().(int32)
	}
	end := int32(-1)
	if max > 0 && int(count) > max {
		end = int32(max)
	}
	var s string
	_ = b.call(n.bus, n.path, ifText+".GetText", int32(0), end).Store(&s)
	return s, int(count)
}

func (b *a11yBus) valueOf(n *node) string {
	if n.ifaces[ifValue] {
		if v, err := b.getProp(n.bus, n.path, ifValue, "CurrentValue"); err == nil {
			if f, ok := v.Value().(float64); ok {
				return strconv.FormatFloat(f, 'f', -1, 64)
			}
		}
	}
	if n.ifaces[ifText] && !isPasswordRole(n.role) {
		s, _ := b.textOf(n, 4000)
		if s != n.name || isTextInputRole(n.role) {
			return s
		}
	}
	return ""
}

type actionInfo struct {
	Name, Desc, Key string
}

func (b *a11yBus) actions(n *node) []actionInfo {
	if !n.ifaces[ifAction] {
		return nil
	}
	var acts []struct{ A, B, C string }
	if b.call(n.bus, n.path, ifAction+".GetActions").Store(&acts) == nil {
		out := make([]actionInfo, len(acts))
		for i, a := range acts {
			out[i] = actionInfo{a.A, a.B, a.C}
		}
		return out
	}
	var cnt int32
	if v, err := b.getProp(n.bus, n.path, ifAction, "NActions"); err == nil {
		cnt, _ = v.Value().(int32)
	}
	var out []actionInfo
	for i := int32(0); i < cnt; i++ {
		var name, key string
		_ = b.call(n.bus, n.path, ifAction+".GetName", i).Store(&name)
		_ = b.call(n.bus, n.path, ifAction+".GetKeyBinding", i).Store(&key)
		out = append(out, actionInfo{Name: name, Key: key})
	}
	return out
}

var pressActions = []string{"click", "press", "activate", "jump", "toggle", "select", "open", "invoke"}

// press runs the element's primary action (click/press/activate...). Returns the action name.
func (b *a11yBus) press(n *node) (string, error) {
	acts := b.actions(n)
	if len(acts) == 0 {
		return "", fmt.Errorf("element has no accessibility action")
	}
	idx := -1
	for _, want := range pressActions {
		for i, a := range acts {
			if strings.EqualFold(a.Name, want) {
				idx = i
				break
			}
		}
		if idx >= 0 {
			break
		}
	}
	if idx < 0 {
		idx = 0
	}
	var ok bool
	if err := b.call(n.bus, n.path, ifAction+".DoAction", int32(idx)).Store(&ok); err != nil {
		return acts[idx].Name, err
	}
	if !ok {
		return acts[idx].Name, fmt.Errorf("the app refused action '%s'", acts[idx].Name)
	}
	return acts[idx].Name, nil
}

func (b *a11yBus) doNamedAction(n *node, names ...string) (string, bool) {
	for i, a := range b.actions(n) {
		for _, want := range names {
			if strings.EqualFold(a.Name, want) {
				var ok bool
				if b.call(n.bus, n.path, ifAction+".DoAction", int32(i)).Store(&ok) == nil && ok {
					return a.Name, true
				}
			}
		}
	}
	return "", false
}

func (b *a11yBus) grabFocus(n *node) bool {
	if !n.ifaces[ifComponent] {
		return false
	}
	var ok bool
	return b.call(n.bus, n.path, ifComponent+".GrabFocus").Store(&ok) == nil && ok
}

func (b *a11yBus) refreshStates(n *node) {
	var st []uint32
	if b.call(n.bus, n.path, ifAccessible+".GetState").Store(&st) == nil {
		for i := 0; i < 2 && i < len(st); i++ {
			n.states[i] = st[i]
		}
	}
}

func (b *a11yBus) refreshName(n *node) {
	if v, err := b.getProp(n.bus, n.path, ifAccessible, "Name"); err == nil {
		n.name, _ = v.Value().(string)
	}
}

func (b *a11yBus) caret(n *node) int {
	if v, err := b.getProp(n.bus, n.path, ifText, "CaretOffset"); err == nil {
		if c, ok := v.Value().(int32); ok {
			return int(c)
		}
	}
	return -1
}

func (b *a11yBus) selection(n *node) (int, int, bool) {
	var cnt int32
	if b.call(n.bus, n.path, ifText+".GetNSelections").Store(&cnt) != nil || cnt == 0 {
		return 0, 0, false
	}
	var s, e int32
	if b.call(n.bus, n.path, ifText+".GetSelection", int32(0)).Store(&s, &e) != nil {
		return 0, 0, false
	}
	if e < s {
		s, e = e, s
	}
	return int(s), int(e), e > s
}

func (b *a11yBus) setSelection(n *node, start, end int) bool {
	var ok bool
	var cnt int32
	_ = b.call(n.bus, n.path, ifText+".GetNSelections").Store(&cnt)
	if cnt > 0 {
		if b.call(n.bus, n.path, ifText+".SetSelection", int32(0), int32(start), int32(end)).Store(&ok) == nil && ok {
			return true
		}
	}
	return b.call(n.bus, n.path, ifText+".AddSelection", int32(start), int32(end)).Store(&ok) == nil && ok
}

func (b *a11yBus) textRange(n *node, start, end int) string {
	var s string
	_ = b.call(n.bus, n.path, ifText+".GetText", int32(start), int32(end)).Store(&s)
	return s
}

func (b *a11yBus) insertText(n *node, pos int, text string) bool {
	var ok bool
	return b.call(n.bus, n.path, ifEditableText+".InsertText", int32(pos), text, int32(len([]rune(text)))).Store(&ok) == nil && ok
}

func (b *a11yBus) deleteText(n *node, start, end int) bool {
	var ok bool
	return b.call(n.bus, n.path, ifEditableText+".DeleteText", int32(start), int32(end)).Store(&ok) == nil && ok
}

func (b *a11yBus) setTextContents(n *node, text string) bool {
	var ok bool
	return b.call(n.bus, n.path, ifEditableText+".SetTextContents", text).Store(&ok) == nil && ok
}

func (b *a11yBus) setCaret(n *node, off int) bool {
	var ok bool
	return b.call(n.bus, n.path, ifText+".SetCaretOffset", int32(off)).Store(&ok) == nil && ok
}

// accessibleAt returns the deepest accessible at screen point (x, y) under n.
func (b *a11yBus) accessibleAt(n *node, x, y float64) *node {
	cur := n
	for i := 0; i < 40; i++ {
		kids := b.childrenOf(cur)
		b.loadAll(kids)
		var hit *node
		// later children are drawn on top: prefer the last showing child containing the point
		for j := len(kids) - 1; j >= 0; j-- {
			k := kids[j]
			if k.hasFrame && k.frame.contains(x, y) && k.has(stShowing) {
				hit = k
				break
			}
		}
		if hit == nil {
			return cur
		}
		cur = hit
	}
	return cur
}

// ---------------------------------------------------------------------------
// Roles

// macRoles maps the macOS AX role names (accepted by find --role) to AT-SPI role names.
var macRoles = map[string][]string{
	"axbutton":           {"push button", "toggle button", "button"},
	"axtextfield":        {"text", "entry", "password text", "spin button", "search box"},
	"axtextarea":         {"text", "document text", "terminal"},
	"axstatictext":       {"label", "static", "text", "caption", "paragraph"},
	"axcheckbox":         {"check box", "toggle button", "switch", "check menu item"},
	"axradiobutton":      {"radio button", "radio menu item"},
	"axmenuitem":         {"menu item", "check menu item", "radio menu item", "tearoff menu item"},
	"axmenu":             {"menu", "popup menu"},
	"axmenubar":          {"menu bar"},
	"axmenubaritem":      {"menu"},
	"axmenubutton":       {"push button", "toggle button", "menu button"},
	"axwindow":           {"frame", "window", "dialog", "alert", "file chooser"},
	"axsheet":            {"dialog", "alert"},
	"axpopupbutton":      {"combo box", "menu button", "push button"},
	"axcombobox":         {"combo box"},
	"axslider":           {"slider"},
	"axincrementor":      {"spin button"},
	"axlink":             {"link"},
	"axtabgroup":         {"page tab list"},
	"axtab":              {"page tab"},
	"axrow":              {"table row", "list item", "tree item"},
	"axcell":             {"table cell"},
	"aximage":            {"image", "icon"},
	"axscrollarea":       {"scroll pane", "viewport"},
	"axscrollbar":        {"scroll bar"},
	"axgroup":            {"panel", "filler", "grouping", "section"},
	"axheading":          {"heading"},
	"axlist":             {"list", "list box"},
	"axtable":            {"table", "tree table"},
	"axoutline":          {"tree", "tree table"},
	"axtoolbar":          {"tool bar"},
	"axprogressindicator": {"progress bar"},
	"axdisclosuretriangle": {"toggle button", "expander"},
	"axsplitter":         {"split pane", "separator"},
	"axapplication":      {"application"},
}

func normRole(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "role_")
	s = strings.TrimPrefix(s, "atspi_role_")
	return strings.NewReplacer("_", "", " ", "", "-", "").Replace(s)
}

// roleMatches reports whether an AT-SPI role name satisfies --role (native or mac name).
func roleMatches(nodeRole, want string) bool {
	if want == "" {
		return true
	}
	w := normRole(want)
	if normRole(nodeRole) == w {
		return true
	}
	if list, ok := macRoles[w]; ok {
		nr := normRole(nodeRole)
		for _, r := range list {
			if normRole(r) == nr {
				return true
			}
		}
	}
	return false
}

func isTextInputRole(r string) bool {
	switch r {
	case "text", "entry", "password text", "document text", "terminal", "spin button", "search box", "editbar":
		return true
	}
	return false
}

func isPasswordRole(r string) bool { return r == "password text" }

var actionableRoles = map[string]bool{
	"push button": true, "toggle button": true, "button": true, "check box": true, "radio button": true,
	"menu item": true, "check menu item": true, "radio menu item": true, "menu": true, "combo box": true,
	"slider": true, "spin button": true, "page tab": true, "link": true, "list item": true, "table row": true,
	"tree item": true, "switch": true, "text": true, "entry": true, "password text": true, "menu button": true,
	"expander": true, "scroll bar": true, "search box": true, "document text": true, "terminal": true,
}

var textRoles = map[string]bool{
	"label": true, "static": true, "heading": true, "paragraph": true, "caption": true, "image": true,
	"icon": true, "table cell": true, "progress bar": true, "status bar": true, "level bar": true,
}

var windowRoles = map[string]bool{"frame": true, "window": true, "dialog": true, "alert": true, "file chooser": true}

var dialogRoles = map[string]bool{"dialog": true, "alert": true, "file chooser": true}
