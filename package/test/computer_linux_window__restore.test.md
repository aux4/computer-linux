# aux4 computer linux window restore

```afterAll
rm -rf state
```

## dry run

### should restore in the background

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux window restore --app mousepad | jq -c '{action, mode}'
```

```expect
{"action":"restore","mode":"background"}
```

## Wayland

### should not be supported without X11

```execute
AUX4_COMPUTER_STATE_DIR=$PWD/state env -u DISPLAY WAYLAND_DISPLAY=wayland-0 aux4 computer linux window restore --app mousepad 2>&1 | jq -c '{unsupported}'
```

```expect
{"unsupported":true}
```
