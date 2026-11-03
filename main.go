package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// argument names, in the order the .aux4 passes them with values(...)
var argSpecs = map[string][]string{
	"info":            {},
	"permissions":     {},
	"screenshot":      {"output", "display", "app", "maxWidth", "annotate", "roles", "limit"},
	"ocr":             {"app", "image", "text", "level", "languages", "mode", "scale", "timeout"},
	"wait":            {"text", "role", "app", "gone", "source", "timeout", "interval"},
	"read":            {"app", "depth"},
	"find":            {"text", "role", "app"},
	"click":           {"x", "y", "button", "count", "element", "app", "mode"},
	"move":            {"x", "y"},
	"drag":            {"x", "y", "toX", "toY"},
	"type":            {"text", "delay", "app", "mode"},
	"shortcut":        {"keys", "delay", "app", "mode"},
	"scroll":          {"x", "y", "dx", "dy", "app", "mode"},
	"cursor":          {},
	"screen":          {},
	"apps":            {},
	"windows":         {},
	"quit":            {"app", "force", "timeout"},
	"hide":            {"app"},
	"unhide":          {"app"},
	"open":            {"app", "mode", "file", "url", "wait", "hidden"},
	"focus":           {"app"},
	"state":           {"app", "depth", "limit"},
	"menu":            {"app", "path", "depth", "mode"},
	"select":          {"app", "all", "text", "occurrence", "start", "length", "element", "mode"},
	"set":             {"app", "element", "value", "checked", "option", "mode"},
	"dialog":          {"app", "press", "mode"},
	"log":             {"limit"},
	"stop":            {},
	"resume":          {},
	"clipboard-copy":  {"text", "app", "mode"},
	"clipboard-paste": {"text", "app", "mode"},
	"clipboard-read":  {},
	"window-list":       {"app"},
	"window-move":       {"app", "title", "index", "mode", "x", "y"},
	"window-resize":     {"app", "title", "index", "mode", "width", "height"},
	"window-minimize":   {"app", "title", "index", "mode"},
	"window-restore":    {"app", "title", "index", "mode"},
	"window-close":      {"app", "title", "index", "mode"},
	"window-fullscreen": {"app", "title", "index", "mode", "enabled"},
	"window-raise":      {"app", "title", "index", "mode"},
}

type handler func(a map[string]string) (obj, *cmdError)

var handlers map[string]handler

// isAction: commands that send input or change state — logged and refused while stopped.
func isAction(cmd string, a map[string]string) bool {
	switch cmd {
	case "click", "type", "shortcut", "scroll", "drag", "move", "set", "select", "open", "focus",
		"quit", "hide", "unhide", "clipboard-copy", "clipboard-paste",
		"window-move", "window-resize", "window-minimize", "window-restore", "window-close", "window-fullscreen", "window-raise":
		return true
	case "menu":
		return a["path"] != ""
	case "dialog":
		return a["press"] != ""
	}
	return false
}

func logName(cmd string) string {
	if strings.HasPrefix(cmd, "clipboard-") || strings.HasPrefix(cmd, "window-") {
		return strings.Replace(cmd, "-", " ", 1)
	}
	return cmd
}

func main() {
	handlers = map[string]handler{
		"info": cmdInfo, "permissions": cmdPermissions,
		"screenshot": cmdScreenshot, "ocr": cmdOCR, "wait": cmdWait,
		"read": cmdRead, "find": cmdFind, "state": cmdState,
		"click": cmdClick, "move": cmdMove, "drag": cmdDrag, "type": cmdType, "shortcut": cmdShortcut, "scroll": cmdScroll,
		"cursor": cmdCursor, "screen": cmdScreen, "apps": cmdApps, "windows": cmdWindows,
		"quit": cmdQuit, "hide": cmdHide, "unhide": cmdUnhide, "open": cmdOpen, "focus": cmdFocus,
		"menu": cmdMenu, "select": cmdSelect, "set": cmdSet, "dialog": cmdDialog,
		"log": cmdLog, "stop": cmdStop, "resume": cmdResume,
		"clipboard-copy": cmdClipboardCopy, "clipboard-paste": cmdClipboardPaste, "clipboard-read": cmdClipboardRead,
		"window-list": cmdWindowList, "window-move": cmdWindowMove, "window-resize": cmdWindowResize,
		"window-minimize": cmdWindowMinimize, "window-restore": cmdWindowRestore, "window-close": cmdWindowClose,
		"window-fullscreen": cmdWindowFullscreen, "window-raise": cmdWindowRaise,
		"__clipboard-serve": nil,
	}
	if len(os.Args) < 2 {
		fail(errf("Usage: aux4-computer-linux <command> [args...]"))
	}
	cmd := os.Args[1]
	if cmd == "__clipboard-serve" {
		clipboardServe()
		return
	}
	// "window move" / "clipboard copy" given as two words
	if (cmd == "window" || cmd == "clipboard") && len(os.Args) > 2 {
		cmd = cmd + "-" + os.Args[2]
		os.Args = append(os.Args[:2], os.Args[3:]...)
	}
	h, ok := handlers[cmd]
	if !ok || h == nil {
		fail(errf("Unknown command '%s'", cmd))
	}
	args := map[string]string{}
	for i, name := range argSpecs[cmd] {
		if 2+i < len(os.Args) {
			args[name] = os.Args[2+i]
		}
	}
	action := isAction(cmd, args)
	started := time.Now()
	if action {
		if stopped, _ := stopState(); stopped {
			e := stoppedError(logName(cmd))
			logAction(logName(cmd), args, args["app"], nil, e, started)
			fail(e)
		}
	}
	res, err := runWithWatchdog(cmd, h, args)
	if action && !(dryRun() && err == nil) {
		app := args["app"]
		if res != nil {
			if s, ok := res["app"].(string); ok && s != "" {
				app = s
			}
		}
		logAction(logName(cmd), args, app, res, err, started)
	}
	if err != nil {
		fail(err)
	}
	fmt.Println(encode(res))
}

// runWithWatchdog runs a handler with a hard deadline so nothing (an app that stops answering
// on D-Bus, a stalled X server) can hang a command forever.
func runWithWatchdog(cmd string, h handler, args map[string]string) (obj, *cmdError) {
	limit := 120 * time.Second
	for _, k := range []string{"timeout", "wait"} {
		if v := args[k]; v != "" {
			if d, e := parseDuration(k, v, 0); e == nil && d > 0 {
				limit = d + 30*time.Second
			}
		}
	}
	type out struct {
		r obj
		e *cmdError
	}
	ch := make(chan out, 1)
	go func() {
		defer func() {
			if p := recover(); p != nil {
				ch <- out{nil, errf("internal error in %s: %v", logName(cmd), p)}
			}
		}()
		r, e := h(args)
		ch <- out{r, e}
	}()
	select {
	case o := <-ch:
		return o.r, o.e
	case <-time.After(limit):
		return nil, errObj(obj{"error": fmt.Sprintf("%s did not finish within %s", logName(cmd), limit), "timeout": limit.Seconds()})
	}
}

func fail(e *cmdError) {
	fmt.Fprintln(os.Stderr, encode(e.fields))
	os.Exit(1)
}
