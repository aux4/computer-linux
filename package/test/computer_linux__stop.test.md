# aux4 computer linux stop

```afterAll
rm -rf state
```

## the shared stop switch

### should create the STOP file in the state directory

```execute
rm -rf state; AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux stop | jq -c '{stopped, file: (.file | endswith("state/STOP"))}'; test -f state/STOP && echo exists
```

```expect
{"stopped":true,"file":true}
exists
```

### should refuse input commands while it exists

```execute
AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux click --x 1 --y 1 2>&1 | jq -c '{stopped, command, says: (.error | test("^Stopped: .*STOP exists"))}'
```

```expect
{"stopped":true,"command":"click","says":true}
```

### should exit with code 1 when refused

```execute
AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux type hi 2>/dev/null; echo "exit=$?"
```

```expect
exit=1
```

### should refuse even a dry run

```execute
AUX4_COMPUTER_DRY_RUN=1 AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux clipboard paste --app mousepad 2>&1 | jq -c '{stopped, command}'
```

```expect
{"stopped":true,"command":"clipboard paste"}
```

### should record refusals in the action log

```execute
AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux log | jq -c '{stopped, commands: [.entries[].command], results: ([.entries[].result] | unique)}'
```

```expect
{"stopped":true,"commands":["click","type","clipboard paste"],"results":["error"]}
```

### should keep read-only commands working

```execute
AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux log | jq '.entries | type'
```

```expect
"array"
```

### should allow input again after resume

```execute
AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux resume | jq -c '{stopped, removed}'; AUX4_COMPUTER_DRY_RUN=1 AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux click --x 1 --y 1 | jq -c '{dryRun}'
```

```expect
{"stopped":false,"removed":true}
{"dryRun":true}
```

## the environment switch

### should refuse input with AUX4_COMPUTER_STOP=1

```execute
AUX4_COMPUTER_STOP=1 AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux shortcut ctrl+c --app mousepad 2>&1 | jq -c '{stopped, says: (.error | test("AUX4_COMPUTER_STOP=1 is set"))}'
```

```expect
{"stopped":true,"says":true}
```
