#### Description

Lists the on-screen windows of every app (normal windows that are mapped and not minimized): `id`, `app`, `pid`,
`title` and `frame` (the content area in screen pixels). `window list` includes minimized windows. Read-only; needs an
X11 session.

#### Usage

```bash
aux4 computer linux windows
```

#### Example

```bash
aux4 computer linux windows
```

```json
{"windows":[{"app":"Galculator","frame":{"height":343,"width":331,"x":474,"y":236},"id":4194307,"pid":5685,"title":"galculator"}]}
```
