package main

import (
	"os"
	"strings"
)

// sessionType reports the graphical session this process sees:
//   x11      DISPLAY set, no Wayland compositor
//   xwayland DISPLAY and WAYLAND_DISPLAY both set: X11 calls reach XWayland, which only sees X11 apps
//   wayland  WAYLAND_DISPLAY set without DISPLAY: no X11 at all
//   none     neither (no graphical session, e.g. ssh)
func sessionType() string {
	disp := os.Getenv("DISPLAY") != ""
	wl := os.Getenv("WAYLAND_DISPLAY") != "" || strings.EqualFold(os.Getenv("XDG_SESSION_TYPE"), "wayland") && !disp
	switch {
	case disp && wl:
		return "xwayland"
	case disp:
		return "x11"
	case wl:
		return "wayland"
	}
	return "none"
}

const waylandHint = "Run the session under X11 (choose an Xorg/X11 session at login) or run the app under XWayland with DISPLAY set; a portal-based Wayland mode is planned. Accessibility commands (read, find, state, click --element, type, set, select, menu, dialog) still work on Wayland."

// requireX11 returns the standard unsupported error when there is no X server to talk to.
func requireX11(command string) *cmdError {
	switch sessionType() {
	case "wayland":
		return unsupported(command, "this is a Wayland session without X11 (WAYLAND_DISPLAY is set, DISPLAY is not); Wayland does not let clients capture the screen or synthesize input", waylandHint)
	case "none":
		return unsupported(command, "no graphical session (DISPLAY is not set)", "Run inside an X11 session, or start one (e.g. Xvfb :99 and export DISPLAY=:99)")
	}
	return nil
}
