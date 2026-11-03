# Live: type, select, clipboard, set (mousepad)

Drives mousepad: background typing (with a concurrent `wait` catching the text as it appears),
select by all/text/range, clipboard copy/paste, setting a text area's value directly, and
scrolling a long document in the background. Also checks mousepad's state shape.

```timeout
20000
```

```beforeAll
aux4 computer linux open mousepad --wait 15s > /dev/null
```

```afterAll
aux4 computer linux quit mousepad --force true > /dev/null 2>&1 || true
```

## typing and wait

### should type in the background and have a concurrent wait catch the text

```timeout
15000
```

```execute
TEXT="Hello from aux4/computer-linux CI"
aux4 computer linux wait --app mousepad --text "$TEXT" --timeout 10s > /tmp/computer-linux-wait-type.json 2> /tmp/computer-linux-wait-type.err &
WP=$!
sleep 1
aux4 computer linux type "$TEXT" --app mousepad | jq -c '{mode, method, typed}'
WRC=0; wait "$WP" || WRC=$?
jq -c --argjson rc "$WRC" '{rc: $rc, waited: (.waited < 10), hasMatches: (.matches | length > 0)}' /tmp/computer-linux-wait-type.json
```

```expect:partial
{"mode":"background","method":"accessibility","typed":33}
{"rc":0,"waited":true,"hasMatches":true}
```

### should find the typed text

```execute
aux4 computer linux find "Hello from aux4/computer-linux CI" --app mousepad | jq -c '.matches | length > 0'
```

```expect
true
```

## select and clipboard

### should select all and have range start at 0

```execute
aux4 computer linux select --app mousepad --all true --mode background | jq -c '{mode, selected, start: .range.start}'
```

```expect
{"mode":"background","selected":"Hello from aux4/computer-linux CI","start":0}
```

### should copy the selection to the clipboard

```execute
aux4 computer linux clipboard copy --app mousepad --mode background | jq -c '{mode, text}'
aux4 computer linux clipboard read | jq -c '.text'
```

```expect:partial
{"mode":"background","text":"Hello from aux4/computer-linux CI"}
"Hello from aux4/computer-linux CI"
```

### should select a word by text

```execute
aux4 computer linux select --app mousepad --text aux4 --mode background | jq -c '{selected, start: .range.start, length: .range.length}'
```

```expect
{"selected":"aux4","start":11,"length":4}
```

### should select a range by start/length

```execute
aux4 computer linux select --app mousepad --start 0 --length 5 --mode background | jq -c '.selected'
```

```expect
"Hello"
```

### should reject selecting text that is not present

```execute
aux4 computer linux select --app mousepad --text "not in the text"
```

```error:partial
*?
```

### should select-all via the ctrl+a shortcut

```execute
aux4 computer linux shortcut ctrl+a --app mousepad --mode background | jq -c '{mode, selected}'
```

```expect
{"mode":"background","selected":33}
```

## set and paste

### should set the text area's value directly

```execute
AREA=$(aux4 computer linux find "" --role AXTextArea --app mousepad | jq -r '[.matches[] | select(.role == "text")][0].ref')
aux4 computer linux set --app mousepad --element "$AREA" --value "Set by aux4/computer-linux CI" --mode background | jq -c '{before, after, mode}'
```

```expect
{"before":"Hello from aux4/computer-linux CI","after":"Set by aux4/computer-linux CI","mode":"background"}
```

### should select start 4 length 2 as 'by'

```execute
aux4 computer linux select --app mousepad --start 4 --length 2 | jq -c '.selected'
```

```expect
"by"
```

### should paste BY over the selection

```execute
aux4 computer linux clipboard paste --text "BY" --app mousepad | jq -c '{mode, pasted}'
```

```expect
{"mode":"background","pasted":"BY"}
```

### should reflect the pasted text in the value

```execute
aux4 computer linux find "" --role AXTextArea --app mousepad | jq -c '[.matches[] | select(.role == "text")][0].value'
```

```expect
"Set BY aux4/computer-linux CI"
```

### should copy and read plain text through the clipboard

```execute
aux4 computer linux clipboard copy --text "plain text" | jq -c '.text'
aux4 computer linux clipboard read | jq -c '.text'
```

```expect:partial
"plain text"
"plain text"
```

## scroll

### should scroll a 300-line document in the background

```execute
AREA=$(aux4 computer linux find "" --role AXTextArea --app mousepad | jq -r '[.matches[] | select(.role == "text")][0].ref')
LINES=$(seq 1 300 | sed 's/^/line /')
aux4 computer linux set --app mousepad --element "$AREA" --value "$LINES" > /dev/null
aux4 computer linux scroll --app mousepad --dy=-400 --mode background | jq -c '{mode, method, scrolled: (.vertical > 0)}'
```

```expect
{"mode":"background","method":"accessibility","scrolled":true}
```

## state

### should report mousepad's state with a text element

```execute
aux4 computer linux state --app mousepad | jq -c '[.elements[] | select(.role == "text")] | length > 0'
```

```expect
true
```
