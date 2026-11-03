#### Description

Types text (any Unicode, emoji included) into the app's focused text element (`--app`, or the frontmost app).

By default it happens **in the background**: the text is inserted at the element's caret (replacing a selection)
through AT-SPI `EditableText`, so the app can stay behind other windows. Editors record it like typing, so the
document shows as modified. When no element has the focus (the window isn't active), the first visible editable text
element is used (multi-line first) and named in `element`. If the element doesn't accept text this way, `auto` types
real keystrokes through XTEST with the window brought forward, then restores the previously active window
(`"method":"xtest"`); `background` errors instead. `--delay` waits before typing (milliseconds).

#### Usage

```bash
aux4 computer linux type <text> [--app <name>] [--delay <ms>] [--mode auto|background|foreground]
```

text      Text to type (positional)
--app     App: window class, accessibility name, process name, desktop id or pid (default: the frontmost app)
--delay   Milliseconds to wait before typing (default: `0`)
--mode    `auto` (default: background, falling back to real input), `background` (error instead of falling back) or `foreground` (real input)

#### Example

```bash
aux4 computer linux type "Hello from aux4" --app mousepad
```

```json
{"app":"Mousepad","element":{"label":"","ref":"0.0.2.0.0.0","role":"text"},"method":"accessibility","mode":"background","typed":15}
```
