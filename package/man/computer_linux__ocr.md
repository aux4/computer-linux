#### Description

Recognizes the text visible in pixels — on the main display, in one app's window, or in an existing PNG — with
Tesseract, and returns each line as an entry in `matches` (the key `find` uses) with its position in the
coordinates `click` uses. Runs locally; nothing is sent anywhere.

Use it for what the accessibility tree doesn't expose (canvas UIs, images, remote desktops, some web content), or to
confirm that what an app reports is what is drawn.

- **No `--app` / `--image`** — reads the main display. `frame` and `center` are screen pixels.
- **`--app`** — reads that app's front window, even when covered. Each entry also has a `windowCenter` (relative to the
  window, the space `click --app` uses) and the output has the window `frame`. A minimized window is restored for the
  capture in `auto` mode and minimized again; `--mode background` errors instead.
- **`--image`** — reads a PNG. Positions are image pixels divided by `--scale` (default `1`). Needs no X11 session.
- **`--text`** — keeps only lines containing that text, ignoring case, whitespace and commas (`1792` matches `1,792`),
  and adds a `match` with the `frame`/`center`/`windowCenter` of just the matched words. No match is `"matches": []`.
- **`--level`** — `accurate` (default) upscales the image 3x before recognition; `fast` 1.5x.
- **`--languages`** — comma-separated BCP 47 (`en-US,pt-BR`) or Tesseract (`eng+por`) codes; default English. A
  language whose Tesseract pack isn't installed errors with the `supported` list.
- **`--timeout`** — a hard deadline (default `60s`): exit `1` with a JSON error carrying the `phase` (`capture` or
  `recognition`), `timeout` and `waited`.

Dark backgrounds are inverted before recognition. Small symbols can be misread: check `confidence` (`0`-`1`) and prefer
`find` when the app exposes the element. Needs Tesseract (`tesseract-ocr`).

#### Usage

```bash
aux4 computer linux ocr [--app <name>] [--image <png>] [--text <filter>] [--level accurate|fast] [--languages <list>] [--mode auto|background|foreground] [--scale <pixels per point>] [--timeout <duration>]
```

--app        Read this app's front window (default: the main display)
--image      Read an existing PNG instead (cannot be combined with `--app`)
--text       Only lines containing this text (default: all lines)
--level      `accurate` or `fast` (default: `accurate`)
--languages  Comma-separated languages, e.g. `en-US,pt-BR` (default: English)
--mode       `auto`, `background` or `foreground` (default: `auto`)
--scale      With `--image`: image pixels per point (default: `1`)
--timeout    Give up after this long (default: `60s`)

#### Example

```bash
aux4 computer linux ocr --app galculator --text 1792
```

```json
{"app":"Galculator","height":343,"level":"accurate","matches":[{"center":{"x":739.7,"y":293},"confidence":0.97,"frame":{"height":26,"width":89.3,"x":695,"y":280},"match":{"center":{"x":739.7,"y":293},"frame":{"height":26,"width":89.3,"x":695,"y":280},"windowCenter":{"x":265.7,"y":57}},"text":"1792","windowCenter":{"x":265.7,"y":57}}],"mode":"background","scale":1,"source":"window","text":"1792","width":331,"window":{"frame":{"height":343,"width":331,"x":474,"y":236},"id":6291459,"title":"galculator"}}
```

```bash
aux4 computer linux ocr --image receipt.png --text total | jq -c '[.matches[].text]'
```

```json
["Total: 1,792"]
```
