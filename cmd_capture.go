package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jezek/xgb/xproto"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const screenGuidance = "Screen capture needs an X11 session (DISPLAY set). "

func requireScreen(command string) *cmdError {
	if denyPermissions() {
		return errObj(obj{"error": "Screen Recording permission not granted: screen capture is denied (AUX4_COMPUTER_DENY_PERMISSIONS)", "permission": "screenRecording"})
	}
	return requireX11(command)
}

// capture is an image of the screen or a window plus where it is on screen.
type capture struct {
	img    *image.RGBA
	origin rect // screen rectangle the image covers (1 image pixel = 1 point)
	app    *app
	win    xproto.Window
	source string
	method string
	mode   string
	reason string
	restored obj
}

func captureTarget(command string, appQ string, displayIdx int, mode string) (*capture, *cmdError) {
	if e := requireScreen(command); e != nil {
		return nil, e
	}
	x, e := X(command)
	if e != nil {
		return nil, e
	}
	if appQ == "" {
		ds := x.displays()
		if displayIdx < 0 || displayIdx >= len(ds) {
			return nil, errf("Invalid --display %d: there are %d displays (see screen)", displayIdx, len(ds))
		}
		r := ds[displayIdx].R
		img, err := x.captureRoot(r)
		if err != nil {
			return nil, errObj(obj{"error": "Screen capture failed: " + err.Error(), "phase": "capture"})
		}
		return &capture{img: img, origin: r, source: "screen", method: "getimage", mode: "background"}, nil
	}
	ap, e := findApp(appQ, false)
	if e != nil {
		return nil, e
	}
	w := ap.frontWindow(x)
	if w == 0 {
		return nil, errObj(obj{"error": ap.Name + " has no window to capture", "app": ap.Name})
	}
	c := &capture{app: ap, win: w, source: "window", mode: "background"}
	if x.minimized(w) || !x.mapped(w) {
		m, reason, err := decideMode(command, mode, false, "the window is not on screen (minimized)")
		if err != nil {
			return nil, err
		}
		_ = m
		prev := x.activeWindow()
		xproto.MapWindow(x.c, w)
		x.sync()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) && (x.minimized(w) || !x.mapped(w)) {
			time.Sleep(40 * time.Millisecond)
		}
		time.Sleep(250 * time.Millisecond)
		img, method, cerr := x.captureWindow(w)
		iconify(x, w)
		restored := obj{"cursor": true, "frontmost": true}
		if prev != 0 && x.activeWindow() != prev {
			_ = x.activate(prev)
			restored["frontmost"] = x.waitActive(prev, time.Second)
		}
		if cerr != nil {
			return nil, errObj(obj{"error": "Window capture failed: " + cerr.Error(), "phase": "capture", "app": ap.Name})
		}
		c.img, c.method, c.mode, c.reason, c.restored = img, method, "foreground", reason, restored
	} else {
		img, method, cerr := x.captureWindow(w)
		if cerr != nil {
			return nil, errObj(obj{"error": "Window capture failed: " + cerr.Error(), "phase": "capture", "app": ap.Name})
		}
		c.img, c.method = img, method
	}
	c.origin, _ = x.geometry(w)
	return c, nil
}

func (c *capture) windowObj(x *xconn) obj {
	if c.win == 0 {
		return nil
	}
	return obj{"frame": c.origin.obj(), "id": uint32(c.win), "title": x.windowTitle(c.win)}
}

func resizeTo(img *image.RGBA, maxWidth int) (*image.RGBA, float64) {
	w := img.Bounds().Dx()
	if maxWidth <= 0 || w <= maxWidth {
		return img, 1
	}
	s := float64(maxWidth) / float64(w)
	h := int(float64(img.Bounds().Dy())*s + 0.5)
	dst := image.NewRGBA(image.Rect(0, 0, maxWidth, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), xdraw.Src, nil)
	return dst, s
}

func writePNG(img image.Image, path string) (string, error) {
	if path == "" {
		f, err := os.CreateTemp("", "aux4-computer-linux-*.png")
		if err != nil {
			return "", err
		}
		path = f.Name()
		f.Close()
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return path, png.Encode(f, img)
}

func cmdScreenshot(a map[string]string) (obj, *cmdError) {
	maxWidth, e := parsePositive("maxWidth", a["maxWidth"], 1568)
	if e != nil {
		return nil, e
	}
	displayIdx, e := parseIntArg("display", a["display"], 0)
	if e != nil {
		return nil, e
	}
	annotate, e := parseBool("annotate", a["annotate"], false)
	if e != nil {
		return nil, e
	}
	roles := a["roles"]
	if roles == "" {
		roles = "actionable"
	}
	if roles != "actionable" && roles != "all" {
		return nil, errf("Invalid --roles '%s': use actionable or all", roles)
	}
	limit, e := parsePositive("limit", a["limit"], 150)
	if e != nil {
		return nil, e
	}
	c, e := captureTarget("screenshot", a["app"], displayIdx, "auto")
	if e != nil {
		return nil, e
	}
	x, _ := X("screenshot")
	img, scale := resizeTo(c.img, maxWidth)
	o := obj{"width": img.Bounds().Dx(), "height": img.Bounds().Dy(), "scale": round2(scale)}
	if c.app != nil {
		o["app"] = c.app.Name
		o["window"] = c.windowObj(x)
	} else {
		o["display"] = displayIdx
		o["frame"] = c.origin.obj()
	}
	if c.mode == "foreground" {
		setFallback(o, "foreground", c.reason, c.restored)
	}
	if annotate {
		ap := c.app
		if ap != nil && ap.A == nil {
			if withA, err := findApp(strconv.Itoa(ap.Pid), true); err == nil {
				ap = withA
			}
		}
		if ap == nil {
			ap, e = frontmostApp(true)
			if e != nil {
				return nil, e
			}
		}
		b, ai, e := ap.needA11y()
		if e != nil {
			return nil, e
		}
		marks, truncated := annotateImage(b, ai, img, c.origin, scale, roles, limit, c.app != nil)
		o["annotated"] = true
		o["marks"] = marks
		o["roles"] = roles
		if c.app == nil {
			o["app"] = ap.Name
		}
		if truncated {
			o["truncated"] = true
			o["note"] = fmt.Sprintf("more elements than --limit %d: only the first %d are marked", limit, limit)
		}
	}
	path, err := writePNG(img, a["output"])
	if err != nil {
		return nil, errf("Cannot write %s: %v", a["output"], err)
	}
	o["path"] = path
	return o, nil
}

// annotateImage draws numbered boxes on the visible elements and returns their marks.
func annotateImage(b *a11yBus, ai *appInfo, img *image.RGBA, origin rect, scale float64, roles string, limit int, withWindow bool) ([]any, bool) {
	nodes, _ := snapshot(b, ai.root, 30, 6000, false)
	var picked []*node
	for _, n := range nodes {
		if !n.hasFrame || menuFamily[n.role] && !n.has(stShowing) {
			continue
		}
		ok := actionableRoles[n.role] || roles == "all" && (textRoles[n.role] || n.role == "label")
		if !ok || strings.Count(n.ref, ".") == 0 {
			continue
		}
		vis := intersect(n.frame, origin)
		if vis.empty() {
			continue
		}
		picked = append(picked, n)
	}
	sort.SliceStable(picked, func(i, j int) bool {
		a, c := picked[i].frame, picked[j].frame
		if int(a.Y/8) != int(c.Y/8) {
			return a.Y < c.Y
		}
		return a.X < c.X
	})
	truncated := len(picked) > limit
	if truncated {
		picked = picked[:limit]
	}
	marks := []any{}
	red := color.RGBA{230, 30, 60, 255}
	var tags []image.Rectangle
	for i, n := range picked {
		vis := intersect(n.frame, origin)
		cx, cy := vis.center()
		ib := image.Rect(int((vis.X-origin.X)*scale), int((vis.Y-origin.Y)*scale), int((vis.X-origin.X+vis.W)*scale), int((vis.Y-origin.Y+vis.H)*scale))
		drawRect(img, ib, red)
		label := fmt.Sprint(i + 1)
		tag := placeTag(ib, len(label), tags, img.Bounds())
		tags = append(tags, tag)
		fillRect(img, tag, red)
		drawText(img, tag.Min.X+2, tag.Max.Y-3, label, color.White)
		m := obj{"n": i + 1, "ref": n.ref, "role": n.role, "label": n.label(), "frame": n.frame.obj(), "center": point(cx, cy),
			"image": obj{"x": ib.Min.X, "y": ib.Min.Y, "width": ib.Dx(), "height": ib.Dy()}}
		if isTextInputRole(n.role) && !isPasswordRole(n.role) {
			m["value"] = cut(b.valueOf(n), 80)
		}
		if withWindow {
			m["windowCenter"] = point(cx-origin.X, cy-origin.Y)
		}
		marks = append(marks, m)
	}
	return marks, truncated
}

func intersect(a, b rect) rect {
	x1, y1 := maxf(a.X, b.X), maxf(a.Y, b.Y)
	x2, y2 := minf(a.X+a.W, b.X+b.W), minf(a.Y+a.H, b.Y+b.H)
	if x2 <= x1 || y2 <= y1 {
		return rect{}
	}
	return rect{x1, y1, x2 - x1, y2 - y1}
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func drawRect(img *image.RGBA, r image.Rectangle, c color.Color) {
	r = r.Intersect(img.Bounds())
	for t := 0; t < 2; t++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.Set(x, r.Min.Y+t, c)
			img.Set(x, r.Max.Y-1-t, c)
		}
		for y := r.Min.Y; y < r.Max.Y; y++ {
			img.Set(r.Min.X+t, y, c)
			img.Set(r.Max.X-1-t, y, c)
		}
	}
}

func fillRect(img *image.RGBA, r image.Rectangle, c color.Color) {
	xdraw.Draw(img, r.Intersect(img.Bounds()), &image.Uniform{c}, image.Point{}, xdraw.Src)
}

func drawText(img *image.RGBA, x, y int, s string, c color.Color) {
	d := &font.Drawer{Dst: img, Src: image.NewUniform(c), Face: basicfont.Face7x13, Dot: fixed.P(x, y)}
	d.DrawString(s)
}

// placeTag puts a number tag at the box's top-left corner, or another corner when that one
// would cover an earlier tag.
func placeTag(box image.Rectangle, digits int, used []image.Rectangle, bounds image.Rectangle) image.Rectangle {
	w, h := digits*7+4, 14
	cands := []image.Point{
		{box.Min.X, box.Min.Y - h}, {box.Min.X, box.Min.Y}, {box.Max.X - w, box.Min.Y},
		{box.Min.X, box.Max.Y - h}, {box.Max.X - w, box.Max.Y - h}, {box.Max.X, box.Min.Y},
	}
	for _, p := range cands {
		r := image.Rect(p.X, p.Y, p.X+w, p.Y+h)
		if !r.In(bounds) {
			continue
		}
		clash := false
		for _, u := range used {
			if r.Overlaps(u) {
				clash = true
				break
			}
		}
		if !clash {
			return r
		}
	}
	p := cands[1]
	return image.Rect(p.X, p.Y, p.X+w, p.Y+h).Intersect(bounds)
}
