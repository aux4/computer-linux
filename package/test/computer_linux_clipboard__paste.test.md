# aux4 computer linux clipboard paste

## dry run

### should paste in the background

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux clipboard paste --text hello --app mousepad | jq -c '{app, mode}'
```

```expect
{"app":"mousepad","mode":"background"}
```
