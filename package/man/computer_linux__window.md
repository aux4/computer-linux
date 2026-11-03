#### Description

The `window` group lists and changes an app's windows through the window manager (EWMH): `list`, `move`,
`resize`, `minimize`, `restore`, `close`, `fullscreen` and `raise`. Everything except `raise` works without activating
the app; if the window manager activated it anyway, the previously active window is put back (`restored`).

A window is picked by `--title` (exact, then contained, case-insensitive) or `--index` (from `window list`); by default
the app's front window. Changing actions return the window `before` and `after` (`frame` in screen pixels,
`minimized`, `fullscreen`, `ref` of the matching accessible window), are recorded in the action log (`window move`...)
and are refused while the stop switch is on. Needs an X11 session with an EWMH window manager.

#### Usage

```bash
aux4 computer linux window <list|move|resize|minimize|restore|close|fullscreen|raise> [--app <name>] [--title <title> | --index <n>]
```

#### Example

```bash
aux4 computer linux window move --app mousepad --x 120 --y 140
```
