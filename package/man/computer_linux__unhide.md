#### Description

Restores an app's minimized windows without bringing it to the front; if the window manager activates it anyway, the
previously active window is put back (`restored`).

#### Usage

```bash
aux4 computer linux unhide <app>
```

--app   App name, window class or pid

#### Example

```bash
aux4 computer linux unhide mousepad
```

```json
{
  "app": "Mousepad",
  "before": {
    "hidden": true
  },
  "changed": true,
  "hidden": false,
  "method": "map",
  "mode": "background",
  "pid": 3757,
  "restored": {
    "frontmost": true
  }
}
```
