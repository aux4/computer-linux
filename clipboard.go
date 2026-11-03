package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

func (x *xconn) helperWindow() (xproto.Window, error) {
	w, err := xproto.NewWindowId(x.c)
	if err != nil {
		return 0, err
	}
	err = xproto.CreateWindowChecked(x.c, x.screen.RootDepth, w, x.root, -10, -10, 1, 1, 0,
		xproto.WindowClassInputOutput, x.screen.RootVisual, xproto.CwEventMask, []uint32{xproto.EventMaskPropertyChange}).Check()
	return w, err
}

// waitEvent waits for an event matching f, up to timeout.
func (x *xconn) waitEvent(timeout time.Duration, f func(xgb.Event) bool) (xgb.Event, bool) {
	ch := make(chan xgb.Event, 16)
	stop := make(chan struct{})
	go func() {
		for {
			ev, err := x.c.WaitForEvent()
			if ev == nil && err == nil {
				return
			}
			select {
			case <-stop:
				return
			case ch <- ev:
			}
		}
	}()
	defer close(stop)
	deadline := time.After(timeout)
	for {
		select {
		case ev := <-ch:
			if ev != nil && f(ev) {
				return ev, true
			}
		case <-deadline:
			return nil, false
		}
	}
}

func (x *xconn) readClipboard() (string, error) {
	owner, err := xproto.GetSelectionOwner(x.c, x.atom("CLIPBOARD")).Reply()
	if err != nil {
		return "", err
	}
	if owner.Owner == 0 {
		return "", nil
	}
	w, err := x.helperWindow()
	if err != nil {
		return "", err
	}
	defer xproto.DestroyWindow(x.c, w)
	prop := x.atom("AUX4_CLIPBOARD")
	for _, target := range []string{"UTF8_STRING", "STRING"} {
		xproto.ConvertSelection(x.c, w, x.atom("CLIPBOARD"), x.atom(target), prop, xproto.TimeCurrentTime)
		ev, ok := x.waitEvent(2*time.Second, func(e xgb.Event) bool {
			_, is := e.(xproto.SelectionNotifyEvent)
			return is
		})
		if !ok {
			return "", fmt.Errorf("the clipboard owner did not answer")
		}
		if ev.(xproto.SelectionNotifyEvent).Property == 0 {
			continue
		}
		r, err := xproto.GetProperty(x.c, true, w, prop, xproto.GetPropertyTypeAny, 0, 1<<24).Reply()
		if err != nil {
			return "", err
		}
		if r.Type == x.atom("INCR") {
			var buf []byte
			for {
				_, ok := x.waitEvent(2*time.Second, func(e xgb.Event) bool {
					p, is := e.(xproto.PropertyNotifyEvent)
					return is && p.Atom == prop && p.State == xproto.PropertyNewValue
				})
				if !ok {
					return string(buf), fmt.Errorf("incomplete incremental transfer")
				}
				chunk, err := xproto.GetProperty(x.c, true, w, prop, xproto.GetPropertyTypeAny, 0, 1<<24).Reply()
				if err != nil || len(chunk.Value) == 0 {
					break
				}
				buf = append(buf, chunk.Value...)
			}
			return string(buf), nil
		}
		return string(r.Value), nil
	}
	return "", nil
}

// writeClipboard hands the text to a background helper process that owns the CLIPBOARD
// selection until another app takes it (X11 clipboards live in the owning process).
func writeClipboard(text string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "__clipboard-serve")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	devnull, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	cmd.Stderr = devnull
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		io.WriteString(stdin, text)
		stdin.Close()
	}()
	ready := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(stdout).ReadString('\n')
		ready <- strings.TrimSpace(line)
	}()
	select {
	case l := <-ready:
		_ = cmd.Process.Release()
		if l != "ok" {
			return fmt.Errorf("the clipboard helper failed: %s", l)
		}
		return nil
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		return fmt.Errorf("the clipboard helper did not start")
	}
}

// clipboardServe is the helper: owns CLIPBOARD and answers requests until replaced.
func clipboardServe() {
	data, _ := io.ReadAll(os.Stdin)
	c, err := xgb.NewConn()
	if err != nil {
		fmt.Println("error: " + err.Error())
		return
	}
	setup := xproto.Setup(c)
	scr := setup.DefaultScreen(c)
	x := &xconn{c: c, root: scr.Root, screen: scr, atoms: map[string]xproto.Atom{}}
	w, err := x.helperWindow()
	if err != nil {
		fmt.Println("error: " + err.Error())
		return
	}
	clip := x.atom("CLIPBOARD")
	xproto.SetSelectionOwner(c, w, clip, xproto.TimeCurrentTime)
	if o, err := xproto.GetSelectionOwner(c, clip).Reply(); err != nil || o.Owner != w {
		fmt.Println("error: could not take the clipboard")
		return
	}
	fmt.Println("ok")
	os.Stdout.Close()
	targets := []string{"TARGETS", "UTF8_STRING", "STRING", "TEXT", "text/plain;charset=utf-8", "text/plain"}
	textTargets := map[xproto.Atom]bool{}
	for _, t := range targets[1:] {
		textTargets[x.atom(t)] = true
	}
	for {
		ev, err := c.WaitForEvent()
		if ev == nil && err == nil {
			return
		}
		switch e := ev.(type) {
		case xproto.SelectionClearEvent:
			return
		case xproto.SelectionRequestEvent:
			prop := e.Property
			if prop == 0 {
				prop = e.Target
			}
			switch {
			case e.Target == x.atom("TARGETS"):
				buf := make([]byte, 4*len(targets))
				for i, t := range targets {
					binary.LittleEndian.PutUint32(buf[i*4:], uint32(x.atom(t)))
				}
				xproto.ChangeProperty(c, xproto.PropModeReplace, e.Requestor, prop, xproto.AtomAtom, 32, uint32(len(targets)), buf)
			case textTargets[e.Target]:
				typ := e.Target
				if e.Target == x.atom("TEXT") {
					typ = x.atom("UTF8_STRING")
				}
				xproto.ChangeProperty(c, xproto.PropModeReplace, e.Requestor, prop, typ, 8, uint32(len(data)), data)
			default:
				prop = 0
			}
			n := xproto.SelectionNotifyEvent{Time: e.Time, Requestor: e.Requestor, Selection: e.Selection, Target: e.Target, Property: prop}
			xproto.SendEvent(c, false, e.Requestor, 0, string(n.Bytes()))
			xproto.GetInputFocus(c).Reply()
		}
	}
}

func clipboardText() (string, *cmdError) {
	x, e := X("clipboard read")
	if e != nil {
		return "", e
	}
	s, err := x.readClipboard()
	if err != nil {
		return "", errf("Cannot read the clipboard: %v", err)
	}
	return s, nil
}

func cmdClipboardRead(a map[string]string) (obj, *cmdError) {
	s, e := clipboardText()
	if e != nil {
		return nil, e
	}
	return obj{"text": s}, nil
}

func cmdClipboardCopy(a map[string]string) (obj, *cmdError) {
	mode, e := parseMode(a["mode"])
	if e != nil {
		return nil, e
	}
	if a["text"] != "" {
		if dryRun() {
			return obj{"dryRun": true, "text": a["text"]}, nil
		}
		if e := requireX11("clipboard copy"); e != nil {
			return nil, e
		}
		if err := writeClipboard(a["text"]); err != nil {
			return nil, errf("Cannot write the clipboard: %v", err)
		}
		return obj{"text": a["text"]}, nil
	}
	if dryRun() {
		m, r, err := decideMode("clipboard copy", mode, true, "")
		if err != nil {
			return nil, err
		}
		o := obj{}
		if a["app"] != "" {
			o["app"] = a["app"]
		}
		return withDryRun(o, m, r), nil
	}
	if e := requireX11("clipboard copy"); e != nil {
		return nil, e
	}
	ap, e := targetApp(a["app"], true)
	if e != nil {
		return nil, e
	}
	o := obj{"app": ap.Name}
	m, reason, err := decideMode("clipboard copy", mode, true, "")
	if err != nil {
		return nil, err
	}
	if m == "background" {
		why := "the app doesn't expose its selection"
		if b, _, ae := ap.needA11y(); ae == nil {
			if n, te := editTarget(b, ap, ""); te == nil && n.ifaces[ifText] {
				s, en, ok := b.selection(n)
				text := ""
				if ok {
					text = b.textRange(n, s, en)
				}
				if text != "" {
					if err := writeClipboard(text); err != nil {
						return nil, errf("Cannot write the clipboard: %v", err)
					}
				}
				o["text"], o["method"], o["mode"] = text, "accessibility", "background"
				return o, nil
			} else if te != nil {
				why = te.Error()
			}
		} else {
			why = ae.Error()
		}
		if mode == "background" {
			return nil, errObj(obj{"error": "Background mode not possible for clipboard copy: " + why, "mode": "background", "app": ap.Name})
		}
		reason = why
	}
	x, xe := X("clipboard copy")
	if xe != nil {
		return nil, xe
	}
	if e := x.requireXTest("clipboard copy"); e != nil {
		return nil, e
	}
	fs := x.beginForeground(ap.frontWindow(x), mode == "auto")
	_ = x.pressKeysym('c', []xproto.Keysym{ksControlL})
	time.Sleep(150 * time.Millisecond)
	restored := fs.end()
	s, _ := x.readClipboard()
	o["text"], o["method"] = s, "xtest"
	setFallback(o, "foreground", reason, restored)
	return o, nil
}

func cmdClipboardPaste(a map[string]string) (obj, *cmdError) {
	mode, e := parseMode(a["mode"])
	if e != nil {
		return nil, e
	}
	if dryRun() {
		m, r, err := decideMode("clipboard paste", mode, true, "")
		if err != nil {
			return nil, err
		}
		o := obj{}
		if a["app"] != "" {
			o["app"] = a["app"]
		}
		return withDryRun(o, m, r), nil
	}
	if e := requireX11("clipboard paste"); e != nil {
		return nil, e
	}
	text := a["text"]
	if text != "" {
		if err := writeClipboard(text); err != nil {
			return nil, errf("Cannot write the clipboard: %v", err)
		}
	} else {
		s, e := clipboardText()
		if e != nil {
			return nil, e
		}
		text = s
	}
	ap, e := targetApp(a["app"], true)
	if e != nil {
		return nil, e
	}
	o := obj{"app": ap.Name, "pasted": text}
	m, reason, err := decideMode("clipboard paste", mode, true, "")
	if err != nil {
		return nil, err
	}
	if m == "background" {
		why := ""
		if b, _, ae := ap.needA11y(); ae != nil {
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
			if text == "" || b.insertText(n, pos, text) {
				b.setCaret(n, pos+len([]rune(text)))
				o["method"], o["mode"], o["element"] = "accessibility", "background", shortElem(n)
				return o, nil
			}
			why = "the app refused the text insertion"
		}
		if mode == "background" {
			return nil, errObj(obj{"error": "Background mode not possible for clipboard paste: " + why, "mode": "background", "app": ap.Name})
		}
		reason = why
	}
	x, xe := X("clipboard paste")
	if xe != nil {
		return nil, xe
	}
	if e := x.requireXTest("clipboard paste"); e != nil {
		return nil, e
	}
	fs := x.beginForeground(ap.frontWindow(x), mode == "auto")
	_ = x.pressKeysym('v', []xproto.Keysym{ksControlL})
	time.Sleep(150 * time.Millisecond)
	o["method"] = "xtest"
	setFallback(o, "foreground", reason, fs.end())
	return o, nil
}
