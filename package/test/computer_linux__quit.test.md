# aux4 computer linux quit

```afterAll
rm -rf state
```

## dry run

### should quit in the background

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux quit mousepad | jq -c '{app, force, mode}'
```

```expect
{"app":"mousepad","force":false,"mode":"background"}
```

## argument validation

### should reject an invalid timeout

```execute
aux4 computer linux quit mousepad --timeout soon
```

```error:partial
Invalid --timeout 'soon'*?
```

### should fail for an app that isn't running

```execute
AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux quit ThisAppDoesNotExist12345
```

```error:partial
App not running*?
```
