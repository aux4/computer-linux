# aux4 computer linux scroll

## mode decision (dry run)

### should scroll in the background by default

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux scroll --dy=-200 --app mousepad | jq -c '{dy, mode}'
```

```expect
{"dy":-200,"mode":"background"}
```

## argument validation

### should reject a non-numeric delta

```execute
aux4 computer linux scroll --dy lots
```

```error:partial
Invalid --dy 'lots'*?
```
