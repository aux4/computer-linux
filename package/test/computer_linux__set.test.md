# aux4 computer linux set

## argument validation

### should require an element

```execute
aux4 computer linux set --app mousepad --value hello
```

```error:partial
Provide --element*?
```

### should require exactly one change

```execute
aux4 computer linux set --app mousepad --element 0.1 --value a --checked true
```

```error:partial
Give exactly one of --value*?
```

### should reject a non-boolean --checked

```execute
aux4 computer linux set --app mousepad --element 0.1 --checked maybe
```

```error:partial
Invalid --checked 'maybe'*?
```

## dry run

### should set in the background

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux set --app mousepad --element 0.1 --value hello | jq -c '{element, mode}'
```

```expect
{"element":"0.1","mode":"background"}
```
