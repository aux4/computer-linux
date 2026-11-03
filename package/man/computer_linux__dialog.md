#### Description

Detects an open dialog or alert of an app and describes it; with `--press`, activates one of its buttons by title —
**in the background**.

Detection looks at the app's accessible top-level windows for a dialog, alert or file chooser (or a modal window).
Without `--app`, the frontmost app is checked first, then every app. It returns `"dialog":null`, or:

- **`kind`** — `alert`, `dialog`, `save-panel` or `open-panel` (file choosers)
- **`title`**, **`text`** — the window title and the labels (the message)
- **`buttons`** — each with `title`, `default` (activated by Enter), `cancel`, `enabled`, and `ref` / `frame` /
  `center` / `windowCenter` like a `find` match (up to 25)
- **`fields`** — text fields, check boxes, combo boxes and radio buttons, like `find` matches (up to 15)

`--press <title>` matches a button case-insensitively (curly apostrophes, a trailing `…` and mnemonic underscores
don't matter), activates it and reports `closed` and, when another dialog replaced it, `next`. The error lists the
buttons when the title doesn't match. Pressing is recorded in the action log and refused while the stop switch is on.

#### Usage

```bash
aux4 computer linux dialog [--app <name>] [--press <button title>] [--mode auto|background|foreground]
```

--app    App to check (default: the frontmost app, then every app)
--press  Title of the button to press
--mode    `auto` (default: background, falling back to real input), `background` (error instead of falling back) or `foreground` (real input)

#### Example

```bash
aux4 computer linux dialog --app mousepad | jq -c '.dialog | {kind, text, buttons: [.buttons[] | {title, default, cancel}]}'
```

```json
{"kind":"alert","text":["Save changes to \"Untitled 1\" before closing?","If you don't save the document, all the changes will be lost."],"buttons":[{"title":"Cancel","default":false,"cancel":true},{"title":"Don't Save","default":false,"cancel":false},{"title":"Save","default":true,"cancel":false}]}
```

```bash
aux4 computer linux dialog --app mousepad --press "Don't Save"
```

```json
{"app":"Mousepad","closed":true,"dialog":{"kind":"alert","title":"Question"},"kind":"alert","method":"click","mode":"background","pressed":"Don't Save"}
```
