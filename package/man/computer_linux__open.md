#### Description

Launches an app, or opens a file or URL.

- **An app** — by desktop id (`org.gnome.Calculator`), desktop entry name (`Calculator`) or executable name. An app
  that is already running is left as it is (`"alreadyRunning":true`).
- **`--file <path>`** — opens the file in `--app`, or with the default app (`xdg-open`) when `--app` is omitted. The
  file must exist.
- **`--url <url>`** — opens the URL (with its scheme) in `--app` or the default handler.
- **`--wait <timeout>`** — returns once the new window is shown (with `--file`, the window whose title has the file
  name), with its `title`, `frame` and how long it `waited`; times out with exit `1`.
- **`--hidden true`** — minimizes the new window once it appears.

X11 window managers focus new windows; in `auto`/`background` mode `open` waits up to 3s for the new window and gives
the focus back to the window that had it (`restored`). `--mode foreground` leaves the new window in front.

#### Usage

```bash
aux4 computer linux open [<app>] [--file <path>] [--url <url>] [--wait <timeout>] [--hidden true] [--mode auto|background|foreground]
```

app       App: desktop id, name or executable (positional; optional with `--file` / `--url`)
--file    File to open
--url     URL to open
--wait    Wait up to this long for the window, e.g. `10s`
--hidden  `true` to minimize the new window (default: `false`)
--mode    `auto` (default: background, falling back to real input), `background` (error instead of falling back) or `foreground` (real input)

#### Example

```bash
aux4 computer linux open --file /tmp/notes.txt --app mousepad --wait 10s
```

```json
{"app":"Mousepad","command":["/usr/bin/mousepad","/tmp/notes.txt"],"desktopId":"org.xfce.mousepad","file":"/tmp/notes.txt","mode":"background","opened":"mousepad","pid":4120,"restored":{"frontmost":true},"waited":0.62,"window":{"frame":{"height":480,"width":640,"x":639,"y":20},"id":6291459,"title":"notes.txt - Mousepad"}}
```
