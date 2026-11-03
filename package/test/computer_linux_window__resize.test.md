# aux4 computer linux window resize

## dry run

### should resize in the background

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux window resize --app mousepad --width 600 | jq -c '{action, mode, requested}'
```

```expect
{"action":"resize","mode":"background","requested":{"width":600}}
```

## argument validation

### should need a size

```execute
aux4 computer linux window resize --app mousepad
```

```error:partial
Provide --width and/or --height*?
```

### should reject a negative width

```execute
aux4 computer linux window resize --app mousepad --width=-5
```

```error:partial
Invalid --width*?
```
