package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type waitCheck struct {
	matches  []any
	source   string
	complete bool // the check finished (a "gone" verdict needs a complete check)
	running  bool
	window   bool
	axTexts  []string
	ocrTexts []string
	ocrNote  string
}

func cmdWait(a map[string]string) (obj, *cmdError) {
	text := a["text"]
	role := a["role"]
	appQ := a["app"]
	gone, e := parseBool("gone", a["gone"], false)
	if e != nil {
		return nil, e
	}
	source := a["source"]
	if source == "" {
		source = "auto"
	}
	if source != "auto" && source != "ax" && source != "ocr" {
		return nil, errf("Invalid --source '%s': use auto, ax or ocr", source)
	}
	timeout, e := parseDuration("timeout", a["timeout"], 10*time.Second)
	if e != nil {
		return nil, e
	}
	interval, e := parseDuration("interval", a["interval"], 500*time.Millisecond)
	if e != nil {
		return nil, e
	}
	if text == "" && role == "" && appQ == "" {
		return nil, errf("Provide --text, --role or --app to wait for")
	}
	if source == "ocr" && role != "" {
		return nil, errf("--source ocr reads pixels and can't match --role")
	}
	if source == "ocr" && text == "" {
		return nil, errf("--source ocr needs --text")
	}
	condition := obj{"gone": gone, "source": source}
	for k, v := range map[string]string{"text": text, "role": role, "app": appQ} {
		if v != "" {
			condition[k] = v
		}
	}
	useOCR := text != "" && (source == "ocr" || source == "auto")
	var notes []string
	if useOCR && (tesseractPath() == "" || requireX11("ocr") != nil || denyPermissions()) {
		if source == "ocr" {
			if tesseractPath() == "" {
				return nil, errObj(obj{"error": "Tesseract is not installed, so text can't be recognized. " + tesseractInstall, "dependency": "tesseract"})
			}
			if e := requireScreen("wait --source ocr"); e != nil {
				return nil, e
			}
		}
		useOCR = false
		notes = append(notes, "OCR skipped: "+map[bool]string{true: "Tesseract is not installed", false: "screen capture is not available"}[tesseractPath() == ""])
	}
	useAX := (text != "" || role != "") && source != "ocr"
	if useAX {
		if _, e := A11y("wait"); e != nil {
			if source == "ax" || !useOCR {
				return nil, e
			}
			useAX = false
			notes = append(notes, "accessibility skipped: "+e.Error())
		}
	}
	start := time.Now()
	deadline := start.Add(timeout)
	var last waitCheck
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		ch := make(chan waitCheck, 1)
		go func() { ch <- waitOnce(text, role, appQ, gone, useAX, useOCR, deadline) }()
		var c waitCheck
		timedOut := false
		select {
		case c = <-ch:
		case <-time.After(remaining):
			timedOut = true
		}
		if timedOut {
			break
		}
		last = c
		satisfied := false
		if gone {
			satisfied = c.complete && len(c.matches) == 0
		} else {
			satisfied = len(c.matches) > 0
		}
		if satisfied {
			o := obj{"waited": secs(time.Since(start)), "source": c.source, "matches": c.matches}
			if gone {
				o["matches"] = []any{}
			}
			if appQ != "" {
				if ap, err := findApp(appQ, false); err == nil {
					o["app"] = ap.Name
				} else {
					o["app"] = appQ
				}
			}
			if len(notes) > 0 {
				o["notes"] = notes
			}
			return o, nil
		}
		if time.Until(deadline) <= interval {
			time.Sleep(time.Until(deadline))
			break
		}
		time.Sleep(interval)
	}
	what := ""
	switch {
	case text != "":
		what = fmt.Sprintf("text '%s'", text)
	case role != "":
		what = "an element with role " + role
	default:
		what = "a window"
	}
	in := ""
	if appQ != "" {
		in = " in " + appQ
	}
	verb := "to appear"
	if gone {
		verb = "to be gone"
	}
	seen := obj{"running": last.running, "window": last.window}
	if useAX {
		seen["ax"] = firstN(last.axTexts, 12)
	}
	if useOCR {
		seen["ocr"] = firstN(last.ocrTexts, 12)
		if last.ocrNote != "" {
			seen["ocrUnavailable"] = last.ocrNote
		}
	}
	if gone {
		seen["present"] = len(last.matches) > 0
	}
	o := obj{"error": fmt.Sprintf("Timed out after %s waiting for %s%s %s", timeout, what, in, verb), "condition": condition, "timeout": timeout.Seconds(), "waited": secs(time.Since(start)), "seen": seen}
	if len(notes) > 0 {
		o["notes"] = notes
	}
	return nil, errObj(o)
}

func firstN(s []string, n int) []string {
	if s == nil {
		return []string{}
	}
	if len(s) > n {
		return s[:n]
	}
	return s
}

func waitOnce(text, role, appQ string, gone, useAX, useOCR bool, deadline time.Time) waitCheck {
	c := waitCheck{}
	var ap *app
	if appQ != "" {
		a, err := findApp(appQ, useAX)
		if err == nil {
			ap = a
			c.running = true
			if x, xe := X("wait"); xe == nil {
				for _, w := range a.Windows {
					if x.mapped(w) && !x.minimized(w) {
						c.window = true
					}
				}
			} else if a.A != nil {
				c.window = true
			}
		}
	} else if useAX {
		if a, err := frontmostApp(true); err == nil {
			ap = a
			c.running, c.window = true, true
		}
	}
	// --app alone: the app is running with an on-screen window
	if text == "" && role == "" {
		c.source, c.complete = "app", true
		if ap != nil && c.window {
			if x, xe := X("wait"); xe == nil {
				for _, w := range ap.Windows {
					if x.mapped(w) && !x.minimized(w) {
						r, _ := x.geometry(w)
						c.matches = append(c.matches, obj{"app": ap.Name, "id": uint32(w), "pid": ap.Pid, "title": x.windowTitle(w), "frame": r.obj()})
					}
				}
			} else {
				c.matches = append(c.matches, obj{"app": ap.Name, "pid": ap.Pid})
			}
		}
		return c
	}
	axDone := !useAX
	if useAX && ap != nil && ap.A != nil {
		if b, _, e := ap.needA11y(); e == nil {
			origin := appOrigin(ap, appQ != "")
			ms := findMatches(b, ap, text, role, origin, true)
			axDone = true
			if len(ms) > 0 {
				c.matches, c.source = ms, "ax"
				if !gone || !useOCR {
					c.complete = true
					return c
				}
			}
			if len(c.axTexts) == 0 {
				nodes, _ := snapshot(b, ap.A.root, 30, 2000, false)
				for _, n := range nodes {
					if l := strings.TrimSpace(n.label()); l != "" && (textRoles[n.role] || n.role == "label" || actionableRoles[n.role]) {
						c.axTexts = append(c.axTexts, cut(l, 60))
					}
				}
			}
		}
	} else if useAX && ap == nil {
		axDone = true
	}
	if useOCR && (ap != nil || appQ == "") {
		remaining := time.Until(deadline)
		if remaining > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), remaining)
			defer cancel()
			appName := ""
			if ap != nil && appQ != "" {
				appName = ap.Name
				if !c.window {
					c.ocrNote = "the window is not on screen"
				}
			}
			if c.ocrNote == "" {
				cap, ce := captureTarget("wait", appName, 0, "background")
				if ce != nil {
					c.ocrNote = ce.Error()
				} else {
					lines, oe := runOCR(ctx, cap.img, "accurate", "eng")
					if oe != nil {
						c.ocrNote = oe.Error()
					} else {
						for _, l := range lines {
							c.ocrTexts = append(c.ocrTexts, cut(l.text, 60))
						}
						var win *rect
						if cap.app != nil {
							r := cap.origin
							win = &r
						}
						ms := ocrMatches(lines, cap.origin, 1, win, text)
						if len(ms) > 0 && len(c.matches) == 0 {
							c.matches, c.source = ms, "ocr"
						}
						if gone {
							if len(ms) == 0 && len(c.matches) > 0 && c.source == "ax" {
								// ax still sees it
							}
							c.source = "ax+ocr"
							if !useAX {
								c.source = "ocr"
							}
							if len(ms) > 0 {
								c.matches = ms
							}
						}
						c.complete = axDone
						return c
					}
				}
			}
		}
	}
	if gone {
		c.source = "ax"
		if !useAX {
			c.source = "ocr"
		}
		// a complete accessibility check with no match counts as gone; a missing OCR doesn't block it
		c.complete = axDone && (useAX || c.ocrNote == "")
	} else {
		c.complete = axDone
	}
	return c
}
