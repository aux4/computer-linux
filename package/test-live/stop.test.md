# Live: real input, action log and the stop switch (galculator)

Exercises real (foreground) cursor movement and a multi-click fallback, the action log recording
prior commands, and the stop switch: input is refused and logged while stopped, read-only
commands keep working, and resume restores input.

```timeout
15000
```

```beforeAll
aux4 computer linux open galculator --wait 15s > /dev/null
```

```afterAll
aux4 computer linux resume > /dev/null 2>&1 || true
aux4 computer linux quit galculator --force true > /dev/null 2>&1 || true
```

## real input (foreground)

### should move the cursor to 300,300

```execute
aux4 computer linux move --x 300 --y 300 | jq -c '.mode'
aux4 computer linux cursor | jq -c '{x, y}'
```

```expect:partial
"foreground"
{"x":300,"y":300}
```

### should double-click in the foreground and restore the cursor

```execute
aux4 computer linux click --app galculator --x 160 --y 58 --count 2 | jq -c '{mode, multiClick: (.fallbackReason | test("multi-click")), restored: .restored.cursor}'
aux4 computer linux cursor | jq -c '{x, y}'
```

```expect:partial
{"mode":"foreground","multiClick":true,"restored":true}
{"x":300,"y":300}
```

## action log

### should have logged click, type, menu, dialog and quit from earlier live tests

Test files run alphabetically (annotate, calculator, dialog, editor, menu, ocr, stop, window), so
only commands from files before this one are guaranteed to be in the log yet.

```execute
aux4 computer linux log --limit 200 | jq -c '[.entries[].command] as $c | {click: ($c | index("click") != null), type: ($c | index("type") != null), menu: ($c | index("menu") != null), dialog: ($c | index("dialog") != null), quit: ($c | index("quit") != null)}'
```

```expect
{"click":true,"type":true,"menu":true,"dialog":true,"quit":true}
```

## stop switch

### should stop and refuse further input

```execute
aux4 computer linux stop | jq -c '.stopped'
```

```expect
true
```

### should refuse a click while stopped and log it as stopped

```execute
aux4 computer linux click --element 0.0 --app galculator
```

```error:partial
*?
```

### should still allow a read-only command while stopped

```execute
aux4 computer linux find "" --role AXButton --app galculator | jq -c '.matches | length > 0'
```

```expect
true
```

### should resume and allow input again

```execute
aux4 computer linux resume | jq -c '.stopped'
R=$(aux4 computer linux find "" --role AXButton --app galculator | jq -r '[.matches[] | select(.label == "AC")][0].ref // empty')
aux4 computer linux click --element "$R" --app galculator | jq -c '.mode'
```

```expect:partial
false
"background"
```

### should quit galculator

```execute
aux4 computer linux quit galculator | jq -c '.terminated'
```

```expect
true
```
