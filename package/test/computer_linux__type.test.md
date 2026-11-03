# aux4 computer linux type

## mode decision (dry run)

### should type in the background by default

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux type 'héllo 👋' --app mousepad
```

```expect:json
{
  "app": "mousepad",
  "dryRun": true,
  "mode": "background",
  "typed": 7
}
```

### should fall back to real keystrokes when the background isn't possible

```execute
AUX4_COMPUTER_DRY_RUN=1 AUX4_COMPUTER_SIMULATE_BACKGROUND=unavailable aux4 computer linux type hello --app mousepad | jq -c '{mode, fallbackReason}'
```

```expect
{"mode":"foreground","fallbackReason":"background simulated as unavailable"}
```

## argument validation

### should reject an invalid delay

```execute
aux4 computer linux type hello --delay soon
```

```error:partial
Invalid --delay 'soon'*?
```

### should reject an unknown mode

```execute
aux4 computer linux type hello --mode sideways
```

```error:partial
Invalid --mode 'sideways'*?
```
