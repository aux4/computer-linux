#### Description

Raises a window and activates it (`_NET_ACTIVE_WINDOW`). Foreground by nature: always `"mode":"foreground"`, nothing
is restored, and `--mode background` is refused.

#### Usage

```bash
aux4 computer linux window raise [--app <name>] [--title <title> | --index <n>]
```

--app     App whose window to use (default: the frontmost app)
--title   Window title: exact match first, then contained (case-insensitive)
--index   Window index from `window list`
--mode    `auto` (default) or `foreground`; `background` is refused

#### Example

```bash
aux4 computer linux window raise --app galculator | jq -c '{frontmost, mode}'
```

```json
{"frontmost":true,"mode":"foreground"}
```
