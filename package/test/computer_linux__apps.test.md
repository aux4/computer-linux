# aux4 computer linux apps

## listing

### should return a list of apps

```execute
env -u DISPLAY -u WAYLAND_DISPLAY aux4 computer linux apps | jq -c '.apps | type'
```

```expect
"array"
```
