#### Description

Sets an element directly through AT-SPI — **in the background** — and returns its value `before` and `after`. The element
is a `ref` from `find`, `state` or `read`. Give exactly one of:

- **`--value <v>`** — replaces the text of a text field or text area (`EditableText`), or the value of a slider or spin
  button (a number). To insert at the caret, use `type`.
- **`--checked true|false`** — activates a check box, toggle button, switch or radio button until it is in that
  state; already there is `"method":"unchanged"`.
- **`--option <title>`** — picks an option of a combo box, radio group, list or tab list by title (case-insensitive);
  the error lists the available `options`.

An empty `--value` can't be passed (it means "not given"). Recorded in the action log (with `--value` truncated);
refused while the stop switch is on.

#### Usage

```bash
aux4 computer linux set --element <ref> (--value <v> | --checked true|false | --option <title>) [--app <name>] [--mode auto|background|foreground]
```

--app     App: window class, accessibility name, process name, desktop id or pid (default: the frontmost app)
--element  Element ref (required)
--value    New text or number
--checked  `true` or `false`
--option   Option title
--mode    `auto` (default: background, falling back to real input), `background` (error instead of falling back) or `foreground` (real input)

#### Example

```bash
aux4 computer linux set --app mousepad --element 0.0.2.0.0.0 --value "Set by aux4"
```

```json
{"after":"Set by aux4","app":"Mousepad","before":"Hello from aux4/computer-linux CI","element":{"label":"","ref":"0.0.2.0.0.0","role":"text"},"method":"EditableText","mode":"background"}
```
