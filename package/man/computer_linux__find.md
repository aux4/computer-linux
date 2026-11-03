#### Description

Searches the accessibility tree of the frontmost app (or `--app`) for elements whose label or value contains the text
(case-insensitive) and/or whose role is `--role`. `--role` takes AT-SPI role names (`push button`, `push_button`,
`ROLE_PUSH_BUTTON`) or the macOS names other providers use (`AXButton`, `AXTextField`, `AXTextArea`, `AXStaticText`,
`AXCheckBox`, `AXMenuItem`, `AXWindow`...). An empty text with a role lists every element with that role. Each match is
like a `read` element plus a `center` (screen pixels) and, with `--app`, a `windowCenter` relative to the app's front
window, ready for `click --app --x --y`. Needs accessibility.

#### Usage

```bash
aux4 computer linux find <text> [--role <role>] [--app <name>]
```

text     Text to search for in labels and values (positional)
--role   Role filter
--app    App to search (default: the frontmost app)

#### Example

```bash
aux4 computer linux find 1792 --app galculator
```

```json
{"app":"Galculator","matches":[{"center":{"x":639.5,"y":294},"enabled":true,"focused":true,"frame":{"height":52,"width":317,"x":481,"y":268},"label":"","ref":"0.0.0.1.0.0.0.0","role":"text","value":"1792","windowCenter":{"x":165.5,"y":58}}]}
```
