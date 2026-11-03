# aux4 computer linux window raise

## foreground by nature

### should always be foreground

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux window raise --app mousepad | jq -c '{action, mode}'
```

```expect
{"action":"raise","mode":"foreground"}
```

### should refuse --mode background

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux window raise --app mousepad --mode background
```

```error:partial
Background mode not possible for window raise*?
```
