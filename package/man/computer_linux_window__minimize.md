#### Description

Minimizes (iconifies) a window with an ICCCM `WM_CHANGE_STATE` request and waits up to 3s for it.

#### Usage

```bash
aux4 computer linux window minimize [--app <name>] [--title <title> | --index <n>] [--mode auto|background|foreground]
```

--app     App whose window to use (default: the frontmost app)
--title   Window title: exact match first, then contained (case-insensitive)
--index   Window index from `window list`
--mode    `auto` (default: background, falling back to real input), `background` (error instead of falling back) or `foreground` (real input)

#### Example

```bash
aux4 computer linux window minimize --app mousepad
```

```json
{
  "action": "minimize",
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
    "minimized": true,
    "ref": "0",
    "title": "*Untitled 1 - Mousepad"
  },
  "app": "Mousepad",
  "before": {
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
  "method": "WM_CHANGE_STATE",
  "mode": "background"
}
```
