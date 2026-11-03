# aux4 computer linux resume

```afterAll
rm -rf state
```

## turning the switch off

### should remove the STOP file

```execute
rm -rf state; AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux stop > /dev/null; AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux resume | jq -c '{stopped, removed}'; test -f state/STOP || echo gone
```

```expect
{"stopped":false,"removed":true}
gone
```

### should say when there was nothing to resume

```execute
AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux resume | jq -c '{stopped, removed}'
```

```expect
{"stopped":false,"removed":false}
```

### should note that the environment switch is still on

```execute
AUX4_COMPUTER_STOP=1 AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux resume | jq -c '{stopped, note}'
```

```expect
{"stopped":true,"note":"AUX4_COMPUTER_STOP=1 is still set in this environment"}
```
