#### Description

Scrolls in the app. `--dy` is the vertical delta in pixels (negative scrolls down), `--dx` the horizontal one.
`--x`/`--y` pick the scroll area (window-relative with `--app`); without them, the scroll area around the focused
text element (or the window's first one) is used.

By default it happens **in the background**: the scroll area's scroll bar value is moved through AT-SPI
(`"method":"accessibility"`, with the new `vertical`/`horizontal` position from `0` to `1`). Otherwise `auto` scrolls
the wheel for real at the point through XTEST (about 50 pixels per wheel step) and restores the pointer and active
window; `background` errors instead. Pass negative values with `=`: `--dy=-200`.

#### Usage

```bash
aux4 computer linux scroll [--x <x> --y <y>] [--dx=<pixels>] [--dy=<pixels>] [--app <name>] [--mode auto|background|foreground]
```

--x      X to scroll at (window-relative with `--app`)
--y      Y to scroll at (window-relative with `--app`)
--dx     Horizontal delta (default: `0`)
--dy     Vertical delta, negative scrolls down (default: `0`)
--app     App: window class, accessibility name, process name, desktop id or pid (default: the frontmost app)
--mode    `auto` (default: background, falling back to real input), `background` (error instead of falling back) or `foreground` (real input)

#### Example

```bash
aux4 computer linux scroll --app mousepad --dy=-400
```

```json
{"app":"Mousepad","dx":0,"dy":-400,"method":"accessibility","mode":"background","screen":{"x":421,"y":372.5},"vertical":0.083}
```
