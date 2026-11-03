# aux4 computer linux drag

## real input

### should always drag with the real pointer

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux drag --x 10 --y 20 --toX 100 --toY 20 | jq -c '{mode, from, to}'
```

```expect
{"mode":"foreground","from":{"x":10,"y":20},"to":{"x":100,"y":20}}
```

### should reject a missing end point

```execute
aux4 computer linux drag --x 10 --y 20 --toX 100 --toY nope
```

```error:partial
Invalid --toY 'nope'*?
```
