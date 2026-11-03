# aux4 computer linux unhide

```afterAll
rm -rf state
```

## dry run

### should act in the background

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux unhide mousepad | jq -c '{mode, hidden}'
```

```expect
{"mode":"background","hidden":false}
```

## a missing app

### should fail with a JSON error

```execute
AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux unhide ThisAppDoesNotExist12345 2>&1 | jq -c '{error: (.error | type)}'
```

```expect
{"error":"string"}
```
