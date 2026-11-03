# aux4/computer-linux

Computer-use for AI agents on Linux: take screenshots (with numbered element marks), read text from pixels (OCR),
read the accessibility tree, find and click UI elements, type, press shortcuts, use menus, select text, set fields,
answer dialogs, scroll, drag, use the clipboard, and manage apps and windows. Every command prints JSON, every
action is recorded in an action log, and a stop switch halts all input at once.

It is the Linux provider of [aux4/computer](https://hub.aux4.io/r/public/packages/aux4/computer): `aux4 computer
<command>` dispatches here on Linux, and the same commands are available directly as `aux4 computer linux <command>`.

It is built to **work alongside the person using the desktop**: actions run in the background by default — through
the AT-SPI accessibility bus, on an app that isn't in front, without moving the pointer or changing the active window
— and fall back to real input only when the background isn't possible.

Native binaries for `linux/amd64`, `linux/arm64` and `linux/386`. Works on X11 sessions; on Wayland the accessibility
commands work and screen capture/input report `"unsupported": true` (see [Platform differences](#platform-differences)).

## Installation

```bash
aux4 aux4 pkger install aux4/computer-linux
```

This installs `aux4/computer` too. Text recognition needs Tesseract (`tesseract-ocr`), which the installer asks your
package manager for.

## Requirements

- An **X11 session** (`DISPLAY` set) for screenshots, OCR of the screen, real input and window management.
- The **AT-SPI accessibility bus** (`at-spi2-core`, part of every desktop) with toolkit accessibility on, for reading
  and acting on apps in the background:

```bash
gsettings set org.gnome.desktop.interface toolkit-accessibility true
```

Apps started before accessibility was enabled must be restarted. GTK 3/4 and Qt apps expose their UI; Electron and
Chromium apps need `--force-renderer-accessibility`.

Check what this session allows:

```bash
aux4 computer linux permissions
```

```json
{
  "accessibility": { "granted": true, "guidance": "Accessibility (AT-SPI) is available.", "toolkitAccessibility": true },
  "input": { "granted": true, "guidance": "Input through XTEST is available." },
  "screenRecording": { "granted": true, "guidance": "Screen capture is available (X11)." }
}
```

`aux4 computer linux info` adds the session type, the capabilities and the contract version.

## Background mode (default)

Every action takes `--mode`:

| Mode | Behavior |
|------|----------|
| `auto` (default) | Act in the background; if that isn't possible, use real input, then restore the pointer and the previously active window |
| `background` | Act in the background only; error instead of falling back |
| `foreground` | Always use real mouse/keyboard input |

Every action's JSON says what happened: `"mode":"background"` or `"mode":"foreground"`, plus a `fallbackReason` (and
what was `restored`) when `auto` fell back. Pass `--app <name>` to target an app that isn't in front.

| Command | In the background | Falls back to real input when |
|---------|-------------------|-------------------------------|
| `click --element` / `--x --y` | Runs the element's action (click/press/activate) through AT-SPI, even when the window is covered; a text field takes the focus | No element with an action at the point; double and right clicks |
| `type` | Inserts the text at the caret of the focused (or first editable) text element | The element doesn't accept text through AT-SPI |
| `clipboard copy` / `paste` | Reads the selection / inserts the clipboard text through AT-SPI | Same as `type` |
| `shortcut --app` | Runs the app's menu item with that accelerator; `ctrl+a` selects all the focused text | No enabled menu item for the keys |
| `menu --path` | Runs the menu item without opening the menu | — (disabled items error) |
| `select`, `set`, `dialog --press` | Sets the selection / value / state, or presses the button through AT-SPI | — |
| `scroll` | Moves the scroll area's scroll bar | No scroll area with scroll bars |
| `window ...`, `hide`, `unhide` | Moves, resizes, minimizes, restores, closes or fullscreens through the window manager | — (`window raise` is always foreground) |
| `open` | Launches the app and gives the focus back to the window that had it | — |
| `screenshot --app`, `ocr --app` | Captures the window even when covered (Composite extension) | The window is minimized (restored for the capture, then minimized again) |

**Always foreground:** `focus`, `move`, `drag`, `window raise`, and `shortcut` without `--app`.

### Coordinates

Coordinates are X11 screen pixels. With `--app`, `--x`/`--y` are relative to the top-left of the app's front window
content — the same space as a `screenshot --app` image (divide image pixels by its `scale`), `find --app`'s
`windowCenter` and `ocr --app`'s `windowCenter`.

```bash
aux4 computer linux click --app galculator --x 100 --y 280
```

```json
{"action":"Click","app":"Galculator","button":"left","count":1,"element":{"label":"2","ref":"0.0.0.1.2.0.3","role":"toggle button"},"mode":"background","screen":{"x":574,"y":516},"x":100,"y":280}
```

## Commands

### Seeing the screen

```bash
aux4 computer linux screenshot [--output file.png] [--display 0] [--app galculator] [--maxWidth 1568]
aux4 computer linux screenshot --app galculator --annotate true [--roles actionable|all] [--limit 150]
aux4 computer linux screen
aux4 computer linux cursor
```

`--annotate true` draws a numbered box on each element and returns `marks` with the `ref`, `role`, `label`, `frame`,
`center`, `windowCenter` and the box in image pixels, so a vision model can say "click 14" and the agent clicks
`--element <ref>`.

### Reading text from pixels

```bash
aux4 computer linux ocr [--app galculator] [--image file.png] [--text "1,792"] [--level accurate|fast] [--languages en-US,pt-BR] [--timeout 60s]
```

Each line of text is a `matches` entry with `text`, `confidence`, `frame` and `center` (plus `windowCenter` with
`--app`). `--text` keeps the lines containing that text — ignoring case, whitespace and commas — and adds a `match`
with the position of just that text. `--image` reads a PNG and needs no X11 session.

```bash
aux4 computer linux ocr --app galculator --text 1792
```

```json
{"app":"Galculator","height":343,"level":"accurate","matches":[{"center":{"x":739.7,"y":293},"confidence":0.97,"frame":{"height":26,"width":89.3,"x":695,"y":280},"match":{"center":{"x":739.7,"y":293},"frame":{"height":26,"width":89.3,"x":695,"y":280},"windowCenter":{"x":265.7,"y":57}},"text":"1792","windowCenter":{"x":265.7,"y":57}}],"mode":"background","scale":1,"source":"window","text":"1792","width":331,"window":{"frame":{"height":343,"width":331,"x":474,"y":236},"id":6291459,"title":"galculator"}}
```

### Waiting for something to appear

```bash
aux4 computer linux wait [--text 1792] [--role AXButton] [--app galculator] [--gone true] [--source auto|ax|ocr] [--timeout 10s] [--interval 500ms]
```

`wait` checks the accessibility tree first and the window's pixels (OCR) when the tree has no match, returns as soon
as the condition holds with how long it `waited`, the `source` and the `matches`; on timeout it exits `1` with what it
saw on the last check.

### One call to see where things stand

```bash
aux4 computer linux state [--app mousepad] [--depth 3] [--limit 120]
```

`state` returns the frontmost app, the target app, its windows, the focused element, the open dialog and a compact
tree of actionable and text elements, each with a `ref` to act on.

### Reading the UI

```bash
aux4 computer linux read [--app mousepad] [--depth 10]
aux4 computer linux find "Save" [--role AXButton] [--app mousepad]
```

Roles are AT-SPI role names (`push button`, `toggle button`, `text`, `label`, `check box`, `combo box`, `menu item`,
`frame`, `dialog`...). `find --role` also accepts the macOS names used by the other providers (`AXButton`,
`AXTextField`, `AXTextArea`, `AXStaticText`, `AXCheckBox`, `AXMenuItem`, `AXWindow`...). A `ref` is a child-index path
valid for the current state of the app.

### Acting on the UI

```bash
aux4 computer linux click --x 100 --y 200 [--app galculator] [--button left|right] [--count 2] [--mode auto]
aux4 computer linux click --element 0.0.0.1.2.0.23 [--app galculator]
aux4 computer linux scroll [--x 100 --y 200] --dy=-200 [--app mousepad]
aux4 computer linux move --x 100 --y 200
aux4 computer linux drag --x 100 --y 200 --toX 400 --toY 200
```

### Menus, selections, fields and dialogs

```bash
aux4 computer linux menu [--app galculator] [--depth 2]
aux4 computer linux menu --app galculator --path "View > Scientific Mode"
aux4 computer linux select --app mousepad --all true
aux4 computer linux select --app mousepad --text "aux4" [--occurrence 2]
aux4 computer linux select --app mousepad --start 0 --length 5 [--element 0.0.2.0.0.0]
aux4 computer linux set --app mousepad --element 0.0.2.0.0.0 --value "new text"
aux4 computer linux set --app my-settings --element 0.1.3 --checked true
aux4 computer linux set --app my-settings --element 0.2.4 --option "Dark"
aux4 computer linux dialog [--app mousepad]
aux4 computer linux dialog --app mousepad --press "Don't Save"
```

### Keyboard

```bash
aux4 computer linux type "Hello, world!" [--app mousepad] [--delay 300]
aux4 computer linux shortcut "ctrl+s" --app mousepad
aux4 computer linux shortcut "ctrl+alt+t"
aux4 computer linux shortcut "enter"
```

Modifiers: `ctrl`, `shift`, `alt`, `super`. `cmd`/`command` mean `ctrl` and `option` means `alt`, so `cmd+s` works on
every provider.

### Clipboard

```bash
aux4 computer linux clipboard copy [--text "..."] [--app mousepad]
aux4 computer linux clipboard paste [--text "..."] [--app mousepad]
aux4 computer linux clipboard read
```

Text only. The X11 clipboard lives in the process that owns it: `copy` leaves a small background helper holding the
text until another app copies something.

### Apps and windows

```bash
aux4 computer linux apps
aux4 computer linux windows
aux4 computer linux open galculator [--wait 10s] [--hidden true]
aux4 computer linux open --file ~/notes.txt [--app mousepad] [--wait 10s]
aux4 computer linux open --url https://aux4.io
aux4 computer linux focus mousepad
aux4 computer linux window list mousepad
aux4 computer linux window move --app mousepad [--title notes | --index 0] --x 200 --y 150
aux4 computer linux window resize --app mousepad --width 800 [--height 600]
aux4 computer linux window minimize|restore|close|raise --app mousepad [--title notes]
aux4 computer linux window fullscreen true|false --app mousepad
aux4 computer linux hide mousepad
aux4 computer linux unhide mousepad
aux4 computer linux quit mousepad [--force true] [--timeout 5s]
```

`--app` matches the window class (`WM_CLASS`), the accessibility name, the process name, a desktop id
(`org.gnome.Calculator`) or a pid. `open` takes a desktop id, an app name or an executable on `PATH`. `quit` closes the
app's windows and waits for it to exit; an unsaved document's question comes back as `dialog` for
`dialog --press`. `--force true` terminates the process (unsaved changes are lost); desktop session processes are never
quit.

## Action log and stop switch

```bash
aux4 computer linux log [--limit 20]
aux4 computer linux stop
aux4 computer linux resume
```

The stop switch and the action log are shared with `aux4 computer` and every provider. Every command that sends input
or changes state appends one JSON line to `actions.jsonl` in `${XDG_STATE_HOME:-~/.local/state}/aux4-computer` — time,
provider, command, app, arguments (typed text cut to 16 characters plus its length), mode and result — including
attempts that failed or were refused. The file is private and rotated at 10MB.

`stop` creates the `STOP` file there: every input or state command then refuses with a JSON error (`"stopped":true`)
until `resume`, while read-only commands keep working. Setting `AUX4_COMPUTER_STOP=1` in an agent's environment stops
it the same way. `AUX4_COMPUTER_STATE_DIR` moves the stop file and the log to another directory.

## Platform differences

Compared with `aux4/computer-mac`:

| Area | Linux |
|------|-------|
| Session | X11: everything. XWayland: X11 apps only for capture/input/windows. Wayland (no `DISPLAY`): `screenshot`, `ocr` (except `--image`), `move`, `drag`, real-key `shortcut`, `cursor`, `screen`, `windows`, `window ...`, `focus`, `hide`/`unhide` and the clipboard return `"unsupported": true` with a `reason` and a `hint`; accessibility commands (`state`, `read`, `find`, `click --element`, `type`, `set`, `select`, `menu`, `dialog`, `wait --source ax`) still work. A portal-based Wayland mode is planned. |
| Permissions | No privacy prompts: `accessibility` = the AT-SPI bus is reachable, `screenRecording` = an X server to capture, plus `input` = the XTEST extension. |
| Roles | AT-SPI role names (`push button`, `text`, `label`...); `find --role` accepts the macOS names too. |
| Coordinates | X11 pixels; window-relative coordinates start at the window content (no title bar). |
| Keys | `cmd` = `ctrl`, `option` = `alt`, `super` for the logo key. |
| OCR | Tesseract instead of the macOS Vision framework; `--languages` needs the Tesseract language packs. |
| Menus | Only apps with a menu bar (GTK/Qt menu bars); header-bar apps expose buttons instead. |
| Dialogs | `kind` is `alert`, `dialog`, `save-panel` or `open-panel` (Linux has no sheets). |
| `hide` / `unhide` | Minimize / restore all the app's windows. |
| `quit` | Closes the app's windows (`--force` sends SIGTERM, then SIGKILL). |
| Data loss | Text written through AT-SPI goes through the app's text buffer, so editors mark the document modified (no `documentEdited:false` warning as on macOS). |

## Output and errors

Every command prints one JSON object on stdout. On failure (bad arguments, missing permission, element not found, app
not running) it prints `{"error": "..."}` on stderr and exits non-zero.

## Troubleshooting

- **`App ... does not expose an accessibility tree`** — turn on toolkit accessibility (above) and restart the app.
  Do not set `NO_AT_BRIDGE=1`.
- **`Accessibility permission not granted: AT-SPI is not available`** — the session has no accessibility bus: install
  `at-spi2-core`, and run a bare X session under `dbus-run-session`.
- **`ocr` errors with `Tesseract is not installed`** — install `tesseract-ocr` (and `tesseract-ocr-<lang>` for other
  languages).
