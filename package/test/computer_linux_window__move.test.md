# aux4 computer linux window move

## dry run

### should move in the background

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux window move --app mousepad --x 120 --y 140 | jq -c '{action, mode, requested}'
```

```expect
{"action":"move","mode":"background","requested":{"x":120,"y":140}}
```

## argument validation

### should need both coordinates

```execute
aux4 computer linux window move --app mousepad --x 120
```

```error:partial
Provide --x and --y*?
```
