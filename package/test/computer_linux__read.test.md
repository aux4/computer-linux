# aux4 computer linux read

## argument validation

### should reject an invalid depth

```execute
aux4 computer linux read --depth none
```

```error:partial
Invalid --depth 'none'*?
```

### should fail for an app that isn't running

```execute
aux4 computer linux read --app ThisAppDoesNotExist12345
```

```error:partial
App not running*?
```
