#### Description

Quits an app the polite way: asks each of its windows to close (`_NET_CLOSE_WINDOW`) and waits up to `--timeout`
for the process to exit (`terminated: true`).

- An app with unsaved changes answers with a dialog instead: the result has `terminated: false`, the `dialog` and a
  `note`. Answer it with `dialog --app <name> --press <button>`.
- A dialog that is already open keeps the app from quitting: it is reported the same way.
- `--force true` sends SIGTERM, then SIGKILL after half the timeout: unsaved changes are lost.
- Desktop session processes (window manager, panel, X server, D-Bus, AT-SPI) are never quit.
- If the app neither exits nor asks within `--timeout`, the command fails (exit `1`).

#### Usage

```bash
aux4 computer linux quit <app> [--force <true|false>] [--timeout <duration>]
```

--app      App name, window class or pid
--force    Terminate the process; unsaved changes are lost (default: `false`)
--timeout  How long to wait, e.g. `5s` (default: `5s`)

#### Example

```bash
aux4 computer linux quit galculator
```

```json
{
  "app": "Galculator",
  "force": false,
  "method": "close-windows",
  "mode": "background",
  "pid": 3756,
  "terminated": true,
  "waited": 0.18
}
```
