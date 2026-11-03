# aux4 computer linux click

```afterAll
rm -rf state
```

## without permission

### should error with guidance, never a silent no-op

```execute
AUX4_COMPUTER_DENY_PERMISSIONS=1 AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux click --x 100 --y 100
```

```error:partial
Accessibility permission not granted*?
```

## mode decision (dry run, never touches the screen)

### should act in the background by default

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux click --x 100 --y 200
```

```expect:json
{
  "button": "left",
  "count": 1,
  "dryRun": true,
  "mode": "background",
  "x": 100,
  "y": 200
}
```

### should fall back to foreground and say why when background isn't possible

```execute
AUX4_COMPUTER_DRY_RUN=1 AUX4_COMPUTER_SIMULATE_BACKGROUND=unavailable aux4 computer linux click --x 100 --y 200 --app galculator
```

```expect:json
{
  "app": "galculator",
  "button": "left",
  "count": 1,
  "dryRun": true,
  "fallbackReason": "background simulated as unavailable",
  "mode": "foreground",
  "x": 100,
  "y": 200
}
```

### should error instead of falling back with --mode background

```execute
AUX4_COMPUTER_DRY_RUN=1 AUX4_COMPUTER_SIMULATE_BACKGROUND=unavailable aux4 computer linux click --x 100 --y 200 --mode background
```

```error:partial
Background mode not possible for click: background simulated as unavailable
```

### should use real input with --mode foreground, without a fallback reason

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux click --x 100 --y 200 --mode foreground | jq -c '{mode, fallbackReason}'
```

```expect
{"mode":"foreground","fallbackReason":null}
```

### should fall back for a double-click, which has no accessibility equivalent

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux click --x 100 --y 200 --count 2 | jq -c '{mode, fallbackReason}'
```

```expect
{"mode":"foreground","fallbackReason":"multi-click has no accessibility equivalent"}
```

### should fall back for a right-click

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux click --x 100 --y 200 --button right | jq -c '{mode, fallbackReason}'
```

```expect
{"mode":"foreground","fallbackReason":"right-click has no accessibility equivalent"}
```

### should press an element ref in the background

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux click --element 0.1.2 --app galculator
```

```expect:json
{
  "app": "galculator",
  "clicked": "0.1.2",
  "dryRun": true,
  "mode": "background"
}
```

## argument validation

### should reject an unknown mode

```execute
aux4 computer linux click --x 1 --y 1 --mode sideways
```

```error:partial
Invalid --mode 'sideways'*?
```

### should reject an unknown button

```execute
aux4 computer linux click --x 1 --y 1 --button middle
```

```error:partial
Invalid --button 'middle'*?
```

### should require coordinates or an element

```execute
aux4 computer linux click
```

```error:partial
Provide --x*?
```

### should reject a non-numeric coordinate

```execute
aux4 computer linux click --x left --y 1
```

```error:partial
Invalid --x 'left'*?
```

## a missing app

### should fail with a JSON error

```execute
AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux click --x 1 --y 1 --app ThisAppDoesNotExist12345
```

```error
{"error":"App not running: ThisAppDoesNotExist12345"}
```
