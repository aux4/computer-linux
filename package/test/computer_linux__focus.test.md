# aux4 computer linux focus

## foreground by nature

### should bring the app forward

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux focus mousepad | jq -c '{focused, mode}'
```

```expect
{"focused":"mousepad","mode":"foreground"}
```
