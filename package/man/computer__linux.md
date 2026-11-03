#### Description

Computer-use on Linux. Every command prints one JSON object on stdout; failures print a JSON error on stderr
and exit `1`.

- **Seeing** — `screenshot` (with numbered element marks via `--annotate`), `ocr` (Tesseract), `state`, `read`,
  `find`, `wait`, `menu`, `dialog`, `screen`, `cursor`, `apps`, `windows`, `window list`
- **Acting in the background** through the AT-SPI accessibility bus — `click`, `type`, `select`, `set`, `menu --path`,
  `dialog --press`, `shortcut --app`, `scroll`, `clipboard copy`/`paste`, `window ...`, `hide`, `unhide`, `open`
- **Real input** through XTEST when needed — `move`, `drag`, double/right clicks, system-wide shortcuts, and the
  foreground fallback of every action
- **Safety** — every input or state change goes to the shared action log and is refused while the shared stop
  switch is on (`stop`, `resume`, `log`)
- **Provider info** — `info` (contract version, session type, capabilities, permissions) and `notes` (platform notes
  for agents)

X11 sessions get everything. On Wayland (no `DISPLAY`) the accessibility commands still work, and screen capture,
input and window management return an error with `"unsupported": true`.

#### Usage

```bash
aux4 computer linux <command> [options]
```

#### Example

```bash
aux4 computer linux info | jq .capabilities
```

```json
{"accessibility":true,"annotate":true,"background":true,"clipboard":true,"dialogs":true,"input":true,"menus":true,"ocr":true,"ocrScreen":true,"screenshot":true,"windows":true}
```
