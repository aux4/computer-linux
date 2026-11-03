# Release notes

First release of the Linux provider for `aux4/computer`.

- `aux4 computer linux <command>`: the full computer-use command surface of the provider contract (v1) —
  `screenshot` (with `--annotate` set-of-marks), `ocr` (Tesseract), `wait`, `state`, `read`, `find`, `click`, `type`,
  `shortcut`, `scroll`, `move`, `drag`, `menu`, `select`, `set`, `dialog`, `open`, `focus`, `quit`, `hide`, `unhide`,
  `apps`, `windows`, `window list|move|resize|minimize|restore|close|fullscreen|raise`, `clipboard copy|paste|read`,
  `cursor`, `screen`, `permissions`, plus `info` and `notes`.
- Background actions through the AT-SPI accessibility bus; real input through XTEST with the pointer and active window
  restored afterwards; window management through EWMH; covered windows captured through the Composite extension.
- Wayland sessions are detected: accessibility commands work, screen capture and input return `"unsupported": true`.
- Shares the stop switch and action log with `aux4 computer` (`AUX4_COMPUTER_STATE_DIR`, `AUX4_COMPUTER_STOP`).
- Native binaries for linux/amd64, linux/arm64 and linux/386, with no C dependencies.
