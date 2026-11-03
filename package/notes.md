## Linux notes (aux4/computer-linux)

Provider name: `linux` — direct calls are `aux4 computer linux <command>`.

### Session type matters

Check `aux4 computer linux info` first: `session` is `x11`, `xwayland`, `wayland` or `none`.

- **x11** — everything works. Screenshots and OCR use the X server, input goes through XTEST, windows through the
  window manager (EWMH), the accessibility tree through AT-SPI.
- **xwayland** (`DISPLAY` and `WAYLAND_DISPLAY` both set) — X11 commands only see X11 apps (XWayland clients);
  Wayland-native apps appear in the accessibility tree but not in `windows`, `screenshot --app` or `ocr --app`.
- **wayland** (no `DISPLAY`) — Wayland does not let clients capture the screen or send input. `screenshot`, `ocr`
  (except `--image`), `move`, `drag`, real-key `shortcut`, `windows`, `window ...`, `focus` and the clipboard return
  an error with `"unsupported": true`, a `reason` and a `hint`. Accessibility commands still work in the background:
  `state`, `read`, `find`, `click --element`, `type`, `set`, `select`, `menu`, `dialog`, `wait` (ax).

### Accessibility (AT-SPI)

- Apps expose their UI through AT-SPI only when toolkit accessibility is on. If `permissions` says accessibility is
  not granted or `toolkitAccessibility` is false: `gsettings set org.gnome.desktop.interface toolkit-accessibility true`,
  then restart the app. GTK 3/4 and Qt apps work; Electron/Chromium apps need `--force-renderer-accessibility`.
- Roles are AT-SPI role names (`push button`, `text`, `label`, `check box`, `menu item`, `frame`, `dialog`...).
  `find --role` also accepts the macOS names (`AXButton`, `AXTextField`, `AXStaticText`, `AXCheckBox`, `AXMenuItem`,
  `AXWindow`...) and `push_button` / `ROLE_PUSH_BUTTON` forms.
- Refs are child-index paths from the app (`0.1.3`): take a new `state`/`find` after the UI changes.
- Widgets that are hidden (not showing) are left out of `read`, `find` and `state`. Menu items are listed by `menu`
  even while their menu is closed, and can be pressed without opening it.
- Text typed or set through AT-SPI goes through the app's own text buffer, so editors mark the document modified as
  with real keystrokes.

### Background vs foreground

- Background (default): `click --element`, `click --x/--y` on an element with an action, `type`, `clipboard paste`,
  `clipboard copy` of a selection, `select`, `set`, `menu --path`, `dialog --press`, `shortcut --app` when the app has a
  menu item with that accelerator (or `ctrl+a`), `scroll` of a scroll area with scroll bars, `window move/resize/
  minimize/restore/close/fullscreen`, `hide`, `unhide`, `open`, `screenshot --app` and `ocr --app` (covered windows are
  captured through the Composite extension).
- Foreground (real input through XTEST; `auto` restores the pointer and the previously active window afterwards):
  double-clicks, right-clicks, `move`, `drag`, `shortcut` without `--app`, and anything whose background path isn't
  available. `focus` and `window raise` are foreground by nature.
- `open` cannot stop a window manager from focusing a new window; in `auto`/`background` it waits up to 3s for the new
  window and gives the focus back to the window that had it (`restored`).

### Keys and coordinates

- `shortcut` takes `ctrl`, `shift`, `alt`, `super`; `cmd`/`command` mean `ctrl` on Linux and `option` means `alt`, so
  `cmd+s` works on every provider. `delete` is Backspace; `forwarddelete` is Delete.
- Coordinates are X11 screen pixels. With `--app`, `--x/--y` and `windowCenter` are relative to the top-left of the
  app's front window content (title bar not included) — the same space as a `screenshot --app` image.
  `screen` reports `scale` from `Xft.dpi` (`1` = 96 dpi).

### OCR

- Text recognition uses Tesseract (`tesseract-ocr`); `info` shows `capabilities.ocr`. `--languages` takes BCP 47
  codes (`en-US`, `pt-BR`) or Tesseract codes (`eng`, `por`); each needs its language pack installed
  (`tesseract-ocr-por`...). Dark backgrounds are inverted and the image is upscaled before recognition.
- Tesseract misreads small symbols more than it misses digits; check `confidence` and prefer `find` where the
  accessibility tree has the text.

### Apps

- `--app` matches the window class (`WM_CLASS`), the accessibility name, the process name, a desktop id
  (`org.gnome.Calculator`) or a pid. `open` takes a desktop id, an app name or an executable on `PATH`.
- `quit` closes the app's windows (as their close buttons do) and waits for the process to exit; a "save changes?"
  dialog is returned as `dialog`. `quit --force true` sends SIGTERM, then SIGKILL. Desktop session processes (window
  manager, panel, X server, D-Bus, AT-SPI) are never quit.
- `hide` minimizes all the app's windows; `unhide` restores them without activating the app.

### Stop switch and log

The stop switch and action log are shared with `aux4 computer`: `${XDG_STATE_HOME:-~/.local/state}/aux4-computer/STOP`
and `actions.jsonl` (or `AUX4_COMPUTER_STATE_DIR`). `AUX4_COMPUTER_STOP=1` stops this process tree.
