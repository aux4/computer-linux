#### Description

Walks the accessibility tree (AT-SPI) of the frontmost app (or `--app`) and returns a flat list of elements, each with
a `ref` (a child-index path such as `0.2.1`, valid for the current state of the app), `role` (the AT-SPI role name,
e.g. `push button`, `text`, `label`), `label`, `value` (the text of text elements, the value of sliders), `frame` in
screen pixels, and `enabled` / `focused`. Hidden widgets are left out; menus are listed by `menu`. `--depth` bounds the
walk (default `10`). Needs accessibility.

#### Usage

```bash
aux4 computer linux read [--app <name>] [--depth <n>]
```

--app     App to read (default: the frontmost app)
--depth   Maximum tree depth (default: `10`)

#### Example

```bash
aux4 computer linux read --app galculator --depth 4 | jq -c '.elements[0]'
```

```json
{"enabled":true,"focused":false,"frame":{"height":368,"width":333,"x":473,"y":216},"label":"galculator","ref":"0","role":"frame","value":""}
```
