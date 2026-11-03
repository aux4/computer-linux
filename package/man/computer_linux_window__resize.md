#### Description

Resizes a window's content to `--width`/`--height` pixels (give one to keep the other). Apps enforce minimum and
maximum sizes: when the result differs, a `note` gives the size it kept.

#### Usage

```bash
aux4 computer linux window resize [--width <pixels>] [--height <pixels>] [--app <name>] [--title <title> | --index <n>] [--mode auto|background|foreground]
```

--app     App whose window to use (default: the frontmost app)
--title   Window title: exact match first, then contained (case-insensitive)
--index   Window index from `window list`
--width   New width (default: keep)
--height  New height (default: keep)
--mode    `auto` (default: background, falling back to real input), `background` (error instead of falling back) or `foreground` (real input)

#### Example

```bash
aux4 computer linux window resize --app mousepad --width 600 --height 400
```

```json
{
  "action": "resize",
  "after": {
    "frame": {
      "height": 400,
      "width": 600,
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
  "method": "_NET_MOVERESIZE_WINDOW",
  "mode": "background",
  "requested": {
    "height": 400,
    "width": 600
  }
}
```
