#### Description

Restores a minimized window by mapping it again, without activating it; if the window manager activated it anyway,
the previously active window is put back (`restored`). Use `--title`/`--index` to pick a minimized window.

#### Usage

```bash
aux4 computer linux window restore [--app <name>] [--title <title> | --index <n>] [--mode auto|background|foreground]
```

--app     App whose window to use (default: the frontmost app)
--title   Window title: exact match first, then contained (case-insensitive)
--index   Window index from `window list`
--mode    `auto` (default: background, falling back to real input), `background` (error instead of falling back) or `foreground` (real input)

#### Example

```bash
aux4 computer linux window restore --app mousepad
```

```json
{
  "action": "restore",
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
  "method": "map",
  "mode": "background",
  "restored": {
    "frontmost": true
  }
}
```
