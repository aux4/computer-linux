# aux4 computer linux move

```afterAll
rm -rf state
```

## real input

### should always move the real pointer

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux move --x 10 --y 20
```

```expect:json
{
  "dryRun": true,
  "mode": "foreground",
  "x": 10,
  "y": 20
}
```

### should not be possible on Wayland

```execute
AUX4_COMPUTER_STATE_DIR=$PWD/state env -u DISPLAY WAYLAND_DISPLAY=wayland-0 aux4 computer linux move --x 10 --y 20 2>&1 | jq -c '{unsupported, session}'
```

```expect
{"unsupported":true,"session":"wayland"}
```
