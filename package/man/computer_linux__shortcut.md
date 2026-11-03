#### Description

Presses a key combo such as `ctrl+s`, `ctrl+shift+n`, `alt+f4` or `enter`. The last part is the key; the others are
modifiers: `ctrl`/`control`, `shift`, `alt`/`option`, `super`/`win`/`meta`. **`cmd`/`command` mean `ctrl`**, so the
same combo works on every provider. Keys: letters, digits, punctuation, `enter`, `tab`, `space`, `escape`,
`backspace` (also `delete`), `forwarddelete`, arrows, `home`, `end`, `pageup`, `pagedown`, `f1`-`f24`.

- **Without `--app`** — a system-wide shortcut, sent as real input to whatever has the focus (`"mode":"foreground"`);
  `--mode background` is refused.
- **With `--app`** — in the background when the app has an enabled menu item with that accelerator: the item is
  activated through AT-SPI (`"method":"menu"`, `menuItem`). `ctrl+a` selects all the text of the focused element
  (`"method":"accessibility"`). Otherwise `auto` brings the window forward, presses the keys and restores the
  previously active window; `background` errors.

`--delay` waits before pressing (milliseconds).

#### Usage

```bash
aux4 computer linux shortcut <keys> [--app <name>] [--delay <ms>] [--mode auto|background|foreground]
```

keys      Key combo, e.g. `ctrl+shift+s` (positional)
--app     Run the shortcut in this app (default: system-wide)
--delay   Milliseconds to wait before pressing (default: `0`)
--mode    `auto` (default: background, falling back to real input), `background` (error instead of falling back) or `foreground` (real input)

#### Example

```bash
aux4 computer linux shortcut ctrl+c --app galculator
```

```json
{"app":"Galculator","keys":"ctrl+c","menuItem":"Copy Display Value","method":"menu","mode":"background"}
```

```bash
aux4 computer linux shortcut ctrl+alt+t
```

```json
{"keys":"ctrl+alt+t","method":"xtest","mode":"foreground"}
```
