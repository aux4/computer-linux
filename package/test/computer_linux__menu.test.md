# aux4 computer linux menu

## dry run

### should press a menu path in the background

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux menu --app galculator --path 'View > Scientific Mode' | jq -c '{path, mode}'
```

```expect
{"path":["View","Scientific Mode"],"mode":"background"}
```

## argument validation

### should reject an invalid depth

```execute
aux4 computer linux menu --app galculator --depth 0
```

```error:partial
Invalid --depth '0'*?
```

### should fail for an app that isn't running

```execute
aux4 computer linux menu --app ThisAppDoesNotExist12345
```

```error:partial
App not running*?
```
