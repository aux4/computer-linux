# aux4 computer linux screenshot

## argument validation

### should reject an invalid --roles

```execute
aux4 computer linux screenshot --annotate true --roles some
```

```error:partial
Invalid --roles 'some'*?
```

### should reject an invalid --maxWidth

```execute
aux4 computer linux screenshot --maxWidth 0
```

```error:partial
Invalid --maxWidth '0'*?
```

## sessions that can't capture

### should explain that Wayland can't capture the screen

```execute
env -u DISPLAY WAYLAND_DISPLAY=wayland-0 aux4 computer linux screenshot 2>&1 | jq -c '{unsupported, session, hint: (.hint | test("X11"))}'
```

```expect
{"unsupported":true,"session":"wayland","hint":true}
```

### should explain that there is no session

```execute
env -u DISPLAY -u WAYLAND_DISPLAY aux4 computer linux screenshot 2>&1 | jq -c '{unsupported, session}'
```

```expect
{"unsupported":true,"session":"none"}
```

### should refuse when screen capture is denied for testing

```execute
AUX4_COMPUTER_DENY_PERMISSIONS=1 aux4 computer linux screenshot
```

```error:partial
Screen Recording permission not granted*?
```
