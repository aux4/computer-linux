#### Description

Blocks until something shows up (or is gone) and returns as soon as it does, instead of sleeping for a guessed
time.

- **`--text`** (optionally with **`--role`**) — an element or text containing it. Each check reads the app's
  accessibility tree first, like `find`; if nothing matches, it reads the app's window (or the main display) with
  OCR, like `ocr`.
- **`--role`** alone — any element with that role (accessibility only).
- **`--app`** alone — the app is running and has an on-screen window. Needs no permission.
- **`--gone true`** — wait until it is no longer there.
- **`--source`** — `auto` (default), `ax` (accessibility only) or `ocr` (pixels only; no `--role`). In `auto`, OCR is
  skipped (and `notes` says why) when Tesseract or screen capture isn't available.
- **`--timeout`** (default `10s`) and **`--interval`** (default `500ms`) accept `500ms`, `10s`, `2m` or seconds.

Text matching ignores case, whitespace and commas. On success it prints `waited` (seconds), `source` (`ax`, `ocr`,
`app`; `ax+ocr` for `--gone`), the `app`, and `matches` shaped like `find`, `ocr` or `windows` results. On timeout it
exits `1` with a JSON error naming the `condition` and what it `seen` on the last check: whether the app was
`running` with a `window`, and the first texts from accessibility (`ax`) and OCR (`ocr`). `wait` only reads; it always
stops at `--timeout`.

#### Usage

```bash
aux4 computer linux wait [<text>] [--role <role>] [--app <name>] [--gone true] [--source auto|ax|ocr] [--timeout 10s] [--interval 500ms]
```

text        Text to wait for (positional or `--text`)
--role      Only elements with this role, e.g. `AXButton` or `push button`
--app       App to watch; alone, wait for it to have an on-screen window
--gone      `true` to wait until it's no longer there (default: `false`)
--source    `auto`, `ax` or `ocr` (default: `auto`)
--timeout   How long to wait (default: `10s`)
--interval  Pause between checks (default: `500ms`)

#### Example

```bash
aux4 computer linux wait --app galculator --text 1792
```

```json
{"app":"Galculator","matches":[{"center":{"x":639.5,"y":294},"enabled":true,"focused":true,"frame":{"height":52,"width":317,"x":481,"y":268},"label":"","ref":"0.0.0.1.0.0.0.0","role":"text","value":"1792","windowCenter":{"x":165.5,"y":58}}],"source":"ax","waited":2.12}
```
