# aux4 computer linux window close

```afterAll
rm -rf state
```

## dry run

### should close in the background

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux window close --app mousepad | jq -c '{action, mode}'
```

```expect
{"action":"close","mode":"background"}
```

## Wayland

### should not be supported without X11

```execute
AUX4_COMPUTER_STATE_DIR=$PWD/state env -u DISPLAY WAYLAND_DISPLAY=wayland-0 aux4 computer linux window close --app mousepad 2>&1 | jq -c '{unsupported}'
```

```expect
{"unsupported":true}
```
