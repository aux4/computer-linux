# aux4 computer linux shortcut

## mode decision (dry run)

### should press a system-wide shortcut as real input

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux shortcut ctrl+alt+t
```

```expect:json
{
  "dryRun": true,
  "keys": "ctrl+alt+t",
  "mode": "foreground"
}
```

### should run a shortcut in an app in the background

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux shortcut cmd+s --app mousepad | jq -c '{app, mode}'
```

```expect
{"app":"mousepad","mode":"background"}
```

### should refuse --mode background without an app

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux shortcut ctrl+c --mode background
```

```error:partial
Background mode not possible for shortcut*?
```

## argument validation

### should reject an unknown modifier

```execute
aux4 computer linux shortcut hyper+x
```

```error:partial
Invalid modifier 'hyper'*?
```

### should reject an unknown key

```execute
aux4 computer linux shortcut ctrl+nosuchkey
```

```error:partial
Unknown key 'nosuchkey'*?
```
