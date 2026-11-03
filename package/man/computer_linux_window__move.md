#### Description

Moves a window so the top-left corner of its content is at `--x`/`--y` (screen pixels) with
`_NET_MOVERESIZE_WINDOW`, and waits for it. Window managers place frames, not content, so the position is corrected
once if needed; when it still differs a `note` says where it went. A full-screen window is refused. Negative values
need `=`: `--x=-1200`.

#### Usage

```bash
aux4 computer linux window move --x <x> --y <y> [--app <name>] [--title <title> | --index <n>] [--mode auto|background|foreground]
```

--app     App whose window to use (default: the frontmost app)
--title   Window title: exact match first, then contained (case-insensitive)
--index   Window index from `window list`
--x       Screen x of the content's top-left corner
--y       Screen y of the content's top-left corner
--mode    `auto` (default: background, falling back to real input), `background` (error instead of falling back) or `foreground` (real input)

#### Example

```bash
aux4 computer linux window move --app mousepad --x 120 --y 140
```

```json
{
  "action": "move",
  "after": {
    "frame": {
      "height": 480,
      "width": 640,
      "x": 120,
      "y": 140
    },
    "fullscreen": false,
    "id": 4194307,
    "index": 0,
    "minimized": false,
    "ref": "0",
    "title": "*Untitled 1 - Mousepad"
  },
  "app": "Mousepad",
  "before": {
    "frame": {
      "height": 480,
      "width": 640,
      "x": 639,
      "y": 20
    },
    "fullscreen": false,
    "id": 4194307,
    "index": 0,
    "minimized": false,
    "ref": "0",
    "title": "*Untitled 1 - Mousepad"
  },
  "method": "_NET_MOVERESIZE_WINDOW",
  "mode": "background",
  "requested": {
    "x": 120,
    "y": 140
  }
}
```
