# aux4 computer linux window fullscreen

## dry run

### should leave full screen in the background

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux window fullscreen false --app mousepad | jq -c '{action, mode, requested}'
```

```expect
{"action":"fullscreen","mode":"background","requested":{"fullscreen":false}}
```
