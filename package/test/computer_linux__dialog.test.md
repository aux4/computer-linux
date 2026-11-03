# aux4 computer linux dialog

## dry run

### should press a dialog button in the background

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux dialog --app mousepad --press "Don't Save" | jq -c '{press, mode}'
```

```expect
{"press":"Don't Save","mode":"background"}
```

### should refuse to fall back with --mode background

```execute
AUX4_COMPUTER_DRY_RUN=1 AUX4_COMPUTER_SIMULATE_BACKGROUND=unavailable aux4 computer linux dialog --app mousepad --press Cancel --mode background
```

```error:partial
Background mode not possible for dialog*?
```
