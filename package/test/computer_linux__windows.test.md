# aux4 computer linux windows

## Wayland

### should not be supported without X11

```execute
env -u DISPLAY WAYLAND_DISPLAY=wayland-0 aux4 computer linux windows 2>&1 | jq -c '{unsupported, command, session}'
```

```expect
{"unsupported":true,"command":"windows","session":"wayland"}
```
