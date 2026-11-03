# aux4 computer linux find

## a missing app

### should fail with a JSON error

```execute
aux4 computer linux find Save --app ThisAppDoesNotExist12345
```

```error
{"error":"App not running: ThisAppDoesNotExist12345"}
```
