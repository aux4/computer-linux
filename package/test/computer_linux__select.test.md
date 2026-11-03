# aux4 computer linux select

## argument validation

### should require exactly one way to select

```execute
aux4 computer linux select --app mousepad
```

```error:partial
Give exactly one of --all true*?
```

### should refuse two ways at once

```execute
aux4 computer linux select --app mousepad --all true --text hello
```

```error:partial
Give exactly one of --all true*?
```

### should need --length with --start

```execute
aux4 computer linux select --app mousepad --start 2
```

```error:partial
--start and --length go together*?
```

## dry run

### should select in the background

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux select --app mousepad --text hello | jq -c '{mode, dryRun}'
```

```expect
{"mode":"background","dryRun":true}
```
