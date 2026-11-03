#### Description

Enters (`true`, the default) or leaves (`false`) full screen through `_NET_WM_STATE_FULLSCREEN` and waits up to 5s.
Leaving full screen puts the window back where the window manager decides.

#### Usage

```bash
aux4 computer linux window fullscreen [true|false] [--app <name>] [--title <title> | --index <n>] [--mode auto|background|foreground]
```

--app     App whose window to use (default: the frontmost app)
--title   Window title: exact match first, then contained (case-insensitive)
--index   Window index from `window list`
--enabled `true` to enter full screen (default), `false` to leave it
--mode    `auto` (default: background, falling back to real input), `background` (error instead of falling back) or `foreground` (real input)

#### Example

```bash
aux4 computer linux window fullscreen true --app mousepad | jq -c '.after.frame, .after.fullscreen'
```

```json
{"height":800,"width":1280,"x":0,"y":0}
true
```
