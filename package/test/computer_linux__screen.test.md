# aux4 computer linux screen

## Wayland

### should not be supported without X11

```execute
env -u DISPLAY WAYLAND_DISPLAY=wayland-0 aux4 computer linux screen 2>&1 | jq -c '{unsupported, command, session}'
```

```expect
{"unsupported":true,"command":"screen","session":"wayland"}
```
