#### Description

Lists an app's menu bar, or activates one of its items by path — **in the background**: the item's action runs through
AT-SPI without opening the menu, moving the pointer or activating the app.

- **Without `--path`** — the menu bar as a tree: each item's `title`, `ref`, `checked: true` for checked toggle/radio
  items, `shortcut` (as a `shortcut` combo, e.g. `ctrl+shift+s`), `enabled: false` for disabled items and `children`.
  `--depth` limits the levels (`1` = menu titles, `2` = their items, the default); deeper items say `"submenu":true`.
  Separators are skipped; at most 600 items.
- **With `--path`** — titles separated by `>` (`'View > Scientific Mode'`), matched case-insensitively (a trailing
  `…`/`...` and mnemonic underscores don't matter). Returns the resolved `path`, `ref` and, for a toggle or radio item,
  its new `checked` state. A title that doesn't match errors with the items `Available` at that level; a disabled item
  errors; a path ending on a submenu lists its `items`.

Apps with a header bar or a "hamburger" menu instead of a menu bar have no menu bar to list; use `find --role AXButton`.
Activating an item is recorded in the action log and refused while the stop switch is on. Needs accessibility.

#### Usage

```bash
aux4 computer linux menu [--app <name>] [--path 'Menu > Item'] [--depth 2] [--mode auto|background|foreground]
```

--app     App: window class, accessibility name, process name, desktop id or pid (default: the frontmost app)
--path   Menu item to activate, titles separated by `>`
--depth  Menu levels to list (default: `2`)
--mode    `auto` (default: background, falling back to real input), `background` (error instead of falling back) or `foreground` (real input)

#### Example

```bash
aux4 computer linux menu --app galculator --path "View > Scientific Mode"
```

```json
{"app":"Galculator","checked":true,"method":"Click","mode":"background","path":["View","Scientific Mode"],"pressed":"Scientific Mode","ref":"0.0.0.0.2.1"}
```
