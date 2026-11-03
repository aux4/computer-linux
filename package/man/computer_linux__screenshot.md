#### Description

Captures a screenshot as a PNG. Capturing never moves the pointer or changes the active window.

- Without `--app`, captures the display at `--display` (an index from `screen`, default `0`).
- With `--app`, captures that app's front window content — **even when other windows cover it** (through the X
  Composite extension). The output has the window `frame` in screen pixels. A point in this image divided by `scale`
  is a window-relative point for `click --app --x --y`.
- A minimized window isn't drawn: `auto` restores it for the capture, minimizes it again and puts the active window
  back (`"mode":"foreground"`, `fallbackReason`, `restored`).
- The image is resized down to `--maxWidth` (default `1568`) keeping its aspect ratio; `scale` is image pixels per
  screen pixel.
- `--output` sets the file; without it a temp file is written and its `path` returned.
- **`--annotate true`** draws a numbered box on each element from the accessibility tree and returns `marks`: `n`,
  `ref` (for `click --element`, `set --element`...), `role`, `label`, `value` (text fields), `frame` and `center` in
  screen pixels, `windowCenter` (with `--app`) and `image` (the box in the PNG's pixels). `--roles actionable`
  (default) marks buttons, toggles, fields, check boxes, radio buttons, combo boxes, sliders, menus, tabs, links and
  list rows; `--roles all` adds labels, images, headings and cells. At most `--limit` marks (default `150`);
  beyond that `truncated` and a `note` say so. Without `--app`, the frontmost app is annotated.

Needs an X11 session (on Wayland it returns `"unsupported": true`); `--annotate` also needs accessibility.

#### Usage

```bash
aux4 computer linux screenshot [--output <path>] [--display <index>] [--app <name>] [--maxWidth <pixels>] [--annotate <true|false>] [--roles <actionable|all>] [--limit <n>]
```

--output     File path to save the PNG to (default: a temp file)
--display    Display index to capture (default: `0`)
--app        Capture this app's front window instead of a display
--maxWidth   Resize down if wider than this (default: `1568`)
--annotate   Draw numbered boxes on the elements and return them as `marks` (default: `false`)
--roles      With `--annotate`: `actionable` (default) or `all`
--limit      With `--annotate`: most marks to draw (default: `150`)

#### Example

```bash
aux4 computer linux screenshot --app galculator --output /tmp/galculator.png
```

```json
{"app":"Galculator","height":343,"path":"/tmp/galculator.png","scale":1,"width":331,"window":{"frame":{"height":343,"width":331,"x":474,"y":236},"id":4194307,"title":"galculator"}}
```

```bash
aux4 computer linux screenshot --app galculator --annotate true --output /tmp/galculator.png | jq '.marks[] | select(.label == "7")'
```

```json
{
  "center": {"x": 574.5, "y": 436},
  "frame": {"height": 34, "width": 59, "x": 545, "y": 419},
  "image": {"height": 34, "width": 59, "x": 71, "y": 183},
  "label": "7",
  "n": 14,
  "ref": "0.0.0.1.2.0.23",
  "role": "toggle button",
  "windowCenter": {"x": 100.5, "y": 200}
}
```
