#### Description

Copies text to the clipboard. With `--text`, puts that text on the clipboard directly. Without it, copies the app's
current selection and returns what was copied: by default **in the background** — the selected text is read through
AT-SPI and put on the clipboard (an empty selection leaves the clipboard as it was and returns `""`). Otherwise `auto`
presses a real `ctrl+c` in the app and restores the previously active window; `background` errors instead.

On X11 the clipboard lives in the process that owns it: a small background helper keeps the text available until
another app copies something.

#### Usage

```bash
aux4 computer linux clipboard copy [--text <text>] [--app <name>] [--mode auto|background|foreground]
```

--text   Text to put on the clipboard; omit to copy the app's selection
--app     App: window class, accessibility name, process name, desktop id or pid (default: the frontmost app)
--mode    `auto` (default: background, falling back to real input), `background` (error instead of falling back) or `foreground` (real input)

#### Example

```bash
aux4 computer linux clipboard copy --app mousepad
```

```json
{"app":"Mousepad","method":"accessibility","mode":"background","text":"Hello from aux4/computer-linux CI"}
```

```bash
aux4 computer linux clipboard copy --text hello
```

```json
{"text":"hello"}
```
