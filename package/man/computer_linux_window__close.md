#### Description

Asks the window to close (`_NET_CLOSE_WINDOW`, like its close button) and waits up to 2s. `closed` is `true` when it went
away (`after` is `null`). When the app asks first — an unsaved document's "save changes?" dialog — `closed` is `false`
and the result has the `dialog` (as `dialog` reports it) and a `note`; answer it with `dialog --app <name> --press
<button>`.

#### Usage

```bash
aux4 computer linux window close [--app <name>] [--title <title> | --index <n>] [--mode auto|background|foreground]
```

--app     App whose window to use (default: the frontmost app)
--title   Window title: exact match first, then contained (case-insensitive)
--index   Window index from `window list`
--mode    `auto` (default: background, falling back to real input), `background` (error instead of falling back) or `foreground` (real input)

#### Example

```bash
aux4 computer linux window close --app mousepad | jq -c '{closed, kind: .dialog.kind, buttons: [.dialog.buttons[].title]}'
```

```json
{"closed":false,"kind":"alert","buttons":["Cancel","Don't Save","Save"]}
```
