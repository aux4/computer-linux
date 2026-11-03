# aux4 computer linux clipboard read

## Wayland

### should not be supported without X11

```execute
env -u DISPLAY WAYLAND_DISPLAY=wayland-0 aux4 computer linux clipboard read 2>&1 | jq -c '{unsupported, command}'
```

```expect
{"unsupported":true,"command":"clipboard read"}
```
