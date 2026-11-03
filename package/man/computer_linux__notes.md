#### Description

Prints markdown notes for agents about using this provider: session types (X11, XWayland, Wayland) and what
each supports, enabling AT-SPI accessibility, role names (AT-SPI names plus the macOS `AX` names `find --role`
accepts), which actions run in the background and which use real input, keys (`cmd` means `ctrl`), coordinates, OCR
with Tesseract, app matching, and the shared stop switch and log. `aux4 ai skill computer prompt` (from
`aux4/computer`) appends these notes to the generic method on Linux.

#### Usage

```bash
aux4 computer linux notes
```

#### Example

```bash
aux4 computer linux notes | head -3
```

```text
## Linux notes (aux4/computer-linux)

Provider name: `linux` — direct calls are `aux4 computer linux <command>`.
```
