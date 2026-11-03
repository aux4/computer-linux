# aux4 computer linux permissions

## reporting status

### should report each permission as granted + guidance

```execute
aux4 computer linux permissions | jq -c '[.[] | (.granted | type), (.guidance | type)] | unique'
```

```expect
["boolean","string"]
```

### should report the three permissions

```execute
aux4 computer linux permissions | jq -c 'keys'
```

```expect
["accessibility","input","screenRecording"]
```

### should report every permission as not granted when denied for testing

```execute
AUX4_COMPUTER_DENY_PERMISSIONS=1 aux4 computer linux permissions | jq -c '[.[] | .granted] | unique'
```

```expect
[false]
```

### should explain that Wayland has no screen capture

```execute
env -u DISPLAY WAYLAND_DISPLAY=wayland-0 aux4 computer linux permissions | jq -c '{granted: .screenRecording.granted, wayland: (.screenRecording.guidance | test("Wayland"))}'
```

```expect
{"granted":false,"wayland":true}
```
