#### Description

Lists an app's windows, including minimized ones: `index` (for `--index`), `id`, `ref` (the accessible window),
`title`, `frame`, `minimized`, `fullscreen`, and `focused`/`main` for the active window or `subrole: dialog` for dialog
windows. At most 50. Read-only. Without X11 it lists the app's accessible windows.

#### Usage

```bash
aux4 computer linux window list [<app>]
```

--app   App whose windows to list (default: the frontmost app)

#### Example

```bash
aux4 computer linux window list mousepad
```

```json
{
  "app": "Mousepad",
  "pid": 3757,
  "windows": [
    {
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
      "title": "*Untitled 1 - Mousepad",
      "focused": true,
      "main": true
    }
  ]
}
```
