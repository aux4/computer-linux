#### Description

Selects text in a text element through AT-SPI — **in the background**: no keystrokes, no focus change. Acts on the app's
focused text element (or the window's first editable one), or on `--element <ref>`. Give exactly one of:

- **`--all true`** — the whole text
- **`--text <text>`** — a match (case-insensitive); `--occurrence N` picks the Nth (default `1`)
- **`--start N --length N`** — a range of characters from `0`

Returns the `selected` text (up to 4000 characters, `selectedTruncated` beyond), its `range` and the `element`. A
following `clipboard copy --app` copies it in the background. Errors name the problem: text not found (with the number
of `occurrences`), a range past the end, or an element without a text selection.

#### Usage

```bash
aux4 computer linux select [--app <name>] (--all true | --text <text> [--occurrence N] | --start N --length N) [--element <ref>] [--mode auto|background|foreground]
```

--app     App: window class, accessibility name, process name, desktop id or pid (default: the frontmost app)
--all         `true` to select all the text
--text        Text to select
--occurrence  Which match of `--text` (default: `1`)
--start       First character (from `0`)
--length      Number of characters
--element     Text element ref (default: the focused text element)
--mode    `auto` (default: background, falling back to real input), `background` (error instead of falling back) or `foreground` (real input)

#### Example

```bash
aux4 computer linux select --app mousepad --text aux4
```

```json
{"app":"Mousepad","element":{"label":"","ref":"0.0.2.0.0.0","role":"text"},"method":"accessibility","mode":"background","range":{"length":4,"start":11},"selected":"aux4"}
```
