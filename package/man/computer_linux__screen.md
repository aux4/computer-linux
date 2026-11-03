#### Description

Lists the displays (RandR monitors, or the whole X screen): `index` (for `screenshot --display`), `id`, `name`, `main`
(the primary monitor), bounds `x`/`y`/`width`/`height` in pixels and `scale` (from `Xft.dpi`, `1` at 96 dpi).
Read-only; needs an X11 session.

#### Usage

```bash
aux4 computer linux screen
```

#### Example

```bash
aux4 computer linux screen
```

```json
{"displays":[{"height":800,"id":344,"index":0,"main":true,"name":"screen","scale":1,"width":1280,"x":0,"y":0}]}
```
