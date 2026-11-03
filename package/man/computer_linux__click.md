#### Description

Clicks the UI. By default (`--mode auto`) the click happens **in the background**: the app's accessibility element is
activated through AT-SPI (its `click`/`press`/`activate` action), so the pointer doesn't move and the active window
doesn't change. A text field that has no action gets the keyboard focus instead.

- `--element <ref>` (from `find`, `read`, `state` or a screenshot mark) activates that element.
- `--x`/`--y` — with `--app`, relative to the app's front window content (the space of `screenshot --app` and
  `windowCenter`); without, screen pixels in the frontmost app. The deepest element at that point is found in the
  app's accessibility tree, even when the window is covered, and the nearest one with an action is activated.
- `--button right` and `--count 2` have no accessibility equivalent and always use real input.

Modes: `auto` falls back to a real click through XTEST when the background isn't possible (the window is brought
forward, then the pointer and the previously active window are restored: `fallbackReason`, `restored`);
`background` errors instead; `foreground` always clicks for real and restores nothing.

#### Usage

```bash
aux4 computer linux click --x <x> --y <y> [--app <name>] [--button left|right] [--count <n>] [--mode auto|background|foreground]
aux4 computer linux click --element <ref> [--app <name>] [--mode auto|background|foreground]
```

--x         X coordinate (window-relative with `--app`)
--y         Y coordinate (window-relative with `--app`)
--button    `left` or `right` (default: `left`)
--count     Click count, e.g. `2` for a double-click (default: `1`)
--element   Element ref to activate instead of x/y
--app     App: window class, accessibility name, process name, desktop id or pid (default: the frontmost app)
--mode    `auto` (default: background, falling back to real input), `background` (error instead of falling back) or `foreground` (real input)

#### Example

```bash
aux4 computer linux click --app galculator --x 100 --y 280
```

```json
{"action":"Click","app":"Galculator","button":"left","count":1,"element":{"label":"2","ref":"0.0.0.1.2.0.3","role":"toggle button"},"mode":"background","screen":{"x":574,"y":516},"x":100,"y":280}
```

```bash
aux4 computer linux click --app galculator --x 160 --y 58 --count 2
```

```json
{"app":"Galculator","button":"left","count":2,"fallbackReason":"multi-click has no accessibility equivalent","mode":"foreground","restored":{"cursor":true,"frontmost":true},"screen":{"x":634,"y":294},"x":160,"y":58}
```
