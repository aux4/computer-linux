#### Description

One call that tells an agent where things stand, without changing anything:

- **`frontmost`** — the app owning the active window (`name`, `pid`, `class`)
- **`app`** — the target app (`--app`, or the frontmost): `name`, `pid`, `class`, `hidden`, `frontmost`
- **`windows`** — its windows (up to 20), as `window list` reports them
- **`focused`** — the focused element shaped like a `find` match, or `null` (an inactive window has no focused element)
- **`dialog`** — the open dialog, as `dialog` reports it, or `null`
- **`elements`** — a compact tree of the dialog, else the active (or first) window: actionable and text elements only,
  each with `ref`, `role`, `label`, `value` (both cut at 80 characters), `frame` (whole pixels), `enabled:false` /
  `focused:true` when so; `root` is the ref the tree starts at. Menus are left to `menu`.

`--depth` counts levels of those elements (layout containers don't count; default `3`, `0` = no tree). `--limit` caps
the list (default `120`); `truncated` and a `note` say when there is more. Password fields never report a value.
Needs accessibility; read-only, so it works while the stop switch is on.

#### Usage

```bash
aux4 computer linux state [--app <name>] [--depth 3] [--limit 120]
```

--app    App to describe (default: the frontmost app)
--depth  Levels of actionable/text elements (default: `3`)
--limit  Maximum number of elements (default: `120`)

#### Example

```bash
aux4 computer linux state --app mousepad
```

```json
{"app":{"class":"Mousepad","frontmost":true,"hidden":false,"name":"Mousepad","pid":5686},"depth":3,"dialog":null,"elements":[{"focused":true,"frame":{"height":449,"width":638,"x":640,"y":48},"label":"","ref":"0.0.2.0.0.0","role":"text"}],"focused":{"center":{"x":959,"y":272.5},"enabled":true,"focused":true,"frame":{"height":449,"width":638,"x":640,"y":48},"label":"","ref":"0.0.2.0.0.0","role":"text","value":"","windowCenter":{"x":320,"y":252.5}},"frontmost":{"class":"Mousepad","name":"Mousepad","pid":5686},"root":"0","windows":[{"focused":true,"frame":{"height":480,"width":640,"x":639,"y":20},"fullscreen":false,"id":6291459,"index":0,"main":true,"minimized":false,"ref":"0","title":"Untitled 1 - Mousepad"}]}
```
