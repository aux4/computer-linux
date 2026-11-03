#### Description

Pastes the clipboard text into the app's focused text element. With `--text`, puts that text on the clipboard first. By
default **in the background**: the text is inserted at the caret (replacing a selection) through AT-SPI. Otherwise
`auto` presses a real `ctrl+v` in the app and restores the previously active window; `background` errors instead.

#### Usage

```bash
aux4 computer linux clipboard paste [--text <text>] [--app <name>] [--mode auto|background|foreground]
```

--text   Text to put on the clipboard before pasting
--app     App: window class, accessibility name, process name, desktop id or pid (default: the frontmost app)
--mode    `auto` (default: background, falling back to real input), `background` (error instead of falling back) or `foreground` (real input)

#### Example

```bash
aux4 computer linux clipboard paste --text "pasted by an agent" --app mousepad
```

```json
{"app":"Mousepad","element":{"label":"","ref":"0.0.2.0.0.0","role":"text"},"method":"accessibility","mode":"background","pasted":"pasted by an agent"}
```
