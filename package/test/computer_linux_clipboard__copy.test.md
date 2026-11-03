# aux4 computer linux clipboard copy

```afterAll
rm -rf state
```

## dry run

### should copy text directly

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux clipboard copy --text hello
```

```expect:json
{
  "dryRun": true,
  "text": "hello"
}
```

### should copy an app's selection in the background

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux clipboard copy --app mousepad | jq -c '{app, mode}'
```

```expect
{"app":"mousepad","mode":"background"}
```

## Wayland

### should not be supported without X11

```execute
AUX4_COMPUTER_STATE_DIR=$PWD/state env -u DISPLAY WAYLAND_DISPLAY=wayland-0 aux4 computer linux clipboard copy --text hello 2>&1 | jq -c '{unsupported}'
```

```expect
{"unsupported":true}
```
