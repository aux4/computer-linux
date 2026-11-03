package main

import (
	"encoding/binary"
	"fmt"
	"image"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/composite"
	"github.com/jezek/xgb/randr"
	"github.com/jezek/xgb/xinerama"
	"github.com/jezek/xgb/xproto"
	"github.com/jezek/xgb/xtest"
)

type xconn struct {
	c         *xgb.Conn
	root      xproto.Window
	screen    *xproto.ScreenInfo
	atoms     map[string]xproto.Atom
	hasXTest  bool
	hasComp   bool
	mu        sync.Mutex
	keymap    *xproto.GetKeyboardMappingReply
	minKey    xproto.Keycode
	maxKey    xproto.Keycode
	modKeys   map[string]xproto.Keycode
}

var xc *xconn
var xcErr error
var xcOnce sync.Once

func init() {
	// xgb logs to stderr on unexpected protocol events; keep stdout/stderr JSON-only.
	xgb.Logger.SetOutput(discard{})
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

// X returns the shared X connection, or an error suitable for a command.
func X(command string) (*xconn, *cmdError) {
	if e := requireX11(command); e != nil {
		return nil, e
	}
	xcOnce.Do(func() {
		c, err := xgb.NewConn()
		if err != nil {
			xcErr = err
			return
		}
		setup := xproto.Setup(c)
		scr := setup.DefaultScreen(c)
		x := &xconn{c: c, root: scr.Root, screen: scr, atoms: map[string]xproto.Atom{}, minKey: setup.MinKeycode, maxKey: setup.MaxKeycode}
		x.hasXTest = xtest.Init(c) == nil
		x.hasComp = composite.Init(c) == nil
		xc = x
	})
	if xcErr != nil {
		return nil, errObj(obj{"error": fmt.Sprintf("Cannot connect to the X server at DISPLAY=%s: %v", os.Getenv("DISPLAY"), xcErr), "session": sessionType()})
	}
	return xc, nil
}

func (x *xconn) atom(name string) xproto.Atom {
	x.mu.Lock()
	defer x.mu.Unlock()
	if a, ok := x.atoms[name]; ok {
		return a
	}
	r, err := xproto.InternAtom(x.c, false, uint16(len(name)), name).Reply()
	if err != nil {
		return 0
	}
	x.atoms[name] = r.Atom
	return r.Atom
}

func (x *xconn) atomName(a xproto.Atom) string {
	r, err := xproto.GetAtomName(x.c, a).Reply()
	if err != nil {
		return ""
	}
	return r.Name
}

func (x *xconn) prop(w xproto.Window, name string) *xproto.GetPropertyReply {
	r, err := xproto.GetProperty(x.c, false, w, x.atom(name), xproto.GetPropertyTypeAny, 0, 1<<20).Reply()
	if err != nil || r == nil || r.Format == 0 {
		return nil
	}
	return r
}

func (x *xconn) propU32s(w xproto.Window, name string) []uint32 {
	r := x.prop(w, name)
	if r == nil || r.Format != 32 {
		return nil
	}
	out := make([]uint32, len(r.Value)/4)
	for i := range out {
		out[i] = binary.LittleEndian.Uint32(r.Value[i*4:])
	}
	return out
}

func (x *xconn) propString(w xproto.Window, name string) string {
	r := x.prop(w, name)
	if r == nil || r.Format != 8 {
		return ""
	}
	return strings.TrimRight(string(r.Value), "\x00")
}

func (x *xconn) windowTitle(w xproto.Window) string {
	if s := x.propString(w, "_NET_WM_NAME"); s != "" && utf8.ValidString(s) {
		return s
	}
	return x.propString(w, "WM_NAME")
}

// wmClass returns (instance, class).
func (x *xconn) wmClass(w xproto.Window) (string, string) {
	r := x.prop(w, "WM_CLASS")
	if r == nil {
		return "", ""
	}
	parts := strings.Split(string(r.Value), "\x00")
	inst, cls := "", ""
	if len(parts) > 0 {
		inst = parts[0]
	}
	if len(parts) > 1 {
		cls = parts[1]
	}
	return inst, cls
}

func (x *xconn) windowPid(w xproto.Window) int {
	v := x.propU32s(w, "_NET_WM_PID")
	if len(v) == 0 {
		return 0
	}
	return int(v[0])
}

func (x *xconn) clientList() []xproto.Window {
	v := x.propU32s(x.root, "_NET_CLIENT_LIST_STACKING")
	if len(v) == 0 {
		v = x.propU32s(x.root, "_NET_CLIENT_LIST")
	}
	out := make([]xproto.Window, 0, len(v))
	for _, id := range v {
		out = append(out, xproto.Window(id))
	}
	return out
}

func (x *xconn) activeWindow() xproto.Window {
	v := x.propU32s(x.root, "_NET_ACTIVE_WINDOW")
	if len(v) == 0 {
		return 0
	}
	return xproto.Window(v[0])
}

func (x *xconn) wmStates(w xproto.Window) []string {
	var out []string
	for _, a := range x.propU32s(w, "_NET_WM_STATE") {
		out = append(out, x.atomName(xproto.Atom(a)))
	}
	return out
}

func (x *xconn) hasState(w xproto.Window, state string) bool {
	for _, s := range x.wmStates(w) {
		if s == state {
			return true
		}
	}
	return false
}

func (x *xconn) windowTypes(w xproto.Window) []string {
	var out []string
	for _, a := range x.propU32s(w, "_NET_WM_WINDOW_TYPE") {
		out = append(out, x.atomName(xproto.Atom(a)))
	}
	return out
}

// minimized: ICCCM WM_STATE IconicState or _NET_WM_STATE_HIDDEN.
func (x *xconn) minimized(w xproto.Window) bool {
	if v := x.propU32s(w, "WM_STATE"); len(v) > 0 && v[0] == 3 {
		return true
	}
	return x.hasState(w, "_NET_WM_STATE_HIDDEN")
}

func (x *xconn) mapped(w xproto.Window) bool {
	r, err := xproto.GetWindowAttributes(x.c, w).Reply()
	return err == nil && r.MapState == xproto.MapStateViewable
}

// geometry returns the window's content rectangle in root coordinates.
func (x *xconn) geometry(w xproto.Window) (rect, bool) {
	g, err := xproto.GetGeometry(x.c, xproto.Drawable(w)).Reply()
	if err != nil {
		return rect{}, false
	}
	t, err := xproto.TranslateCoordinates(x.c, w, x.root, 0, 0).Reply()
	if err != nil {
		return rect{}, false
	}
	return rect{X: float64(t.DstX), Y: float64(t.DstY), W: float64(g.Width), H: float64(g.Height)}, true
}

// frameExtents returns left, right, top, bottom decorations from _NET_FRAME_EXTENTS.
func (x *xconn) frameExtents(w xproto.Window) [4]int {
	var e [4]int
	v := x.propU32s(w, "_NET_FRAME_EXTENTS")
	for i := 0; i < 4 && i < len(v); i++ {
		e[i] = int(v[i])
	}
	return e
}

// topLevel returns the ancestor of w that is a direct child of root (the WM frame, or w itself).
func (x *xconn) topLevel(w xproto.Window) xproto.Window {
	cur := w
	for i := 0; i < 32; i++ {
		t, err := xproto.QueryTree(x.c, cur).Reply()
		if err != nil || t.Parent == 0 || t.Parent == x.root {
			return cur
		}
		cur = t.Parent
	}
	return cur
}

func (x *xconn) sendClientMessage(w xproto.Window, msgType string, data ...uint32) error {
	d := make([]uint32, 5)
	copy(d, data)
	ev := xproto.ClientMessageEvent{Format: 32, Window: w, Type: x.atom(msgType), Data: xproto.ClientMessageDataUnionData32New(d)}
	mask := uint32(xproto.EventMaskSubstructureRedirect | xproto.EventMaskSubstructureNotify)
	return xproto.SendEventChecked(x.c, false, x.root, mask, string(ev.Bytes())).Check()
}

func (x *xconn) activate(w xproto.Window) error {
	// source indication 2 = pager/user request: window managers honour it without focus-stealing prevention.
	err := x.sendClientMessage(w, "_NET_ACTIVE_WINDOW", 2, uint32(xproto.TimeCurrentTime), uint32(x.activeWindow()))
	xproto.SetInputFocus(x.c, xproto.InputFocusPointerRoot, w, xproto.TimeCurrentTime)
	return err
}

func (x *xconn) waitActive(w xproto.Window, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if x.activeWindow() == w {
			return true
		}
		time.Sleep(25 * time.Millisecond)
	}
	return x.activeWindow() == w
}

func (x *xconn) pointer() (int, int) {
	r, err := xproto.QueryPointer(x.c, x.root).Reply()
	if err != nil {
		return 0, 0
	}
	return int(r.RootX), int(r.RootY)
}

// ---------------------------------------------------------------------------
// Displays

type display struct {
	Index int
	ID    int
	Main  bool
	R     rect
	Name  string
}

func (x *xconn) displays() []display {
	var out []display
	if randr.Init(x.c) == nil {
		if r, err := randr.GetMonitors(x.c, x.root, true).Reply(); err == nil && len(r.Monitors) > 0 {
			for i, m := range r.Monitors {
				out = append(out, display{Index: i, ID: int(m.Name), Main: m.Primary, R: rect{float64(m.X), float64(m.Y), float64(m.Width), float64(m.Height)}, Name: x.atomName(m.Name)})
			}
		}
	}
	if len(out) == 0 && xinerama.Init(x.c) == nil {
		if r, err := xinerama.QueryScreens(x.c).Reply(); err == nil {
			for i, s := range r.ScreenInfo {
				out = append(out, display{Index: i, ID: i, R: rect{float64(s.XOrg), float64(s.YOrg), float64(s.Width), float64(s.Height)}})
			}
		}
	}
	if len(out) == 0 {
		out = []display{{Index: 0, ID: 0, R: rect{0, 0, float64(x.screen.WidthInPixels), float64(x.screen.HeightInPixels)}}}
	}
	hasMain := false
	for _, d := range out {
		hasMain = hasMain || d.Main
	}
	if !hasMain {
		out[0].Main = true
	}
	return out
}

// scale is the desktop scale factor: Xft.dpi / 96 from the X resource database (1 when unset).
func (x *xconn) scale() float64 {
	res := x.propString(x.root, "RESOURCE_MANAGER")
	for _, line := range strings.Split(res, "\n") {
		if strings.HasPrefix(line, "Xft.dpi:") {
			if dpi, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(line, "Xft.dpi:")), 64); err == nil && dpi > 0 {
				return round2(dpi / 96)
			}
		}
	}
	return 1
}

// ---------------------------------------------------------------------------
// Screen capture

func (x *xconn) getImage(d xproto.Drawable, r rect) (*image.RGBA, error) {
	w, h := int(r.W), int(r.H)
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("empty capture area")
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Fetch in horizontal bands so each reply stays well under the maximum request size.
	band := 256
	for y0 := 0; y0 < h; y0 += band {
		bh := band
		if y0+bh > h {
			bh = h - y0
		}
		rep, err := xproto.GetImage(x.c, xproto.ImageFormatZPixmap, d, int16(int(r.X)), int16(int(r.Y)+y0), uint16(w), uint16(bh), 0xffffffff).Reply()
		if err != nil {
			return nil, err
		}
		data := rep.Data
		bpp := len(data) / (w * bh)
		if bpp < 3 {
			return nil, fmt.Errorf("unsupported visual depth %d", rep.Depth)
		}
		stride := len(data) / bh
		for yy := 0; yy < bh; yy++ {
			for xx := 0; xx < w; xx++ {
				i := yy*stride + xx*bpp
				o := img.PixOffset(xx, y0+yy)
				img.Pix[o+0] = data[i+2]
				img.Pix[o+1] = data[i+1]
				img.Pix[o+2] = data[i+0]
				img.Pix[o+3] = 0xff
			}
		}
	}
	return img, nil
}

func (x *xconn) captureRoot(r rect) (*image.RGBA, error) {
	return x.getImage(xproto.Drawable(x.root), r)
}

// obscured reports whether another mapped top-level window above w overlaps it.
func (x *xconn) obscured(w xproto.Window) bool {
	wr, ok := x.geometry(w)
	if !ok {
		return false
	}
	top := x.topLevel(w)
	t, err := xproto.QueryTree(x.c, x.root).Reply()
	if err != nil {
		return false
	}
	above := false
	for _, ch := range t.Children { // bottom to top
		if ch == top {
			above = true
			continue
		}
		if !above || !x.mapped(ch) {
			continue
		}
		cr, ok := x.geometry(ch)
		if !ok {
			continue
		}
		if cr.X < wr.X+wr.W && cr.X+cr.W > wr.X && cr.Y < wr.Y+wr.H && cr.Y+cr.H > wr.Y {
			return true
		}
	}
	return false
}

// captureWindow captures the content of client window w, even when covered, through the
// Composite extension (redirecting the top-level window offscreen while this process runs).
// Returns the image, the method used and an error.
func (x *xconn) captureWindow(w xproto.Window) (*image.RGBA, string, error) {
	wr, ok := x.geometry(w)
	if !ok {
		return nil, "", fmt.Errorf("window is gone")
	}
	if !x.mapped(w) {
		return nil, "", fmt.Errorf("window is not on screen (minimized or unmapped)")
	}
	if x.obscured(w) && x.hasComp {
		top := x.topLevel(w)
		if composite.RedirectWindowChecked(x.c, top, composite.RedirectAutomatic).Check() == nil {
			// The app repaints the parts that were covered into the offscreen pixmap.
			time.Sleep(250 * time.Millisecond)
			pix, err := xproto.NewPixmapId(x.c)
			if err == nil && composite.NameWindowPixmapChecked(x.c, top, pix).Check() == nil {
				tr, terr := xproto.TranslateCoordinates(x.c, w, top, 0, 0).Reply()
				if terr == nil {
					img, err := x.getImage(xproto.Drawable(pix), rect{float64(tr.DstX), float64(tr.DstY), wr.W, wr.H})
					xproto.FreePixmap(x.c, pix)
					composite.UnredirectWindow(x.c, top, composite.RedirectAutomatic)
					if err == nil {
						return img, "composite", nil
					}
				}
			}
			composite.UnredirectWindow(x.c, top, composite.RedirectAutomatic)
		}
	}
	img, err := x.getImage(xproto.Drawable(w), rect{0, 0, wr.W, wr.H})
	if err != nil {
		img, err = x.captureRoot(wr)
	}
	return img, "getimage", err
}

// ---------------------------------------------------------------------------
// Input through XTest (real input: moves the pointer / goes to the focused window)

func (x *xconn) requireXTest(command string) *cmdError {
	if !x.hasXTest {
		return unsupported(command, "the X server has no XTEST extension, which is needed to send input", "Enable XTEST in the X server (Xorg and Xvfb have it by default)")
	}
	return nil
}

func (x *xconn) motion(px, py int) {
	xtest.FakeInput(x.c, xproto.MotionNotify, 0, 0, x.root, int16(px), int16(py), 0)
}

func (x *xconn) button(b byte, down bool) {
	t := byte(xproto.ButtonRelease)
	if down {
		t = xproto.ButtonPress
	}
	xtest.FakeInput(x.c, t, b, 0, x.root, 0, 0, 0)
}

func (x *xconn) key(code xproto.Keycode, down bool) {
	t := byte(xproto.KeyRelease)
	if down {
		t = xproto.KeyPress
	}
	xtest.FakeInput(x.c, t, byte(code), 0, x.root, 0, 0, 0)
}

func (x *xconn) sync() {
	xproto.GetInputFocus(x.c).Reply()
}

func (x *xconn) loadKeymap() {
	if x.keymap != nil {
		return
	}
	count := byte(int(x.maxKey) - int(x.minKey) + 1)
	r, err := xproto.GetKeyboardMapping(x.c, x.minKey, count).Reply()
	if err == nil {
		x.keymap = r
	}
}

// keycodeFor finds a keycode + shift level producing keysym ks.
func (x *xconn) keycodeFor(ks xproto.Keysym) (xproto.Keycode, int, bool) {
	x.loadKeymap()
	if x.keymap == nil {
		return 0, 0, false
	}
	per := int(x.keymap.KeysymsPerKeycode)
	n := len(x.keymap.Keysyms) / per
	for level := 0; level < 2 && level < per; level++ {
		for i := 0; i < n; i++ {
			if x.keymap.Keysyms[i*per+level] == ks {
				return xproto.Keycode(int(x.minKey) + i), level, true
			}
		}
	}
	return 0, 0, false
}

// spareKeycode returns a keycode with no keysyms, for temporarily mapping a character.
func (x *xconn) spareKeycode() (xproto.Keycode, bool) {
	x.loadKeymap()
	if x.keymap == nil {
		return 0, false
	}
	per := int(x.keymap.KeysymsPerKeycode)
	n := len(x.keymap.Keysyms) / per
	for i := n - 1; i >= 0; i-- {
		empty := true
		for l := 0; l < per; l++ {
			if x.keymap.Keysyms[i*per+l] != 0 {
				empty = false
				break
			}
		}
		if empty {
			return xproto.Keycode(int(x.minKey) + i), true
		}
	}
	return 0, false
}

func (x *xconn) remapKey(code xproto.Keycode, ks xproto.Keysym) {
	per := int(x.keymap.KeysymsPerKeycode)
	syms := make([]xproto.Keysym, per)
	syms[0], syms[1] = ks, ks
	xproto.ChangeKeyboardMapping(x.c, 1, code, byte(per), syms)
	i := int(code) - int(x.minKey)
	copy(x.keymap.Keysyms[i*per:(i+1)*per], syms)
	x.sync()
}

func (x *xconn) unmapKey(code xproto.Keycode) {
	per := int(x.keymap.KeysymsPerKeycode)
	syms := make([]xproto.Keysym, per)
	xproto.ChangeKeyboardMapping(x.c, 1, code, byte(per), syms)
	i := int(code) - int(x.minKey)
	copy(x.keymap.Keysyms[i*per:(i+1)*per], syms)
	x.sync()
}

const (
	ksShiftL   = 0xffe1
	ksControlL = 0xffe3
	ksAltL     = 0xffe9
	ksSuperL   = 0xffeb
)

func runeKeysym(r rune) xproto.Keysym {
	switch r {
	case '\n', '\r':
		return 0xff0d
	case '\t':
		return 0xff09
	case '\b':
		return 0xff08
	}
	if r >= 0x20 && r <= 0x7e || r >= 0xa0 && r <= 0xff {
		return xproto.Keysym(r)
	}
	return xproto.Keysym(0x01000000 + int(r))
}

// pressKeysym presses (and releases) one keysym, holding the given modifier keysyms. A keysym
// that isn't on the keyboard is mapped on a spare keycode for the duration of the press.
func (x *xconn) pressKeysym(ks xproto.Keysym, mods []xproto.Keysym) error {
	code, level, ok := x.keycodeFor(ks)
	remapped := false
	if !ok {
		spare, ok2 := x.spareKeycode()
		if !ok2 {
			return fmt.Errorf("no free keycode to type keysym 0x%x", ks)
		}
		x.remapKey(spare, ks)
		time.Sleep(30 * time.Millisecond)
		code, level, remapped = spare, 0, true
	}
	var held []xproto.Keycode
	for _, m := range mods {
		if mc, _, ok := x.keycodeFor(m); ok {
			x.key(mc, true)
			held = append(held, mc)
		}
	}
	shift := false
	if level == 1 && !containsKeysym(mods, ksShiftL) {
		if sc, _, ok := x.keycodeFor(ksShiftL); ok {
			x.key(sc, true)
			shift = true
			held = append(held, sc)
		}
	}
	_ = shift
	x.key(code, true)
	x.key(code, false)
	for i := len(held) - 1; i >= 0; i-- {
		x.key(held[i], false)
	}
	x.sync()
	if remapped {
		time.Sleep(30 * time.Millisecond)
		x.unmapKey(code)
	}
	return nil
}

func containsKeysym(list []xproto.Keysym, k xproto.Keysym) bool {
	for _, v := range list {
		if v == k {
			return true
		}
	}
	return false
}

func (x *xconn) typeText(text string) error {
	for _, r := range text {
		if err := x.pressKeysym(runeKeysym(r), nil); err != nil {
			return err
		}
		time.Sleep(8 * time.Millisecond)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Foreground helper: bring a window forward, act, restore pointer + previously active window.

type foregroundSession struct {
	x         *xconn
	prevWin   xproto.Window
	px, py    int
	activated bool
	restore   bool
}

func (x *xconn) beginForeground(target xproto.Window, restore bool) *foregroundSession {
	fs := &foregroundSession{x: x, prevWin: x.activeWindow(), restore: restore}
	fs.px, fs.py = x.pointer()
	if target != 0 && x.activeWindow() != target {
		if x.minimized(target) {
			xproto.MapWindow(x.c, target)
		}
		_ = x.activate(target)
		x.waitActive(target, 1500*time.Millisecond)
		time.Sleep(80 * time.Millisecond)
		fs.activated = true
	}
	return fs
}

// end restores the cursor and the previously active window and reports what was restored.
func (fs *foregroundSession) end() obj {
	if !fs.restore {
		return nil
	}
	x := fs.x
	x.sync()
	time.Sleep(60 * time.Millisecond)
	cx, cy := x.pointer()
	restored := obj{"cursor": true, "frontmost": true}
	if cx != fs.px || cy != fs.py {
		x.motion(fs.px, fs.py)
		x.sync()
	}
	if fs.activated && fs.prevWin != 0 && x.activeWindow() != fs.prevWin {
		_ = x.activate(fs.prevWin)
		restored["frontmost"] = x.waitActive(fs.prevWin, 1000*time.Millisecond)
	}
	return restored
}
