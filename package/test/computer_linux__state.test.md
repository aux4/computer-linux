# aux4 computer linux state

## argument validation

### should reject a negative depth

```execute
aux4 computer linux state --depth=-1 --app mousepad
```

```error:partial
Invalid --depth*?
```

### should reject an invalid limit

```execute
aux4 computer linux state --limit 0 --app mousepad
```

```error:partial
Invalid --limit '0'*?
```

### should fail for an app that isn't running

```execute
aux4 computer linux state --app ThisAppDoesNotExist12345
```

```error:partial
App not running*?
```
