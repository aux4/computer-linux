#### Description

Reports what this session allows, as `granted` plus a `guidance` string for each:

- **accessibility** — the AT-SPI accessibility bus is reachable. Needed by `read`, `find`, `state`, `menu`, `select`,
  `set`, `dialog` and every background action. `toolkitAccessibility` says whether toolkits were asked to expose their
  UI (`org.a11y.Status IsEnabled`); when it's `false`, run
  `gsettings set org.gnome.desktop.interface toolkit-accessibility true` and restart the apps.
- **screenRecording** — the X server can be captured (`screenshot`, `ocr` without `--image`). Not granted on Wayland.
- **input** — the X server has the XTEST extension, used for real mouse and keyboard input. Not granted on Wayland.

Every command that needs one of these fails fast with the same guidance rather than doing nothing.

#### Usage

```bash
aux4 computer linux permissions
```

#### Example

```bash
aux4 computer linux permissions
```

```json
{
  "accessibility": {
    "granted": true,
    "guidance": "Accessibility (AT-SPI) is available.",
    "toolkitAccessibility": true
  },
  "input": {
    "granted": true,
    "guidance": "Input through XTEST is available."
  },
  "screenRecording": {
    "granted": true,
    "guidance": "Screen capture is available (X11)."
  }
}
```
